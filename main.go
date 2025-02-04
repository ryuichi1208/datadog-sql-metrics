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

type DBClient interface {
	QueryRow(query string) (float64, error)
}

type SQLDB struct {
	DB *sql.DB
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

	fmt.Println("Metric sent successfully:", metricName)
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
	return fetchMetricFromDB(p.DB, query)
}

func run() error {
	yamlFile := flag.String("config", "config.yaml", "Path to the YAML configuration file")
	versionFlag := flag.Bool("version", false, "Print the version information")
	flag.Parse()

	if *versionFlag {
		fmt.Println("Version 1.0.0")
		return nil
	}

	apiKey := os.Getenv("DATADOG_API_KEY")
	if apiKey == "" {
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

	db, err := sql.Open(dbType, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}
	defer db.Close()

	client := &DatadogClient{APIKey: apiKey}

	config, err := loadConfig(*yamlFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	dbClient := &SQLDB{DB: db}

	for _, metric := range config.Metrics {
		var value float64
		if metric.Query != "" {
			fetchedValue, err := dbClient.QueryRow(metric.Query)
			if err != nil {
				log.Printf("Error fetching metric %s from DB: %v", metric.Name, err)
				continue
			}
			value = fetchedValue
		}

		err := client.SendMetric(metric.Name, value, metric.Tags, metric.Host)
		if err != nil {
			log.Printf("Failed to send metric: %v", err)
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
