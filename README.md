# datadog-sql-metrics

## Description

This tool executes SQL queries specified in YAML against PostgreSQL and sends the retrieved metrics to the Datadog API.
Only SELECT queries are allowed; other operations such as INSERT, UPDATE, DELETE, DROP, or any non-read queries cannot be executed.

## Usage

Set up your Datadog API key and PostgreSQL connection information:

```
export DATADOG_API_KEY="your-datadog-api-key"
export DATABASE_URL="postgres://user:password@localhost:5432/mydb?sslmode=disable"
```

## YAML Configuration

Create a YAML file to define metrics and SQL queries. By default, the tool uses config.yaml.

```yaml
metrics:
  - name: "custom.metric.cpu_usage"
    tags: ["env:test", "team:sre"]
    host: "server-01"
    query: "SELECT age FROM users LIMIT 1;"
```
