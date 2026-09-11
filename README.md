# DVD Keeper

DVDの記録管理を行うサービスを担います

## 概要

このプロジェクトは、DVDの記録管理を実施します

### 主な機能

- **オブジェクト管理**: s3上のDVD情報を管理します
- **メタデータ同期**: S3へのファイルアップロードを契機にDynamoDBへメタデータを自動登録します
- **UI**: WEBUIから、キーワードおよびFuzzy検索を行います(予定)

### ワークフロー

```
S3 Upload → S3イベント通知 → Lambda → DynamoDB
```

1. S3の指定パス配下にファイルをアップロード
2. S3イベント通知がLambdaを起動
3. Lambdaがファイル名（拡張子除去）とContent-TypeをDynamoDBに保存

### アーキテクチャ
- ストレージ
  S3
- メタデータ
  DynamoDB
- コンピュート
  Lambda (Python 3.14)
- IaC
  AWS SAM
- Web(予定)
  CloudFront + API Gateway + Cognito
- AI(予定)
  RAG

### 技術スタック

- Python 3.14
- AWS SAM CLI
- uv (パッケージ管理)

## セットアップ

### パラメータストア

| パラメータパス | 説明 |
|---|---|
| `/dvd-keeper/dynamo/dbname` | DynamoDB テーブル名 |
| `/dvd-keeper/s3/object-home-path` | データ格納先path(s3://...) |

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
