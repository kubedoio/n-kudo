package network

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// IPConfig represents static IP configuration for a network interface
type IPConfig struct {
	Address string `json:"address" yaml:"address"` // CIDR notation, e.g., "192.168.1.100/24"
	Gateway string `json:"gateway,omitempty" yaml:"gateway,omitempty"` // e.g., "192.168.1.1"
}

// NetworkInterface represents a network interface configuration for a VM
type NetworkInterface struct {
	ID       string    `json:"id" yaml:"id"`             // e.g., "eth0", "eth1"
	TapName  string    `json:"tap_name" yaml:"tap_name"` // TAP device name
	MacAddr  string    `json:"mac,omitempty" yaml:"mac,omitempty"` // Optional MAC address
	Bridge   string    `json:"bridge" yaml:"bridge"`     // Bridge to attach to
	IPConfig *IPConfig `json:"ip_config,omitempty" yaml:"ip_config,omitempty"` // Optional static IP
}

// CreateTAPDevice creates a TAP device with optional bridge attachment
func CreateTAPDevice(name string, bridge string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("TAP device name is required")
	}

	// Create the TAP device
	cmd := exec.Command("ip", "tuntap", "add", "dev", name, "mode", "tap")
	if out, err := cmd.CombinedOutput(); err != nil {
		// Ignore "already exists" errors
		if !strings.Contains(strings.ToLower(string(out)), "exists") {
			return fmt.Errorf("create TAP device %s: %w: %s", name, err, string(out))
		}
	}

	// Attach to bridge if specified
	if strings.TrimSpace(bridge) != "" {
		cmd = exec.Command("ip", "link", "set", name, "master", bridge)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("attach TAP %s to bridge %s: %w: %s", name, bridge, err, string(out))
		}
	}

	// Bring the interface up
	cmd = exec.Command("ip", "link", "set", name, "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("bring up TAP %s: %w: %s", name, err, string(out))
	}

	return nil
}

// DeleteTAPDevice removes a TAP device
func DeleteTAPDevice(name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}

	cmd := exec.Command("ip", "link", "del", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		lower := strings.ToLower(string(out))
		if strings.Contains(lower, "cannot find device") || strings.Contains(lower, "not found") {
			return nil // Already deleted
		}
		return fmt.Errorf("delete TAP device %s: %w: %s", name, err, string(out))
	}
	return nil
}

// ConfigureInterface sets IP address on interface
func ConfigureInterface(name string, cfg IPConfig) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("interface name is required")
	}
	if strings.TrimSpace(cfg.Address) == "" {
		return fmt.Errorf("IP address is required")
	}

	// Validate CIDR format
	if _, _, err := net.ParseCIDR(cfg.Address); err != nil {
		return fmt.Errorf("invalid IP address %s: %w", cfg.Address, err)
	}

	// Add IP address to interface
	cmd := exec.Command("ip", "addr", "add", cfg.Address, "dev", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Ignore "already exists" errors
		if !strings.Contains(strings.ToLower(string(out)), "exists") {
			return fmt.Errorf("configure IP %s on %s: %w: %s", cfg.Address, name, err, string(out))
		}
	}

	// Add default route if gateway is specified
	if strings.TrimSpace(cfg.Gateway) != "" {
		cmd = exec.Command("ip", "route", "add", "default", "via", cfg.Gateway, "dev", name)
		if out, err := cmd.CombinedOutput(); err != nil {
			// Ignore "already exists" or "File exists" errors
			lower := strings.ToLower(string(out))
			if !strings.Contains(lower, "exists") && !strings.Contains(lower, "file exists") {
				return fmt.Errorf("add default route via %s: %w: %s", cfg.Gateway, err, string(out))
			}
		}
	}

	return nil
}

// GenerateMAC generates a unique MAC address based on VM ID and interface index
// Uses the OUI 52:54:00 (QEMU/KVM range) followed by hash-based bytes
func GenerateMAC(vmID string, ifaceIdx int) string {
	h := sha256.New()
	h.Write([]byte(vmID))
	h.Write([]byte(fmt.Sprintf("iface-%d", ifaceIdx)))
	sum := h.Sum(nil)

	// Use OUI 52:54:00 (QEMU/KVM) + 3 bytes from hash
	// Set local bit (bit 1 of first byte) to indicate locally administered
	mac := net.HardwareAddr{
		0x52, // 01010010 - QEMU OUI
		0x54, // 01010100
		0x00, // 00000000
		sum[0],
		sum[1],
		sum[2],
	}

	// Set locally administered bit (bit 1 of first byte)
	mac[0] = (mac[0] & 0xfe) | 0x02

	return mac.String()
}

// ValidateMAC validates a MAC address string
func ValidateMAC(mac string) error {
	if strings.TrimSpace(mac) == "" {
		return nil // Empty is valid (will be auto-generated)
	}
	if _, err := net.ParseMAC(mac); err != nil {
		return fmt.Errorf("invalid MAC address %s: %w", mac, err)
	}
	return nil
}

// GenerateTAPName generates a TAP device name based on VM ID and interface index
func GenerateTAPName(vmID string, ifaceIdx int) string {
	h := sha256.New()
	h.Write([]byte(vmID))
	h.Write([]byte(fmt.Sprintf("tap-%d", ifaceIdx)))
	sum := h.Sum(nil)

	// Use first 8 hex chars of hash for uniqueness
	suffix := hex.EncodeToString(sum[:4])[:8]

	// TAP names are limited to 15 characters
	name := fmt.Sprintf("tap%s", suffix)
	if len(name) > 15 {
		name = name[:15]
	}
	return name
}

// FormatCHNetArg formats a network argument for Cloud Hypervisor
// Format: tap=<tap_name>[,mac=<mac>][,ip=<ip>][,bridge=<bridge>]
func FormatCHNetArg(iface NetworkInterface) string {
	parts := []string{fmt.Sprintf("tap=%s", iface.TapName)}

	if strings.TrimSpace(iface.MacAddr) != "" {
		parts = append(parts, fmt.Sprintf("mac=%s", iface.MacAddr))
	}

	if iface.IPConfig != nil && strings.TrimSpace(iface.IPConfig.Address) != "" {
		parts = append(parts, fmt.Sprintf("ip=%s", iface.IPConfig.Address))
	}

	if strings.TrimSpace(iface.Bridge) != "" {
		parts = append(parts, fmt.Sprintf("bridge=%s", iface.Bridge))
	}

	return strings.Join(parts, ",")
}

// SetupInterfaces creates and configures all network interfaces for a VM
func SetupInterfaces(vmID string, ifaces []NetworkInterface) error {
	for i, iface := range ifaces {
		// Generate default values if not provided
		if strings.TrimSpace(iface.ID) == "" {
			iface.ID = fmt.Sprintf("eth%d", i)
		}
		if strings.TrimSpace(iface.TapName) == "" {
			iface.TapName = GenerateTAPName(vmID, i)
		}
		if strings.TrimSpace(iface.MacAddr) == "" {
			iface.MacAddr = GenerateMAC(vmID, i)
		}

		// Create TAP device
		if err := CreateTAPDevice(iface.TapName, iface.Bridge); err != nil {
			return fmt.Errorf("setup interface %s: %w", iface.ID, err)
		}

		// Configure IP if specified
		if iface.IPConfig != nil {
			if err := ConfigureInterface(iface.TapName, *iface.IPConfig); err != nil {
				return fmt.Errorf("configure IP for %s: %w", iface.ID, err)
			}
		}

		ifaces[i] = iface
	}
	return nil
}

// CleanupInterfaces removes all TAP devices for a VM
func CleanupInterfaces(ifaces []NetworkInterface) error {
	var errs []string
	for _, iface := range ifaces {
		if strings.TrimSpace(iface.TapName) == "" {
			continue
		}
		if err := DeleteTAPDevice(iface.TapName); err != nil {
			errs = append(errs, fmt.Sprintf("failed to delete %s: %v", iface.TapName, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errs, "; "))
	}
	return nil
}
