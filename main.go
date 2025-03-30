package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

const datadogAPI = "https://api.datadoghq.com/api/v1/series"

type MetricSender interface {
	SendMetric(metricName string, value float64, tags []string, host string) error
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
	Timestamp string      `json:"timestamp"`
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
}

type DBClient interface {
	QueryRow(query string) (float64, error)
}

type SQLDB struct {
	DB *sql.DB
}

func logJSON(level, message string, data interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Data:      data,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log: %v", err)
		return
	}

	fmt.Println(string(jsonData))
}

func (d *DatadogClient) SendMetric(metricName string, value float64, tags []string, host string) error {
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
		logJSON("debug", "Sending metric to Datadog", map[string]interface{}{
			"metric":  metricName,
			"value":   value,
			"tags":    tags,
			"host":    host,
			"url":     datadogAPI,
			"payload": string(payload),
		})
	}

	if d.DryRun {
		logJSON("info", "Dry run mode - skipping actual metric submission", map[string]interface{}{
			"metric": metricName,
			"value":  value,
			"tags":   tags,
			"host":   host,
		})
		return nil
	}

	req, err := http.NewRequest("POST", datadogAPI, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", d.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected response code: %d", resp.StatusCode)
	}

	logJSON("info", "Metric sent successfully", map[string]interface{}{
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

func fetchMetricFromDB(db *sql.DB, query string) (float64, error) {
	var value interface{}
	err := db.QueryRow(query).Scan(&value)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected data type: %T", v)
	}
}

func (p *SQLDB) QueryRow(query string) (float64, error) {
	startTime := time.Now()
	value, err := fetchMetricFromDB(p.DB, query)
	duration := time.Since(startTime)

	// Log the query execution time
	logJSON("info", "Query execution completed", map[string]interface{}{
		"query_time_ms": float64(duration.Microseconds()) / 1000.0,
		"query":         query,
	})

	return value, err
}

func run() error {
	yamlFile := flag.String("config", "config.yaml", "Path to the YAML configuration file")
	versionFlag := flag.Bool("version", false, "Print the version information")
	debugFlag := flag.Bool("debug", false, "Enable debug mode")
	dryRunFlag := flag.Bool("dry-run", false, "Dry run mode - don't actually send metrics to Datadog")
	flag.Parse()

	if *versionFlag {
		fmt.Println("Version 1.0.0")
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

	dbType := os.Getenv("DATABASE_TYPE") // `postgres` または `mysql`
	if dbType == "" {
		dbType = "postgres" // デフォルトは PostgreSQL
	}

	if *debugFlag {
		logJSON("debug", "Debug mode enabled", map[string]interface{}{
			"config":        *yamlFile,
			"database_url":  dbURL,
			"database_type": dbType,
			"dry_run":       *dryRunFlag,
		})
	}

	if *dryRunFlag {
		logJSON("info", "Dry run mode enabled - no metrics will be sent to Datadog", nil)
	}

	db, err := sql.Open(dbType, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}
	defer db.Close()

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
		logJSON("debug", "Configuration file loaded", map[string]interface{}{
			"metrics_count": len(config.Metrics),
		})
	}

	dbClient := &SQLDB{DB: db}

	for _, metric := range config.Metrics {
		var value float64
		if metric.Query != "" {
			if *debugFlag {
				logJSON("debug", "Executing SQL query", map[string]interface{}{
					"metric": metric.Name,
					"query":  metric.Query,
				})
			}

			startTime := time.Now()
			fetchedValue, err := dbClient.QueryRow(metric.Query)
			duration := time.Since(startTime)

			if err != nil {
				logJSON("error", "Error fetching metric from DB", map[string]interface{}{
					"metric":        metric.Name,
					"error":         err.Error(),
					"query_time_ms": float64(duration.Microseconds()) / 1000.0,
				})
				continue
			}
			value = fetchedValue

			if *debugFlag {
				logJSON("debug", "SQL query result", map[string]interface{}{
					"metric":        metric.Name,
					"value":         value,
					"query_time_ms": float64(duration.Microseconds()) / 1000.0,
				})
			}
		}

		err := client.SendMetric(metric.Name, value, metric.Tags, metric.Host)
		if err != nil {
			logJSON("error", "Failed to send metric", map[string]interface{}{
				"metric": metric.Name,
				"error":  err.Error(),
			})
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		logJSON("fatal", "Execution error", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}
}
