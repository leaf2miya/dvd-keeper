import os
from urllib.parse import unquote_plus

import boto3

s3_client = boto3.client("s3")
dynamodb = boto3.resource("dynamodb")
table = dynamodb.Table(os.environ["DYNAMODB_TABLE_NAME"])


def handler(event, context):
    for record in event["Records"]:
        bucket = record["s3"]["bucket"]["name"]
        key = unquote_plus(record["s3"]["object"]["key"])

        head = s3_client.head_object(Bucket=bucket, Key=key)
        content_type = head["ContentType"]

        basename = os.path.basename(key)
        filename = os.path.splitext(basename)[0]

        table.put_item(Item={"filename": filename, "content_type": content_type})
