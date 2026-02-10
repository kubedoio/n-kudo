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

// User represents a tenant user with authentication
type User struct {
	ID                string     `json:"id"`
	TenantID          string     `json:"tenant_id"`
	Email             string     `json:"email"`
	DisplayName       string     `json:"display_name"`
	Role              string     `json:"role"` // OWNER, ADMIN, OPERATOR, VIEWER
	IsActive          bool       `json:"is_active"`
	EmailVerified     bool       `json:"email_verified"`
	PasswordHash      string     `json:"-"` // Never expose in JSON
	LastLoginAt       *time.Time `json:"last_login_at,omitempty"`
	PasswordChangedAt time.Time  `json:"password_changed_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
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
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"` // Note: KeyHash is intentionally omitted from JSON
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type EnrollmentToken struct {
	ID        string
	TenantID  string
	SiteID    string
	TokenHash string
	ExpiresAt time.Time
}

type EnrollmentTokenWithStatus struct {
	ID                string     `json:"id"`
	SiteID            string     `json:"site_id"`
	SiteName          string     `json:"site_name"`
	CreatedAt         time.Time  `json:"created_at"`
	ExpiresAt         time.Time  `json:"expires_at"`
	Consumed          bool       `json:"consumed"`
	ConsumedAt        *time.Time `json:"consumed_at,omitempty"`
	ConsumedByAgentID *string    `json:"consumed_by_agent_id,omitempty"`
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
	Deduplicated   bool        `json:"deduplicated,omitempty"`
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

type ExecutionWithTimestamps struct {
	ID            string    `json:"id"`
	PlanID        string    `json:"plan_id"`
	OperationID   string    `json:"operation_id"`
	OperationType string    `json:"operation_type"`
	State         string    `json:"state"`
	VMID          string    `json:"vm_id"`
	ErrorCode     *string   `json:"error_code,omitempty"`
	ErrorMessage  *string   `json:"error_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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

type CertificateHistory struct {
	ID           string     `json:"id"`
	AgentID      string     `json:"agent_id"`
	Serial       string     `json:"serial"`
	IssuedAt     time.Time  `json:"issued_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
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

// AuditEvent represents an audit log entry with chain integrity support
type AuditEvent struct {
	ID           int64     `json:"id"`
	TenantID     string    `json:"tenant_id"`
	SiteID       string    `json:"site_id,omitempty"`
	ActorType    string    `json:"actor_type"`
	ActorUserID  *string   `json:"actor_user_id,omitempty"`
	ActorAgentID *string   `json:"actor_agent_id,omitempty"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	RequestID    string    `json:"request_id,omitempty"`
	SourceIP     string    `json:"source_ip,omitempty"`
	MetadataJSON []byte    `json:"metadata_json,omitempty"`
	OccurredAt   time.Time `json:"occurred_at"`
	// Chain integrity fields
	PrevHash   string `json:"prev_hash"`
	EntryHash  string `json:"entry_hash"`
	ChainValid bool   `json:"chain_valid"`
}

// TenantUsage represents current resource usage for a tenant
type TenantUsage struct {
	Sites       int `json:"sites"`
	Agents      int `json:"agents"`
	VMs         int `json:"vms"`
	ActivePlans int `json:"active_plans"`
	APIKeys     int `json:"api_keys"`
}

// ProjectInvitation represents an invitation to join a project
type ProjectInvitation struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	Email            string     `json:"email"`
	Role             string     `json:"role"`
	InvitedByUserID  string     `json:"invited_by_user_id"`
	TokenHash        string     `json:"-"`
	Status           string     `json:"status"`
	ExpiresAt        time.Time  `json:"expires_at"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty"`
	DeclinedAt       *time.Time `json:"declined_at,omitempty"`
	CancelledAt      *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ProjectInvitationWithProject includes project details for user invitations
type ProjectInvitationWithProject struct {
	ProjectInvitation
	ProjectName string `json:"project_name"`
	ProjectSlug string `json:"project_slug"`
}

// QuotaLimits represents quota limits for a tenant
type QuotaLimits struct {
	MaxSites           int `json:"max_sites"`
	MaxAgentsPerSite   int `json:"max_agents_per_site"`
	MaxVMsPerAgent     int `json:"max_vms_per_agent"`
	MaxConcurrentPlans int `json:"max_concurrent_plans"`
	MaxAPIKeys         int `json:"max_api_keys"`
}

// AuditEventInput represents the input for creating a new audit event
type AuditEventInput struct {
	TenantID     string
	SiteID       string
	ActorType    string
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
	RequestID    string
	SourceIP     string
	Metadata     []byte
}

// CRLEntry represents a revoked certificate entry
type CRLEntry struct {
	SerialNumber string    `json:"serial_number"`
	RevokedAt    time.Time `json:"revoked_at"`
	Reason       int       `json:"reason"`
	AgentID      string    `json:"agent_id,omitempty"`
}

type Repo interface {
	Close() error
	CreateTenant(ctx context.Context, t Tenant) (Tenant, error)
	ListTenants(ctx context.Context) ([]Tenant, error)
	GetTenantByID(ctx context.Context, tenantID string) (Tenant, error)

	// User authentication methods
	CreateUser(ctx context.Context, user User) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByEmailAndTenant(ctx context.Context, email, tenantID string) (User, error)
	GetUserByID(ctx context.Context, tenantID, userID string) (User, error)
	UpdateUserLastLogin(ctx context.Context, tenantID, userID string) error
	UpdateUserPassword(ctx context.Context, tenantID, userID, passwordHash string) error
	EmailExists(ctx context.Context, email string) (bool, error)
	
	// Email verification methods
	CreateEmailVerificationToken(ctx context.Context, userID, tenantID, tokenHash string, expiresAt time.Time) error
	VerifyEmailToken(ctx context.Context, tokenHash string) (userID, tenantID string, err error)
	MarkEmailVerified(ctx context.Context, tenantID, userID string) error
	
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
	ListEnrollmentTokens(ctx context.Context, tenantID string) ([]EnrollmentTokenWithStatus, error)
	ListExecutions(ctx context.Context, tenantID, siteID string, statuses []string, limit int) ([]ExecutionWithTimestamps, error)
	ListAPIKeys(ctx context.Context, tenantID string) ([]APIKey, error)
	DeleteAPIKey(ctx context.Context, tenantID, keyID string) error
	UnenrollAgent(ctx context.Context, agentID string) error
	UpdateAgentCertificate(ctx context.Context, agentID, certSerial, refreshTokenHash string) error
	ListCertificateHistory(ctx context.Context, agentID string, limit int) ([]CertificateHistory, error)
	RecordCertificateIssuance(ctx context.Context, history CertificateHistory) error

	// Audit chain integrity methods
	GetLastAuditEvent(ctx context.Context) (*AuditEvent, error)
	WriteAuditEvent(ctx context.Context, event *AuditEvent) error
	UpdateAuditEventValidity(ctx context.Context, id int64, valid bool) error
	ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]AuditEvent, error)

	// CRL methods
	RevokeCertificate(ctx context.Context, serial string, reason int, agentID string) error
	IsCertificateRevoked(ctx context.Context, serial string) (bool, error)
	ListRevokedCertificates(ctx context.Context) ([]CRLEntry, error)

	// Tenant quota and usage methods
	GetTenantUsage(ctx context.Context, tenantID string) (*TenantUsage, error)
	GetTenantLimits(ctx context.Context, tenantID string) (*QuotaLimits, error)
	SetTenantLimits(ctx context.Context, tenantID string, limits QuotaLimits) error

	// Team invitation methods
	CreateInvitation(ctx context.Context, invitation ProjectInvitation) error
	GetInvitationByToken(ctx context.Context, tokenHash string) (*ProjectInvitation, error)
	ListPendingInvitations(ctx context.Context, tenantID string) ([]ProjectInvitation, error)
	ListUserInvitations(ctx context.Context, email string) ([]ProjectInvitationWithProject, error)
	AcceptInvitation(ctx context.Context, invitationID, userID string) error
	DeclineInvitation(ctx context.Context, invitationID string) error
	CancelInvitation(ctx context.Context, tenantID, invitationID string) error
	GetInvitationByID(ctx context.Context, tenantID, invitationID string) (*ProjectInvitation, error)

	// VXLAN network methods
	CreateVXLANNetwork(ctx context.Context, tenantID, siteID string, network VXLANNetwork) (VXLANNetwork, error)
	ListVXLANNetworks(ctx context.Context, tenantID, siteID string) ([]VXLANNetwork, error)
	GetVXLANNetwork(ctx context.Context, tenantID, networkID string) (VXLANNetwork, error)
	GetVXLANNetworkByVNI(ctx context.Context, tenantID string, vni int) (VXLANNetwork, error)
	DeleteVXLANNetwork(ctx context.Context, tenantID, networkID string) error
	VXLANNetworkBelongsToTenant(ctx context.Context, networkID, tenantID string) (bool, error)

	// VXLAN tunnel methods
	CreateVXLANTunnel(ctx context.Context, tunnel VXLANTunnel) (VXLANTunnel, error)
	ListVXLANTunnels(ctx context.Context, networkID string) ([]VXLANTunnel, error)
	GetVXLANTunnel(ctx context.Context, networkID, hostID string) (VXLANTunnel, error)
	UpdateVXLANTunnelStatus(ctx context.Context, tunnelID string, status string) error
	DeleteVXLANTunnel(ctx context.Context, tunnelID string) error

	// VM network attachment methods
	AttachVMToNetwork(ctx context.Context, attachment VMNetworkAttachment) (VMNetworkAttachment, error)
	DetachVMFromNetwork(ctx context.Context, vmID, networkID string) error
	ListVMNetworkAttachments(ctx context.Context, vmID string) ([]VMNetworkAttachment, error)
	ListNetworkVMAttachments(ctx context.Context, networkID string) ([]VMNetworkAttachment, error)
}

// VXLANNetwork represents a VXLAN network
type VXLANNetwork struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	SiteID    string    `json:"site_id"`
	Name      string    `json:"name"`
	VNI       int       `json:"vni"`
	CIDR      string    `json:"cidr"`
	Gateway   string    `json:"gateway,omitempty"`
	MTU       int       `json:"mtu"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// VXLANTunnel represents a VXLAN tunnel endpoint on a host
type VXLANTunnel struct {
	ID        string    `json:"id"`
	NetworkID string    `json:"network_id"`
	HostID    string    `json:"host_id"`
	LocalIP   string    `json:"local_ip"`
	VTEPName  string    `json:"vtep_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// VMNetworkAttachment represents a VM's attachment to a VXLAN network
type VMNetworkAttachment struct {
	ID         string    `json:"id"`
	VMID       string    `json:"vm_id"`
	NetworkID  string    `json:"network_id"`
	IPAddress  string    `json:"ip_address,omitempty"`
	MACAddress string    `json:"mac_address,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
