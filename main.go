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

	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

// Datadog APIエンドポイント
const datadogAPI = "https://api.datadoghq.com/api/v1/series"

// MetricSender インターフェース
type MetricSender interface {
	SendMetric(metricName string, value float64, tags []string, host string) error
}

// DatadogClient は Datadog にメトリクスを送信する実装
type DatadogClient struct {
	APIKey string
}

// Metric YAML で定義されるデータ構造
type Config struct {
	Metrics []MetricConfig `yaml:"metrics"`
}

type MetricConfig struct {
	Name  string   `yaml:"name"`
	Tags  []string `yaml:"tags"`
	Host  string   `yaml:"host"`
	Query string   `yaml:"query,omitempty"` // SQL クエリ（省略可能）
}

// Metric データの構造体（Datadog API用）
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

// DBClient インターフェース: テスト時にモックを使えるようにする
type DBClient interface {
	QueryRow(query string) (float64, error)
}

// PostgresDB は 実際の PostgreSQL に接続する実装
type PostgresDB struct {
	DB *sql.DB
}

// SendMetric メソッド: Datadog にメトリクスを送信
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

// YAML 設定を読み込む関数
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

// PostgreSQL から値を取得する関数
func fetchMetricFromDB(db *sql.DB, query string) (float64, error) {
	var value interface{} // 型が int か float64 か不明なため interface{} で受け取る
	err := db.QueryRow(query).Scan(&value)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	// 取得した値の型を確認して float64 に変換
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

// QueryRow メソッド: SQL を実行し、値を取得する
func (p *PostgresDB) QueryRow(query string) (float64, error) {
	var value interface{}
	err := p.DB.QueryRow(query).Scan(&value)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	// 型を判定して float64 に変換
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

func run() error {
	// コマンドライン引数から YAML ファイルを指定可能
	yamlFile := flag.String("config", "config.yaml", "Path to the YAML configuration file")
	// version情報を表示
	versionFlag := flag.Bool("version", false, "Print the version information")
	flag.Parse()

	if *versionFlag {
		_version()
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
	if err := validateDBURL(dbURL); err != nil {
		return fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	// PostgreSQL 接続
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}
	defer db.Close()

	client := &DatadogClient{APIKey: apiKey}

	// 設定ファイルの読み込み
	config, err := loadConfig(*yamlFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var value float64
	for _, metric := range config.Metrics {
		if err := validateQuery(metric.Query); err != nil {
			log.Printf("%v", err)
		}

		if metric.Query != "" {
			fetchedValue, err := fetchMetricFromDB(db, metric.Query)
			if err != nil {
				log.Printf("Error fetching metric %s from DB: %v", metric.Name, err)
				continue
			}
			value = fetchedValue
		}

		err := client.SendMetric(metric.Name, value, metric.Tags, metric.Host)
		if err != nil {
			return fmt.Errorf("failed to send metric: %w", err)
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
