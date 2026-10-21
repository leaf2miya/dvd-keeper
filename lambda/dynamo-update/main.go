// Package main は DynamoDB Streaming の INSERT イベントをトリガーに
// スクレイパーでコンテンツIDを取得し DynamoDB を更新する Lambda ハンドラーです。
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/leaf2miya/dvd-keeper/scraper/avwiki"
)

var reSymbols = regexp.MustCompile(`[^A-Z0-9]`)
var reLetterDigit = regexp.MustCompile(`([A-Z])([0-9])`)
var reDigitLetter = regexp.MustCompile(`([0-9])([A-Z])`)

// s3PutObjecter は S3 への PutObject 操作を抽象化するインターフェースです。
type s3PutObjecter interface {
	PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
}

// failedRecord は S3 に保存する失敗レコードの構造体です。
type failedRecord struct {
	Reason string                     `json:"reason"`
	Event  events.DynamoDBEventRecord `json:"event"`
}

// buildKeywords は filename から最大3種類の検索キーワードを生成します。
func buildKeywords(filename string) []string {
	k1 := strings.ToUpper(strings.ReplaceAll(filename, "_", "-"))
	k2 := reSymbols.ReplaceAllString(k1, "")
	k3 := reLetterDigit.ReplaceAllString(k2, "${1}-${2}")
	k3 = reDigitLetter.ReplaceAllString(k3, "${1}-${2}")

	seen := map[string]bool{}
	keywords := []string{}
	for _, k := range []string{k1, k2, k3} {
		if !seen[k] {
			seen[k] = true
			keywords = append(keywords, k)
		}
	}
	return keywords
}

// search は keywords を順に試し、最初に succeeded になった Result を返します。
func search(client *avwiki.Client, keywords []string) (*avwiki.Result, error) {
	var lastErr error
	for _, kw := range keywords {
		result, err := client.Search(kw)
		if err != nil {
			lastErr = err
			continue
		}
		if result.Status == "succeeded" {
			return result, nil
		}
		lastErr = fmt.Errorf("status failed for keyword %q: %s", kw, result.ErrorMessage)
	}
	return nil, lastErr
}

// saveFailureToS3 は失敗レコードをイベント情報ごと S3 に JSON 形式で保存します。
func saveFailureToS3(ctx context.Context, s3Client s3PutObjecter, bucket, prefix, filename, reason string, record events.DynamoDBEventRecord) error {
	rec := failedRecord{Reason: reason, Event: record}
	body, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal failed record: %w", err)
	}

	key := fmt.Sprintf("%s/%d_%s.json", strings.Trim(prefix, "/"), time.Now().UnixMilli(), filename)
	_, err = s3Client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/json"),
	})
	return err
}

// updateDynamoDB は DynamoDB のレコードにスクレイパー取得結果を反映します。
func updateDynamoDB(ctx context.Context, dbClient *dynamodb.Client, tableName, filename string, result *avwiki.Result) error {
	_, err := dbClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"filename": &types.AttributeValueMemberS{Value: filename},
		},
		UpdateExpression: aws.String(
			"SET manufacturer_code = :mc, fanza_code = :fc, mgs_code = :mgsc, sokmil_code = :sc",
		),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":mc":   &types.AttributeValueMemberS{Value: result.ManufacturerCode},
			":fc":   &types.AttributeValueMemberS{Value: result.FanzaCode},
			":mgsc": &types.AttributeValueMemberS{Value: result.MgsCode},
			":sc":   &types.AttributeValueMemberS{Value: result.SokmilCode},
		},
	})
	return err
}

// parseS3Path は "s3://bucket/prefix" 形式の文字列からバケット名とプレフィックスを返します。
func parseS3Path(s3Path string) (bucket, prefix string, err error) {
	trimmed := strings.TrimPrefix(s3Path, "s3://")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 1 || parts[0] == "" {
		return "", "", fmt.Errorf("invalid s3 path: %s", s3Path)
	}
	bucket = parts[0]
	if len(parts) == 2 {
		prefix = parts[1]
	}
	return bucket, prefix, nil
}

func handler(ctx context.Context, event events.DynamoDBEvent) error {
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")
	failureS3Path := os.Getenv("FAILURE_S3_PATH")

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	dbClient := dynamodb.NewFromConfig(cfg)
	s3Client := awss3.NewFromConfig(cfg)
	scraperClient := avwiki.NewClient()

	failureBucket, failurePrefix, err := parseS3Path(failureS3Path)
	if err != nil {
		return fmt.Errorf("parse failure s3 path: %w", err)
	}

	var processErrors []error
	for _, record := range event.Records {
		if record.EventName != "INSERT" {
			continue
		}

		filenameAttr, ok := record.Change.NewImage["filename"]
		if !ok {
			log.Printf("skip record: no filename attribute")
			continue
		}
		filename := filenameAttr.String()

		keywords := buildKeywords(filename)
		result, searchErr := search(scraperClient, keywords)
		if searchErr != nil || result == nil {
			reason := "search returned nil"
			if searchErr != nil {
				reason = searchErr.Error()
			}
			log.Printf("search failed for %q: %s", filename, reason)
			if s3Err := saveFailureToS3(ctx, s3Client, failureBucket, failurePrefix, filename, reason, record); s3Err != nil {
				log.Printf("save to S3 failed: %v", s3Err)
			}
			processErrors = append(processErrors, fmt.Errorf("search failed for %q: %s", filename, reason))
			continue
		}

		if dbErr := updateDynamoDB(ctx, dbClient, tableName, filename, result); dbErr != nil {
			log.Printf("dynamodb update failed for %q: %v", filename, dbErr)
			if s3Err := saveFailureToS3(ctx, s3Client, failureBucket, failurePrefix, filename, dbErr.Error(), record); s3Err != nil {
				log.Printf("save to S3 failed: %v", s3Err)
			}
			processErrors = append(processErrors, dbErr)
			continue
		}

		log.Printf("successfully updated %q", filename)
	}

	if len(processErrors) > 0 {
		return fmt.Errorf("%d record(s) failed processing", len(processErrors))
	}
	return nil
}

func main() {
	lambda.Start(handler)
}
