package main

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

// stubS3Client は s3PutObjecter のスタブ実装です。
type stubS3Client struct {
	capturedInput *awss3.PutObjectInput
}

func (s *stubS3Client) PutObject(_ context.Context, params *awss3.PutObjectInput, _ ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	s.capturedInput = params
	return &awss3.PutObjectOutput{}, nil
}

func TestSaveFailureToS3(t *testing.T) {
	stub := &stubS3Client{}
	record := events.DynamoDBEventRecord{
		EventName: "INSERT",
		Change: events.DynamoDBStreamRecord{
			NewImage: map[string]events.DynamoDBAttributeValue{
				"filename": events.NewStringAttribute("HOGE-123"),
			},
		},
	}

	err := saveFailureToS3(context.Background(), stub, "my-bucket", "failures", "HOGE-123", "search failed", record)
	if err != nil {
		t.Fatalf("saveFailureToS3 returned error: %v", err)
	}

	if stub.capturedInput == nil {
		t.Fatal("PutObject was not called")
	}

	if *stub.capturedInput.Bucket != "my-bucket" {
		t.Errorf("Bucket = %q, want %q", *stub.capturedInput.Bucket, "my-bucket")
	}
	if !strings.Contains(*stub.capturedInput.Key, "HOGE-123") {
		t.Errorf("Key %q does not contain filename", *stub.capturedInput.Key)
	}

	body, _ := io.ReadAll(stub.capturedInput.Body)
	var rec failedRecord
	if jsonErr := json.Unmarshal(body, &rec); jsonErr != nil {
		t.Fatalf("body is not valid JSON: %v", jsonErr)
	}
	if rec.Reason != "search failed" {
		t.Errorf("Reason = %q, want %q", rec.Reason, "search failed")
	}
	filenameAttr, ok := rec.Event.Change.NewImage["filename"]
	if !ok {
		t.Fatal("event.change.new_image.filename not found in body")
	}
	if filenameAttr.String() != "HOGE-123" {
		t.Errorf("event filename = %q, want %q", filenameAttr.String(), "HOGE-123")
	}
}

func TestBuildKeywords(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     []string
	}{
		{
			name:     "underscore to hyphen and uppercase",
			filename: "hoge_123",
			want:     []string{"HOGE-123", "HOGE123", "HOGE-123"},
		},
		{
			name:     "already uppercase with hyphen",
			filename: "HOGE-123",
			want:     []string{"HOGE-123", "HOGE123"},
		},
		{
			name:     "alphanumeric only generates hyphen variant",
			filename: "HOGE123",
			want:     []string{"HOGE123", "HOGE-123"},
		},
		{
			name:     "all three variants differ",
			filename: "ab_cd-123",
			want:     []string{"AB-CD-123", "ABCD123", "ABCD-123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildKeywords(tt.filename)
			// deduplicated expected
			seen := map[string]bool{}
			want := []string{}
			for _, w := range tt.want {
				if !seen[w] {
					seen[w] = true
					want = append(want, w)
				}
			}
			if len(got) != len(want) {
				t.Errorf("buildKeywords(%q) len = %d, want %d; got %v", tt.filename, len(got), len(want), got)
				return
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("buildKeywords(%q)[%d] = %q, want %q", tt.filename, i, got[i], want[i])
				}
			}
		})
	}
}

func TestParseS3Path(t *testing.T) {
	tests := []struct {
		name       string
		s3Path     string
		wantBucket string
		wantPrefix string
		wantErr    bool
	}{
		{
			name:       "with prefix",
			s3Path:     "s3://my-bucket/path/to/dir",
			wantBucket: "my-bucket",
			wantPrefix: "path/to/dir",
		},
		{
			name:       "bucket only",
			s3Path:     "s3://my-bucket",
			wantBucket: "my-bucket",
			wantPrefix: "",
		},
		{
			name:    "invalid path",
			s3Path:  "s3://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, prefix, err := parseS3Path(tt.s3Path)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseS3Path(%q) error = %v, wantErr %v", tt.s3Path, err, tt.wantErr)
				return
			}
			if bucket != tt.wantBucket {
				t.Errorf("parseS3Path(%q) bucket = %q, want %q", tt.s3Path, bucket, tt.wantBucket)
			}
			if prefix != tt.wantPrefix {
				t.Errorf("parseS3Path(%q) prefix = %q, want %q", tt.s3Path, prefix, tt.wantPrefix)
			}
		})
	}
}
