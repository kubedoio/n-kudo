package vxlan

import (
	"time"
)

// Default VXLAN constants
const (
	DefaultVXLANPort = 4789
	DefaultMTU       = 1450 // Account for VXLAN overhead (50 bytes)
	MinVNI           = 1
	MaxVNI           = 16777215 // 24-bit VNI
)

// TunnelStatus represents the current status of a VXLAN tunnel
type TunnelStatus string

const (
	TunnelStatusPending    TunnelStatus = "pending"
	TunnelStatusCreating   TunnelStatus = "creating"
	TunnelStatusActive     TunnelStatus = "active"
	TunnelStatusFailed     TunnelStatus = "failed"
	TunnelStatusTearingDown TunnelStatus = "tearing_down"
	TunnelStatusDestroyed  TunnelStatus = "destroyed"
)

// VXLANConfig holds configuration for creating a VXLAN tunnel endpoint
type VXLANConfig struct {
	VNI         int    `json:"vni"`          // VXLAN Network Identifier (1-16777215)
	VTEPName    string `json:"vtep_name"`    // e.g., "vxlan100"
	LocalIP     string `json:"local_ip"`     // Local VTEP IP
	RemoteIP    string `json:"remote_ip"`    // Remote VTEP IP (unicast) or multicast group
	Port        int    `json:"port"`         // UDP port (default 4789)
	MTU         int    `json:"mtu"`          // MTU (default 1450 to account for VXLAN overhead)
	ParentIface string `json:"parent_iface"` // Underlay interface
}

// Validate checks if the VXLAN configuration is valid
func (c *VXLANConfig) Validate() error {
	if c.VNI < MinVNI || c.VNI > MaxVNI {
		return &ValidationError{Field: "VNI", Message: "must be between 1 and 16777215"}
	}
	if c.VTEPName == "" {
		return &ValidationError{Field: "VTEPName", Message: "is required"}
	}
	if c.LocalIP == "" {
		return &ValidationError{Field: "LocalIP", Message: "is required"}
	}
	if c.Port == 0 {
		c.Port = DefaultVXLANPort
	}
	if c.MTU == 0 {
		c.MTU = DefaultMTU
	}
	return nil
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + " " + e.Message
}

// VXLANTunnel represents a configured VXLAN tunnel endpoint
type VXLANTunnel struct {
	Config      VXLANConfig  `json:"config"`
	BridgeName  string       `json:"bridge_name"`  // Bridge to attach VTEP to
	Status      TunnelStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	LastError   string       `json:"last_error,omitempty"`
}

// VXLANTunnelStatus provides detailed status of a VXLAN tunnel
type VXLANTunnelStatus struct {
	VNI         int          `json:"vni"`
	VTEPName    string       `json:"vtep_name"`
	LocalIP     string       `json:"local_ip"`
	RemoteIP    string       `json:"remote_ip,omitempty"`
	Status      TunnelStatus `json:"status"`
	BridgeName  string       `json:"bridge_name,omitempty"`
	MTU         int          `json:"mtu"`
	CreatedAt   time.Time    `json:"created_at"`
	LastError   string       `json:"last_error,omitempty"`
}

// FDBEntry represents a forwarding database entry for VXLAN
type FDBEntry struct {
	MAC      string `json:"mac"`
	RemoteIP string `json:"remote_ip"`
	VTEPName string `json:"vtep_name"`
	State    string `json:"state"` // "permanent", "dynamic", etc.
}

// BridgeInfo holds information about a Linux bridge
type BridgeInfo struct {
	Name      string   `json:"name"`
	Interfaces []string `json:"interfaces"`
	MACAddr   string   `json:"mac_addr"`
	MTU       int      `json:"mtu"`
}
