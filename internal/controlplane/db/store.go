package store

import (
	"context"
	"time"
)

type Tenant struct {
	ID            string    `json:"id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	PrimaryRegion string    `json:"primary_region"`
	RetentionDays int       `json:"data_retention_days"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Site struct {
	ID                string     `json:"id"`
	TenantID          string     `json:"tenant_id"`
	Name              string     `json:"name"`
	ExternalKey       string     `json:"external_key,omitempty"`
	LocationCountry   string     `json:"location_country_code,omitempty"`
	ConnectivityState string     `json:"connectivity_state"`
	LastHeartbeatAt   *time.Time `json:"last_heartbeat_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type Host struct {
	ID                       string     `json:"id"`
	TenantID                 string     `json:"tenant_id"`
	SiteID                   string     `json:"site_id"`
	Hostname                 string     `json:"hostname"`
	CPUCoresTotal            int        `json:"cpu_cores_total"`
	MemoryBytesTotal         int64      `json:"memory_bytes_total"`
	StorageBytesTotal        int64      `json:"storage_bytes_total"`
	KVMAvailable             bool       `json:"kvm_available"`
	CloudHypervisorAvailable bool       `json:"cloud_hypervisor_available"`
	LastFactsAt              *time.Time `json:"last_facts_at,omitempty"`
	AgentState               string     `json:"agent_state,omitempty"`
	AgentLastHeartbeatAt     *time.Time `json:"agent_last_heartbeat_at,omitempty"`
}

type MicroVM struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	SiteID           string     `json:"site_id"`
	HostID           string     `json:"host_id"`
	Name             string     `json:"name"`
	State            string     `json:"state"`
	VCPUCount        int        `json:"vcpu_count"`
	MemoryMiB        int64      `json:"memory_mib"`
	LastTransitionAt *time.Time `json:"last_transition_at,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type APIKey struct {
	ID        string
	TenantID  string
	Name      string
	KeyHash   string
	ExpiresAt *time.Time
}

type EnrollmentToken struct {
	ID        string
	TenantID  string
	SiteID    string
	TokenHash string
	ExpiresAt time.Time
}

type Agent struct {
	ID               string
	TenantID         string
	SiteID           string
	HostID           string
	CertSerial       string
	RefreshTokenHash string
	AgentVersion     string
	OS               string
	Arch             string
	KernelVersion    string
	State            string
	LastHeartbeatAt  *time.Time
}

type Plan struct {
	ID             string      `json:"id"`
	TenantID       string      `json:"tenant_id"`
	SiteID         string      `json:"site_id"`
	IdempotencyKey string      `json:"idempotency_key"`
	PlanVersion    int64       `json:"plan_version"`
	Status         string      `json:"status"`
	OperationsJSON []byte      `json:"operations_json"`
	CreatedAt      time.Time   `json:"created_at"`
	Executions     []Execution `json:"executions,omitempty"`
}

type PlanAction struct {
	ID            string `json:"id"`
	PlanID        string `json:"plan_id"`
	OperationID   string `json:"operation_id"`
	OperationType string `json:"operation"`
	VMID          string `json:"vm_id,omitempty"`
	PayloadJSON   []byte `json:"payload_json"`
}

type LeasedPlan struct {
	PlanID      string       `json:"plan_id"`
	ExecutionID string       `json:"execution_id"`
	Actions     []PlanAction `json:"actions"`
}

type Execution struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	SiteID        string     `json:"site_id"`
	HostID        string     `json:"host_id,omitempty"`
	AgentID       string     `json:"agent_id,omitempty"`
	PlanID        string     `json:"plan_id"`
	VMID          string     `json:"vm_id,omitempty"`
	OperationID   string     `json:"operation_id"`
	OperationType string     `json:"operation_type"`
	State         string     `json:"state"`
	ErrorCode     string     `json:"error_code,omitempty"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type ExecutionLog struct {
	ID          int64     `json:"id"`
	TenantID    string    `json:"tenant_id"`
	ExecutionID string    `json:"execution_id"`
	Sequence    int64     `json:"sequence"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	EmittedAt   time.Time `json:"emitted_at"`
	IngestedAt  time.Time `json:"ingested_at"`
}

type Heartbeat struct {
	AgentID                  string
	HeartbeatSeq             int64
	AgentVersion             string
	OS                       string
	Arch                     string
	KernelVersion            string
	Hostname                 string
	CPUCoresTotal            int
	MemoryBytesTotal         int64
	StorageBytesTotal        int64
	KVMAvailable             bool
	CloudHypervisorAvailable bool
	MicroVMs                 []MicroVMHeartbeat
	ExecutionUpdates         []ExecutionUpdate
}

type MicroVMHeartbeat struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	State     string    `json:"state"`
	VCPUCount int       `json:"vcpu_count"`
	MemoryMiB int64     `json:"memory_mib"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ExecutionUpdate struct {
	ExecutionID  string    `json:"execution_id"`
	State        string    `json:"state"`
	ErrorCode    string    `json:"error_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type LogIngest struct {
	AgentID string
	Entries []LogIngestEntry
}

type LogIngestEntry struct {
	ExecutionID string    `json:"execution_id"`
	Sequence    int64     `json:"sequence"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	EmittedAt   time.Time `json:"emitted_at"`
}

type APIKeyValidation struct {
	TenantID string
}

type ApplyPlanInput struct {
	TenantID        string
	SiteID          string
	IdempotencyKey  string
	ClientRequestID string
	Actions         []ApplyPlanAction
}

type ApplyPlanAction struct {
	OperationID string `json:"operation_id"`
	Operation   string `json:"operation"`
	VMID        string `json:"vm_id,omitempty"`
	Name        string `json:"name,omitempty"`
	VCPUCount   int    `json:"vcpu_count,omitempty"`
	MemoryMiB   int64  `json:"memory_mib,omitempty"`
}

type ApplyPlanResult struct {
	Plan         Plan        `json:"plan"`
	Executions   []Execution `json:"executions"`
	Deduplicated bool        `json:"deduplicated"`
}

type PlanResultReport struct {
	PlanID      string                 `json:"plan_id"`
	ExecutionID string                 `json:"execution_id"`
	Results     []PlanActionResultItem `json:"results"`
}

type PlanActionResultItem struct {
	ActionID   string    `json:"action_id"`
	OK         bool      `json:"ok"`
	ErrorCode  string    `json:"error_code,omitempty"`
	Message    string    `json:"message,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

type TokenConsumeResult struct {
	TokenID  string
	TenantID string
	SiteID   string
}

type Repo interface {
	CreateTenant(ctx context.Context, t Tenant) (Tenant, error)
	CreateAPIKey(ctx context.Context, key APIKey) (APIKey, error)
	ValidateAPIKey(ctx context.Context, keyHash string) (APIKeyValidation, error)
	CreateSite(ctx context.Context, site Site) (Site, error)
	ListSites(ctx context.Context, tenantID string) ([]Site, error)
	IssueEnrollmentToken(ctx context.Context, token EnrollmentToken) (EnrollmentToken, error)
	ConsumeEnrollmentToken(ctx context.Context, tokenHash string, now time.Time) (TokenConsumeResult, error)
	CreateAgentFromEnrollment(ctx context.Context, tokenID string, agent Agent, hostname string) (Agent, error)
	GetAgentByID(ctx context.Context, agentID string) (Agent, error)
	IngestHeartbeat(ctx context.Context, hb Heartbeat) error
	ApplyPlan(ctx context.Context, input ApplyPlanInput) (ApplyPlanResult, error)
	LeasePendingPlans(ctx context.Context, agentID string, limit int, leaseTTL time.Duration) ([]LeasedPlan, error)
	ReportPlanResult(ctx context.Context, agentID string, report PlanResultReport) error
	IngestLogs(ctx context.Context, req LogIngest) (accepted int64, dropped int64, err error)
	SweepOfflineAgents(ctx context.Context, staleBefore time.Time) (int64, error)
	ListHosts(ctx context.Context, tenantID, siteID string) ([]Host, error)
	ListVMs(ctx context.Context, tenantID, siteID string) ([]MicroVM, error)
	ListExecutionLogs(ctx context.Context, tenantID, executionID string, limit int) ([]ExecutionLog, error)
	WriteAudit(ctx context.Context, tenantID, siteID, actorType, actorID, action, resourceType, resourceID, requestID, sourceIP string, metadata []byte) error
	SiteBelongsToTenant(ctx context.Context, siteID, tenantID string) (bool, error)
	ExecutionBelongsToTenant(ctx context.Context, executionID, tenantID string) (bool, error)
}
