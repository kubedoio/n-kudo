package enroll

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
	seq     atomic.Uint64
}

type EnrollRequest struct {
	EnrollmentToken string            `json:"enrollment_token"`
	AgentVersion    string            `json:"agent_version"`
	RequestedHost   string            `json:"requested_hostname"`
	CSRPEM          string            `json:"csr_pem"`
	Labels          map[string]string `json:"labels,omitempty"`
	Fingerprint     HostFingerprint   `json:"host_fingerprint"`
	BootstrapNonce  string            `json:"bootstrap_nonce"`
}

type HostFingerprint struct {
	MachineIDSHA256  string `json:"machine_id_sha256,omitempty"`
	PrimaryMACSHA256 string `json:"primary_mac_sha256,omitempty"`
}

type EnrollResponse struct {
	TenantID             string `json:"tenant_id"`
	SiteID               string `json:"site_id"`
	HostID               string `json:"host_id"`
	AgentID              string `json:"agent_id"`
	ClientCertificatePEM string `json:"client_certificate_pem"`
	CACertificatePEM     string `json:"ca_certificate_pem"`
	RefreshToken         string `json:"refresh_token"`
	HeartbeatEndpoint    string `json:"heartbeat_endpoint"`
	HeartbeatIntervalSec int    `json:"heartbeat_interval_seconds"`
}

func (c *Client) Enroll(ctx context.Context, req EnrollRequest) (EnrollResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return EnrollResponse{}, fmt.Errorf("marshal enroll request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/v1/enroll", bytes.NewReader(payload))
	if err != nil {
		return EnrollResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return EnrollResponse{}, fmt.Errorf("enroll request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return EnrollResponse{}, fmt.Errorf("enroll failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out EnrollResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return EnrollResponse{}, fmt.Errorf("decode enroll response: %w", err)
	}
	if out.AgentID == "" || out.SiteID == "" {
		return EnrollResponse{}, fmt.Errorf("invalid enroll response: missing identifiers")
	}
	return out, nil
}

func BuildFingerprint() HostFingerprint {
	f := HostFingerprint{}
	if machineID, err := os.ReadFile("/etc/machine-id"); err == nil {
		sum := sha256.Sum256(bytes.TrimSpace(machineID))
		f.MachineIDSHA256 = hex.EncodeToString(sum[:])
	}
	if mac := firstMAC(); mac != "" {
		sum := sha256.Sum256([]byte(mac))
		f.PrimaryMACSHA256 = hex.EncodeToString(sum[:])
	}
	return f
}

func firstMAC() string {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "lo") {
			continue
		}
		mac, err := os.ReadFile("/sys/class/net/" + e.Name() + "/address")
		if err != nil {
			continue
		}
		m := strings.TrimSpace(string(mac))
		if m != "" {
			return m
		}
	}
	return ""
}

func NewNonce() string {
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}

// UnenrollRequest represents a request to unenroll an agent
type UnenrollRequest struct {
	AgentID string `json:"agent_id"`
	Reason  string `json:"reason,omitempty"`
}

// Unenroll sends an unenrollment request to the control plane
func (c *Client) Unenroll(ctx context.Context, agentID string) error {
	req := UnenrollRequest{
		AgentID: agentID,
		Reason:  "user-initiated",
	}
	return c.postJSON(ctx, "/v1/unenroll", req, nil)
}

// RenewRequest represents a certificate renewal request
type RenewRequest struct {
	AgentID       string `json:"agent_id"`
	CSRPEM        string `json:"csr_pem"`
	RefreshToken  string `json:"refresh_token"`
}

// RenewResponse represents the response from a certificate renewal request
type RenewResponse struct {
	ClientCertificatePEM string `json:"client_certificate_pem"`
	CACertificatePEM     string `json:"ca_certificate_pem"`
	ExpiresAt            string `json:"expires_at"`
	RefreshToken         string `json:"refresh_token,omitempty"`
}

// RenewCertificate requests a new certificate from the control plane
func (c *Client) RenewCertificate(ctx context.Context, agentID, csrPEM, refreshToken string) (*RenewResponse, error) {
	req := RenewRequest{
		AgentID:      agentID,
		CSRPEM:       csrPEM,
		RefreshToken: refreshToken,
	}
	var resp RenewResponse
	if err := c.postJSON(ctx, "/v1/renew", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
