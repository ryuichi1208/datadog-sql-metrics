package main

import (
	"fmt"
	"testing"
)

// MockDB: テスト用のモックデータベース
type MockDB struct {
	QueryResults map[string]float64
	Err          error
}

// QueryRow: モックとして動作し、事前に設定された値を返す
func (m *MockDB) QueryRow(query string) (float64, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.QueryResults[query], nil
}

// MockMetricSender: テスト用のモック送信
type MockMetricSender struct {
	SentMetrics []DataSeries
	Err         error
}

// SendMetric: モックとしてデータを保存し、Datadog に送信したように見せる
func (m *MockMetricSender) SendMetric(metricName string, value float64, tags []string, host string) error {
	if m.Err != nil {
		return m.Err
	}
	m.SentMetrics = append(m.SentMetrics, DataSeries{
		Metric: metricName,
		Points: [][]float64{{float64(0), value}},
		Tags:   tags,
		Host:   host,
		Type:   "gauge",
	})
	return nil
}

// ✅ **バリデーションのテスト**
func TestValidateQuery_ValidSelect(t *testing.T) {
	query := "SELECT age FROM users LIMIT 1;"
	if err := validateQuery(query); err != nil {
		t.Fatalf("Expected valid query, got error: %v", err)
	}
}

func TestValidateQuery_InvalidInsert(t *testing.T) {
	query := "INSERT INTO users (name, age) VALUES ('John', 30);"
	if err := validateQuery(query); err == nil {
		t.Fatal("Expected error for INSERT query, but got nil")
	}
}

func TestValidateQuery_InvalidDrop(t *testing.T) {
	query := "DROP TABLE users;"
	if err := validateQuery(query); err == nil {
		t.Fatal("Expected error for DROP query, but got nil")
	}
}

func TestValidateQuery_ValidSelectWithWhitespace(t *testing.T) {
	query := "   SELECT name FROM users   "
	if err := validateQuery(query); err != nil {
		t.Fatalf("Expected valid query, got error: %v", err)
	}
}

// ✅ **fetchMetricFromDB のテスト**
func TestFetchMetricFromDB(t *testing.T) {
	mockDB := &MockDB{
		QueryResults: map[string]float64{
			"SELECT age FROM users LIMIT 1;": 30.0,
		},
	}

	query := "SELECT age FROM users LIMIT 1;"
	value, err := fetchMetricFromDB(mockDB, query)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedValue := 30.0
	if value != expectedValue {
		t.Errorf("Expected %f, got %f", expectedValue, value)
	}
}

// ✅ **run の正常系テスト**
func TestRun_Success(t *testing.T) {
	mockDB := &MockDB{
		QueryResults: map[string]float64{
			"SELECT age FROM users LIMIT 1;": 30.0,
		},
	}
	mockSender := &MockMetricSender{}

	err := run(mockDB, mockSender, "test_config.yaml")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(mockSender.SentMetrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(mockSender.SentMetrics))
	}

	metric := mockSender.SentMetrics[0]
	expectedMetricName := "user.age"
	if metric.Metric != expectedMetricName {
		t.Errorf("Expected metric name '%s', got '%s'", expectedMetricName, metric.Metric)
	}

	expectedValue := 30.0
	if metric.Points[0][1] != expectedValue {
		t.Errorf("Expected metric value %f, got %f", expectedValue, metric.Points[0][1])
	}
}

// ✅ **run のエラーケース: 無効なクエリ**
func TestRun_InvalidQuery(t *testing.T) {
	mockDB := &MockDB{}
	mockSender := &MockMetricSender{}

	err := run(mockDB, mockSender, "invalid_query_config.yaml")
	if err == nil {
		t.Fatal("Expected error for invalid query, but got nil")
	}
}

// ✅ **run のエラーケース: Datadog API の失敗**
func TestRun_DatadogFailure(t *testing.T) {
	mockDB := &MockDB{
		QueryResults: map[string]float64{
			"SELECT age FROM users LIMIT 1;": 30.0,
		},
	}
	mockSender := &MockMetricSender{
		Err: fmt.Errorf("failed to send to Datadog"),
	}

	err := run(mockDB, mockSender, "test_config.yaml")
	if err == nil {
		t.Fatal("Expected error for Datadog failure, but got nil")
	}
}
