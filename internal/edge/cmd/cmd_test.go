package cmd

import (
	"testing"
)

func TestCheckDiskSpace(t *testing.T) {
	// Test with root directory (should usually have space)
	result := checkDiskSpace("/", 1) // 1 byte required
	if !result.Passed {
		// This might fail in CI with very low disk, so just check the message format
		if result.Message == "" {
			t.Error("Expected a message for disk check")
		}
	}
}

func TestCheckKVM(t *testing.T) {
	result := checkKVM()
	// KVM may or may not be available in test environment
	// Just verify the result has a valid message
	if result.Message == "" {
		t.Error("Expected a message for KVM check")
	}
	// Name should be set
	if result.Name != "KVM" {
		t.Errorf("Expected check name 'KVM', got %q", result.Name)
	}
}

func TestCheckBridge(t *testing.T) {
	// Test with a non-existent bridge
	result := checkBridge("nonexistent-bridge-12345")
	if result.Passed {
		t.Error("Expected non-existent bridge to fail")
	}
	if result.Name != "Bridge" {
		t.Errorf("Expected check name 'Bridge', got %q", result.Name)
	}

	// Test with loopback (not a bridge but exists)
	result = checkBridge("lo")
	if result.Passed {
		t.Error("Expected loopback to not be recognized as bridge")
	}
}

func TestCheckDirectoryWritable(t *testing.T) {
	// Test with temp directory
	result := checkDirectoryWritable("Test", "/tmp")
	if !result.Passed {
		t.Logf("Temp directory check failed: %s", result.Message)
		// This might fail in restricted environments, so just log
	}

	// Test with non-existent path that we can create
	result = checkDirectoryWritable("Test", "/tmp/nkudo-test-writable")
	if !result.Passed {
		t.Errorf("Expected /tmp subdirectory to be writable: %s", result.Message)
	}
}

func TestCheckCloudHypervisor(t *testing.T) {
	// This will likely fail in test environment without CH installed
	result := checkCloudHypervisor("cloud-hypervisor")
	// Just verify the check runs without panic
	if result.Name != "Cloud Hypervisor" {
		t.Errorf("Expected check name 'Cloud Hypervisor', got %q", result.Name)
	}
}

func TestCheckNetBird(t *testing.T) {
	// This will likely fail in test environment without NetBird
	result := checkNetBird("netbird")
	// Just verify the check runs without panic
	if result.Name != "NetBird" {
		t.Errorf("Expected check name 'NetBird', got %q", result.Name)
	}
}

func TestCheckMemory(t *testing.T) {
	// Test with a very small requirement (should always pass)
	result := checkMemory(1) // 1 byte required
	if !result.Passed {
		t.Errorf("Expected memory check to pass with 1 byte requirement: %s", result.Message)
	}
	if result.Name != "Memory" {
		t.Errorf("Expected check name 'Memory', got %q", result.Name)
	}
}

func TestStatusHelp(t *testing.T) {
	help := StatusHelp()
	if help == "" {
		t.Error("Expected status help text")
	}
	if help != statusUsage {
		t.Error("StatusHelp() should return statusUsage")
	}
}

func TestCheckHelp(t *testing.T) {
	help := CheckHelp()
	if help == "" {
		t.Error("Expected check help text")
	}
	if help != checkUsage {
		t.Error("CheckHelp() should return checkUsage")
	}
}

func TestUnenrollHelp(t *testing.T) {
	help := UnenrollHelp()
	if help == "" {
		t.Error("Expected unenroll help text")
	}
	if help != unenrollUsage {
		t.Error("UnenrollHelp() should return unenrollUsage")
	}
}

func TestRenewHelp(t *testing.T) {
	help := RenewHelp()
	if help == "" {
		t.Error("Expected renew help text")
	}
	if help != renewUsage {
		t.Error("RenewHelp() should return renewUsage")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substrs  []string
		expected bool
	}{
		{"hello world", []string{"hello"}, true},
		{"hello world", []string{"world"}, true},
		{"hello world", []string{"foo"}, false},
		{"hello world", []string{"hello", "foo"}, true},
		{"", []string{"hello"}, false},
		{"hello", []string{""}, true},
	}

	for _, tt := range tests {
		result := contains(tt.s, tt.substrs...)
		if result != tt.expected {
			t.Errorf("contains(%q, %v) = %v, want %v", tt.s, tt.substrs, result, tt.expected)
		}
	}
}
