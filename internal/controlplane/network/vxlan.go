package network

import (
	"time"
)

// VXLANNetwork represents a VXLAN network in the control plane
type VXLANNetwork struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	SiteID    string    `json:"site_id"`
	Name      string    `json:"name"`
	VNI       int       `json:"vni"`
	CIDR      string    `json:"cidr"`
	Gateway   string    `json:"gateway,omitempty"`
	MTU       int       `json:"mtu"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// VXLANTunnel represents a VXLAN tunnel endpoint on a host
type VXLANTunnel struct {
	ID         string    `json:"id"`
	NetworkID  string    `json:"network_id"`
	HostID     string    `json:"host_id"`
	LocalIP    string    `json:"local_ip"`
	VTEPName   string    `json:"vtep_name"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// VMNetworkAttachment represents a VM's attachment to a VXLAN network
type VMNetworkAttachment struct {
	ID         string    `json:"id"`
	VMID       string    `json:"vm_id"`
	NetworkID  string    `json:"network_id"`
	IPAddress  string    `json:"ip_address,omitempty"`
	MACAddress string    `json:"mac_address,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateVXLANNetworkRequest represents a request to create a VXLAN network
type CreateVXLANNetworkRequest struct {
	Name    string `json:"name"`
	VNI     int    `json:"vni"`
	CIDR    string `json:"cidr"`
	Gateway string `json:"gateway,omitempty"`
	MTU     int    `json:"mtu,omitempty"`
}

// CreateVXLANTunnelRequest represents a request to create a VXLAN tunnel
type CreateVXLANTunnelRequest struct {
	HostID  string `json:"host_id"`
	LocalIP string `json:"local_ip"`
}

// AttachVMToNetworkRequest represents a request to attach a VM to a network
type AttachVMToNetworkRequest struct {
	IPAddress  string `json:"ip_address,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`
}

// VXLANNetworkWithTunnels includes network details with its tunnels
type VXLANNetworkWithTunnels struct {
	VXLANNetwork
	Tunnels []VXLANTunnel `json:"tunnels,omitempty"`
}

// VXLANTunnelStatus represents the status of a VXLAN tunnel
type VXLANTunnelStatus string

const (
	VXLANTunnelStatusPending    VXLANTunnelStatus = "pending"
	VXLANTunnelStatusCreating   VXLANTunnelStatus = "creating"
	VXLANTunnelStatusActive     VXLANTunnelStatus = "active"
	VXLANTunnelStatusFailed     VXLANTunnelStatus = "failed"
	VXLANTunnelStatusDestroying VXLANTunnelStatus = "destroying"
)

// Validate performs basic validation on a VXLANNetwork
func (n *VXLANNetwork) Validate() error {
	if n.Name == "" {
		return &ValidationError{Field: "Name", Message: "is required"}
	}
	if n.VNI < 1 || n.VNI > 16777215 {
		return &ValidationError{Field: "VNI", Message: "must be between 1 and 16777215"}
	}
	if n.CIDR == "" {
		return &ValidationError{Field: "CIDR", Message: "is required"}
	}
	if n.MTU == 0 {
		n.MTU = 1450 // Default MTU for VXLAN
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + " " + e.Message
}
