// Package audit provides audit log chain integrity verification using cryptographic hashing.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

// GenesisHash is the hash used for the first entry in the chain (or when no previous entry exists).
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// ChainVerificationResult represents the result of a chain verification operation.
type ChainVerificationResult struct {
	Valid      bool  `json:"valid"`       // True if entire chain is valid
	Total      int   `json:"total"`       // Total number of events checked
	Invalid    int   `json:"invalid"`     // Number of invalid events found
	FirstValid int64 `json:"first_valid"` // ID of the first valid event (or 0 if none)
}

// ChainManager handles audit event chain integrity operations.
type ChainManager struct {
	repo store.Repo
}

// NewChainManager creates a new ChainManager with the given repository.
func NewChainManager(repo store.Repo) *ChainManager {
	return &ChainManager{repo: repo}
}

// CreateAuditEvent creates a new audit event with proper chain integrity.
// It fetches the last audit event, sets the PrevHash, calculates the EntryHash,
// and stores the event.
func (cm *ChainManager) CreateAuditEvent(ctx context.Context, input store.AuditEventInput) (*store.AuditEvent, error) {
	// Get the last audit event to determine prev_hash
	lastEvent, err := cm.repo.GetLastAuditEvent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get last audit event: %w", err)
	}

	// Determine prev_hash
	prevHash := GenesisHash
	if lastEvent != nil {
		prevHash = lastEvent.EntryHash
	}

	// Prepare metadata
	metadata := input.Metadata
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}

	// Build the actor IDs based on actor type
	var actorUserID, actorAgentID *string
	switch input.ActorType {
	case "USER":
		if input.ActorID != "" {
			actorUserID = &input.ActorID
		}
	case "AGENT":
		if input.ActorID != "" {
			actorAgentID = &input.ActorID
		}
	}

	// Create the event (without hash for now - will be calculated)
	event := &store.AuditEvent{
		TenantID:     input.TenantID,
		SiteID:       input.SiteID,
		ActorType:    input.ActorType,
		ActorUserID:  actorUserID,
		ActorAgentID: actorAgentID,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		RequestID:    input.RequestID,
		SourceIP:     input.SourceIP,
		MetadataJSON: metadata,
		OccurredAt:   time.Now().UTC(),
		PrevHash:     prevHash,
		ChainValid:   true,
	}

	// Calculate the entry hash
	event.EntryHash = cm.calculateHash(event)

	// Store the event
	if err := cm.repo.WriteAuditEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to write audit event: %w", err)
	}

	return event, nil
}

// calculateHash computes the SHA256 hash of the event data.
// The hash is calculated over the JSON representation of the event
// WITHOUT the EntryHash and ID fields (since EntryHash is what we're calculating,
// and ID is a database-assigned identifier not part of the event data).
func (cm *ChainManager) calculateHash(event *store.AuditEvent) string {
	// Create a copy of the event without hash fields for hashing
	hashData := struct {
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
		PrevHash     string    `json:"prev_hash"`
		ChainValid   bool      `json:"chain_valid"`
	}{
		TenantID:     event.TenantID,
		SiteID:       event.SiteID,
		ActorType:    event.ActorType,
		ActorUserID:  event.ActorUserID,
		ActorAgentID: event.ActorAgentID,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   event.ResourceID,
		RequestID:    event.RequestID,
		SourceIP:     event.SourceIP,
		MetadataJSON: event.MetadataJSON,
		OccurredAt:   event.OccurredAt,
		PrevHash:     event.PrevHash,
		ChainValid:   event.ChainValid,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(hashData)
	if err != nil {
		// In the unlikely event of a marshal error, return a zero hash
		return GenesisHash
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// VerifyChain verifies the entire audit log chain integrity.
// It checks that:
// 1. Each event's EntryHash matches the calculated hash
// 2. Each event's PrevHash matches the previous event's EntryHash
// 3. The chain starts with a valid GenesisHash reference
func (cm *ChainManager) VerifyChain(ctx context.Context) (*ChainVerificationResult, error) {
	// Get all audit events ordered by ID
	events, err := cm.repo.ListAuditEvents(ctx, "", 0) // Empty tenantID = all events, 0 = no limit
	if err != nil {
		return nil, fmt.Errorf("failed to list audit events: %w", err)
	}

	result := &ChainVerificationResult{
		Valid:   true,
		Total:   len(events),
		Invalid: 0,
	}

	if len(events) == 0 {
		return result, nil
	}

	var prevHash string
	var firstValidFound bool

	for i, event := range events {
		isValid := true

		// Check if this is the first event
		if i == 0 {
			// First event should reference GenesisHash
			if event.PrevHash != GenesisHash {
				isValid = false
			}
		} else {
			// Subsequent events should reference the previous event's hash
			if event.PrevHash != prevHash {
				isValid = false
			}
		}

		// Verify the entry hash
		calculatedHash := cm.calculateHash(&event)
		if event.EntryHash != calculatedHash {
			isValid = false
		}

		// Update validity in database if changed
		if event.ChainValid != isValid {
			if err := cm.repo.UpdateAuditEventValidity(ctx, event.ID, isValid); err != nil {
				// Log error but continue verification
				// In production, this should be logged properly
				_ = err
			}
		}

		if !isValid {
			result.Valid = false
			result.Invalid++
		} else if !firstValidFound {
			result.FirstValid = event.ID
			firstValidFound = true
		}

		prevHash = event.EntryHash
	}

	return result, nil
}

// VerifyEvent verifies a single audit event's integrity.
// It checks that the EntryHash matches the calculated hash and
// that the PrevHash correctly references the previous event.
func (cm *ChainManager) VerifyEvent(ctx context.Context, id int64) (bool, error) {
	// Get all events up to and including this one
	events, err := cm.repo.ListAuditEvents(ctx, "", 0)
	if err != nil {
		return false, fmt.Errorf("failed to list audit events: %w", err)
	}

	// Find the target event and verify it
	var targetEvent *store.AuditEvent
	var targetIndex int
	for i, event := range events {
		if event.ID == id {
			targetEvent = &events[i]
			targetIndex = i
			break
		}
	}

	if targetEvent == nil {
		return false, fmt.Errorf("audit event with id %d not found", id)
	}

	// Verify the hash
	calculatedHash := cm.calculateHash(targetEvent)
	if targetEvent.EntryHash != calculatedHash {
		if targetEvent.ChainValid {
			_ = cm.repo.UpdateAuditEventValidity(ctx, id, false)
		}
		return false, nil
	}

	// Verify the chain link
	var expectedPrevHash string
	if targetIndex == 0 {
		expectedPrevHash = GenesisHash
	} else {
		expectedPrevHash = events[targetIndex-1].EntryHash
	}

	if targetEvent.PrevHash != expectedPrevHash {
		if targetEvent.ChainValid {
			_ = cm.repo.UpdateAuditEventValidity(ctx, id, false)
		}
		return false, nil
	}

	// Mark as valid if it wasn't already
	if !targetEvent.ChainValid {
		_ = cm.repo.UpdateAuditEventValidity(ctx, id, true)
	}

	return true, nil
}

// GetChainInfo returns information about the current state of the audit chain.
func (cm *ChainManager) GetChainInfo(ctx context.Context) (map[string]interface{}, error) {
	lastEvent, err := cm.repo.GetLastAuditEvent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get last audit event: %w", err)
	}

	// Get total count
	events, err := cm.repo.ListAuditEvents(ctx, "", 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit events: %w", err)
	}

	info := map[string]interface{}{
		"total_events": len(events),
		"genesis_hash": GenesisHash,
	}

	if lastEvent != nil {
		info["last_event_id"] = lastEvent.ID
		info["last_entry_hash"] = lastEvent.EntryHash
		info["chain_head_valid"] = lastEvent.ChainValid
	} else {
		info["last_event_id"] = nil
		info["last_entry_hash"] = GenesisHash
		info["chain_head_valid"] = true
	}

	return info, nil
}
