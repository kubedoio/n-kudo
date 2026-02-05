package cloudhypervisor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVMSpecValidate(t *testing.T) {
	base := VMSpec{
		Name:       "vm-a",
		VCPU:       1,
		MemMB:      512,
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
	bad.MACAddress = "not-a-mac"
	if err := bad.Validate(); err == nil {
		t.Fatal("expected invalid mac validation error")
	}
}

func TestRenderCHArgs(t *testing.T) {
	p := &Provider{}
	meta := vmMeta{
		APISocketPath:    "/tmp/vm/api.sock",
		DiskPath:         "/tmp/vm/disk.raw",
		CloudInitISOPath: "/tmp/vm/cloud-init.iso",
		ConsolePath:      "/tmp/vm/console.log",
		Spec: VMSpec{
			VCPU:       2,
			MemMB:      1024,
			TapName:    "tap-test0",
			MACAddress: "02:00:00:00:00:01",
		},
	}

	args := p.renderCHArgs(meta)
	full := strings.Join(args, " ")
	mustContain := []string{
		"--api-socket /tmp/vm/api.sock",
		"--cpus boot=2",
		"--memory size=1024M",
		"--disk path=/tmp/vm/disk.raw",
		"--disk path=/tmp/vm/cloud-init.iso,readonly=on",
		"--serial file=/tmp/vm/console.log",
		"--net tap=tap-test0,mac=02:00:00:00:00:01",
	}
	for _, s := range mustContain {
		if !strings.Contains(full, s) {
			t.Fatalf("expected args to contain %q, got: %s", s, full)
		}
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

	logData, err := provider.CollectConsoleLog(ctx, vmID)
	if err != nil {
		t.Fatalf("CollectConsoleLog failed: %v", err)
	}
	if !strings.Contains(string(logData), "dry-run: cloud-hypervisor started") {
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
	if !strings.Contains(string(commands), "cloud-hypervisor") {
		t.Fatalf("expected cloud-hypervisor command in log, got:\n%s", string(commands))
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
