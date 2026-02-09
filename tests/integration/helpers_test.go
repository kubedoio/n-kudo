package integration_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kubedoio/n-kudo/internal/edge/executor"
)

// mustJSON marshals v to JSON, failing the test on error
func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return data
}

// getStringField extracts a string field from a map, failing the test if missing or wrong type
func getStringField(t *testing.T, data map[string]interface{}, key string) string {
	t.Helper()
	val, ok := data[key]
	if !ok {
		t.Fatalf("response missing %s", key)
	}
	str, ok := val.(string)
	if !ok {
		t.Fatalf("expected %s to be string, got %T", key, val)
	}
	return str
}

// getSliceField extracts a slice field from a map, failing the test if missing or wrong type
func getSliceField(t *testing.T, data map[string]interface{}, key string) []interface{} {
	t.Helper()
	val, ok := data[key]
	if !ok {
		t.Fatalf("response missing %s", key)
	}
	slice, ok := val.([]interface{})
	if !ok {
		t.Fatalf("expected %s to be []interface{}, got %T", key, val)
	}
	return slice
}

// getMapField extracts a map field from a map, failing the test if missing or wrong type
func getMapField(t *testing.T, data map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	val, ok := data[key]
	if !ok {
		t.Fatalf("response missing %s", key)
	}
	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("expected %s to be map[string]interface{}, got %T", key, val)
	}
	return m
}

// getBoolField extracts a bool field from a map, failing the test if missing or wrong type
func getBoolField(t *testing.T, data map[string]interface{}, key string) bool {
	t.Helper()
	val, ok := data[key]
	if !ok {
		t.Fatalf("response missing %s", key)
	}
	b, ok := val.(bool)
	if !ok {
		t.Fatalf("expected %s to be bool, got %T", key, val)
	}
	return b
}

// testLogSink is a test helper that logs to the test output
type testLogSink struct {
	t *testing.T
}

func (s *testLogSink) Write(ctx context.Context, entry executor.LogEntry) {
	s.t.Logf("[LOG] %s: %s", entry.Level, entry.Message)
}
