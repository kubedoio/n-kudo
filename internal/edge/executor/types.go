package executor

import (
	"context"
	"encoding/json"
	"time"
)

type ActionType string

const (
	ActionMicroVMCreate    ActionType = "MicroVMCreate"
	ActionMicroVMStart     ActionType = "MicroVMStart"
	ActionMicroVMStop      ActionType = "MicroVMStop"
	ActionMicroVMDelete    ActionType = "MicroVMDelete"
	ActionMicroVMPause     ActionType = "MicroVMPause"
	ActionMicroVMResume    ActionType = "MicroVMResume"
	ActionMicroVMSnapshot  ActionType = "MicroVMSnapshot"
	ActionCommandExecute   ActionType = "CommandExecute"
)

type Plan struct {
	PlanID      string   `json:"plan_id,omitempty"`
	ExecutionID string   `json:"execution_id"`
	Actions     []Action `json:"actions"`
}

type Action struct {
	ActionID      string          `json:"action_id"`
	Type          ActionType      `json:"type"`
	Params        json.RawMessage `json:"params"`
	DesiredState  string          `json:"desired_state,omitempty"`
	TimeoutSecond int             `json:"timeout"`
}

// IPConfig represents static IP configuration for a network interface
type IPConfig struct {
	Address string `json:"address"` // CIDR notation, e.g., "192.168.1.100/24"
	Gateway string `json:"gateway,omitempty"` // e.g., "192.168.1.1"
}

// NetworkInterface represents a network interface configuration for a VM
type NetworkInterface struct {
	ID       string    `json:"id"`        // e.g., "eth0", "eth1"
	TapName  string    `json:"tap_name"`  // TAP device name
	MacAddr  string    `json:"mac,omitempty"` // Optional MAC address
	Bridge   string    `json:"bridge"`    // Bridge to attach to
	IPConfig *IPConfig `json:"ip_config,omitempty"` // Optional static IP
}

type MicroVMParams struct {
	VMID       string             `json:"vm_id"`
	Name       string             `json:"name"`
	KernelPath string             `json:"kernel_path"`
	RootfsPath string             `json:"rootfs_path"`
	TapIface   string             `json:"tap_iface,omitempty"` // Deprecated: use Networks instead
	Networks   []NetworkInterface `json:"networks,omitempty"`  // Multiple network interfaces
	VCPU       int                `json:"vcpu"`
	MemoryMiB  int                `json:"memory_mib"`
	ExtraArgs  []string           `json:"extra_args,omitempty"`
}

// GetNetworks returns the list of network interfaces for the VM.
// If Networks is empty, it falls back to the deprecated TapIface field.
func (p MicroVMParams) GetNetworks() []NetworkInterface {
	if len(p.Networks) > 0 {
		return p.Networks
	}
	// Backward compatibility: create a single network from TapIface
	if p.TapIface != "" {
		return []NetworkInterface{
			{
				ID:      "eth0",
				TapName: p.TapIface,
			},
		}
	}
	return nil
}

// PrimaryNetwork returns the primary (first) network interface, or nil if none.
func (p MicroVMParams) PrimaryNetwork() *NetworkInterface {
	networks := p.GetNetworks()
	if len(networks) > 0 {
		return &networks[0]
	}
	return nil
}

type PauseParams struct {
	VMID string `json:"vm_id"`
}

type ResumeParams struct {
	VMID string `json:"vm_id"`
}

type SnapshotParams struct {
	VMID         string `json:"vm_id"`
	SnapshotName string `json:"snapshot_name"`
}

type CommandParams struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Timeout int      `json:"timeout_seconds"`
	Dir     string   `json:"working_dir,omitempty"`
}

type ActionResult struct {
	ExecutionID string    `json:"execution_id"`
	ActionID    string    `json:"action_id"`
	OK          bool      `json:"ok"`
	ErrorCode   string    `json:"error_code,omitempty"`
	Message     string    `json:"message"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
}

type PlanResult struct {
	PlanID      string         `json:"plan_id,omitempty"`
	ExecutionID string         `json:"execution_id"`
	Results     []ActionResult `json:"results"`
}

type MicroVMProvider interface {
	Create(context.Context, MicroVMParams) error
	Start(context.Context, string) error
	Stop(context.Context, string) error
	Delete(context.Context, string) error
	GetProcessID(context.Context, string) (int, error)
}

type LogEntry struct {
	ExecutionID string `json:"execution_id"`
	ActionID    string `json:"action_id,omitempty"`
	Level       string `json:"level"`
	Message     string `json:"message"`
}

type LogSink interface {
	Write(context.Context, LogEntry)
}
