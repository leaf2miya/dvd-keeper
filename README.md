# ISO Keeper

DVDの記録管理を行うサービスを担います

## 概要

このプロジェクトは、DVDの記録管理を実施します

### 主な機能

- **オブジェクト管理**: s3上のDVD情報を管理します
- **UI**: WEBUIから、キーワードおよびFuzzy検索を行います(予定)

### アーキテクチャ
- ストレージ
  s3
- メタデータ
  DynamoDB
- Web(予定)
  CloudFront + API Gateway + Cognito
- AI(予定)
  RAG

### 技術スタック

- Python 3
- AWS SAM CLI
- uv (パッケージ管理)

## セットアップ
### パラメータストア

| パラメータパス | 説明 |
|---|---|
| `/dvd-keeper/dynamo/dbname` | DynamoDB テーブル名 |
| `/dvd-keeper/s3/object-home-path` | データ格納先path(s3://...) |


