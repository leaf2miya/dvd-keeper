import sys

import boto3
import pytest
from moto import mock_aws


@pytest.fixture
def aws_env(monkeypatch):
    monkeypatch.setenv("AWS_ACCESS_KEY_ID", "testing")
    monkeypatch.setenv("AWS_SECRET_ACCESS_KEY", "testing")
    monkeypatch.setenv("AWS_SECURITY_TOKEN", "testing")
    monkeypatch.setenv("AWS_SESSION_TOKEN", "testing")
    monkeypatch.setenv("AWS_DEFAULT_REGION", "ap-northeast-1")
    monkeypatch.setenv("DYNAMODB_TABLE_NAME", "dvd-keeper-test")
    monkeypatch.setenv("MARKDOWN_OUTPUT_PATH", "s3://test-bucket/markdown")


@pytest.fixture
def setup_aws(aws_env):
    with mock_aws():
        s3 = boto3.client("s3", region_name="ap-northeast-1")
        s3.create_bucket(
            Bucket="test-bucket",
            CreateBucketConfiguration={"LocationConstraint": "ap-northeast-1"},
        )

        dynamodb = boto3.resource("dynamodb", region_name="ap-northeast-1")
        dynamodb.create_table(
            TableName="dvd-keeper-test",
            KeySchema=[{"AttributeName": "filename", "KeyType": "HASH"}],
            AttributeDefinitions=[
                {"AttributeName": "filename", "AttributeType": "S"}
            ],
            BillingMode="PAY_PER_REQUEST",
        )

        sys.modules.pop("src.handlers.dynamo_to_markdown", None)
        from src.handlers.dynamo_to_markdown import (
            full_sync_handler,
            generate_markdown,
            parse_s3_path,
        )

        yield {
            "s3": s3,
            "dynamodb": dynamodb,
            "full_sync_handler": full_sync_handler,
            "generate_markdown": generate_markdown,
            "parse_s3_path": parse_s3_path,
        }


class TestParseS3Path:
    def test_with_prefix(self, setup_aws):
        bucket, prefix = setup_aws["parse_s3_path"]("s3://my-bucket/some/prefix")
        assert bucket == "my-bucket"
        assert prefix == "some/prefix"

    def test_without_prefix(self, setup_aws):
        bucket, prefix = setup_aws["parse_s3_path"]("s3://my-bucket")
        assert bucket == "my-bucket"
        assert prefix == ""

    def test_invalid_path(self, setup_aws):
        with pytest.raises(ValueError):
            setup_aws["parse_s3_path"]("s3://")


class TestGenerateMarkdown:
    def test_full_item(self, setup_aws):
        item = {
            "filename": "NHDTB-096",
            "content_type": "application/iso-image",
            "manufacturer_code": "NHDTB-096",
            "fanza_code": "1nhdtb00096",
            "mgs_code": "116NHDTB-096",
            "sokmil_code": "item282537",
        }
        md = setup_aws["generate_markdown"](item)
        assert "# filename: NHDTB-096" in md
        assert "## 概要" in md
        assert "NHDTB-096 は DVD コンテンツです" in md
        assert "FANZA: 1nhdtb00096" in md
        assert "MGS: 116NHDTB-096" in md
        assert "SOKMIL: item282537" in md
        assert "## レコード" in md
        assert "**filename**" in md

    def test_partial_item_no_platform_codes(self, setup_aws):
        item = {
            "filename": "TEST-001",
            "content_type": "application/iso-image",
        }
        md = setup_aws["generate_markdown"](item)
        assert "# filename: TEST-001" in md
        assert "TEST-001 は DVD コンテンツです" in md
        assert "FANZA" not in md
        assert "MGS" not in md

    def test_markdown_ends_with_newline(self, setup_aws):
        item = {"filename": "TEST-001"}
        md = setup_aws["generate_markdown"](item)
        assert md.endswith("\n")


class TestFullSyncHandler:
    def test_writes_markdown_files(self, setup_aws):
        table = setup_aws["dynamodb"].Table("dvd-keeper-test")
        table.put_item(Item={
            "filename": "NHDTB-096",
            "content_type": "application/iso-image",
            "manufacturer_code": "NHDTB-096",
        })
        table.put_item(Item={
            "filename": "TEST-001",
            "content_type": "application/iso-image",
        })

        result = setup_aws["full_sync_handler"]({}, None)

        assert result["status"] == "success"
        assert result["count"] == 2

        resp = setup_aws["s3"].list_objects_v2(Bucket="test-bucket", Prefix="markdown/")
        keys = [obj["Key"] for obj in resp.get("Contents", [])]
        assert "markdown/NHDTB-096.md" in keys
        assert "markdown/TEST-001.md" in keys

    def test_deletes_existing_files_before_sync(self, setup_aws):
        # 既存ファイルを配置
        setup_aws["s3"].put_object(
            Bucket="test-bucket",
            Key="markdown/OLD-FILE.md",
            Body=b"old content",
        )

        table = setup_aws["dynamodb"].Table("dvd-keeper-test")
        table.put_item(Item={"filename": "NEW-001"})

        setup_aws["full_sync_handler"]({}, None)

        resp = setup_aws["s3"].list_objects_v2(Bucket="test-bucket", Prefix="markdown/")
        keys = [obj["Key"] for obj in resp.get("Contents", [])]
        assert "markdown/OLD-FILE.md" not in keys
        assert "markdown/NEW-001.md" in keys

    def test_empty_table(self, setup_aws):
        result = setup_aws["full_sync_handler"]({}, None)
        assert result["status"] == "success"
        assert result["count"] == 0

    def test_markdown_content_is_valid(self, setup_aws):
        table = setup_aws["dynamodb"].Table("dvd-keeper-test")
        table.put_item(Item={
            "filename": "NHDTB-096",
            "fanza_code": "1nhdtb00096",
        })

        setup_aws["full_sync_handler"]({}, None)

        obj = setup_aws["s3"].get_object(Bucket="test-bucket", Key="markdown/NHDTB-096.md")
        content = obj["Body"].read().decode("utf-8")
        assert "# filename: NHDTB-096" in content
        assert "FANZA: 1nhdtb00096" in content
