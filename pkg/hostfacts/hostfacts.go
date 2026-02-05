package hostfacts

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"os/exec"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Facts struct {
	CollectedAt time.Time   `json:"collected_at"`
	CPUCores    int         `json:"cpu_cores"`
	MemoryTotal uint64      `json:"memory_total_bytes"`
	MemoryFree  uint64      `json:"memory_free_bytes"`
	Disks       []DiskFact  `json:"disks"`
	OSType      string      `json:"os"`
	OSVersion   string      `json:"os_version"`
	Kernel      string      `json:"kernel"`
	Arch        string      `json:"arch"`
	KVM         KVMFact     `json:"kvm"`
	Interfaces  []IfaceFact `json:"interfaces"`
	Bridges     []string    `json:"bridges"`
}

type DiskFact struct {
	Mountpoint string `json:"mountpoint"`
	TotalBytes uint64 `json:"total_bytes"`
	FreeBytes  uint64 `json:"free_bytes"`
}

type KVMFact struct {
	Present      bool   `json:"present"`
	Readable     bool   `json:"readable"`
	Writable     bool   `json:"writable"`
	CheckMessage string `json:"check_message,omitempty"`
}

type IfaceFact struct {
	Name      string   `json:"name"`
	MAC       string   `json:"mac"`
	Addresses []string `json:"addresses"`
	IsBridge  bool     `json:"is_bridge"`
}

func Collect() (Facts, error) {
	facts := Facts{CollectedAt: time.Now().UTC()}
	facts.CPUCores = runtime.NumCPU()
	facts.Arch = runtime.GOARCH
	facts.OSType, facts.OSVersion = readOSRelease()
	facts.Kernel = readKernelVersion()
	facts.MemoryTotal, facts.MemoryFree = readMemInfo()

	disks, err := collectDisks()
	if err != nil {
		return Facts{}, err
	}
	facts.Disks = disks
	facts.KVM = collectKVM()
	ifaces, bridges := collectInterfaces()
	facts.Interfaces = ifaces
	facts.Bridges = bridges
	return facts, nil
}

func (f Facts) JSON() ([]byte, error) {
	return json.MarshalIndent(f, "", "  ")
}

func readOSRelease() (string, string) {
	b, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return runtime.GOOS, "unknown"
	}
	vals := map[string]string{}
	s := bufio.NewScanner(strings.NewReader(string(b)))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		vals[parts[0]] = strings.Trim(parts[1], `"`)
	}
	name := vals["ID"]
	if name == "" {
		name = runtime.GOOS
	}
	version := vals["VERSION_ID"]
	if version == "" {
		version = vals["VERSION"]
	}
	if version == "" {
		version = "unknown"
	}
	return name, version
}

func readKernelVersion() string {
	b, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err == nil {
		return strings.TrimSpace(string(b))
	}
	out, err := exec.Command("uname", "-r").CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func readMemInfo() (total uint64, free uint64) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	s := bufio.NewScanner(strings.NewReader(string(b)))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "MemTotal:") {
			total = parseMemInfoKiB(line)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			free = parseMemInfoKiB(line)
		}
	}
	return total * 1024, free * 1024
}

func parseMemInfoKiB(line string) uint64 {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}
	v, _ := strconv.ParseUint(parts[1], 10, 64)
	return v
}

func collectDisks() ([]DiskFact, error) {
	mounts := []string{"/", "/var", "/home"}
	seen := map[string]bool{}
	out := make([]DiskFact, 0, len(mounts))
	for _, m := range mounts {
		m = filepath.Clean(m)
		if seen[m] {
			continue
		}
		seen[m] = true
		var st syscall.Statfs_t
		if err := syscall.Statfs(m, &st); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			continue
		}
		out = append(out, DiskFact{
			Mountpoint: m,
			TotalBytes: st.Blocks * uint64(st.Bsize),
			FreeBytes:  st.Bavail * uint64(st.Bsize),
		})
	}
	if len(out) == 0 {
		return nil, errors.New("no disk facts available")
	}
	return out, nil
}

func collectKVM() KVMFact {
	fi, err := os.Stat("/dev/kvm")
	if err != nil {
		return KVMFact{Present: false, CheckMessage: err.Error()}
	}
	res := KVMFact{Present: true}
	if fi.Mode()&os.ModeDevice == 0 {
		res.CheckMessage = "/dev/kvm is not a device"
	}
	f, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		res.Readable = true // open O_RDWR implies readable check failed on write bits/ownership.
		res.Writable = false
		res.CheckMessage = err.Error()
		return res
	}
	res.Readable = true
	res.Writable = true
	f.Close()
	return res
}

func collectInterfaces() ([]IfaceFact, []string) {
	nics, err := net.Interfaces()
	if err != nil {
		return nil, nil
	}
	out := make([]IfaceFact, 0, len(nics))
	bridges := make([]string, 0)
	for _, nic := range nics {
		if nic.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := nic.Addrs()
		ips := make([]string, 0, len(addrs))
		for _, a := range addrs {
			ips = append(ips, a.String())
		}
		isBridge := false
		if _, err := os.Stat(filepath.Join("/sys/class/net", nic.Name, "bridge")); err == nil {
			isBridge = true
			bridges = append(bridges, nic.Name)
		}
		out = append(out, IfaceFact{
			Name:      nic.Name,
			MAC:       nic.HardwareAddr.String(),
			Addresses: ips,
			IsBridge:  isBridge,
		})
	}
	return out, bridges
}
