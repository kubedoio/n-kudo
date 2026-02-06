package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemoryRepoLeasePendingPlansRespectsLeaseTTL(t *testing.T) {
	repo, tenantID, siteID := newMemoryRepoWithTenantSite(t)
	agent1 := newAgent(t, repo, tenantID, siteID, "host-a")
	agent2 := newAgent(t, repo, tenantID, siteID, "host-b")

	_, err := repo.ApplyPlan(context.Background(), ApplyPlanInput{
		TenantID:       tenantID,
		SiteID:         siteID,
		IdempotencyKey: "lease-test",
		Actions: []ApplyPlanAction{
			{OperationID: "create-a", Operation: "CREATE", VMID: "vm-1", Name: "vm-1", VCPUCount: 1, MemoryMiB: 128},
		},
	})
	if err != nil {
		t.Fatalf("apply plan: %v", err)
	}

	leased1, err := repo.LeasePendingPlans(context.Background(), agent1.ID, 1, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("lease plans (agent1): %v", err)
	}
	if len(leased1) != 1 {
		t.Fatalf("expected first lease to return 1 plan, got %d", len(leased1))
	}

	leased2, err := repo.LeasePendingPlans(context.Background(), agent2.ID, 1, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("lease plans (agent2): %v", err)
	}
	if len(leased2) != 0 {
		t.Fatalf("expected second agent to get 0 plans while lease is active, got %d", len(leased2))
	}

	time.Sleep(70 * time.Millisecond)
	leased3, err := repo.LeasePendingPlans(context.Background(), agent2.ID, 1, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("lease plans (agent2 retry): %v", err)
	}
	if len(leased3) != 1 {
		t.Fatalf("expected second agent to lease after ttl expiry, got %d", len(leased3))
	}
}

func TestMemoryRepoReportPlanResultRollsPlanState(t *testing.T) {
	repo, tenantID, siteID := newMemoryRepoWithTenantSite(t)
	agent := newAgent(t, repo, tenantID, siteID, "host-a")

	applied, err := repo.ApplyPlan(context.Background(), ApplyPlanInput{
		TenantID:       tenantID,
		SiteID:         siteID,
		IdempotencyKey: "result-test",
		Actions: []ApplyPlanAction{
			{OperationID: "create-a", Operation: "CREATE", VMID: "vm-1", Name: "vm-1", VCPUCount: 1, MemoryMiB: 128},
			{OperationID: "start-a", Operation: "START", VMID: "vm-1"},
		},
	})
	if err != nil {
		t.Fatalf("apply plan: %v", err)
	}

	if err := repo.ReportPlanResult(context.Background(), agent.ID, PlanResultReport{
		PlanID:      applied.Plan.ID,
		ExecutionID: applied.Plan.ID,
		Results: []PlanActionResultItem{
			{ActionID: "create-a", OK: true, FinishedAt: time.Now().UTC()},
		},
	}); err != nil {
		t.Fatalf("report result (partial): %v", err)
	}
	if got := repo.plans[applied.Plan.ID].Status; got != "IN_PROGRESS" {
		t.Fatalf("expected plan status IN_PROGRESS after partial result, got %s", got)
	}

	if err := repo.ReportPlanResult(context.Background(), agent.ID, PlanResultReport{
		PlanID:      applied.Plan.ID,
		ExecutionID: applied.Plan.ID,
		Results: []PlanActionResultItem{
			{ActionID: "start-a", OK: false, ErrorCode: "START_FAILED", Message: "failed", FinishedAt: time.Now().UTC()},
		},
	}); err != nil {
		t.Fatalf("report result (final): %v", err)
	}
	if got := repo.plans[applied.Plan.ID].Status; got != "FAILED" {
		t.Fatalf("expected plan status FAILED after failed action, got %s", got)
	}
}

func TestMemoryRepoSweepOfflineAgentsUpdatesHostState(t *testing.T) {
	repo, tenantID, siteID := newMemoryRepoWithTenantSite(t)
	agent := newAgent(t, repo, tenantID, siteID, "host-a")

	old := time.Now().UTC().Add(-2 * time.Minute)
	stale := repo.agents[agent.ID]
	stale.State = "ONLINE"
	stale.LastHeartbeatAt = &old
	repo.agents[agent.ID] = stale

	updated, err := repo.SweepOfflineAgents(context.Background(), time.Now().UTC().Add(-60*time.Second))
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected 1 updated agent, got %d", updated)
	}

	hosts, err := repo.ListHosts(context.Background(), tenantID, siteID)
	if err != nil {
		t.Fatalf("list hosts: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].AgentState != "OFFLINE" {
		t.Fatalf("expected host agent_state OFFLINE, got %s", hosts[0].AgentState)
	}
}

func newMemoryRepoWithTenantSite(t *testing.T) (*MemoryRepo, string, string) {
	t.Helper()
	repo := NewMemoryRepo()
	tenantID := uuid.NewString()
	siteID := uuid.NewString()
	if _, err := repo.CreateTenant(context.Background(), Tenant{
		ID:            tenantID,
		Slug:          "tenant-" + tenantID[:8],
		Name:          "Tenant",
		PrimaryRegion: "us-east-1",
		RetentionDays: 30,
	}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if _, err := repo.CreateSite(context.Background(), Site{
		ID:       siteID,
		TenantID: tenantID,
		Name:     "site-a",
	}); err != nil {
		t.Fatalf("create site: %v", err)
	}
	return repo, tenantID, siteID
}

func newAgent(t *testing.T, repo *MemoryRepo, tenantID, siteID, hostname string) Agent {
	t.Helper()
	agent, err := repo.CreateAgentFromEnrollment(context.Background(), "", Agent{
		ID:               uuid.NewString(),
		TenantID:         tenantID,
		SiteID:           siteID,
		HostID:           uuid.NewString(),
		CertSerial:       "serial-" + uuid.NewString(),
		RefreshTokenHash: "rt-" + uuid.NewString(),
		AgentVersion:     "test",
		OS:               "linux",
		Arch:             "amd64",
	}, hostname)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	return agent
}
