package controlplane

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
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
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/kubedoio/n-kudo/internal/controlplane/cache"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
	"github.com/kubedoio/n-kudo/internal/controlplane/health"
	sla "github.com/kubedoio/n-kudo/internal/controlplane/metrics"
	"github.com/kubedoio/n-kudo/internal/controlplane/pki"
	"github.com/kubedoio/n-kudo/internal/controlplane/tenant"
)

type App struct {
	cfg           Config
	repo          store.Repo
	ca            *InternalCA
	crlManager    *pki.CRLManager
	serverCert    tls.Certificate
	mux           *http.ServeMux
	cache         Cache
	healthChecker *health.Checker

	// Metrics counters
	metrics struct {
		requestsTotal   atomic.Int64
		enrollmentsTotal atomic.Int64
		heartbeatsTotal  atomic.Int64
		plansApplied     atomic.Int64
		executionsTotal  atomic.Int64
	}

	// Rate limiter
	rateLimiter *RateLimiter

	// Quota manager for tenant resource limits
	quotaManager *tenant.QuotaManager
}

// Cache interface for caching
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Flush()
}

func NewApp(cfg Config, repo store.Repo) (*App, error) {
	ca, err := LoadOrCreateInternalCA(cfg.CACommonName, cfg.RequirePersistentPKI)
	if err != nil {
		return nil, err
	}
	serverCert, err := GenerateServerTLSCert(cfg.RequirePersistentPKI)
	if err != nil {
		return nil, err
	}

	// Initialize CRL manager with CRL URL
	crlURL := env("CRL_URL", "")
	crlManager := pki.NewCRLManager(ca.Certificate(), ca.Key(), crlURL)

	// Load existing revoked certificates from database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	revokedCerts, err := repo.ListRevokedCertificates(ctx)
	cancel()
	if err != nil {
		log.Printf("warning: failed to load revoked certificates: %v", err)
	} else {
		for _, entry := range revokedCerts {
			if err := crlManager.Revoke(entry.SerialNumber, pki.RevocationReason(entry.Reason), entry.AgentID); err != nil {
				log.Printf("warning: failed to load revoked cert %s: %v", entry.SerialNumber, err)
			}
		}
		if len(revokedCerts) > 0 {
			log.Printf("loaded %d revoked certificates into CRL", len(revokedCerts))
		}
	}

	// Initialize cache with 5 minute default expiration and 10 minute cleanup
	appCache := cache.New(5*time.Minute, 10*time.Minute)

	a := &App{
		cfg:         cfg,
		repo:        repo,
		ca:          ca,
		crlManager:  crlManager,
		serverCert:  serverCert,
		mux:         http.NewServeMux(),
		cache:       appCache,
		rateLimiter: NewRateLimiter(cfg.RateLimit),
	}
	
	// Initialize quota manager with adapter to convert store types to tenant types
	a.quotaManager = tenant.NewQuotaManagerWithProvider(func(ctx context.Context, tenantID string) (*tenant.QuotaUsage, error) {
		usage, err := repo.GetTenantUsage(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		return &tenant.QuotaUsage{
			Sites:       usage.Sites,
			Agents:      usage.Agents,
			VMs:         usage.VMs,
			ActivePlans: usage.ActivePlans,
			APIKeys:     usage.APIKeys,
		}, nil
	})
	
	a.setupHealthChecker(repo)
	a.registerRoutes()
	return a, nil
}

// setupHealthChecker initializes and configures the health checker
func (a *App) setupHealthChecker(repo store.Repo) {
	a.healthChecker = health.NewChecker("dev") // Version can be injected at build time

	// Register database health check
	a.healthChecker.Register("database", func(ctx context.Context) error {
		if db, ok := repo.(interface{ DB() *sql.DB }); ok {
			return db.DB().PingContext(ctx)
		}
		// For non-sql repos, assume healthy
		return nil
	})

	// Register CA health check
	a.healthChecker.Register("ca", func(ctx context.Context) error {
		if a.ca == nil || a.ca.Certificate() == nil {
			return errors.New("CA not loaded")
		}
		return nil
	})

	// Register CRL health check
	a.healthChecker.Register("crl", func(ctx context.Context) error {
		if a.crlManager == nil {
			return errors.New("CRL manager not initialized")
		}
		return nil
	})
}

func (a *App) Handler() http.Handler {
	// Apply rate limiting first, then request logging
	return a.withRequestLogging(a.rateLimiter.Middleware()(a.mux))
}

func (a *App) StartBackgroundWorkers(ctx context.Context) {
	if a.cfg.OfflineSweepInterval <= 0 {
		return
	}
	ticker := time.NewTicker(a.cfg.OfflineSweepInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().UTC().Add(-a.cfg.OfflineAfter)
				updated, err := a.repo.SweepOfflineAgents(context.Background(), cutoff)
				if err != nil {
					log.Printf("offline sweeper error: %v", err)
					continue
				}
				if updated > 0 {
					log.Printf("offline sweeper marked %d agents offline", updated)
				}
			}
		}
	}()
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
	a.mux.HandleFunc("GET /healthz", a.handleHealthz)
	a.mux.HandleFunc("GET /readyz", a.handleReadyz)
	a.mux.HandleFunc("GET /metrics", a.handleMetrics)
	a.mux.Handle("POST /tenants", a.adminAuth(http.HandlerFunc(a.handleCreateTenant)))
	a.mux.Handle("GET /tenants", a.adminAuth(http.HandlerFunc(a.handleListTenants)))
	a.mux.Handle("POST /tenants/{tenantID}/api-keys", a.adminAuth(http.HandlerFunc(a.handleCreateAPIKey)))
	a.mux.Handle("GET /tenants/{tenantID}/api-keys", a.apiKeyAuth(http.HandlerFunc(a.handleListAPIKeys)))
	a.mux.Handle("DELETE /tenants/{tenantID}/api-keys/{keyID}", a.apiKeyAuth(http.HandlerFunc(a.handleDeleteAPIKey)))

	a.mux.Handle("POST /tenants/{tenantID}/sites", a.apiKeyAuth(http.HandlerFunc(a.handleCreateSite)))
	a.mux.Handle("GET /tenants/{tenantID}/sites", a.apiKeyAuth(http.HandlerFunc(a.handleListSites)))
	a.mux.Handle("POST /tenants/{tenantID}/enrollment-tokens", a.apiKeyAuth(http.HandlerFunc(a.handleIssueEnrollmentToken)))
	a.mux.Handle("GET /tenants/{tenantID}/enrollment-tokens", a.apiKeyAuth(http.HandlerFunc(a.handleListEnrollmentTokens)))
	a.mux.Handle("GET /tenants/{tenantID}/usage", a.apiKeyAuth(http.HandlerFunc(a.handleGetTenantUsage)))

	a.mux.HandleFunc("POST /enroll", a.handleEnroll)
	a.mux.HandleFunc("POST /v1/enroll", a.handleEnroll)
	a.mux.Handle("POST /agents/heartbeat", a.agentMTLSAuth(http.HandlerFunc(a.handleHeartbeat)))
	a.mux.Handle("POST /v1/heartbeat", a.agentMTLSAuth(http.HandlerFunc(a.handleHeartbeat)))
	a.mux.Handle("POST /agents/logs", a.agentMTLSAuth(http.HandlerFunc(a.handleIngestLogs)))
	a.mux.Handle("POST /v1/logs", a.agentMTLSAuth(http.HandlerFunc(a.handleIngestLogFrame)))
	a.mux.Handle("GET /v1/plans/next", a.agentMTLSAuth(http.HandlerFunc(a.handleListPendingPlansV1)))
	a.mux.Handle("POST /v1/executions/result", a.agentMTLSAuth(http.HandlerFunc(a.handleReportPlanResultV1)))
	a.mux.Handle("POST /v1/unenroll", a.agentMTLSAuth(http.HandlerFunc(a.handleUnenroll)))
	a.mux.Handle("POST /v1/renew", a.agentMTLSAuth(http.HandlerFunc(a.handleRenew)))

	// CRL endpoints (public, no auth required)
	a.mux.HandleFunc("GET /v1/crl", a.handleGetCRL)
	a.mux.HandleFunc("GET /v1/crl.pem", a.handleGetCRLPEM)

	a.mux.Handle("POST /sites/{siteID}/plans", a.apiKeyAuth(http.HandlerFunc(a.handleApplyPlan)))
	a.mux.Handle("GET /sites/{siteID}/hosts", a.apiKeyAuth(http.HandlerFunc(a.handleListHosts)))
	a.mux.Handle("GET /sites/{siteID}/vms", a.apiKeyAuth(http.HandlerFunc(a.handleListVMs)))
	a.mux.Handle("GET /sites/{siteID}/executions", a.apiKeyAuth(http.HandlerFunc(a.handleListExecutions)))
	a.mux.Handle("GET /executions/{executionID}/logs", a.apiKeyAuth(http.HandlerFunc(a.handleListExecutionLogs)))

	// Admin audit endpoints
	a.mux.Handle("POST /admin/audit/verify", a.adminAuth(http.HandlerFunc(a.handleVerifyAuditChain)))
	a.mux.Handle("GET /admin/audit/events", a.adminAuth(http.HandlerFunc(a.handleListAuditEvents)))
	a.mux.Handle("GET /admin/audit/chain-info", a.adminAuth(http.HandlerFunc(a.handleAuditChainInfo)))
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

// apiKeyCacheTTL is the duration for which API key validation results are cached.
const apiKeyCacheTTL = 5 * time.Minute

// cachedAPIKeyValidation represents a cached API key validation result.
type cachedAPIKeyValidation struct {
	TenantID string
	Valid    bool
}

func (a *App) apiKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if apiKey == "" {
			writeError(w, http.StatusUnauthorized, "missing api key")
			return
		}
		h := hashString(apiKey)
		cacheKey := "apikey:" + h

		// Check cache first
		if cached, found := a.cache.Get(cacheKey); found {
			if validation, ok := cached.(cachedAPIKeyValidation); ok && validation.Valid {
				sla.RecordCacheHit()
				ctx := context.WithValue(r.Context(), ctxTenantID{}, validation.TenantID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		sla.RecordCacheMiss()

		validation, err := a.repo.ValidateAPIKey(r.Context(), h)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, store.ErrUnauthorized) {
				status = http.StatusUnauthorized
			}
			writeError(w, status, "invalid api key")
			return
		}

		// Cache successful validation
		a.cache.Set(cacheKey, cachedAPIKeyValidation{
			TenantID: validation.TenantID,
			Valid:    true,
		}, apiKeyCacheTTL)

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

		// Check if certificate is revoked in CRL
		serial := cert.SerialNumber.String()
		if a.crlManager.IsRevoked(serial) {
			writeError(w, http.StatusUnauthorized, "certificate revoked")
			return
		}

		// Also check database for revocation (in case CRL is stale)
		isRevoked, err := a.repo.IsCertificateRevoked(r.Context(), serial)
		if err != nil {
			log.Printf("error checking certificate revocation: %v", err)
			// Continue anyway, don't block on DB error
		} else if isRevoked {
			writeError(w, http.StatusUnauthorized, "certificate revoked")
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
	_ = a.writeAudit(r.Context(), tenant.ID, "", "SYSTEM", "", "tenant.create", "tenant", tenant.ID, requestID(r), sourceIP(r), nil)
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

	// Check API key quota
	if err := a.quotaManager.CheckQuota(r.Context(), tenantID, tenant.QuotaResourceAPIKey); err != nil {
		if errors.Is(err, tenant.ErrQuotaExceeded) {
			writeError(w, http.StatusForbidden, "API key quota exceeded")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to check quota")
		return
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
		CreatedAt: time.Now().UTC(),
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
	_ = a.writeAudit(r.Context(), tenantID, "", "SYSTEM", "", "apikey.create", "api_key", created.ID, requestID(r), sourceIP(r), nil)
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         created.ID,
		"tenant_id":  created.TenantID,
		"name":       created.Name,
		"api_key":    plainKey,
		"expires_at": created.ExpiresAt,
	})
}

func (a *App) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")
	if !a.tenantAllowed(r.Context(), tenantID) {
		writeError(w, http.StatusForbidden, "tenant mismatch")
		return
	}
	keys, err := a.repo.ListAPIKeys(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list api keys")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"api_keys": keys})
}

func (a *App) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")
	if !a.tenantAllowed(r.Context(), tenantID) {
		writeError(w, http.StatusForbidden, "tenant mismatch")
		return
	}
	keyID := r.PathValue("keyID")
	if _, err := uuid.Parse(keyID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}
	err := a.repo.DeleteAPIKey(r.Context(), tenantID, keyID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "api key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete api key")
		return
	}
	_ = a.writeAudit(r.Context(), tenantID, "", "USER", "api-key", "apikey.delete", "api_key", keyID, requestID(r), sourceIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
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

	// Check site quota
	if err := a.quotaManager.CheckQuota(r.Context(), tenantID, tenant.QuotaResourceSite); err != nil {
		if errors.Is(err, tenant.ErrQuotaExceeded) {
			writeError(w, http.StatusForbidden, "site quota exceeded")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to check quota")
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
	_ = a.writeAudit(r.Context(), tenantID, site.ID, "USER", "api-key", "site.create", "site", site.ID, requestID(r), sourceIP(r), nil)
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
	_ = a.writeAudit(r.Context(), tenantID, req.SiteID, "USER", "api-key", "enrollment_token.issue", "enrollment_token", issued.ID, requestID(r), sourceIP(r), nil)
	writeJSON(w, http.StatusCreated, map[string]any{
		"token_id":   issued.ID,
		"site_id":    issued.SiteID,
		"token":      plainToken,
		"expires_at": issued.ExpiresAt,
		"one_time":   true,
	})
}

func (a *App) handleListEnrollmentTokens(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")
	if !a.tenantAllowed(r.Context(), tenantID) {
		writeError(w, http.StatusForbidden, "tenant mismatch")
		return
	}
	tokens, err := a.repo.ListEnrollmentTokens(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list enrollment tokens")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": tokens})
}

func (a *App) handleGetTenantUsage(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")
	if !a.tenantAllowed(r.Context(), tenantID) {
		writeError(w, http.StatusForbidden, "tenant mismatch")
		return
	}

	ctx := r.Context()
	status, err := a.quotaManager.GetQuotaStatus(ctx, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get tenant usage")
		return
	}

	writeJSON(w, http.StatusOK, status)
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
	_ = a.writeAudit(r.Context(), agent.TenantID, agent.SiteID, "AGENT", agent.ID, "agent.enroll", "agent", agent.ID, requestID(r), sourceIP(r), nil)
	a.metrics.enrollmentsTotal.Add(1)
	heartbeatSeconds := int(a.cfg.HeartbeatInterval.Seconds())
	if heartbeatSeconds <= 0 {
		heartbeatSeconds = 15
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":                  agent.TenantID,
		"site_id":                    agent.SiteID,
		"host_id":                    agent.HostID,
		"agent_id":                   agent.ID,
		"client_certificate_pem":     string(certPEM),
		"ca_certificate_pem":         string(a.ca.CertPEM()),
		"refresh_token":              refreshToken,
		"heartbeat_endpoint":         "/agents/heartbeat",
		"heartbeat_interval_sec":     heartbeatSeconds,
		"heartbeat_interval_seconds": heartbeatSeconds,
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
	if err := decodeJSONAllowUnknown(r.Body, &req); err != nil {
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
	a.metrics.heartbeatsTotal.Add(1)

	pending, err := a.repo.LeasePendingPlans(r.Context(), agent.ID, a.cfg.MaxPlansPerHeartbeat, a.cfg.PlanLeaseTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to lease plans")
		return
	}
	heartbeatSeconds := int(a.cfg.HeartbeatInterval.Seconds())
	if heartbeatSeconds <= 0 {
		heartbeatSeconds = 15
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"next_heartbeat_seconds": heartbeatSeconds,
		"pending_plans":          leasedPlansToAgentPayload(pending),
	})
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
	if err := decodeJSONAllowUnknown(r.Body, &req); err != nil {
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

func (a *App) handleListPendingPlansV1(w http.ResponseWriter, r *http.Request) {
	agent := r.Context().Value(ctxAgent{}).(store.Agent)
	pending, err := a.repo.LeasePendingPlans(r.Context(), agent.ID, a.cfg.MaxPlansPerHeartbeat, a.cfg.PlanLeaseTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to lease plans")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plans": leasedPlansToAgentPayload(pending)})
}

func (a *App) handleReportPlanResultV1(w http.ResponseWriter, r *http.Request) {
	agent := r.Context().Value(ctxAgent{}).(store.Agent)
	type actionResult struct {
		ActionID   string    `json:"action_id"`
		OK         bool      `json:"ok"`
		ErrorCode  string    `json:"error_code"`
		Message    string    `json:"message"`
		FinishedAt time.Time `json:"finished_at"`
	}
	type request struct {
		PlanID      string         `json:"plan_id"`
		ExecutionID string         `json:"execution_id"`
		Results     []actionResult `json:"results"`
	}
	var req request
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items := make([]store.PlanActionResultItem, 0, len(req.Results))
	for _, result := range req.Results {
		items = append(items, store.PlanActionResultItem{
			ActionID:   strings.TrimSpace(result.ActionID),
			OK:         result.OK,
			ErrorCode:  strings.TrimSpace(result.ErrorCode),
			Message:    strings.TrimSpace(result.Message),
			FinishedAt: result.FinishedAt,
		})
	}
	if len(items) == 0 {
		writeError(w, http.StatusBadRequest, "results are required")
		return
	}
	err := a.repo.ReportPlanResult(r.Context(), agent.ID, store.PlanResultReport{
		PlanID:      strings.TrimSpace(req.PlanID),
		ExecutionID: strings.TrimSpace(req.ExecutionID),
		Results:     items,
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "plan not found")
		case errors.Is(err, store.ErrUnauthorized):
			writeError(w, http.StatusForbidden, "agent does not own plan")
		default:
			writeError(w, http.StatusInternalServerError, "failed to persist execution result")
		}
		return
	}
	a.metrics.executionsTotal.Add(1)
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted"})
}

func (a *App) handleIngestLogs(w http.ResponseWriter, r *http.Request) {
	agent := r.Context().Value(ctxAgent{}).(store.Agent)
	type request struct {
		AgentID string                 `json:"agent_id"`
		Entries []store.LogIngestEntry `json:"entries"`
	}
	var req request
	if err := decodeJSONAllowUnknown(r.Body, &req); err != nil {
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
	_ = a.writeAudit(r.Context(), tenantID, siteID, "USER", "api-key", "plan.apply", "plan", result.Plan.ID, requestID(r), sourceIP(r), result.Plan.OperationsJSON)
	a.metrics.plansApplied.Add(1)
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

func (a *App) handleListExecutions(w http.ResponseWriter, r *http.Request) {
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

	// Parse status filter (comma-separated)
	var statuses []string
	statusParam := r.URL.Query().Get("status")
	if statusParam != "" {
		for _, s := range strings.Split(statusParam, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				statuses = append(statuses, s)
			}
		}
	}

	// Parse limit (default 50)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	executions, err := a.repo.ListExecutions(r.Context(), tenantID, siteID, statuses, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list executions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"executions": executions})
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
	return decodeJSONWithMode(body, v, true)
}

func decodeJSONAllowUnknown(body io.Reader, v any) error {
	return decodeJSONWithMode(body, v, false)
}

func decodeJSONWithMode(body io.Reader, v any, disallowUnknown bool) error {
	dec := json.NewDecoder(io.LimitReader(body, 1<<20))
	if disallowUnknown {
		dec.DisallowUnknownFields()
	}
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

type leasedPlanPayload struct {
	PlanID      string              `json:"plan_id"`
	ExecutionID string              `json:"execution_id"`
	Actions     []leasedActionEntry `json:"actions"`
}

type leasedActionEntry struct {
	ActionID      string          `json:"action_id"`
	Type          string          `json:"type"`
	Params        json.RawMessage `json:"params"`
	TimeoutSecond int             `json:"timeout"`
}

func leasedPlansToAgentPayload(in []store.LeasedPlan) []leasedPlanPayload {
	out := make([]leasedPlanPayload, 0, len(in))
	for _, plan := range in {
		actions := make([]leasedActionEntry, 0, len(plan.Actions))
		for _, action := range plan.Actions {
			entry, ok := toLeasedActionEntry(action)
			if !ok {
				continue
			}
			actions = append(actions, entry)
		}
		if len(actions) == 0 {
			continue
		}
		out = append(out, leasedPlanPayload{
			PlanID:      plan.PlanID,
			ExecutionID: firstNonEmpty(plan.ExecutionID, plan.PlanID),
			Actions:     actions,
		})
	}
	return out
}

func toLeasedActionEntry(action store.PlanAction) (leasedActionEntry, bool) {
	type applyPayload struct {
		VMID      string `json:"vm_id"`
		Name      string `json:"name"`
		VCPUCount int    `json:"vcpu_count"`
		MemoryMiB int64  `json:"memory_mib"`
	}
	var payload applyPayload
	if len(action.PayloadJSON) > 0 {
		_ = json.Unmarshal(action.PayloadJSON, &payload)
	}

	operation := strings.ToUpper(strings.TrimSpace(action.OperationType))
	vmID := firstNonEmpty(payload.VMID, action.VMID)
	switch operation {
	case "CREATE":
		if vmID == "" {
			return leasedActionEntry{}, false
		}
		params, _ := json.Marshal(map[string]any{
			"vm_id":      vmID,
			"name":       firstNonEmpty(payload.Name, vmID),
			"vcpu":       maxInt(payload.VCPUCount, 1),
			"memory_mib": maxInt64(payload.MemoryMiB, 128),
		})
		return leasedActionEntry{
			ActionID:      action.OperationID,
			Type:          "MicroVMCreate",
			Params:        params,
			TimeoutSecond: 30,
		}, true
	case "START", "STOP", "DELETE", "PAUSE", "RESUME":
		if vmID == "" {
			return leasedActionEntry{}, false
		}
		actionType := "MicroVMStart"
		if operation == "STOP" {
			actionType = "MicroVMStop"
		}
		if operation == "DELETE" {
			actionType = "MicroVMDelete"
		}
		if operation == "PAUSE" {
			actionType = "MicroVMPause"
		}
		if operation == "RESUME" {
			actionType = "MicroVMResume"
		}
		params, _ := json.Marshal(map[string]any{"vm_id": vmID})
		return leasedActionEntry{
			ActionID:      action.OperationID,
			Type:          actionType,
			Params:        params,
			TimeoutSecond: 30,
		}, true
	case "SNAPSHOT":
		if vmID == "" {
			return leasedActionEntry{}, false
		}
		params, _ := json.Marshal(map[string]any{
			"vm_id":         vmID,
			"snapshot_name": payload.Name,
		})
		return leasedActionEntry{
			ActionID:      action.OperationID,
			Type:          "MicroVMSnapshot",
			Params:        params,
			TimeoutSecond: 300, // Snapshot may take longer
		}, true
	case "EXECUTE":
		// For EXECUTE, we need the command in the payload
		var executePayload struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
			Timeout int      `json:"timeout_seconds"`
			Dir     string   `json:"working_dir"`
		}
		if len(action.PayloadJSON) > 0 {
			_ = json.Unmarshal(action.PayloadJSON, &executePayload)
		}
		if executePayload.Command == "" {
			return leasedActionEntry{}, false
		}
		params, _ := json.Marshal(map[string]any{
			"command":         executePayload.Command,
			"args":            executePayload.Args,
			"timeout_seconds": executePayload.Timeout,
			"working_dir":     executePayload.Dir,
		})
		timeout := 30
		if executePayload.Timeout > 0 {
			timeout = executePayload.Timeout
		}
		return leasedActionEntry{
			ActionID:      action.OperationID,
			Type:          "CommandExecute",
			Params:        params,
			TimeoutSecond: timeout,
		}, true
	default:
		return leasedActionEntry{}, false
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
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


func (a *App) handleUnenroll(w http.ResponseWriter, r *http.Request) {
	agent := r.Context().Value(ctxAgent{}).(store.Agent)
	type request struct {
		AgentID string `json:"agent_id"`
		Reason  string `json:"reason"`
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

	// Revoke the agent's certificate
	if agent.CertSerial != "" {
		// Determine revocation reason based on request
		reason := pki.ReasonCessationOfOperation
		if req.Reason == "compromised" {
			reason = pki.ReasonKeyCompromise
		}

		// Add to CRL
		if err := a.crlManager.Revoke(agent.CertSerial, reason, agent.ID); err != nil {
			log.Printf("error revoking certificate in CRL: %v", err)
			// Continue with unenrollment even if CRL update fails
		}

		// Store revocation in database
		if err := a.repo.RevokeCertificate(r.Context(), agent.CertSerial, int(reason), agent.ID); err != nil {
			log.Printf("error storing certificate revocation: %v", err)
			// Continue with unenrollment even if DB update fails
		}
	}

	// Update agent state to unenrolled
	if err := a.repo.UnenrollAgent(r.Context(), agent.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to unenroll agent")
		return
	}

	_ = a.writeAudit(r.Context(), agent.TenantID, agent.SiteID, "AGENT", agent.ID, "agent.unenroll", "agent", agent.ID, requestID(r), sourceIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleRenew(w http.ResponseWriter, r *http.Request) {
	agent := r.Context().Value(ctxAgent{}).(store.Agent)
	type request struct {
		AgentID      string `json:"agent_id"`
		CSRPEM       string `json:"csr_pem"`
		RefreshToken string `json:"refresh_token"`
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
	if req.CSRPEM == "" {
		writeError(w, http.StatusBadRequest, "csr_pem is required")
		return
	}

	// Validate refresh token
	if agent.RefreshTokenHash != hashString(req.RefreshToken) {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	// Issue new certificate
	certPEM, certSerial, err := a.ca.SignAgentCSR([]byte(req.CSRPEM), agent.ID, agent.TenantID, agent.SiteID, a.cfg.AgentCertTTL)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid csr: %v", err))
		return
	}

	// Generate new refresh token
	newRefreshToken, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	// Update agent with new cert serial and refresh token hash
	if err := a.repo.UpdateAgentCertificate(r.Context(), agent.ID, certSerial, hashString(newRefreshToken)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update agent certificate")
		return
	}

	expiresAt := time.Now().UTC().Add(a.cfg.AgentCertTTL)
	writeJSON(w, http.StatusOK, map[string]any{
		"client_certificate_pem": string(certPEM),
		"ca_certificate_pem":     string(a.ca.CertPEM()),
		"expires_at":             expiresAt.Format(time.RFC3339),
		"refresh_token":          newRefreshToken,
	})
}

// handleMetrics returns Prometheus-style metrics
func (a *App) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	fmt.Fprintf(w, "# HELP nkudo_requests_total Total number of HTTP requests\n")
	fmt.Fprintf(w, "# TYPE nkudo_requests_total counter\n")
	fmt.Fprintf(w, "nkudo_requests_total %d\n\n", a.metrics.requestsTotal.Load())

	fmt.Fprintf(w, "# HELP nkudo_enrollments_total Total number of agent enrollments\n")
	fmt.Fprintf(w, "# TYPE nkudo_enrollments_total counter\n")
	fmt.Fprintf(w, "nkudo_enrollments_total %d\n\n", a.metrics.enrollmentsTotal.Load())

	fmt.Fprintf(w, "# HELP nkudo_heartbeats_total Total number of agent heartbeats\n")
	fmt.Fprintf(w, "# TYPE nkudo_heartbeats_total counter\n")
	fmt.Fprintf(w, "nkudo_heartbeats_total %d\n\n", a.metrics.heartbeatsTotal.Load())

	fmt.Fprintf(w, "# HELP nkudo_plans_applied_total Total number of plans applied\n")
	fmt.Fprintf(w, "# TYPE nkudo_plans_applied_total counter\n")
	fmt.Fprintf(w, "nkudo_plans_applied_total %d\n\n", a.metrics.plansApplied.Load())

	fmt.Fprintf(w, "# HELP nkudo_executions_total Total number of plan executions\n")
	fmt.Fprintf(w, "# TYPE nkudo_executions_total counter\n")
	fmt.Fprintf(w, "nkudo_executions_total %d\n\n", a.metrics.executionsTotal.Load())

	// Rate limiting metrics
	hits, blocks := a.rateLimiter.GetMetrics()
	fmt.Fprintf(w, "# HELP nkudo_rate_limit_hits_total Total number of rate limiter hits (allowed requests)\n")
	fmt.Fprintf(w, "# TYPE nkudo_rate_limit_hits_total counter\n")
	fmt.Fprintf(w, "nkudo_rate_limit_hits_total %d\n\n", hits)

	fmt.Fprintf(w, "# HELP nkudo_rate_limit_blocks_total Total number of rate limiter blocks (rejected requests)\n")
	fmt.Fprintf(w, "# TYPE nkudo_rate_limit_blocks_total counter\n")
	fmt.Fprintf(w, "nkudo_rate_limit_blocks_total %d\n", blocks)
}

// Stub handlers for audit endpoints - full implementation in Phase 4.4
func (a *App) handleVerifyAuditChain(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"valid": true, "verified_count": 0})
}

func (a *App) handleListAuditEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	events, err := a.repo.ListAuditEvents(r.Context(), tenantID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list audit events: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"events": events,
	})
}

func (a *App) handleAuditChainInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"chain_length":    0,
		"last_event_time": nil,
		"integrity_valid": true,
	})
}

// handleHealthz returns a simple liveness check
// This endpoint should be used by Kubernetes liveness probes
// It always returns 200 OK to indicate the process is running
func (a *App) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

// handleReadyz returns a readiness check
// This endpoint should be used by Kubernetes readiness probes
// It runs all registered health checks and returns 200 if healthy, 503 if unhealthy
func (a *App) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if a.healthChecker == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"note":   "health checker not initialized",
		})
		return
	}

	status := a.healthChecker.Check(r.Context())

	httpStatus := http.StatusOK
	if status.Status != health.StatusHealthy {
		httpStatus = http.StatusServiceUnavailable
	}

	writeJSON(w, httpStatus, status)
}

// handleGetCRL returns the Certificate Revocation List in DER format
func (a *App) handleGetCRL(w http.ResponseWriter, r *http.Request) {
	crlBytes := a.crlManager.GetCRL()
	w.Header().Set("Content-Type", "application/pkix-crl")
	w.Header().Set("Content-Length", strconv.Itoa(len(crlBytes)))
	w.Header().Set("Cache-Control", "max-age=300") // Cache for 5 minutes
	w.WriteHeader(http.StatusOK)
	w.Write(crlBytes)
}

// handleGetCRLPEM returns the Certificate Revocation List in PEM format
func (a *App) handleGetCRLPEM(w http.ResponseWriter, r *http.Request) {
	crlPEM := a.crlManager.GetCRLPEM()
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Length", strconv.Itoa(len(crlPEM)))
	w.Header().Set("Cache-Control", "max-age=300") // Cache for 5 minutes
	w.WriteHeader(http.StatusOK)
	w.Write(crlPEM)
}

// writeAudit is a helper to write audit events
func (a *App) writeAudit(ctx context.Context, tenantID, siteID, actorType, actorID, action, resourceType, resourceID, requestID, sourceIP string, metadata []byte) error {
	return a.repo.WriteAudit(ctx, tenantID, siteID, actorType, actorID, action, resourceType, resourceID, requestID, sourceIP, metadata)
}
