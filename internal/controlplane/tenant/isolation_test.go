package tenant

import (
	"context"
	"errors"
	"testing"
	"time"

	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

type mockRepo struct {
	siteBelongsToTenant      func(ctx context.Context, siteID, tenantID string) (bool, error)
	executionBelongsToTenant func(ctx context.Context, executionID, tenantID string) (bool, error)
}

func (m *mockRepo) SiteBelongsToTenant(ctx context.Context, siteID, tenantID string) (bool, error) {
	if m.siteBelongsToTenant != nil {
		return m.siteBelongsToTenant(ctx, siteID, tenantID)
	}
	return true, nil
}

func (m *mockRepo) ExecutionBelongsToTenant(ctx context.Context, executionID, tenantID string) (bool, error) {
	if m.executionBelongsToTenant != nil {
		return m.executionBelongsToTenant(ctx, executionID, tenantID)
	}
	return true, nil
}

// Stub implementations for store.Repo interface
func (m *mockRepo) Close() error { return nil }
func (m *mockRepo) CreateTenant(ctx context.Context, t store.Tenant) (store.Tenant, error) { return t, nil }
func (m *mockRepo) ListTenants(ctx context.Context) ([]store.Tenant, error) { return nil, nil }
func (m *mockRepo) GetTenantByID(ctx context.Context, tenantID string) (store.Tenant, error) { return store.Tenant{}, nil }
func (m *mockRepo) CreateUser(ctx context.Context, user store.User) (store.User, error) { return user, nil }
func (m *mockRepo) GetUserByEmail(ctx context.Context, email string) (store.User, error) { return store.User{}, nil }
func (m *mockRepo) GetUserByEmailAndTenant(ctx context.Context, email, tenantID string) (store.User, error) { return store.User{}, nil }
func (m *mockRepo) GetUserByID(ctx context.Context, tenantID, userID string) (store.User, error) { return store.User{}, nil }
func (m *mockRepo) UpdateUserLastLogin(ctx context.Context, tenantID, userID string) error { return nil }
func (m *mockRepo) UpdateUserPassword(ctx context.Context, tenantID, userID, passwordHash string) error { return nil }
func (m *mockRepo) EmailExists(ctx context.Context, email string) (bool, error) { return false, nil }
func (m *mockRepo) CreateEmailVerificationToken(ctx context.Context, userID, tenantID, tokenHash string, expiresAt time.Time) error { return nil }
func (m *mockRepo) VerifyEmailToken(ctx context.Context, tokenHash string) (userID, tenantID string, err error) { return "", "", nil }
func (m *mockRepo) MarkEmailVerified(ctx context.Context, tenantID, userID string) error { return nil }
func (m *mockRepo) CreateAPIKey(ctx context.Context, key store.APIKey) (store.APIKey, error) { return key, nil }
func (m *mockRepo) ValidateAPIKey(ctx context.Context, keyHash string) (store.APIKeyValidation, error) { return store.APIKeyValidation{}, nil }
func (m *mockRepo) CreateSite(ctx context.Context, site store.Site) (store.Site, error) { return site, nil }
func (m *mockRepo) ListSites(ctx context.Context, tenantID string) ([]store.Site, error) { return nil, nil }
func (m *mockRepo) IssueEnrollmentToken(ctx context.Context, token store.EnrollmentToken) (store.EnrollmentToken, error) { return token, nil }
func (m *mockRepo) ConsumeEnrollmentToken(ctx context.Context, tokenHash string, now time.Time) (store.TokenConsumeResult, error) { return store.TokenConsumeResult{}, nil }
func (m *mockRepo) CreateAgentFromEnrollment(ctx context.Context, tokenID string, agent store.Agent, hostname string) (store.Agent, error) { return agent, nil }
func (m *mockRepo) GetAgentByID(ctx context.Context, agentID string) (store.Agent, error) { return store.Agent{}, nil }
func (m *mockRepo) IngestHeartbeat(ctx context.Context, hb store.Heartbeat) error { return nil }
func (m *mockRepo) ApplyPlan(ctx context.Context, input store.ApplyPlanInput) (store.ApplyPlanResult, error) { return store.ApplyPlanResult{}, nil }
func (m *mockRepo) LeasePendingPlans(ctx context.Context, agentID string, limit int, leaseTTL time.Duration) ([]store.LeasedPlan, error) { return nil, nil }
func (m *mockRepo) ReportPlanResult(ctx context.Context, agentID string, report store.PlanResultReport) error { return nil }
func (m *mockRepo) IngestLogs(ctx context.Context, req store.LogIngest) (accepted int64, dropped int64, err error) { return 0, 0, nil }
func (m *mockRepo) SweepOfflineAgents(ctx context.Context, staleBefore time.Time) (int64, error) { return 0, nil }
func (m *mockRepo) ListHosts(ctx context.Context, tenantID, siteID string) ([]store.Host, error) { return nil, nil }
func (m *mockRepo) ListVMs(ctx context.Context, tenantID, siteID string) ([]store.MicroVM, error) { return nil, nil }
func (m *mockRepo) ListExecutionLogs(ctx context.Context, tenantID, executionID string, limit int) ([]store.ExecutionLog, error) { return nil, nil }
func (m *mockRepo) WriteAudit(ctx context.Context, tenantID, siteID, actorType, actorID, action, resourceType, resourceID, requestID, sourceIP string, metadata []byte) error { return nil }
func (m *mockRepo) ListEnrollmentTokens(ctx context.Context, tenantID string) ([]store.EnrollmentTokenWithStatus, error) { return nil, nil }
func (m *mockRepo) ListExecutions(ctx context.Context, tenantID, siteID string, statuses []string, limit int) ([]store.ExecutionWithTimestamps, error) { return nil, nil }
func (m *mockRepo) ListAPIKeys(ctx context.Context, tenantID string) ([]store.APIKey, error) { return nil, nil }
func (m *mockRepo) DeleteAPIKey(ctx context.Context, tenantID, keyID string) error { return nil }
func (m *mockRepo) UnenrollAgent(ctx context.Context, agentID string) error { return nil }
func (m *mockRepo) UpdateAgentCertificate(ctx context.Context, agentID, certSerial, refreshTokenHash string) error { return nil }
func (m *mockRepo) ListCertificateHistory(ctx context.Context, agentID string, limit int) ([]store.CertificateHistory, error) { return nil, nil }
func (m *mockRepo) RecordCertificateIssuance(ctx context.Context, history store.CertificateHistory) error { return nil }
func (m *mockRepo) GetLastAuditEvent(ctx context.Context) (*store.AuditEvent, error) { return nil, nil }
func (m *mockRepo) WriteAuditEvent(ctx context.Context, event *store.AuditEvent) error { return nil }
func (m *mockRepo) UpdateAuditEventValidity(ctx context.Context, id int64, valid bool) error { return nil }
func (m *mockRepo) ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]store.AuditEvent, error) { return nil, nil }
func (m *mockRepo) RevokeCertificate(ctx context.Context, serial string, reason int, agentID string) error { return nil }
func (m *mockRepo) IsCertificateRevoked(ctx context.Context, serial string) (bool, error) { return false, nil }
func (m *mockRepo) ListRevokedCertificates(ctx context.Context) ([]store.CRLEntry, error) { return nil, nil }
func (m *mockRepo) GetTenantUsage(ctx context.Context, tenantID string) (*store.TenantUsage, error) { return nil, nil }
func (m *mockRepo) GetTenantLimits(ctx context.Context, tenantID string) (*store.QuotaLimits, error) { return nil, nil }
func (m *mockRepo) SetTenantLimits(ctx context.Context, tenantID string, limits store.QuotaLimits) error { return nil }
func (m *mockRepo) CreateInvitation(ctx context.Context, invitation store.ProjectInvitation) error { return nil }
func (m *mockRepo) GetInvitationByToken(ctx context.Context, tokenHash string) (*store.ProjectInvitation, error) { return nil, nil }
func (m *mockRepo) ListPendingInvitations(ctx context.Context, tenantID string) ([]store.ProjectInvitation, error) { return nil, nil }
func (m *mockRepo) ListUserInvitations(ctx context.Context, email string) ([]store.ProjectInvitationWithProject, error) { return nil, nil }
func (m *mockRepo) AcceptInvitation(ctx context.Context, invitationID, userID string) error { return nil }
func (m *mockRepo) DeclineInvitation(ctx context.Context, invitationID string) error { return nil }
func (m *mockRepo) CancelInvitation(ctx context.Context, tenantID, invitationID string) error { return nil }
func (m *mockRepo) GetInvitationByID(ctx context.Context, tenantID, invitationID string) (*store.ProjectInvitation, error) { return nil, nil }
func (m *mockRepo) CreateVXLANNetwork(ctx context.Context, tenantID, siteID string, network store.VXLANNetwork) (store.VXLANNetwork, error) { return network, nil }
func (m *mockRepo) ListVXLANNetworks(ctx context.Context, tenantID, siteID string) ([]store.VXLANNetwork, error) { return nil, nil }
func (m *mockRepo) GetVXLANNetwork(ctx context.Context, tenantID, networkID string) (store.VXLANNetwork, error) { return store.VXLANNetwork{}, nil }
func (m *mockRepo) GetVXLANNetworkByVNI(ctx context.Context, tenantID string, vni int) (store.VXLANNetwork, error) { return store.VXLANNetwork{}, nil }
func (m *mockRepo) DeleteVXLANNetwork(ctx context.Context, tenantID, networkID string) error { return nil }
func (m *mockRepo) VXLANNetworkBelongsToTenant(ctx context.Context, networkID, tenantID string) (bool, error) { return false, nil }
func (m *mockRepo) CreateVXLANTunnel(ctx context.Context, tunnel store.VXLANTunnel) (store.VXLANTunnel, error) { return tunnel, nil }
func (m *mockRepo) ListVXLANTunnels(ctx context.Context, networkID string) ([]store.VXLANTunnel, error) { return nil, nil }
func (m *mockRepo) GetVXLANTunnel(ctx context.Context, networkID, hostID string) (store.VXLANTunnel, error) { return store.VXLANTunnel{}, nil }
func (m *mockRepo) UpdateVXLANTunnelStatus(ctx context.Context, tunnelID string, status string) error { return nil }
func (m *mockRepo) DeleteVXLANTunnel(ctx context.Context, tunnelID string) error { return nil }
func (m *mockRepo) AttachVMToNetwork(ctx context.Context, attachment store.VMNetworkAttachment) (store.VMNetworkAttachment, error) { return attachment, nil }
func (m *mockRepo) DetachVMFromNetwork(ctx context.Context, vmID, networkID string) error { return nil }
func (m *mockRepo) ListVMNetworkAttachments(ctx context.Context, vmID string) ([]store.VMNetworkAttachment, error) { return nil, nil }
func (m *mockRepo) ListNetworkVMAttachments(ctx context.Context, networkID string) ([]store.VMNetworkAttachment, error) { return nil, nil }

func TestEnforceTenantAccess(t *testing.T) {
	tests := []struct {
		name        string
		tenantID    string
		resource    Resource
		mockRepo    *mockRepo
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid access - matching tenant ID",
			tenantID: "tenant-1",
			resource: Resource{
				TenantID: "tenant-1",
				Type:     ResourceTypeAgent,
				ID:       "agent-1",
			},
			wantErr: false,
		},
		{
			name:        "invalid access - mismatched tenant ID",
			tenantID:    "tenant-1",
			resource: Resource{
				TenantID: "tenant-2",
				Type:     ResourceTypeAgent,
				ID:       "agent-1",
			},
			wantErr:     true,
			errContains: "tenant isolation violation",
		},
		{
			name:        "missing tenant ID",
			tenantID:    "",
			resource: Resource{
				TenantID: "tenant-1",
				Type:     ResourceTypeAgent,
				ID:       "agent-1",
			},
			wantErr:     true,
			errContains: "tenant ID is required",
		},
		{
			name:        "missing resource ID",
			tenantID:    "tenant-1",
			resource: Resource{
				TenantID: "tenant-1",
				Type:     ResourceTypeAgent,
				ID:       "",
			},
			wantErr:     true,
			errContains: "resource ID is required",
		},
		{
			name:     "site access - allowed",
			tenantID: "tenant-1",
			resource: Resource{
				Type: ResourceTypeSite,
				ID:   "site-1",
			},
			mockRepo: &mockRepo{
				siteBelongsToTenant: func(ctx context.Context, siteID, tenantID string) (bool, error) {
					return siteID == "site-1" && tenantID == "tenant-1", nil
				},
			},
			wantErr: false,
		},
		{
			name:     "site access - denied",
			tenantID: "tenant-1",
			resource: Resource{
				Type: ResourceTypeSite,
				ID:   "site-2",
			},
			mockRepo: &mockRepo{
				siteBelongsToTenant: func(ctx context.Context, siteID, tenantID string) (bool, error) {
					return false, nil
				},
			},
			wantErr:     true,
			errContains: "does not belong to tenant",
		},
		{
			name:     "execution access - allowed",
			tenantID: "tenant-1",
			resource: Resource{
				Type: ResourceTypeExecution,
				ID:   "exec-1",
			},
			mockRepo: &mockRepo{
				executionBelongsToTenant: func(ctx context.Context, executionID, tenantID string) (bool, error) {
					return executionID == "exec-1" && tenantID == "tenant-1", nil
				},
			},
			wantErr: false,
		},
		{
			name:     "execution access - denied",
			tenantID: "tenant-1",
			resource: Resource{
				Type: ResourceTypeExecution,
				ID:   "exec-2",
			},
			mockRepo: &mockRepo{
				executionBelongsToTenant: func(ctx context.Context, executionID, tenantID string) (bool, error) {
					return false, nil
				},
			},
			wantErr:     true,
			errContains: "does not belong to tenant",
		},
		{
			name:     "unknown resource type",
			tenantID: "tenant-1",
			resource: Resource{
				Type: "unknown",
				ID:   "res-1",
			},
			wantErr:     true,
			errContains: "unknown resource type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockRepo
			if repo == nil {
				repo = &mockRepo{}
			}
			enforcer := NewIsolationEnforcer(repo)
			ctx := context.Background()

			err := enforcer.EnforceTenantAccess(ctx, tt.tenantID, tt.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnforceTenantAccess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("EnforceTenantAccess() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestEnforceTenantAccessWithResource(t *testing.T) {
	repo := &mockRepo{
		siteBelongsToTenant: func(ctx context.Context, siteID, tenantID string) (bool, error) {
			return siteID == "site-1" && tenantID == "tenant-1", nil
		},
	}
	enforcer := NewIsolationEnforcer(repo)
	ctx := context.Background()

	// Test allowed access
	err := enforcer.EnforceTenantAccessWithResource(ctx, "tenant-1", ResourceTypeSite, "site-1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test denied access
	err = enforcer.EnforceTenantAccessWithResource(ctx, "tenant-1", ResourceTypeSite, "site-2")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !errors.Is(err, ErrTenantIsolationViolation) {
		t.Errorf("Expected ErrTenantIsolationViolation, got %v", err)
	}
}

func TestRepoErrors(t *testing.T) {
	repo := &mockRepo{
		siteBelongsToTenant: func(ctx context.Context, siteID, tenantID string) (bool, error) {
			return false, errors.New("database error")
		},
		executionBelongsToTenant: func(ctx context.Context, executionID, tenantID string) (bool, error) {
			return false, errors.New("database error")
		},
	}
	enforcer := NewIsolationEnforcer(repo)
	ctx := context.Background()

	// Test site access with DB error
	err := enforcer.EnforceTenantAccess(ctx, "tenant-1", Resource{Type: ResourceTypeSite, ID: "site-1"})
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !contains(err.Error(), "failed to verify site ownership") {
		t.Errorf("Expected 'failed to verify site ownership' error, got %v", err)
	}

	// Test execution access with DB error
	err = enforcer.EnforceTenantAccess(ctx, "tenant-1", Resource{Type: ResourceTypeExecution, ID: "exec-1"})
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !contains(err.Error(), "failed to verify execution ownership") {
		t.Errorf("Expected 'failed to verify execution ownership' error, got %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
