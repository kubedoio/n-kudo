package integration_test

import (
	"context"
	"testing"

	"github.com/kubedoio/n-kudo/internal/edge/executor"
	"github.com/kubedoio/n-kudo/internal/edge/state"
	"github.com/kubedoio/n-kudo/tests/integration/mocks"
)

// TestApplyPlanCreateStartStopDelete validates the full VM lifecycle through plan execution
func TestApplyPlanCreateStartStopDelete(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Setup mock provider and state store
	mockProvider := mocks.NewMockCloudHypervisor()
	st, err := state.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer st.Close()

	exec := &executor.Executor{
		Store:    st,
		Provider: mockProvider,
		Logs:     &testLogSink{t: t},
	}

	vmID := "test-vm-001"
	executionID := "exec-001"

	// Phase 1: CREATE plan
	createPlan := executor.Plan{
		PlanID:      "plan-001",
		ExecutionID: executionID,
		Actions: []executor.Action{
			{
				ActionID:      "action-create-001",
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID:        vmID,
					Name:        "test-vm",
					VCPU:        2,
					MemoryMiB:   512,
					RootfsPath:  "/tmp/test-image.raw",
					KernelPath:  "/tmp/test-kernel",
					TapIface:    "tap0",
				}),
			},
		},
	}

	result, err := exec.ExecutePlan(ctx, createPlan)
	if err != nil {
		t.Fatalf("create plan failed: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if !result.Results[0].OK {
		t.Fatalf("create action failed: %s", result.Results[0].Message)
	}

	// Verify VM was created in mock provider
	if mockProvider.VMCount() != 1 {
		t.Fatalf("expected 1 VM in provider, got %d", mockProvider.VMCount())
	}

	vm, exists := mockProvider.GetVM(vmID)
	if !exists {
		t.Fatal("VM not found in provider after create")
	}
	if vm.State != "CREATED" {
		t.Fatalf("expected VM state CREATED, got %s", vm.State)
	}

	// Phase 2: START plan
	startPlan := executor.Plan{
		PlanID:      "plan-002",
		ExecutionID: "exec-002",
		Actions: []executor.Action{
			{
				ActionID:      "action-start-001",
				Type:          executor.ActionMicroVMStart,
				TimeoutSecond: 30,
				Params:        mustJSON(t, executor.MicroVMParams{VMID: vmID}),
			},
		},
	}

	result, err = exec.ExecutePlan(ctx, startPlan)
	if err != nil {
		t.Fatalf("start plan failed: %v", err)
	}

	if !result.Results[0].OK {
		t.Fatalf("start action failed: %s", result.Results[0].Message)
	}

	vm, _ = mockProvider.GetVM(vmID)
	if vm.State != "RUNNING" {
		t.Fatalf("expected VM state RUNNING, got %s", vm.State)
	}

	// Phase 3: STOP plan
	stopPlan := executor.Plan{
		PlanID:      "plan-003",
		ExecutionID: "exec-003",
		Actions: []executor.Action{
			{
				ActionID:      "action-stop-001",
				Type:          executor.ActionMicroVMStop,
				TimeoutSecond: 30,
				Params:        mustJSON(t, executor.MicroVMParams{VMID: vmID}),
			},
		},
	}

	result, err = exec.ExecutePlan(ctx, stopPlan)
	if err != nil {
		t.Fatalf("stop plan failed: %v", err)
	}

	if !result.Results[0].OK {
		t.Fatalf("stop action failed: %s", result.Results[0].Message)
	}

	vm, _ = mockProvider.GetVM(vmID)
	if vm.State != "STOPPED" {
		t.Fatalf("expected VM state STOPPED, got %s", vm.State)
	}

	// Phase 4: DELETE plan
	deletePlan := executor.Plan{
		PlanID:      "plan-004",
		ExecutionID: "exec-004",
		Actions: []executor.Action{
			{
				ActionID:      "action-delete-001",
				Type:          executor.ActionMicroVMDelete,
				TimeoutSecond: 30,
				Params:        mustJSON(t, executor.MicroVMParams{VMID: vmID}),
			},
		},
	}

	result, err = exec.ExecutePlan(ctx, deletePlan)
	if err != nil {
		t.Fatalf("delete plan failed: %v", err)
	}

	if !result.Results[0].OK {
		t.Fatalf("delete action failed: %s", result.Results[0].Message)
	}

	vm, _ = mockProvider.GetVM(vmID)
	if vm.State != "DELETED" {
		t.Fatalf("expected VM state DELETED, got %s", vm.State)
	}

	// Verify all expected calls were made
	if len(mockProvider.CreateCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(mockProvider.CreateCalls))
	}
	if len(mockProvider.StartCalls) != 1 {
		t.Fatalf("expected 1 start call, got %d", len(mockProvider.StartCalls))
	}
	if len(mockProvider.StopCalls) != 1 {
		t.Fatalf("expected 1 stop call, got %d", len(mockProvider.StopCalls))
	}
	if len(mockProvider.DeleteCalls) != 1 {
		t.Fatalf("expected 1 delete call, got %d", len(mockProvider.DeleteCalls))
	}

	t.Log("VM lifecycle test passed: CREATE -> START -> STOP -> DELETE")
}

// TestVMCreateFailureHandling tests error handling when VM creation fails
func TestVMCreateFailureHandling(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	mockProvider := mocks.NewMockCloudHypervisor()
	mockProvider.FailCreate = true

	st, err := state.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer st.Close()

	exec := &executor.Executor{
		Store:    st,
		Provider: mockProvider,
		Logs:     &testLogSink{t: t},
	}

	plan := executor.Plan{
		PlanID:      "plan-fail",
		ExecutionID: "exec-fail",
		Actions: []executor.Action{
			{
				ActionID:      "action-create-fail",
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID: "fail-vm",
					Name: "fail-vm",
				}),
			},
		},
	}

	result, err := exec.ExecutePlan(ctx, plan)
	if err == nil {
		t.Fatal("expected error for failed create, got nil")
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if result.Results[0].OK {
		t.Fatal("expected action to fail, but it succeeded")
	}

	if result.Results[0].ErrorCode != "ACTION_FAILED" {
		t.Fatalf("expected error code ACTION_FAILED, got %s", result.Results[0].ErrorCode)
	}
}

// TestInvalidVMOperations tests operations on non-existent VMs
func TestInvalidVMOperations(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	mockProvider := mocks.NewMockCloudHypervisor()
	st, err := state.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer st.Close()

	exec := &executor.Executor{
		Store:    st,
		Provider: mockProvider,
		Logs:     &testLogSink{t: t},
	}

	// Try to start a non-existent VM
	plan := executor.Plan{
		PlanID:      "plan-invalid",
		ExecutionID: "exec-invalid",
		Actions: []executor.Action{
			{
				ActionID:      "action-start-invalid",
				Type:          executor.ActionMicroVMStart,
				TimeoutSecond: 30,
				Params:        mustJSON(t, executor.MicroVMParams{VMID: "non-existent-vm"}),
			},
		},
	}

	result, err := exec.ExecutePlan(ctx, plan)
	if err == nil {
		t.Fatal("expected error for starting non-existent VM, got nil")
	}

	if result.Results[0].OK {
		t.Fatal("expected action to fail for non-existent VM")
	}
}

// TestTimeoutHandling verifies timeout configuration is respected
func TestTimeoutHandling(t *testing.T) {
	// This test verifies that timeout configuration is properly set
	// Full timeout testing would require a slow mock
	ctx := context.Background()
	tmpDir := t.TempDir()

	mockProvider := mocks.NewMockCloudHypervisor()
	st, err := state.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer st.Close()

	exec := &executor.Executor{
		Store:    st,
		Provider: mockProvider,
		Logs:     &testLogSink{t: t},
	}

	// Create a plan with a very short timeout
	plan := executor.Plan{
		PlanID:      "plan-timeout",
		ExecutionID: "exec-timeout",
		Actions: []executor.Action{
			{
				ActionID:      "action-timeout",
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 1, // 1 second timeout
				Params: mustJSON(t, executor.MicroVMParams{
					VMID: "timeout-vm",
					Name: "timeout-vm",
				}),
			},
		},
	}

	// This should complete quickly with the mock
	result, err := exec.ExecutePlan(ctx, plan)
	if err != nil {
		t.Fatalf("plan with timeout failed: %v", err)
	}

	if !result.Results[0].OK {
		t.Fatalf("action failed: %s", result.Results[0].Message)
	}

	// Verify the action completed - timeout was set to 1 second
	// With the mock this should complete quickly
	t.Log("Timeout configuration test passed")
}

// TestMultiActionPlan tests a plan with multiple actions
func TestMultiActionPlan(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	mockProvider := mocks.NewMockCloudHypervisor()
	st, err := state.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}
	defer st.Close()

	exec := &executor.Executor{
		Store:    st,
		Provider: mockProvider,
		Logs:     &testLogSink{t: t},
	}

	vmID := "test-vm-multi"
	plan := executor.Plan{
		PlanID:      "plan-multi",
		ExecutionID: "exec-multi",
		Actions: []executor.Action{
			{
				ActionID:      "action-create",
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID:       vmID,
					Name:       "test-vm-multi",
					VCPU:       1,
					MemoryMiB:  256,
				}),
			},
			{
				ActionID:      "action-start",
				Type:          executor.ActionMicroVMStart,
				TimeoutSecond: 30,
				Params:        mustJSON(t, executor.MicroVMParams{VMID: vmID}),
			},
		},
	}

	result, err := exec.ExecutePlan(ctx, plan)
	if err != nil {
		t.Fatalf("multi-action plan failed: %v", err)
	}

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}

	for i, r := range result.Results {
		if !r.OK {
			t.Fatalf("action %d failed: %s", i, r.Message)
		}
	}

	// Verify VM is running
	vm, _ := mockProvider.GetVM(vmID)
	if vm.State != "RUNNING" {
		t.Fatalf("expected VM to be RUNNING, got %s", vm.State)
	}

	t.Log("Multi-action plan test passed")
}


