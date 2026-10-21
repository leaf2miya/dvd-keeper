# DVD Keeper

DVDの記録管理を行うサービスを担います

## 概要

このプロジェクトは、DVDの記録管理を実施します

### 主な機能

- **オブジェクト管理**: s3上のDVD情報を管理します
- **メタデータ同期**: S3へのファイルアップロードを契機にDynamoDBへメタデータを自動登録します
- **メタデータ更新**: DynamoDB INSERTをトリガに追加情報を自動取得し反映します
- **UI**: WEBUIから、キーワードおよびFuzzy検索を行います(予定)

### ワークフロー

```
S3 Upload → S3イベント通知 → Lambda(S3→DynamoDB) → DynamoDB
                                      ↓ (失敗時)              ↓ (INSERT)
                               Lambda Destinations   Lambda(DynamoDB Update)
                                 → SNS通知              ↓               ↓ (失敗時)
                                                    DynamoDB更新      S3保存
```

1. S3の指定パス配下にファイルをアップロード
2. S3イベント通知がLambda(S3→DynamoDB)を起動
3. LambdaがファイルメタデータをDynamoDBに保存
4. Lambda実行失敗時、Lambda DestinationsによりSNSへ通知
5. WebSearchを通して追加情報を取得しDynamoDBを更新
6. 取得失敗時はS3に失敗レコードを保存（次回実行用）

### アーキテクチャ
- ストレージ
  S3
- メタデータ
  DynamoDB
- 通知
  SNS (Lambda実行失敗時)
- コンピュート
  Lambda (Python 3.14 / Go)
- IaC
  AWS SAM
- Web(予定)
  CloudFront + API Gateway + Cognito
- AI(予定)
  RAG

### 技術スタック

- Python 3.14
- Go
- AWS SAM CLI
- uv (パッケージ管理)

## セットアップ

### パラメータストア

| パラメータパス | 説明 |
|---|---|
| `/dvd-keeper/dynamo/dbname` | DynamoDB テーブル名 |
| `/dvd-keeper/s3/object-home-path` | データ格納先path(s3://...) |
| `/dvd-keeper/sns/failure-topic` | Lambda失敗通知用SNSトピックARN |
| `/dvd-keeper/s3/event-failed-store` | DynamoDB更新失敗レコード保存先パス(s3://...) |

### SAM デプロイ設定

`samconfig.example.toml` をコピーして `samconfig.toml` を作成してください。

```bash
cp samconfig.example.toml samconfig.toml
```

必要に応じて `samconfig.toml` の値を環境に合わせて編集してください。

### 依存インストール

```bash
uv sync
```

### デプロイ

```bash
sam build
sam deploy
```

### S3 イベント通知設定

デプロイ後、既存のS3バケットにLambdaへのイベント通知を設定します。

```bash
aws s3api put-bucket-notification-configuration \
  --bucket <バケット名> \
  --notification-configuration '{
    "LambdaFunctionConfigurations": [
      {
        "LambdaFunctionArn": "<Lambda関数のARN>",
        "Events": ["s3:ObjectCreated:*"],
        "Filter": {
          "Key": {
            "FilterRules": [
              {"Name": "prefix", "Value": "<object-home-path のプレフィックス>"}
            ]
          }
        }
      }
    ]
  }'
```

- `<バケット名>`: SSMパラメータ `/dvd-keeper/s3/object-home-path` のバケット部分
- `<Lambda関数のARN>`: `sam deploy` の出力 `S3ToDynamoFunctionArn` を参照
- `<object-home-path のプレフィックス>`: SSMパラメータのパス部分（バケット名以降）

## テスト

```bash
uv run pytest
```

### DynamoDB Update Lambda テスト

```bash
cd lambda/dynamo-update
go test -v ./...
```

## DynamoDB Update Lambda (`lambda/dynamo-update/`)

### 概要

DynamoDB InsertをトリガにWebSearchを実施し追加情報を取得。取得した情報でDynamoDBを更新する

### 環境変数

| 変数名 | 説明 | 参照SSMパラメータ |
|---|---|---|
| `DYNAMODB_TABLE_NAME` | DynamoDB テーブル名 | `/dvd-keeper/dynamo/dbname` |
| `FAILURE_S3_PATH` | 失敗レコード保存先パス(s3://...) | `/dvd-keeper/s3/event-failed-store` |

### 検索キーワード変換ロジック

1. filename を大文字化 + `_` → `-` に変換
2. 失敗時: 記号を除去し英数字のみに変換
3. 失敗時: 英字と数字の境界にハイフンを挿入

### ビルド

```bash
cd lambda/dynamo-update
go build ./...
```

## av-wiki.net スクレイパー (`scraper/avwiki/`)

### 概要

av-wiki.net をスクレイピングして、各プラットフォーム（FANZA / MGS / SOKMIL）のコンテンツIDを取得する Go モジュールです。

### セットアップ

```bash
cd scraper
go mod download
```

### 使い方

```go
import "github.com/leaf2miya/dvd-keeper/scraper/avwiki"

client := avwiki.NewClient()
result, err := client.Search("NHDTB-096")
```

レスポンス:

```json
{
  "status": "succeeded",
  "error_message": "",
  "manufacturer_code": "NHDTB-096",
  "fanza_code": "1nhdtb00096",
  "mgs_code": "116NHDTB-096",
  "sokmil_code": "item282537"
}
```

### ビルド

```bash
cd scraper
go build ./...
```

### テスト

```bash
cd scraper
go test -v ./...
```

## DMM 商品検索 API クライアント (`dmm-client/`)

### 概要

DMM アフィリエイト API v3 の商品情報 API を利用して商品を検索する Go クライアントです。

### セットアップ

```bash
cd dmm-client
go mod download
```

### 使い方

```go
import dmmclient "github.com/leaf2miya/dvd-keeper/dmm-client"

c, err := dmmclient.NewClient("your-affiliate-id-990", "your-api-id", "DMM.com")
if err != nil {
    log.Fatal(err)
}

resp, err := c.SearchItemList(dmmclient.SearchItemListParams{
    Keyword: "DVD",
    Length:  20,
})
```

### ビルド

```bash
cd dmm-client
go build ./...
```

### テスト

```bash
cd dmm-client
go test ./...
```
