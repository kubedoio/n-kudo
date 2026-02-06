package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) CreateTenant(ctx context.Context, t Tenant) (Tenant, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO tenants (id, slug, name, primary_region, data_retention_days)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, slug, name, primary_region, data_retention_days, created_at, updated_at`,
		t.ID, t.Slug, t.Name, t.PrimaryRegion, t.RetentionDays,
	)
	var out Tenant
	if err := row.Scan(&out.ID, &out.Slug, &out.Name, &out.PrimaryRegion, &out.RetentionDays, &out.CreatedAt, &out.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return Tenant{}, ErrConflict
		}
		return Tenant{}, err
	}
	return out, nil
}

func (r *PostgresRepo) CreateAPIKey(ctx context.Context, key APIKey) (APIKey, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO api_keys (id, tenant_id, name, key_hash, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, tenant_id, name, key_hash, expires_at`, key.ID, key.TenantID, key.Name, key.KeyHash, key.ExpiresAt)
	var out APIKey
	if err := row.Scan(&out.ID, &out.TenantID, &out.Name, &out.KeyHash, &out.ExpiresAt); err != nil {
		if isUniqueViolation(err) {
			return APIKey{}, ErrConflict
		}
		return APIKey{}, err
	}
	return out, nil
}

func (r *PostgresRepo) ValidateAPIKey(ctx context.Context, keyHash string) (APIKeyValidation, error) {
	var tenantID string
	err := r.db.QueryRowContext(ctx, `
UPDATE api_keys
SET last_used_at = now()
WHERE key_hash = $1
  AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > now())
RETURNING tenant_id`, keyHash).Scan(&tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return APIKeyValidation{}, ErrUnauthorized
	}
	if err != nil {
		return APIKeyValidation{}, err
	}
	return APIKeyValidation{TenantID: tenantID}, nil
}

func (r *PostgresRepo) CreateSite(ctx context.Context, site Site) (Site, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO sites (id, tenant_id, name, external_key, location_country_code)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, tenant_id, name, COALESCE(external_key,''), COALESCE(location_country_code,''), connectivity_state, last_heartbeat_at, created_at`,
		site.ID, site.TenantID, site.Name, nullable(site.ExternalKey), nullable(site.LocationCountry),
	)
	var out Site
	if err := row.Scan(
		&out.ID,
		&out.TenantID,
		&out.Name,
		&out.ExternalKey,
		&out.LocationCountry,
		&out.ConnectivityState,
		&out.LastHeartbeatAt,
		&out.CreatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			return Site{}, ErrConflict
		}
		return Site{}, err
	}
	return out, nil
}

func (r *PostgresRepo) ListSites(ctx context.Context, tenantID string) ([]Site, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, tenant_id, name, COALESCE(external_key,''), COALESCE(location_country_code,''), connectivity_state, last_heartbeat_at, created_at
FROM sites
WHERE tenant_id = $1
ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Site, 0)
	for rows.Next() {
		var s Site
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.ExternalKey, &s.LocationCountry, &s.ConnectivityState, &s.LastHeartbeatAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PostgresRepo) IssueEnrollmentToken(ctx context.Context, token EnrollmentToken) (EnrollmentToken, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO enrollment_tokens (id, tenant_id, site_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, tenant_id, site_id, token_hash, expires_at`,
		token.ID, token.TenantID, token.SiteID, token.TokenHash, token.ExpiresAt)
	var out EnrollmentToken
	if err := row.Scan(&out.ID, &out.TenantID, &out.SiteID, &out.TokenHash, &out.ExpiresAt); err != nil {
		return EnrollmentToken{}, err
	}
	return out, nil
}

func (r *PostgresRepo) ConsumeEnrollmentToken(ctx context.Context, tokenHash string, now time.Time) (TokenConsumeResult, error) {
	row := r.db.QueryRowContext(ctx, `
UPDATE enrollment_tokens
SET used_at = $2
WHERE token_hash = $1
  AND used_at IS NULL
  AND expires_at > $2
RETURNING id, tenant_id, site_id`, tokenHash, now)
	var out TokenConsumeResult
	if err := row.Scan(&out.TokenID, &out.TenantID, &out.SiteID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TokenConsumeResult{}, ErrTokenInvalid
		}
		return TokenConsumeResult{}, err
	}
	return out, nil
}

func (r *PostgresRepo) CreateAgentFromEnrollment(ctx context.Context, tokenID string, agent Agent, hostname string) (Agent, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Agent{}, err
	}
	defer tx.Rollback()

	var hostID string
	err = tx.QueryRowContext(ctx, `
INSERT INTO hosts (id, tenant_id, site_id, hostname, updated_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (tenant_id, site_id, hostname)
DO UPDATE SET updated_at = now()
RETURNING id`, agent.HostID, agent.TenantID, agent.SiteID, hostname).Scan(&hostID)
	if err != nil {
		return Agent{}, err
	}
	agent.HostID = hostID

	err = tx.QueryRowContext(ctx, `
INSERT INTO agents (
  id, tenant_id, site_id, host_id, enrollment_token_hash, refresh_token_hash,
  cert_serial, agent_version, os, arch, kernel_version, state, enrolled_at, last_heartbeat_at
)
VALUES ($1, $2, $3, $4, (SELECT token_hash FROM enrollment_tokens WHERE id=$5), $6, $7, $8, $9, $10, $11, 'ONLINE', now(), now())
RETURNING id, tenant_id, site_id, host_id, cert_serial, refresh_token_hash, agent_version, os, arch, COALESCE(kernel_version, ''), state::text, last_heartbeat_at`,
		agent.ID,
		agent.TenantID,
		agent.SiteID,
		agent.HostID,
		tokenID,
		agent.RefreshTokenHash,
		agent.CertSerial,
		agent.AgentVersion,
		agent.OS,
		agent.Arch,
		nullable(agent.KernelVersion),
	).Scan(
		&agent.ID,
		&agent.TenantID,
		&agent.SiteID,
		&agent.HostID,
		&agent.CertSerial,
		&agent.RefreshTokenHash,
		&agent.AgentVersion,
		&agent.OS,
		&agent.Arch,
		&agent.KernelVersion,
		&agent.State,
		&agent.LastHeartbeatAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Agent{}, ErrConflict
		}
		return Agent{}, err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE sites SET connectivity_state='ONLINE', last_heartbeat_at=now(), updated_at=now()
WHERE id=$1 AND tenant_id=$2`, agent.SiteID, agent.TenantID); err != nil {
		return Agent{}, err
	}

	if err := tx.Commit(); err != nil {
		return Agent{}, err
	}
	return agent, nil
}

func (r *PostgresRepo) GetAgentByID(ctx context.Context, agentID string) (Agent, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, tenant_id, site_id, host_id, cert_serial, refresh_token_hash, agent_version, os, arch, COALESCE(kernel_version, ''), state::text, last_heartbeat_at
FROM agents
WHERE id = $1`, agentID)
	var a Agent
	if err := row.Scan(&a.ID, &a.TenantID, &a.SiteID, &a.HostID, &a.CertSerial, &a.RefreshTokenHash, &a.AgentVersion, &a.OS, &a.Arch, &a.KernelVersion, &a.State, &a.LastHeartbeatAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Agent{}, ErrNotFound
		}
		return Agent{}, err
	}
	return a, nil
}

func (r *PostgresRepo) IngestHeartbeat(ctx context.Context, hb Heartbeat) error {
	agent, err := r.GetAgentByID(ctx, hb.AgentID)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
UPDATE agents
SET heartbeat_seq = GREATEST(heartbeat_seq, $1),
    agent_version = $2,
    os = $3,
    arch = $4,
    kernel_version = $5,
    state = 'ONLINE',
    last_heartbeat_at = $6,
    updated_at = $6
WHERE id = $7 AND tenant_id = $8`, hb.HeartbeatSeq, hb.AgentVersion, hb.OS, hb.Arch, nullable(hb.KernelVersion), now, agent.ID, agent.TenantID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE hosts
SET hostname = $1,
    cpu_cores_total = $2,
    memory_bytes_total = $3,
    storage_bytes_total = $4,
    kvm_available = $5,
    cloud_hypervisor_available = $6,
    last_facts_at = $7,
    updated_at = $7
WHERE id = $8 AND tenant_id = $9`, hb.Hostname, hb.CPUCoresTotal, hb.MemoryBytesTotal, hb.StorageBytesTotal, hb.KVMAvailable, hb.CloudHypervisorAvailable, now, agent.HostID, agent.TenantID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE sites
SET connectivity_state = 'ONLINE',
    last_heartbeat_at = $1,
    updated_at = $1
WHERE id = $2 AND tenant_id = $3`, now, agent.SiteID, agent.TenantID); err != nil {
		return err
	}

	for _, vm := range hb.MicroVMs {
		vmID := vm.ID
		if vmID == "" {
			continue
		}
		updateAt := vm.UpdatedAt
		if updateAt.IsZero() {
			updateAt = now
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO microvms (
  id, tenant_id, site_id, host_id, name, state, vcpu_count, memory_mib, last_transition_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
ON CONFLICT (id)
DO UPDATE SET
  state = EXCLUDED.state,
  host_id = EXCLUDED.host_id,
  vcpu_count = EXCLUDED.vcpu_count,
  memory_mib = EXCLUDED.memory_mib,
  last_transition_at = EXCLUDED.last_transition_at,
  updated_at = EXCLUDED.updated_at`,
			vmID, agent.TenantID, agent.SiteID, agent.HostID, vm.Name, normalizeMicroVMState(vm.State), vm.VCPUCount, vm.MemoryMiB, updateAt); err != nil {
			return err
		}
	}

	planIDs := make(map[string]struct{})
	for _, upd := range hb.ExecutionUpdates {
		state := normalizeExecutionState(upd.State)
		updateAt := upd.UpdatedAt
		if updateAt.IsZero() {
			updateAt = now
		}
		var planID string
		err := tx.QueryRowContext(ctx, `
UPDATE executions
SET state = $1,
    error_code = $2,
    error_message = $3,
    started_at = CASE WHEN $1 = 'IN_PROGRESS' AND started_at IS NULL THEN $4 ELSE started_at END,
    completed_at = CASE WHEN $1 IN ('SUCCEEDED','FAILED') THEN $4 ELSE completed_at END,
    updated_at = $4
WHERE id = $5
  AND tenant_id = $6
RETURNING plan_id`, state, nullable(upd.ErrorCode), nullable(upd.ErrorMessage), updateAt, upd.ExecutionID, agent.TenantID).Scan(&planID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return err
		}
		planIDs[planID] = struct{}{}
	}

	for planID := range planIDs {
		if err := r.rollupPlanStatusTx(ctx, tx, planID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresRepo) ApplyPlan(ctx context.Context, input ApplyPlanInput) (ApplyPlanResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ApplyPlanResult{}, err
	}
	defer tx.Rollback()

	var siteExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sites WHERE id=$1 AND tenant_id=$2)`, input.SiteID, input.TenantID).Scan(&siteExists); err != nil {
		return ApplyPlanResult{}, err
	}
	if !siteExists {
		return ApplyPlanResult{}, ErrUnauthorized
	}

	if existing, ok, err := r.getPlanByIdempotencyTx(ctx, tx, input.TenantID, input.IdempotencyKey); err != nil {
		return ApplyPlanResult{}, err
	} else if ok {
		if err := tx.Commit(); err != nil {
			return ApplyPlanResult{}, err
		}
		existing.Deduplicated = true
		return existing, nil
	}

	if _, err := tx.ExecContext(ctx, `SELECT id FROM sites WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, input.SiteID, input.TenantID); err != nil {
		return ApplyPlanResult{}, err
	}

	var planVersion int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(plan_version), 0) + 1 FROM plans WHERE site_id=$1`, input.SiteID).Scan(&planVersion); err != nil {
		return ApplyPlanResult{}, err
	}

	operationsJSON, err := json.Marshal(input.Actions)
	if err != nil {
		return ApplyPlanResult{}, err
	}

	plan := Plan{
		ID:             newUUID(),
		TenantID:       input.TenantID,
		SiteID:         input.SiteID,
		IdempotencyKey: input.IdempotencyKey,
		PlanVersion:    planVersion,
		Status:         "PENDING",
		OperationsJSON: operationsJSON,
	}

	if err := tx.QueryRowContext(ctx, `
INSERT INTO plans (id, tenant_id, site_id, idempotency_key, client_request_id, plan_version, status, operations_json)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING created_at`, plan.ID, plan.TenantID, plan.SiteID, plan.IdempotencyKey, nullable(input.ClientRequestID), plan.PlanVersion, plan.Status, plan.OperationsJSON).Scan(&plan.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			if existing, ok, err2 := r.getPlanByIdempotencyTx(ctx, tx, input.TenantID, input.IdempotencyKey); err2 == nil && ok {
				if err := tx.Commit(); err != nil {
					return ApplyPlanResult{}, err
				}
				existing.Deduplicated = true
				return existing, nil
			}
			return ApplyPlanResult{}, ErrConflict
		}
		return ApplyPlanResult{}, err
	}

	execs := make([]Execution, 0, len(input.Actions))
	for _, action := range input.Actions {
		opType := normalizeOperation(action.Operation)
		actionID := action.OperationID
		if actionID == "" {
			actionID = newUUID()
		}
		vmID := action.VMID
		if opType == "CREATE" {
			if vmID == "" {
				vmID = newUUID()
			}
			name := action.Name
			if strings.TrimSpace(name) == "" {
				name = "vm-" + vmID[:8]
			}
			if _, err := tx.ExecContext(ctx, `
INSERT INTO microvms (id, tenant_id, site_id, host_id, name, state, vcpu_count, memory_mib, last_transition_at, updated_at)
VALUES ($1,$2,$3,NULL,$4,'CREATING',$5,$6,now(),now())
ON CONFLICT (id)
DO UPDATE SET
  name = EXCLUDED.name,
  vcpu_count = EXCLUDED.vcpu_count,
  memory_mib = EXCLUDED.memory_mib,
  updated_at = now()`, vmID, input.TenantID, input.SiteID, name, max(action.VCPUCount, 1), max64(action.MemoryMiB, 128)); err != nil {
				return ApplyPlanResult{}, err
			}
		}

		payloadJSON, err := json.Marshal(action)
		if err != nil {
			return ApplyPlanResult{}, err
		}

		if _, err := tx.ExecContext(ctx, `
INSERT INTO plan_actions (id, tenant_id, plan_id, operation_id, operation_type, vm_id, payload_json)
VALUES ($1,$2,$3,$4,$5,$6,$7)`, newUUID(), input.TenantID, plan.ID, actionID, opType, nullable(vmID), payloadJSON); err != nil {
			return ApplyPlanResult{}, err
		}

		exec := Execution{
			ID:            newUUID(),
			TenantID:      input.TenantID,
			SiteID:        input.SiteID,
			PlanID:        plan.ID,
			VMID:          vmID,
			OperationID:   actionID,
			OperationType: opType,
			State:         "PENDING",
		}
		if err := tx.QueryRowContext(ctx, `
INSERT INTO executions (id, tenant_id, site_id, plan_id, vm_id, operation_id, operation_type, state)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING updated_at`, exec.ID, exec.TenantID, exec.SiteID, exec.PlanID, nullable(exec.VMID), exec.OperationID, exec.OperationType, exec.State).Scan(&exec.UpdatedAt); err != nil {
			return ApplyPlanResult{}, err
		}
		execs = append(execs, exec)
	}

	if err := tx.Commit(); err != nil {
		return ApplyPlanResult{}, err
	}
	return ApplyPlanResult{Plan: plan, Executions: execs}, nil
}

func (r *PostgresRepo) LeasePendingPlans(ctx context.Context, agentID string, limit int, leaseTTL time.Duration) ([]LeasedPlan, error) {
	agent, err := r.GetAgentByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 1
	}
	if leaseTTL <= 0 {
		leaseTTL = 30 * time.Second
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	leaseUntil := now.Add(leaseTTL)
	rows, err := tx.QueryContext(ctx, `
WITH candidate AS (
  SELECT id
  FROM plans
  WHERE tenant_id = $2
    AND site_id = $3
    AND status IN ('PENDING','IN_PROGRESS')
    AND (leased_by_agent_id = $1 OR lease_expires_at IS NULL OR lease_expires_at <= $4)
  ORDER BY created_at ASC
  LIMIT $5
  FOR UPDATE SKIP LOCKED
)
UPDATE plans p
SET leased_by_agent_id = $1,
    lease_expires_at = $6,
    status = CASE WHEN p.status = 'PENDING' THEN 'IN_PROGRESS' ELSE p.status END,
    started_at = CASE WHEN p.started_at IS NULL THEN $4 ELSE p.started_at END,
    updated_at = $4
FROM candidate c
WHERE p.id = c.id
RETURNING p.id`,
		agent.ID, agent.TenantID, agent.SiteID, now, limit, leaseUntil)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	planIDs := make([]string, 0)
	for rows.Next() {
		var planID string
		if err := rows.Scan(&planID); err != nil {
			return nil, err
		}
		planIDs = append(planIDs, planID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]LeasedPlan, 0, len(planIDs))
	for _, planID := range planIDs {
		actionRows, err := tx.QueryContext(ctx, `
SELECT pa.id, pa.plan_id, pa.operation_id, pa.operation_type, COALESCE(pa.vm_id::text,''), pa.payload_json
FROM plan_actions pa
JOIN executions e
  ON e.tenant_id = pa.tenant_id
 AND e.plan_id = pa.plan_id
 AND e.operation_id = pa.operation_id
WHERE pa.tenant_id = $1
  AND pa.plan_id = $2
  AND e.state IN ('PENDING','IN_PROGRESS')
ORDER BY pa.created_at ASC`, agent.TenantID, planID)
		if err != nil {
			return nil, err
		}
		actions := make([]PlanAction, 0)
		for actionRows.Next() {
			var action PlanAction
			if err := actionRows.Scan(&action.ID, &action.PlanID, &action.OperationID, &action.OperationType, &action.VMID, &action.PayloadJSON); err != nil {
				actionRows.Close()
				return nil, err
			}
			actions = append(actions, action)
		}
		if err := actionRows.Err(); err != nil {
			actionRows.Close()
			return nil, err
		}
		actionRows.Close()
		if len(actions) == 0 {
			continue
		}
		out = append(out, LeasedPlan{
			PlanID:      planID,
			ExecutionID: planID,
			Actions:     actions,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PostgresRepo) ReportPlanResult(ctx context.Context, agentID string, report PlanResultReport) error {
	agent, err := r.GetAgentByID(ctx, agentID)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	planID := strings.TrimSpace(report.PlanID)
	if planID == "" {
		planID, err = r.resolvePlanIDTx(ctx, tx, agent.TenantID, agent.SiteID, report.ExecutionID)
		if err != nil {
			return err
		}
	}
	if planID == "" {
		return ErrNotFound
	}

	var exists bool
	if err := tx.QueryRowContext(ctx, `
SELECT EXISTS(
  SELECT 1
  FROM plans
  WHERE id = $1
    AND tenant_id = $2
    AND site_id = $3
)`, planID, agent.TenantID, agent.SiteID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	now := time.Now().UTC()
	for _, result := range report.Results {
		actionID := strings.TrimSpace(result.ActionID)
		if actionID == "" {
			continue
		}

		state := "FAILED"
		errorCode := nullable(chooseStringPG(result.ErrorCode, "ACTION_FAILED"))
		errorMessage := nullable(chooseStringPG(result.Message, "action failed"))
		if result.OK {
			state = "SUCCEEDED"
			errorCode = nil
			errorMessage = nil
		}
		completedAt := result.FinishedAt
		if completedAt.IsZero() {
			completedAt = now
		}

		var vmID string
		var operationType string
		err := tx.QueryRowContext(ctx, `
UPDATE executions
SET state = $1,
    error_code = $2,
    error_message = $3,
    host_id = $4,
    agent_id = $5,
    started_at = COALESCE(started_at, $6),
    completed_at = $6,
    updated_at = $6
WHERE tenant_id = $7
  AND site_id = $8
  AND plan_id = $9
  AND operation_id = $10
RETURNING COALESCE(vm_id::text, ''), operation_type`,
			state,
			errorCode,
			errorMessage,
			nullable(agent.HostID),
			nullable(agent.ID),
			completedAt,
			agent.TenantID,
			agent.SiteID,
			planID,
			actionID,
		).Scan(&vmID, &operationType)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return err
		}
		if err := r.applyExecutionVMStateTx(ctx, tx, agent.TenantID, agent.SiteID, nullable(agent.HostID), vmID, operationType, state, completedAt); err != nil {
			return err
		}
	}

	if err := r.rollupPlanStatusTx(ctx, tx, planID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *PostgresRepo) IngestLogs(ctx context.Context, req LogIngest) (accepted int64, dropped int64, err error) {
	agent, err := r.GetAgentByID(ctx, req.AgentID)
	if err != nil {
		return 0, 0, err
	}
	for _, entry := range req.Entries {
		sev := normalizeSeverity(entry.Severity)
		if entry.EmittedAt.IsZero() {
			entry.EmittedAt = time.Now().UTC()
		}
		res, err := r.db.ExecContext(ctx, `
INSERT INTO execution_logs (tenant_id, execution_id, sequence, severity, message, emitted_at)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (tenant_id, execution_id, sequence) DO NOTHING`,
			agent.TenantID, entry.ExecutionID, entry.Sequence, sev, entry.Message, entry.EmittedAt)
		if err != nil {
			dropped++
			continue
		}
		n, err := res.RowsAffected()
		if err != nil {
			dropped++
			continue
		}
		if n == 0 {
			dropped++
			continue
		}
		accepted++
	}
	return accepted, dropped, nil
}

func (r *PostgresRepo) SweepOfflineAgents(ctx context.Context, staleBefore time.Time) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
UPDATE agents
SET state = 'OFFLINE',
    updated_at = now()
WHERE state <> 'OFFLINE'
  AND last_heartbeat_at IS NOT NULL
  AND last_heartbeat_at < $1`, staleBefore)
	if err != nil {
		return 0, err
	}
	updated, _ := res.RowsAffected()

	if _, err := tx.ExecContext(ctx, `
UPDATE sites s
SET connectivity_state = CASE
      WHEN EXISTS (
        SELECT 1
        FROM agents a
        WHERE a.tenant_id = s.tenant_id
          AND a.site_id = s.id
          AND a.state = 'ONLINE'
      ) THEN 'ONLINE'
      ELSE 'OFFLINE'
    END,
    last_heartbeat_at = (
      SELECT MAX(a.last_heartbeat_at)
      FROM agents a
      WHERE a.tenant_id = s.tenant_id
        AND a.site_id = s.id
    ),
    updated_at = now()`); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return updated, nil
}

func (r *PostgresRepo) ListHosts(ctx context.Context, tenantID, siteID string) ([]Host, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT h.id, h.tenant_id, h.site_id, h.hostname, h.cpu_cores_total, h.memory_bytes_total,
       h.storage_bytes_total, h.kvm_available, h.cloud_hypervisor_available, h.last_facts_at,
       COALESCE(a.state::text, 'OFFLINE') as agent_state, a.last_heartbeat_at
FROM hosts h
LEFT JOIN agents a ON a.host_id = h.id AND a.tenant_id = h.tenant_id
WHERE h.tenant_id = $1 AND h.site_id = $2
ORDER BY h.hostname ASC`, tenantID, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Host, 0)
	for rows.Next() {
		var h Host
		if err := rows.Scan(&h.ID, &h.TenantID, &h.SiteID, &h.Hostname, &h.CPUCoresTotal, &h.MemoryBytesTotal, &h.StorageBytesTotal, &h.KVMAvailable, &h.CloudHypervisorAvailable, &h.LastFactsAt, &h.AgentState, &h.AgentLastHeartbeatAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *PostgresRepo) ListVMs(ctx context.Context, tenantID, siteID string) ([]MicroVM, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, tenant_id, site_id, COALESCE(host_id::text,''), name, state::text, vcpu_count, memory_mib, last_transition_at, updated_at
FROM microvms
WHERE tenant_id = $1 AND site_id = $2
ORDER BY updated_at DESC`, tenantID, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MicroVM, 0)
	for rows.Next() {
		var vm MicroVM
		if err := rows.Scan(&vm.ID, &vm.TenantID, &vm.SiteID, &vm.HostID, &vm.Name, &vm.State, &vm.VCPUCount, &vm.MemoryMiB, &vm.LastTransitionAt, &vm.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, vm)
	}
	return out, rows.Err()
}

func (r *PostgresRepo) ListExecutionLogs(ctx context.Context, tenantID, executionID string, limit int) ([]ExecutionLog, error) {
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, tenant_id, execution_id, sequence, severity, message, emitted_at, ingested_at
FROM execution_logs
WHERE tenant_id = $1 AND execution_id = $2
ORDER BY sequence ASC
LIMIT $3`, tenantID, executionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ExecutionLog, 0)
	for rows.Next() {
		var l ExecutionLog
		if err := rows.Scan(&l.ID, &l.TenantID, &l.ExecutionID, &l.Sequence, &l.Severity, &l.Message, &l.EmittedAt, &l.IngestedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *PostgresRepo) WriteAudit(ctx context.Context, tenantID, siteID, actorType, actorID, action, resourceType, resourceID, requestID, sourceIP string, metadata []byte) error {
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}
	var actorUserID any
	var actorAgentID any
	switch actorType {
	case "USER":
		actorUserID = nullable(actorID)
	case "AGENT":
		actorAgentID = nullable(actorID)
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO audit_events (
  tenant_id, site_id, actor_type, actor_user_id, actor_agent_id,
  action, resource_type, resource_id, request_id, source_ip, metadata_json
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb)`,
		tenantID,
		nullable(siteID),
		actorType,
		actorUserID,
		actorAgentID,
		action,
		resourceType,
		resourceID,
		nullable(requestID),
		nullable(sourceIP),
		string(metadata),
	); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepo) SiteBelongsToTenant(ctx context.Context, siteID, tenantID string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sites WHERE id=$1 AND tenant_id=$2)`, siteID, tenantID).Scan(&ok)
	return ok, err
}

func (r *PostgresRepo) ExecutionBelongsToTenant(ctx context.Context, executionID, tenantID string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM executions WHERE id=$1 AND tenant_id=$2)`, executionID, tenantID).Scan(&ok)
	return ok, err
}

func (r *PostgresRepo) getPlanByIdempotencyTx(ctx context.Context, tx *sql.Tx, tenantID, idempotency string) (ApplyPlanResult, bool, error) {
	row := tx.QueryRowContext(ctx, `
SELECT id, tenant_id, site_id, idempotency_key, plan_version, status, operations_json, created_at
FROM plans
WHERE tenant_id = $1 AND idempotency_key = $2`, tenantID, idempotency)
	var plan Plan
	if err := row.Scan(&plan.ID, &plan.TenantID, &plan.SiteID, &plan.IdempotencyKey, &plan.PlanVersion, &plan.Status, &plan.OperationsJSON, &plan.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ApplyPlanResult{}, false, nil
		}
		return ApplyPlanResult{}, false, err
	}

	rows, err := tx.QueryContext(ctx, `
SELECT id, tenant_id, site_id, COALESCE(host_id::text,''), COALESCE(agent_id::text,''), plan_id,
       COALESCE(vm_id::text,''), operation_id, operation_type, state::text,
       COALESCE(error_code,''), COALESCE(error_message,''), updated_at, started_at, completed_at
FROM executions
WHERE plan_id = $1
ORDER BY created_at ASC`, plan.ID)
	if err != nil {
		return ApplyPlanResult{}, false, err
	}
	defer rows.Close()
	execs := make([]Execution, 0)
	for rows.Next() {
		var e Execution
		if err := rows.Scan(&e.ID, &e.TenantID, &e.SiteID, &e.HostID, &e.AgentID, &e.PlanID, &e.VMID, &e.OperationID, &e.OperationType, &e.State, &e.ErrorCode, &e.ErrorMessage, &e.UpdatedAt, &e.StartedAt, &e.CompletedAt); err != nil {
			return ApplyPlanResult{}, false, err
		}
		execs = append(execs, e)
	}
	if err := rows.Err(); err != nil {
		return ApplyPlanResult{}, false, err
	}
	return ApplyPlanResult{Plan: plan, Executions: execs}, true, nil
}

func (r *PostgresRepo) resolvePlanIDTx(ctx context.Context, tx *sql.Tx, tenantID, siteID, executionOrPlanID string) (string, error) {
	executionOrPlanID = strings.TrimSpace(executionOrPlanID)
	if executionOrPlanID == "" {
		return "", nil
	}

	var planID string
	err := tx.QueryRowContext(ctx, `
SELECT id
FROM plans
WHERE id = $1
  AND tenant_id = $2
  AND site_id = $3`, executionOrPlanID, tenantID, siteID).Scan(&planID)
	if err == nil {
		return planID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	err = tx.QueryRowContext(ctx, `
SELECT plan_id
FROM executions
WHERE id = $1
  AND tenant_id = $2
  AND site_id = $3`, executionOrPlanID, tenantID, siteID).Scan(&planID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return planID, nil
}

func (r *PostgresRepo) rollupPlanStatusTx(ctx context.Context, tx *sql.Tx, planID string) error {
	_, err := tx.ExecContext(ctx, `
WITH rollup AS (
  SELECT
    SUM(CASE WHEN state = 'FAILED' THEN 1 ELSE 0 END) AS failed,
    SUM(CASE WHEN state IN ('PENDING','IN_PROGRESS') THEN 1 ELSE 0 END) AS active,
    SUM(CASE WHEN state = 'IN_PROGRESS' THEN 1 ELSE 0 END) AS in_progress,
    SUM(CASE WHEN state = 'SUCCEEDED' THEN 1 ELSE 0 END) AS completed
  FROM executions
  WHERE plan_id = $1
)
UPDATE plans
SET status = CASE
    WHEN (SELECT failed FROM rollup) > 0 THEN 'FAILED'
    WHEN (SELECT active FROM rollup) = 0 THEN 'SUCCEEDED'
    WHEN (SELECT in_progress FROM rollup) > 0
      OR ((SELECT active FROM rollup) > 0 AND (SELECT completed FROM rollup) > 0) THEN 'IN_PROGRESS'
    ELSE 'PENDING'
  END,
  started_at = CASE
    WHEN started_at IS NULL AND (SELECT in_progress FROM rollup) > 0 THEN now()
    ELSE started_at
  END,
  completed_at = CASE
    WHEN (SELECT failed FROM rollup) > 0 OR (SELECT active FROM rollup) = 0 THEN now()
    ELSE completed_at
  END,
  leased_by_agent_id = CASE
    WHEN (SELECT failed FROM rollup) > 0 OR (SELECT active FROM rollup) = 0 THEN NULL
    ELSE leased_by_agent_id
  END,
  lease_expires_at = CASE
    WHEN (SELECT failed FROM rollup) > 0 OR (SELECT active FROM rollup) = 0 THEN NULL
    ELSE lease_expires_at
  END,
  updated_at = now()
WHERE id = $1`, planID)
	return err
}

func (r *PostgresRepo) applyExecutionVMStateTx(ctx context.Context, tx *sql.Tx, tenantID, siteID string, hostID any, vmID, operationType, executionState string, at time.Time) error {
	vmID = strings.TrimSpace(vmID)
	if vmID == "" {
		return nil
	}

	if strings.EqualFold(executionState, "FAILED") {
		_, err := tx.ExecContext(ctx, `
UPDATE microvms
SET state = 'ERROR',
    last_transition_at = $3,
    updated_at = $3
WHERE id = $1
  AND tenant_id = $2`, vmID, tenantID, at)
		return err
	}
	if !strings.EqualFold(executionState, "SUCCEEDED") {
		return nil
	}

	switch strings.ToUpper(strings.TrimSpace(operationType)) {
	case "DELETE":
		_, err := tx.ExecContext(ctx, `DELETE FROM microvms WHERE id = $1 AND tenant_id = $2`, vmID, tenantID)
		return err
	case "CREATE", "START", "STOP":
		nextState := "STOPPED"
		if strings.EqualFold(operationType, "START") {
			nextState = "RUNNING"
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO microvms (id, tenant_id, site_id, host_id, name, state, vcpu_count, memory_mib, last_transition_at, updated_at)
VALUES ($1, $2, $3, $4, $1, $5, 1, 128, $6, $6)
ON CONFLICT (id)
DO UPDATE SET
  host_id = COALESCE(EXCLUDED.host_id, microvms.host_id),
  state = EXCLUDED.state,
  last_transition_at = EXCLUDED.last_transition_at,
  updated_at = EXCLUDED.updated_at`, vmID, tenantID, siteID, hostID, nextState, at)
		return err
	default:
		return nil
	}
}

func nullable(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func chooseStringPG(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}

func normalizeOperation(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "CREATE", "START", "STOP", "DELETE":
		return s
	default:
		return "CREATE"
	}
}

func normalizeExecutionState(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "PENDING", "IN_PROGRESS", "SUCCEEDED", "FAILED":
		return s
	default:
		return "IN_PROGRESS"
	}
}

func normalizeMicroVMState(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "CREATING", "STOPPED", "RUNNING", "DELETING", "ERROR":
		return s
	default:
		return "CREATING"
	}
}

func normalizeSeverity(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "DEBUG", "INFO", "WARN", "ERROR":
		return s
	default:
		return "INFO"
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
