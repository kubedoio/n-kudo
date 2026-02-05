package logstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Entry struct {
	TenantID    string    `json:"tenant_id,omitempty"`
	SiteID      string    `json:"site_id,omitempty"`
	AgentID     string    `json:"agent_id,omitempty"`
	ExecutionID string    `json:"execution_id"`
	ActionID    string    `json:"action_id,omitempty"`
	Sequence    uint64    `json:"sequence"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	EmittedAt   time.Time `json:"emitted_at"`
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
	seq     atomic.Uint64
}

func (c *Client) NextSequence() uint64 {
	return c.seq.Add(1)
}

func (c *Client) Stream(ctx context.Context, entry Entry) error {
	if entry.ExecutionID == "" {
		return fmt.Errorf("execution_id required")
	}
	if entry.EmittedAt.IsZero() {
		entry.EmittedAt = time.Now().UTC()
	}
	if entry.Sequence == 0 {
		entry.Sequence = c.NextSequence()
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/v1/logs", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("log stream status=%d", resp.StatusCode)
	}
	return nil
}
