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

- Python 3.12
- AWS SAM CLI
- uv (パッケージ管理)

## セットアップ
### パラメータストア

| パラメータパス | 説明 |
|---|---|
| `/iso-keeper/dynamo-name` | DynamoDB テーブル名 |
| `/iso-keeper/s3-isos-path` | iso格納先path(s3://...) |
| `/iso-keeper/s3-metadata-path` | メタデータ格納先path(s3://...) |


