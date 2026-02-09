package tenant

import (
	"context"
	"errors"
	"testing"
)

type mockUsageRepo struct {
	getTenantUsage func(ctx context.Context, tenantID string) (*QuotaUsage, error)
}

func (m *mockUsageRepo) GetTenantUsage(ctx context.Context, tenantID string) (*QuotaUsage, error) {
	if m.getTenantUsage != nil {
		return m.getTenantUsage(ctx, tenantID)
	}
	return &QuotaUsage{}, nil
}

func TestDefaultQuotaLimits(t *testing.T) {
	limits := DefaultQuotaLimits()

	if limits.MaxSites != 10 {
		t.Errorf("Expected MaxSites = 10, got %d", limits.MaxSites)
	}
	if limits.MaxAgentsPerSite != 100 {
		t.Errorf("Expected MaxAgentsPerSite = 100, got %d", limits.MaxAgentsPerSite)
	}
	if limits.MaxVMsPerAgent != 50 {
		t.Errorf("Expected MaxVMsPerAgent = 50, got %d", limits.MaxVMsPerAgent)
	}
	if limits.MaxConcurrentPlans != 100 {
		t.Errorf("Expected MaxConcurrentPlans = 100, got %d", limits.MaxConcurrentPlans)
	}
	if limits.MaxAPIKeys != 20 {
		t.Errorf("Expected MaxAPIKeys = 20, got %d", limits.MaxAPIKeys)
	}
}

func TestQuotaManager_GetLimits(t *testing.T) {
	repo := &mockUsageRepo{}
	qm := NewQuotaManager(repo)

	// Test default limits
	limits := qm.GetLimits("tenant-1")
	defaultLimits := DefaultQuotaLimits()
	if limits != defaultLimits {
		t.Errorf("Expected default limits, got %v", limits)
	}

	// Test custom limits
	customLimits := QuotaLimits{
		MaxSites:           5,
		MaxAgentsPerSite:   50,
		MaxVMsPerAgent:     25,
		MaxConcurrentPlans: 50,
		MaxAPIKeys:         10,
	}
	qm.SetLimits("tenant-2", customLimits)

	limits = qm.GetLimits("tenant-2")
	if limits != customLimits {
		t.Errorf("Expected custom limits, got %v", limits)
	}

	// Test that other tenants still get defaults
	limits = qm.GetLimits("tenant-3")
	if limits != defaultLimits {
		t.Errorf("Expected default limits for tenant-3, got %v", limits)
	}
}

func TestQuotaManager_CheckQuota(t *testing.T) {
	tests := []struct {
		name         string
		usage        *QuotaUsage
		resourceType QuotaResourceType
		wantErr      bool
		errIs        error
	}{
		{
			name: "site quota - allowed",
			usage: &QuotaUsage{
				Sites: 5,
			},
			resourceType: QuotaResourceSite,
			wantErr:      false,
		},
		{
			name: "site quota - exceeded",
			usage: &QuotaUsage{
				Sites: 10,
			},
			resourceType: QuotaResourceSite,
			wantErr:      true,
			errIs:        ErrQuotaExceeded,
		},
		{
			name: "agent quota - allowed",
			usage: &QuotaUsage{
				Agents: 50,
			},
			resourceType: QuotaResourceAgent,
			wantErr:      false,
		},
		{
			name: "agent quota - exceeded",
			usage: &QuotaUsage{
				Agents: 100,
			},
			resourceType: QuotaResourceAgent,
			wantErr:      true,
			errIs:        ErrQuotaExceeded,
		},
		{
			name: "vm quota - allowed",
			usage: &QuotaUsage{
				VMs: 25,
			},
			resourceType: QuotaResourceVM,
			wantErr:      false,
		},
		{
			name: "vm quota - exceeded",
			usage: &QuotaUsage{
				VMs: 50,
			},
			resourceType: QuotaResourceVM,
			wantErr:      true,
			errIs:        ErrQuotaExceeded,
		},
		{
			name: "plan quota - allowed",
			usage: &QuotaUsage{
				ActivePlans: 50,
			},
			resourceType: QuotaResourcePlan,
			wantErr:      false,
		},
		{
			name: "plan quota - exceeded",
			usage: &QuotaUsage{
				ActivePlans: 100,
			},
			resourceType: QuotaResourcePlan,
			wantErr:      true,
			errIs:        ErrQuotaExceeded,
		},
		{
			name: "api key quota - allowed",
			usage: &QuotaUsage{
				APIKeys: 10,
			},
			resourceType: QuotaResourceAPIKey,
			wantErr:      false,
		},
		{
			name: "api key quota - exceeded",
			usage: &QuotaUsage{
				APIKeys: 20,
			},
			resourceType: QuotaResourceAPIKey,
			wantErr:      true,
			errIs:        ErrQuotaExceeded,
		},
		{
			name:         "unknown resource type",
			usage:        &QuotaUsage{},
			resourceType: "unknown",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUsageRepo{
				getTenantUsage: func(ctx context.Context, tenantID string) (*QuotaUsage, error) {
					return tt.usage, nil
				},
			}
			qm := NewQuotaManager(repo)
			ctx := context.Background()

			err := qm.CheckQuota(ctx, "tenant-1", tt.resourceType)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckQuota() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errIs != nil && !errors.Is(err, tt.errIs) {
				t.Errorf("CheckQuota() error = %v, expected error containing %v", err, tt.errIs)
			}
		})
	}
}

func TestQuotaManager_CheckQuotaWithCount(t *testing.T) {
	repo := &mockUsageRepo{
		getTenantUsage: func(ctx context.Context, tenantID string) (*QuotaUsage, error) {
			return &QuotaUsage{
				Sites: 5,
			}, nil
		},
	}
	qm := NewQuotaManager(repo)
	ctx := context.Background()

	// Test with count that stays within limit
	err := qm.CheckQuotaWithCount(ctx, "tenant-1", QuotaResourceSite, 3)
	if err != nil {
		t.Errorf("Expected no error for count 3, got %v", err)
	}

	// Test with count that exceeds limit
	err = qm.CheckQuotaWithCount(ctx, "tenant-1", QuotaResourceSite, 6)
	if err == nil {
		t.Error("Expected error for count 6, got nil")
	}
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Errorf("Expected ErrQuotaExceeded, got %v", err)
	}

	// Test with zero count (should always pass)
	err = qm.CheckQuotaWithCount(ctx, "tenant-1", QuotaResourceSite, 0)
	if err != nil {
		t.Errorf("Expected no error for count 0, got %v", err)
	}

	// Test with negative count (should always pass)
	err = qm.CheckQuotaWithCount(ctx, "tenant-1", QuotaResourceSite, -1)
	if err != nil {
		t.Errorf("Expected no error for count -1, got %v", err)
	}
}

func TestQuotaManager_GetUsage(t *testing.T) {
	expectedUsage := &QuotaUsage{
		Sites:       5,
		Agents:      10,
		VMs:         20,
		ActivePlans: 3,
		APIKeys:     7,
	}

	repo := &mockUsageRepo{
		getTenantUsage: func(ctx context.Context, tenantID string) (*QuotaUsage, error) {
			if tenantID == "tenant-1" {
				return expectedUsage, nil
			}
			return nil, errors.New("tenant not found")
		},
	}
	qm := NewQuotaManager(repo)
	ctx := context.Background()

	// Test successful retrieval
	usage, err := qm.GetUsage(ctx, "tenant-1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if usage != expectedUsage {
		t.Errorf("Expected %v, got %v", expectedUsage, usage)
	}

	// Test empty tenant ID
	_, err = qm.GetUsage(ctx, "")
	if err == nil {
		t.Error("Expected error for empty tenant ID, got nil")
	}

	// Test repo error
	_, err = qm.GetUsage(ctx, "tenant-2")
	if err == nil {
		t.Error("Expected error for tenant-2, got nil")
	}
}

func TestQuotaManager_GetQuotaUsagePercent(t *testing.T) {
	repo := &mockUsageRepo{
		getTenantUsage: func(ctx context.Context, tenantID string) (*QuotaUsage, error) {
			return &QuotaUsage{
				Sites:       5,
				Agents:      50,
				VMs:         25,
				ActivePlans: 50,
				APIKeys:     10,
			}, nil
		},
	}
	qm := NewQuotaManager(repo)
	ctx := context.Background()

	percentages, err := qm.GetQuotaUsagePercent(ctx, "tenant-1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check percentages (50% of defaults)
	expectedPercentages := map[string]float64{
		"sites":        50.0,
		"agents":       50.0,
		"vms":          50.0,
		"active_plans": 50.0,
		"api_keys":     50.0,
	}

	for key, expected := range expectedPercentages {
		if actual, ok := percentages[key]; !ok {
			t.Errorf("Missing percentage for %s", key)
		} else if actual != expected {
			t.Errorf("Expected %s = %f, got %f", key, expected, actual)
		}
	}
}

func TestQuotaManager_GetQuotaStatus(t *testing.T) {
	usage := &QuotaUsage{
		Sites:       5,
		Agents:      10,
		VMs:         20,
		ActivePlans: 3,
		APIKeys:     7,
	}

	repo := &mockUsageRepo{
		getTenantUsage: func(ctx context.Context, tenantID string) (*QuotaUsage, error) {
			return usage, nil
		},
	}
	qm := NewQuotaManager(repo)
	ctx := context.Background()

	status, err := qm.GetQuotaStatus(ctx, "tenant-1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if status.Limits != DefaultQuotaLimits() {
		t.Errorf("Expected default limits, got %v", status.Limits)
	}
	if status.Usage != *usage {
		t.Errorf("Expected usage %v, got %v", *usage, status.Usage)
	}
	if len(status.Percentages) == 0 {
		t.Error("Expected non-empty percentages")
	}
}

func TestQuotaManager_CustomLimits(t *testing.T) {
	repo := &mockUsageRepo{
		getTenantUsage: func(ctx context.Context, tenantID string) (*QuotaUsage, error) {
			return &QuotaUsage{
				Sites: 8,
			}, nil
		},
	}
	qm := NewQuotaManager(repo)
	ctx := context.Background()

	// Set custom limits with MaxSites = 10
	customLimits := QuotaLimits{
		MaxSites:           10,
		MaxAgentsPerSite:   50,
		MaxVMsPerAgent:     25,
		MaxConcurrentPlans: 50,
		MaxAPIKeys:         10,
	}
	qm.SetLimits("tenant-custom", customLimits)

	// With 8 sites and limit of 10, should be allowed
	err := qm.CheckQuota(ctx, "tenant-custom", QuotaResourceSite)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Now test with a tenant using default limits (MaxSites = 10, but same usage)
	// This should also pass since 8 < 10
	err = qm.CheckQuota(ctx, "tenant-default", QuotaResourceSite)
	if err != nil {
		t.Errorf("Expected no error for default tenant, got %v", err)
	}
}

func TestCheckQuotaRepoError(t *testing.T) {
	repo := &mockUsageRepo{
		getTenantUsage: func(ctx context.Context, tenantID string) (*QuotaUsage, error) {
			return nil, errors.New("database connection failed")
		},
	}
	qm := NewQuotaManager(repo)
	ctx := context.Background()

	err := qm.CheckQuota(ctx, "tenant-1", QuotaResourceSite)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !contains(err.Error(), "failed to get quota usage") {
		t.Errorf("Expected 'failed to get quota usage' error, got %v", err)
	}
}
