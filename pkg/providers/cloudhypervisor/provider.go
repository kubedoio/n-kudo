package cloudhypervisor

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/n-kudo/n-kudo-edge/pkg/executor"
	"github.com/n-kudo/n-kudo-edge/pkg/state"
)

const (
	defaultRuntimeDir = "/var/lib/nkudo-edge/vms"
	defaultImagesDir  = "/var/lib/nkudo-edge/images"

	defaultCHBinary        = "cloud-hypervisor"
	defaultIPBinary        = "ip"
	defaultCloudLocalDSBin = "cloud-localds"
	defaultGenISOImageBin  = "genisoimage"

	stateFileName    = "state.json"
	pidFileName      = "ch.pid"
	commandsFileName = "commands.log"
)

type Provider struct {
	Binary            string
	State             *state.Store
	RuntimeDir        string
	ImagesDir         string
	IPBinary          string
	CloudLocalDSBin   string
	GenISOImageBin    string
	DefaultBridgeName string
	DryRun            bool
	StopTimeout       time.Duration

	mu         sync.Mutex
	nextDryPID int
}

type vmMeta struct {
	VMID             string    `json:"vm_id"`
	Spec             VMSpec    `json:"spec"`
	DiskPath         string    `json:"disk_path"`
	CachedBaseImage  string    `json:"cached_base_image,omitempty"`
	CloudInitISOPath string    `json:"cloud_init_iso_path"`
	APISocketPath    string    `json:"api_socket_path"`
	StdoutPath       string    `json:"stdout_path"`
	StderrPath       string    `json:"stderr_path"`
	ConsolePath      string    `json:"console_path"`
	PID              int       `json:"pid"`
	Status           VMStatus  `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

var _ VMProvider = (*Provider)(nil)
var _ executor.MicroVMProvider = (*Provider)(nil)

func (p *Provider) CreateVM(ctx context.Context, spec VMSpec) (string, error) {
	return p.createVM(ctx, spec, "")
}

func (p *Provider) StartVM(ctx context.Context, vmID string) error {
	if err := p.ensureDefaults(); err != nil {
		return err
	}
	meta, err := p.loadMeta(vmID)
	if err != nil {
		return err
	}
	if p.DryRun && meta.Status == VMStatusRunning {
		return nil
	}
	if meta.Status == VMStatusRunning && processAlive(meta.PID) {
		return nil
	}

	args := p.renderCHArgs(meta)
	if err := p.appendCommand(vmID, renderCommand(p.Binary, args...)); err != nil {
		return err
	}

	if p.DryRun {
		p.mu.Lock()
		if p.nextDryPID == 0 {
			p.nextDryPID = 40000
		}
		p.nextDryPID++
		meta.PID = p.nextDryPID
		p.mu.Unlock()

		meta.Status = VMStatusRunning
		if err := os.WriteFile(meta.StdoutPath, []byte("dry-run: cloud-hypervisor started\n"), 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(p.vmDir(vmID), pidFileName), []byte(strconv.Itoa(meta.PID)+"\n"), 0o644); err != nil {
			return err
		}
		if err := p.saveMeta(meta); err != nil {
			return err
		}
		return p.syncStateStore(meta)
	}

	if _, err := exec.LookPath(p.Binary); err != nil {
		return fmt.Errorf("cloud-hypervisor binary not found (%s): %w", p.Binary, err)
	}

	stdout, err := os.OpenFile(meta.StdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer stdout.Close()
	stderr, err := os.OpenFile(meta.StderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer stderr.Close()

	cmd := exec.CommandContext(ctx, p.Binary, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start cloud-hypervisor: %w", err)
	}
	pid := cmd.Process.Pid
	go func(targetVMID string, targetPID int, child *exec.Cmd) {
		_ = child.Wait()
		_ = p.markStoppedIfCurrent(targetVMID, targetPID)
	}(vmID, pid, cmd)

	meta.PID = pid
	meta.Status = VMStatusRunning
	if err := os.WriteFile(filepath.Join(p.vmDir(vmID), pidFileName), []byte(strconv.Itoa(meta.PID)+"\n"), 0o644); err != nil {
		return err
	}
	if err := p.saveMeta(meta); err != nil {
		return err
	}
	return p.syncStateStore(meta)
}

func (p *Provider) StopVM(ctx context.Context, vmID string) error {
	if err := p.ensureDefaults(); err != nil {
		return err
	}
	meta, err := p.loadMeta(vmID)
	if err != nil {
		return err
	}
	if p.DryRun {
		meta.Status = VMStatusStopped
		meta.PID = 0
		if err := p.saveMeta(meta); err != nil {
			return err
		}
		return p.syncStateStore(meta)
	}
	if meta.Status != VMStatusRunning || meta.PID <= 0 || !processAlive(meta.PID) {
		meta.Status = VMStatusStopped
		meta.PID = 0
		if err := p.saveMeta(meta); err != nil {
			return err
		}
		return p.syncStateStore(meta)
	}

	_ = p.appendCommand(vmID, renderCommand("PUT", "unix://"+meta.APISocketPath, "/api/v1/vm.shutdown"))
	_ = p.shutdownViaAPISocket(ctx, meta.APISocketPath)

	dead := waitUntilDead(meta.PID, p.StopTimeout)
	if !dead {
		proc, _ := os.FindProcess(meta.PID)
		if proc != nil {
			_ = p.appendCommand(vmID, fmt.Sprintf("kill -TERM %d", meta.PID))
			_ = proc.Signal(syscall.SIGTERM)
			dead = waitUntilDead(meta.PID, 5*time.Second)
		}
	}
	if !dead {
		proc, _ := os.FindProcess(meta.PID)
		if proc != nil {
			_ = p.appendCommand(vmID, fmt.Sprintf("kill -KILL %d", meta.PID))
			_ = proc.Signal(syscall.SIGKILL)
		}
	}

	meta.PID = 0
	meta.Status = VMStatusStopped
	if err := p.saveMeta(meta); err != nil {
		return err
	}
	return p.syncStateStore(meta)
}

func (p *Provider) DeleteVM(ctx context.Context, vmID string) error {
	if err := p.ensureDefaults(); err != nil {
		return err
	}
	meta, err := p.loadMeta(vmID)
	if err != nil {
		if errors.Is(err, ErrVMNotFound) {
			if p.State != nil {
				_ = p.State.DeleteMicroVM(vmID)
			}
			_ = os.RemoveAll(p.vmDir(vmID))
			return nil
		}
		return err
	}

	_ = p.StopVM(ctx, vmID)
	_ = p.cleanupTap(ctx, vmID, meta.Spec.TapName)

	if err := os.RemoveAll(p.vmDir(vmID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove vm dir: %w", err)
	}
	if p.State != nil {
		_ = p.State.DeleteMicroVM(vmID)
	}
	return nil
}

func (p *Provider) GetVMStatus(_ context.Context, vmID string) (VMStatus, error) {
	if err := p.ensureDefaults(); err != nil {
		return "", err
	}
	meta, err := p.loadMeta(vmID)
	if err != nil {
		return "", err
	}
	if p.DryRun {
		return meta.Status, nil
	}
	if meta.Status == VMStatusRunning && !processAlive(meta.PID) {
		meta.Status = VMStatusStopped
		meta.PID = 0
		if err := p.saveMeta(meta); err != nil {
			return "", err
		}
		_ = p.syncStateStore(meta)
	}
	return meta.Status, nil
}

func (p *Provider) CollectConsoleLog(_ context.Context, vmID string) ([]byte, error) {
	if err := p.ensureDefaults(); err != nil {
		return nil, err
	}
	meta, err := p.loadMeta(vmID)
	if err != nil {
		return nil, err
	}
	files := []string{meta.ConsolePath, meta.StdoutPath, meta.StderrPath}
	var out bytes.Buffer
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil || len(b) == 0 {
			continue
		}
		out.WriteString("== " + filepath.Base(f) + " ==\n")
		out.Write(b)
		if b[len(b)-1] != '\n' {
			out.WriteByte('\n')
		}
	}
	return out.Bytes(), nil
}

// Create keeps executor.MicroVMProvider compatibility.
func (p *Provider) Create(ctx context.Context, params executor.MicroVMParams) error {
	spec := VMSpec{
		Name:       firstNonEmpty(params.Name, params.VMID, "vm"),
		VCPU:       firstPositive(params.VCPU, 1),
		MemMB:      firstPositive(params.MemoryMiB, 256),
		DiskPath:   params.RootfsPath,
		TapName:    firstNonEmpty(params.TapIface, defaultTapName(params.VMID)),
		BridgeName: firstNonEmpty(p.DefaultBridgeName, "br0"),
	}
	_, err := p.createVM(ctx, spec, params.VMID)
	return err
}

// Start keeps executor.MicroVMProvider compatibility.
func (p *Provider) Start(ctx context.Context, vmID string) error { return p.StartVM(ctx, vmID) }

// Stop keeps executor.MicroVMProvider compatibility.
func (p *Provider) Stop(ctx context.Context, vmID string) error { return p.StopVM(ctx, vmID) }

// Delete keeps executor.MicroVMProvider compatibility.
func (p *Provider) Delete(ctx context.Context, vmID string) error { return p.DeleteVM(ctx, vmID) }

func (p *Provider) BuildCreateParamsFromVM(vm state.MicroVM) executor.MicroVMParams {
	return executor.MicroVMParams{
		VMID:       vm.ID,
		Name:       vm.Name,
		RootfsPath: vm.RootfsPath,
		TapIface:   vm.TapIface,
		VCPU:       1,
		MemoryMiB:  256,
	}
}

func (p *Provider) PIDString(vmID string) string {
	meta, err := p.loadMeta(vmID)
	if err != nil {
		return ""
	}
	if meta.PID <= 0 {
		return ""
	}
	return strconv.Itoa(meta.PID)
}

func (p *Provider) createVM(ctx context.Context, spec VMSpec, requestedID string) (vmID string, err error) {
	if err := p.ensureDefaults(); err != nil {
		return "", err
	}
	spec.normalize()
	if err := spec.Validate(); err != nil {
		return "", err
	}
	if requestedID == "" {
		requestedID, err = generateVMID(spec.Name)
		if err != nil {
			return "", err
		}
	}
	vmID = requestedID
	vmDir := p.vmDir(vmID)
	if err := os.MkdirAll(vmDir, 0o755); err != nil {
		return "", err
	}
	if _, statErr := os.Stat(filepath.Join(vmDir, stateFileName)); statErr == nil {
		return vmID, nil
	}

	tapCreated := false
	defer func() {
		if err == nil {
			return
		}
		if tapCreated {
			_ = p.cleanupTap(context.Background(), vmID, spec.TapName)
		}
		_ = os.RemoveAll(vmDir)
	}()

	diskPath, cachedPath, err := p.prepareDisk(vmDir, spec.DiskPath, spec.DiskSizeMB)
	if err != nil {
		return "", err
	}

	isoPath, err := p.prepareCloudInitISO(ctx, vmID, vmDir, spec)
	if err != nil {
		return "", err
	}

	if err := p.setupTap(ctx, vmID, spec.TapName, spec.BridgeName); err != nil {
		return "", err
	}
	tapCreated = true

	meta := vmMeta{
		VMID:             vmID,
		Spec:             spec,
		DiskPath:         diskPath,
		CachedBaseImage:  cachedPath,
		CloudInitISOPath: isoPath,
		APISocketPath:    filepath.Join(vmDir, "api.sock"),
		StdoutPath:       filepath.Join(vmDir, "stdout.log"),
		StderrPath:       filepath.Join(vmDir, "stderr.log"),
		ConsolePath:      filepath.Join(vmDir, "console.log"),
		Status:           VMStatusCreated,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := p.saveMeta(meta); err != nil {
		return "", err
	}
	if err := p.syncStateStore(meta); err != nil {
		return "", err
	}
	return vmID, nil
}

func (p *Provider) ensureDefaults() error {
	if p.RuntimeDir == "" {
		p.RuntimeDir = defaultRuntimeDir
	}
	if p.ImagesDir == "" {
		p.ImagesDir = defaultImagesDir
	}
	if p.Binary == "" {
		p.Binary = defaultCHBinary
	}
	if p.IPBinary == "" {
		p.IPBinary = defaultIPBinary
	}
	if p.CloudLocalDSBin == "" {
		p.CloudLocalDSBin = defaultCloudLocalDSBin
	}
	if p.GenISOImageBin == "" {
		p.GenISOImageBin = defaultGenISOImageBin
	}
	if p.DefaultBridgeName == "" {
		p.DefaultBridgeName = "br0"
	}
	if p.StopTimeout <= 0 {
		p.StopTimeout = 15 * time.Second
	}
	if err := os.MkdirAll(p.RuntimeDir, 0o755); err != nil {
		return err
	}
	return os.MkdirAll(p.ImagesDir, 0o755)
}

func (p *Provider) prepareDisk(vmDir, sourcePath string, diskSizeMB int) (diskPath string, cachePath string, err error) {
	sourcePath = strings.TrimSpace(sourcePath)
	ext := ".raw"
	if strings.EqualFold(filepath.Ext(sourcePath), ".qcow2") {
		ext = ".qcow2"
	}
	diskPath = filepath.Join(vmDir, "disk"+ext)
	if sourcePath == "" {
		if err := createSparseFile(diskPath, int64(diskSizeMB)*1024*1024); err != nil {
			return "", "", fmt.Errorf("create empty disk: %w", err)
		}
		return diskPath, "", nil
	}

	base, err := p.cacheBaseImage(sourcePath)
	if err != nil {
		return "", "", err
	}
	if err := copyFile(base, diskPath); err != nil {
		return "", "", fmt.Errorf("clone base image: %w", err)
	}
	return diskPath, base, nil
}

func (p *Provider) cacheBaseImage(sourcePath string) (string, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return "", fmt.Errorf("stat source disk %s: %w", sourcePath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("source disk path is a directory: %s", sourcePath)
	}

	sum := sha256.Sum256([]byte(sourcePath))
	name := filepath.Base(sourcePath)
	cachePath := filepath.Join(p.ImagesDir, fmt.Sprintf("%s-%s", hex.EncodeToString(sum[:8]), name))
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}
	if err := copyFile(sourcePath, cachePath); err != nil {
		return "", fmt.Errorf("cache base image: %w", err)
	}
	return cachePath, nil
}

func (p *Provider) prepareCloudInitISO(ctx context.Context, vmID, vmDir string, spec VMSpec) (string, error) {
	isoPath := spec.CloudInitISOPath
	if isoPath == "" {
		isoPath = filepath.Join(vmDir, "cloud-init.iso")
	}
	seedDir := filepath.Join(vmDir, "seed")
	if err := os.MkdirAll(seedDir, 0o755); err != nil {
		return "", err
	}
	metaPath := filepath.Join(seedDir, "meta-data")
	userPath := filepath.Join(seedDir, "user-data")

	metaData := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", vmID, spec.Hostname)
	if err := os.WriteFile(metaPath, []byte(metaData), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(userPath, []byte(renderUserData(spec)), 0o644); err != nil {
		return "", err
	}

	var cmd []string
	if p.DryRun {
		cmd = []string{p.CloudLocalDSBin, isoPath, userPath, metaPath}
		if err := os.WriteFile(isoPath, []byte("dry-run cloud-init seed"), 0o644); err != nil {
			return "", err
		}
		if err := p.appendCommand(vmID, renderCommand(cmd[0], cmd[1:]...)); err != nil {
			return "", err
		}
		return isoPath, nil
	}

	if _, err := exec.LookPath(p.CloudLocalDSBin); err == nil {
		cmd = []string{p.CloudLocalDSBin, isoPath, userPath, metaPath}
	} else if _, err := exec.LookPath(p.GenISOImageBin); err == nil {
		cmd = []string{
			p.GenISOImageBin,
			"-output", isoPath,
			"-volid", "cidata",
			"-joliet",
			"-rock",
			userPath,
			metaPath,
		}
	} else {
		return "", errors.New("cloud-init ISO builder not found (need cloud-localds or genisoimage)")
	}

	if err := p.appendCommand(vmID, renderCommand(cmd[0], cmd[1:]...)); err != nil {
		return "", err
	}
	if err := runCmd(ctx, cmd[0], cmd[1:]...); err != nil {
		return "", fmt.Errorf("build cloud-init iso: %w", err)
	}
	return isoPath, nil
}

func (p *Provider) setupTap(ctx context.Context, vmID, tapName, bridgeName string) error {
	cmds := renderTapSetupCommands(p.IPBinary, tapName, bridgeName)
	for _, cmd := range cmds {
		if err := p.appendCommand(vmID, renderCommand(cmd[0], cmd[1:]...)); err != nil {
			return err
		}
		if p.DryRun {
			continue
		}
		if err := runCmd(ctx, cmd[0], cmd[1:]...); err != nil {
			if len(cmd) >= 3 && cmd[1] == "tuntap" && strings.Contains(strings.ToLower(err.Error()), "exists") {
				continue
			}
			return fmt.Errorf("setup tap: %w", err)
		}
	}
	return nil
}

func (p *Provider) cleanupTap(ctx context.Context, vmID, tapName string) error {
	if strings.TrimSpace(tapName) == "" {
		return nil
	}
	cmd := []string{p.IPBinary, "link", "del", tapName}
	if err := p.appendCommand(vmID, renderCommand(cmd[0], cmd[1:]...)); err != nil {
		return err
	}
	if p.DryRun {
		return nil
	}
	if err := runCmd(ctx, cmd[0], cmd[1:]...); err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "cannot find device") || strings.Contains(lower, "not found") {
			return nil
		}
		return err
	}
	return nil
}

func (p *Provider) renderCHArgs(meta vmMeta) []string {
	args := []string{
		"--api-socket", meta.APISocketPath,
		"--cpus", fmt.Sprintf("boot=%d", meta.Spec.VCPU),
		"--memory", fmt.Sprintf("size=%dM", meta.Spec.MemMB),
		"--disk", fmt.Sprintf("path=%s", meta.DiskPath),
		"--disk", fmt.Sprintf("path=%s,readonly=on", meta.CloudInitISOPath),
		"--serial", fmt.Sprintf("file=%s", meta.ConsolePath),
		"--console", "off",
	}
	netCfg := fmt.Sprintf("tap=%s", meta.Spec.TapName)
	if meta.Spec.MACAddress != "" {
		netCfg = netCfg + ",mac=" + meta.Spec.MACAddress
	}
	args = append(args, "--net", netCfg)
	return args
}

func (p *Provider) vmDir(vmID string) string { return filepath.Join(p.RuntimeDir, vmID) }

func (p *Provider) loadMeta(vmID string) (vmMeta, error) {
	b, err := os.ReadFile(filepath.Join(p.vmDir(vmID), stateFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return vmMeta{}, ErrVMNotFound
		}
		return vmMeta{}, err
	}
	var meta vmMeta
	if err := json.Unmarshal(b, &meta); err != nil {
		return vmMeta{}, err
	}
	return meta, nil
}

func (p *Provider) saveMeta(meta vmMeta) error {
	meta.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(p.vmDir(meta.VMID), stateFileName), b, 0o600)
}

func (p *Provider) appendCommand(vmID, command string) error {
	path := filepath.Join(p.vmDir(vmID), commandsFileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.WriteString(f, time.Now().UTC().Format(time.RFC3339)+" "+command+"\n")
	return err
}

func (p *Provider) markStoppedIfCurrent(vmID string, pid int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	meta, err := p.loadMeta(vmID)
	if err != nil {
		return nil
	}
	if meta.PID != pid {
		return nil
	}
	meta.PID = 0
	meta.Status = VMStatusStopped
	if err := p.saveMeta(meta); err != nil {
		return err
	}
	return p.syncStateStore(meta)
}

func (p *Provider) syncStateStore(meta vmMeta) error {
	if p.State == nil {
		return nil
	}
	return p.State.UpsertMicroVM(state.MicroVM{
		ID:         meta.VMID,
		Name:       meta.Spec.Name,
		RootfsPath: meta.DiskPath,
		TapIface:   meta.Spec.TapName,
		CHPID:      meta.PID,
		Status:     strings.ToUpper(string(meta.Status)),
	})
}

func (p *Provider) shutdownViaAPISocket(ctx context.Context, socketPath string) error {
	if _, err := os.Stat(socketPath); err != nil {
		return err
	}
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   3 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://localhost/api/v1/vm.shutdown", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("shutdown response: %s", resp.Status)
	}
	return nil
}

func generateVMID(name string) (string, error) {
	slug := slugify(name)
	if slug == "" {
		slug = "vm"
	}
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", slug, hex.EncodeToString(random[:])), nil
}

func slugify(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	if in == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range in {
		isWord := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isWord {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 32 {
		out = out[:32]
	}
	return out
}

func renderUserData(spec VMSpec) string {
	if strings.TrimSpace(spec.UserData) != "" {
		return spec.UserData
	}
	var b strings.Builder
	b.WriteString("#cloud-config\n")
	b.WriteString("hostname: ")
	b.WriteString(spec.Hostname)
	b.WriteByte('\n')
	b.WriteString("users:\n")
	b.WriteString("  - name: nkudo\n")
	b.WriteString("    sudo: ALL=(ALL) NOPASSWD:ALL\n")
	b.WriteString("    groups: [sudo]\n")
	b.WriteString("    shell: /bin/bash\n")
	if len(spec.SSHAuthorizedKeys) == 0 {
		b.WriteString("    ssh_authorized_keys: []\n")
	} else {
		b.WriteString("    ssh_authorized_keys:\n")
		for _, key := range spec.SSHAuthorizedKeys {
			b.WriteString("      - ")
			b.WriteString(strings.TrimSpace(key))
			b.WriteByte('\n')
		}
	}
	b.WriteString("package_update: false\n")
	return b.String()
}

func renderTapSetupCommands(ipBinary, tapName, bridgeName string) [][]string {
	return [][]string{
		{ipBinary, "tuntap", "add", "dev", tapName, "mode", "tap"},
		{ipBinary, "link", "set", tapName, "master", bridgeName},
		{ipBinary, "link", "set", tapName, "up"},
	}
}

func renderCommand(name string, args ...string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, shellEscape(name))
	for _, arg := range args {
		parts = append(parts, shellEscape(arg))
	}
	return strings.Join(parts, " ")
}

func shellEscape(s string) string {
	if s == "" {
		return "''"
	}
	if strings.IndexFunc(s, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' || r == '\\'
	}) == -1 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func runCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if trimmed := strings.TrimSpace(string(out)); trimmed != "" {
			return fmt.Errorf("%w: %s", err, trimmed)
		}
		return err
	}
	return nil
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, syscall.Signal(0))
	if err == nil {
		return true
	}
	return !errors.Is(err, syscall.ESRCH)
}

func waitUntilDead(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return !processAlive(pid)
}

func createSparseFile(path string, sizeBytes int64) error {
	if sizeBytes <= 0 {
		return errors.New("sizeBytes must be > 0")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Truncate(sizeBytes); err != nil {
		return err
	}
	return f.Sync()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func defaultTapName(vmID string) string {
	vmID = strings.TrimSpace(vmID)
	if vmID == "" {
		return "tap0"
	}
	name := "tap-" + slugify(vmID)
	if len(name) > 15 {
		name = name[:15]
	}
	return name
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}
