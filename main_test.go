package main

import (
	"context"
	"os"
	"testing"
	"time"
)

// MockMetricSender: テスト用のモック実装
type MockMetricSender struct {
	SentMetrics []DataSeries
}

// Mock の SendMetric メソッド
func (m *MockMetricSender) SendMetric(ctx context.Context, metricName string, value float64, tags []string, host string) error {
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
	// Try to load the real config file first
	config, err := loadConfig("config.yaml")
	if err != nil {
		// If real config can't be loaded, create a temporary test file
		t.Logf("Could not load config.yaml: %v", err)
		t.Log("Creating temporary config file for testing")

		tempFile := "test_config.yaml"
		testConfig := []byte(`metrics:
  - name: "custom.metric.cpu_usage"
    tags: ["env:test", "team:sre"]
    host: "server-01"
    query: "SELECT age FROM users LIMIT 1;"`)

		err = os.WriteFile(tempFile, testConfig, 0644)
		if err != nil {
			t.Fatalf("Failed to write test config file: %v", err)
		}
		defer os.Remove(tempFile) // Clean up after test

		// Load the temporary config
		config, err = loadConfig(tempFile)
		if err != nil {
			t.Fatalf("Failed to load test config: %v", err)
		}
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
	ctx := context.Background()

	err := mockSender.SendMetric(ctx, metricName, value, tags, host)
	if err != nil {
		t.Fatalf("SendMetric failed: %v", err)
	}

	if len(mockSender.SentMetrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(mockSender.SentMetrics))
	}

	sent := mockSender.SentMetrics[0]
	if sent.Metric != metricName {
		t.Errorf("Expected metric name '%s', got '%s'", metricName, sent.Metric)
	}
	if sent.Host != host {
		t.Errorf("Expected host '%s', got '%s'", host, sent.Host)
	}
	if len(sent.Points) != 1 || len(sent.Points[0]) != 2 || sent.Points[0][1] != value {
		t.Errorf("Expected value %f, got points %v", value, sent.Points)
	}
}
