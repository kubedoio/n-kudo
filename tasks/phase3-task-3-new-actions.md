# Phase 3 Task 3: New Action Types

## Task Description
Add new action types to the executor: pause/resume, snapshot, and command execution.

## New Actions

### 1. MicroVMPause
**Purpose:** Pause a running VM (freeze state)

**API:**
```go
type MicroVMPauseAction struct {
    VMID string `json:"vm_id"`
}
```

**Behavior:**
- Send SIGSTOP to the cloud-hypervisor process
- Update VM state to "PAUSED"
- Preserve memory state

**Implementation in executor:**
```go
func (e *Executor) executePause(ctx context.Context, params MicroVMParams) error {
    // Find CH process by VM ID
    pid, err := e.provider.GetProcessID(ctx, params.VMID)
    if err != nil {
        return err
    }
    
    // Send SIGSTOP
    process, err := os.FindProcess(pid)
    if err != nil {
        return err
    }
    
    return process.Signal(syscall.SIGSTOP)
}
```

### 2. MicroVMResume
**Purpose:** Resume a paused VM

**API:**
```go
type MicroVMResumeAction struct {
    VMID string `json:"vm_id"`
}
```

**Behavior:**
- Send SIGCONT to the cloud-hypervisor process
- Update VM state to "RUNNING"

**Implementation:**
```go
func (e *Executor) executeResume(ctx context.Context, params MicroVMParams) error {
    pid, err := e.provider.GetProcessID(ctx, params.VMID)
    if err != nil {
        return err
    }
    
    process, err := os.FindProcess(pid)
    if err != nil {
        return err
    }
    
    return process.Signal(syscall.SIGCONT)
}
```

### 3. MicroVMSnapshot
**Purpose:** Create a VM snapshot for backup

**API:**
```go
type MicroVMSnapshotAction struct {
    VMID         string `json:"vm_id"`
    SnapshotName string `json:"snapshot_name"`
}
```

**Behavior:**
- Pause VM briefly
- Copy disk image to snapshot directory
- Save VM config
- Resume VM
- Store snapshot metadata

**Implementation:**
```go
func (e *Executor) executeSnapshot(ctx context.Context, params SnapshotParams) error {
    // Get VM info
    vm, err := e.state.GetMicroVM(params.VMID)
    if err != nil {
        return err
    }
    
    // Create snapshot directory
    snapshotDir := filepath.Join(e.snapshotDir, params.SnapshotName)
    if err := os.MkdirAll(snapshotDir, 0755); err != nil {
        return err
    }
    
    // Pause VM briefly
    if err := e.executePause(ctx, MicroVMParams{VMID: params.VMID}); err != nil {
        return err
    }
    defer e.executeResume(ctx, MicroVMParams{VMID: params.VMID})
    
    // Copy disk
    srcDisk := filepath.Join(e.runtimeDir, params.VMID, "disk.img")
    dstDisk := filepath.Join(snapshotDir, "disk.img")
    if err := copyFile(srcDisk, dstDisk); err != nil {
        return err
    }
    
    // Save config
    configPath := filepath.Join(snapshotDir, "config.json")
    config := map[string]interface{}{
        "vm_id": params.VMID,
        "name": vm.Name,
        "vcpu_count": vm.VCPUs,
        "memory_mib": vm.Memory,
        "created_at": time.Now().UTC(),
    }
    
    data, _ := json.MarshalIndent(config, "", "  ")
    return os.WriteFile(configPath, data, 0644)
}
```

### 4. CommandExecute
**Purpose:** Execute arbitrary commands on the host

**API:**
```go
type CommandExecuteAction struct {
    Command string   `json:"command"`
    Args    []string `json:"args"`
    Timeout int      `json:"timeout_seconds"`
    Dir     string   `json:"working_dir,omitempty"`
}
```

**Behavior:**
- Execute command with timeout
- Capture stdout/stderr
- Return exit code and output

**Implementation:**
```go
func (e *Executor) executeCommand(ctx context.Context, params CommandParams) CommandResult {
    timeout := time.Duration(params.Timeout) * time.Second
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    cmd := exec.CommandContext(ctx, params.Command, params.Args...)
    cmd.Dir = params.Dir
    
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    err := cmd.Run()
    
    exitCode := 0
    if err != nil {
        if exitError, ok := err.(*exec.ExitError); ok {
            exitCode = exitError.ExitCode()
        } else {
            exitCode = -1
        }
    }
    
    return CommandResult{
        ExitCode: exitCode,
        Stdout:   stdout.String(),
        Stderr:   stderr.String(),
    }
}
```

## Files to Modify

### Modified Files

**`internal/edge/executor/executor.go`**
- Add new action types to switch statement
- Implement handler methods

**`internal/edge/executor/types.go`**
- Add new action parameter types

### New Files

**`internal/edge/executor/pause.go`**
- Pause/resume implementation

**`internal/edge/executor/snapshot.go`**
- Snapshot implementation

**`internal/edge/executor/command.go`**
- Command execution implementation

## Control Plane Changes

Update `internal/controlplane/api/server.go` to support new action types in plan actions.

## API Changes

Add to OpenAPI spec:
```yaml
PlanAction:
  operation:
    enum: [CREATE, START, STOP, DELETE, PAUSE, RESUME, SNAPSHOT, EXECUTE]
```

## Testing

Add tests in `internal/edge/executor/`:
- `pause_test.go` - Test pause/resume
- `snapshot_test.go` - Test snapshot creation
- `command_test.go` - Test command execution

## Definition of Done
- [ ] All 4 new actions implemented
- [ ] Actions integrate with existing executor
- [ ] Proper error handling
- [ ] Tests pass

## Estimated Effort
8-10 hours
