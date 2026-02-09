package tenant

import (
	"context"
	"errors"
	"fmt"
)

// ErrTenantIsolationViolation is returned when a tenant tries to access a resource they don't own
var ErrTenantIsolationViolation = errors.New("tenant isolation violation: resource does not belong to tenant")

// ResourceType represents the type of a tenant resource
type ResourceType string

const (
	ResourceTypeSite      ResourceType = "site"
	ResourceTypeAgent     ResourceType = "agent"
	ResourceTypeVM        ResourceType = "vm"
	ResourceTypePlan      ResourceType = "plan"
	ResourceTypeExecution ResourceType = "execution"
	ResourceTypeAPIKey    ResourceType = "api_key"
)

// Resource represents a tenant-scoped resource
type Resource struct {
	TenantID string
	Type     ResourceType
	ID       string
}

// Repo defines the interface for tenant-related repository operations
type Repo interface {
	SiteBelongsToTenant(ctx context.Context, siteID, tenantID string) (bool, error)
	ExecutionBelongsToTenant(ctx context.Context, executionID, tenantID string) (bool, error)
}

// IsolationEnforcer enforces tenant isolation rules
type IsolationEnforcer struct {
	repo Repo
}

// NewIsolationEnforcer creates a new IsolationEnforcer
func NewIsolationEnforcer(repo Repo) *IsolationEnforcer {
	return &IsolationEnforcer{repo: repo}
}

// EnforceTenantAccess checks if the given tenantID owns the specified resource
// Returns ErrTenantIsolationViolation if the resource does not belong to the tenant
func (e *IsolationEnforcer) EnforceTenantAccess(ctx context.Context, tenantID string, resource Resource) error {
	if tenantID == "" {
		return errors.New("tenant ID is required")
	}
	if resource.ID == "" {
		return errors.New("resource ID is required")
	}

	// Direct tenant ID comparison for in-memory resources
	if resource.TenantID != "" && resource.TenantID != tenantID {
		return fmt.Errorf("%w: expected tenant %s, got %s", ErrTenantIsolationViolation, resource.TenantID, tenantID)
	}

	// For resources loaded from DB, verify ownership
	switch resource.Type {
	case ResourceTypeSite:
		return e.enforceSiteAccess(ctx, tenantID, resource.ID)
	case ResourceTypeExecution:
		return e.enforceExecutionAccess(ctx, tenantID, resource.ID)
	case ResourceTypeAgent, ResourceTypeVM, ResourceTypePlan, ResourceTypeAPIKey:
		// These are typically verified through their parent resources (site)
		// or by direct tenant_id check in the resource struct
		if resource.TenantID != "" && resource.TenantID == tenantID {
			return nil
		}
		return fmt.Errorf("%w: %s access denied for tenant %s", ErrTenantIsolationViolation, resource.Type, tenantID)
	default:
		return fmt.Errorf("unknown resource type: %s", resource.Type)
	}
}

// enforceSiteAccess checks if the tenant owns the site
func (e *IsolationEnforcer) enforceSiteAccess(ctx context.Context, tenantID, siteID string) error {
	belongs, err := e.repo.SiteBelongsToTenant(ctx, siteID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to verify site ownership: %w", err)
	}
	if !belongs {
		return fmt.Errorf("%w: site %s does not belong to tenant %s", ErrTenantIsolationViolation, siteID, tenantID)
	}
	return nil
}

// enforceExecutionAccess checks if the tenant owns the execution
func (e *IsolationEnforcer) enforceExecutionAccess(ctx context.Context, tenantID, executionID string) error {
	belongs, err := e.repo.ExecutionBelongsToTenant(ctx, executionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to verify execution ownership: %w", err)
	}
	if !belongs {
		return fmt.Errorf("%w: execution %s does not belong to tenant %s", ErrTenantIsolationViolation, executionID, tenantID)
	}
	return nil
}

// EnforceTenantAccessWithResource loads the resource and checks tenant ownership
// This is useful when you have the resource ID but need to verify it belongs to the tenant
func (e *IsolationEnforcer) EnforceTenantAccessWithResource(ctx context.Context, tenantID string, resourceType ResourceType, resourceID string) error {
	resource := Resource{
		ID:   resourceID,
		Type: resourceType,
	}
	return e.EnforceTenantAccess(ctx, tenantID, resource)
}
