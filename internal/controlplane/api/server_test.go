package controlplane

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kubedoio/n-kudo/internal/controlplane/db"
)

func TestEnrollmentHappyPath(t *testing.T) {
	app, repo, _, _, enrollToken := newTestAppWithEnrollmentToken(t)

	csrPEM := makeCSR(t)
	payload := map[string]any{
		"enrollment_token": enrollToken,
		"hostname":         "edge-host-1",
		"agent_version":    "0.1.0",
		"os":               "linux",
		"arch":             "amd64",
		"csr_pem":          string(csrPEM),
	}
	rec := doJSON(t, app.Handler(), "POST", "/enroll", "", payload, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	mustDecode(t, rec.Body.Bytes(), &resp)
	agentID, _ := resp["agent_id"].(string)
	if agentID == "" {
		t.Fatalf("missing agent_id in response: %s", rec.Body.String())
	}
	if _, err := repo.GetAgentByID(context.Background(), agentID); err != nil {
		t.Fatalf("agent not persisted: %v", err)
	}

	// one-time token must fail on reuse
	rec2 := doJSON(t, app.Handler(), "POST", "/enroll", "", payload, nil)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on token reuse, got %d", rec2.Code)
	}
}

func TestEnrollmentV1RequestedHostname(t *testing.T) {
	app, repo, _, _, enrollToken := newTestAppWithEnrollmentToken(t)

	csrPEM := makeCSR(t)
	payload := map[string]any{
		"enrollment_token":   enrollToken,
		"requested_hostname": "edge-host-v1",
		"agent_version":      "0.1.0",
		"csr_pem":            string(csrPEM),
		"labels":             map[string]string{"os": "linux", "arch": "amd64"},
		"bootstrap_nonce":    "nonce-1",
	}
	rec := doJSON(t, app.Handler(), "POST", "/v1/enroll", "", payload, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	mustDecode(t, rec.Body.Bytes(), &resp)
	agentID, _ := resp["agent_id"].(string)
	if agentID == "" {
		t.Fatalf("missing agent_id in response: %s", rec.Body.String())
	}
	if _, err := repo.GetAgentByID(context.Background(), agentID); err != nil {
		t.Fatalf("agent not persisted: %v", err)
	}
}

func TestHeartbeatIngest(t *testing.T) {
	app, repo, tenantID, siteID, enrollToken := newTestAppWithEnrollmentToken(t)
	plainAPIKey := "nk_test_key"
	_, err := repo.CreateAPIKey(context.Background(), store.APIKey{ID: uuid.NewString(), TenantID: tenantID, Name: "dashboard", KeyHash: hashString(plainAPIKey)})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}

	csrPEM := makeCSR(t)
	enrollResp := enroll(t, app, enrollToken, csrPEM)
	agentID := enrollResp["agent_id"].(string)
	certPEM := enrollResp["client_certificate_pem"].(string)
	cert := parseCert(t, []byte(certPEM))

	hbPayload := map[string]any{
		"agent_id":                   agentID,
		"heartbeat_seq":              1,
		"hostname":                   "edge-host-1",
		"agent_version":              "0.1.0",
		"os":                         "linux",
		"arch":                       "amd64",
		"cpu_cores_total":            8,
		"memory_bytes_total":         int64(8 * 1024 * 1024 * 1024),
		"storage_bytes_total":        int64(100 * 1024 * 1024 * 1024),
		"kvm_available":              true,
		"cloud_hypervisor_available": true,
	}
	rec := doJSON(t, app.Handler(), "POST", "/agents/heartbeat", "", hbPayload, &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}})
	if rec.Code != http.StatusOK {
		t.Fatalf("heartbeat status=%d body=%s", rec.Code, rec.Body.String())
	}

	hosts, err := repo.ListHosts(context.Background(), tenantID, siteID)
	if err != nil {
		t.Fatalf("list hosts: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].CPUCoresTotal != 8 || !hosts[0].KVMAvailable {
		t.Fatalf("host facts not ingested: %+v", hosts[0])
	}
}

func TestHeartbeatV1HostFactsCompatibility(t *testing.T) {
	app, repo, tenantID, siteID, enrollToken := newTestAppWithEnrollmentToken(t)
	plainAPIKey := "nk_test_key"
	_, err := repo.CreateAPIKey(context.Background(), store.APIKey{ID: uuid.NewString(), TenantID: tenantID, Name: "dashboard", KeyHash: hashString(plainAPIKey)})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}

	csrPEM := makeCSR(t)
	enrollResp := enroll(t, app, enrollToken, csrPEM)
	agentID := enrollResp["agent_id"].(string)
	certPEM := enrollResp["client_certificate_pem"].(string)
	cert := parseCert(t, []byte(certPEM))

	hbPayload := map[string]any{
		"agent_id":   agentID,
		"tenant_id":  tenantID,
		"site_id":    siteID,
		"host_id":    "host-v1-compat",
		"sent_at":    time.Now().UTC().Format(time.RFC3339Nano),
		"extra_root": "ignored",
		"host_facts": map[string]any{
			"cpu_cores":          4,
			"memory_total_bytes": int64(4 * 1024 * 1024 * 1024),
			"disks": []map[string]any{
				{"mountpoint": "/", "total_bytes": int64(80 * 1024 * 1024 * 1024)},
			},
			"os":     "linux",
			"arch":   "amd64",
			"kernel": "6.8.0",
			"kvm": map[string]any{
				"present":  true,
				"readable": true,
				"writable": true,
			},
		},
		"microvms": []map[string]any{
			{
				"id":         "vm-compat-1",
				"name":       "vm-compat-1",
				"status":     "RUNNING",
				"updated_at": time.Now().UTC().Format(time.RFC3339Nano),
				"ignored":    true,
			},
		},
		"netbird_status": map[string]any{
			"connected": true,
		},
	}
	rec := doJSON(t, app.Handler(), "POST", "/v1/heartbeat", "", hbPayload, &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}})
	if rec.Code != http.StatusOK {
		t.Fatalf("heartbeat status=%d body=%s", rec.Code, rec.Body.String())
	}

	hosts, err := repo.ListHosts(context.Background(), tenantID, siteID)
	if err != nil {
		t.Fatalf("list hosts: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].CPUCoresTotal != 4 {
		t.Fatalf("host facts not ingested from host_facts payload: %+v", hosts[0])
	}
	vms, err := repo.ListVMs(context.Background(), tenantID, siteID)
	if err != nil {
		t.Fatalf("list vms: %v", err)
	}
	if len(vms) != 1 || vms[0].State != "RUNNING" {
		t.Fatalf("expected RUNNING vm from compat payload, got: %+v", vms)
	}
}

func TestPlanSubmissionAndLogs(t *testing.T) {
	app, repo, tenantID, siteID, enrollToken := newTestAppWithEnrollmentToken(t)
	plainAPIKey := "nk_test_key"
	_, err := repo.CreateAPIKey(context.Background(), store.APIKey{ID: uuid.NewString(), TenantID: tenantID, Name: "dashboard", KeyHash: hashString(plainAPIKey)})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}

	csrPEM := makeCSR(t)
	enrollResp := enroll(t, app, enrollToken, csrPEM)
	agentID := enrollResp["agent_id"].(string)
	certPEM := enrollResp["client_certificate_pem"].(string)
	cert := parseCert(t, []byte(certPEM))

	planPayload := map[string]any{
		"idempotency_key": "plan-123",
		"actions": []map[string]any{
			{"operation": "CREATE", "name": "vm-1", "vcpu_count": 2, "memory_mib": 512},
		},
	}
	rec := doJSON(t, app.Handler(), "POST", "/sites/"+siteID+"/plans", plainAPIKey, planPayload, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("apply plan status=%d body=%s", rec.Code, rec.Body.String())
	}
	var planResp struct {
		Executions []store.Execution `json:"executions"`
	}
	mustDecode(t, rec.Body.Bytes(), &planResp)
	if len(planResp.Executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(planResp.Executions))
	}
	execID := planResp.Executions[0].ID

	logsPayload := map[string]any{
		"agent_id": agentID,
		"entries": []map[string]any{
			{"execution_id": execID, "sequence": 1, "severity": "INFO", "message": "start", "emitted_at": time.Now().UTC().Format(time.RFC3339Nano)},
			{"execution_id": execID, "sequence": 2, "severity": "INFO", "message": "done", "emitted_at": time.Now().UTC().Format(time.RFC3339Nano)},
			{"execution_id": execID, "sequence": 2, "severity": "INFO", "message": "dup", "emitted_at": time.Now().UTC().Format(time.RFC3339Nano)},
		},
	}
	logRec := doJSON(t, app.Handler(), "POST", "/agents/logs", "", logsPayload, &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}})
	if logRec.Code != http.StatusOK {
		t.Fatalf("ingest logs status=%d body=%s", logRec.Code, logRec.Body.String())
	}
	var ingestResp map[string]any
	mustDecode(t, logRec.Body.Bytes(), &ingestResp)
	if ingestResp["accepted_frames"].(float64) != 2 {
		t.Fatalf("expected accepted_frames=2, got %v", ingestResp["accepted_frames"])
	}
	if ingestResp["dropped_frames"].(float64) != 1 {
		t.Fatalf("expected dropped_frames=1, got %v", ingestResp["dropped_frames"])
	}

	logsRec := doJSON(t, app.Handler(), "GET", "/executions/"+execID+"/logs", plainAPIKey, nil, nil)
	if logsRec.Code != http.StatusOK {
		t.Fatalf("list logs status=%d body=%s", logsRec.Code, logsRec.Body.String())
	}
	var logsResp struct {
		Logs []store.ExecutionLog `json:"logs"`
	}
	mustDecode(t, logsRec.Body.Bytes(), &logsResp)
	if len(logsResp.Logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logsResp.Logs))
	}
}

func TestStrictDecodeStillEnforcedForAdminEndpoints(t *testing.T) {
	app, repo, tenantID, _, _ := newTestAppWithEnrollmentToken(t)
	plainAPIKey := "nk_test_key"
	_, err := repo.CreateAPIKey(context.Background(), store.APIKey{ID: uuid.NewString(), TenantID: tenantID, Name: "dashboard", KeyHash: hashString(plainAPIKey)})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}

	rec := doJSON(t, app.Handler(), "POST", "/tenants/"+tenantID+"/sites", plainAPIKey, map[string]any{
		"name":        "strict-site",
		"extra_field": "must-fail",
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected strict decoder 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHeartbeatPlanDeliveryAndResultPersistence(t *testing.T) {
	app, repo, tenantID, siteID, enrollToken := newTestAppWithEnrollmentToken(t)
	plainAPIKey := "nk_test_key"
	_, err := repo.CreateAPIKey(context.Background(), store.APIKey{ID: uuid.NewString(), TenantID: tenantID, Name: "dashboard", KeyHash: hashString(plainAPIKey)})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}

	csrPEM := makeCSR(t)
	enrollResp := enroll(t, app, enrollToken, csrPEM)
	agentID := enrollResp["agent_id"].(string)
	certPEM := enrollResp["client_certificate_pem"].(string)
	cert := parseCert(t, []byte(certPEM))

	planPayload := map[string]any{
		"idempotency_key": "plan-run-once",
		"actions": []map[string]any{
			{
				"operation_id": "create-vm-1",
				"operation":    "CREATE",
				"vm_id":        "vm-runonce-1",
				"name":         "vm-runonce-1",
				"vcpu_count":   1,
				"memory_mib":   256,
			},
			{
				"operation_id": "start-vm-1",
				"operation":    "START",
				"vm_id":        "vm-runonce-1",
			},
		},
	}
	applyRec := doJSON(t, app.Handler(), "POST", "/sites/"+siteID+"/plans", plainAPIKey, planPayload, nil)
	if applyRec.Code != http.StatusOK {
		t.Fatalf("apply plan status=%d body=%s", applyRec.Code, applyRec.Body.String())
	}
	var applyResp struct {
		PlanID     string            `json:"plan_id"`
		Executions []store.Execution `json:"executions"`
	}
	mustDecode(t, applyRec.Body.Bytes(), &applyResp)
	if len(applyResp.Executions) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(applyResp.Executions))
	}

	hbPayload := map[string]any{
		"agent_id":        agentID,
		"heartbeat_seq":   1,
		"tenant_id":       tenantID,
		"site_id":         siteID,
		"host_id":         "host-runonce",
		"sent_at":         time.Now().UTC().Format(time.RFC3339Nano),
		"extra_heartbeat": "ignored",
		"host_facts": map[string]any{
			"cpu_cores":          2,
			"memory_total_bytes": int64(2 * 1024 * 1024 * 1024),
			"disks": []map[string]any{
				{"mountpoint": "/", "total_bytes": int64(50 * 1024 * 1024 * 1024)},
			},
			"os":     "linux",
			"arch":   "amd64",
			"kernel": "6.8.0",
			"kvm": map[string]any{
				"present":  true,
				"readable": true,
				"writable": true,
			},
		},
		"netbird_status": map[string]any{"connected": true},
	}
	hbRec := doJSON(t, app.Handler(), "POST", "/v1/heartbeat", "", hbPayload, &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}})
	if hbRec.Code != http.StatusOK {
		t.Fatalf("heartbeat status=%d body=%s", hbRec.Code, hbRec.Body.String())
	}
	var hbResp struct {
		PendingPlans []struct {
			PlanID      string `json:"plan_id"`
			ExecutionID string `json:"execution_id"`
			Actions     []struct {
				ActionID string `json:"action_id"`
				Type     string `json:"type"`
			} `json:"actions"`
		} `json:"pending_plans"`
	}
	mustDecode(t, hbRec.Body.Bytes(), &hbResp)
	if len(hbResp.PendingPlans) != 1 {
		t.Fatalf("expected 1 pending plan, got %d", len(hbResp.PendingPlans))
	}
	if hbResp.PendingPlans[0].PlanID != applyResp.PlanID {
		t.Fatalf("expected pending plan id %s, got %s", applyResp.PlanID, hbResp.PendingPlans[0].PlanID)
	}
	if len(hbResp.PendingPlans[0].Actions) != 2 {
		t.Fatalf("expected 2 leased actions, got %d", len(hbResp.PendingPlans[0].Actions))
	}

	logRec := doJSON(t, app.Handler(), "POST", "/v1/logs", "", map[string]any{
		"execution_id": applyResp.Executions[0].ID,
		"sequence":     1,
		"level":        "INFO",
		"message":      "run once started",
		"emitted_at":   time.Now().UTC().Format(time.RFC3339Nano),
		"tenant_id":    tenantID,
		"action_id":    "create-vm-1",
	}, &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}})
	if logRec.Code != http.StatusAccepted {
		t.Fatalf("log frame status=%d body=%s", logRec.Code, logRec.Body.String())
	}

	results := make([]map[string]any, 0, len(hbResp.PendingPlans[0].Actions))
	for _, action := range hbResp.PendingPlans[0].Actions {
		results = append(results, map[string]any{
			"action_id":   action.ActionID,
			"ok":          true,
			"message":     "ok",
			"finished_at": time.Now().UTC().Format(time.RFC3339Nano),
		})
	}
	resultRec := doJSON(t, app.Handler(), "POST", "/v1/executions/result", "", map[string]any{
		"plan_id":      hbResp.PendingPlans[0].PlanID,
		"execution_id": hbResp.PendingPlans[0].ExecutionID,
		"results":      results,
	}, &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}})
	if resultRec.Code != http.StatusAccepted {
		t.Fatalf("report result status=%d body=%s", resultRec.Code, resultRec.Body.String())
	}

	dedupRec := doJSON(t, app.Handler(), "POST", "/sites/"+siteID+"/plans", plainAPIKey, planPayload, nil)
	if dedupRec.Code != http.StatusOK {
		t.Fatalf("dedupe plan status=%d body=%s", dedupRec.Code, dedupRec.Body.String())
	}
	var dedupResp struct {
		PlanStatus string `json:"plan_status"`
	}
	mustDecode(t, dedupRec.Body.Bytes(), &dedupResp)
	if dedupResp.PlanStatus != "SUCCEEDED" {
		t.Fatalf("expected final plan status SUCCEEDED, got %s", dedupResp.PlanStatus)
	}

	vmsRec := doJSON(t, app.Handler(), "GET", "/sites/"+siteID+"/vms", plainAPIKey, nil, nil)
	if vmsRec.Code != http.StatusOK {
		t.Fatalf("list vms status=%d body=%s", vmsRec.Code, vmsRec.Body.String())
	}
	var vmsResp struct {
		VMs []store.MicroVM `json:"vms"`
	}
	mustDecode(t, vmsRec.Body.Bytes(), &vmsResp)
	if len(vmsResp.VMs) == 0 {
		t.Fatalf("expected at least one vm")
	}

	logsRec := doJSON(t, app.Handler(), "GET", "/executions/"+applyResp.Executions[0].ID+"/logs", plainAPIKey, nil, nil)
	if logsRec.Code != http.StatusOK {
		t.Fatalf("list logs status=%d body=%s", logsRec.Code, logsRec.Body.String())
	}
	var logsResp struct {
		Logs []store.ExecutionLog `json:"logs"`
	}
	mustDecode(t, logsRec.Body.Bytes(), &logsResp)
	if len(logsResp.Logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logsResp.Logs))
	}
}

func newTestAppWithEnrollmentToken(t *testing.T) (*App, *store.MemoryRepo, string, string, string) {
	t.Helper()
	repo := store.NewMemoryRepo()
	cfg := LoadConfig()
	cfg.AdminKey = "admin"
	app, err := NewApp(cfg, repo)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	tenantID := uuid.NewString()
	siteID := uuid.NewString()
	_, err = repo.CreateTenant(context.Background(), store.Tenant{ID: tenantID, Slug: "acme", Name: "Acme", PrimaryRegion: "eu-central-1", RetentionDays: 30})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	_, err = repo.CreateSite(context.Background(), store.Site{ID: siteID, TenantID: tenantID, Name: "site-1"})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}
	enrollToken := "enroll-token-1"
	_, err = repo.IssueEnrollmentToken(context.Background(), store.EnrollmentToken{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		SiteID:    siteID,
		TokenHash: hashString(enrollToken),
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return app, repo, tenantID, siteID, enrollToken
}

func doJSON(t *testing.T, h http.Handler, method, path, apiKey string, body any, tlsState *tls.ConnectionState) *httptest.ResponseRecorder {
	t.Helper()
	var buf []byte
	if body != nil {
		var err error
		buf, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	req.TLS = tlsState
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func enroll(t *testing.T, app *App, enrollToken string, csrPEM []byte) map[string]any {
	t.Helper()
	rec := doJSON(t, app.Handler(), "POST", "/enroll", "", map[string]any{
		"enrollment_token": enrollToken,
		"hostname":         "edge-host-1",
		"agent_version":    "0.1.0",
		"os":               "linux",
		"arch":             "amd64",
		"csr_pem":          string(csrPEM),
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("enroll status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	mustDecode(t, rec.Body.Bytes(), &resp)
	return resp
}

func mustDecode(t *testing.T, b []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("decode json: %v body=%s", err, string(b))
	}
}

func makeCSR(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tpl := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: "pending-agent",
		},
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, tpl, key)
	if err != nil {
		t.Fatalf("create csr: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})
}

func parseCert(t *testing.T, certPEM []byte) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatalf("failed to decode cert pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	if cert.SerialNumber.Cmp(big.NewInt(0)) <= 0 {
		t.Fatalf("invalid serial")
	}
	return cert
}
