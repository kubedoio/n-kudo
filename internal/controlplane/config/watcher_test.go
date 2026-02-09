package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	// Test empty path
	_, err := NewWatcher("", func(path string, content []byte) error { return nil })
	if err == nil {
		t.Error("expected error for empty path")
	}

	// Test nil callback
	_, err = NewWatcher("/tmp/test.conf", nil)
	if err == nil {
		t.Error("expected error for nil callback")
	}

	// Test valid creation
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")
	if err := os.WriteFile(configPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	callback := func(path string, content []byte) error {
		return nil
	}

	w, err := NewWatcher(configPath, callback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected watcher to be created")
	}
	if w.interval != 30*time.Second {
		t.Errorf("expected default interval 30s, got %v", w.interval)
	}
}

func TestWithInterval(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")
	if err := os.WriteFile(configPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(configPath, func(path string, content []byte) error { return nil }, WithInterval(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if w.interval != 5*time.Second {
		t.Errorf("expected interval 5s, got %v", w.interval)
	}

	// Test minimum interval enforcement
	w2, err := NewWatcher(configPath, func(path string, content []byte) error { return nil }, WithInterval(500*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if w2.interval != time.Second {
		t.Errorf("expected minimum interval 1s, got %v", w2.interval)
	}
}

func TestWatcherCheck(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")
	if err := os.WriteFile(configPath, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	callCount := 0
	var lastContent []byte

	callback := func(path string, content []byte) error {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		lastContent = content
		return nil
	}

	w, err := NewWatcher(configPath, callback)
	if err != nil {
		t.Fatal(err)
	}

	// First check should not trigger callback (initial hash is already set)
	if err := w.check(); err != nil {
		t.Fatalf("check error: %v", err)
	}

	mu.Lock()
	if callCount != 0 {
		t.Errorf("expected callback not called on first check, got %d", callCount)
	}
	mu.Unlock()

	// Modify the file
	if err := os.WriteFile(configPath, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}

	// Check again - should trigger callback
	if err := w.check(); err != nil {
		t.Fatalf("check error: %v", err)
	}

	mu.Lock()
	if callCount != 1 {
		t.Errorf("expected callback called once, got %d", callCount)
	}
	if string(lastContent) != "changed" {
		t.Errorf("expected content 'changed', got %s", string(lastContent))
	}
	mu.Unlock()
}

func TestWatcherNoChange(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")
	if err := os.WriteFile(configPath, []byte("stable"), 0644); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	callCount := 0

	callback := func(path string, content []byte) error {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		return nil
	}

	w, err := NewWatcher(configPath, callback)
	if err != nil {
		t.Fatal(err)
	}

	// Check multiple times without modifying file
	for i := 0; i < 3; i++ {
		if err := w.check(); err != nil {
			t.Fatalf("check error: %v", err)
		}
	}

	mu.Lock()
	if callCount != 0 {
		t.Errorf("expected callback not called, got %d calls", callCount)
	}
	mu.Unlock()
}

func TestWatcherMultipleChanges(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")
	if err := os.WriteFile(configPath, []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var contents [][]byte

	callback := func(path string, content []byte) error {
		mu.Lock()
		defer mu.Unlock()
		contents = append(contents, content)
		return nil
	}

	w, err := NewWatcher(configPath, callback)
	if err != nil {
		t.Fatal(err)
	}

	// Make multiple changes
	os.WriteFile(configPath, []byte("v2"), 0644)
	w.check()

	os.WriteFile(configPath, []byte("v3"), 0644)
	w.check()

	mu.Lock()
	if len(contents) != 2 {
		t.Errorf("expected 2 changes, got %d", len(contents))
	}
	if len(contents) >= 1 && string(contents[0]) != "v2" {
		t.Errorf("expected first change 'v2', got %s", string(contents[0]))
	}
	if len(contents) >= 2 && string(contents[1]) != "v3" {
		t.Errorf("expected second change 'v3', got %s", string(contents[1]))
	}
	mu.Unlock()
}

func TestWatcherFileCreated(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "new.conf")

	var mu sync.Mutex
	callCount := 0

	callback := func(path string, content []byte) error {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		return nil
	}

	w, err := NewWatcher(configPath, callback)
	if err != nil {
		t.Fatal(err)
	}

	// File doesn't exist initially, check should not error
	if err := w.check(); err != nil {
		t.Fatalf("check error: %v", err)
	}

	mu.Lock()
	if callCount != 0 {
		t.Errorf("expected callback not called when file doesn't exist, got %d", callCount)
	}
	mu.Unlock()

	// Create the file
	if err := os.WriteFile(configPath, []byte("created"), 0644); err != nil {
		t.Fatal(err)
	}

	// Check again - should trigger callback
	if err := w.check(); err != nil {
		t.Fatalf("check error: %v", err)
	}

	mu.Lock()
	if callCount != 1 {
		t.Errorf("expected callback called once after file creation, got %d", callCount)
	}
	mu.Unlock()
}

func TestCalculateHash(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")
	content := []byte("test content for hashing")
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(configPath, func(path string, content []byte) error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	hash1, err := w.calculateHash()
	if err != nil {
		t.Fatal(err)
	}
	if hash1 == "" {
		t.Error("expected non-empty hash")
	}

	// Same content should produce same hash
	hash2, err := w.calculateHash()
	if err != nil {
		t.Fatal(err)
	}
	if hash1 != hash2 {
		t.Error("expected same hash for same content")
	}

	// Different content should produce different hash
	if err := os.WriteFile(configPath, []byte("different content"), 0644); err != nil {
		t.Fatal(err)
	}
	hash3, err := w.calculateHash()
	if err != nil {
		t.Fatal(err)
	}
	if hash1 == hash3 {
		t.Error("expected different hash for different content")
	}
}

func TestCalculateHashNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.conf")

	w, err := NewWatcher(configPath, func(path string, content []byte) error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.calculateHash()
	if !os.IsNotExist(err) {
		t.Errorf("expected file not found error, got %v", err)
	}
}

func TestSimpleCallback(t *testing.T) {
	callback := SimpleCallback()
	if callback == nil {
		t.Fatal("expected non-nil callback")
	}

	// Should not error
	if err := callback("/tmp/test", []byte("test")); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLastHash(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")
	if err := os.WriteFile(configPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(configPath, func(path string, content []byte) error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	initialHash := w.LastHash()
	if initialHash == "" {
		t.Error("expected non-empty initial hash")
	}
}

func TestWatcherStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")
	if err := os.WriteFile(configPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(configPath, func(path string, content []byte) error { return nil }, WithInterval(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	// Start and stop should not panic
	ctx, cancel := context.WithCancel(t.Context())
	w.Start(ctx)

	// Let it run briefly
	time.Sleep(150 * time.Millisecond)

	cancel()
	w.Stop()
}

func TestWatcherCallbackError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")
	if err := os.WriteFile(configPath, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}

	expectedErr := errors.New("callback error")
	callback := func(path string, content []byte) error {
		return expectedErr
	}

	w, err := NewWatcher(configPath, callback)
	if err != nil {
		t.Fatal(err)
	}

	// Modify file and check
	os.WriteFile(configPath, []byte("changed"), 0644)
	err = w.check()
	if err == nil {
		t.Error("expected error from callback")
	}
}
