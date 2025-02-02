package main

import (
	"testing"
	"time"
)

// MockMetricSender: テスト用のモック実装
type MockMetricSender struct {
	SentMetrics []DataSeries
}

// Mock の SendMetric メソッド
func (m *MockMetricSender) SendMetric(metricName string, value float64, tags []string, host string) error {
	m.SentMetrics = append(m.SentMetrics, DataSeries{
		Metric: metricName,
		Points: [][]float64{{float64(time.Now().Unix()), value}},
		Tags:   tags,
		Host:   host,
		Type:   "gauge",
	})
	return nil
}

// YAML 設定のロードテスト
func TestLoadConfig(t *testing.T) {
	config, err := loadConfig("config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(config.Metrics) == 0 {
		t.Fatal("Expected at least one metric in config, got zero")
	}

	expectedMetric := "custom.metric.cpu_usage"
	if config.Metrics[0].Name != expectedMetric {
		t.Errorf("Expected metric name '%s', got '%s'", expectedMetric, config.Metrics[0].Name)
	}
}

// Mock を使ったメトリクス送信テスト
func TestMockMetricSender(t *testing.T) {
	mockSender := &MockMetricSender{}

	metricName := "test.metric"
	value := 42.0
	tags := []string{"env:test"}
	host := "test-host"

	err := mockSender.SendMetric(metricName, value, tags, host)
	if err != nil {
		t.Fatalf("SendMetric failed: %v", err)
	}

	if len(mockSender.SentMetrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(mockSender.SentMetrics))
	}
}
