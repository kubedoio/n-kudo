package netbird

import (
	"testing"
)

func TestParseJSONStatusConnected(t *testing.T) {
	parsed := map[string]any{
		"status":       "Connected",
		"peer_id":      "peer-123",
		"netbirdIp":    "100.95.1.20/16",
		"network_id":   "nw-01",
		"management":   map[string]any{"url": "https://netbird.example.com"},
		"is_connected": true,
	}

	st := parseJSONStatus("{}", parsed)
	if !st.Connected {
		t.Fatalf("expected connected=true")
	}
	if st.PeerID != "peer-123" {
		t.Fatalf("unexpected peer id: %s", st.PeerID)
	}
	if st.IPv4 != "100.95.1.20" {
		t.Fatalf("unexpected ipv4: %s", st.IPv4)
	}
	if st.NetworkID != "nw-01" {
		t.Fatalf("unexpected network id: %s", st.NetworkID)
	}
	if st.ManagementURL != "https://netbird.example.com" {
		t.Fatalf("unexpected management url: %s", st.ManagementURL)
	}
}

func TestParseJSONStatusDisconnected(t *testing.T) {
	parsed := map[string]any{
		"peerState": "Disconnected",
		"peerID":    "peer-x",
	}

	st := parseJSONStatus("{}", parsed)
	if st.Connected {
		t.Fatalf("expected connected=false")
	}
	if st.PeerID != "peer-x" {
		t.Fatalf("unexpected peer id: %s", st.PeerID)
	}
}

func TestNormalizeURL(t *testing.T) {
	u, err := normalizeURL("10.10.0.5:8080/healthz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := u.String(); got != "http://10.10.0.5:8080/healthz" {
		t.Fatalf("unexpected normalized url: %s", got)
	}
}

func TestNormalizedProbeTypeDefault(t *testing.T) {
	if got := normalizedProbeType(""); got != ProbeTypeHTTP {
		t.Fatalf("expected default probe type http, got %s", got)
	}
	if got := normalizedProbeType("PING"); got != ProbeTypePing {
		t.Fatalf("expected ping, got %s", got)
	}
}

func TestControlPlaneConnected(t *testing.T) {
	if !(Snapshot{State: StateConnected}).ControlPlaneConnected() {
		t.Fatal("expected connected snapshot to map to true")
	}
	if (Snapshot{State: StateDegraded}).ControlPlaneConnected() {
		t.Fatal("expected degraded snapshot to map to false")
	}
}

func TestToControlPlaneStatus(t *testing.T) {
	snap := Snapshot{
		State:          StateConnected,
		Reason:         "network ready",
		ServiceRunning: true,
		ProbeOK:        true,
		ProbeOutput:    "status=200",
		Peer: Status{
			PeerID:    "peer-1",
			IPv4:      "100.90.1.10",
			NetworkID: "nw-abc",
			Raw:       "raw-status",
		},
	}
	out := snap.ToControlPlaneStatus()
	if !out.Connected {
		t.Fatal("expected connected=true")
	}
	if out.State != StateConnected {
		t.Fatalf("unexpected state: %s", out.State)
	}
	if out.PeerID != "peer-1" || out.IPv4 != "100.90.1.10" || out.NetworkID != "nw-abc" {
		t.Fatal("unexpected peer fields")
	}
	if out.ProbeNotes != "status=200" {
		t.Fatalf("unexpected probe notes: %s", out.ProbeNotes)
	}
}
