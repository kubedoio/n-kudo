package executor

import (
	"context"
	"encoding/json"
	"time"
)

type ActionType string

const (
	ActionMicroVMCreate ActionType = "MicroVMCreate"
	ActionMicroVMStart  ActionType = "MicroVMStart"
	ActionMicroVMStop   ActionType = "MicroVMStop"
	ActionMicroVMDelete ActionType = "MicroVMDelete"
)

type Plan struct {
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

type MicroVMParams struct {
	VMID       string   `json:"vm_id"`
	Name       string   `json:"name"`
	KernelPath string   `json:"kernel_path"`
	RootfsPath string   `json:"rootfs_path"`
	TapIface   string   `json:"tap_iface,omitempty"`
	VCPU       int      `json:"vcpu"`
	MemoryMiB  int      `json:"memory_mib"`
	ExtraArgs  []string `json:"extra_args,omitempty"`
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
	ExecutionID string         `json:"execution_id"`
	Results     []ActionResult `json:"results"`
}

type MicroVMProvider interface {
	Create(context.Context, MicroVMParams) error
	Start(context.Context, string) error
	Stop(context.Context, string) error
	Delete(context.Context, string) error
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
