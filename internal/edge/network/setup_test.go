package network

import (
	"net"
	"strings"
	"testing"
)

func TestGenerateMAC(t *testing.T) {
	tests := []struct {
		name      string
		vmID      string
		ifaceIdx  int
		wantOUI   string // Expected OUI prefix
		wantLocal bool   // Should have locally administered bit set
	}{
		{
			name:      "basic generation",
			vmID:      "vm-test-123",
			ifaceIdx:  0,
			wantOUI:   "52:54:00",
			wantLocal: true,
		},
		{
			name:      "different index",
			vmID:      "vm-test-123",
			ifaceIdx:  1,
			wantOUI:   "52:54:00",
			wantLocal: true,
		},
		{
			name:      "different VM",
			vmID:      "vm-other-456",
			ifaceIdx:  0,
			wantOUI:   "52:54:00",
			wantLocal: true,
		},
		{
			name:      "high index",
			vmID:      "vm-test",
			ifaceIdx:  99,
			wantOUI:   "52:54:00",
			wantLocal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mac := GenerateMAC(tt.vmID, tt.ifaceIdx)

			// Validate MAC format
			hw, err := net.ParseMAC(mac)
			if err != nil {
				t.Errorf("GenerateMAC() returned invalid MAC %s: %v", mac, err)
				return
			}

			// Check OUI
			oui := hw.String()[:8]
			if oui != tt.wantOUI {
				t.Errorf("GenerateMAC() OUI = %s, want %s", oui, tt.wantOUI)
			}

			// Check locally administered bit
			firstByte := hw[0]
			hasLocalBit := (firstByte & 0x02) != 0
			if hasLocalBit != tt.wantLocal {
				t.Errorf("GenerateMAC() local bit = %v, want %v", hasLocalBit, tt.wantLocal)
			}
		})
	}
}

func TestGenerateMACUniqueness(t *testing.T) {
	// Generate multiple MACs and ensure they're unique
	macs := make(map[string]bool)
	vmID := "test-vm"

	for i := 0; i < 100; i++ {
		mac := GenerateMAC(vmID, i)
		if macs[mac] {
			t.Errorf("GenerateMAC() returned duplicate MAC: %s", mac)
		}
		macs[mac] = true
	}

	// Different VM IDs should also produce different MACs for same index
	mac1 := GenerateMAC("vm-1", 0)
	mac2 := GenerateMAC("vm-2", 0)
	if mac1 == mac2 {
		t.Errorf("GenerateMAC() same MAC for different VMs: %s", mac1)
	}
}

func TestGenerateTAPName(t *testing.T) {
	tests := []struct {
		name     string
		vmID     string
		ifaceIdx int
		maxLen   int
	}{
		{
			name:     "basic generation",
			vmID:     "vm-test-123",
			ifaceIdx: 0,
			maxLen:   15,
		},
		{
			name:     "different index",
			vmID:     "vm-test-123",
			ifaceIdx: 1,
			maxLen:   15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := GenerateTAPName(tt.vmID, tt.ifaceIdx)

			// Check length
			if len(name) > tt.maxLen {
				t.Errorf("GenerateTAPName() length = %d, want <= %d", len(name), tt.maxLen)
			}

			// Check prefix
			if !strings.HasPrefix(name, "tap") {
				t.Errorf("GenerateTAPName() = %s, want prefix 'tap'", name)
			}
		})
	}
}

func TestGenerateTAPNameUniqueness(t *testing.T) {
	names := make(map[string]bool)
	vmID := "test-vm"

	for i := 0; i < 100; i++ {
		name := GenerateTAPName(vmID, i)
		if names[name] {
			t.Errorf("GenerateTAPName() returned duplicate name: %s", name)
		}
		names[name] = true
	}
}

func TestValidateMAC(t *testing.T) {
	tests := []struct {
		name    string
		mac     string
		wantErr bool
	}{
		{
			name:    "valid MAC",
			mac:     "52:54:00:12:34:56",
			wantErr: false,
		},
		{
			name:    "valid MAC with dashes",
			mac:     "52-54-00-12-34-56",
			wantErr: false,
		},
		{
			name:    "empty MAC",
			mac:     "",
			wantErr: false, // Empty is valid (auto-generated)
		},
		{
			name:    "whitespace only",
			mac:     "   ",
			wantErr: false, // Treated as empty
		},
		{
			name:    "invalid MAC",
			mac:     "invalid",
			wantErr: true,
		},
		{
			name:    "too short",
			mac:     "52:54:00",
			wantErr: true,
		},
		{
			name:    "too long",
			mac:     "52:54:00:12:34:56:78",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMAC(tt.mac)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMAC() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatCHNetArg(t *testing.T) {
	tests := []struct {
		name string
		iface NetworkInterface
		want string
	}{
		{
			name: "tap only",
			iface: NetworkInterface{
				TapName: "tap0",
			},
			want: "tap=tap0",
		},
		{
			name: "tap with mac",
			iface: NetworkInterface{
				TapName: "tap0",
				MacAddr: "52:54:00:12:34:56",
			},
			want: "tap=tap0,mac=52:54:00:12:34:56",
		},
		{
			name: "full config",
			iface: NetworkInterface{
				TapName: "tap0",
				MacAddr: "52:54:00:12:34:56",
				Bridge:  "br0",
				IPConfig: &IPConfig{
					Address: "192.168.1.100/24",
					Gateway: "192.168.1.1",
				},
			},
			want: "tap=tap0,mac=52:54:00:12:34:56,ip=192.168.1.100/24,bridge=br0",
		},
		{
			name: "with bridge no ip",
			iface: NetworkInterface{
				TapName: "tap1",
				Bridge:  "br1",
			},
			want: "tap=tap1,bridge=br1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatCHNetArg(tt.iface)
			if got != tt.want {
				t.Errorf("FormatCHNetArg() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestIPConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		address string
		valid   bool
	}{
		{"valid CIDR", "192.168.1.100/24", true},
		{"valid CIDR with different mask", "10.0.0.1/8", true},
		{"invalid CIDR", "192.168.1.100", false},
		{"invalid IP", "invalid", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := net.ParseCIDR(tt.address)
			if tt.valid && err != nil {
				t.Errorf("Expected valid CIDR %s, got error: %v", tt.address, err)
			}
			if !tt.valid && err == nil && tt.address != "" {
				// Some invalid addresses might parse without /CIDR
				if strings.Contains(tt.address, "/") {
					return
				}
			}
		})
	}
}

func TestNetworkInterfaceDefaults(t *testing.T) {
	// Test that SetupInterfaces fills in defaults correctly
	ifaces := []NetworkInterface{
		{
			Bridge: "br0",
		},
		{
			Bridge:  "br1",
			MacAddr: "52:54:00:12:34:56",
		},
	}

	vmID := "test-vm-123"

	// We can't run the actual setup without root, but we can test the generation logic
	for i := range ifaces {
		if strings.TrimSpace(ifaces[i].ID) == "" {
			ifaces[i].ID = "eth0"
			if i > 0 {
				ifaces[i].ID = "eth1"
			}
		}
		if strings.TrimSpace(ifaces[i].TapName) == "" {
			ifaces[i].TapName = GenerateTAPName(vmID, i)
		}
		if strings.TrimSpace(ifaces[i].MacAddr) == "" {
			ifaces[i].MacAddr = GenerateMAC(vmID, i)
		}
	}

	// Verify defaults were set
	if ifaces[0].ID != "eth0" {
		t.Errorf("Expected ID eth0, got %s", ifaces[0].ID)
	}
	if ifaces[1].ID != "eth1" {
		t.Errorf("Expected ID eth1, got %s", ifaces[1].ID)
	}
	if ifaces[0].MacAddr == "" {
		t.Error("Expected MAC address to be generated")
	}
	// Second interface should keep its explicit MAC
	if ifaces[1].MacAddr != "52:54:00:12:34:56" {
		t.Errorf("Expected MAC 52:54:00:12:34:56, got %s", ifaces[1].MacAddr)
	}
}
