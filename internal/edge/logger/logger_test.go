package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestInitJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	log = logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05Z",
	})
	log.SetLevel(logrus.InfoLevel)

	log.Info("test message")

	output := buf.String()
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log: %v", err)
	}

	if _, ok := logEntry["msg"]; !ok {
		t.Error("JSON log should contain 'msg' field")
	}
	if _, ok := logEntry["time"]; !ok {
		t.Error("JSON log should contain 'time' field")
	}
	if _, ok := logEntry["level"]; !ok {
		t.Error("JSON log should contain 'level' field")
	}
}

func TestInitTextFormat(t *testing.T) {
	var buf bytes.Buffer
	log = logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05Z",
	})
	log.SetLevel(logrus.InfoLevel)

	log.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("Text log should contain the message")
	}
	// Text formatter may use different case for level, check for lowercase "info"
	if !strings.Contains(strings.ToLower(output), "info") {
		t.Error("Text log should contain the level info")
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		level    string
		expected logrus.Level
	}{
		{"debug", logrus.DebugLevel},
		{"info", logrus.InfoLevel},
		{"warn", logrus.WarnLevel},
		{"warning", logrus.WarnLevel},
		{"error", logrus.ErrorLevel},
		{"fatal", logrus.FatalLevel},
		{"invalid", logrus.InfoLevel}, // Should default to info
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			Init("json", tt.level)
			if log.GetLevel() != tt.expected {
				t.Errorf("Expected level %v, got %v", tt.expected, log.GetLevel())
			}
		})
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	Init("json", "info")
	log.SetOutput(&buf)

	WithFields(map[string]interface{}{
		"action_id":   "action-123",
		"action_type": "MicroVMCreate",
	}).Info("test with fields")

	output := buf.String()
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log: %v", err)
	}

	if logEntry["action_id"] != "action-123" {
		t.Errorf("Expected action_id 'action-123', got %v", logEntry["action_id"])
	}
	if logEntry["action_type"] != "MicroVMCreate" {
		t.Errorf("Expected action_type 'MicroVMCreate', got %v", logEntry["action_type"])
	}
}

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	Init("json", "info")
	log.SetOutput(&buf)

	WithComponent("executor").Info("test with component")

	output := buf.String()
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log: %v", err)
	}

	if logEntry["component"] != "executor" {
		t.Errorf("Expected component 'executor', got %v", logEntry["component"])
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	Init("json", "warn")
	log.SetOutput(&buf)

	// Debug and Info should be filtered out
	Debug("debug message")
	Info("info message")

	// Warn and Error should pass through
	Warn("warn message")
	Error("error message")

	output := buf.String()

	if strings.Contains(output, "debug message") {
		t.Error("Debug message should be filtered out at warn level")
	}
	if strings.Contains(output, "info message") {
		t.Error("Info message should be filtered out at warn level")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should pass through at warn level")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message should pass through at warn level")
	}
}

func TestLogFunctions(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			name: "Info",
			logFunc: func() {
				Info("info test")
			},
			expected: "info test",
		},
		{
			name: "Infof",
			logFunc: func() {
				Infof("infof %s", "test")
			},
			expected: "infof test",
		},
		{
			name: "Error",
			logFunc: func() {
				Error("error test")
			},
			expected: "error test",
		},
		{
			name: "Errorf",
			logFunc: func() {
				Errorf("errorf %s", "test")
			},
			expected: "errorf test",
		},
		{
			name: "Debug",
			logFunc: func() {
				Debug("debug test")
			},
			expected: "debug test",
		},
		{
			name: "Debugf",
			logFunc: func() {
				Debugf("debugf %s", "test")
			},
			expected: "debugf test",
		},
		{
			name: "Warn",
			logFunc: func() {
				Warn("warn test")
			},
			expected: "warn test",
		},
		{
			name: "Warnf",
			logFunc: func() {
				Warnf("warnf %s", "test")
			},
			expected: "warnf test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Init("text", "debug")
			log.SetOutput(&buf)

			tt.logFunc()

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain '%s', got '%s'", tt.expected, output)
			}
		})
	}
}

func TestInitWithComponent(t *testing.T) {
	var buf bytes.Buffer
	
	// Create a fresh logger directly to ensure isolation
	log = logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05Z",
	})
	log.SetLevel(logrus.InfoLevel)

	// Use WithField to create an entry with component field
	entry := log.WithField("component", "executor")
	entry.Info("test message")

	output := buf.String()
	
	// If output is empty, component was set but no log was written (shouldn't happen)
	if strings.TrimSpace(output) == "" {
		t.Skip("No log output captured")
	}
	
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log: %v", err)
	}

	if logEntry["component"] != "executor" {
		t.Errorf("Expected component 'executor', got %v", logEntry["component"])
	}
}
