package cloudhypervisor

import (
	"context"
	"errors"
	"fmt"
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

// NetworkInterface represents a network interface for a VM
type NetworkInterface struct {
	ID       string `json:"id" yaml:"id"`               // Interface ID, e.g., "eth0"
	TapName  string `json:"tap_name" yaml:"tap_name"`   // TAP device name
	MacAddr  string `json:"mac,omitempty" yaml:"mac,omitempty"` // MAC address
	Bridge   string `json:"bridge" yaml:"bridge"`       // Bridge to attach to
	IPAddr   string `json:"ip,omitempty" yaml:"ip,omitempty"`   // IP address in CIDR notation
}

// VMSpec is the provider-facing schema for microVM lifecycle operations.
type VMSpec struct {
	Name              string             `json:"name" yaml:"name"`
	VCPU              int                `json:"vcpu" yaml:"vcpu"`
	MemMB             int                `json:"mem_mb" yaml:"mem_mb"`
	DiskPath          string             `json:"disk_path" yaml:"disk_path"`
	CloudInitISOPath  string             `json:"cloud_init_iso_path" yaml:"cloud_init_iso_path"`
	TapName           string             `json:"tap_name" yaml:"tap_name"` // Deprecated: use Networks instead
	BridgeName        string             `json:"bridge_name" yaml:"bridge_name"`
	MACAddress        string             `json:"mac,omitempty" yaml:"mac,omitempty"`
	Hostname          string             `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	SSHAuthorizedKeys []string           `json:"ssh_authorized_keys,omitempty" yaml:"ssh_authorized_keys,omitempty"`
	UserData          string             `json:"user_data,omitempty" yaml:"user_data,omitempty"`
	DiskSizeMB        int                `json:"disk_size_mb,omitempty" yaml:"disk_size_mb,omitempty"`
	Networks          []NetworkInterface `json:"networks,omitempty" yaml:"networks,omitempty"` // Multiple network interfaces
}

func (s *VMSpec) normalize() {
	s.Name = strings.TrimSpace(s.Name)
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

	// Normalize network interfaces
	for i := range s.Networks {
		s.Networks[i].ID = strings.TrimSpace(s.Networks[i].ID)
		s.Networks[i].TapName = strings.TrimSpace(s.Networks[i].TapName)
		s.Networks[i].MacAddr = strings.TrimSpace(strings.ToLower(s.Networks[i].MacAddr))
		s.Networks[i].Bridge = strings.TrimSpace(s.Networks[i].Bridge)
		s.Networks[i].IPAddr = strings.TrimSpace(s.Networks[i].IPAddr)
		if s.Networks[i].ID == "" {
			s.Networks[i].ID = fmt.Sprintf("eth%d", i)
		}
	}
}

// GetNetworks returns the list of network interfaces for the VM.
// If Networks is empty, it falls back to the deprecated TapName/BridgeName fields.
func (s VMSpec) GetNetworks() []NetworkInterface {
	if len(s.Networks) > 0 {
		return s.Networks
	}
	// Backward compatibility: create a single network from TapName
	if s.TapName != "" {
		bridge := s.BridgeName
		if bridge == "" {
			bridge = "br0"
		}
		return []NetworkInterface{
			{
				ID:      "eth0",
				TapName: s.TapName,
				MacAddr: s.MACAddress,
				Bridge:  bridge,
			},
		}
	}
	return nil
}

// PrimaryNetwork returns the primary (first) network interface, or nil if none.
func (s VMSpec) PrimaryNetwork() *NetworkInterface {
	networks := s.GetNetworks()
	if len(networks) > 0 {
		return &networks[0]
	}
	return nil
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

	// Validate network configuration (either Networks or deprecated TapName)
	networks := s.GetNetworks()
	if len(networks) == 0 {
		return errors.New("at least one network interface is required")
	}

	// Validate each network interface
	for i, iface := range networks {
		if iface.TapName == "" {
			return fmt.Errorf("network[%d]: tap_name is required", i)
		}
		if iface.Bridge == "" {
			return fmt.Errorf("network[%d]: bridge is required", i)
		}
		if iface.MacAddr != "" {
			if _, err := net.ParseMAC(iface.MacAddr); err != nil {
				return fmt.Errorf("network[%d]: invalid mac: %w", i, err)
			}
		}
		if iface.IPAddr != "" {
			if _, _, err := net.ParseCIDR(iface.IPAddr); err != nil {
				return fmt.Errorf("network[%d]: invalid ip: %w", i, err)
			}
		}
	}

	// Validate deprecated MACAddress field if specified
	if s.MACAddress != "" {
		if _, err := net.ParseMAC(s.MACAddress); err != nil {
			return fmt.Errorf("invalid mac: %w", err)
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
