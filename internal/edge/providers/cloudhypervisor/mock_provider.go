package cloudhypervisor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockProvider is a test double for the Cloud Hypervisor provider.
type MockProvider struct {
	mu  sync.Mutex
	vms map[string]*MockVM
}

// MockVM represents a mocked VM instance.
type MockVM struct {
	ID     string
	Name   string
	State  string // CREATING, RUNNING, STOPPED, DELETED
	Config VMConfig
}

// VMConfig holds the configuration for a mocked VM.
type VMConfig struct {
	VCPUs  int
	Memory int64
	Kernel string
	Rootfs string
}

// VMInfo contains basic information about a VM for listing.
type VMInfo struct {
	ID    string
	Name  string
	State string
}

// NewMockProvider creates a new MockProvider with initialized storage.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		vms: make(map[string]*MockVM),
	}
}

// GetVMCount returns the number of VMs (thread-safe).
func (m *MockProvider) GetVMCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.vms)
}

// GetVM returns a copy of a VM by ID (thread-safe).
func (m *MockProvider) GetVM(id string) (MockVM, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	vm, ok := m.vms[id]
	if !ok {
		return MockVM{}, false
	}
	// Return a copy to avoid races
	return *vm, true
}

// CreateVM creates a new VM in the mock provider.
func (m *MockProvider) CreateVM(ctx context.Context, id, name string, vcpus int, memory int64, kernel, rootfs string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.vms[id] = &MockVM{
		ID:    id,
		Name:  name,
		State: "CREATING",
		Config: VMConfig{
			VCPUs:  vcpus,
			Memory: memory,
			Kernel: kernel,
			Rootfs: rootfs,
		},
	}
	// Simulate async creation transitioning to RUNNING
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Millisecond):
			m.mu.Lock()
			defer m.mu.Unlock()
			if vm, ok := m.vms[id]; ok {
				vm.State = "RUNNING"
			}
		}
	}()
	return nil
}

// StartVM starts a VM if it exists.
func (m *MockProvider) StartVM(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if vm, ok := m.vms[id]; ok {
		vm.State = "RUNNING"
		return nil
	}
	return fmt.Errorf("VM not found: %s", id)
}

// StopVM stops a VM if it exists.
func (m *MockProvider) StopVM(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if vm, ok := m.vms[id]; ok {
		vm.State = "STOPPED"
		return nil
	}
	return fmt.Errorf("VM not found: %s", id)
}

// DeleteVM deletes a VM if it exists.
func (m *MockProvider) DeleteVM(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.vms[id]; ok {
		delete(m.vms, id)
		return nil
	}
	return fmt.Errorf("VM not found: %s", id)
}

// GetVMStatus returns the status of a VM.
func (m *MockProvider) GetVMStatus(ctx context.Context, id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if vm, ok := m.vms[id]; ok {
		return vm.State, nil
	}
	return "", fmt.Errorf("VM not found: %s", id)
}

// ListVMs returns a list of all VMs.
func (m *MockProvider) ListVMs(ctx context.Context) ([]VMInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var vms []VMInfo
	for _, vm := range m.vms {
		vms = append(vms, VMInfo{
			ID:    vm.ID,
			Name:  vm.Name,
			State: vm.State,
		})
	}
	return vms, nil
}

// MicroVMProvider interface implementation for executor compatibility.

// Create implements the executor.MicroVMProvider interface.
func (m *MockProvider) Create(ctx context.Context, params MicroVMParams) error {
	return m.CreateVM(ctx, params.VMID, params.Name, params.VCPU, int64(params.MemoryMiB*1024*1024), params.KernelPath, params.RootfsPath)
}

// Start implements the executor.MicroVMProvider interface.
func (m *MockProvider) Start(ctx context.Context, vmID string) error {
	return m.StartVM(ctx, vmID)
}

// Stop implements the executor.MicroVMProvider interface.
func (m *MockProvider) Stop(ctx context.Context, vmID string) error {
	return m.StopVM(ctx, vmID)
}

// Delete implements the executor.MicroVMProvider interface.
func (m *MockProvider) Delete(ctx context.Context, vmID string) error {
	return m.DeleteVM(ctx, vmID)
}

// MicroVMParams mirrors the executor.MicroVMParams type to avoid import cycles in tests.
type MicroVMParams struct {
	VMID       string
	Name       string
	KernelPath string
	RootfsPath string
	TapIface   string
	VCPU       int
	MemoryMiB  int
}
