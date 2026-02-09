package integration_test

import (
	"context"
	"testing"

	"github.com/kubedoio/n-kudo/internal/edge/executor"
	"github.com/kubedoio/n-kudo/internal/edge/state"
	"github.com/kubedoio/n-kudo/tests/integration/mocks"
)

// TestApplyPlanIdempotency validates that duplicate plans with the same idempotency key
// do not result in duplicate operations
func TestApplyPlanIdempotency(t *testing.T) {
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

	vmID := "idempotent-vm-001"
	actionID := "action-create-001"

	// First execution of the plan
	plan1 := executor.Plan{
		PlanID:      "plan-001",
		ExecutionID: "exec-001",
		Actions: []executor.Action{
			{
				ActionID:      actionID,
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID:       vmID,
					Name:       "idempotent-vm",
					VCPU:       1,
					MemoryMiB:  256,
					RootfsPath: "/tmp/test.raw",
				}),
			},
		},
	}

	result1, err := exec.ExecutePlan(ctx, plan1)
	if err != nil {
		t.Fatalf("first plan execution failed: %v", err)
	}

	if !result1.Results[0].OK {
		t.Fatalf("first action failed: %s", result1.Results[0].Message)
	}

	// Verify VM was created
	if mockProvider.VMCount() != 1 {
		t.Fatalf("expected 1 VM after first execution, got %d", mockProvider.VMCount())
	}

	// Second execution with the same action ID (should use cache)
	plan2 := executor.Plan{
		PlanID:      "plan-002", // Different plan ID
		ExecutionID: "exec-002", // Different execution ID
		Actions: []executor.Action{
			{
				ActionID:      actionID, // Same action ID
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID:       vmID,
					Name:       "idempotent-vm",
					VCPU:       1,
					MemoryMiB:  256,
					RootfsPath: "/tmp/test.raw",
				}),
			},
		},
	}

	result2, err := exec.ExecutePlan(ctx, plan2)
	if err != nil {
		t.Fatalf("second plan execution failed: %v", err)
	}

	if !result2.Results[0].OK {
		t.Fatalf("second action failed: %s", result2.Results[0].Message)
	}

	// Verify the result indicates it came from cache (message should indicate reuse)
	if result2.Results[0].Message != "ok" {
		t.Fatalf("expected message 'ok', got: %s", result2.Results[0].Message)
	}

	// Most importantly: verify only 1 VM was created (not 2)
	if mockProvider.VMCount() != 1 {
		t.Fatalf("idempotency violated: expected 1 VM, got %d", mockProvider.VMCount())
	}

	// Verify provider was only called once (second execution used cache)
	if len(mockProvider.CreateCalls) != 1 {
		t.Fatalf("expected 1 create call (idempotency), got %d", len(mockProvider.CreateCalls))
	}

	t.Log("Idempotency test passed: duplicate action ID was cached")
}

// TestActionCachePersistence validates that action cache survives across executor instances
func TestActionCachePersistence(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	mockProvider := mocks.NewMockCloudHypervisor()

	// First executor instance
	st1, err := state.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open state store: %v", err)
	}

	exec1 := &executor.Executor{
		Store:    st1,
		Provider: mockProvider,
		Logs:     &testLogSink{t: t},
	}

	vmID := "persistent-vm-001"
	actionID := "action-persistent-001"

	plan := executor.Plan{
		PlanID:      "plan-001",
		ExecutionID: "exec-001",
		Actions: []executor.Action{
			{
				ActionID:      actionID,
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID:       vmID,
					Name:       "persistent-vm",
					VCPU:       1,
					MemoryMiB:  256,
					RootfsPath: "/tmp/test.raw",
				}),
			},
		},
	}

	result1, err := exec1.ExecutePlan(ctx, plan)
	if err != nil {
		t.Fatalf("first execution failed: %v", err)
	}

	if !result1.Results[0].OK {
		t.Fatalf("first action failed: %s", result1.Results[0].Message)
	}

	st1.Close()

	// Second executor instance with fresh state store (simulating restart)
	st2, err := state.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to reopen state store: %v", err)
	}
	defer st2.Close()

	exec2 := &executor.Executor{
		Store:    st2,
		Provider: mockProvider,
		Logs:     &testLogSink{t: t},
	}

	// Execute same plan again
	plan2 := executor.Plan{
		PlanID:      "plan-002",
		ExecutionID: "exec-002",
		Actions: []executor.Action{
			{
				ActionID:      actionID, // Same action ID
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID:       vmID,
					Name:       "persistent-vm",
					VCPU:       1,
					MemoryMiB:  256,
					RootfsPath: "/tmp/test.raw",
				}),
			},
		},
	}

	result2, err := exec2.ExecutePlan(ctx, plan2)
	if err != nil {
		t.Fatalf("second execution failed: %v", err)
	}

	if !result2.Results[0].OK {
		t.Fatalf("second action failed: %s", result2.Results[0].Message)
	}

	// Verify still only 1 VM (cache was persisted)
	if mockProvider.VMCount() != 1 {
		t.Fatalf("cache persistence failed: expected 1 VM, got %d", mockProvider.VMCount())
	}

	if len(mockProvider.CreateCalls) != 1 {
		t.Fatalf("expected 1 create call after cache restore, got %d", len(mockProvider.CreateCalls))
	}

	t.Log("Cache persistence test passed")
}

// TestDifferentActionIDsNotDeduped validates that different action IDs result in separate operations
func TestDifferentActionIDsNotDeduped(t *testing.T) {
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

	// First VM
	plan1 := executor.Plan{
		PlanID:      "plan-001",
		ExecutionID: "exec-001",
		Actions: []executor.Action{
			{
				ActionID:      "action-vm-001",
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID:       "vm-001",
					Name:       "vm-001",
					VCPU:       1,
					MemoryMiB:  256,
					RootfsPath: "/tmp/test.raw",
				}),
			},
		},
	}

	result1, err := exec.ExecutePlan(ctx, plan1)
	if err != nil {
		t.Fatalf("first plan failed: %v", err)
	}
	if !result1.Results[0].OK {
		t.Fatalf("first action failed: %s", result1.Results[0].Message)
	}

	// Second VM with different action ID (should create new VM)
	plan2 := executor.Plan{
		PlanID:      "plan-002",
		ExecutionID: "exec-002",
		Actions: []executor.Action{
			{
				ActionID:      "action-vm-002", // Different action ID
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID:       "vm-002", // Different VM
					Name:       "vm-002",
					VCPU:       1,
					MemoryMiB:  256,
					RootfsPath: "/tmp/test2.raw",
				}),
			},
		},
	}

	result2, err := exec.ExecutePlan(ctx, plan2)
	if err != nil {
		t.Fatalf("second plan failed: %v", err)
	}
	if !result2.Results[0].OK {
		t.Fatalf("second action failed: %s", result2.Results[0].Message)
	}

	// Verify 2 VMs were created
	if mockProvider.VMCount() != 2 {
		t.Fatalf("expected 2 VMs, got %d", mockProvider.VMCount())
	}

	if len(mockProvider.CreateCalls) != 2 {
		t.Fatalf("expected 2 create calls, got %d", len(mockProvider.CreateCalls))
	}

	t.Log("Different action IDs correctly created separate VMs")
}

// TestFailedActionCached validates that failed actions ARE cached (idempotency includes failures)
// This ensures consistent behavior - the same action ID always produces the same result
func TestFailedActionCached(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	mockProvider := mocks.NewMockCloudHypervisor()
	mockProvider.FailCreate = true // Force creation to fail

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

	actionID := "action-fail-001"

	// First attempt - should fail
	plan1 := executor.Plan{
		PlanID:      "plan-001",
		ExecutionID: "exec-001",
		Actions: []executor.Action{
			{
				ActionID:      actionID,
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID:       "fail-vm",
					Name:       "fail-vm",
					VCPU:       1,
					MemoryMiB:  256,
					RootfsPath: "/tmp/test.raw",
				}),
			},
		},
	}

	result1, err := exec.ExecutePlan(ctx, plan1)
	if err == nil {
		t.Fatal("expected first execution to fail")
	}
	if result1.Results[0].OK {
		t.Fatal("expected first action to fail")
	}

	// Second attempt with same action ID - should use cached failure (not retry)
	plan2 := executor.Plan{
		PlanID:      "plan-002",
		ExecutionID: "exec-002",
		Actions: []executor.Action{
			{
				ActionID:      actionID, // Same action ID
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID:       "fail-vm",
					Name:       "fail-vm",
					VCPU:       1,
					MemoryMiB:  256,
					RootfsPath: "/tmp/test.raw",
				}),
			},
		},
	}

	result2, err := exec.ExecutePlan(ctx, plan2)
	if err == nil {
		t.Fatal("expected second execution to also fail (cached)")
	}
	if result2.Results[0].OK {
		t.Fatal("expected second action to fail (from cache)")
	}

	// Verify only 1 create call was made (second used cache)
	if len(mockProvider.CreateCalls) != 1 {
		t.Fatalf("expected 1 create call (cached failure), got %d", len(mockProvider.CreateCalls))
	}

	t.Log("Failed action correctly cached - idempotency preserves failure results")
}

// TestMixedIdempotencyInPlan validates idempotency works with multiple actions in one plan
func TestMixedIdempotencyInPlan(t *testing.T) {
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

	// First plan creates 2 VMs
	plan1 := executor.Plan{
		PlanID:      "plan-001",
		ExecutionID: "exec-001",
		Actions: []executor.Action{
			{
				ActionID:      "action-create-001",
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID: "vm-001", Name: "vm-001",
				}),
			},
			{
				ActionID:      "action-create-002",
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID: "vm-002", Name: "vm-002",
				}),
			},
		},
	}

	result1, err := exec.ExecutePlan(ctx, plan1)
	if err != nil {
		t.Fatalf("first plan failed: %v", err)
	}
	if len(result1.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result1.Results))
	}
	for i, r := range result1.Results {
		if !r.OK {
			t.Fatalf("action %d failed: %s", i, r.Message)
		}
	}

	// Second plan tries to create same VMs again
	plan2 := executor.Plan{
		PlanID:      "plan-002",
		ExecutionID: "exec-002",
		Actions: []executor.Action{
			{
				ActionID:      "action-create-001", // Same action ID
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID: "vm-001", Name: "vm-001",
				}),
			},
			{
				ActionID:      "action-create-002", // Same action ID
				Type:          executor.ActionMicroVMCreate,
				TimeoutSecond: 60,
				Params: mustJSON(t, executor.MicroVMParams{
					VMID: "vm-002", Name: "vm-002",
				}),
			},
		},
	}

	result2, err := exec.ExecutePlan(ctx, plan2)
	if err != nil {
		t.Fatalf("second plan failed: %v", err)
	}

	if len(result2.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result2.Results))
	}
	for i, r := range result2.Results {
		if !r.OK {
			t.Fatalf("result %d failed: %s", i, r.Message)
		}
	}

	// Verify still only 2 VMs
	if mockProvider.VMCount() != 2 {
		t.Fatalf("expected 2 VMs (cached), got %d", mockProvider.VMCount())
	}

	// Verify only 2 create calls total (both cached)
	if len(mockProvider.CreateCalls) != 2 {
		t.Fatalf("expected 2 create calls (both cached), got %d", len(mockProvider.CreateCalls))
	}

	t.Log("Multi-action plan idempotency test passed")
}
