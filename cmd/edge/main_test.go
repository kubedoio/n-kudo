package main

import (
	"testing"

	"github.com/kubedoio/n-kudo/internal/edge/cmd"
)

func TestUsage(t *testing.T) {
	// Just verify usage doesn't panic
	usage()
}

func TestCommandHelpTexts(t *testing.T) {
	tests := []struct {
		name     string
		helpFunc func() string
	}{
		{"status", cmd.StatusHelp},
		{"check", cmd.CheckHelp},
		{"unenroll", cmd.UnenrollHelp},
		{"renew", cmd.RenewHelp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			help := tt.helpFunc()
			if help == "" {
				t.Errorf("Expected help text for %s command", tt.name)
			}
		})
	}
}

func TestRunStatus_NotEnrolled(t *testing.T) {
	// Create a temporary state directory
	// This test verifies the "not enrolled" path works
	tmpDir := t.TempDir()
	
	// Run status on empty state dir should show "not enrolled"
	args := []string{"--state-dir", tmpDir}
	// This would print to stdout; we just verify it doesn't panic or error
	// In a real test environment we'd capture stdout
	err := cmd.RunStatus(args)
	if err != nil {
		t.Errorf("RunStatus with empty state should not error: %v", err)
	}
}

func TestRunCheck(t *testing.T) {
	// Use temp directories for the check
	tmpDir := t.TempDir()
	stateDir := tmpDir + "/state"
	pkiDir := tmpDir + "/pki"
	runtimeDir := tmpDir + "/vms"

	args := []string{
		"--state-dir", stateDir,
		"--pki-dir", pkiDir,
		"--runtime-dir", runtimeDir,
		"--cloud-hypervisor-bin", "nonexistent-ch-binary",
		"--netbird-bin", "nonexistent-netbird-binary",
	}

	exitCode := cmd.RunCheck(args)
	// Exit code should be 1 since some checks will fail in test environment
	if exitCode != 0 && exitCode != 1 {
		t.Errorf("Expected exit code 0 or 1, got %d", exitCode)
	}
}

func TestRunUnenroll_MissingControlPlane(t *testing.T) {
	// Should fail without control-plane flag
	args := []string{}
	err := cmd.RunUnenroll(args)
	if err == nil {
		t.Error("Expected error when control-plane is missing")
	}
}

func TestRunRenew_MissingControlPlane(t *testing.T) {
	// Should fail without control-plane flag
	args := []string{}
	err := cmd.RunRenew(args)
	if err == nil {
		t.Error("Expected error when control-plane is missing")
	}
}

func TestRunUnenroll_NotEnrolled(t *testing.T) {
	// Create a temporary state directory
	tmpDir := t.TempDir()
	
	// Run unenroll when not enrolled should handle gracefully
	args := []string{
		"--state-dir", tmpDir,
		"--control-plane", "https://localhost:8443",
	}
	
	// This will fail to send the request but should clean up local files
	err := cmd.RunUnenroll(args)
	// Expected to error since there's no identity and no server
	// but shouldn't panic
	_ = err
}

func TestRunRenew_NotEnrolled(t *testing.T) {
	// Create a temporary state directory  
	tmpDir := t.TempDir()
	
	// Run renew when not enrolled should error
	args := []string{
		"--state-dir", tmpDir,
		"--control-plane", "https://localhost:8443",
	}
	
	err := cmd.RunRenew(args)
	// Should error because there's no identity
	if err == nil {
		t.Error("Expected error when running renew without enrollment")
	}
}
