package executor

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/kubedoio/n-kudo/internal/edge/state"
)

func TestExecutor_CommandExecute(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(CommandParams{
		Command: "echo",
		Args:    []string{"hello", "world"},
		Timeout: 10,
	})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionCommandExecute,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute plan failed: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if !result.Results[0].OK {
		t.Errorf("expected OK result, got error: %s", result.Results[0].Message)
	}
}

func TestExecutor_CommandExecute_WithStderr(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(CommandParams{
		Command: "sh",
		Args:    []string{"-c", "echo 'error output' >&2; exit 1"},
		Timeout: 10,
	})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionCommandExecute,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	// Command that exits with non-zero should not fail the action itself,
	// but the result should indicate the exit code
	if err != nil {
		t.Fatalf("execute plan failed unexpectedly: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	// Action should succeed even if command exits non-zero (we capture exit code)
	if !result.Results[0].OK {
		t.Errorf("expected OK result (even with non-zero exit), got error: %s", result.Results[0].Message)
	}
}

func TestExecutor_CommandExecute_MissingCommand(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(CommandParams{Command: ""})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionCommandExecute,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error for missing command")
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if result.Results[0].OK {
		t.Error("expected action to fail with empty command")
	}
}

func TestExecutor_CommandExecute_DefaultTimeout(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	// No timeout specified - should use default
	params, _ := json.Marshal(CommandParams{
		Command: "echo",
		Args:    []string{"test"},
	})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionCommandExecute,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute plan failed: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if !result.Results[0].OK {
		t.Errorf("expected OK result, got error: %s", result.Results[0].Message)
	}
}

func TestExecutor_CommandExecute_WithWorkingDir(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(CommandParams{
		Command: "pwd",
		Dir:     "/tmp",
		Timeout: 10,
	})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionCommandExecute,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute plan failed: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if !result.Results[0].OK {
		t.Errorf("expected OK result, got error: %s", result.Results[0].Message)
	}
}

func TestExecutor_CommandExecute_InvalidCommand(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(CommandParams{
		Command: "this-command-does-not-exist-12345",
		Timeout: 5,
	})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionCommandExecute,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	// Invalid command returns error - action may fail or succeed with captured error
	if err != nil {
		t.Logf("Plan failed as expected for invalid command: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	// Either the action failed or succeeded with an error code
	if !result.Results[0].OK && result.Results[0].ErrorCode != "ACTION_FAILED" {
		t.Errorf("expected action to fail with ACTION_FAILED, got: %v", result.Results[0])
	}
}
