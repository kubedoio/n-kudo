package tenant

import (
	"context"
	"errors"
	"testing"
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
