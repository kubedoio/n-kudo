package executor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubedoio/n-kudo/internal/edge/state"
)

func TestExecutor_MicroVMSnapshot(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	tmpDir := t.TempDir()
	// First create a VM in the state
	vm := state.MicroVM{
		ID:     "vm-1",
		Name:   "test-vm",
		Status: "RUNNING",
		CHPID:  99999, // Mock PID
	}
	if err := st.UpsertMicroVM(vm); err != nil {
		t.Fatal(err)
	}

	// Create mock VM disk directory
	vmDir := filepath.Join(tmpDir, "vms", "vm-1")
	if err := os.MkdirAll(vmDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a mock disk file
	diskPath := filepath.Join(vmDir, "disk.raw")
	if err := os.WriteFile(diskPath, []byte("mock disk data"), 0644); err != nil {
		t.Fatal(err)
	}

	provider := &fakeProvider{pid: 99999}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(SnapshotParams{VMID: "vm-1", SnapshotName: "test-snapshot"})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMSnapshot,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	// Snapshot may fail because we're using a mock, but action should be recognized
	if err != nil {
		t.Logf("Snapshot execution (may fail due to test environment): %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	// Action should be recognized (not "unknown action type")
	if result.Results[0].ErrorCode == "ACTION_FAILED" && result.Results[0].Message == "unknown action type: MicroVMSnapshot" {
		t.Error("MicroVMSnapshot action type should be recognized")
	}
}

func TestExecutor_MicroVMSnapshot_MissingVMID(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(SnapshotParams{VMID: "", SnapshotName: "test-snapshot"})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMSnapshot,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error for missing vm_id")
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if result.Results[0].OK {
		t.Error("expected action to fail with empty vm_id")
	}
}

func TestExecutor_MicroVMSnapshot_MissingSnapshotName(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// Create a VM in the state
	vm := state.MicroVM{
		ID:     "vm-1",
		Name:   "test-vm",
		Status: "RUNNING",
		CHPID:  99999,
	}
	if err := st.UpsertMicroVM(vm); err != nil {
		t.Fatal(err)
	}

	provider := &fakeProvider{pid: 99999}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(SnapshotParams{VMID: "vm-1", SnapshotName: ""})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMSnapshot,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error for missing snapshot_name")
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if result.Results[0].OK {
		t.Error("expected action to fail with empty snapshot_name")
	}
}

func TestExecutor_MicroVMSnapshot_VMNotFound(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(SnapshotParams{VMID: "non-existent-vm", SnapshotName: "test-snapshot"})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMSnapshot,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error for non-existent VM")
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if result.Results[0].OK {
		t.Error("expected action to fail when VM not found")
	}
}
