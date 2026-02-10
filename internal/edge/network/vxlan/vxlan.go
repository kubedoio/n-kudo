package vxlan

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
)

// CreateVTEP creates a VXLAN tunnel endpoint interface
func CreateVTEP(cfg VXLANConfig) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid VXLAN config: %w", err)
	}

	// Check if interface already exists
	_, err := netlink.LinkByName(cfg.VTEPName)
	if err == nil {
		return &VTEPExistsError{Name: cfg.VTEPName}
	}

	// Parse local IP
	localIP := net.ParseIP(cfg.LocalIP)
	if localIP == nil {
		return fmt.Errorf("invalid local IP: %s", cfg.LocalIP)
	}

	// Create VXLAN link attributes
	attrs := netlink.LinkAttrs{
		Name: cfg.VTEPName,
		MTU:  cfg.MTU,
	}

	// Create VXLAN interface
	vxlan := &netlink.Vxlan{
		LinkAttrs: attrs,
		VxlanId:   cfg.VNI,
		Port:      cfg.Port,
	}

	// Set local VTEP IP
	if localIP.To4() != nil {
		vxlan.SrcAddr = localIP.To4()
	} else {
		vxlan.SrcAddr = localIP
	}

	// Set remote IP if specified (unicast mode)
	if cfg.RemoteIP != "" {
		remoteIP := net.ParseIP(cfg.RemoteIP)
		if remoteIP == nil {
			return fmt.Errorf("invalid remote IP: %s", cfg.RemoteIP)
		}
		if remoteIP.To4() != nil {
			vxlan.Group = remoteIP.To4()
		} else {
			vxlan.Group = remoteIP
		}
		// Enable learning for unicast mode
		vxlan.Learning = true
	} else {
		// Multicast mode - learning is enabled by default
		vxlan.Learning = true
	}

	// Add the link
	if err := netlink.LinkAdd(vxlan); err != nil {
		return fmt.Errorf("failed to create VXLAN interface: %w", err)
	}

	// Bring the interface up
	if err := netlink.LinkSetUp(vxlan); err != nil {
		// Try to clean up on failure
		_ = netlink.LinkDel(vxlan)
		return fmt.Errorf("failed to bring up VXLAN interface: %w", err)
	}

	return nil
}

// DeleteVTEP removes a VXLAN interface
func DeleteVTEP(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return &VTEPNotFoundError{Name: name}
		}
		return fmt.Errorf("failed to find VTEP: %w", err)
	}

	// Verify it's a VXLAN interface
	if _, ok := link.(*netlink.Vxlan); !ok {
		return fmt.Errorf("interface %s is not a VXLAN interface", name)
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete VTEP: %w", err)
	}

	return nil
}

// GetVTEP returns information about a VXLAN interface
func GetVTEP(name string) (*netlink.Vxlan, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return nil, &VTEPNotFoundError{Name: name}
		}
		return nil, fmt.Errorf("failed to find VTEP: %w", err)
	}

	vxlan, ok := link.(*netlink.Vxlan)
	if !ok {
		return nil, fmt.Errorf("interface %s is not a VXLAN interface", name)
	}

	return vxlan, nil
}

// AddFDBEntry adds a forwarding database entry for unicast VXLAN
func AddFDBEntry(vtepName string, remoteIP string, mac string) error {
	// Validate MAC address
	_, err := net.ParseMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %w", err)
	}

	// Validate remote IP
	ip := net.ParseIP(remoteIP)
	if ip == nil {
		return fmt.Errorf("invalid remote IP: %s", remoteIP)
	}
	_ = ip // Used for validation

	// Verify the VTEP interface exists
	_, err = netlink.LinkByName(vtepName)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return &VTEPNotFoundError{Name: vtepName}
		}
		return fmt.Errorf("failed to find VTEP: %w", err)
	}

	// Use bridge fdb command to add the entry
	// This is more reliable than netlink for FDB entries
	cmd := exec.Command("bridge", "fdb", "add", mac, "dev", vtepName, "dst", remoteIP)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add FDB entry: %w, output: %s", err, string(output))
	}

	return nil
}

// DeleteFDBEntry removes a forwarding database entry
func DeleteFDBEntry(vtepName string, mac string) error {
	// Validate MAC address
	_, err := net.ParseMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %w", err)
	}

	cmd := exec.Command("bridge", "fdb", "del", mac, "dev", vtepName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete FDB entry: %w, output: %s", err, string(output))
	}

	return nil
}

// ListFDBEntries lists all FDB entries for a VTEP
func ListFDBEntries(vtepName string) ([]FDBEntry, error) {
	cmd := exec.Command("bridge", "fdb", "show", "dev", vtepName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list FDB entries: %w", err)
	}

	var entries []FDBEntry
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse FDB entry line
		// Format: MAC dev VTEP_NAME dst REMOTE_IP [permanent|dynamic]
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		entry := FDBEntry{
			MAC:      fields[0],
			VTEPName: vtepName,
		}

		// Look for dst keyword
		for i := 0; i < len(fields)-1; i++ {
			if fields[i] == "dst" {
				entry.RemoteIP = fields[i+1]
			}
		}

		// Get state (last field)
		if len(fields) > 0 {
			lastField := fields[len(fields)-1]
			if lastField == "permanent" || lastField == "dynamic" || lastField == "static" {
				entry.State = lastField
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// AttachToBridge attaches a VTEP to a Linux bridge
func AttachToBridge(vtepName, bridgeName string) error {
	// Get the VTEP interface
	vtep, err := netlink.LinkByName(vtepName)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return &VTEPNotFoundError{Name: vtepName}
		}
		return fmt.Errorf("failed to find VTEP: %w", err)
	}

	// Get the bridge interface
	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return &BridgeNotFoundError{Name: bridgeName}
		}
		return fmt.Errorf("failed to find bridge: %w", err)
	}

	// Check if bridge is actually a bridge
	if _, ok := bridge.(*netlink.Bridge); !ok {
		return fmt.Errorf("interface %s is not a bridge", bridgeName)
	}

	// Set the bridge as master for the VTEP
	if err := netlink.LinkSetMaster(vtep, bridge); err != nil {
		return fmt.Errorf("failed to attach VTEP to bridge: %w", err)
	}

	return nil
}

// DetachFromBridge detaches a VTEP from its bridge
func DetachFromBridge(vtepName string) error {
	vtep, err := netlink.LinkByName(vtepName)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return &VTEPNotFoundError{Name: vtepName}
		}
		return fmt.Errorf("failed to find VTEP: %w", err)
	}

	if err := netlink.LinkSetNoMaster(vtep); err != nil {
		return fmt.Errorf("failed to detach VTEP from bridge: %w", err)
	}

	return nil
}

// CreateBridge creates a Linux bridge interface
func CreateBridge(name string, mtu int) error {
	// Check if interface already exists
	_, err := netlink.LinkByName(name)
	if err == nil {
		return &BridgeExistsError{Name: name}
	}

	if mtu == 0 {
		mtu = DefaultMTU
	}

	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
			MTU:  mtu,
		},
	}

	if err := netlink.LinkAdd(bridge); err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}

	// Bring the bridge up
	if err := netlink.LinkSetUp(bridge); err != nil {
		_ = netlink.LinkDel(bridge)
		return fmt.Errorf("failed to bring up bridge: %w", err)
	}

	return nil
}

// DeleteBridge removes a Linux bridge interface
func DeleteBridge(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return &BridgeNotFoundError{Name: name}
		}
		return fmt.Errorf("failed to find bridge: %w", err)
	}

	if _, ok := link.(*netlink.Bridge); !ok {
		return fmt.Errorf("interface %s is not a bridge", name)
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete bridge: %w", err)
	}

	return nil
}

// GetBridgeInfo returns information about a bridge
func GetBridgeInfo(name string) (*BridgeInfo, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return nil, &BridgeNotFoundError{Name: name}
		}
		return nil, fmt.Errorf("failed to find bridge: %w", err)
	}

	bridge, ok := link.(*netlink.Bridge)
	if !ok {
		return nil, fmt.Errorf("interface %s is not a bridge", name)
	}

	// Get attached interfaces
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("failed to list links: %w", err)
	}

	var interfaces []string
	for _, l := range links {
		if l.Attrs().MasterIndex == bridge.Attrs().Index {
			interfaces = append(interfaces, l.Attrs().Name)
		}
	}

	return &BridgeInfo{
		Name:       name,
		Interfaces: interfaces,
		MACAddr:    bridge.Attrs().HardwareAddr.String(),
		MTU:        bridge.Attrs().MTU,
	}, nil
}

// SetLinkUp brings a network interface up
func SetLinkUp(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to find interface: %w", err)
	}

	return netlink.LinkSetUp(link)
}

// SetLinkDown brings a network interface down
func SetLinkDown(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to find interface: %w", err)
	}

	return netlink.LinkSetDown(link)
}

// IsVTEPExists checks if a VTEP interface exists
func IsVTEPExists(name string) bool {
	_, err := GetVTEP(name)
	return err == nil
}

// IsBridgeExists checks if a bridge interface exists
func IsBridgeExists(name string) bool {
	_, err := GetBridgeInfo(name)
	return err == nil
}

// WaitForVTEP waits for a VTEP interface to become available
func WaitForVTEP(ctx context.Context, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for VTEP %s", name)
		case <-ticker.C:
			if IsVTEPExists(name) {
				return nil
			}
		}
	}
}

// Custom error types

type VTEPExistsError struct {
	Name string
}

func (e *VTEPExistsError) Error() string {
	return fmt.Sprintf("VTEP %s already exists", e.Name)
}

type VTEPNotFoundError struct {
	Name string
}

func (e *VTEPNotFoundError) Error() string {
	return fmt.Sprintf("VTEP %s not found", e.Name)
}

type BridgeExistsError struct {
	Name string
}

func (e *BridgeExistsError) Error() string {
	return fmt.Sprintf("bridge %s already exists", e.Name)
}

type BridgeNotFoundError struct {
	Name string
}

func (e *BridgeNotFoundError) Error() string {
	return fmt.Sprintf("bridge %s not found", e.Name)
}

// GenerateVTEPName generates a VTEP interface name from VNI
func GenerateVTEPName(vni int) string {
	return fmt.Sprintf("vxlan%d", vni)
}

// GenerateBridgeName generates a bridge name from VNI
func GenerateBridgeName(vni int) string {
	return fmt.Sprintf("br-vxlan%d", vni)
}

// VNIBytes converts a VNI to its 3-byte representation
func VNIBytes(vni int) [3]byte {
	var b [3]byte
	b[0] = byte(vni >> 16)
	b[1] = byte(vni >> 8)
	b[2] = byte(vni)
	return b
}

// ParseVNI parses a VNI from a byte slice
func ParseVNI(b []byte) int {
	if len(b) < 3 {
		return 0
	}
	return int(b[0])<<16 | int(b[1])<<8 | int(b[2])
}

// htons converts a uint16 to network byte order
func htons(i uint16) uint16 {
	return binary.BigEndian.Uint16([]byte{byte(i >> 8), byte(i)})
}
