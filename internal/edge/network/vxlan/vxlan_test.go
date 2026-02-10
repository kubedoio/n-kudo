package vxlan

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// skipIfNotRoot skips the test if not running as root
func skipIfNotRoot(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root privileges")
	}
}

// skipIfNotLinux skips the test if not running on Linux
func skipIfNotLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Test requires Linux")
	}
}

// cleanupInterface removes a network interface if it exists
func cleanupInterface(t *testing.T, name string) {
	exec.Command("ip", "link", "del", name).Run()
}

func TestVXLANConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  VXLANConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: VXLANConfig{
				VNI:      100,
				VTEPName: "vxlan100",
				LocalIP:  "10.0.0.1",
				Port:     4789,
				MTU:      1450,
			},
			wantErr: false,
		},
		{
			name: "VNI too low",
			config: VXLANConfig{
				VNI:      0,
				VTEPName: "vxlan0",
				LocalIP:  "10.0.0.1",
			},
			wantErr: true,
			errMsg:  "VNI",
		},
		{
			name: "VNI too high",
			config: VXLANConfig{
				VNI:      16777216,
				VTEPName: "vxlan16777216",
				LocalIP:  "10.0.0.1",
			},
			wantErr: true,
			errMsg:  "VNI",
		},
		{
			name: "missing VTEPName",
			config: VXLANConfig{
				VNI:     100,
				LocalIP: "10.0.0.1",
			},
			wantErr: true,
			errMsg:  "VTEPName",
		},
		{
			name: "missing LocalIP",
			config: VXLANConfig{
				VNI:      100,
				VTEPName: "vxlan100",
			},
			wantErr: true,
			errMsg:  "LocalIP",
		},
		{
			name: "defaults applied",
			config: VXLANConfig{
				VNI:      100,
				VTEPName: "vxlan100",
				LocalIP:  "10.0.0.1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, should contain %v", err, tt.errMsg)
				}
			}
			// Check defaults were applied
			if !tt.wantErr {
				if tt.config.Port == 0 {
					t.Errorf("Port default not applied")
				}
				if tt.config.MTU == 0 {
					t.Errorf("MTU default not applied")
				}
			}
		})
	}
}

func TestGenerateVTEPName(t *testing.T) {
	tests := []struct {
		vni  int
		want string
	}{
		{100, "vxlan100"},
		{0, "vxlan0"},
		{16777215, "vxlan16777215"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := GenerateVTEPName(tt.vni)
			if got != tt.want {
				t.Errorf("GenerateVTEPName(%d) = %s, want %s", tt.vni, got, tt.want)
			}
		})
	}
}

func TestGenerateBridgeName(t *testing.T) {
	tests := []struct {
		vni  int
		want string
	}{
		{100, "br-vxlan100"},
		{0, "br-vxlan0"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := GenerateBridgeName(tt.vni)
			if got != tt.want {
				t.Errorf("GenerateBridgeName(%d) = %s, want %s", tt.vni, got, tt.want)
			}
		})
	}
}

func TestVNIBytes(t *testing.T) {
	tests := []struct {
		vni  int
		want [3]byte
	}{
		{0, [3]byte{0, 0, 0}},
		{1, [3]byte{0, 0, 1}},
		{256, [3]byte{0, 1, 0}},
		{65536, [3]byte{1, 0, 0}},
		{16777215, [3]byte{255, 255, 255}},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.vni)), func(t *testing.T) {
			got := VNIBytes(tt.vni)
			if got != tt.want {
				t.Errorf("VNIBytes(%d) = %v, want %v", tt.vni, got, tt.want)
			}
		})
	}
}

func TestParseVNI(t *testing.T) {
	tests := []struct {
		b    []byte
		want int
	}{
		{[]byte{0, 0, 0}, 0},
		{[]byte{0, 0, 1}, 1},
		{[]byte{0, 1, 0}, 256},
		{[]byte{1, 0, 0}, 65536},
		{[]byte{255, 255, 255}, 16777215},
		{[]byte{0, 0}, 0},       // too short
		{[]byte{0, 0, 1, 0}, 1}, // extra bytes ignored
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.want)), func(t *testing.T) {
			got := ParseVNI(tt.b)
			if got != tt.want {
				t.Errorf("ParseVNI(%v) = %d, want %d", tt.b, got, tt.want)
			}
		})
	}
}

func TestVTEPExistsError(t *testing.T) {
	err := &VTEPExistsError{Name: "vxlan100"}
	expected := "VTEP vxlan100 already exists"
	if err.Error() != expected {
		t.Errorf("VTEPExistsError.Error() = %s, want %s", err.Error(), expected)
	}
}

func TestVTEPNotFoundError(t *testing.T) {
	err := &VTEPNotFoundError{Name: "vxlan100"}
	expected := "VTEP vxlan100 not found"
	if err.Error() != expected {
		t.Errorf("VTEPNotFoundError.Error() = %s, want %s", err.Error(), expected)
	}
}

func TestBridgeExistsError(t *testing.T) {
	err := &BridgeExistsError{Name: "br0"}
	expected := "bridge br0 already exists"
	if err.Error() != expected {
		t.Errorf("BridgeExistsError.Error() = %s, want %s", err.Error(), expected)
	}
}

func TestBridgeNotFoundError(t *testing.T) {
	err := &BridgeNotFoundError{Name: "br0"}
	expected := "bridge br0 not found"
	if err.Error() != expected {
		t.Errorf("BridgeNotFoundError.Error() = %s, want %s", err.Error(), expected)
	}
}

// Integration tests - require root and Linux

func TestCreateVTEP_Integration(t *testing.T) {
	skipIfNotRoot(t)
	skipIfNotLinux(t)

	vtepName := "vxlan-test-100"
	cleanupInterface(t, vtepName)
	defer cleanupInterface(t, vtepName)

	config := VXLANConfig{
		VNI:         100,
		VTEPName:    vtepName,
		LocalIP:     "127.0.0.1",
		Port:        4789,
		MTU:         1450,
		ParentIface: "lo",
	}

	err := CreateVTEP(config)
	if err != nil {
		t.Fatalf("CreateVTEP() error = %v", err)
	}

	// Verify VTEP exists
	if !IsVTEPExists(vtepName) {
		t.Errorf("VTEP %s should exist after creation", vtepName)
	}

	// Try to create duplicate - should fail
	err = CreateVTEP(config)
	if err == nil {
		t.Error("CreateVTEP() should fail for duplicate VTEP")
	}
	if _, ok := err.(*VTEPExistsError); !ok {
		t.Errorf("CreateVTEP() error type = %T, want *VTEPExistsError", err)
	}
}

func TestDeleteVTEP_Integration(t *testing.T) {
	skipIfNotRoot(t)
	skipIfNotLinux(t)

	vtepName := "vxlan-test-101"
	cleanupInterface(t, vtepName)
	defer cleanupInterface(t, vtepName)

	// Try to delete non-existent VTEP
	err := DeleteVTEP(vtepName)
	if err == nil {
		t.Error("DeleteVTEP() should fail for non-existent VTEP")
	}
	if _, ok := err.(*VTEPNotFoundError); !ok {
		t.Errorf("DeleteVTEP() error type = %T, want *VTEPNotFoundError", err)
	}

	// Create then delete
	config := VXLANConfig{
		VNI:      101,
		VTEPName: vtepName,
		LocalIP:  "127.0.0.1",
	}

	if err := CreateVTEP(config); err != nil {
		t.Fatalf("CreateVTEP() error = %v", err)
	}

	if err := DeleteVTEP(vtepName); err != nil {
		t.Errorf("DeleteVTEP() error = %v", err)
	}

	if IsVTEPExists(vtepName) {
		t.Errorf("VTEP %s should not exist after deletion", vtepName)
	}
}

func TestCreateBridge_Integration(t *testing.T) {
	skipIfNotRoot(t)
	skipIfNotLinux(t)

	bridgeName := "br-test-vxlan"
	cleanupInterface(t, bridgeName)
	defer cleanupInterface(t, bridgeName)

	err := CreateBridge(bridgeName, 1500)
	if err != nil {
		t.Fatalf("CreateBridge() error = %v", err)
	}

	if !IsBridgeExists(bridgeName) {
		t.Errorf("Bridge %s should exist after creation", bridgeName)
	}

	// Try to create duplicate - should fail
	err = CreateBridge(bridgeName, 1500)
	if err == nil {
		t.Error("CreateBridge() should fail for duplicate bridge")
	}
	if _, ok := err.(*BridgeExistsError); !ok {
		t.Errorf("CreateBridge() error type = %T, want *BridgeExistsError", err)
	}
}

func TestAttachToBridge_Integration(t *testing.T) {
	skipIfNotRoot(t)
	skipIfNotLinux(t)

	vtepName := "vxlan-test-102"
	bridgeName := "br-test-attach"

	cleanupInterface(t, vtepName)
	cleanupInterface(t, bridgeName)
	defer cleanupInterface(t, vtepName)
	defer cleanupInterface(t, bridgeName)

	// Create bridge and VTEP
	if err := CreateBridge(bridgeName, 0); err != nil {
		t.Fatalf("CreateBridge() error = %v", err)
	}

	config := VXLANConfig{
		VNI:      102,
		VTEPName: vtepName,
		LocalIP:  "127.0.0.1",
	}

	if err := CreateVTEP(config); err != nil {
		t.Fatalf("CreateVTEP() error = %v", err)
	}

	// Attach VTEP to bridge
	if err := AttachToBridge(vtepName, bridgeName); err != nil {
		t.Errorf("AttachToBridge() error = %v", err)
	}

	// Verify attachment
	info, err := GetBridgeInfo(bridgeName)
	if err != nil {
		t.Fatalf("GetBridgeInfo() error = %v", err)
	}

	found := false
	for _, iface := range info.Interfaces {
		if iface == vtepName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("VTEP %s should be attached to bridge %s", vtepName, bridgeName)
	}

	// Detach
	if err := DetachFromBridge(vtepName); err != nil {
		t.Errorf("DetachFromBridge() error = %v", err)
	}
}

func TestManager_SetupTeardownTunnel_Integration(t *testing.T) {
	skipIfNotRoot(t)
	skipIfNotLinux(t)

	manager := NewManager()
	defer manager.Stop()

	vni := 200
	vtepName := GenerateVTEPName(vni)
	bridgeName := vtepName + "-br"

	// Cleanup any leftovers
	cleanupInterface(t, vtepName)
	cleanupInterface(t, bridgeName)

	config := VXLANConfig{
		VNI:      vni,
		VTEPName: vtepName,
		LocalIP:  "127.0.0.1",
		Port:     4789,
		MTU:      1450,
	}

	ctx := context.Background()

	// Setup tunnel
	if err := manager.SetupTunnel(ctx, config); err != nil {
		t.Fatalf("SetupTunnel() error = %v", err)
	}

	// Verify tunnel exists
	tunnel, err := manager.GetTunnel(vni)
	if err != nil {
		t.Errorf("GetTunnel() error = %v", err)
	}
	if tunnel.Config.VNI != vni {
		t.Errorf("Tunnel VNI = %d, want %d", tunnel.Config.VNI, vni)
	}

	// Verify interface exists
	if !IsVTEPExists(vtepName) {
		t.Errorf("VTEP %s should exist", vtepName)
	}
	if !IsBridgeExists(bridgeName) {
		t.Errorf("Bridge %s should exist", bridgeName)
	}

	// Teardown tunnel
	if err := manager.TeardownTunnel(vni); err != nil {
		t.Errorf("TeardownTunnel() error = %v", err)
	}

	// Verify cleanup
	if IsVTEPExists(vtepName) {
		t.Errorf("VTEP %s should not exist after teardown", vtepName)
	}
}

func TestManager_ListTunnels(t *testing.T) {
	manager := NewManager()
	defer manager.Stop()

	// Initially empty
	tunnels := manager.ListTunnels()
	if len(tunnels) != 0 {
		t.Errorf("Initial tunnel count = %d, want 0", len(tunnels))
	}

	// Add some tunnels manually (without creating interfaces)
	manager.mu.Lock()
	manager.tunnels["vxlan1"] = &VXLANTunnel{
		Config:   VXLANConfig{VNI: 1, VTEPName: "vxlan1"},
		Status:   TunnelStatusActive,
		CreatedAt: time.Now().UTC(),
	}
	manager.tunnels["vxlan2"] = &VXLANTunnel{
		Config:   VXLANConfig{VNI: 2, VTEPName: "vxlan2"},
		Status:   TunnelStatusActive,
		CreatedAt: time.Now().UTC(),
	}
	manager.mu.Unlock()

	tunnels = manager.ListTunnels()
	if len(tunnels) != 2 {
		t.Errorf("Tunnel count = %d, want 2", len(tunnels))
	}

	// Verify TunnelCount
	if count := manager.TunnelCount(); count != 2 {
		t.Errorf("TunnelCount() = %d, want 2", count)
	}
}

func TestManager_GetTunnelStatus(t *testing.T) {
	manager := NewManager()
	defer manager.Stop()

	// Add a tunnel manually
	manager.mu.Lock()
	manager.tunnels["vxlan1"] = &VXLANTunnel{
		Config:     VXLANConfig{VNI: 1, VTEPName: "vxlan1", LocalIP: "10.0.0.1"},
		BridgeName: "br-vxlan1",
		Status:     TunnelStatusActive,
		CreatedAt:  time.Now().UTC(),
	}
	manager.mu.Unlock()

	status, err := manager.GetTunnelStatus(1)
	if err != nil {
		t.Errorf("GetTunnelStatus() error = %v", err)
	}
	if status.VNI != 1 {
		t.Errorf("Status.VNI = %d, want 1", status.VNI)
	}
	if status.VTEPName != "vxlan1" {
		t.Errorf("Status.VTEPName = %s, want vxlan1", status.VTEPName)
	}

	// Non-existent tunnel
	_, err = manager.GetTunnelStatus(999)
	if err == nil {
		t.Error("GetTunnelStatus() should fail for non-existent tunnel")
	}
}

func TestWaitForVTEP(t *testing.T) {
	skipIfNotRoot(t)
	skipIfNotLinux(t)

	vtepName := "vxlan-test-wait"
	cleanupInterface(t, vtepName)
	defer cleanupInterface(t, vtepName)

	ctx := context.Background()

	// Test timeout
	shortCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err := WaitForVTEP(shortCtx, vtepName, 200*time.Millisecond)
	if err == nil {
		t.Error("WaitForVTEP() should timeout for non-existent VTEP")
	}

	// Create VTEP in background
	go func() {
		time.Sleep(50 * time.Millisecond)
		config := VXLANConfig{
			VNI:      300,
			VTEPName: vtepName,
			LocalIP:  "127.0.0.1",
		}
		_ = CreateVTEP(config)
	}()

	// Wait for VTEP
	err = WaitForVTEP(ctx, vtepName, 2*time.Second)
	if err != nil {
		t.Errorf("WaitForVTEP() error = %v", err)
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "VNI", Message: "must be positive"}
	expected := "VNI must be positive"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %s, want %s", err.Error(), expected)
	}
}

func TestTunnelStatus_String(t *testing.T) {
	statuses := []TunnelStatus{
		TunnelStatusPending,
		TunnelStatusCreating,
		TunnelStatusActive,
		TunnelStatusFailed,
		TunnelStatusTearingDown,
		TunnelStatusDestroyed,
	}

	for _, status := range statuses {
		s := string(status)
		if s == "" {
			t.Errorf("TunnelStatus %v has empty string representation", status)
		}
	}
}
