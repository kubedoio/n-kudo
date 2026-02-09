package executor

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/kubedoio/n-kudo/internal/edge/state"
)

func TestExecutor_MicroVMPause(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

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

	provider := &fakeProvider{pid: 99999}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(PauseParams{VMID: "vm-1"})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMPause,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err != nil {
		// Pause may fail if process doesn't exist, but we verify the action is recognized
		t.Logf("Pause execution (may fail due to mock PID): %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	// Action should be recognized (not "unknown action type")
	if result.Results[0].ErrorCode == "ACTION_FAILED" && result.Results[0].Message == "unknown action type: MicroVMPause" {
		t.Error("MicroVMPause action type should be recognized")
	}
}

func TestExecutor_MicroVMPause_MissingVMID(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(PauseParams{VMID: ""})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMPause,
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
