package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (e *Executor) executeSnapshot(ctx context.Context, action Action) error {
	var params SnapshotParams
	if err := json.Unmarshal(action.Params, &params); err != nil {
		return fmt.Errorf("unmarshal snapshot params: %w", err)
	}

	if params.VMID == "" {
		return fmt.Errorf("vm_id is required")
	}
	if params.SnapshotName == "" {
		return fmt.Errorf("snapshot_name is required")
	}

	// Get VM info from state
	vm, ok, err := e.Store.GetMicroVM(params.VMID)
	if err != nil {
		return fmt.Errorf("failed to get VM state: %w", err)
	}
	if !ok {
		return fmt.Errorf("VM not found: %s", params.VMID)
	}

	// Determine snapshot directory
	snapshotDir := filepath.Join("/var/lib/nkudo-edge/snapshots", params.SnapshotName)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot dir: %w", err)
	}

	// Pause VM briefly for consistent snapshot
	pauseParams, _ := json.Marshal(PauseParams{VMID: params.VMID})
	if err := e.executePause(ctx, Action{Params: pauseParams}); err != nil {
		return fmt.Errorf("failed to pause VM for snapshot: %w", err)
	}
	defer func() {
		resumeParams, _ := json.Marshal(ResumeParams{VMID: params.VMID})
		_ = e.executeResume(ctx, Action{Params: resumeParams})
	}()

	// Copy disk - use runtime directory for disk image
	srcDisk := filepath.Join("/var/lib/nkudo-edge/vms", params.VMID, "disk.raw")
	if _, err := os.Stat(srcDisk); err != nil {
		// Try qcow2 if raw doesn't exist
		srcDisk = filepath.Join("/var/lib/nkudo-edge/vms", params.VMID, "disk.qcow2")
		if _, err := os.Stat(srcDisk); err != nil {
			return fmt.Errorf("disk image not found for VM %s", params.VMID)
		}
	}
	dstDisk := filepath.Join(snapshotDir, filepath.Base(srcDisk))
	if err := copyFileSnapshot(srcDisk, dstDisk); err != nil {
		return fmt.Errorf("failed to copy disk: %w", err)
	}

	// Save config
	config := map[string]interface{}{
		"vm_id":      params.VMID,
		"name":       vm.Name,
		"vcpu_count": 1, // Default value
		"memory_mib": 256,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	configPath := filepath.Join(snapshotDir, "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// copyFileSnapshot copies a file from src to dst
func copyFileSnapshot(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	buf := make([]byte, 1024*1024) // 1MB buffer
	for {
		n, err := srcFile.Read(buf)
		if n > 0 {
			if _, writeErr := dstFile.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			break
		}
	}

	return dstFile.Sync()
}
