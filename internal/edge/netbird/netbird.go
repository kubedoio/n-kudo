package netbird

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Binary string
}

type ConnectivityState string

const (
	StateConnected     ConnectivityState = "Connected"
	StateDegraded      ConnectivityState = "Degraded"
	StateNotConfigured ConnectivityState = "NotConfigured"
)

type Status struct {
	Connected     bool   `json:"connected"`
	State         string `json:"state,omitempty"`
	Reason        string `json:"reason,omitempty"`
	PeerID        string `json:"peer_id,omitempty"`
	IPv4          string `json:"ipv4,omitempty"`
	ManagementURL string `json:"management_url,omitempty"`
	NetworkID     string `json:"network_id,omitempty"`
	Raw           string `json:"raw,omitempty"`
}

type ProbeType string

const (
	ProbeTypeHTTP ProbeType = "http"
	ProbeTypePing ProbeType = "ping"
)

type ProbeConfig struct {
	Type          ProbeType     `json:"type,omitempty"`
	Target        string        `json:"target,omitempty"`
	Timeout       time.Duration `json:"timeout,omitempty"`
	HTTPStatusMin int           `json:"http_status_min,omitempty"`
	HTTPStatusMax int           `json:"http_status_max,omitempty"`
	HTTPUserAgent string        `json:"http_user_agent,omitempty"`
}

type Config struct {
	Enabled        bool        `json:"enabled"`
	AutoJoin       bool        `json:"auto_join"`
	SetupKey       string      `json:"setup_key,omitempty"`
	Hostname       string      `json:"hostname,omitempty"`
	RequireService bool        `json:"require_service"`
	InstallCommand []string    `json:"install_command,omitempty"`
	Probe          ProbeConfig `json:"probe,omitempty"`
}

type Snapshot struct {
	State           ConnectivityState `json:"state"`
	Reason          string            `json:"reason,omitempty"`
	CheckedAt       time.Time         `json:"checked_at"`
	CLIInstalled    bool              `json:"cli_installed"`
	ServiceDetected bool              `json:"service_detected"`
	ServiceRunning  bool              `json:"service_running"`
	InstallAttempt  bool              `json:"install_attempt"`
	ProbeOK         bool              `json:"probe_ok"`
	ProbeOutput     string            `json:"probe_output,omitempty"`
	Peer            Status            `json:"peer"`
}

type ControlPlaneStatus struct {
	State      ConnectivityState `json:"state"`
	Connected  bool              `json:"connected"`
	Reason     string            `json:"reason,omitempty"`
	PeerID     string            `json:"peer_id,omitempty"`
	IPv4       string            `json:"ipv4,omitempty"`
	NetworkID  string            `json:"network_id,omitempty"`
	CheckedAt  time.Time         `json:"checked_at"`
	RawStatus  string            `json:"raw_status,omitempty"`
	ServiceUp  bool              `json:"service_up"`
	ProbeOK    bool              `json:"probe_ok"`
	ProbeNotes string            `json:"probe_notes,omitempty"`
}

func (s Snapshot) ControlPlaneConnected() bool {
	return s.State == StateConnected
}

func (s Snapshot) ToControlPlaneStatus() ControlPlaneStatus {
	return ControlPlaneStatus{
		State:      s.State,
		Connected:  s.ControlPlaneConnected(),
		Reason:     s.Reason,
		PeerID:     s.Peer.PeerID,
		IPv4:       s.Peer.IPv4,
		NetworkID:  s.Peer.NetworkID,
		CheckedAt:  s.CheckedAt,
		RawStatus:  s.Peer.Raw,
		ServiceUp:  s.ServiceRunning,
		ProbeOK:    s.ProbeOK,
		ProbeNotes: s.ProbeOutput,
	}
}

func (c Client) Evaluate(ctx context.Context, cfg Config) (Snapshot, error) {
	now := time.Now().UTC()
	snap := Snapshot{
		State:     StateNotConfigured,
		CheckedAt: now,
		ProbeOK:   true,
	}
	if !cfg.Enabled {
		snap.Reason = "netbird integration disabled"
		return snap, nil
	}

	bin := c.binary()
	if path, err := exec.LookPath(bin); err == nil && path != "" {
		snap.CLIInstalled = true
	}

	if !snap.CLIInstalled && len(cfg.InstallCommand) > 0 {
		snap.InstallAttempt = true
		if err := c.runInstall(ctx, cfg.InstallCommand); err != nil {
			snap.State = StateDegraded
			snap.Reason = fmt.Sprintf("netbird install failed: %v", err)
			return snap, nil
		}
		if path, err := exec.LookPath(bin); err == nil && path != "" {
			snap.CLIInstalled = true
		}
	}

	if !snap.CLIInstalled {
		snap.Reason = "netbird CLI not installed"
		return snap, nil
	}

	serviceRunning, serviceDetected, _ := c.DetectService(ctx)
	snap.ServiceRunning = serviceRunning
	snap.ServiceDetected = serviceDetected

	setupKey := strings.TrimSpace(cfg.SetupKey)
	allowJoin := cfg.AutoJoin && setupKey != ""

	status, statusErr := c.Status(ctx)
	if statusErr == nil {
		snap.Peer = status
	}

	if (statusErr != nil || !status.Connected) && allowJoin {
		if err := c.Join(ctx, setupKey, cfg.Hostname); err != nil {
			setupKey = ""
			snap.State = StateDegraded
			snap.Reason = "netbird join failed"
			snap.ProbeOutput = err.Error()
			return snap, nil
		}
		setupKey = ""
		status, statusErr = c.Status(ctx)
		if statusErr == nil {
			snap.Peer = status
		}
	}

	if statusErr != nil {
		snap.ProbeOutput = statusErr.Error()
		if allowJoin {
			snap.State = StateDegraded
			snap.Reason = "unable to read netbird status after join"
		} else {
			snap.State = StateNotConfigured
			snap.Reason = "netbird peer is not configured"
		}
		return snap, nil
	}

	if !status.Connected {
		if status.PeerID == "" && status.IPv4 == "" && status.ManagementURL == "" {
			snap.State = StateNotConfigured
			snap.Reason = "netbird peer is not configured"
		} else {
			snap.State = StateDegraded
			snap.Reason = "netbird peer disconnected"
		}
		return snap, nil
	}

	if cfg.RequireService && snap.ServiceDetected && !snap.ServiceRunning {
		snap.State = StateDegraded
		snap.Reason = "netbird service is not running"
		return snap, nil
	}

	if strings.TrimSpace(cfg.Probe.Target) != "" {
		probeOK, probeOutput := c.probe(ctx, cfg.Probe)
		snap.ProbeOK = probeOK
		snap.ProbeOutput = probeOutput
		if !probeOK {
			snap.State = StateDegraded
			snap.Reason = "mesh endpoint probe failed"
			return snap, nil
		}
	}

	snap.State = StateConnected
	snap.Reason = "network ready"
	return snap, nil
}

func (c Client) DetectService(ctx context.Context) (running bool, detected bool, details string) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := exec.LookPath("systemctl"); err == nil {
		if c.commandOK(ctx, "systemctl", "is-active", "--quiet", "netbird") {
			return true, true, "systemctl netbird active"
		}
		if c.commandOK(ctx, "systemctl", "is-active", "--quiet", "netbird.service") {
			return true, true, "systemctl netbird.service active"
		}
		return false, true, "systemctl checked"
	}

	if _, err := exec.LookPath("pgrep"); err == nil {
		if c.commandOK(ctx, "pgrep", "-x", "netbird") || c.commandOK(ctx, "pgrep", "-x", "netbirdd") {
			return true, true, "netbird process found"
		}
		return false, true, "pgrep checked"
	}

	return false, false, "service manager unavailable"
}

func (c Client) runInstall(ctx context.Context, cmd []string) error {
	if len(cmd) == 0 {
		return errors.New("install command is empty")
	}
	args := append([]string(nil), cmd[1:]...)
	installCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(installCtx, cmd[0], args...)
	out, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w output=%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c Client) commandOK(ctx context.Context, name string, args ...string) bool {
	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func (c Client) probe(ctx context.Context, probe ProbeConfig) (bool, string) {
	target := strings.TrimSpace(probe.Target)
	if target == "" {
		return true, ""
	}

	timeout := probe.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	switch normalizedProbeType(probe.Type) {
	case ProbeTypePing:
		return c.pingProbe(ctx, target, timeout)
	default:
		return c.httpProbe(ctx, target, timeout, probe)
	}
}

func (c Client) pingProbe(ctx context.Context, target string, timeout time.Duration) (bool, string) {
	sec := int(timeout.Seconds())
	if sec < 1 {
		sec = 1
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout+2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, "ping", "-c", "1", "-W", strconv.Itoa(sec), target)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, strings.TrimSpace(string(out))
	}
	return true, strings.TrimSpace(string(out))
}

func (c Client) httpProbe(ctx context.Context, target string, timeout time.Duration, probe ProbeConfig) (bool, string) {
	u, err := normalizeURL(target)
	if err != nil {
		return false, err.Error()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err.Error()
	}

	ua := strings.TrimSpace(probe.HTTPUserAgent)
	if ua == "" {
		ua = "nkudo-agent-netbird-check/0.1"
	}
	req.Header.Set("User-Agent", ua)

	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	minCode := probe.HTTPStatusMin
	maxCode := probe.HTTPStatusMax
	if minCode == 0 {
		minCode = 200
	}
	if maxCode == 0 {
		maxCode = 399
	}

	ok := resp.StatusCode >= minCode && resp.StatusCode <= maxCode
	return ok, fmt.Sprintf("status=%d", resp.StatusCode)
}

func normalizeURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("url is empty")
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "http://" + trimmed
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return nil, err
	}
	if u.Host == "" {
		return nil, errors.New("url host is empty")
	}
	return u, nil
}

func normalizedProbeType(v ProbeType) ProbeType {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(ProbeTypePing):
		return ProbeTypePing
	default:
		return ProbeTypeHTTP
	}
}

func (c Client) binary() string {
	if strings.TrimSpace(c.Binary) == "" {
		return "netbird"
	}
	return c.Binary
}

func (c Client) Join(ctx context.Context, setupKey, hostname string) error {
	if strings.TrimSpace(setupKey) == "" {
		return errors.New("netbird setup key required")
	}
	args := []string{"up", "--setup-key", setupKey}
	if hostname != "" {
		args = append(args, "--hostname", hostname)
	}
	cmd := exec.CommandContext(ctx, c.binary(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netbird join failed: %w output=%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c Client) Status(ctx context.Context) (Status, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.binary(), "status", "--json")
	out, err := cmd.CombinedOutput()
	if err == nil {
		var parsed map[string]any
		if jsonErr := json.Unmarshal(out, &parsed); jsonErr == nil {
			return parseJSONStatus(string(out), parsed), nil
		}
	}

	cmd = exec.CommandContext(ctx, c.binary(), "status")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return Status{}, fmt.Errorf("netbird status failed: %w", err)
	}
	raw := strings.TrimSpace(string(out))
	return Status{Connected: strings.Contains(strings.ToLower(raw), "connected"), Raw: raw}, nil
}

func parseJSONStatus(raw string, parsed map[string]any) Status {
	status := Status{Raw: raw}
	if v, ok := lookupString(parsed, "status", "peer_state", "peerState"); ok {
		status.Connected = strings.EqualFold(v, "connected")
	}
	if v, ok := lookupBool(parsed, "connected", "is_connected", "isConnected"); ok {
		status.Connected = v
	}
	if v, ok := lookupString(parsed, "peer_id", "peerID", "peerId"); ok {
		status.PeerID = v
	}
	if v, ok := lookupString(parsed, "ip", "ipv4", "netbird_ip", "netbirdIp"); ok {
		status.IPv4 = strings.TrimSpace(strings.SplitN(v, "/", 2)[0])
	}
	if v, ok := lookupString(parsed, "management_url", "managementURL"); ok {
		status.ManagementURL = v
	}
	if mgmt, ok := parsed["management"].(map[string]any); ok {
		if v, ok := lookupString(mgmt, "url"); ok && status.ManagementURL == "" {
			status.ManagementURL = v
		}
	}
	if v, ok := lookupString(parsed, "network_id", "networkID"); ok {
		status.NetworkID = v
	}
	return status
}

func lookupString(values map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			switch v := raw.(type) {
			case string:
				s := strings.TrimSpace(v)
				if s != "" {
					return s, true
				}
			}
		}
	}
	return "", false
}

func lookupBool(values map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			if v, ok := raw.(bool); ok {
				return v, true
			}
		}
	}
	return false, false
}
