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
	tokenCreated  map[string]time.Time

	plans             map[string]Plan
	planActions       map[string][]PlanAction
	planLeases        map[string]planLease
	planByIdempotency map[string]string
	executions        map[string]Execution
	executionLogs     map[string][]ExecutionLog
	microVMs          map[string]MicroVM
	audits            []AuditRecord
	crlEntries        map[string]*CRLEntry
}

type planLease struct {
	AgentID   string
	ExpiresAt time.Time
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
		tokenCreated:      map[string]time.Time{},
		plans:             map[string]Plan{},
		planActions:       map[string][]PlanAction{},
		planLeases:        map[string]planLease{},
		planByIdempotency: map[string]string{},
		executions:        map[string]Execution{},
		executionLogs:     map[string][]ExecutionLog{},
		microVMs:          map[string]MicroVM{},
		audits:            []AuditRecord{},
		crlEntries:        map[string]*CRLEntry{},
	}
}

// Close is a no-op for MemoryRepo since it doesn't hold external resources.
func (m *MemoryRepo) Close() error {
	return nil
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

func (m *MemoryRepo) ListAPIKeys(_ context.Context, tenantID string) ([]APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]APIKey, 0)
	for _, key := range m.apiKeysByHash {
		if key.TenantID == tenantID {
			out = append(out, key)
		}
	}
	// Sort by CreatedAt descending
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *MemoryRepo) DeleteAPIKey(_ context.Context, tenantID, keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Find the key by ID and tenant
	found := false
	var keyHashToDelete string
	for hash, key := range m.apiKeysByHash {
		if key.ID == keyID && key.TenantID == tenantID {
			found = true
			keyHashToDelete = hash
			break
		}
	}
	if !found {
		return ErrNotFound
	}
	delete(m.apiKeysByHash, keyHashToDelete)
	return nil
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
	m.tokenCreated[token.ID] = time.Now().UTC()
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
	now := time.Now().UTC()
	agent.State = "ONLINE"
	agent.LastHeartbeatAt = &now
	m.agents[agent.ID] = agent
	site := m.sites[agent.SiteID]
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
	agent.State = "ONLINE"
	agent.LastHeartbeatAt = &now
	m.agents[agent.ID] = agent

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
		m.updateVMStateFromExecutionLocked(e, now)
		planIDs[e.PlanID] = struct{}{}
	}
	for planID := range planIDs {
		m.rollupPlanLocked(planID, now)
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
	actions := make([]PlanAction, 0, len(input.Actions))
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

		payload, _ := json.Marshal(action)
		actions = append(actions, PlanAction{
			ID:            uuid.NewString(),
			PlanID:        plan.ID,
			OperationID:   opID,
			OperationType: strings.ToUpper(action.Operation),
			VMID:          vmID,
			PayloadJSON:   payload,
		})
	}
	m.planActions[plan.ID] = actions
	return ApplyPlanResult{Plan: plan, Executions: execs}, nil
}

func (m *MemoryRepo) LeasePendingPlans(_ context.Context, agentID string, limit int, leaseTTL time.Duration) ([]LeasedPlan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[agentID]
	if !ok {
		return nil, ErrNotFound
	}
	if limit <= 0 {
		limit = 1
	}
	if leaseTTL <= 0 {
		leaseTTL = 30 * time.Second
	}

	now := time.Now().UTC()
	candidates := make([]Plan, 0)
	for _, plan := range m.plans {
		if plan.TenantID != agent.TenantID || plan.SiteID != agent.SiteID {
			continue
		}
		if !isRunnablePlanStatus(plan.Status) {
			continue
		}
		lease, leased := m.planLeases[plan.ID]
		if leased && lease.AgentID != agentID && lease.ExpiresAt.After(now) {
			continue
		}
		if !m.planHasPendingExecutionsLocked(plan.ID) {
			continue
		}
		candidates = append(candidates, plan)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].CreatedAt.Before(candidates[j].CreatedAt) })

	out := make([]LeasedPlan, 0, min(limit, len(candidates)))
	for _, plan := range candidates {
		if len(out) >= limit {
			break
		}
		m.planLeases[plan.ID] = planLease{
			AgentID:   agentID,
			ExpiresAt: now.Add(leaseTTL),
		}
		if plan.Status == "PENDING" {
			plan.Status = "IN_PROGRESS"
			m.plans[plan.ID] = plan
		}

		operationIDs := make(map[string]struct{})
		for _, exec := range m.executions {
			if exec.PlanID != plan.ID {
				continue
			}
			if exec.State == "PENDING" || exec.State == "IN_PROGRESS" {
				operationIDs[exec.OperationID] = struct{}{}
			}
		}

		actions := make([]PlanAction, 0)
		for _, action := range m.planActions[plan.ID] {
			if _, ok := operationIDs[action.OperationID]; !ok {
				continue
			}
			copied := action
			copied.PayloadJSON = append([]byte(nil), action.PayloadJSON...)
			actions = append(actions, copied)
		}
		if len(actions) == 0 {
			continue
		}
		out = append(out, LeasedPlan{
			PlanID:      plan.ID,
			ExecutionID: plan.ID,
			Actions:     actions,
		})
	}

	return out, nil
}

func (m *MemoryRepo) ReportPlanResult(_ context.Context, agentID string, report PlanResultReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[agentID]
	if !ok {
		return ErrNotFound
	}

	planID := strings.TrimSpace(report.PlanID)
	if planID == "" {
		planID = m.resolvePlanIDLocked(report.ExecutionID)
	}
	if planID == "" {
		return ErrNotFound
	}
	plan, ok := m.plans[planID]
	if !ok {
		return ErrNotFound
	}
	if plan.TenantID != agent.TenantID || plan.SiteID != agent.SiteID {
		return ErrUnauthorized
	}

	now := time.Now().UTC()
	for _, result := range report.Results {
		actionID := strings.TrimSpace(result.ActionID)
		if actionID == "" {
			continue
		}
		execID := m.executionIDByOperationLocked(planID, actionID)
		if execID == "" {
			continue
		}
		exec := m.executions[execID]
		updatedAt := result.FinishedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}
		exec.AgentID = agent.ID
		exec.HostID = agent.HostID
		exec.UpdatedAt = updatedAt
		if exec.StartedAt == nil {
			started := updatedAt
			exec.StartedAt = &started
		}

		if result.OK {
			exec.State = "SUCCEEDED"
			exec.ErrorCode = ""
			exec.ErrorMessage = ""
		} else {
			exec.State = "FAILED"
			exec.ErrorCode = chooseString(result.ErrorCode, "ACTION_FAILED")
			exec.ErrorMessage = chooseString(result.Message, "action failed")
		}
		completed := updatedAt
		exec.CompletedAt = &completed
		m.executions[execID] = exec
		m.updateVMStateFromExecutionLocked(exec, updatedAt)
	}

	m.rollupPlanLocked(planID, now)
	return nil
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

func (m *MemoryRepo) SweepOfflineAgents(_ context.Context, staleBefore time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var updated int64
	for id, agent := range m.agents {
		if agent.LastHeartbeatAt == nil {
			continue
		}
		if agent.LastHeartbeatAt.After(staleBefore) || agent.State == "OFFLINE" {
			continue
		}
		agent.State = "OFFLINE"
		m.agents[id] = agent
		updated++
	}
	for siteID := range m.sites {
		m.refreshSiteConnectivityLocked(siteID)
	}
	return updated, nil
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
					host.AgentState = normalizeAgentState(a.State)
					host.AgentLastHeartbeatAt = a.LastHeartbeatAt
					break
				}
			}
			if host.AgentState == "" {
				host.AgentState = "OFFLINE"
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

func (m *MemoryRepo) ListExecutions(_ context.Context, tenantID, siteID string, statuses []string, limit int) ([]ExecutionWithTimestamps, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	statusMap := make(map[string]struct{})
	for _, s := range statuses {
		statusMap[strings.ToUpper(strings.TrimSpace(s))] = struct{}{}
	}
	out := make([]ExecutionWithTimestamps, 0)
	for _, e := range m.executions {
		if e.TenantID != tenantID || e.SiteID != siteID {
			continue
		}
		if len(statusMap) > 0 {
			if _, ok := statusMap[e.State]; !ok {
				continue
			}
		}
		execWithTs := ExecutionWithTimestamps{
			ID:            e.ID,
			PlanID:        e.PlanID,
			OperationID:   e.OperationID,
			OperationType: e.OperationType,
			State:         e.State,
			VMID:          e.VMID,
			UpdatedAt:     e.UpdatedAt,
		}
		if e.ErrorCode != "" {
			errCode := e.ErrorCode
			execWithTs.ErrorCode = &errCode
		}
		if e.ErrorMessage != "" {
			errMsg := e.ErrorMessage
			execWithTs.ErrorMessage = &errMsg
		}
		// Get CreatedAt from plan if available
		if plan, ok := m.plans[e.PlanID]; ok {
			execWithTs.CreatedAt = plan.CreatedAt
		}
		out = append(out, execWithTs)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryRepo) ListEnrollmentTokens(_ context.Context, tenantID string) ([]EnrollmentTokenWithStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]EnrollmentTokenWithStatus, 0)
	for _, token := range m.tokensByHash {
		if token.TenantID != tenantID {
			continue
		}
		site, ok := m.sites[token.SiteID]
		if !ok {
			continue
		}

		t := EnrollmentTokenWithStatus{
			ID:        token.ID,
			SiteID:    token.SiteID,
			SiteName:  site.Name,
			CreatedAt: m.tokenCreated[token.ID],
			ExpiresAt: token.ExpiresAt,
			Consumed:  m.tokenUsed[token.ID],
		}

		// Find consumed_at and agent_id by looking up agents
		for _, agent := range m.agents {
			if agent.SiteID == token.SiteID {
				t.ConsumedByAgentID = &agent.ID
				if agent.LastHeartbeatAt != nil {
					t.ConsumedAt = agent.LastHeartbeatAt
				}
				break
			}
		}

		out = append(out, t)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *MemoryRepo) planHasPendingExecutionsLocked(planID string) bool {
	for _, exec := range m.executions {
		if exec.PlanID != planID {
			continue
		}
		if exec.State == "PENDING" || exec.State == "IN_PROGRESS" {
			return true
		}
	}
	return false
}

func (m *MemoryRepo) executionIDByOperationLocked(planID, operationID string) string {
	for id, exec := range m.executions {
		if exec.PlanID == planID && exec.OperationID == operationID {
			return id
		}
	}
	return ""
}

func (m *MemoryRepo) resolvePlanIDLocked(executionOrPlanID string) string {
	executionOrPlanID = strings.TrimSpace(executionOrPlanID)
	if executionOrPlanID == "" {
		return ""
	}
	if _, ok := m.plans[executionOrPlanID]; ok {
		return executionOrPlanID
	}
	if exec, ok := m.executions[executionOrPlanID]; ok {
		return exec.PlanID
	}
	return ""
}

func (m *MemoryRepo) rollupPlanLocked(planID string, now time.Time) {
	plan, ok := m.plans[planID]
	if !ok {
		return
	}
	failed := false
	active := false
	inProgress := false
	completed := false
	for _, exec := range m.executions {
		if exec.PlanID != planID {
			continue
		}
		switch exec.State {
		case "FAILED":
			failed = true
		case "PENDING":
			active = true
		case "IN_PROGRESS":
			active = true
			inProgress = true
		case "SUCCEEDED":
			completed = true
		}
	}
	switch {
	case failed:
		plan.Status = "FAILED"
	case !active:
		plan.Status = "SUCCEEDED"
	case inProgress || completed:
		plan.Status = "IN_PROGRESS"
	default:
		plan.Status = "PENDING"
	}
	m.plans[plan.ID] = plan
	if !isRunnablePlanStatus(plan.Status) {
		delete(m.planLeases, planID)
		return
	}
	if lease, ok := m.planLeases[planID]; ok && lease.ExpiresAt.Before(now) {
		delete(m.planLeases, planID)
	}
}

func (m *MemoryRepo) updateVMStateFromExecutionLocked(exec Execution, at time.Time) {
	if strings.TrimSpace(exec.VMID) == "" {
		return
	}
	if !at.IsZero() {
		t := at
		vm, ok := m.microVMs[exec.VMID]
		if !ok {
			vm = MicroVM{
				ID:       exec.VMID,
				TenantID: exec.TenantID,
				SiteID:   exec.SiteID,
				HostID:   exec.HostID,
				Name:     exec.VMID,
			}
		}
		vm.LastTransitionAt = &t
		vm.UpdatedAt = t

		switch exec.State {
		case "FAILED":
			vm.State = "ERROR"
			m.microVMs[exec.VMID] = vm
			return
		case "SUCCEEDED":
			switch strings.ToUpper(exec.OperationType) {
			case "CREATE":
				vm.State = "STOPPED"
			case "START":
				vm.State = "RUNNING"
			case "STOP":
				vm.State = "STOPPED"
			case "DELETE":
				delete(m.microVMs, exec.VMID)
				return
			}
		}
		m.microVMs[exec.VMID] = vm
	}
}

func (m *MemoryRepo) refreshSiteConnectivityLocked(siteID string) {
	site, ok := m.sites[siteID]
	if !ok {
		return
	}
	state := "OFFLINE"
	var last *time.Time
	for _, agent := range m.agents {
		if agent.SiteID != siteID {
			continue
		}
		if agent.LastHeartbeatAt != nil {
			if last == nil || agent.LastHeartbeatAt.After(*last) {
				t := *agent.LastHeartbeatAt
				last = &t
			}
		}
		if normalizeAgentState(agent.State) == "ONLINE" {
			state = "ONLINE"
		}
	}
	site.ConnectivityState = state
	site.LastHeartbeatAt = last
	m.sites[siteID] = site
}

func isRunnablePlanStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PENDING", "IN_PROGRESS":
		return true
	default:
		return false
	}
}

func normalizeAgentState(state string) string {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "ONLINE":
		return "ONLINE"
	case "DEGRADED":
		return "DEGRADED"
	default:
		return "OFFLINE"
	}
}

func chooseString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func (m *MemoryRepo) UnenrollAgent(_ context.Context, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[agentID]
	if !ok {
		return ErrNotFound
	}
	agent.State = "UNENROLLED"
	agent.CertSerial = ""
	agent.RefreshTokenHash = ""
	m.agents[agentID] = agent
	return nil
}

func (m *MemoryRepo) UpdateAgentCertificate(_ context.Context, agentID, certSerial, refreshTokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[agentID]
	if !ok {
		return ErrNotFound
	}
	agent.CertSerial = certSerial
	agent.RefreshTokenHash = refreshTokenHash
	m.agents[agentID] = agent
	return nil
}

// In-memory storage for certificate history
type certHistoryEntry struct {
	CertificateHistory
}

var certHistoryStore = struct {
	sync.Mutex
	entries map[string][]certHistoryEntry
}{entries: make(map[string][]certHistoryEntry)}

func (m *MemoryRepo) ListCertificateHistory(_ context.Context, agentID string, limit int) ([]CertificateHistory, error) {
	certHistoryStore.Lock()
	defer certHistoryStore.Unlock()

	if limit <= 0 || limit > 1000 {
		limit = 50
	}

	entries, ok := certHistoryStore.entries[agentID]
	if !ok {
		return []CertificateHistory{}, nil
	}

	out := make([]CertificateHistory, 0, min(limit, len(entries)))
	for i := len(entries) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, entries[i].CertificateHistory)
	}
	return out, nil
}

func (m *MemoryRepo) RecordCertificateIssuance(_ context.Context, history CertificateHistory) error {
	certHistoryStore.Lock()
	defer certHistoryStore.Unlock()

	if history.ID == "" {
		history.ID = uuid.NewString()
	}

	certHistoryStore.entries[history.AgentID] = append(certHistoryStore.entries[history.AgentID], certHistoryEntry{history})
	return nil
}

// crlEntryKey is used as a key for the revoked certificates map
type crlEntryKey struct {
	serial string
}

// RevokeCertificate adds a certificate to the revocation list
func (m *MemoryRepo) RevokeCertificate(_ context.Context, serial string, reason int, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.crlEntries == nil {
		m.crlEntries = make(map[string]*CRLEntry)
	}
	m.crlEntries[serial] = &CRLEntry{
		SerialNumber: serial,
		RevokedAt:    time.Now().UTC(),
		Reason:       reason,
		AgentID:      agentID,
	}
	return nil
}

// IsCertificateRevoked checks if a certificate serial is revoked
func (m *MemoryRepo) IsCertificateRevoked(_ context.Context, serial string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.crlEntries == nil {
		return false, nil
	}
	_, exists := m.crlEntries[serial]
	return exists, nil
}

// ListRevokedCertificates returns all revoked certificates
func (m *MemoryRepo) ListRevokedCertificates(_ context.Context) ([]CRLEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]CRLEntry, 0, len(m.crlEntries))
	for _, entry := range m.crlEntries {
		out = append(out, *entry)
	}
	return out, nil
}

// GetTenantLimits returns the quota limits for a tenant
func (m *MemoryRepo) GetTenantLimits(_ context.Context, tenantID string) (*QuotaLimits, error) {
	// MemoryRepo returns default limits
	defaultLimits := QuotaLimits{
		MaxSites:           10,
		MaxAgentsPerSite:   100,
		MaxVMsPerAgent:     50,
		MaxConcurrentPlans: 100,
		MaxAPIKeys:         20,
	}
	return &defaultLimits, nil
}

// SetTenantLimits sets the quota limits for a tenant
func (m *MemoryRepo) SetTenantLimits(_ context.Context, tenantID string, limits QuotaLimits) error {
	// MemoryRepo stores limits in memory - for now just accept the call
	return nil
}

// GetTenantUsage returns current resource usage counts for a tenant
func (m *MemoryRepo) GetTenantUsage(_ context.Context, tenantID string) (*TenantUsage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	usage := &TenantUsage{}

	// Count sites
	for _, site := range m.sites {
		if site.TenantID == tenantID {
			usage.Sites++
		}
	}

	// Count agents
	for _, agent := range m.agents {
		if agent.TenantID == tenantID {
			usage.Agents++
		}
	}

	// Count VMs
	for _, vm := range m.microVMs {
		if vm.TenantID == tenantID {
			usage.VMs++
		}
	}

	// Count active plans
	for _, plan := range m.plans {
		if plan.TenantID == tenantID && (plan.Status == "PENDING" || plan.Status == "IN_PROGRESS") {
			usage.ActivePlans++
		}
	}

	// Count API keys
	for _, key := range m.apiKeysByHash {
		if key.TenantID == tenantID {
			usage.APIKeys++
		}
	}

	return usage, nil
}

// GetLastAuditEvent returns the last audit event for chain integrity
func (m *MemoryRepo) GetLastAuditEvent(_ context.Context) (*AuditEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// MemoryRepo uses simple AuditRecord, not full AuditEvent chain
	return nil, nil
}

// WriteAuditEvent writes a full audit event with chain integrity
func (m *MemoryRepo) WriteAuditEvent(_ context.Context, event *AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// MemoryRepo uses simple AuditRecord, convert and store
	actorID := ""
	if event.ActorUserID != nil {
		actorID = *event.ActorUserID
	} else if event.ActorAgentID != nil {
		actorID = *event.ActorAgentID
	}
	m.audits = append(m.audits, AuditRecord{
		TenantID:     event.TenantID,
		SiteID:       event.SiteID,
		ActorType:    event.ActorType,
		ActorID:      actorID,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   event.ResourceID,
		RequestID:    event.RequestID,
		SourceIP:     event.SourceIP,
		Metadata:     event.MetadataJSON,
	})
	return nil
}

// UpdateAuditEventValidity updates the chain validity of an audit event
func (m *MemoryRepo) UpdateAuditEventValidity(_ context.Context, id int64, valid bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// MemoryRepo doesn't track chain validity per event
	return nil
}

// ListAuditEvents returns audit events for a tenant
func (m *MemoryRepo) ListAuditEvents(_ context.Context, tenantID string, limit int) ([]AuditEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	events := make([]AuditEvent, 0)
	for i := len(m.audits) - 1; i >= 0 && len(events) < limit; i-- {
		audit := m.audits[i]
		if tenantID == "" || audit.TenantID == tenantID {
			events = append(events, AuditEvent{
				TenantID:     audit.TenantID,
				SiteID:       audit.SiteID,
				ActorType:    audit.ActorType,
				Action:       audit.Action,
				ResourceType: audit.ResourceType,
				ResourceID:   audit.ResourceID,
				RequestID:    audit.RequestID,
				SourceIP:     audit.SourceIP,
				MetadataJSON: audit.Metadata,
			})
		}
	}
	return events, nil
}
