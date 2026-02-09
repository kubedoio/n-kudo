package enroll

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kubedoio/n-kudo/internal/edge/executor"
)

func TestResolveTokenMissing(t *testing.T) {
	_, err := ResolveToken(TokenSource{})
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "enrollment token is required") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestResolveTokenFileNotFound(t *testing.T) {
	_, err := ResolveToken(TokenSource{FilePath: "/nonexistent/path/token.txt"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "read token file") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestClientEnroll(t *testing.T) {
	expectedReq := EnrollRequest{
		EnrollmentToken: "test-token-123",
		AgentVersion:    "v1.0.0",
		RequestedHost:   "test-host",
		Labels:          map[string]string{"env": "test"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/enroll" {
			t.Errorf("expected /v1/enroll, got %s", r.URL.Path)
		}

		var req EnrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.EnrollmentToken != expectedReq.EnrollmentToken {
			t.Errorf("expected token %s, got %s", expectedReq.EnrollmentToken, req.EnrollmentToken)
		}

		resp := EnrollResponse{
			TenantID:             "tenant-1",
			SiteID:               "site-1",
			HostID:               "host-1",
			AgentID:              "agent-1",
			ClientCertificatePEM: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
			CACertificatePEM:     "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----",
			RefreshToken:         "refresh-123",
			HeartbeatEndpoint:    "/v1/heartbeat",
			HeartbeatIntervalSec: 60,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
	}

	req := EnrollRequest{
		EnrollmentToken: expectedReq.EnrollmentToken,
		AgentVersion:    expectedReq.AgentVersion,
		RequestedHost:   expectedReq.RequestedHost,
		Labels:          expectedReq.Labels,
		CSRPEM:          "mock-csr",
	}

	resp, err := client.Enroll(context.Background(), req)
	if err != nil {
		t.Fatalf("enroll failed: %v", err)
	}

	if resp.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", resp.TenantID)
	}
	if resp.SiteID != "site-1" {
		t.Errorf("expected site-1, got %s", resp.SiteID)
	}
	if resp.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", resp.AgentID)
	}
	if resp.RefreshToken != "refresh-123" {
		t.Errorf("expected refresh-123, got %s", resp.RefreshToken)
	}
}

func TestClientEnrollInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return response missing required fields
		resp := map[string]string{
			"tenant_id": "tenant-1",
			// Missing site_id and agent_id
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
	}

	req := EnrollRequest{
		EnrollmentToken: "test-token",
	}

	_, err := client.Enroll(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid response")
	}
	if !strings.Contains(err.Error(), "invalid enroll response") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestClientEnrollServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
	}

	req := EnrollRequest{
		EnrollmentToken: "test-token",
	}

	_, err := client.Enroll(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for server error")
	}
	if !strings.Contains(err.Error(), "enroll failed status=500") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestClientHeartbeat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/heartbeat" {
			t.Errorf("expected /v1/heartbeat, got %s", r.URL.Path)
		}

		var req HeartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.TenantID != "tenant-1" {
			t.Errorf("expected tenant-1, got %s", req.TenantID)
		}

		resp := HeartbeatResponse{
			NextHeartbeatSeconds: 30,
			PendingPlans:         []executor.Plan{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
	}

	req := HeartbeatRequest{
		TenantID: "tenant-1",
		SiteID:   "site-1",
		HostID:   "host-1",
		AgentID:  "agent-1",
	}

	resp, err := client.Heartbeat(context.Background(), req)
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	if resp.NextHeartbeatSeconds != 30 {
		t.Errorf("expected 30 seconds, got %d", resp.NextHeartbeatSeconds)
	}
}

func TestClientFetchPlans(t *testing.T) {
	expectedSiteID := "site-1"
	expectedAgentID := "agent-1"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the path starts correctly
		if !strings.HasPrefix(r.URL.Path, "/v1/plans/next") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify query parameters
		querySiteID := r.URL.Query().Get("site_id")
		queryAgentID := r.URL.Query().Get("agent_id")
		if querySiteID != expectedSiteID {
			t.Errorf("expected site_id %s, got %s", expectedSiteID, querySiteID)
		}
		if queryAgentID != expectedAgentID {
			t.Errorf("expected agent_id %s, got %s", expectedAgentID, queryAgentID)
		}

		resp := map[string]any{
			"plans": []map[string]any{
				{
					"plan_id": "plan-1",
					"actions": []map[string]any{
						{
							"action_id": "act-1",
							"type":      "MicroVMCreate",
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
	}

	plans, err := client.FetchPlans(context.Background(), expectedSiteID, expectedAgentID)
	if err != nil {
		t.Fatalf("fetch plans failed: %v", err)
	}

	if len(plans) != 1 {
		t.Errorf("expected 1 plan, got %d", len(plans))
	}
}

func TestClientReportPlanResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/executions/result" {
			t.Errorf("expected /v1/executions/result, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
	}

	result := executor.PlanResult{
		PlanID:      "plan-1",
		ExecutionID: "exec-1",
		Results: []executor.ActionResult{
			{
				ExecutionID: "exec-1",
				ActionID:    "act-1",
				OK:          true,
				Message:     "success",
			},
		},
	}

	err := client.ReportPlanResult(context.Background(), result)
	if err != nil {
		t.Fatalf("report plan result failed: %v", err)
	}
}

func TestClientStreamLog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/logs" {
			t.Errorf("expected /v1/logs, got %s", r.URL.Path)
		}

		var entry LogEntry
		if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
			t.Fatalf("failed to decode log entry: %v", err)
		}

		if entry.ExecutionID != "exec-1" {
			t.Errorf("expected exec-1, got %s", entry.ExecutionID)
		}
		if entry.Level != "INFO" {
			t.Errorf("expected INFO, got %s", entry.Level)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
	}

	entry := LogEntry{
		ExecutionID: "exec-1",
		Level:       "INFO",
		Message:     "Test log message",
	}

	err := client.StreamLog(context.Background(), entry)
	if err != nil {
		t.Fatalf("stream log failed: %v", err)
	}
}

func TestClientStreamLogMissingExecutionID(t *testing.T) {
	client := &Client{BaseURL: "http://localhost"}

	entry := LogEntry{
		Level:   "INFO",
		Message: "Test log message",
	}

	err := client.StreamLog(context.Background(), entry)
	if err == nil {
		t.Fatal("expected error for missing execution_id")
	}
	if !strings.Contains(err.Error(), "execution_id required") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestNewNonce(t *testing.T) {
	nonce1 := NewNonce()
	nonce2 := NewNonce()

	if nonce1 == "" {
		t.Error("expected non-empty nonce")
	}

	if nonce1 == nonce2 {
		t.Error("expected unique nonces")
	}

	// Should be a valid Unix timestamp
	_, err := fmt.Sscanf(nonce1, "%d", new(int64))
	if err != nil {
		t.Errorf("expected nonce to be a valid integer: %v", err)
	}
}

func TestBuildFingerprint(t *testing.T) {
	fp := BuildFingerprint()

	// Either MachineID or PrimaryMAC should be set (or both)
	if fp.MachineIDSHA256 == "" && fp.PrimaryMACSHA256 == "" {
		t.Error("expected at least one fingerprint component to be set")
	}

	// If set, should be valid hex strings (64 characters for SHA256)
	if fp.MachineIDSHA256 != "" {
		if len(fp.MachineIDSHA256) != 64 {
			t.Errorf("expected 64 char hex string for machine ID, got %d", len(fp.MachineIDSHA256))
		}
	}

	if fp.PrimaryMACSHA256 != "" {
		if len(fp.PrimaryMACSHA256) != 64 {
			t.Errorf("expected 64 char hex string for MAC, got %d", len(fp.PrimaryMACSHA256))
		}
	}
}

func TestGenerateCSR(t *testing.T) {
	// Generate a test private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	// Create a CSR template
	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "test-agent",
			Organization: []string{"Test Org"},
		},
		DNSNames:    []string{"localhost"},
		IPAddresses: nil,
	}

	// Generate the CSR
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		t.Fatalf("failed to create CSR: %v", err)
	}

	// Encode to PEM
	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})

	if csrPEM == nil {
		t.Fatal("failed to encode CSR to PEM")
	}

	// Verify it starts with the expected header
	if !bytes.HasPrefix(csrPEM, []byte("-----BEGIN CERTIFICATE REQUEST-----")) {
		t.Error("CSR PEM does not have expected header")
	}

	// Parse it back to verify it's valid
	block, _ := pem.Decode(csrPEM)
	if block == nil {
		t.Fatal("failed to decode CSR PEM")
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse CSR: %v", err)
	}

	if csr.Subject.CommonName != "test-agent" {
		t.Errorf("expected CommonName 'test-agent', got %s", csr.Subject.CommonName)
	}
}

func TestClientNextSequence(t *testing.T) {
	client := &Client{}

	seq1 := client.NextSequence()
	seq2 := client.NextSequence()
	seq3 := client.NextSequence()

	if seq1 != 1 {
		t.Errorf("expected first sequence to be 1, got %d", seq1)
	}
	if seq2 != 2 {
		t.Errorf("expected second sequence to be 2, got %d", seq2)
	}
	if seq3 != 3 {
		t.Errorf("expected third sequence to be 3, got %d", seq3)
	}
}
