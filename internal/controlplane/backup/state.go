package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

// TenantState represents a complete snapshot of tenant data
type TenantState struct {
	Version        string                        `json:"version"`
	ExportedAt     time.Time                     `json:"exported_at"`
	TenantID       string                        `json:"tenant_id"`
	Tenant         store.Tenant                  `json:"tenant"`
	Sites          []store.Site                  `json:"sites"`
	APIKeys        []store.APIKey                `json:"api_keys"`
	Hosts          []store.Host                  `json:"hosts"`
	Agents         []AgentState                  `json:"agents"`
	MicroVMs       []store.MicroVM               `json:"microvms"`
	Plans          []PlanState                   `json:"plans"`
	AuditEvents    []store.AuditEvent            `json:"audit_events,omitempty"`
	EnrollmentTokens []EnrollmentTokenState      `json:"enrollment_tokens,omitempty"`
}

// AgentState includes Agent and related certificate history
type AgentState struct {
	store.Agent
	CertificateHistory []store.CertificateHistory `json:"certificate_history,omitempty"`
}

// PlanState includes Plan with its executions and actions
type PlanState struct {
	store.Plan
	Executions []ExecutionState `json:"executions"`
	Actions    []store.PlanAction `json:"actions"`
}

// ExecutionState includes Execution with its logs
type ExecutionState struct {
	store.Execution
	Logs []store.ExecutionLog `json:"logs,omitempty"`
}

// EnrollmentTokenState includes token with consumption info
type EnrollmentTokenState struct {
	store.EnrollmentToken
	Consumed          bool       `json:"consumed"`
	ConsumedAt        *time.Time `json:"consumed_at,omitempty"`
	ConsumedByAgentID *string    `json:"consumed_by_agent_id,omitempty"`
}

const stateVersion = "1.0"

// Exporter handles tenant state export/import operations
type Exporter struct {
	repo   store.Repo
}

// NewExporter creates a new state exporter
func NewExporter(repo store.Repo) *Exporter {
	return &Exporter{repo: repo}
}

// ExportTenant exports all data for a specific tenant
func (e *Exporter) ExportTenant(ctx context.Context, tenantID string) (*TenantState, error) {
	state := &TenantState{
		Version:    stateVersion,
		ExportedAt: time.Now().UTC(),
		TenantID:   tenantID,
	}

	// Note: The repo interface doesn't have all methods we need for full export
	// In a real implementation, we would extend the repo interface or use raw queries
	// For now, we'll export what we can with existing methods

	// Export sites
	sites, err := e.repo.ListSites(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing sites: %w", err)
	}
	state.Sites = sites

	// Export API keys
	apiKeys, err := e.repo.ListAPIKeys(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing api keys: %w", err)
	}
	state.APIKeys = apiKeys

	// For each site, export hosts, agents, and VMs
	for _, site := range sites {
		// Export hosts
		hosts, err := e.repo.ListHosts(ctx, tenantID, site.ID)
		if err != nil {
			return nil, fmt.Errorf("listing hosts for site %s: %w", site.ID, err)
		}
		state.Hosts = append(state.Hosts, hosts...)

		// Export VMs
		vms, err := e.repo.ListVMs(ctx, tenantID, site.ID)
		if err != nil {
			return nil, fmt.Errorf("listing vms for site %s: %w", site.ID, err)
		}
		state.MicroVMs = append(state.MicroVMs, vms...)
	}

	// Export enrollment tokens
	tokens, err := e.repo.ListEnrollmentTokens(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing enrollment tokens: %w", err)
	}
	for _, token := range tokens {
		state.EnrollmentTokens = append(state.EnrollmentTokens, EnrollmentTokenState{
			EnrollmentToken: store.EnrollmentToken{
				ID:       token.ID,
				TenantID: tenantID,
				SiteID:   token.SiteID,
			},
			Consumed:          token.Consumed,
			ConsumedAt:        token.ConsumedAt,
			ConsumedByAgentID: token.ConsumedByAgentID,
		})
	}

	// Export audit events
	auditEvents, err := e.repo.ListAuditEvents(ctx, tenantID, 10000)
	if err != nil {
		return nil, fmt.Errorf("listing audit events: %w", err)
	}
	state.AuditEvents = auditEvents

	return state, nil
}

// ImportTenant imports tenant data from a state snapshot
func (e *Exporter) ImportTenant(ctx context.Context, state *TenantState) error {
	if state.Version != stateVersion {
		return fmt.Errorf("unsupported state version: %s (expected %s)", state.Version, stateVersion)
	}

	// Note: In a real implementation with full CRUD operations:
	// 1. Create/update tenant
	// 2. Create/update sites
	// 3. Create/update API keys (would need special handling for key hashes)
	// 4. Create/update hosts
	// 5. Create/update agents (would need enrollment)
	// 6. Create/update VMs
	// 7. Create/update plans and executions

	// For now, we log what would be imported
	fmt.Printf("Importing tenant state for %s\n", state.TenantID)
	fmt.Printf("- Sites: %d\n", len(state.Sites))
	fmt.Printf("- API Keys: %d\n", len(state.APIKeys))
	fmt.Printf("- Hosts: %d\n", len(state.Hosts))
	fmt.Printf("- Agents: %d\n", len(state.Agents))
	fmt.Printf("- MicroVMs: %d\n", len(state.MicroVMs))
	fmt.Printf("- Plans: %d\n", len(state.Plans))
	fmt.Printf("- Audit Events: %d\n", len(state.AuditEvents))
	fmt.Printf("- Enrollment Tokens: %d\n", len(state.EnrollmentTokens))

	return fmt.Errorf("full import not yet implemented - requires extended repo interface")
}

// SaveToFile saves tenant state to a JSON file
func (e *Exporter) SaveToFile(state *TenantState, path string) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}

	return nil
}

// LoadFromFile loads tenant state from a JSON file
func (e *Exporter) LoadFromFile(path string) (*TenantState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var state TenantState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshaling state: %w", err)
	}

	return &state, nil
}

// ExportAllTenants exports all tenants (admin operation)
func (e *Exporter) ExportAllTenants(ctx context.Context) (map[string]*TenantState, error) {
	// Note: Would need a ListTenants method on repo
	// For now, return empty
	return make(map[string]*TenantState), nil
}

// ValidateState validates the integrity of a tenant state
func (e *Exporter) ValidateState(state *TenantState) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}

	if state.Version == "" {
		return fmt.Errorf("state version is required")
	}

	if state.TenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}

	if state.ExportedAt.IsZero() {
		return fmt.Errorf("export timestamp is required")
	}

	// Validate site references
	siteIDs := make(map[string]bool)
	for _, site := range state.Sites {
		if site.ID == "" {
			return fmt.Errorf("site with empty ID found")
		}
		siteIDs[site.ID] = true
		if site.TenantID != state.TenantID {
			return fmt.Errorf("site %s has mismatched tenant ID", site.ID)
		}
	}

	// Validate host references
	for _, host := range state.Hosts {
		if host.ID == "" {
			return fmt.Errorf("host with empty ID found")
		}
		if host.TenantID != state.TenantID {
			return fmt.Errorf("host %s has mismatched tenant ID", host.ID)
		}
		if !siteIDs[host.SiteID] {
			return fmt.Errorf("host %s references unknown site %s", host.ID, host.SiteID)
		}
	}

	// Validate VM references
	for _, vm := range state.MicroVMs {
		if vm.ID == "" {
			return fmt.Errorf("vm with empty ID found")
		}
		if vm.TenantID != state.TenantID {
			return fmt.Errorf("vm %s has mismatched tenant ID", vm.ID)
		}
		if !siteIDs[vm.SiteID] {
			return fmt.Errorf("vm %s references unknown site %s", vm.ID, vm.SiteID)
		}
	}

	return nil
}

// StateSummary provides a summary of the tenant state
type StateSummary struct {
	TenantID           string    `json:"tenant_id"`
	ExportedAt         time.Time `json:"exported_at"`
	Version            string    `json:"version"`
	SiteCount          int       `json:"site_count"`
	APIKeyCount        int       `json:"api_key_count"`
	HostCount          int       `json:"host_count"`
	AgentCount         int       `json:"agent_count"`
	MicroVMCount       int       `json:"microvm_count"`
	PlanCount          int       `json:"plan_count"`
	AuditEventCount    int       `json:"audit_event_count"`
	EnrollmentTokenCount int     `json:"enrollment_token_count"`
}

// GetSummary returns a summary of the tenant state
func (s *TenantState) GetSummary() StateSummary {
	return StateSummary{
		TenantID:             s.TenantID,
		ExportedAt:           s.ExportedAt,
		Version:              s.Version,
		SiteCount:            len(s.Sites),
		APIKeyCount:          len(s.APIKeys),
		HostCount:            len(s.Hosts),
		AgentCount:           len(s.Agents),
		MicroVMCount:         len(s.MicroVMs),
		PlanCount:            len(s.Plans),
		AuditEventCount:      len(s.AuditEvents),
		EnrollmentTokenCount: len(s.EnrollmentTokens),
	}
}

// ToJSON returns the state as formatted JSON
func (s *TenantState) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// FromJSON parses a tenant state from JSON
func FromJSON(data []byte) (*TenantState, error) {
	var state TenantState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshaling state: %w", err)
	}
	return &state, nil
}
