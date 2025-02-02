# datadog-sql-metics

## Description

このツールは、YAML で指定した SQL クエリ を PostgreSQL で実行し、取得したメトリクスを Datadog API に送信します。
クエリは SELECT のみ許可 し、INSERT, UPDATE, DELETE, DROP などの読み取り以外のクエリは 実行できません。

## Usage

Datadog API キーと PostgreSQL の接続情報を設定します。

```
export DATADOG_API_KEY="your-datadog-api-key"
export DATABASE_URL="postgres://user:password@localhost:5432/mydb?sslmode=disable"
```

## YAML Configuration

メトリクスと SQL クエリを定義する YAML ファイルを作成します。デフォルトでは config.yaml を使用します。

```yaml
queries:
  - name: "example"
    query: "SELECT COUNT(*) FROM example_table"
    tags:
      - "environment:production"
```
