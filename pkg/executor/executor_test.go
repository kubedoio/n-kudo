package executor

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"

	"github.com/n-kudo/n-kudo-edge/pkg/state"
)

type fakeProvider struct {
	mu     sync.Mutex
	create int
	start  int
	stop   int
	delete int
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
