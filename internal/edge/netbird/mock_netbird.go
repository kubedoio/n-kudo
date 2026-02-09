package netbird

import (
	"context"
	"encoding/json"
	"time"
)

// MockClient is a test double for the NetBird client.
type MockClient struct {
	StatusResponse string
	StatusError    error
	JoinCalled     bool
	JoinSetupKey   string
	JoinHostname   string
	JoinError      error
	LeaveCalled    bool
	LeaveError     error
}

// NewMockClient creates a new MockClient with default status response.
func NewMockClient() *MockClient {
	return &MockClient{
		StatusResponse: `{
			"daemon": {"state": "running"},
			"signal": {"state": "connected"},
			"management": {"state": "connected"},
			"status": "Connected",
			"peer_id": "mock-peer-123",
			"netbirdIp": "100.64.0.1/16"
		}`,
	}
}

// Status returns a mocked status.
func (m *MockClient) Status(ctx context.Context) (Status, error) {
	if m.StatusError != nil {
		return Status{}, m.StatusError
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(m.StatusResponse), &parsed); err != nil {
		// Return a basic status if parsing fails
		return Status{
			Connected: true,
			Raw:       m.StatusResponse,
		}, nil
	}

	return parseJSONStatus(m.StatusResponse, parsed), nil
}

// Join records that join was called.
func (m *MockClient) Join(ctx context.Context, setupKey, hostname string) error {
	m.JoinCalled = true
	m.JoinSetupKey = setupKey
	m.JoinHostname = hostname
	return m.JoinError
}

// Leave records that leave was called.
func (m *MockClient) Leave() error {
	m.LeaveCalled = true
	return m.LeaveError
}

// MockSnapshot creates a mock Snapshot for testing.
func MockSnapshot(state ConnectivityState) Snapshot {
	return Snapshot{
		State:           state,
		CheckedAt:       time.Now().UTC(),
		CLIInstalled:    true,
		ServiceDetected: true,
		ServiceRunning:  state == StateConnected,
		ProbeOK:         state == StateConnected,
		Reason:          mockReason(state),
		Peer: Status{
			Connected: state == StateConnected,
			PeerID:    "mock-peer-123",
			IPv4:      "100.64.0.1",
			NetworkID: "mock-network",
			Raw:       `{"mock": true}`,
		},
	}
}

func mockReason(state ConnectivityState) string {
	switch state {
	case StateConnected:
		return "network ready"
	case StateDegraded:
		return "degraded state"
	case StateNotConfigured:
		return "not configured"
	default:
		return "unknown"
	}
}

// MockConfig creates a mock Config for testing.
func MockConfig(enabled bool) Config {
	return Config{
		Enabled:        enabled,
		AutoJoin:       true,
		SetupKey:       "mock-setup-key",
		Hostname:       "mock-host",
		RequireService: true,
		Probe: ProbeConfig{
			Type:    ProbeTypeHTTP,
			Target:  "http://localhost:8080/health",
			Timeout: 5 * time.Second,
		},
	}
}
