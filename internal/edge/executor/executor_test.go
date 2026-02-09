package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kubedoio/n-kudo/internal/edge/state"
)

type fakeProvider struct {
	mu       sync.Mutex
	create   int
	start    int
	stop     int
	delete   int
	pause    int
	resume   int
	pid      int
}

func (f *fakeProvider) Create(context.Context, MicroVMParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.create++
	return nil
}
func (f *fakeProvider) Start(context.Context, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.start++
	return nil
}
func (f *fakeProvider) Stop(context.Context, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stop++
	return nil
}
func (f *fakeProvider) Delete(context.Context, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.delete++
	return nil
}
func (f *fakeProvider) GetProcessID(ctx context.Context, vmID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.pid > 0 {
		return f.pid, nil
	}
	return 0, fmt.Errorf("VM not running: %s", vmID)
}

type noOpSink struct{}

func (n *noOpSink) Write(context.Context, LogEntry) {}

func TestExecutorIdempotencyByActionID(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(MicroVMParams{VMID: "vm-1", Name: "demo", KernelPath: "/k", RootfsPath: "/r"})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMCreate,
			Params:   params,
		}},
	}

	if _, err := exec.ExecutePlan(context.Background(), plan); err != nil {
		t.Fatalf("first execute failed: %v", err)
	}
	if _, err := exec.ExecutePlan(context.Background(), plan); err != nil {
		t.Fatalf("second execute failed: %v", err)
	}

	if provider.create != 1 {
		t.Fatalf("expected create to run once, got %d", provider.create)
	}
}

func TestExecutor_MicroVMCreate(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(MicroVMParams{VMID: "vm-1", Name: "test-vm", VCPU: 2, MemoryMiB: 512})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMCreate,
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

	if provider.create != 1 {
		t.Errorf("expected create to be called once, got %d", provider.create)
	}
}

func TestExecutor_MicroVMStart(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(MicroVMParams{VMID: "vm-1"})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMStart,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute plan failed: %v", err)
	}

	if !result.Results[0].OK {
		t.Errorf("expected OK result, got error: %s", result.Results[0].Message)
	}

	if provider.start != 1 {
		t.Errorf("expected start to be called once, got %d", provider.start)
	}
}

func TestExecutor_MicroVMStop(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(MicroVMParams{VMID: "vm-1"})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMStop,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute plan failed: %v", err)
	}

	if !result.Results[0].OK {
		t.Errorf("expected OK result, got error: %s", result.Results[0].Message)
	}

	if provider.stop != 1 {
		t.Errorf("expected stop to be called once, got %d", provider.stop)
	}
}

func TestExecutor_MicroVMDelete(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(MicroVMParams{VMID: "vm-1"})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMDelete,
			Params:   params,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute plan failed: %v", err)
	}

	if !result.Results[0].OK {
		t.Errorf("expected OK result, got error: %s", result.Results[0].Message)
	}

	if provider.delete != 1 {
		t.Errorf("expected delete to be called once, got %d", provider.delete)
	}
}

func TestExecutor_MultipleActions(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	createParams, _ := json.Marshal(MicroVMParams{VMID: "vm-1", Name: "test", VCPU: 1, MemoryMiB: 256})
	stopParams, _ := json.Marshal(MicroVMParams{VMID: "vm-1"})

	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{
			{
				ActionID: "act-1",
				Type:     ActionMicroVMCreate,
				Params:   createParams,
			},
			{
				ActionID: "act-2",
				Type:     ActionMicroVMStop,
				Params:   stopParams,
			},
		},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute plan failed: %v", err)
	}

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}

	if provider.create != 1 || provider.stop != 1 {
		t.Errorf("expected create=1 and stop=1, got create=%d, stop=%d", provider.create, provider.stop)
	}
}

func TestExecutor_ExecutionIDRequired(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	plan := Plan{
		ExecutionID: "",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     ActionMicroVMCreate,
			Params:   []byte(`{}`),
		}},
	}

	_, err = exec.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error for missing execution_id")
	}
}

func TestExecutor_UnknownActionType(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID: "act-1",
			Type:     "UnknownAction",
			Params:   []byte(`{}`),
		}},
	}

	_, err = exec.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error for unknown action type")
	}
}

func TestExecutor_ActionFailureStopsPlan(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// Create a failing provider
	failingProvider := &fakeFailingProvider{}
	exec := &Executor{Store: st, Provider: failingProvider, Logs: &noOpSink{}}

	createParams, _ := json.Marshal(MicroVMParams{VMID: "vm-1", Name: "test"})
	stopParams, _ := json.Marshal(MicroVMParams{VMID: "vm-2"})

	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{
			{
				ActionID: "act-1",
				Type:     ActionMicroVMCreate,
				Params:   createParams,
			},
			{
				ActionID: "act-2",
				Type:     ActionMicroVMStop,
				Params:   stopParams,
			},
		},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error when action fails")
	}

	// First action succeeds, second fails
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}

	if !result.Results[0].OK {
		t.Error("expected first action to succeed")
	}

	if result.Results[1].OK {
		t.Error("expected second action to fail")
	}
}

type fakeFailingProvider struct{}

func (f *fakeFailingProvider) Create(ctx context.Context, params MicroVMParams) error {
	if params.VMID == "vm-1" {
		return nil // First VM succeeds
	}
	return fmt.Errorf("mock create failure")
}

func (f *fakeFailingProvider) Start(ctx context.Context, vmID string) error {
	return nil
}

func (f *fakeFailingProvider) Stop(ctx context.Context, vmID string) error {
	if vmID == "vm-2" {
		return fmt.Errorf("stop failed")
	}
	return nil
}

func (f *fakeFailingProvider) Delete(ctx context.Context, vmID string) error {
	return nil
}

func (f *fakeFailingProvider) GetProcessID(ctx context.Context, vmID string) (int, error) {
	return 0, fmt.Errorf("VM not running: %s", vmID)
}

func TestExecutor_ActionTimeout(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(MicroVMParams{VMID: "vm-1"})
	plan := Plan{
		ExecutionID: "exec-1",
		Actions: []Action{{
			ActionID:      "act-1",
			Type:          ActionMicroVMCreate,
			Params:        params,
			TimeoutSecond: 30,
		}},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute plan failed: %v", err)
	}

	if !result.Results[0].OK {
		t.Errorf("expected OK result, got error: %s", result.Results[0].Message)
	}
}

func TestExecutor_ActionCaching(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	params, _ := json.Marshal(MicroVMParams{VMID: "vm-1", Name: "test"})
	action := Action{
		ActionID: "cached-act-1",
		Type:     ActionMicroVMCreate,
		Params:   params,
	}

	// First execution
	plan1 := Plan{ExecutionID: "exec-1", Actions: []Action{action}}
	_, err = exec.ExecutePlan(context.Background(), plan1)
	if err != nil {
		t.Fatalf("first execute failed: %v", err)
	}

	// Second execution with same action ID
	plan2 := Plan{ExecutionID: "exec-2", Actions: []Action{action}}
	result2, err := exec.ExecutePlan(context.Background(), plan2)
	if err != nil {
		t.Fatalf("second execute failed: %v", err)
	}

	if !result2.Results[0].OK {
		t.Errorf("expected cached result to be OK")
	}

	// Provider should only be called once due to caching
	if provider.create != 1 {
		t.Errorf("expected create to be called once (cached), got %d", provider.create)
	}
}

func TestExecutor_EmptyPlan(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeProvider{}
	exec := &Executor{Store: st, Provider: provider, Logs: &noOpSink{}}

	plan := Plan{
		ExecutionID: "exec-1",
		Actions:     []Action{},
	}

	result, err := exec.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute plan failed: %v", err)
	}

	if len(result.Results) != 0 {
		t.Errorf("expected 0 results for empty plan, got %d", len(result.Results))
	}
}
