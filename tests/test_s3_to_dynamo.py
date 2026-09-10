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

        sys.modules.pop("src.handlers.s3_to_dynamo", None)
        from src.handlers.s3_to_dynamo import handler

        yield {"s3": s3, "dynamodb": dynamodb, "handler": handler}


def _make_event(bucket, key):
    return {
        "Records": [
            {
                "s3": {
                    "bucket": {"name": bucket},
                    "object": {"key": key},
                }
            }
        ]
    }


def test_basic_upload(setup_aws):
    setup_aws["s3"].put_object(
        Bucket="test-bucket",
        Key="dvd/movie.iso",
        Body=b"dummy",
        ContentType="application/iso-image",
    )

    setup_aws["handler"](_make_event("test-bucket", "dvd/movie.iso"), None)

    table = setup_aws["dynamodb"].Table("dvd-keeper-test")
    item = table.get_item(Key={"filename": "movie"})["Item"]
    assert item["filename"] == "movie"
    assert item["content_type"] == "application/iso-image"


def test_nested_directory_path(setup_aws):
    setup_aws["s3"].put_object(
        Bucket="test-bucket",
        Key="dvd/sub/dir/title.iso",
        Body=b"dummy",
        ContentType="application/iso-image",
    )

    setup_aws["handler"](_make_event("test-bucket", "dvd/sub/dir/title.iso"), None)

    table = setup_aws["dynamodb"].Table("dvd-keeper-test")
    item = table.get_item(Key={"filename": "title"})["Item"]
    assert item["filename"] == "title"
    assert item["content_type"] == "application/iso-image"


def test_file_without_extension(setup_aws):
    setup_aws["s3"].put_object(
        Bucket="test-bucket",
        Key="dvd/noext",
        Body=b"dummy",
        ContentType="application/octet-stream",
    )

    setup_aws["handler"](_make_event("test-bucket", "dvd/noext"), None)

    table = setup_aws["dynamodb"].Table("dvd-keeper-test")
    item = table.get_item(Key={"filename": "noext"})["Item"]
    assert item["filename"] == "noext"
    assert item["content_type"] == "application/octet-stream"
