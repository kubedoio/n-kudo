package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// ChangeCallback is called when a config file change is detected
type ChangeCallback func(path string, content []byte) error

// Watcher polls a configuration file for changes and invokes a callback
// when the file content has changed
type Watcher struct {
	path      string
	interval  time.Duration
	onChange  ChangeCallback
	lastHash  string
	stopCh    chan struct{}
	wg        sync.WaitGroup
	mu        sync.RWMutex
}

// WatcherOption is a functional option for configuring the Watcher
type WatcherOption func(*Watcher)

// WithInterval sets the polling interval for the watcher
func WithInterval(interval time.Duration) WatcherOption {
	return func(w *Watcher) {
		w.interval = interval
	}
}

// NewWatcher creates a new config file watcher
func NewWatcher(path string, onChange ChangeCallback, opts ...WatcherOption) (*Watcher, error) {
	if path == "" {
		return nil, fmt.Errorf("config file path is required")
	}
	if onChange == nil {
		return nil, fmt.Errorf("onChange callback is required")
	}

	w := &Watcher{
		path:     path,
		interval: 30 * time.Second, // default interval
		onChange: onChange,
		stopCh:   make(chan struct{}),
	}

	for _, opt := range opts {
		opt(w)
	}

	if w.interval < time.Second {
		w.interval = time.Second // minimum 1 second interval
	}

	// Calculate initial hash
	hash, err := w.calculateHash()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("initial hash calculation: %w", err)
	}
	w.lastHash = hash

	return w, nil
}

// Start begins watching the config file for changes
func (w *Watcher) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.watch(ctx)
	log.Printf("config watcher started for %s (interval: %v)", w.path, w.interval)
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	log.Printf("config watcher stopped for %s", w.path)
}

// LastHash returns the last known hash of the config file
func (w *Watcher) LastHash() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastHash
}

func (w *Watcher) watch(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			if err := w.check(); err != nil {
				log.Printf("config watcher error: %v", err)
			}
		}
	}
}

func (w *Watcher) check() error {
	currentHash, err := w.calculateHash()
	if err != nil {
		// If file doesn't exist, treat as no change if we had a hash before
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	w.mu.Lock()
	lastHash := w.lastHash
	w.mu.Unlock()

	if currentHash == lastHash {
		return nil // no change
	}

	// File has changed
	content, err := os.ReadFile(w.path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	log.Printf("config file changed: %s", w.path)

	if err := w.onChange(w.path, content); err != nil {
		return fmt.Errorf("onChange callback: %w", err)
	}

	w.mu.Lock()
	w.lastHash = currentHash
	w.mu.Unlock()

	return nil
}

func (w *Watcher) calculateHash() (string, error) {
	f, err := os.Open(w.path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// HotReloadableConfig is an interface for configurations that can be hot-reloaded
type HotReloadableConfig interface {
	Reload(content []byte) error
}

// SimpleCallback creates a simple callback that logs changes
func SimpleCallback() ChangeCallback {
	return func(path string, content []byte) error {
		log.Printf("config file %s changed (%d bytes)", path, len(content))
		return nil
	}
}
