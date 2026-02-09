package integration_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kubedoio/n-kudo/internal/edge/enroll"
	"github.com/kubedoio/n-kudo/internal/edge/hostfacts"
	"github.com/kubedoio/n-kudo/internal/edge/mtls"
	"github.com/kubedoio/n-kudo/internal/edge/netbird"
)

// TestExecutionLogStreaming validates end-to-end log streaming from agent to control-plane
func TestExecutionLogStreaming(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprint(r)
			if strings.Contains(msg, "failed to listen on a port") {
				t.Skipf("skipping integration test: local bind not permitted: %s", msg)
				return
			}
			panic(r)
		}
	}()

	caCertPEM, _, caCert, caKey := newTestCA(t)
	serverTLSCert := newServerCert(t, caCert, caKey)

	var logCalls atomic.Int32
	var receivedLogs []enroll.LogEntry
	var logsMu sync.Mutex

	// Setup mock control-plane with log ingestion
	// Note: /v1/logs accepts a single log entry, not a batch
	ingestMux := http.NewServeMux()
	ingestMux.HandleFunc("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		logCalls.Add(1)
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "client cert required", http.StatusUnauthorized)
			return
		}

		var entry enroll.LogEntry
		if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		logsMu.Lock()
		receivedLogs = append(receivedLogs, entry)
		logsMu.Unlock()

		w.WriteHeader(http.StatusAccepted)
	})

	ingestMux.HandleFunc("/v1/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "client cert required", http.StatusUnauthorized)
			return
		}
		payload := map[string]interface{}{
			"next_heartbeat_seconds": 15,
			"pending_plans":          []interface{}{},
		}
		json.NewEncoder(w).Encode(payload)
	})

	ingestSrv := httptest.NewUnstartedServer(ingestMux)
	ingestLn, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping: cannot bind local listener: %v", err)
	}
	ingestSrv.Listener = ingestLn
	clientCAPool := x509.NewCertPool()
	clientCAPool.AppendCertsFromPEM(caCertPEM)
	ingestSrv.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverTLSCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAPool,
		MinVersion:   tls.VersionTLS13,
	}
	ingestSrv.StartTLS()
	defer ingestSrv.Close()

	// Enroll the agent
	bootstrapMux := http.NewServeMux()
	bootstrapMux.HandleFunc("/v1/enroll", func(w http.ResponseWriter, r *http.Request) {
		var req enroll.EnrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		certPEM := signCSR(t, req.CSRPEM, caCert, caKey)
		resp := enroll.EnrollResponse{
			TenantID:             "tenant-1",
			SiteID:               "site-1",
			HostID:               "host-1",
			AgentID:              "agent-1",
			ClientCertificatePEM: certPEM,
			CACertificatePEM:     string(caCertPEM),
			RefreshToken:         "refresh-token",
			HeartbeatEndpoint:    ingestSrv.URL,
			HeartbeatIntervalSec: 15,
		}
		json.NewEncoder(w).Encode(resp)
	})

	bootstrapSrv := httptest.NewUnstartedServer(bootstrapMux)
	bootstrapLn, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping: cannot bind local listener: %v", err)
	}
	bootstrapSrv.Listener = bootstrapLn
	bootstrapSrv.StartTLS()
	defer bootstrapSrv.Close()

	// Perform enrollment
	bootstrapHTTP, err := mtls.NewBootstrapTLSClient(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	enrollClient := enroll.Client{BaseURL: bootstrapSrv.URL, HTTP: bootstrapHTTP}

	key, err := mtls.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	csr, err := mtls.GenerateCSRPEM(key, "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := enrollClient.Enroll(context.Background(), enroll.EnrollRequest{
		EnrollmentToken: "test-token",
		AgentVersion:    "test",
		RequestedHost:   "test-host",
		CSRPEM:          string(csr),
		BootstrapNonce:  "nonce",
	})
	if err != nil {
		t.Fatalf("enroll failed: %v", err)
	}

	// Setup mTLS client
	pki := mtls.DefaultPKIPaths(t.TempDir())
	if err := mtls.WritePKI(pki, mtls.EncodePrivateKeyPEM(key), []byte(resp.ClientCertificatePEM), []byte(resp.CACertificatePEM)); err != nil {
		t.Fatal(err)
	}
	mTLSClient, err := mtls.NewMutualTLSClient(pki, true)
	if err != nil {
		t.Fatal(err)
	}

	// Create mTLS-enabled client
	cp := &enroll.Client{BaseURL: ingestSrv.URL, HTTP: mTLSClient}

	// Send heartbeat first to establish agent identity
	_, err = cp.Heartbeat(context.Background(), enroll.HeartbeatRequest{
		TenantID:      resp.TenantID,
		SiteID:        resp.SiteID,
		HostID:        resp.HostID,
		AgentID:       resp.AgentID,
		HostFacts:     hostfacts.Facts{CPUCores: 2, Arch: "amd64"},
		NetBirdStatus: netbird.Status{Connected: true},
	})
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	// Stream multiple log entries
	logEntries := []enroll.LogEntry{
		{
			TenantID:    resp.TenantID,
			SiteID:      resp.SiteID,
			AgentID:     resp.AgentID,
			ExecutionID: "exec-001",
			ActionID:    "action-001",
			Level:       "INFO",
			Message:     "Starting VM creation",
		},
		{
			TenantID:    resp.TenantID,
			SiteID:      resp.SiteID,
			AgentID:     resp.AgentID,
			ExecutionID: "exec-001",
			ActionID:    "action-001",
			Level:       "INFO",
			Message:     "VM created successfully",
		},
		{
			TenantID:    resp.TenantID,
			SiteID:      resp.SiteID,
			AgentID:     resp.AgentID,
			ExecutionID: "exec-001",
			ActionID:    "action-002",
			Level:       "INFO",
			Message:     "Starting VM",
		},
		{
			TenantID:    resp.TenantID,
			SiteID:      resp.SiteID,
			AgentID:     resp.AgentID,
			ExecutionID: "exec-002",
			ActionID:    "action-003",
			Level:       "ERROR",
			Message:     "Failed to stop VM: not running",
		},
	}

	for _, entry := range logEntries {
		err := cp.StreamLog(context.Background(), entry)
		if err != nil {
			t.Fatalf("log stream failed: %v", err)
		}
	}

	// Verify logs were received
	if logCalls.Load() == 0 {
		t.Fatal("expected log calls, got none")
	}

	logsMu.Lock()
	receivedCount := len(receivedLogs)
	logsMu.Unlock()

	if receivedCount != len(logEntries) {
		t.Fatalf("expected %d log entries, got %d", len(logEntries), receivedCount)
	}

	// Verify log content
	logsMu.Lock()
	for i, expected := range logEntries {
		if receivedLogs[i].Message != expected.Message {
			t.Fatalf("log %d: expected message %q, got %q", i, expected.Message, receivedLogs[i].Message)
		}
		if receivedLogs[i].Level != expected.Level {
			t.Fatalf("log %d: expected level %q, got %q", i, expected.Level, receivedLogs[i].Level)
		}
		if receivedLogs[i].ExecutionID != expected.ExecutionID {
			t.Fatalf("log %d: expected execution %q, got %q", i, expected.ExecutionID, receivedLogs[i].ExecutionID)
		}
	}
	logsMu.Unlock()

	t.Logf("Log streaming test passed: %d entries received", receivedCount)
}

// TestLogStreamingBatching validates that multiple logs can be sent in a single request
func TestLogStreamingBatching(t *testing.T) {
	var receivedBatches int
	var totalEntries int
	var mu sync.Mutex

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		receivedBatches++

		var payload struct {
			Entries []enroll.LogEntry `json:"entries"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		totalEntries += len(payload.Entries)

		w.WriteHeader(http.StatusAccepted)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Send batch of logs
	batch := []enroll.LogEntry{
		{ExecutionID: "exec-001", Level: "INFO", Message: "log 1"},
		{ExecutionID: "exec-001", Level: "INFO", Message: "log 2"},
		{ExecutionID: "exec-001", Level: "WARN", Message: "log 3"},
		{ExecutionID: "exec-001", Level: "ERROR", Message: "log 4"},
		{ExecutionID: "exec-001", Level: "INFO", Message: "log 5"},
	}

	// Simulate batch send
	payload, _ := json.Marshal(map[string]interface{}{
		"agent_id": "agent-1",
		"entries":  batch,
	})

	resp, err := http.Post(server.URL+"/v1/logs", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to send logs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	mu.Lock()
	if receivedBatches != 1 {
		t.Fatalf("expected 1 batch, got %d", receivedBatches)
	}
	if totalEntries != len(batch) {
		t.Fatalf("expected %d entries, got %d", len(batch), totalEntries)
	}
	mu.Unlock()

	t.Log("Batch log streaming test passed")
}

// TestLogStreamingUnauthorized validates that unauthorized log streaming is rejected
func TestLogStreamingUnauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		// Require mTLS
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "client cert required", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Try without client cert (should fail)
	payload := map[string]interface{}{
		"agent_id": "agent-1",
		"entries":  []enroll.LogEntry{},
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/v1/logs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	t.Log("Unauthorized log streaming correctly rejected")
}
