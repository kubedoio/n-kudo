package store

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryRepo struct {
	mu sync.Mutex

	tenants map[string]Tenant
	sites   map[string]Site
	hosts   map[string]Host
	agents  map[string]Agent

	apiKeysByHash map[string]APIKey
	tokensByHash  map[string]EnrollmentToken
	tokenUsed     map[string]bool

	plans             map[string]Plan
	planByIdempotency map[string]string
	executions        map[string]Execution
	executionLogs     map[string][]ExecutionLog
	microVMs          map[string]MicroVM
	audits            []AuditRecord
}

type AuditRecord struct {
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

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		tenants:           map[string]Tenant{},
		sites:             map[string]Site{},
		hosts:             map[string]Host{},
		agents:            map[string]Agent{},
		apiKeysByHash:     map[string]APIKey{},
		tokensByHash:      map[string]EnrollmentToken{},
		tokenUsed:         map[string]bool{},
		plans:             map[string]Plan{},
		planByIdempotency: map[string]string{},
		executions:        map[string]Execution{},
		executionLogs:     map[string][]ExecutionLog{},
		microVMs:          map[string]MicroVM{},
		audits:            []AuditRecord{},
	}
}

func (m *MemoryRepo) CreateTenant(_ context.Context, t Tenant) (Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.tenants {
		if existing.Slug == t.Slug {
			return Tenant{}, ErrConflict
		}
	}
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	m.tenants[t.ID] = t
	return t, nil
}

func (m *MemoryRepo) CreateAPIKey(_ context.Context, key APIKey) (APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tenants[key.TenantID]; !ok {
		return APIKey{}, ErrNotFound
	}
	if _, exists := m.apiKeysByHash[key.KeyHash]; exists {
		return APIKey{}, ErrConflict
	}
	m.apiKeysByHash[key.KeyHash] = key
	return key, nil
}

func (m *MemoryRepo) ValidateAPIKey(_ context.Context, keyHash string) (APIKeyValidation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.apiKeysByHash[keyHash]
	if !ok {
		return APIKeyValidation{}, ErrUnauthorized
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now().UTC()) {
		return APIKeyValidation{}, ErrUnauthorized
	}
	return APIKeyValidation{TenantID: key.TenantID}, nil
}

func (m *MemoryRepo) CreateSite(_ context.Context, site Site) (Site, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tenants[site.TenantID]; !ok {
		return Site{}, ErrUnauthorized
	}
	for _, s := range m.sites {
		if s.TenantID == site.TenantID && s.Name == site.Name {
			return Site{}, ErrConflict
		}
	}
	site.ConnectivityState = "OFFLINE"
	site.CreatedAt = time.Now().UTC()
	m.sites[site.ID] = site
	return site, nil
}

func (m *MemoryRepo) ListSites(_ context.Context, tenantID string) ([]Site, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Site, 0)
	for _, s := range m.sites {
		if s.TenantID == tenantID {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *MemoryRepo) IssueEnrollmentToken(_ context.Context, token EnrollmentToken) (EnrollmentToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sites[token.SiteID]; !ok {
		return EnrollmentToken{}, ErrUnauthorized
	}
	m.tokensByHash[token.TokenHash] = token
	m.tokenUsed[token.ID] = false
	return token, nil
}

func (m *MemoryRepo) ConsumeEnrollmentToken(_ context.Context, tokenHash string, now time.Time) (TokenConsumeResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	token, ok := m.tokensByHash[tokenHash]
	if !ok || token.ExpiresAt.Before(now) {
		return TokenConsumeResult{}, ErrTokenInvalid
	}
	if m.tokenUsed[token.ID] {
		return TokenConsumeResult{}, ErrTokenInvalid
	}
	m.tokenUsed[token.ID] = true
	return TokenConsumeResult{TokenID: token.ID, TenantID: token.TenantID, SiteID: token.SiteID}, nil
}

func (m *MemoryRepo) CreateAgentFromEnrollment(_ context.Context, _ string, agent Agent, hostname string) (Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var hostID string
	for _, h := range m.hosts {
		if h.TenantID == agent.TenantID && h.SiteID == agent.SiteID && h.Hostname == hostname {
			hostID = h.ID
			break
		}
	}
	if hostID == "" {
		hostID = agent.HostID
		if hostID == "" {
			hostID = uuid.NewString()
		}
		m.hosts[hostID] = Host{ID: hostID, TenantID: agent.TenantID, SiteID: agent.SiteID, Hostname: hostname}
	}
	agent.HostID = hostID
	m.agents[agent.ID] = agent
	site := m.sites[agent.SiteID]
	now := time.Now().UTC()
	site.ConnectivityState = "ONLINE"
	site.LastHeartbeatAt = &now
	m.sites[site.ID] = site
	return agent, nil
}

func (m *MemoryRepo) GetAgentByID(_ context.Context, agentID string) (Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.agents[agentID]
	if !ok {
		return Agent{}, ErrNotFound
	}
	return a, nil
}

func (m *MemoryRepo) IngestHeartbeat(_ context.Context, hb Heartbeat) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[hb.AgentID]
	if !ok {
		return ErrNotFound
	}
	now := time.Now().UTC()
	host := m.hosts[agent.HostID]
	host.Hostname = hb.Hostname
	host.CPUCoresTotal = hb.CPUCoresTotal
	host.MemoryBytesTotal = hb.MemoryBytesTotal
	host.StorageBytesTotal = hb.StorageBytesTotal
	host.KVMAvailable = hb.KVMAvailable
	host.CloudHypervisorAvailable = hb.CloudHypervisorAvailable
	host.LastFactsAt = &now
	m.hosts[host.ID] = host

	site := m.sites[agent.SiteID]
	site.ConnectivityState = "ONLINE"
	site.LastHeartbeatAt = &now
	m.sites[site.ID] = site

	for _, vm := range hb.MicroVMs {
		if vm.ID == "" {
			continue
		}
		cur := m.microVMs[vm.ID]
		cur.ID = vm.ID
		cur.TenantID = agent.TenantID
		cur.SiteID = agent.SiteID
		cur.HostID = agent.HostID
		cur.Name = vm.Name
		cur.State = strings.ToUpper(vm.State)
		cur.VCPUCount = vm.VCPUCount
		cur.MemoryMiB = vm.MemoryMiB
		t := vm.UpdatedAt
		if t.IsZero() {
			t = now
		}
		cur.LastTransitionAt = &t
		cur.UpdatedAt = t
		m.microVMs[vm.ID] = cur
	}

	planIDs := map[string]struct{}{}
	for _, upd := range hb.ExecutionUpdates {
		e, ok := m.executions[upd.ExecutionID]
		if !ok {
			continue
		}
		e.State = strings.ToUpper(upd.State)
		e.ErrorCode = upd.ErrorCode
		e.ErrorMessage = upd.ErrorMessage
		e.UpdatedAt = now
		if e.State == "IN_PROGRESS" && e.StartedAt == nil {
			e.StartedAt = &now
		}
		if e.State == "SUCCEEDED" || e.State == "FAILED" {
			e.CompletedAt = &now
		}
		m.executions[e.ID] = e
		planIDs[e.PlanID] = struct{}{}
	}
	for planID := range planIDs {
		plan := m.plans[planID]
		failed := false
		active := false
		inProgress := false
		for _, e := range m.executions {
			if e.PlanID != planID {
				continue
			}
			if e.State == "FAILED" {
				failed = true
			}
			if e.State == "PENDING" || e.State == "IN_PROGRESS" {
				active = true
			}
			if e.State == "IN_PROGRESS" {
				inProgress = true
			}
		}
		switch {
		case failed:
			plan.Status = "FAILED"
		case !active:
			plan.Status = "SUCCEEDED"
		case inProgress:
			plan.Status = "IN_PROGRESS"
		default:
			plan.Status = "PENDING"
		}
		m.plans[plan.ID] = plan
	}
	return nil
}

func (m *MemoryRepo) ApplyPlan(_ context.Context, input ApplyPlanInput) (ApplyPlanResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sites[input.SiteID]
	if !ok || s.TenantID != input.TenantID {
		return ApplyPlanResult{}, ErrUnauthorized
	}
	key := input.TenantID + ":" + input.IdempotencyKey
	if planID, ok := m.planByIdempotency[key]; ok {
		plan := m.plans[planID]
		execs := make([]Execution, 0)
		for _, e := range m.executions {
			if e.PlanID == planID {
				execs = append(execs, e)
			}
		}
		sort.Slice(execs, func(i, j int) bool { return execs[i].UpdatedAt.Before(execs[j].UpdatedAt) })
		return ApplyPlanResult{Plan: plan, Executions: execs, Deduplicated: true}, nil
	}

	planVersion := int64(1)
	for _, p := range m.plans {
		if p.SiteID == input.SiteID && p.PlanVersion >= planVersion {
			planVersion = p.PlanVersion + 1
		}
	}
	opsJSON, _ := json.Marshal(input.Actions)
	plan := Plan{
		ID:             uuid.NewString(),
		TenantID:       input.TenantID,
		SiteID:         input.SiteID,
		IdempotencyKey: input.IdempotencyKey,
		PlanVersion:    planVersion,
		Status:         "PENDING",
		OperationsJSON: opsJSON,
		CreatedAt:      time.Now().UTC(),
	}
	m.plans[plan.ID] = plan
	m.planByIdempotency[key] = plan.ID
	execs := make([]Execution, 0, len(input.Actions))
	for _, action := range input.Actions {
		opID := action.OperationID
		if opID == "" {
			opID = uuid.NewString()
		}
		vmID := action.VMID
		if strings.ToUpper(action.Operation) == "CREATE" && vmID == "" {
			vmID = uuid.NewString()
		}
		if vmID != "" {
			vm := m.microVMs[vmID]
			vm.ID = vmID
			vm.TenantID = input.TenantID
			vm.SiteID = input.SiteID
			if strings.TrimSpace(action.Name) != "" {
				vm.Name = action.Name
			}
			vm.State = "CREATING"
			if action.VCPUCount > 0 {
				vm.VCPUCount = action.VCPUCount
			}
			if action.MemoryMiB > 0 {
				vm.MemoryMiB = action.MemoryMiB
			}
			vm.UpdatedAt = time.Now().UTC()
			m.microVMs[vmID] = vm
		}
		e := Execution{
			ID:            uuid.NewString(),
			TenantID:      input.TenantID,
			SiteID:        input.SiteID,
			PlanID:        plan.ID,
			VMID:          vmID,
			OperationID:   opID,
			OperationType: strings.ToUpper(action.Operation),
			State:         "PENDING",
			UpdatedAt:     time.Now().UTC(),
		}
		m.executions[e.ID] = e
		execs = append(execs, e)
	}
	return ApplyPlanResult{Plan: plan, Executions: execs}, nil
}

func (m *MemoryRepo) IngestLogs(_ context.Context, req LogIngest) (accepted int64, dropped int64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[req.AgentID]
	if !ok {
		return 0, 0, ErrNotFound
	}
	for _, entry := range req.Entries {
		exec, ok := m.executions[entry.ExecutionID]
		if !ok || exec.TenantID != agent.TenantID {
			dropped++
			continue
		}
		dup := false
		for _, existing := range m.executionLogs[entry.ExecutionID] {
			if existing.Sequence == entry.Sequence {
				dup = true
				break
			}
		}
		if dup {
			dropped++
			continue
		}
		m.executionLogs[entry.ExecutionID] = append(m.executionLogs[entry.ExecutionID], ExecutionLog{
			ID:          time.Now().UnixNano(),
			TenantID:    agent.TenantID,
			ExecutionID: entry.ExecutionID,
			Sequence:    entry.Sequence,
			Severity:    strings.ToUpper(entry.Severity),
			Message:     entry.Message,
			EmittedAt:   entry.EmittedAt,
			IngestedAt:  time.Now().UTC(),
		})
		accepted++
	}
	return accepted, dropped, nil
}

func (m *MemoryRepo) ListHosts(_ context.Context, tenantID, siteID string) ([]Host, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Host, 0)
	for _, h := range m.hosts {
		if h.TenantID == tenantID && h.SiteID == siteID {
			host := h
			for _, a := range m.agents {
				if a.HostID == h.ID {
					host.AgentState = "ONLINE"
					break
				}
			}
			out = append(out, host)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Hostname < out[j].Hostname })
	return out, nil
}

func (m *MemoryRepo) ListVMs(_ context.Context, tenantID, siteID string) ([]MicroVM, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MicroVM, 0)
	for _, vm := range m.microVMs {
		if vm.TenantID == tenantID && vm.SiteID == siteID {
			out = append(out, vm)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (m *MemoryRepo) ListExecutionLogs(_ context.Context, tenantID, executionID string, limit int) ([]ExecutionLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	exec, ok := m.executions[executionID]
	if !ok || exec.TenantID != tenantID {
		return nil, ErrNotFound
	}
	entries := append([]ExecutionLog(nil), m.executionLogs[executionID]...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Sequence < entries[j].Sequence })
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

func (m *MemoryRepo) WriteAudit(_ context.Context, tenantID, siteID, actorType, actorID, action, resourceType, resourceID, requestID, sourceIP string, metadata []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audits = append(m.audits, AuditRecord{
		TenantID:     tenantID,
		SiteID:       siteID,
		ActorType:    actorType,
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		RequestID:    requestID,
		SourceIP:     sourceIP,
		Metadata:     append([]byte(nil), metadata...),
	})
	return nil
}

func (m *MemoryRepo) SiteBelongsToTenant(_ context.Context, siteID, tenantID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sites[siteID]
	return ok && s.TenantID == tenantID, nil
}

func (m *MemoryRepo) ExecutionBelongsToTenant(_ context.Context, executionID, tenantID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.executions[executionID]
	return ok && e.TenantID == tenantID, nil
}
