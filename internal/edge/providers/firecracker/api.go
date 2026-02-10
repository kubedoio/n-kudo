package firecracker

import (
	"context"
	"errors"
	"net"
	"strings"
)

var ErrVMNotFound = errors.New("vm not found")

type VMStatus string

const (
	VMStatusCreated VMStatus = "created"
	VMStatusRunning VMStatus = "running"
	VMStatusStopped VMStatus = "stopped"
	VMStatusDeleted VMStatus = "deleted"
)

// VMSpec is the provider-facing schema for microVM lifecycle operations.
type VMSpec struct {
	Name              string   `json:"name" yaml:"name"`
	VCPU              int      `json:"vcpu" yaml:"vcpu"`
	MemMB             int      `json:"mem_mb" yaml:"mem_mb"`
	KernelPath        string   `json:"kernel_path" yaml:"kernel_path"`
	DiskPath          string   `json:"disk_path" yaml:"disk_path"`
	CloudInitISOPath  string   `json:"cloud_init_iso_path" yaml:"cloud_init_iso_path"`
	TapName           string   `json:"tap_name" yaml:"tap_name"`
	BridgeName        string   `json:"bridge_name" yaml:"bridge_name"`
	MACAddress        string   `json:"mac,omitempty" yaml:"mac,omitempty"`
	Hostname          string   `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	SSHAuthorizedKeys []string `json:"ssh_authorized_keys,omitempty" yaml:"ssh_authorized_keys,omitempty"`
	UserData          string   `json:"user_data,omitempty" yaml:"user_data,omitempty"`
	DiskSizeMB        int      `json:"disk_size_mb,omitempty" yaml:"disk_size_mb,omitempty"`
	KernelArgs        string   `json:"kernel_args,omitempty" yaml:"kernel_args,omitempty"`
}

func (s *VMSpec) normalize() {
	s.Name = strings.TrimSpace(s.Name)
	s.KernelPath = strings.TrimSpace(s.KernelPath)
	s.DiskPath = strings.TrimSpace(s.DiskPath)
	s.CloudInitISOPath = strings.TrimSpace(s.CloudInitISOPath)
	s.TapName = strings.TrimSpace(s.TapName)
	s.BridgeName = strings.TrimSpace(s.BridgeName)
	s.MACAddress = strings.TrimSpace(strings.ToLower(s.MACAddress))
	s.Hostname = strings.TrimSpace(s.Hostname)
	if s.Hostname == "" {
		s.Hostname = s.Name
	}
	if s.DiskSizeMB <= 0 {
		s.DiskSizeMB = 10 * 1024
	}
	if s.KernelArgs == "" {
		s.KernelArgs = "console=ttyS0 reboot=k panic=1 pci=off"
	}
}

func (s VMSpec) Validate() error {
	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.VCPU <= 0 {
		return errors.New("vcpu must be > 0")
	}
	if s.MemMB <= 0 {
		return errors.New("mem_mb must be > 0")
	}
	if s.KernelPath == "" {
		return errors.New("kernel_path is required")
	}
	if s.TapName == "" {
		return errors.New("tap_name is required")
	}
	if s.BridgeName == "" {
		return errors.New("bridge_name is required")
	}
	if s.MACAddress != "" {
		if _, err := net.ParseMAC(s.MACAddress); err != nil {
			return errors.New("invalid mac: " + err.Error())
		}
	}
	return nil
}

// VMProvider is the clean API surface expected by the agent.
type VMProvider interface {
	CreateVM(ctx context.Context, spec VMSpec) (string, error)
	StartVM(ctx context.Context, vmID string) error
	StopVM(ctx context.Context, vmID string) error
	DeleteVM(ctx context.Context, vmID string) error
	GetVMStatus(ctx context.Context, vmID string) (VMStatus, error)
	CollectConsoleLog(ctx context.Context, vmID string) ([]byte, error)
}
