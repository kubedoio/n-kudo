package tenant

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrQuotaExceeded is returned when a tenant exceeds their quota limit
var ErrQuotaExceeded = errors.New("quota exceeded")

// QuotaLimits defines the maximum resources a tenant can have
type QuotaLimits struct {
	MaxSites           int `json:"max_sites"`
	MaxAgentsPerSite   int `json:"max_agents_per_site"`
	MaxVMsPerAgent     int `json:"max_vms_per_agent"`
	MaxConcurrentPlans int `json:"max_concurrent_plans"`
	MaxAPIKeys         int `json:"max_api_keys"`
}

// DefaultQuotaLimits returns sensible default quota limits
func DefaultQuotaLimits() QuotaLimits {
	return QuotaLimits{
		MaxSites:           10,
		MaxAgentsPerSite:   100,
		MaxVMsPerAgent:     50,
		MaxConcurrentPlans: 100,
		MaxAPIKeys:         20,
	}
}

// QuotaUsage represents current resource usage for a tenant
type QuotaUsage struct {
	Sites           int `json:"sites"`
	Agents          int `json:"agents"`
	VMs             int `json:"vms"`
	ActivePlans     int `json:"active_plans"`
	APIKeys         int `json:"api_keys"`
	TotalExecutions int `json:"total_executions,omitempty"`
}

// UsageRepo defines the interface for getting tenant usage information
type UsageRepo interface {
	GetTenantUsage(ctx context.Context, tenantID string) (*QuotaUsage, error)
}

// QuotaUsageProvider is a function type that provides quota usage
type QuotaUsageProvider func(ctx context.Context, tenantID string) (*QuotaUsage, error)

// QuotaManager manages tenant quotas and usage tracking
type QuotaManager struct {
	mu         sync.RWMutex
	limits     map[string]QuotaLimits // tenantID -> limits
	defaultLim QuotaLimits
	repo       UsageRepo
	provider   QuotaUsageProvider
}

// NewQuotaManager creates a new QuotaManager with default limits
func NewQuotaManager(repo UsageRepo) *QuotaManager {
	return &QuotaManager{
		limits:     make(map[string]QuotaLimits),
		defaultLim: DefaultQuotaLimits(),
		repo:       repo,
	}
}

// NewQuotaManagerWithProvider creates a new QuotaManager with a custom provider function
func NewQuotaManagerWithProvider(provider QuotaUsageProvider) *QuotaManager {
	return &QuotaManager{
		limits:     make(map[string]QuotaLimits),
		defaultLim: DefaultQuotaLimits(),
		provider:   provider,
	}
}

// SetLimits sets custom quota limits for a specific tenant
func (qm *QuotaManager) SetLimits(tenantID string, limits QuotaLimits) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.limits[tenantID] = limits
}

// GetLimits returns the quota limits for a tenant
// Returns default limits if no custom limits are set
func (qm *QuotaManager) GetLimits(tenantID string) QuotaLimits {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	if limits, ok := qm.limits[tenantID]; ok {
		return limits
	}
	return qm.defaultLim
}

// GetUsage returns the current quota usage for a tenant
func (qm *QuotaManager) GetUsage(ctx context.Context, tenantID string) (*QuotaUsage, error) {
	if tenantID == "" {
		return nil, errors.New("tenant ID is required")
	}
	if qm.provider != nil {
		return qm.provider(ctx, tenantID)
	}
	if qm.repo != nil {
		return qm.repo.GetTenantUsage(ctx, tenantID)
	}
	return nil, errors.New("no usage provider configured")
}

// ResourceType represents the type of resource being checked
type QuotaResourceType string

const (
	QuotaResourceSite      QuotaResourceType = "site"
	QuotaResourceAgent     QuotaResourceType = "agent"
	QuotaResourceVM        QuotaResourceType = "vm"
	QuotaResourcePlan      QuotaResourceType = "plan"
	QuotaResourceAPIKey    QuotaResourceType = "api_key"
)

// CheckQuota checks if the tenant can create a new resource of the given type
// Returns ErrQuotaExceeded if the quota would be exceeded
func (qm *QuotaManager) CheckQuota(ctx context.Context, tenantID string, resourceType QuotaResourceType) error {
	limits := qm.GetLimits(tenantID)
	usage, err := qm.GetUsage(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get quota usage: %w", err)
	}

	var limit int
	var current int
	var resourceName string

	switch resourceType {
	case QuotaResourceSite:
		limit = limits.MaxSites
		current = usage.Sites
		resourceName = "sites"
	case QuotaResourceAgent:
		limit = limits.MaxAgentsPerSite
		current = usage.Agents
		resourceName = "agents"
	case QuotaResourceVM:
		limit = limits.MaxVMsPerAgent
		current = usage.VMs
		resourceName = "VMs"
	case QuotaResourcePlan:
		limit = limits.MaxConcurrentPlans
		current = usage.ActivePlans
		resourceName = "active plans"
	case QuotaResourceAPIKey:
		limit = limits.MaxAPIKeys
		current = usage.APIKeys
		resourceName = "API keys"
	default:
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}

	if current >= limit {
		return fmt.Errorf("%w: cannot create %s (limit: %d, current: %d)", 
			ErrQuotaExceeded, resourceName, limit, current)
	}

	return nil
}

// CheckQuotaWithCount checks quota with a specific count (for batch operations)
func (qm *QuotaManager) CheckQuotaWithCount(ctx context.Context, tenantID string, resourceType QuotaResourceType, count int) error {
	if count <= 0 {
		return nil
	}

	limits := qm.GetLimits(tenantID)
	usage, err := qm.GetUsage(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get quota usage: %w", err)
	}

	var limit int
	var current int
	var resourceName string

	switch resourceType {
	case QuotaResourceSite:
		limit = limits.MaxSites
		current = usage.Sites
		resourceName = "sites"
	case QuotaResourceAgent:
		limit = limits.MaxAgentsPerSite
		current = usage.Agents
		resourceName = "agents"
	case QuotaResourceVM:
		limit = limits.MaxVMsPerAgent
		current = usage.VMs
		resourceName = "VMs"
	case QuotaResourcePlan:
		limit = limits.MaxConcurrentPlans
		current = usage.ActivePlans
		resourceName = "active plans"
	case QuotaResourceAPIKey:
		limit = limits.MaxAPIKeys
		current = usage.APIKeys
		resourceName = "API keys"
	default:
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}

	if current+count > limit {
		return fmt.Errorf("%w: cannot create %d %s (limit: %d, current: %d, would be: %d)", 
			ErrQuotaExceeded, count, resourceName, limit, current, current+count)
	}

	return nil
}

// GetQuotaUsagePercent returns the quota usage percentage for each resource type
func (qm *QuotaManager) GetQuotaUsagePercent(ctx context.Context, tenantID string) (map[string]float64, error) {
	limits := qm.GetLimits(tenantID)
	usage, err := qm.GetUsage(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	percentages := make(map[string]float64)
	
	if limits.MaxSites > 0 {
		percentages["sites"] = float64(usage.Sites) / float64(limits.MaxSites) * 100
	}
	if limits.MaxAgentsPerSite > 0 {
		percentages["agents"] = float64(usage.Agents) / float64(limits.MaxAgentsPerSite) * 100
	}
	if limits.MaxVMsPerAgent > 0 {
		percentages["vms"] = float64(usage.VMs) / float64(limits.MaxVMsPerAgent) * 100
	}
	if limits.MaxConcurrentPlans > 0 {
		percentages["active_plans"] = float64(usage.ActivePlans) / float64(limits.MaxConcurrentPlans) * 100
	}
	if limits.MaxAPIKeys > 0 {
		percentages["api_keys"] = float64(usage.APIKeys) / float64(limits.MaxAPIKeys) * 100
	}

	return percentages, nil
}

// QuotaStatus combines limits, usage, and percentages for a tenant
type QuotaStatus struct {
	Limits      QuotaLimits         `json:"limits"`
	Usage       QuotaUsage          `json:"usage"`
	Percentages map[string]float64  `json:"quota_usage_percent"`
}

// GetQuotaStatus returns the complete quota status for a tenant
func (qm *QuotaManager) GetQuotaStatus(ctx context.Context, tenantID string) (*QuotaStatus, error) {
	limits := qm.GetLimits(tenantID)
	usage, err := qm.GetUsage(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	percentages, err := qm.GetQuotaUsagePercent(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &QuotaStatus{
		Limits:      limits,
		Usage:       *usage,
		Percentages: percentages,
	}, nil
}
