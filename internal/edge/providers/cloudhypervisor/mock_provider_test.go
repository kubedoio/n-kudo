package cloudhypervisor

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMockProvider_CreateVM(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	err := provider.CreateVM(ctx, "vm-1", "test-vm", 2, 512*1024*1024, "/kernel", "/rootfs")
	if err != nil {
		t.Fatalf("CreateVM failed: %v", err)
	}

	// Verify VM was created
	if provider.GetVMCount() != 1 {
		t.Errorf("expected 1 VM, got %d", provider.GetVMCount())
	}

	vm, ok := provider.GetVM("vm-1")
	if !ok {
		t.Fatal("VM not found in map")
	}

	if vm.Name != "test-vm" {
		t.Errorf("expected name 'test-vm', got %s", vm.Name)
	}

	if vm.Config.VCPUs != 2 {
		t.Errorf("expected 2 vCPUs, got %d", vm.Config.VCPUs)
	}

	// Wait for async state transition with polling
	start := time.Now()
	for time.Since(start) < time.Second {
		vm, _ = provider.GetVM("vm-1")
		if vm.State == "RUNNING" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Get fresh copy for final check
	vm, _ = provider.GetVM("vm-1")
	if vm.State != "RUNNING" {
		t.Errorf("expected state RUNNING, got %s", vm.State)
	}
}

func TestMockProvider_StartVM(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	// Pre-create a stopped VM
	err := provider.CreateVM(ctx, "vm-1", "test-vm", 1, 256, "", "")
	if err != nil {
		t.Fatalf("CreateVM failed: %v", err)
	}
	
	// Wait for initial creation to complete, then stop
	start := time.Now()
	for time.Since(start) < time.Second {
		vm, _ := provider.GetVM("vm-1")
		if vm.State == "RUNNING" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	
	// Stop the VM first
	err = provider.StopVM(ctx, "vm-1")
	if err != nil {
		t.Fatalf("StopVM failed: %v", err)
	}
	
	// Now test StartVM
	err = provider.StartVM(ctx, "vm-1")
	if err != nil {
		t.Fatalf("StartVM failed: %v", err)
	}

	vm, ok := provider.GetVM("vm-1")
	if !ok {
		t.Fatal("VM not found")
	}
	if vm.State != "RUNNING" {
		t.Errorf("expected state RUNNING, got %s", vm.State)
	}
}

func TestMockProvider_StartVM_NotFound(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	err := provider.StartVM(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent VM")
	}
}

func TestMockProvider_StopVM(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	// Pre-create a running VM
	err := provider.CreateVM(ctx, "vm-1", "test-vm", 1, 256, "", "")
	if err != nil {
		t.Fatalf("CreateVM failed: %v", err)
	}
	
	// Wait for VM to be RUNNING
	start := time.Now()
	for time.Since(start) < time.Second {
		vm, _ := provider.GetVM("vm-1")
		if vm.State == "RUNNING" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	err = provider.StopVM(ctx, "vm-1")
	if err != nil {
		t.Fatalf("StopVM failed: %v", err)
	}

	vm, ok := provider.GetVM("vm-1")
	if !ok {
		t.Fatal("VM not found")
	}
	if vm.State != "STOPPED" {
		t.Errorf("expected state STOPPED, got %s", vm.State)
	}
}

func TestMockProvider_StopVM_NotFound(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	err := provider.StopVM(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent VM")
	}
}

func TestMockProvider_DeleteVM(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	// Pre-create a VM
	err := provider.CreateVM(ctx, "vm-1", "test-vm", 1, 256, "", "")
	if err != nil {
		t.Fatalf("CreateVM failed: %v", err)
	}

	err = provider.DeleteVM(ctx, "vm-1")
	if err != nil {
		t.Fatalf("DeleteVM failed: %v", err)
	}

	if _, ok := provider.GetVM("vm-1"); ok {
		t.Error("VM should have been deleted")
	}
}

func TestMockProvider_DeleteVM_NotFound(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	err := provider.DeleteVM(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent VM")
	}
}

func TestMockProvider_GetVMStatus(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	// Pre-create a VM
	err := provider.CreateVM(ctx, "vm-1", "test-vm", 1, 256, "", "")
	if err != nil {
		t.Fatalf("CreateVM failed: %v", err)
	}
	
	// Wait for VM to be RUNNING
	start := time.Now()
	for time.Since(start) < time.Second {
		vm, _ := provider.GetVM("vm-1")
		if vm.State == "RUNNING" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	status, err := provider.GetVMStatus(ctx, "vm-1")
	if err != nil {
		t.Fatalf("GetVMStatus failed: %v", err)
	}

	if status != "RUNNING" {
		t.Errorf("expected status RUNNING, got %s", status)
	}
}

func TestMockProvider_GetVMStatus_NotFound(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	_, err := provider.GetVMStatus(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent VM")
	}
}

func TestMockProvider_ListVMs(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	// Pre-create VMs
	err := provider.CreateVM(ctx, "vm-1", "test-vm-1", 1, 256, "", "")
	if err != nil {
		t.Fatalf("CreateVM vm-1 failed: %v", err)
	}
	err = provider.CreateVM(ctx, "vm-2", "test-vm-2", 1, 256, "", "")
	if err != nil {
		t.Fatalf("CreateVM vm-2 failed: %v", err)
	}
	
	// Wait for both VMs to be RUNNING
	start := time.Now()
	for time.Since(start) < time.Second {
		vm1, ok1 := provider.GetVM("vm-1")
		vm2, ok2 := provider.GetVM("vm-2")
		if ok1 && ok2 && vm1.State == "RUNNING" && vm2.State == "RUNNING" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	
	// Stop vm-2
	err = provider.StopVM(ctx, "vm-2")
	if err != nil {
		t.Fatalf("StopVM failed: %v", err)
	}

	vms, err := provider.ListVMs(ctx)
	if err != nil {
		t.Fatalf("ListVMs failed: %v", err)
	}

	if len(vms) != 2 {
		t.Errorf("expected 2 VMs, got %d", len(vms))
	}

	// Verify VM info
	found := make(map[string]bool)
	for _, vm := range vms {
		found[vm.ID] = true
		if vm.ID == "vm-1" && vm.State != "RUNNING" {
			t.Errorf("expected vm-1 to be RUNNING, got %s", vm.State)
		}
		if vm.ID == "vm-2" && vm.State != "STOPPED" {
			t.Errorf("expected vm-2 to be STOPPED, got %s", vm.State)
		}
	}

	if !found["vm-1"] || !found["vm-2"] {
		t.Error("not all VMs found in list")
	}
}

func TestMockProvider_ListVMs_Empty(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	vms, err := provider.ListVMs(ctx)
	if err != nil {
		t.Fatalf("ListVMs failed: %v", err)
	}

	if len(vms) != 0 {
		t.Errorf("expected 0 VMs, got %d", len(vms))
	}
}

// Test MicroVMProvider interface implementation

func TestMockProvider_Create(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	params := MicroVMParams{
		VMID:       "vm-1",
		Name:       "test-vm",
		KernelPath: "/kernel",
		RootfsPath: "/rootfs",
		VCPU:       2,
		MemoryMiB:  512,
	}

	err := provider.Create(ctx, params)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, ok := provider.GetVM("vm-1"); !ok {
		t.Error("VM should have been created")
	}
}

func TestMockProvider_Start(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	// Pre-create a VM
	err := provider.CreateVM(ctx, "vm-1", "test-vm", 1, 256, "", "")
	if err != nil {
		t.Fatalf("CreateVM failed: %v", err)
	}
	
	// Wait for initial creation to complete, then stop
	start := time.Now()
	for time.Since(start) < time.Second {
		vm, _ := provider.GetVM("vm-1")
		if vm.State == "RUNNING" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	
	// Stop the VM first
	err = provider.StopVM(ctx, "vm-1")
	if err != nil {
		t.Fatalf("StopVM failed: %v", err)
	}

	err = provider.Start(ctx, "vm-1")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	vm, ok := provider.GetVM("vm-1")
	if !ok {
		t.Fatal("VM not found")
	}
	if vm.State != "RUNNING" {
		t.Errorf("expected state RUNNING, got %s", vm.State)
	}
}

func TestMockProvider_Stop(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	// Pre-create a running VM
	err := provider.CreateVM(ctx, "vm-1", "test-vm", 1, 256, "", "")
	if err != nil {
		t.Fatalf("CreateVM failed: %v", err)
	}
	
	// Wait for VM to be RUNNING
	start := time.Now()
	for time.Since(start) < time.Second {
		vm, _ := provider.GetVM("vm-1")
		if vm.State == "RUNNING" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	err = provider.Stop(ctx, "vm-1")
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	vm, ok := provider.GetVM("vm-1")
	if !ok {
		t.Fatal("VM not found")
	}
	if vm.State != "STOPPED" {
		t.Errorf("expected state STOPPED, got %s", vm.State)
	}
}

func TestMockProvider_Delete(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	// Pre-create a VM
	err := provider.CreateVM(ctx, "vm-1", "test-vm", 1, 256, "", "")
	if err != nil {
		t.Fatalf("CreateVM failed: %v", err)
	}

	err = provider.Delete(ctx, "vm-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, ok := provider.GetVM("vm-1"); ok {
		t.Error("VM should have been deleted")
	}
}

func TestMockProvider_ConcurrentAccess(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()
	numGoroutines := 10

	// Create multiple VMs concurrently
	done := make(chan bool, numGoroutines)
	errChan := make(chan error, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			vmID := fmt.Sprintf("vm-%d", id)
			err := provider.CreateVM(ctx, vmID, "test", 1, 256, "", "")
			errChan <- err
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Collect errors after goroutines complete
	for i := 0; i < numGoroutines; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("CreateVM failed: %v", err)
		}
	}

	if provider.GetVMCount() != numGoroutines {
		t.Errorf("expected %d VMs, got %d", numGoroutines, provider.GetVMCount())
	}
}
