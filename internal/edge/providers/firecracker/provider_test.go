package firecracker

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kubedoio/n-kudo/internal/edge/executor"
)

func TestVMSpecValidate(t *testing.T) {
	base := VMSpec{
		Name:       "vm-a",
		VCPU:       1,
		MemMB:      512,
		KernelPath: "/path/to/kernel",
		TapName:    "tap0",
		BridgeName: "br0",
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("expected valid spec, got err=%v", err)
	}

	bad := base
	bad.Name = ""
	if err := bad.Validate(); err == nil {
		t.Fatal("expected missing name validation error")
	}

	bad = base
	bad.VCPU = 0
	if err := bad.Validate(); err == nil {
		t.Fatal("expected vcpu validation error")
	}

	bad = base
	bad.KernelPath = ""
	if err := bad.Validate(); err == nil {
		t.Fatal("expected kernel_path validation error")
	}

	bad = base
	bad.MACAddress = "not-a-mac"
	if err := bad.Validate(); err == nil {
		t.Fatal("expected invalid mac validation error")
	}
}

func TestRenderUserData(t *testing.T) {
	spec := VMSpec{
		Hostname: "test-vm",
		SSHAuthorizedKeys: []string{
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAItest user@example",
		},
	}
	userData := renderUserData(spec)
	if !strings.Contains(userData, "hostname: test-vm") {
		t.Errorf("expected hostname in user-data, got: %s", userData)
	}
	if !strings.Contains(userData, "ssh-ed25519") {
		t.Errorf("expected SSH key in user-data, got: %s", userData)
	}
	if !strings.Contains(userData, "nkudo") {
		t.Errorf("expected default user nkudo, got: %s", userData)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Test VM", "test-vm"},
		{"My_VM_123", "my-vm-123"},
		{"  spaces  ", "spaces"},
		{"a@b#c", "a-b-c"},
		{"", ""},
	}
	for _, tc := range tests {
		got := slugify(tc.input)
		if got != tc.expected {
			t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestGenerateVMID(t *testing.T) {
	vmID, err := generateVMID("test-vm")
	if err != nil {
		t.Fatalf("generateVMID failed: %v", err)
	}
	if !strings.HasPrefix(vmID, "test-vm-") {
		t.Errorf("expected vmID to start with 'test-vm-', got: %s", vmID)
	}
	if len(vmID) < 10 {
		t.Errorf("expected vmID to have reasonable length, got: %s", vmID)
	}
}

func TestDefaultTapName(t *testing.T) {
	if got := defaultTapName(""); got != "tap0" {
		t.Errorf("defaultTapName(\"\") = %q, want tap0", got)
	}
	if got := defaultTapName("my-vm"); got != "tap-my-vm" {
		t.Errorf("defaultTapName(\"my-vm\") = %q, want tap-my-vm", got)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", "a", "b"); got != "a" {
		t.Errorf("firstNonEmpty = %q, want a", got)
	}
	if got := firstNonEmpty("", "  ", ""); got != "" {
		t.Errorf("firstNonEmpty = %q, want empty", got)
	}
}

func TestFirstPositive(t *testing.T) {
	if got := firstPositive(0, 0, 5, 10); got != 5 {
		t.Errorf("firstPositive = %d, want 5", got)
	}
	if got := firstPositive(0, 0, 0); got != 0 {
		t.Errorf("firstPositive = %d, want 0", got)
	}
}

func TestShellEscape(t *testing.T) {
	if got := shellEscape("hello"); got != "hello" {
		t.Errorf("shellEscape(\"hello\") = %q, want hello", got)
	}
	if got := shellEscape("hello world"); got != "'hello world'" {
		t.Errorf("shellEscape(\"hello world\") = %q, want 'hello world'", got)
	}
	if got := shellEscape("it's"); got != "'it'\\''s'" {
		t.Errorf("shellEscape(\"it's\") = %q", got)
	}
}

func TestRenderCommand(t *testing.T) {
	cmd := renderCommand("firecracker", "--api-sock", "/path/to/sock")
	if !strings.Contains(cmd, "firecracker") {
		t.Errorf("expected firecracker in command: %s", cmd)
	}
	if !strings.Contains(cmd, "--api-sock") {
		t.Errorf("expected --api-sock in command: %s", cmd)
	}
}

func TestProviderCreate(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	vmsDir := filepath.Join(root, "vms")
	imagesDir := filepath.Join(root, "images")

	provider := &Provider{
		RuntimeDir:        vmsDir,
		ImagesDir:         imagesDir,
		DryRun:            true,
		DefaultBridgeName: "br-test0",
	}

	// Create a test disk image first
	testRootfs := filepath.Join(root, "test-rootfs.raw")
	if err := os.WriteFile(testRootfs, []byte("test-rootfs"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test with executor.MicroVMParams
	params := executor.MicroVMParams{
		VMID:       "test-vm-123",
		Name:       "test-vm",
		KernelPath: "/path/to/kernel",
		RootfsPath: testRootfs,
		TapIface:   "tap-test0",
		VCPU:       2,
		MemoryMiB:  512,
	}

	if err := provider.Create(ctx, params); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify VM directory was created
	vmDir := filepath.Join(vmsDir, "test-vm-123")
	if _, err := os.Stat(vmDir); err != nil {
		t.Fatalf("expected vm dir to exist: %v", err)
	}

	// Verify state file was created
	statePath := filepath.Join(vmDir, stateFileName)
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected state file to exist: %v", err)
	}

	// Load and verify state
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("failed to read state: %v", err)
	}
	var meta vmMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("failed to unmarshal state: %v", err)
	}
	if meta.VMID != "test-vm-123" {
		t.Errorf("expected VMID test-vm-123, got %s", meta.VMID)
	}
	if meta.Spec.Name != "test-vm" {
		t.Errorf("expected Name test-vm, got %s", meta.Spec.Name)
	}
	if meta.Spec.VCPU != 2 {
		t.Errorf("expected VCPU 2, got %d", meta.Spec.VCPU)
	}
	if meta.Spec.MemMB != 512 {
		t.Errorf("expected MemMB 512, got %d", meta.Spec.MemMB)
	}
}

func TestDryRunLifecycle(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	vmsDir := filepath.Join(root, "vms")
	imagesDir := filepath.Join(root, "images")

	if err := os.MkdirAll(vmsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a base disk image
	baseDisk := filepath.Join(root, "base.raw")
	if err := os.WriteFile(baseDisk, []byte("base-image"), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := &Provider{
		RuntimeDir:        vmsDir,
		ImagesDir:         imagesDir,
		DryRun:            true,
		DefaultBridgeName: "br-test0",
	}

	spec := VMSpec{
		Name:              "demo-vm",
		VCPU:              1,
		MemMB:             512,
		KernelPath:        "/path/to/vmlinux",
		DiskPath:          baseDisk,
		TapName:           "tap-demo0",
		BridgeName:        "br-test0",
		SSHAuthorizedKeys: []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAItest user@example"},
	}

	vmID, err := provider.CreateVM(ctx, spec)
	if err != nil {
		t.Fatalf("CreateVM failed: %v", err)
	}

	vmDir := filepath.Join(vmsDir, vmID)
	if _, err := os.Stat(filepath.Join(vmDir, "disk.raw")); err != nil {
		t.Fatalf("expected cloned vm disk, stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vmDir, "cloud-init.iso")); err != nil {
		t.Fatalf("expected cloud-init iso, stat error: %v", err)
	}

	status, err := provider.GetVMStatus(ctx, vmID)
	if err != nil {
		t.Fatalf("GetVMStatus(created) failed: %v", err)
	}
	if status != VMStatusCreated {
		t.Fatalf("expected created, got %s", status)
	}

	if err := provider.StartVM(ctx, vmID); err != nil {
		t.Fatalf("StartVM failed: %v", err)
	}
	status, err = provider.GetVMStatus(ctx, vmID)
	if err != nil {
		t.Fatalf("GetVMStatus(running) failed: %v", err)
	}
	if status != VMStatusRunning {
		t.Fatalf("expected running, got %s", status)
	}

	// Check that we can get process ID
	pid, err := provider.GetProcessID(ctx, vmID)
	if err != nil {
		t.Fatalf("GetProcessID failed: %v", err)
	}
	if pid <= 0 {
		t.Fatalf("expected positive PID, got %d", pid)
	}

	logData, err := provider.CollectConsoleLog(ctx, vmID)
	if err != nil {
		t.Fatalf("CollectConsoleLog failed: %v", err)
	}
	if !strings.Contains(string(logData), "dry-run: firecracker started") {
		t.Fatalf("expected dry-run log marker, got: %s", string(logData))
	}

	if err := provider.StopVM(ctx, vmID); err != nil {
		t.Fatalf("StopVM failed: %v", err)
	}
	status, err = provider.GetVMStatus(ctx, vmID)
	if err != nil {
		t.Fatalf("GetVMStatus(stopped) failed: %v", err)
	}
	if status != VMStatusStopped {
		t.Fatalf("expected stopped, got %s", status)
	}

	commandsPath := filepath.Join(vmDir, commandsFileName)
	commands, err := os.ReadFile(commandsPath)
	if err != nil {
		t.Fatalf("read commands.log failed: %v", err)
	}
	if !strings.Contains(string(commands), "ip tuntap add dev tap-demo0 mode tap") {
		t.Fatalf("expected tap create command in log, got:\n%s", string(commands))
	}
	if !strings.Contains(string(commands), "firecracker") {
		t.Fatalf("expected firecracker command in log, got:\n%s", string(commands))
	}

	if err := provider.DeleteVM(ctx, vmID); err != nil {
		t.Fatalf("DeleteVM first call failed: %v", err)
	}
	if err := provider.DeleteVM(ctx, vmID); err != nil {
		t.Fatalf("DeleteVM idempotent call failed: %v", err)
	}
	if _, err := os.Stat(vmDir); !os.IsNotExist(err) {
		t.Fatalf("expected vm dir removed, stat err=%v", err)
	}
}

func TestFCClient(t *testing.T) {
	// Create a mock Firecracker API server
	mux := http.NewServeMux()
	var receivedBody map[string]interface{}

	mux.HandleFunc("/machine-config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/actions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	listener, err := net.Listen("unix", filepath.Join(t.TempDir(), "test.sock"))
	if err != nil {
		t.Fatalf("failed to create unix listener: %v", err)
	}
	defer listener.Close()

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	// Create client
	client := newFCClient(listener.Addr().String())

	// Test PUT
	ctx := context.Background()
	cfg := MachineConfig{VcpuCount: 2, MemSizeMiB: 512}
	if err := client.put(ctx, "/machine-config", cfg); err != nil {
		t.Fatalf("put failed: %v", err)
	}

	if receivedBody == nil {
		t.Fatal("expected request body to be captured")
	}
	if receivedBody["vcpu_count"] != float64(2) {
		t.Errorf("expected vcpu_count=2, got %v", receivedBody["vcpu_count"])
	}
	if receivedBody["mem_size_mib"] != float64(512) {
		t.Errorf("expected mem_size_mib=512, got %v", receivedBody["mem_size_mib"])
	}

	// Test action
	action := Action{ActionType: ActionTypeInstanceStart}
	if err := client.put(ctx, "/actions", action); err != nil {
		t.Fatalf("put action failed: %v", err)
	}
}

func TestWaitForSocket(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "test.sock")

	// Test timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- waitForSocket(socketPath, 50*time.Millisecond)
	}()

	select {
	case err := <-errChan:
		if err == nil {
			t.Fatal("expected timeout error")
		}
	case <-ctx.Done():
		t.Fatal("test timed out")
	}

	// Test success - create socket after short delay
	socketPath2 := filepath.Join(dir, "test2.sock")
	listenerReady := make(chan bool)
	go func() {
		listener, err := net.Listen("unix", socketPath2)
		if err != nil {
			listenerReady <- false
			return
		}
		listenerReady <- true
		defer listener.Close()
		// Keep listener open for a bit
		time.Sleep(1 * time.Second)
	}()
	
	// Wait for listener to be created
	select {
	case ready := <-listenerReady:
		if !ready {
			t.Fatal("failed to create listener")
		}
	case <-time.After(100 * time.Millisecond):
		// Continue anyway, the file might exist
	}

	if err := waitForSocket(socketPath2, 500*time.Millisecond); err != nil {
		t.Fatalf("waitForSocket failed: %v", err)
	}
}

func TestProcessAlive(t *testing.T) {
	// PID 0 should not be alive
	if processAlive(0) {
		t.Error("PID 0 should not be alive")
	}

	// PID -1 should not be alive
	if processAlive(-1) {
		t.Error("PID -1 should not be alive")
	}

	// Current process should be alive
	if !processAlive(os.Getpid()) {
		t.Error("current process should be alive")
	}
}

func TestWaitUntilDead(t *testing.T) {
	// Test with non-existent process
	if !waitUntilDead(999999, 100*time.Millisecond) {
		t.Error("waitUntilDead should return true for non-existent process")
	}

	// Test with current process (should timeout)
	if waitUntilDead(os.Getpid(), 50*time.Millisecond) {
		t.Error("waitUntilDead should return false for running process (timeout)")
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	content := []byte("hello world")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	copied, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(copied) != string(content) {
		t.Errorf("copied content mismatch: got %q, want %q", string(copied), string(content))
	}
}

func TestCreateSparseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sparse.raw")

	if err := createSparseFile(path, 1024*1024); err != nil {
		t.Fatalf("createSparseFile failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 1024*1024 {
		t.Errorf("expected size 1MB, got %d", info.Size())
	}

	// Test invalid size
	if err := createSparseFile(filepath.Join(dir, "invalid.raw"), 0); err == nil {
		t.Error("expected error for zero size")
	}
}

func TestProviderDuplicateCreate(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	provider := &Provider{
		RuntimeDir:        filepath.Join(root, "vms"),
		ImagesDir:         filepath.Join(root, "images"),
		DryRun:            true,
		DefaultBridgeName: "br0",
	}

	// Create a test disk image
	testDisk := filepath.Join(root, "test.raw")
	if err := os.WriteFile(testDisk, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	spec := VMSpec{
		Name:       "test",
		VCPU:       1,
		MemMB:      256,
		KernelPath: "/path/to/kernel",
		DiskPath:   testDisk,
		TapName:    "tap0",
		BridgeName: "br0",
	}

	vmID1, err := provider.CreateVM(ctx, spec)
	if err != nil {
		t.Fatalf("first CreateVM failed: %v", err)
	}

	// Create with same spec but different name to get different ID
	spec2 := spec
	spec2.Name = "test2"
	vmID2, err := provider.CreateVM(ctx, spec2)
	if err != nil {
		t.Fatalf("second CreateVM failed: %v", err)
	}

	// Different specs should produce different VM IDs
	if vmID1 == vmID2 {
		t.Errorf("expected different VM IDs for different specs, got same: %s", vmID1)
	}

	// Verify both VMs exist
	vmDir1 := filepath.Join(provider.RuntimeDir, vmID1)
	vmDir2 := filepath.Join(provider.RuntimeDir, vmID2)
	if _, err := os.Stat(vmDir1); err != nil {
		t.Errorf("VM1 dir should exist: %v", err)
	}
	if _, err := os.Stat(vmDir2); err != nil {
		t.Errorf("VM2 dir should exist: %v", err)
	}
}

// Ensure Provider implements executor.MicroVMProvider
var _ executor.MicroVMProvider = (*Provider)(nil)
