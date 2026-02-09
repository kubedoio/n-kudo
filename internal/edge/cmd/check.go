package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

const checkUsage = `Usage: edge check [options]

Pre-flight check for requirements. Verifies that the system is ready to
run the edge agent.

Options:
  --state-dir string    State directory (default "/var/lib/nkudo-edge/state")
  --pki-dir string      PKI directory (default "/var/lib/nkudo-edge/pki")
  --runtime-dir string  Runtime directory (default "/var/lib/nkudo-edge/vms")
  --cloud-hypervisor-bin string  Cloud Hypervisor binary path (default "cloud-hypervisor")
  --netbird-bin string  NetBird binary path (default "netbird")

Exit codes:
  0  All checks passed
  1  One or more checks failed
`

// CheckOptions holds the configuration for the check command
type CheckOptions struct {
	StateDir      string
	PKIDir        string
	RuntimeDir    string
	CHBinary      string
	NetbirdBinary string
}

// CheckResult represents the result of a single check
type CheckResult struct {
	Name    string
	Passed  bool
	Message string
}

// RunCheck executes the check command
func RunCheck(args []string) int {
	opts := CheckOptions{}
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.StringVar(&opts.StateDir, "state-dir", "/var/lib/nkudo-edge/state", "State directory")
	fs.StringVar(&opts.PKIDir, "pki-dir", "/var/lib/nkudo-edge/pki", "PKI directory")
	fs.StringVar(&opts.RuntimeDir, "runtime-dir", "/var/lib/nkudo-edge/vms", "Runtime directory")
	fs.StringVar(&opts.CHBinary, "cloud-hypervisor-bin", "cloud-hypervisor", "Cloud Hypervisor binary path")
	fs.StringVar(&opts.NetbirdBinary, "netbird-bin", "netbird", "NetBird binary path")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	results := runAllChecks(opts)

	passed := 0
	failed := 0
	for _, r := range results {
		symbol := "✓"
		if !r.Passed {
			symbol = "✗"
			failed++
		} else {
			passed++
		}
		fmt.Printf("%s %s\n", symbol, r.Message)
	}

	fmt.Println()
	if failed > 0 {
		fmt.Printf("Some checks failed (%d passed, %d failed).\n", passed, failed)
		fmt.Println("Please fix the issues above before running the edge agent.")
		return 1
	}
	fmt.Println("All checks passed! System is ready to run the edge agent.")
	return 0
}

func runAllChecks(opts CheckOptions) []CheckResult {
	results := []CheckResult{}

	// KVM check
	results = append(results, checkKVM())

	// Cloud Hypervisor binary check
	results = append(results, checkCloudHypervisor(opts.CHBinary))

	// Bridge check
	results = append(results, checkBridge("br0"))

	// Directory checks
	results = append(results, checkDirectoryWritable("State", opts.StateDir))
	results = append(results, checkDirectoryWritable("PKI", opts.PKIDir))
	results = append(results, checkDirectoryWritable("Runtime", opts.RuntimeDir))

	// NetBird check
	results = append(results, checkNetBird(opts.NetbirdBinary))

	// Disk space check
	results = append(results, checkDiskSpace("/var", 50*1024*1024*1024)) // 50GB

	// Memory check
	results = append(results, checkMemory(4 * 1024 * 1024 * 1024)) // 4GB

	return results
}

func checkKVM() CheckResult {
	info, err := os.Stat("/dev/kvm")
	if err != nil {
		return CheckResult{
			Name:    "KVM",
			Passed:  false,
			Message: "KVM not available (/dev/kvm not found)",
		}
	}

	if info.Mode()&os.ModeDevice == 0 {
		return CheckResult{
			Name:    "KVM",
			Passed:  false,
			Message: "KVM not available (/dev/kvm is not a device)",
		}
	}

	// Check read/write access
	f, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return CheckResult{
			Name:    "KVM",
			Passed:  false,
			Message: fmt.Sprintf("KVM not accessible (permission denied: %v)", err),
		}
	}
	f.Close()

	return CheckResult{
		Name:    "KVM",
		Passed:  true,
		Message: "KVM available",
	}
}

func checkCloudHypervisor(binary string) CheckResult {
	path, err := exec.LookPath(binary)
	if err != nil {
		// Try to find it in common locations
		commonPaths := []string{
			"/usr/bin/cloud-hypervisor",
			"/usr/local/bin/cloud-hypervisor",
			"/opt/cloud-hypervisor/cloud-hypervisor",
		}
		for _, p := range commonPaths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	if path == "" {
		return CheckResult{
			Name:    "Cloud Hypervisor",
			Passed:  false,
			Message: "Cloud Hypervisor binary not found",
		}
	}

	// Check if executable
	info, err := os.Stat(path)
	if err != nil {
		return CheckResult{
			Name:    "Cloud Hypervisor",
			Passed:  false,
			Message: fmt.Sprintf("Cloud Hypervisor binary not accessible (%v)", err),
		}
	}

	if info.Mode()&0o111 == 0 {
		return CheckResult{
			Name:    "Cloud Hypervisor",
			Passed:  false,
			Message: fmt.Sprintf("Cloud Hypervisor binary not executable (%s)", path),
		}
	}

	return CheckResult{
		Name:    "Cloud Hypervisor",
		Passed:  true,
		Message: fmt.Sprintf("Cloud Hypervisor binary found (%s)", path),
	}
}

func checkBridge(bridgeName string) CheckResult {
	bridgePath := fmt.Sprintf("/sys/class/net/%s", bridgeName)
	_, err := os.Stat(bridgePath)
	if err != nil {
		return CheckResult{
			Name:    "Bridge",
			Passed:  false,
			Message: fmt.Sprintf("Bridge %s does not exist", bridgeName),
		}
	}

	// Check if it's actually a bridge
	bridgeSubPath := fmt.Sprintf("/sys/class/net/%s/bridge", bridgeName)
	_, err = os.Stat(bridgeSubPath)
	if err != nil {
		return CheckResult{
			Name:    "Bridge",
			Passed:  false,
			Message: fmt.Sprintf("Interface %s exists but is not a bridge", bridgeName),
		}
	}

	return CheckResult{
		Name:    "Bridge",
		Passed:  true,
		Message: fmt.Sprintf("Bridge %s exists", bridgeName),
	}
}

func checkDirectoryWritable(name, path string) CheckResult {
	// Try to create the directory if it doesn't exist
	if err := os.MkdirAll(path, 0o700); err != nil {
		return CheckResult{
			Name:    name,
			Passed:  false,
			Message: fmt.Sprintf("%s directory not creatable (%s: %v)", name, path, err),
		}
	}

	// Check if writable by creating a test file
	testFile := fmt.Sprintf("%s/.write-test-%d", path, os.Getpid())
	f, err := os.Create(testFile)
	if err != nil {
		return CheckResult{
			Name:    name,
			Passed:  false,
			Message: fmt.Sprintf("%s directory not writable (%s)", name, path),
		}
	}
	f.Close()
	os.Remove(testFile)

	return CheckResult{
		Name:    name,
		Passed:  true,
		Message: fmt.Sprintf("%s directory writable (%s)", name, path),
	}
}

func checkNetBird(binary string) CheckResult {
	_, err := exec.LookPath(binary)
	if err != nil {
		return CheckResult{
			Name:    "NetBird",
			Passed:  false,
			Message: "NetBird not installed",
		}
	}

	return CheckResult{
		Name:    "NetBird",
		Passed:  true,
		Message: "NetBird installed",
	}
}

func checkDiskSpace(path string, minBytes uint64) CheckResult {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		// Try root if the specific path fails
		if err := syscall.Statfs("/", &stat); err != nil {
			return CheckResult{
				Name:    "Disk",
				Passed:  false,
				Message: "Could not check disk space",
			}
		}
	}

	available := stat.Bavail * uint64(stat.Bsize)
	availableGB := available / (1024 * 1024 * 1024)
	minGB := minBytes / (1024 * 1024 * 1024)

	if available < minBytes {
		return CheckResult{
			Name:    "Disk",
			Passed:  false,
			Message: fmt.Sprintf("Disk space insufficient (%d GB available, %d GB required)", availableGB, minGB),
		}
	}

	return CheckResult{
		Name:    "Disk",
		Passed:  true,
		Message: fmt.Sprintf("Disk space sufficient (%d GB available)", availableGB),
	}
}

func checkMemory(minBytes uint64) CheckResult {
	var memInfo syscall.Sysinfo_t
	if err := syscall.Sysinfo(&memInfo); err != nil {
		// Fallback for non-Linux systems
		if runtime.GOOS != "linux" {
			return CheckResult{
				Name:    "Memory",
				Passed:  true,
				Message: "Memory check skipped (non-Linux system)",
			}
		}
		return CheckResult{
			Name:    "Memory",
			Passed:  false,
			Message: "Could not check memory",
		}
	}

	totalRAM := memInfo.Totalram * uint64(memInfo.Unit)
	totalGB := totalRAM / (1024 * 1024 * 1024)
	minGB := minBytes / (1024 * 1024 * 1024)

	if totalRAM < minBytes {
		return CheckResult{
			Name:    "Memory",
			Passed:  false,
			Message: fmt.Sprintf("Memory insufficient (%d GB total, %d GB recommended)", totalGB, minGB),
		}
	}

	return CheckResult{
		Name:    "Memory",
		Passed:  true,
		Message: fmt.Sprintf("Memory sufficient (%d GB total)", totalGB),
	}
}

// CheckHelp returns the help text for the check command
func CheckHelp() string {
	return checkUsage
}
