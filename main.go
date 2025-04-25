package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

const datadogAPI = "https://api.datadoghq.com/api/v1/series"

type MetricSender interface {
	SendMetric(ctx context.Context, metricName string, value float64, tags []string, host string) error
}

type DatadogClient struct {
	APIKey string
	Debug  bool
	DryRun bool
}

type Config struct {
	Metrics []MetricConfig `yaml:"metrics"`
}

type MetricConfig struct {
	Name  string   `yaml:"name"`
	Tags  []string `yaml:"tags"`
	Host  string   `yaml:"host"`
	Query string   `yaml:"query,omitempty"`
}

type Metric struct {
	Series []DataSeries `json:"series"`
}

type DataSeries struct {
	Metric string      `json:"metric"`
	Points [][]float64 `json:"points"`
	Tags   []string    `json:"tags,omitempty"`
	Host   string      `json:"host,omitempty"`
	Type   string      `json:"type,omitempty"`
}

type LogEntry struct {
	Timestamp string          `json:"timestamp"`
	Level     string          `json:"level"`
	Message   string          `json:"message"`
	Data      interface{}     `json:"data,omitempty"`
	Ctx       context.Context `json:"-"`
}

type DBClient interface {
	QueryRow(ctx context.Context, query string) (float64, error)
}

type SQLDB struct {
	DB *sql.DB
}

func logJSON(ctx context.Context, level, message string, data interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Data:      data,
		Ctx:       ctx,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log: %v", err)
		return
	}

	fmt.Println(string(jsonData))
}

func (d *DatadogClient) SendMetric(ctx context.Context, metricName string, value float64, tags []string, host string) error {
	timestamp := float64(time.Now().Unix())

	metricData := Metric{
		Series: []DataSeries{
			{
				Metric: metricName,
				Points: [][]float64{{timestamp, value}},
				Tags:   tags,
				Host:   host,
				Type:   "gauge",
			},
		},
	}

	payload, err := json.Marshal(metricData)
	if err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	if d.Debug {
		logJSON(ctx, "debug", "Sending metric to Datadog", map[string]interface{}{
			"metric":  metricName,
			"value":   value,
			"tags":    tags,
			"host":    host,
			"url":     datadogAPI,
			"payload": string(payload),
		})
	}

	if d.DryRun {
		logJSON(ctx, "info", "Dry run mode - skipping actual metric submission", map[string]interface{}{
			"metric": metricName,
			"value":  value,
			"tags":   tags,
			"host":   host,
		})
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", datadogAPI, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", d.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logJSON(ctx, "warn", "Datadog request cancelled or timed out", map[string]interface{}{"error": err.Error()})
			return fmt.Errorf("datadog request failed due to context: %w", err)
		}
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			logJSON(ctx, "warn", "Failed to close response body", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected response code: %d", resp.StatusCode)
	}

	logJSON(ctx, "info", "Metric sent successfully", map[string]interface{}{
		"metric": metricName,
		"status": resp.StatusCode,
	})

	return nil
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

func fetchMetricFromDB(ctx context.Context, db *sql.DB, query string) (float64, error) {
	var value interface{}
	err := db.QueryRowContext(ctx, query).Scan(&value)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logJSON(ctx, "warn", "Database query cancelled or timed out", map[string]interface{}{"query": query, "error": err.Error()})
			return 0, fmt.Errorf("database query failed due to context: %w", err)
		}
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case []byte:
		f, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return 0, fmt.Errorf("could not convert byte slice to float64: %w", err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unexpected data type: %T", v)
	}
}

func (p *SQLDB) QueryRow(ctx context.Context, query string) (float64, error) {
	startTime := time.Now()
	value, err := fetchMetricFromDB(ctx, p.DB, query)
	duration := time.Since(startTime)

	logJSON(ctx, "info", "Query execution completed", map[string]interface{}{
		"query_time_ms": float64(duration.Microseconds()) / 1000.0,
		"query":         query,
		"error":         nil,
	})
	if err != nil {
		logJSON(ctx, "error", "Query execution failed", map[string]interface{}{
			"query_time_ms": float64(duration.Microseconds()) / 1000.0,
			"query":         query,
			"error":         err.Error(),
		})
	}

	return value, err
}

func run(ctx context.Context) error {
	yamlFile := flag.String("config", "config.yaml", "Path to the YAML configuration file")
	versionFlag := flag.Bool("version", false, "Print the version information")
	debugFlag := flag.Bool("debug", false, "Enable debug mode")
	dryRunFlag := flag.Bool("dry-run", false, "Dry run mode - don't actually send metrics to Datadog")
	timeout := flag.Duration("timeout", 30*time.Second, "Global timeout for operations like DB query and API call")
	flag.Parse()

	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	if *versionFlag {
		_version()
		return nil
	}

	apiKey := os.Getenv("DATADOG_API_KEY")
	if apiKey == "" && !*dryRunFlag {
		return fmt.Errorf("DATADOG_API_KEY is not set")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}

	if err := validateDBURL(dbURL); err != nil {
		return fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	dbType := os.Getenv("DATABASE_TYPE")
	if dbType == "" {
		dbType = "postgres"
	}

	if *debugFlag {
		logJSON(ctx, "debug", "Debug mode enabled", map[string]interface{}{
			"config":        *yamlFile,
			"database_url":  dbURL,
			"database_type": dbType,
			"dry_run":       *dryRunFlag,
			"timeout":       timeout.String(),
		})
	}

	if *dryRunFlag {
		logJSON(ctx, "info", "Dry run mode enabled - no metrics will be sent to Datadog", nil)
	}

	db, err := sql.Open(dbType, dbURL)
	if err != nil {
		return fmt.Errorf("failed to initialize DB connection: %w", err)
	}
	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			logJSON(ctx, "warn", "Failed to close database connection", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err = db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}

	client := &DatadogClient{
		APIKey: apiKey,
		Debug:  *debugFlag,
		DryRun: *dryRunFlag,
	}

	config, err := loadConfig(*yamlFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if *debugFlag {
		logJSON(ctx, "debug", "Configuration file loaded", map[string]interface{}{
			"metrics_count": len(config.Metrics),
		})
	}

	dbClient := &SQLDB{DB: db}

	for _, metric := range config.Metrics {
		if err := validateQuery(metric.Query); err != nil {
			logJSON(ctx, "error", "Invalid query in config", map[string]interface{}{
				"metric": metric.Name,
				"query":  metric.Query,
				"error":  err.Error(),
			})
			continue
		}

		var value float64
		if metric.Query != "" {
			if *debugFlag {
				logJSON(ctx, "debug", "Executing SQL query", map[string]interface{}{
					"metric": metric.Name,
					"query":  metric.Query,
				})
			}

			fetchedValue, errDb := dbClient.QueryRow(ctx, metric.Query)

			if errDb != nil {
				logJSON(ctx, "error", "Error fetching metric from DB", map[string]interface{}{
					"metric": metric.Name,
					"error":  errDb.Error(),
				})
				continue
			}
			value = fetchedValue

			if *debugFlag {
				logJSON(ctx, "debug", "SQL query result", map[string]interface{}{
					"metric": metric.Name,
					"value":  value,
				})
			}
		}

		errSend := client.SendMetric(ctx, metric.Name, value, metric.Tags, metric.Host)
		if errSend != nil {
			logJSON(ctx, "error", "Failed to send metric", map[string]interface{}{
				"metric": metric.Name,
				"error":  errSend.Error(),
			})
		}
	}

	return nil
}

func main() {
	ctx := context.Background()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		logJSON(context.Background(), "fatal", "Execution error", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}
}
