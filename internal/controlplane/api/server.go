package controlplane

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kubedoio/n-kudo/internal/controlplane/db"
)

type App struct {
	cfg        Config
	repo       store.Repo
	ca         *InternalCA
	serverCert tls.Certificate
	mux        *http.ServeMux
}

func NewApp(cfg Config, repo store.Repo) (*App, error) {
	ca, err := LoadOrCreateInternalCA(cfg.CACommonName)
	if err != nil {
		return nil, err
	}
	serverCert, err := GenerateServerTLSCert()
	if err != nil {
		return nil, err
	}
	a := &App{cfg: cfg, repo: repo, ca: ca, serverCert: serverCert, mux: http.NewServeMux()}
	a.registerRoutes()
	return a, nil
}

func (a *App) Handler() http.Handler {
	return a.withRequestLogging(a.mux)
}

func (a *App) TLSConfig() (*tls.Config, error) {
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(a.ca.CertPEM()); !ok {
		return nil, errors.New("failed to parse internal ca pem")
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{a.serverCert},
		ClientCAs:    pool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
	}, nil
}

func (a *App) registerRoutes() {
	a.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC()})
	})
	a.mux.Handle("POST /tenants", a.adminAuth(http.HandlerFunc(a.handleCreateTenant)))
	a.mux.Handle("POST /tenants/{tenantID}/api-keys", a.adminAuth(http.HandlerFunc(a.handleCreateAPIKey)))

	a.mux.Handle("POST /tenants/{tenantID}/sites", a.apiKeyAuth(http.HandlerFunc(a.handleCreateSite)))
	a.mux.Handle("GET /tenants/{tenantID}/sites", a.apiKeyAuth(http.HandlerFunc(a.handleListSites)))
	a.mux.Handle("POST /tenants/{tenantID}/enrollment-tokens", a.apiKeyAuth(http.HandlerFunc(a.handleIssueEnrollmentToken)))

	a.mux.HandleFunc("POST /enroll", a.handleEnroll)
	a.mux.HandleFunc("POST /v1/enroll", a.handleEnroll)
	a.mux.Handle("POST /agents/heartbeat", a.agentMTLSAuth(http.HandlerFunc(a.handleHeartbeat)))
	a.mux.Handle("POST /v1/heartbeat", a.agentMTLSAuth(http.HandlerFunc(a.handleHeartbeat)))
	a.mux.Handle("POST /agents/logs", a.agentMTLSAuth(http.HandlerFunc(a.handleIngestLogs)))
	a.mux.Handle("POST /v1/logs", a.agentMTLSAuth(http.HandlerFunc(a.handleIngestLogFrame)))
	a.mux.Handle("GET /v1/plans/next", a.agentMTLSAuth(http.HandlerFunc(a.handleListPendingPlansV1)))
	a.mux.Handle("POST /v1/executions/result", a.agentMTLSAuth(http.HandlerFunc(a.handleReportPlanResultV1)))

	a.mux.Handle("POST /sites/{siteID}/plans", a.apiKeyAuth(http.HandlerFunc(a.handleApplyPlan)))
	a.mux.Handle("GET /sites/{siteID}/hosts", a.apiKeyAuth(http.HandlerFunc(a.handleListHosts)))
	a.mux.Handle("GET /sites/{siteID}/vms", a.apiKeyAuth(http.HandlerFunc(a.handleListVMs)))
	a.mux.Handle("GET /executions/{executionID}/logs", a.apiKeyAuth(http.HandlerFunc(a.handleListExecutionLogs)))
}

func (a *App) withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func (a *App) adminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !secureEqual(strings.TrimSpace(r.Header.Get("X-Admin-Key")), strings.TrimSpace(a.cfg.AdminKey)) {
			writeError(w, http.StatusUnauthorized, "invalid admin key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) apiKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if apiKey == "" {
			writeError(w, http.StatusUnauthorized, "missing api key")
			return
		}
		h := hashString(apiKey)
		validation, err := a.repo.ValidateAPIKey(r.Context(), h)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, store.ErrUnauthorized) {
				status = http.StatusUnauthorized
			}
			writeError(w, status, "invalid api key")
			return
		}
		ctx := context.WithValue(r.Context(), ctxTenantID{}, validation.TenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) agentMTLSAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			writeError(w, http.StatusUnauthorized, "missing client certificate")
			return
		}
		cert := r.TLS.PeerCertificates[0]
		agentID := strings.TrimSpace(cert.Subject.CommonName)
		if agentID == "" {
			writeError(w, http.StatusUnauthorized, "invalid client certificate")
			return
		}
		agent, err := a.repo.GetAgentByID(r.Context(), agentID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unknown agent")
			return
		}
		if cert.SerialNumber.String() != agent.CertSerial {
			writeError(w, http.StatusUnauthorized, "certificate serial mismatch")
			return
		}
		ctx := context.WithValue(r.Context(), ctxAgent{}, agent)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type ctxTenantID struct{}
type ctxAgent struct{}

func (a *App) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Slug              string `json:"slug"`
		Name              string `json:"name"`
		PrimaryRegion     string `json:"primary_region"`
		DataRetentionDays int    `json:"data_retention_days"`
	}
	var req request
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Slug == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "slug and name are required")
		return
	}
	if req.PrimaryRegion == "" {
		req.PrimaryRegion = "eu-central-1"
	}
	if req.DataRetentionDays == 0 {
		req.DataRetentionDays = 30
	}
	tenant, err := a.repo.CreateTenant(r.Context(), store.Tenant{
		ID:            uuid.NewString(),
		Slug:          req.Slug,
		Name:          req.Name,
		PrimaryRegion: req.PrimaryRegion,
		RetentionDays: req.DataRetentionDays,
	})
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeError(w, http.StatusConflict, "tenant slug already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create tenant")
		return
	}
	_ = a.repo.WriteAudit(r.Context(), tenant.ID, "", "SYSTEM", "", "tenant.create", "tenant", tenant.ID, requestID(r), sourceIP(r), nil)
	writeJSON(w, http.StatusCreated, tenant)
}

func (a *App) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")
	if _, err := uuid.Parse(tenantID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant id")
		return
	}
	type request struct {
		Name             string `json:"name"`
		ExpiresInSeconds int64  `json:"expires_in_seconds"`
	}
	var req request
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "default"
	}
	plainKey, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate api key")
		return
	}
	plainKey = "nk_" + plainKey
	var exp *time.Time
	if req.ExpiresInSeconds > 0 {
		t := time.Now().UTC().Add(time.Duration(req.ExpiresInSeconds) * time.Second)
		exp = &t
	}
	created, err := a.repo.CreateAPIKey(r.Context(), store.APIKey{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		Name:      req.Name,
		KeyHash:   hashString(plainKey),
		ExpiresAt: exp,
	})
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeError(w, http.StatusConflict, "api key conflict")
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tenant not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create api key")
		return
	}
	_ = a.repo.WriteAudit(r.Context(), tenantID, "", "SYSTEM", "", "apikey.create", "api_key", created.ID, requestID(r), sourceIP(r), nil)
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         created.ID,
		"tenant_id":  created.TenantID,
		"name":       created.Name,
		"api_key":    plainKey,
		"expires_at": created.ExpiresAt,
	})
}

func (a *App) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")
	if !a.tenantAllowed(r.Context(), tenantID) {
		writeError(w, http.StatusForbidden, "tenant mismatch")
		return
	}
	type request struct {
		Name                string `json:"name"`
		ExternalKey         string `json:"external_key"`
		LocationCountryCode string `json:"location_country_code"`
	}
	var req request
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	site, err := a.repo.CreateSite(r.Context(), store.Site{
		ID:              uuid.NewString(),
		TenantID:        tenantID,
		Name:            req.Name,
		ExternalKey:     req.ExternalKey,
		LocationCountry: strings.ToUpper(req.LocationCountryCode),
	})
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeError(w, http.StatusConflict, "site already exists")
			return
		}
		if errors.Is(err, store.ErrUnauthorized) {
			writeError(w, http.StatusNotFound, "tenant not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create site")
		return
	}
	_ = a.repo.WriteAudit(r.Context(), tenantID, site.ID, "USER", "api-key", "site.create", "site", site.ID, requestID(r), sourceIP(r), nil)
	writeJSON(w, http.StatusCreated, site)
}

func (a *App) handleListSites(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")
	if !a.tenantAllowed(r.Context(), tenantID) {
		writeError(w, http.StatusForbidden, "tenant mismatch")
		return
	}
	sites, err := a.repo.ListSites(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sites")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sites": sites})
}

func (a *App) handleIssueEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")
	if !a.tenantAllowed(r.Context(), tenantID) {
		writeError(w, http.StatusForbidden, "tenant mismatch")
		return
	}
	type request struct {
		SiteID           string `json:"site_id"`
		ExpiresInSeconds int64  `json:"expires_in_seconds"`
	}
	var req request
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := uuid.Parse(req.SiteID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid site_id")
		return
	}
	ok, err := a.repo.SiteBelongsToTenant(r.Context(), req.SiteID, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "site lookup failed")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "site not found")
		return
	}
	ttl := a.cfg.DefaultTokenTTL
	if req.ExpiresInSeconds > 0 {
		ttl = time.Duration(req.ExpiresInSeconds) * time.Second
	}
	plainToken, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	expiresAt := time.Now().UTC().Add(ttl)
	issued, err := a.repo.IssueEnrollmentToken(r.Context(), store.EnrollmentToken{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		SiteID:    req.SiteID,
		TokenHash: hashString(plainToken),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	_ = a.repo.WriteAudit(r.Context(), tenantID, req.SiteID, "USER", "api-key", "enrollment_token.issue", "enrollment_token", issued.ID, requestID(r), sourceIP(r), nil)
	writeJSON(w, http.StatusCreated, map[string]any{
		"token_id":   issued.ID,
		"site_id":    issued.SiteID,
		"token":      plainToken,
		"expires_at": issued.ExpiresAt,
		"one_time":   true,
	})
}

func (a *App) handleEnroll(w http.ResponseWriter, r *http.Request) {
	type request struct {
		EnrollmentToken   string            `json:"enrollment_token"`
		AgentVersion      string            `json:"agent_version"`
		Hostname          string            `json:"hostname"`
		RequestedHostname string            `json:"requested_hostname"`
		OS                string            `json:"os"`
		Arch              string            `json:"arch"`
		KernelVersion     string            `json:"kernel_version"`
		CSRPEM            string            `json:"csr_pem"`
		Labels            map[string]string `json:"labels"`
		BootstrapNonce    string            `json:"bootstrap_nonce"`
	}
	var req request
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	hostname := firstNonEmpty(req.Hostname, req.RequestedHostname)
	if req.EnrollmentToken == "" || req.CSRPEM == "" || hostname == "" {
		writeError(w, http.StatusBadRequest, "enrollment_token, hostname and csr_pem are required")
		return
	}
	consume, err := a.repo.ConsumeEnrollmentToken(r.Context(), hashString(req.EnrollmentToken), time.Now().UTC())
	if err != nil {
		if errors.Is(err, store.ErrTokenInvalid) {
			writeError(w, http.StatusUnauthorized, "invalid or expired enrollment token")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to validate enrollment token")
		return
	}
	agentID := uuid.NewString()
	certPEM, certSerial, err := a.ca.SignAgentCSR([]byte(req.CSRPEM), agentID, consume.TenantID, consume.SiteID, a.cfg.AgentCertTTL)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid csr: %v", err))
		return
	}
	refreshToken, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}
	agent, err := a.repo.CreateAgentFromEnrollment(r.Context(), consume.TokenID, store.Agent{
		ID:               agentID,
		TenantID:         consume.TenantID,
		SiteID:           consume.SiteID,
		HostID:           uuid.NewString(),
		CertSerial:       certSerial,
		RefreshTokenHash: hashString(refreshToken),
		AgentVersion:     valueOr(req.AgentVersion, "mvp1"),
		OS:               valueOr(req.OS, "linux"),
		Arch:             valueOr(req.Arch, "amd64"),
		KernelVersion:    req.KernelVersion,
	}, hostname)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeError(w, http.StatusConflict, "agent already exists for host")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create agent")
		return
	}
	_ = a.repo.WriteAudit(r.Context(), agent.TenantID, agent.SiteID, "AGENT", agent.ID, "agent.enroll", "agent", agent.ID, requestID(r), sourceIP(r), nil)
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":                  agent.TenantID,
		"site_id":                    agent.SiteID,
		"host_id":                    agent.HostID,
		"agent_id":                   agent.ID,
		"client_certificate_pem":     string(certPEM),
		"ca_certificate_pem":         string(a.ca.CertPEM()),
		"refresh_token":              refreshToken,
		"heartbeat_endpoint":         "/agents/heartbeat",
		"heartbeat_interval_sec":     15,
		"heartbeat_interval_seconds": 15,
	})
}

func (a *App) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	agent := r.Context().Value(ctxAgent{}).(store.Agent)
	type hostFacts struct {
		CPUCores    int   `json:"cpu_cores"`
		MemoryTotal int64 `json:"memory_total_bytes"`
		Disks       []struct {
			Mountpoint string `json:"mountpoint"`
			TotalBytes int64  `json:"total_bytes"`
			FreeBytes  int64  `json:"free_bytes"`
		} `json:"disks"`
		OS     string `json:"os"`
		Arch   string `json:"arch"`
		Kernel string `json:"kernel"`
		KVM    struct {
			Present  bool `json:"present"`
			Readable bool `json:"readable"`
			Writable bool `json:"writable"`
		} `json:"kvm"`
	}
	type vmCompat struct {
		ID         string    `json:"id"`
		Name       string    `json:"name"`
		State      string    `json:"state"`
		Status     string    `json:"status"`
		VCPUCount  int       `json:"vcpu_count"`
		MemoryMiB  int64     `json:"memory_mib"`
		KernelPath string    `json:"kernel_path"`
		RootfsPath string    `json:"rootfs_path"`
		TapIface   string    `json:"tap_iface"`
		CHPID      int       `json:"ch_pid"`
		UpdatedAt  time.Time `json:"updated_at"`
	}
	type request struct {
		AgentID                  string                  `json:"agent_id"`
		HeartbeatSeq             int64                   `json:"heartbeat_seq"`
		AgentVersion             string                  `json:"agent_version"`
		OS                       string                  `json:"os"`
		Arch                     string                  `json:"arch"`
		KernelVersion            string                  `json:"kernel_version"`
		Hostname                 string                  `json:"hostname"`
		CPUCoresTotal            int                     `json:"cpu_cores_total"`
		MemoryBytesTotal         int64                   `json:"memory_bytes_total"`
		StorageBytesTotal        int64                   `json:"storage_bytes_total"`
		KVMAvailable             bool                    `json:"kvm_available"`
		CloudHypervisorAvailable bool                    `json:"cloud_hypervisor_available"`
		MicroVMs                 []vmCompat              `json:"microvms"`
		ExecutionUpdates         []store.ExecutionUpdate `json:"execution_updates"`
		HostFacts                hostFacts               `json:"host_facts"`
	}
	var req request
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.AgentID != "" && req.AgentID != agent.ID {
		writeError(w, http.StatusForbidden, "agent_id mismatch")
		return
	}
	if req.CPUCoresTotal == 0 && req.HostFacts.CPUCores > 0 {
		req.CPUCoresTotal = req.HostFacts.CPUCores
	}
	if req.MemoryBytesTotal == 0 && req.HostFacts.MemoryTotal > 0 {
		req.MemoryBytesTotal = req.HostFacts.MemoryTotal
	}
	if req.StorageBytesTotal == 0 && len(req.HostFacts.Disks) > 0 {
		var sum int64
		for _, d := range req.HostFacts.Disks {
			sum += d.TotalBytes
		}
		req.StorageBytesTotal = sum
	}
	if req.OS == "" {
		req.OS = req.HostFacts.OS
	}
	if req.Arch == "" {
		req.Arch = req.HostFacts.Arch
	}
	if req.KernelVersion == "" {
		req.KernelVersion = req.HostFacts.Kernel
	}
	if !req.KVMAvailable {
		req.KVMAvailable = req.HostFacts.KVM.Present && req.HostFacts.KVM.Readable && req.HostFacts.KVM.Writable
	}
	if !req.CloudHypervisorAvailable {
		req.CloudHypervisorAvailable = req.KVMAvailable
	}
	if req.Hostname == "" {
		req.Hostname = "unknown"
	}
	vms := make([]store.MicroVMHeartbeat, 0, len(req.MicroVMs))
	for _, vm := range req.MicroVMs {
		if strings.TrimSpace(vm.ID) == "" {
			continue
		}
		state := firstNonEmpty(vm.State, vm.Status)
		if state == "" {
			state = "CREATING"
		}
		vcpu := vm.VCPUCount
		if vcpu <= 0 {
			vcpu = 1
		}
		mem := vm.MemoryMiB
		if mem <= 0 {
			mem = 256
		}
		vms = append(vms, store.MicroVMHeartbeat{
			ID:        vm.ID,
			Name:      firstNonEmpty(vm.Name, vm.ID),
			State:     state,
			VCPUCount: vcpu,
			MemoryMiB: mem,
			UpdatedAt: vm.UpdatedAt,
		})
	}
	err := a.repo.IngestHeartbeat(r.Context(), store.Heartbeat{
		AgentID:                  agent.ID,
		HeartbeatSeq:             req.HeartbeatSeq,
		AgentVersion:             valueOr(req.AgentVersion, agent.AgentVersion),
		OS:                       valueOr(req.OS, agent.OS),
		Arch:                     valueOr(req.Arch, agent.Arch),
		KernelVersion:            valueOr(req.KernelVersion, agent.KernelVersion),
		Hostname:                 req.Hostname,
		CPUCoresTotal:            req.CPUCoresTotal,
		MemoryBytesTotal:         req.MemoryBytesTotal,
		StorageBytesTotal:        req.StorageBytesTotal,
		KVMAvailable:             req.KVMAvailable,
		CloudHypervisorAvailable: req.CloudHypervisorAvailable,
		MicroVMs:                 vms,
		ExecutionUpdates:         req.ExecutionUpdates,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to ingest heartbeat")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"next_heartbeat_seconds": 15, "pending_plans": []any{}})
}

func (a *App) handleIngestLogFrame(w http.ResponseWriter, r *http.Request) {
	agent := r.Context().Value(ctxAgent{}).(store.Agent)
	type request struct {
		ExecutionID string    `json:"execution_id"`
		Sequence    int64     `json:"sequence"`
		Level       string    `json:"level"`
		Message     string    `json:"message"`
		EmittedAt   time.Time `json:"emitted_at"`
	}
	var req request
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.ExecutionID) == "" {
		writeError(w, http.StatusBadRequest, "execution_id required")
		return
	}
	if req.Sequence <= 0 {
		req.Sequence = time.Now().UTC().UnixNano()
	}
	if req.EmittedAt.IsZero() {
		req.EmittedAt = time.Now().UTC()
	}
	_, _, err := a.repo.IngestLogs(r.Context(), store.LogIngest{
		AgentID: agent.ID,
		Entries: []store.LogIngestEntry{{
			ExecutionID: req.ExecutionID,
			Sequence:    req.Sequence,
			Severity:    req.Level,
			Message:     req.Message,
			EmittedAt:   req.EmittedAt,
		}},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to ingest logs")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (a *App) handleListPendingPlansV1(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"plans": []any{}})
}

func (a *App) handleReportPlanResultV1(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}

func (a *App) handleIngestLogs(w http.ResponseWriter, r *http.Request) {
	agent := r.Context().Value(ctxAgent{}).(store.Agent)
	type request struct {
		AgentID string                 `json:"agent_id"`
		Entries []store.LogIngestEntry `json:"entries"`
	}
	var req request
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.AgentID != "" && req.AgentID != agent.ID {
		writeError(w, http.StatusForbidden, "agent_id mismatch")
		return
	}
	accepted, dropped, err := a.repo.IngestLogs(r.Context(), store.LogIngest{AgentID: agent.ID, Entries: req.Entries})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to ingest logs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"accepted_frames": accepted, "dropped_frames": dropped})
}

func (a *App) handleApplyPlan(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value(ctxTenantID{}).(string)
	siteID := r.PathValue("siteID")
	ok, err := a.repo.SiteBelongsToTenant(r.Context(), siteID, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "site lookup failed")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "site not found")
		return
	}
	type request struct {
		IdempotencyKey  string                  `json:"idempotency_key"`
		ClientRequestID string                  `json:"client_request_id"`
		Actions         []store.ApplyPlanAction `json:"actions"`
	}
	var req request
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		writeError(w, http.StatusBadRequest, "idempotency_key is required")
		return
	}
	if len(req.Actions) == 0 {
		writeError(w, http.StatusBadRequest, "actions are required")
		return
	}
	result, err := a.repo.ApplyPlan(r.Context(), store.ApplyPlanInput{
		TenantID:        tenantID,
		SiteID:          siteID,
		IdempotencyKey:  req.IdempotencyKey,
		ClientRequestID: req.ClientRequestID,
		Actions:         req.Actions,
	})
	if err != nil {
		if errors.Is(err, store.ErrUnauthorized) {
			writeError(w, http.StatusForbidden, "tenant mismatch")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to apply plan")
		return
	}
	_ = a.repo.WriteAudit(r.Context(), tenantID, siteID, "USER", "api-key", "plan.apply", "plan", result.Plan.ID, requestID(r), sourceIP(r), result.Plan.OperationsJSON)
	writeJSON(w, http.StatusOK, map[string]any{
		"plan_id":      result.Plan.ID,
		"plan_version": result.Plan.PlanVersion,
		"plan_status":  result.Plan.Status,
		"deduplicated": result.Deduplicated,
		"executions":   result.Executions,
	})
}

func (a *App) handleListHosts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value(ctxTenantID{}).(string)
	siteID := r.PathValue("siteID")
	ok, err := a.repo.SiteBelongsToTenant(r.Context(), siteID, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "site lookup failed")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "site not found")
		return
	}
	hosts, err := a.repo.ListHosts(r.Context(), tenantID, siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list hosts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hosts": hosts})
}

func (a *App) handleListVMs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value(ctxTenantID{}).(string)
	siteID := r.PathValue("siteID")
	ok, err := a.repo.SiteBelongsToTenant(r.Context(), siteID, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "site lookup failed")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "site not found")
		return
	}
	vms, err := a.repo.ListVMs(r.Context(), tenantID, siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list microvms")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vms": vms})
}

func (a *App) handleListExecutionLogs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value(ctxTenantID{}).(string)
	executionID := r.PathValue("executionID")
	ok, err := a.repo.ExecutionBelongsToTenant(r.Context(), executionID, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "execution lookup failed")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "execution not found")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := a.repo.ListExecutionLogs(r.Context(), tenantID, executionID, limit)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "execution not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to list logs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs})
}

func (a *App) tenantAllowed(ctx context.Context, pathTenant string) bool {
	tenant, ok := ctx.Value(ctxTenantID{}).(string)
	return ok && tenant == pathTenant
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func decodeJSON(body io.Reader, v any) error {
	dec := json.NewDecoder(io.LimitReader(body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func secureEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func valueOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func requestID(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Request-ID")); v != "" {
		return v
	}
	return uuid.NewString()
}

func sourceIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return host
}
