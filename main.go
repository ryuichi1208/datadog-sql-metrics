package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

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
	Value float64  `yaml:"value"`
	Tags  []string `yaml:"tags"`
	Host  string   `yaml:"host"`
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

func main() {
	apiKey := os.Getenv("DATADOG_API_KEY")
	if apiKey == "" {
		fmt.Println("DATADOG_API_KEY is not set")
		return
	}

	client := &DatadogClient{APIKey: apiKey}

	// 設定ファイルの読み込み
	config, err := loadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// YAML から読み込んだメトリクスを送信
	for _, metric := range config.Metrics {
		err := client.SendMetric(metric.Name, metric.Value, metric.Tags, metric.Host)
		if err != nil {
			fmt.Printf("Error sending metric %s: %v\n", metric.Name, err)
		}
	}
}
