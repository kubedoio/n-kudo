package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"syscall"
)

func (e *Executor) executeResume(ctx context.Context, action Action) error {
	var params ResumeParams
	if err := json.Unmarshal(action.Params, &params); err != nil {
		return fmt.Errorf("unmarshal resume params: %w", err)
	}

	if params.VMID == "" {
		return fmt.Errorf("vm_id is required")
	}

	// Get VM process ID
	pid, err := e.Provider.GetProcessID(ctx, params.VMID)
	if err != nil {
		return fmt.Errorf("failed to get process ID: %w", err)
	}

	// Send SIGCONT
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGCONT); err != nil {
		return fmt.Errorf("failed to resume VM: %w", err)
	}

	// Update state
	if vm, ok, _ := e.Store.GetMicroVM(params.VMID); ok {
		vm.Status = "RUNNING"
		_ = e.Store.UpsertMicroVM(vm)
	}

	return nil
}
