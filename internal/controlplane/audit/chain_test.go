package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

// mockRepo is a mock implementation of store.Repo for testing
type mockRepo struct {
	events       []store.AuditEvent
	writeErr     error
	listErr      error
	getLastErr   error
	updateErr    error
	lastWrite    *store.AuditEvent
	validityUpdates map[int64]bool
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		events:          make([]store.AuditEvent, 0),
		validityUpdates: make(map[int64]bool),
	}
}

func (m *mockRepo) GetLastAuditEvent(ctx context.Context) (*store.AuditEvent, error) {
	if m.getLastErr != nil {
		return nil, m.getLastErr
	}
	if len(m.events) == 0 {
		return nil, nil
	}
	last := m.events[len(m.events)-1]
	return &last, nil
}

func (m *mockRepo) WriteAuditEvent(ctx context.Context, event *store.AuditEvent) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	if event.ID == 0 {
		event.ID = int64(len(m.events) + 1)
	}
	m.events = append(m.events, *event)
	m.lastWrite = event
	return nil
}

func (m *mockRepo) UpdateAuditEventValidity(ctx context.Context, id int64, valid bool) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.validityUpdates[id] = valid
	for i := range m.events {
		if m.events[i].ID == id {
			m.events[i].ChainValid = valid
			break
		}
	}
	return nil
}

func (m *mockRepo) ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]store.AuditEvent, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	
	var filtered []store.AuditEvent
	for _, e := range m.events {
		if tenantID == "" || e.TenantID == tenantID {
			filtered = append(filtered, e)
		}
	}
	
	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

// Required interface methods - not used in tests
func (m *mockRepo) CreateTenant(ctx context.Context, t store.Tenant) (store.Tenant, error) { return t, nil }
func (m *mockRepo) CreateAPIKey(ctx context.Context, key store.APIKey) (store.APIKey, error) { return key, nil }
func (m *mockRepo) ValidateAPIKey(ctx context.Context, keyHash string) (store.APIKeyValidation, error) { return store.APIKeyValidation{}, nil }
func (m *mockRepo) CreateSite(ctx context.Context, site store.Site) (store.Site, error) { return site, nil }
func (m *mockRepo) ListSites(ctx context.Context, tenantID string) ([]store.Site, error) { return nil, nil }
func (m *mockRepo) IssueEnrollmentToken(ctx context.Context, token store.EnrollmentToken) (store.EnrollmentToken, error) { return token, nil }
func (m *mockRepo) ConsumeEnrollmentToken(ctx context.Context, tokenHash string, now time.Time) (store.TokenConsumeResult, error) { return store.TokenConsumeResult{}, nil }
func (m *mockRepo) CreateAgentFromEnrollment(ctx context.Context, tokenID string, agent store.Agent, hostname string) (store.Agent, error) { return agent, nil }
func (m *mockRepo) GetAgentByID(ctx context.Context, agentID string) (store.Agent, error) { return store.Agent{}, nil }
func (m *mockRepo) IngestHeartbeat(ctx context.Context, hb store.Heartbeat) error { return nil }
func (m *mockRepo) ApplyPlan(ctx context.Context, input store.ApplyPlanInput) (store.ApplyPlanResult, error) { return store.ApplyPlanResult{}, nil }
func (m *mockRepo) LeasePendingPlans(ctx context.Context, agentID string, limit int, leaseTTL time.Duration) ([]store.LeasedPlan, error) { return nil, nil }
func (m *mockRepo) ReportPlanResult(ctx context.Context, agentID string, report store.PlanResultReport) error { return nil }
func (m *mockRepo) IngestLogs(ctx context.Context, req store.LogIngest) (accepted int64, dropped int64, err error) { return 0, 0, nil }
func (m *mockRepo) SweepOfflineAgents(ctx context.Context, staleBefore time.Time) (int64, error) { return 0, nil }
func (m *mockRepo) ListHosts(ctx context.Context, tenantID, siteID string) ([]store.Host, error) { return nil, nil }
func (m *mockRepo) ListVMs(ctx context.Context, tenantID, siteID string) ([]store.MicroVM, error) { return nil, nil }
func (m *mockRepo) ListExecutionLogs(ctx context.Context, tenantID, executionID string, limit int) ([]store.ExecutionLog, error) { return nil, nil }
func (m *mockRepo) WriteAudit(ctx context.Context, tenantID, siteID, actorType, actorID, action, resourceType, resourceID, requestID, sourceIP string, metadata []byte) error { return nil }
func (m *mockRepo) SiteBelongsToTenant(ctx context.Context, siteID, tenantID string) (bool, error) { return true, nil }
func (m *mockRepo) ExecutionBelongsToTenant(ctx context.Context, executionID, tenantID string) (bool, error) { return true, nil }
func (m *mockRepo) ListEnrollmentTokens(ctx context.Context, tenantID string) ([]store.EnrollmentTokenWithStatus, error) { return nil, nil }
func (m *mockRepo) ListExecutions(ctx context.Context, tenantID, siteID string, statuses []string, limit int) ([]store.ExecutionWithTimestamps, error) { return nil, nil }
func (m *mockRepo) ListAPIKeys(ctx context.Context, tenantID string) ([]store.APIKey, error) { return nil, nil }
func (m *mockRepo) DeleteAPIKey(ctx context.Context, tenantID, keyID string) error { return nil }
func (m *mockRepo) UnenrollAgent(ctx context.Context, agentID string) error { return nil }
func (m *mockRepo) UpdateAgentCertificate(ctx context.Context, agentID, certSerial, refreshTokenHash string) error { return nil }
func (m *mockRepo) IsCertificateRevoked(ctx context.Context, serial string) (bool, error) { return false, nil }
func (m *mockRepo) RevokeCertificate(ctx context.Context, serial string, reason int, agentID string) error { return nil }
func (m *mockRepo) ListRevokedCertificates(ctx context.Context) ([]store.CRLEntry, error) { return nil, nil }
func (m *mockRepo) ListCertificateHistory(ctx context.Context, agentID string, limit int) ([]store.CertificateHistory, error) { return nil, nil }
func (m *mockRepo) RecordCertificateIssuance(ctx context.Context, history store.CertificateHistory) error { return nil }
func (m *mockRepo) Close() error { return nil }
func (m *mockRepo) GetTenantLimits(ctx context.Context, tenantID string) (*store.QuotaLimits, error) { return &store.QuotaLimits{}, nil }
func (m *mockRepo) SetTenantLimits(ctx context.Context, tenantID string, limits store.QuotaLimits) error { return nil }
func (m *mockRepo) GetTenantUsage(ctx context.Context, tenantID string) (*store.TenantUsage, error) { return &store.TenantUsage{}, nil }

func TestNewChainManager(t *testing.T) {
	repo := newMockRepo()
	cm := NewChainManager(repo)
	
	if cm == nil {
		t.Fatal("NewChainManager returned nil")
	}
	// Verify the chain manager was created with the repo
	if cm == nil {
		t.Error("ChainManager not initialized")
	}
}

func TestCreateAuditEvent_FirstEvent(t *testing.T) {
	repo := newMockRepo()
	cm := NewChainManager(repo)
	ctx := context.Background()
	
	input := store.AuditEventInput{
		TenantID:     "tenant-1",
		SiteID:       "site-1",
		ActorType:    "USER",
		ActorID:      "user-1",
		Action:       "test.create",
		ResourceType: "test",
		ResourceID:   "resource-1",
		RequestID:    "req-1",
		SourceIP:     "192.168.1.1",
		Metadata:     []byte(`{"key": "value"}`),
	}
	
	event, err := cm.CreateAuditEvent(ctx, input)
	if err != nil {
		t.Fatalf("CreateAuditEvent failed: %v", err)
	}
	
	// Check event properties
	if event.ID != 1 {
		t.Errorf("Expected ID=1, got %d", event.ID)
	}
	if event.TenantID != input.TenantID {
		t.Error("TenantID mismatch")
	}
	if event.PrevHash != GenesisHash {
		t.Errorf("First event should have GenesisHash as PrevHash, got %s", event.PrevHash)
	}
	if event.EntryHash == "" {
		t.Error("EntryHash should not be empty")
	}
	if !event.ChainValid {
		t.Error("ChainValid should be true for new events")
	}
	
	// Verify it was stored
	if len(repo.events) != 1 {
		t.Errorf("Expected 1 event in repo, got %d", len(repo.events))
	}
}

func TestCreateAuditEvent_ChainContinuity(t *testing.T) {
	repo := newMockRepo()
	cm := NewChainManager(repo)
	ctx := context.Background()
	
	// Create first event
	input1 := store.AuditEventInput{
		TenantID:     "tenant-1",
		ActorType:    "USER",
		ActorID:      "user-1",
		Action:       "action-1",
		ResourceType: "resource",
		ResourceID:   "res-1",
	}
	
	event1, err := cm.CreateAuditEvent(ctx, input1)
	if err != nil {
		t.Fatalf("First CreateAuditEvent failed: %v", err)
	}
	
	// Create second event
	input2 := store.AuditEventInput{
		TenantID:     "tenant-1",
		ActorType:    "USER",
		ActorID:      "user-2",
		Action:       "action-2",
		ResourceType: "resource",
		ResourceID:   "res-2",
	}
	
	event2, err := cm.CreateAuditEvent(ctx, input2)
	if err != nil {
		t.Fatalf("Second CreateAuditEvent failed: %v", err)
	}
	
	// Check chain continuity
	if event2.PrevHash != event1.EntryHash {
		t.Errorf("Second event PrevHash should match first event EntryHash.\nPrevHash: %s\nEntryHash: %s", event2.PrevHash, event1.EntryHash)
	}
	
	// Verify different hashes
	if event1.EntryHash == event2.EntryHash {
		t.Error("Different events should have different EntryHashes")
	}
}

func TestCreateAuditEvent_DifferentActorTypes(t *testing.T) {
	repo := newMockRepo()
	cm := NewChainManager(repo)
	ctx := context.Background()
	
	tests := []struct {
		name      string
		actorType string
		actorID   string
	}{
		{"USER actor", "USER", "user-1"},
		{"AGENT actor", "AGENT", "agent-1"},
		{"SYSTEM actor", "SYSTEM", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := store.AuditEventInput{
				TenantID:     "tenant-1",
				ActorType:    tt.actorType,
				ActorID:      tt.actorID,
				Action:       "test.action",
				ResourceType: "test",
				ResourceID:   "res-1",
			}
			
			event, err := cm.CreateAuditEvent(ctx, input)
			if err != nil {
				t.Fatalf("CreateAuditEvent failed: %v", err)
			}
			
			switch tt.actorType {
			case "USER":
				if event.ActorUserID == nil || *event.ActorUserID != tt.actorID {
					t.Error("ActorUserID not set correctly for USER")
				}
			case "AGENT":
				if event.ActorAgentID == nil || *event.ActorAgentID != tt.actorID {
					t.Error("ActorAgentID not set correctly for AGENT")
				}
			case "SYSTEM":
				if event.ActorUserID != nil || event.ActorAgentID != nil {
					t.Error("Actor IDs should be nil for SYSTEM")
				}
			}
		})
	}
}

func TestCreateAuditEvent_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.writeErr = errors.New("database error")
	cm := NewChainManager(repo)
	ctx := context.Background()
	
	input := store.AuditEventInput{
		TenantID:     "tenant-1",
		ActorType:    "USER",
		Action:       "test.action",
		ResourceType: "test",
		ResourceID:   "res-1",
	}
	
	_, err := cm.CreateAuditEvent(ctx, input)
	if err == nil {
		t.Error("Expected error when repo fails, got nil")
	}
}

func TestCalculateHash(t *testing.T) {
	repo := newMockRepo()
	cm := NewChainManager(repo)
	
	event := &store.AuditEvent{
		TenantID:     "tenant-1",
		ActorType:    "USER",
		Action:       "test.action",
		ResourceType: "test",
		ResourceID:   "res-1",
		OccurredAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		PrevHash:     GenesisHash,
		ChainValid:   true,
	}
	
	hash1 := cm.calculateHash(event)
	hash2 := cm.calculateHash(event)
	
	// Hash should be deterministic
	if hash1 != hash2 {
		t.Error("calculateHash should be deterministic")
	}
	
	// Hash should not be empty
	if hash1 == "" {
		t.Error("Hash should not be empty")
	}
	
	// Hash should be 64 characters (SHA256 hex)
	if len(hash1) != 64 {
		t.Errorf("Hash should be 64 characters, got %d", len(hash1))
	}
	
	// Changing a field should change the hash
	event.Action = "different.action"
	hash3 := cm.calculateHash(event)
	if hash1 == hash3 {
		t.Error("Changing event data should change the hash")
	}
}

func TestVerifyChain_EmptyChain(t *testing.T) {
	repo := newMockRepo()
	cm := NewChainManager(repo)
	ctx := context.Background()
	
	result, err := cm.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}
	
	if !result.Valid {
		t.Error("Empty chain should be considered valid")
	}
	if result.Total != 0 {
		t.Errorf("Expected Total=0, got %d", result.Total)
	}
}

func TestVerifyChain_ValidChain(t *testing.T) {
	repo := newMockRepo()
	cm := NewChainManager(repo)
	ctx := context.Background()
	
	// Create a chain of 3 valid events
	for i := 0; i < 3; i++ {
		input := store.AuditEventInput{
			TenantID:     "tenant-1",
			ActorType:    "USER",
			Action:       "action-" + string(rune('0'+i)),
			ResourceType: "test",
			ResourceID:   "res-" + string(rune('0'+i)),
		}
		_, err := cm.CreateAuditEvent(ctx, input)
		if err != nil {
			t.Fatalf("CreateAuditEvent failed: %v", err)
		}
	}
	
	result, err := cm.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}
	
	if !result.Valid {
		t.Error("Valid chain should pass verification")
	}
	if result.Total != 3 {
		t.Errorf("Expected Total=3, got %d", result.Total)
	}
	if result.Invalid != 0 {
		t.Errorf("Expected Invalid=0, got %d", result.Invalid)
	}
	if result.FirstValid == 0 {
		t.Error("FirstValid should be set")
	}
}

func TestVerifyChain_InvalidFirstEvent(t *testing.T) {
	repo := newMockRepo()
	cm := NewChainManager(repo)
	ctx := context.Background()
	
	// Create an event with wrong prev_hash (simulating tampered data)
	event := &store.AuditEvent{
		ID:           1,
		TenantID:     "tenant-1",
		ActorType:    "USER",
		Action:       "test.action",
		ResourceType: "test",
		ResourceID:   "res-1",
		OccurredAt:   time.Now().UTC(),
		PrevHash:     "wrong-hash", // Should be GenesisHash
		EntryHash:    "",           // Will be calculated
		ChainValid:   true,
	}
	event.EntryHash = cm.calculateHash(event)
	repo.events = append(repo.events, *event)
	
	result, err := cm.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}
	
	if result.Valid {
		t.Error("Chain with invalid first event should be invalid")
	}
	if result.Invalid != 1 {
		t.Errorf("Expected Invalid=1, got %d", result.Invalid)
	}
}

func TestVerifyEvent(t *testing.T) {
	repo := newMockRepo()
	cm := NewChainManager(repo)
	ctx := context.Background()
	
	// Create a valid event
	input := store.AuditEventInput{
		TenantID:     "tenant-1",
		ActorType:    "USER",
		Action:       "test.action",
		ResourceType: "test",
		ResourceID:   "res-1",
	}
	
	event, err := cm.CreateAuditEvent(ctx, input)
	if err != nil {
		t.Fatalf("CreateAuditEvent failed: %v", err)
	}
	
	// Verify the event
	valid, err := cm.VerifyEvent(ctx, event.ID)
	if err != nil {
		t.Fatalf("VerifyEvent failed: %v", err)
	}
	if !valid {
		t.Error("Valid event should pass verification")
	}
	
	// Verify non-existent event
	_, err = cm.VerifyEvent(ctx, 999)
	if err == nil {
		t.Error("Expected error for non-existent event")
	}
}

func TestGetChainInfo(t *testing.T) {
	repo := newMockRepo()
	cm := NewChainManager(repo)
	ctx := context.Background()
	
	// Empty chain
	info, err := cm.GetChainInfo(ctx)
	if err != nil {
		t.Fatalf("GetChainInfo failed: %v", err)
	}
	
	if info["total_events"] != 0 {
		t.Error("Empty chain should have 0 events")
	}
	if info["last_entry_hash"] != GenesisHash {
		t.Error("Empty chain should have GenesisHash as last entry")
	}
	
	// Add an event
	input := store.AuditEventInput{
		TenantID:     "tenant-1",
		ActorType:    "USER",
		Action:       "test.action",
		ResourceType: "test",
		ResourceID:   "res-1",
	}
	
	event, err := cm.CreateAuditEvent(ctx, input)
	if err != nil {
		t.Fatalf("CreateAuditEvent failed: %v", err)
	}
	
	info, err = cm.GetChainInfo(ctx)
	if err != nil {
		t.Fatalf("GetChainInfo failed: %v", err)
	}
	
	if info["total_events"] != 1 {
		t.Errorf("Expected 1 event, got %d", info["total_events"])
	}
	if info["last_event_id"] != event.ID {
		t.Error("Last event ID mismatch")
	}
	if info["last_entry_hash"] != event.EntryHash {
		t.Error("Last entry hash mismatch")
	}
	if info["genesis_hash"] != GenesisHash {
		t.Error("Genesis hash should be constant")
	}
}

func TestGenesisHash(t *testing.T) {
	// Verify GenesisHash constant
	expected := "0000000000000000000000000000000000000000000000000000000000000000"
	if GenesisHash != expected {
		t.Errorf("GenesisHash = %s, want %s", GenesisHash, expected)
	}
	
	if len(GenesisHash) != 64 {
		t.Error("GenesisHash should be 64 characters (SHA256 hex)")
	}
}

func TestChainVerificationResult(t *testing.T) {
	result := ChainVerificationResult{
		Valid:      true,
		Total:      10,
		Invalid:    2,
		FirstValid: 3,
	}
	
	if !result.Valid {
		t.Error("Valid should be true")
	}
	if result.Total != 10 {
		t.Error("Total mismatch")
	}
	if result.Invalid != 2 {
		t.Error("Invalid mismatch")
	}
	if result.FirstValid != 3 {
		t.Error("FirstValid mismatch")
	}
}
