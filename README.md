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

Run the tool:

```
# Basic usage
./datadog-sql-metrics

# Specify a different config file
./datadog-sql-metrics -config /path/to/your/config.yaml
```

## Command Line Options

The following command line options are available:

```
  -config string
        Path to the YAML configuration file (default "config.yaml")
  -debug
        Enable debug mode for detailed JSON-formatted logs
  -dry-run
        Dry run mode - don't actually send metrics to Datadog
  -version
        Print the version information
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

## Output Format

Logs are output in JSON format with timestamps:

```json
{"timestamp":"2023-03-30T12:34:56Z","level":"info","message":"Metric sent successfully","data":{"metric":"custom.metric.cpu_usage","status":202}}
```

In debug mode, more detailed information is logged:

```json
{"timestamp":"2023-03-30T12:34:55Z","level":"debug","message":"Executing SQL query","data":{"metric":"custom.metric.cpu_usage","query":"SELECT age FROM users LIMIT 1;"}}
{"timestamp":"2023-03-30T12:34:55Z","level":"debug","message":"SQL query result","data":{"metric":"custom.metric.cpu_usage","value":25}}
```
