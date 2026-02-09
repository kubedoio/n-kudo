package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/kubedoio/n-kudo/internal/edge/executor"
)

// MockCloudHypervisor is a test double for the Cloud Hypervisor provider
type MockCloudHypervisor struct {
	mu sync.RWMutex

	// VM state tracking
	vms map[string]*MockVM

	// Behavior configuration
	FailCreate bool
	FailStart  bool
	FailStop   bool
	FailDelete bool

	// Call tracking
	CreateCalls []executor.MicroVMParams
	StartCalls  []string
	StopCalls   []string
	DeleteCalls []string
}

type MockVM struct {
	ID        string
	Name      string
	State     string // PENDING, CREATED, RUNNING, STOPPED, DELETED
	VCPU      int
	MemoryMiB int
}

func NewMockCloudHypervisor() *MockCloudHypervisor {
	return &MockCloudHypervisor{
		vms:         make(map[string]*MockVM),
		CreateCalls: make([]executor.MicroVMParams, 0),
		StartCalls:  make([]string, 0),
		StopCalls:   make([]string, 0),
		DeleteCalls: make([]string, 0),
	}
}

func (m *MockCloudHypervisor) Create(ctx context.Context, params executor.MicroVMParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreateCalls = append(m.CreateCalls, params)

	if m.FailCreate {
		return fmt.Errorf("mock create failure")
	}

	m.vms[params.VMID] = &MockVM{
		ID:        params.VMID,
		Name:      params.Name,
		State:     "CREATED",
		VCPU:      params.VCPU,
		MemoryMiB: params.MemoryMiB,
	}
	return nil
}

func (m *MockCloudHypervisor) Start(ctx context.Context, vmID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StartCalls = append(m.StartCalls, vmID)

	if m.FailStart {
		return fmt.Errorf("mock start failure")
	}

	vm, exists := m.vms[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found", vmID)
	}
	if vm.State == "DELETED" {
		return fmt.Errorf("vm %s is deleted", vmID)
	}
	vm.State = "RUNNING"
	return nil
}

func (m *MockCloudHypervisor) Stop(ctx context.Context, vmID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StopCalls = append(m.StopCalls, vmID)

	if m.FailStop {
		return fmt.Errorf("mock stop failure")
	}

	vm, exists := m.vms[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found", vmID)
	}
	if vm.State == "DELETED" {
		return fmt.Errorf("vm %s is deleted", vmID)
	}
	vm.State = "STOPPED"
	return nil
}

func (m *MockCloudHypervisor) Delete(ctx context.Context, vmID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.DeleteCalls = append(m.DeleteCalls, vmID)

	if m.FailDelete {
		return fmt.Errorf("mock delete failure")
	}

	vm, exists := m.vms[vmID]
	if !exists {
		return nil // Idempotent delete
	}
	vm.State = "DELETED"
	return nil
}

func (m *MockCloudHypervisor) GetProcessID(ctx context.Context, vmID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vm, exists := m.vms[vmID]
	if !exists {
		return 0, fmt.Errorf("vm %s not found", vmID)
	}
	if vm.State == "RUNNING" {
		return 12345, nil // Mock PID
	}
	return 0, fmt.Errorf("vm %s is not running", vmID)
}

func (m *MockCloudHypervisor) GetStatus(ctx context.Context, vmID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vm, exists := m.vms[vmID]
	if !exists {
		return "", fmt.Errorf("vm %s not found", vmID)
	}
	return vm.State, nil
}

func (m *MockCloudHypervisor) GetVM(vmID string) (*MockVM, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	vm, exists := m.vms[vmID]
	return vm, exists
}

func (m *MockCloudHypervisor) VMCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.vms)
}

func (m *MockCloudHypervisor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vms = make(map[string]*MockVM)
	m.CreateCalls = m.CreateCalls[:0]
	m.StartCalls = m.StartCalls[:0]
	m.StopCalls = m.StopCalls[:0]
	m.DeleteCalls = m.DeleteCalls[:0]
	m.FailCreate = false
	m.FailStart = false
	m.FailStop = false
	m.FailDelete = false
}
