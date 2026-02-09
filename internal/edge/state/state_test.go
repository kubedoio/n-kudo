package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateSaveAndLoad(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state")

	// Open state store
	store, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer store.Close()

	// Save identity
	identity := Identity{
		TenantID:     "tenant-1",
		SiteID:       "site-1",
		HostID:       "host-1",
		AgentID:      "agent-1",
		RefreshToken: "refresh-token-123",
	}

	if err := store.SaveIdentity(identity); err != nil {
		t.Errorf("SaveIdentity failed: %v", err)
	}

	// Load identity
	loaded, err := store.LoadIdentity()
	if err != nil {
		t.Errorf("LoadIdentity failed: %v", err)
	}

	if loaded.AgentID != identity.AgentID {
		t.Errorf("Expected AgentID %s, got %s", identity.AgentID, loaded.AgentID)
	}
	if loaded.TenantID != identity.TenantID {
		t.Errorf("Expected TenantID %s, got %s", identity.TenantID, loaded.TenantID)
	}
	if loaded.SiteID != identity.SiteID {
		t.Errorf("Expected SiteID %s, got %s", identity.SiteID, loaded.SiteID)
	}
	if loaded.RefreshToken != identity.RefreshToken {
		t.Errorf("Expected RefreshToken %s, got %s", identity.RefreshToken, loaded.RefreshToken)
	}
}

func TestStateIdentityNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state")

	store, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer store.Close()

	_, err = store.LoadIdentity()
	if err == nil {
		t.Error("expected error for missing identity")
	}
}

func TestStateMicroVMOperations(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state")

	store, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer store.Close()

	// Test Upsert
	vm := MicroVM{
		ID:         "vm-1",
		Name:       "test-vm",
		KernelPath: "/path/to/kernel",
		RootfsPath: "/path/to/rootfs",
		TapIface:   "tap0",
		CHPID:      1234,
		Status:     "RUNNING",
	}

	if err := store.UpsertMicroVM(vm); err != nil {
		t.Errorf("UpsertMicroVM failed: %v", err)
	}

	// Test Get
	loaded, found, err := store.GetMicroVM("vm-1")
	if err != nil {
		t.Errorf("GetMicroVM failed: %v", err)
	}
	if !found {
		t.Error("expected VM to be found")
	}
	if loaded.ID != vm.ID {
		t.Errorf("Expected ID %s, got %s", vm.ID, loaded.ID)
	}
	if loaded.Name != vm.Name {
		t.Errorf("Expected Name %s, got %s", vm.Name, loaded.Name)
	}

	// Test Get non-existent
	_, found, err = store.GetMicroVM("vm-nonexistent")
	if err != nil {
		t.Errorf("GetMicroVM failed: %v", err)
	}
	if found {
		t.Error("expected VM to not be found")
	}

	// Test Update (same ID)
	vm.Status = "STOPPED"
	vm.CHPID = 0
	if err := store.UpsertMicroVM(vm); err != nil {
		t.Errorf("UpsertMicroVM (update) failed: %v", err)
	}

	loaded, _, _ = store.GetMicroVM("vm-1")
	if loaded.Status != "STOPPED" {
		t.Errorf("Expected status STOPPED, got %s", loaded.Status)
	}

	// Test List
	vm2 := MicroVM{
		ID:     "vm-2",
		Name:   "test-vm-2",
		Status: "RUNNING",
	}
	if err := store.UpsertMicroVM(vm2); err != nil {
		t.Fatalf("UpsertMicroVM failed: %v", err)
	}

	vms, err := store.ListMicroVMs()
	if err != nil {
		t.Errorf("ListMicroVMs failed: %v", err)
	}
	if len(vms) != 2 {
		t.Errorf("Expected 2 VMs, got %d", len(vms))
	}

	// Test Delete
	if err := store.DeleteMicroVM("vm-1"); err != nil {
		t.Errorf("DeleteMicroVM failed: %v", err)
	}

	_, found, _ = store.GetMicroVM("vm-1")
	if found {
		t.Error("expected VM to be deleted")
	}

	vms, _ = store.ListMicroVMs()
	if len(vms) != 1 {
		t.Errorf("Expected 1 VM after delete, got %d", len(vms))
	}
}

func TestStateMicroVMWithoutID(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state")

	store, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer store.Close()

	vm := MicroVM{
		ID:   "",
		Name: "test-vm",
	}

	err = store.UpsertMicroVM(vm)
	if err == nil {
		t.Error("expected error for VM without ID")
	}
}

func TestStateActionRecordOperations(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state")

	store, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer store.Close()

	// Test Put
	record := ActionRecord{
		ActionID:    "action-1",
		ExecutionID: "exec-1",
		OK:          true,
		ErrorCode:   "",
		Message:     "success",
	}

	if err := store.PutActionRecord(record); err != nil {
		t.Errorf("PutActionRecord failed: %v", err)
	}

	// Test Get
	loaded, found, err := store.GetActionRecord("action-1")
	if err != nil {
		t.Errorf("GetActionRecord failed: %v", err)
	}
	if !found {
		t.Error("expected record to be found")
	}
	if loaded.ActionID != record.ActionID {
		t.Errorf("Expected ActionID %s, got %s", record.ActionID, loaded.ActionID)
	}
	if loaded.ExecutionID != record.ExecutionID {
		t.Errorf("Expected ExecutionID %s, got %s", record.ExecutionID, loaded.ExecutionID)
	}
	if !loaded.OK {
		t.Error("expected OK to be true")
	}
	if loaded.Message != record.Message {
		t.Errorf("Expected Message %s, got %s", record.Message, loaded.Message)
	}

	// Test Get non-existent
	_, found, err = store.GetActionRecord("action-nonexistent")
	if err != nil {
		t.Errorf("GetActionRecord failed: %v", err)
	}
	if found {
		t.Error("expected record to not be found")
	}

	// Test Update (same action ID)
	record.OK = false
	record.ErrorCode = "FAILED"
	record.Message = "action failed"
	if err := store.PutActionRecord(record); err != nil {
		t.Errorf("PutActionRecord (update) failed: %v", err)
	}

	loaded, _, _ = store.GetActionRecord("action-1")
	if loaded.OK {
		t.Error("expected OK to be false after update")
	}
	if loaded.ErrorCode != "FAILED" {
		t.Errorf("expected ErrorCode FAILED, got %s", loaded.ErrorCode)
	}
}

func TestStateActionRecordWithoutID(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state")

	store, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer store.Close()

	record := ActionRecord{
		ActionID:    "",
		ExecutionID: "exec-1",
		OK:          true,
	}

	err = store.PutActionRecord(record)
	if err == nil {
		t.Error("expected error for record without ActionID")
	}
}

func TestStatePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state")

	// Create and populate store
	store1, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}

	identity := Identity{
		TenantID: "tenant-1",
		SiteID:   "site-1",
		AgentID:  "agent-1",
	}
	if err := store1.SaveIdentity(identity); err != nil {
		t.Fatalf("SaveIdentity failed: %v", err)
	}

	vm := MicroVM{
		ID:     "vm-1",
		Name:   "test-vm",
		Status: "RUNNING",
	}
	if err := store1.UpsertMicroVM(vm); err != nil {
		t.Fatalf("UpsertMicroVM failed: %v", err)
	}

	record := ActionRecord{
		ActionID:    "action-1",
		ExecutionID: "exec-1",
		OK:          true,
		Message:     "success",
	}
	if err := store1.PutActionRecord(record); err != nil {
		t.Fatalf("PutActionRecord failed: %v", err)
	}

	store1.Close()

	// Reopen store and verify data persists
	store2, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to reopen state store: %v", err)
	}
	defer store2.Close()

	loadedIdentity, err := store2.LoadIdentity()
	if err != nil {
		t.Errorf("LoadIdentity failed after reopen: %v", err)
	}
	if loadedIdentity.AgentID != "agent-1" {
		t.Errorf("Expected AgentID agent-1 after reopen, got %s", loadedIdentity.AgentID)
	}

	loadedVM, found, _ := store2.GetMicroVM("vm-1")
	if !found {
		t.Error("expected VM to be found after reopen")
	}
	if loadedVM.Name != "test-vm" {
		t.Errorf("Expected VM name test-vm after reopen, got %s", loadedVM.Name)
	}

	loadedRecord, found, _ := store2.GetActionRecord("action-1")
	if !found {
		t.Error("expected record to be found after reopen")
	}
	if loadedRecord.Message != "success" {
		t.Errorf("Expected message 'success' after reopen, got %s", loadedRecord.Message)
	}
}

func TestStateEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state")

	// Create the directory and an empty state file
	os.MkdirAll(statePath, 0o700)
	emptyFile := filepath.Join(statePath, "edge-state.json")
	os.WriteFile(emptyFile, []byte{}, 0o600)

	store, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state store with empty file: %v", err)
	}
	defer store.Close()

	// Should be able to use the store normally
	vms, err := store.ListMicroVMs()
	if err != nil {
		t.Errorf("ListMicroVMs failed: %v", err)
	}
	if len(vms) != 0 {
		t.Errorf("Expected 0 VMs, got %d", len(vms))
	}
}

func TestStateCorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state")

	// Create the directory and a corrupted state file
	os.MkdirAll(statePath, 0o700)
	corruptedFile := filepath.Join(statePath, "edge-state.json")
	os.WriteFile(corruptedFile, []byte("not valid json"), 0o600)

	_, err := Open(statePath)
	if err == nil {
		t.Error("expected error for corrupted state file")
	}
}

func TestStateUpdatedAt(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state")

	store, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer store.Close()

	// Save a VM
	before := time.Now().UTC()
	vm := MicroVM{
		ID:     "vm-1",
		Name:   "test-vm",
		Status: "RUNNING",
	}
	if err := store.UpsertMicroVM(vm); err != nil {
		t.Fatalf("UpsertMicroVM failed: %v", err)
	}

	loaded, _, _ := store.GetMicroVM("vm-1")
	if loaded.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
	if loaded.UpdatedAt.Before(before) {
		t.Error("expected UpdatedAt to be after the save time")
	}

	// Save an action record
	record := ActionRecord{
		ActionID:    "action-1",
		ExecutionID: "exec-1",
		OK:          true,
	}
	if err := store.PutActionRecord(record); err != nil {
		t.Fatalf("PutActionRecord failed: %v", err)
	}

	loadedRecord, _, _ := store.GetActionRecord("action-1")
	if loadedRecord.UpdatedAt.IsZero() {
		t.Error("expected action record UpdatedAt to be set")
	}

	// Save identity
	identity := Identity{
		AgentID: "agent-1",
	}
	store.SaveIdentity(identity)

	loadedIdentity, _ := store.LoadIdentity()
	if loadedIdentity.SavedAt.IsZero() {
		t.Error("expected identity SavedAt to be set")
	}
}

func TestStateListMicroVMsSorting(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state")

	store, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer store.Close()

	// Add VMs in non-alphabetical order
	vms := []MicroVM{
		{ID: "vm-c", Name: "c"},
		{ID: "vm-a", Name: "a"},
		{ID: "vm-b", Name: "b"},
	}

	for _, vm := range vms {
		if err := store.UpsertMicroVM(vm); err != nil {
			t.Fatalf("UpsertMicroVM failed: %v", err)
		}
	}

	list, err := store.ListMicroVMs()
	if err != nil {
		t.Fatalf("ListMicroVMs failed: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("expected 3 VMs, got %d", len(list))
	}

	// Verify sorted order (by ID)
	if list[0].ID != "vm-a" {
		t.Errorf("expected first VM to be vm-a, got %s", list[0].ID)
	}
	if list[1].ID != "vm-b" {
		t.Errorf("expected second VM to be vm-b, got %s", list[1].ID)
	}
	if list[2].ID != "vm-c" {
		t.Errorf("expected third VM to be vm-c, got %s", list[2].ID)
	}
}

func TestStateDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "nested", "state", "path")

	store, err := Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state store with nested path: %v", err)
	}
	defer store.Close()

	// Verify directory was created
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("expected state directory to be created")
	}
}
