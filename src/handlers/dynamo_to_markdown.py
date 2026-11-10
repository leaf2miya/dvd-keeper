"""DynamoDB のデータを Markdown 形式で S3 に出力するハンドラモジュール。"""

import logging
import os
from typing import Any

import boto3

logger = logging.getLogger(__name__)

s3_client = boto3.client("s3")
dynamodb = boto3.resource("dynamodb")

COLUMN_DESCRIPTIONS: dict[str, str] = {
    "filename": "ファイル名",
    "content_type": "コンテンツタイプ",
    "manufacturer_code": "メーカーコード",
    "fanza_code": "FANZAコード",
    "mgs_code": "MGSコード",
    "sokmil_code": "SOKMILコード",
}

PRIMARY_KEY = "filename"


def parse_s3_path(s3_path: str) -> tuple[str, str]:
    """s3://bucket/prefix 形式の文字列からバケット名とプレフィックスを返す。

    Args:
        s3_path: s3://bucket/prefix 形式のS3パス。

    Returns:
        (bucket, prefix) のタプル。

    Raises:
        ValueError: 不正なS3パス形式の場合。
    """
    trimmed = s3_path.removeprefix("s3://")
    parts = trimmed.split("/", 1)
    if not parts[0]:
        raise ValueError(f"Invalid s3 path: {s3_path}")
    bucket = parts[0]
    prefix = parts[1] if len(parts) == 2 else ""
    return bucket, prefix


def generate_markdown(item: dict[str, Any]) -> str:
    """DynamoDB のレコードから RAG 検索向け Markdown 文字列を生成する。

    Args:
        item: DynamoDB レコードの辞書。

    Returns:
        Markdown 形式の文字列。
    """
    primary_value = item.get(PRIMARY_KEY, "")

    # 概要セクション用の自然文を構築（RAG検索でヒットしやすくするため）
    summary_parts = [f"{primary_value} は DVD コンテンツです。"]
    if item.get("manufacturer_code"):
        summary_parts.append(f"メーカーコード: {item['manufacturer_code']}、")
    if item.get("fanza_code"):
        summary_parts.append(f"FANZA: {item['fanza_code']}、")
    if item.get("mgs_code"):
        summary_parts.append(f"MGS: {item['mgs_code']}、")
    if item.get("sokmil_code"):
        summary_parts.append(f"SOKMIL: {item['sokmil_code']}、")
    summary = " ".join(summary_parts).rstrip("、 ")
    if not summary.endswith("。"):
        summary += " で取り扱いがあります。"

    lines = [
        f"# {PRIMARY_KEY}: {primary_value}",
        "## 概要",
        summary,
        "## レコード",
    ]

    for key, value in item.items():
        description = COLUMN_DESCRIPTIONS.get(key, key)
        lines.append(f"- **{key}** {description}: {value}")

    return "\n".join(lines) + "\n"


def _delete_all_objects(bucket: str, prefix: str) -> None:
    """指定S3パス配下のオブジェクトをすべて削除する。

    Args:
        bucket: S3バケット名。
        prefix: 削除対象のプレフィックス。
    """
    paginator = s3_client.get_paginator("list_objects_v2")
    prefix_with_slash = prefix.rstrip("/") + "/" if prefix else ""

    for page in paginator.paginate(Bucket=bucket, Prefix=prefix_with_slash):
        objects = page.get("Contents", [])
        if not objects:
            continue
        delete_keys = [{"Key": obj["Key"]} for obj in objects]
        s3_client.delete_objects(
            Bucket=bucket,
            Delete={"Objects": delete_keys},
        )
        logger.info("Deleted %d objects from s3://%s/%s", len(delete_keys), bucket, prefix_with_slash)


def _put_markdown(bucket: str, prefix: str, filename: str, content: str) -> None:
    """Markdown コンテンツを S3 に保存する。

    Args:
        bucket: S3バケット名。
        prefix: 保存先プレフィックス。
        filename: ファイル名（拡張子なし）。
        content: Markdown コンテンツ。
    """
    prefix_with_slash = prefix.rstrip("/") + "/" if prefix else ""
    key = f"{prefix_with_slash}{filename}.md"
    s3_client.put_object(
        Bucket=bucket,
        Key=key,
        Body=content.encode("utf-8"),
        ContentType="text/markdown; charset=utf-8",
    )


def full_sync_handler(event: dict[str, Any], context: Any) -> dict[str, Any]:
    """DynamoDB を全件スキャンして Markdown を S3 に全量出力する Lambda ハンドラ。

    既存の Markdown ファイルを全削除後、全件を再生成する。

    Args:
        event: Lambda イベント（EventBridge スケジュールまたは手動実行）。
        context: Lambda コンテキスト。boto3 に標準型定義がなく本関数内では使用しないため Any。

    Returns:
        処理結果を含む辞書。

    Raises:
        Exception: 処理中にエラーが発生した場合。
    """
    table_name = os.environ["DYNAMODB_TABLE_NAME"]
    markdown_s3_path = os.environ["MARKDOWN_OUTPUT_PATH"]

    bucket, prefix = parse_s3_path(markdown_s3_path)
    table = dynamodb.Table(table_name)

    logger.info("Starting full sync: table=%s, output=s3://%s/%s", table_name, bucket, prefix)

    _delete_all_objects(bucket, prefix)

    items: list[dict[str, Any]] = []
    scan_kwargs: dict[str, Any] = {}
    while True:
        response = table.scan(**scan_kwargs)
        items.extend(response.get("Items", []))
        last_key = response.get("LastEvaluatedKey")
        if not last_key:
            break
        scan_kwargs["ExclusiveStartKey"] = last_key

    logger.info("Scanned %d items from DynamoDB", len(items))

    for item in items:
        filename = item.get(PRIMARY_KEY)
        if not filename:
            logger.warning("Skipping item without primary key: %s", item)
            continue
        content = generate_markdown(item)
        _put_markdown(bucket, prefix, str(filename), content)

    logger.info("Full sync completed: %d files written", len(items))
    return {"status": "success", "count": len(items)}
