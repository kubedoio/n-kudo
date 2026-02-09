# Phase 2 Task 4: Edge Agent Tests

## Task Description
Implement comprehensive tests for the edge agent including mocks for external dependencies.

## Prerequisites
- Edge agent code complete
- Cloud Hypervisor provider implemented

## Acceptance Criteria
- [ ] Mock Cloud Hypervisor for testing
- [ ] Mock NetBird for testing
- [ ] Unit tests for executor action handlers
- [ ] Integration test for enrollment + plan execution
- [ ] Tests pass with `go test ./internal/edge/...`

## Mock Implementations

### 1. Mock Cloud Hypervisor

Create `internal/edge/providers/cloudhypervisor/mock_provider.go`:
```go
package cloudhypervisor

import "context"

type MockProvider struct {
    VMs map[string]*MockVM
}

type MockVM struct {
    ID     string
    Name   string
    State  string // CREATING, RUNNING, STOPPED, DELETED
    Config VMConfig
}

type VMConfig struct {
    VCPUs   int
    Memory  int64
    Kernel  string
    Rootfs  string
}

func NewMockProvider() *MockProvider {
    return &MockProvider{
        VMs: make(map[string]*MockVM),
    }
}

func (m *MockProvider) CreateVM(ctx context.Context, id, name string, vcpus int, memory int64, kernel, rootfs string) error {
    m.VMs[id] = &MockVM{
        ID:     id,
        Name:   name,
        State:  "CREATING",
        Config: VMConfig{VCPUs: vcpus, Memory: memory, Kernel: kernel, Rootfs: rootfs},
    }
    // Simulate async creation
    go func() {
        m.VMs[id].State = "RUNNING"
    }()
    return nil
}

func (m *MockProvider) StartVM(ctx context.Context, id string) error {
    if vm, ok := m.VMs[id]; ok {
        vm.State = "RUNNING"
        return nil
    }
    return fmt.Errorf("VM not found: %s", id)
}

func (m *MockProvider) StopVM(ctx context.Context, id string) error {
    if vm, ok := m.VMs[id]; ok {
        vm.State = "STOPPED"
        return nil
    }
    return fmt.Errorf("VM not found: %s", id)
}

func (m *MockProvider) DeleteVM(ctx context.Context, id string) error {
    if _, ok := m.VMs[id]; ok {
        delete(m.VMs, id)
        return nil
    }
    return fmt.Errorf("VM not found: %s", id)
}

func (m *MockProvider) GetVMStatus(ctx context.Context, id string) (string, error) {
    if vm, ok := m.VMs[id]; ok {
        return vm.State, nil
    }
    return "", fmt.Errorf("VM not found: %s", id)
}

func (m *MockProvider) ListVMs(ctx context.Context) ([]VMInfo, error) {
    var vms []VMInfo
    for _, vm := range m.VMs {
        vms = append(vms, VMInfo{
            ID:    vm.ID,
            Name:  vm.Name,
            State: vm.State,
        })
    }
    return vms, nil
}
```

### 2. Mock NetBird

Create `internal/edge/netbird/mock_netbird.go`:
```go
package netbird

type MockClient struct {
    StatusResponse string
    JoinCalled     bool
    LeaveCalled    bool
}

func NewMockClient() *MockClient {
    return &MockClient{
        StatusResponse: `{
            "daemon": {"state": "running"},
            "signal": {"state": "connected"},
            "management": {"state": "connected"}
        }`,
    }
}

func (m *MockClient) Status() (*Status, error) {
    return ParseStatus([]byte(m.StatusResponse))
}

func (m *MockClient) Join() error {
    m.JoinCalled = true
    return nil
}

func (m *MockClient) Leave() error {
    m.LeaveCalled = true
    return nil
}
```

## Unit Tests for Executor

### `internal/edge/executor/executor_test.go` (Enhance)

Add tests for action handlers:
```go
func TestExecutor_MicroVMCreate(t *testing.T) {
    mockProvider := cloudhypervisor.NewMockProvider()
    exec := New(mockProvider, nil)
    
    action := Action{
        ID:   "action-1",
        Type: "MicroVMCreate",
        Params: map[string]interface{}{
            "vm_id":      "vm-1",
            "name":       "test-vm",
            "vcpu":       2,
            "memory_mib": 512,
        },
    }
    
    result := exec.Execute(context.Background(), action)
    
    if result.Error != "" {
        t.Errorf("Expected no error, got: %s", result.Error)
    }
    
    // Verify VM created
    vm, err := mockProvider.GetVMStatus(context.Background(), "vm-1")
    if err != nil {
        t.Errorf("VM not found: %v", err)
    }
    if vm != "RUNNING" {
        t.Errorf("Expected RUNNING, got: %s", vm)
    }
}

func TestExecutor_MicroVMStart(t *testing.T) {
    mockProvider := cloudhypervisor.NewMockProvider()
    // Pre-create a stopped VM
    mockProvider.CreateVM(context.Background(), "vm-1", "test-vm", 2, 512, "", "")
    mockProvider.StopVM(context.Background(), "vm-1")
    
    exec := New(mockProvider, nil)
    
    action := Action{
        ID:   "action-1",
        Type: "MicroVMStart",
        Params: map[string]interface{}{
            "vm_id": "vm-1",
        },
    }
    
    result := exec.Execute(context.Background(), action)
    
    if result.Error != "" {
        t.Errorf("Expected no error, got: %s", result.Error)
    }
    
    vm, _ := mockProvider.GetVMStatus(context.Background(), "vm-1")
    if vm != "RUNNING" {
        t.Errorf("Expected RUNNING, got: %s", vm)
    }
}

func TestExecutor_Idempotency(t *testing.T) {
    mockProvider := cloudhypervisor.NewMockProvider()
    exec := New(mockProvider, nil)
    
    action := Action{
        ID:   "action-1",
        Type: "MicroVMCreate",
        Params: map[string]interface{}{
            "vm_id":      "vm-1",
            "name":       "test-vm",
            "vcpu":       2,
            "memory_mib": 512,
        },
    }
    
    // Execute twice with same action ID
    result1 := exec.Execute(context.Background(), action)
    result2 := exec.Execute(context.Background(), action)
    
    if result1.Error != "" || result2.Error != "" {
        t.Error("Expected no errors")
    }
    
    // Should only have one VM
    vms, _ := mockProvider.ListVMs(context.Background())
    if len(vms) != 1 {
        t.Errorf("Expected 1 VM, got: %d", len(vms))
    }
}
```

## Integration Test

### `tests/integration/edge_enrollment_test.go`

```go
func TestEdgeAgent_EnrollmentAndPlanExecution(t *testing.T) {
    // Setup test server and database
    srv, repo := setupTestServer(t)
    defer srv.Close()
    
    // 1. Create tenant and site
    tenant := createTestTenant(t, repo)
    site := createTestSite(t, repo, tenant.ID)
    
    // 2. Issue enrollment token
    token := createTestEnrollmentToken(t, repo, site.ID)
    
    // 3. Simulate agent enrollment
    agent, cert := enrollAgent(t, srv.URL, token)
    
    // 4. Verify agent created
    if agent.ID == "" {
        t.Fatal("Agent not created")
    }
    
    // 5. Simulate heartbeat
    sendHeartbeat(t, srv.URL, cert, agent.ID)
    
    // 6. Apply plan
    plan := applyPlan(t, srv.URL, tenant.APIKey, site.ID, []PlanAction{
        {Type: "MicroVMCreate", Params: map[string]interface{}{
            "vm_id": "vm-1", "name": "test-vm", "vcpu": 2, "memory_mib": 512,
        }},
    })
    
    // 7. Simulate agent fetching pending plans
    plans := fetchPendingPlans(t, srv.URL, cert, agent.ID)
    if len(plans) != 1 {
        t.Fatalf("Expected 1 pending plan, got: %d", len(plans))
    }
    
    // 8. Simulate agent executing plan
    reportResult(t, srv.URL, cert, agent.ID, plans[0].ID, "SUCCEEDED")
    
    // 9. Verify execution status
    execution := getExecution(t, repo, plans[0].ID)
    if execution.State != "SUCCEEDED" {
        t.Errorf("Expected SUCCEEDED, got: %s", execution.State)
    }
}
```

## Test Coverage Goals

| Package | Target |
|---------|--------|
| `executor` | >80% |
| `enroll` | >80% |
| `cloudhypervisor` | >70% |
| `netbird` | >70% |

## Running Tests

```bash
cd /srv/data01/kubedo/n-kudo

# Run all edge tests
go test ./internal/edge/... -v

# Run with coverage
go test ./internal/edge/... -cover

# Run specific package
go test ./internal/edge/executor/... -v
```

## Definition of Done
- [ ] Mock Cloud Hypervisor implemented
- [ ] Mock NetBird implemented
- [ ] Executor action handler tests
- [ ] Integration test for full flow
- [ ] Coverage goals met
- [ ] All tests pass

## Estimated Effort
6-8 hours
