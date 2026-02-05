package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/n-kudo/n-kudo-edge/pkg/executor"
	"github.com/n-kudo/n-kudo-edge/pkg/hostfacts"
	"github.com/n-kudo/n-kudo-edge/pkg/netbird"
	"github.com/n-kudo/n-kudo-edge/pkg/state"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

type HeartbeatRequest struct {
	TenantID       string             `json:"tenant_id"`
	SiteID         string             `json:"site_id"`
	HostID         string             `json:"host_id"`
	AgentID        string             `json:"agent_id"`
	SentAt         time.Time          `json:"sent_at"`
	HostFacts      hostfacts.Facts    `json:"host_facts"`
	NetBirdStatus  netbird.Status     `json:"netbird_status"`
	MicroVMs       []state.MicroVM    `json:"microvms"`
}

type HeartbeatResponse struct {
	NextHeartbeatSeconds int             `json:"next_heartbeat_seconds"`
	PendingPlans         []executor.Plan `json:"pending_plans"`
}

func (c *Client) Heartbeat(ctx context.Context, req HeartbeatRequest) (HeartbeatResponse, error) {
	var out HeartbeatResponse
	if req.SentAt.IsZero() {
		req.SentAt = time.Now().UTC()
	}
	if err := c.postJSON(ctx, "/v1/heartbeat", req, &out); err != nil {
		return HeartbeatResponse{}, err
	}
	return out, nil
}

func (c *Client) FetchPlans(ctx context.Context, siteID, agentID string) ([]executor.Plan, error) {
	var out struct {
		Plans []executor.Plan `json:"plans"`
	}
	path := fmt.Sprintf("/v1/plans/next?site_id=%s&agent_id=%s", siteID, agentID)
	if err := c.getJSON(ctx, path, &out); err != nil {
		return nil, err
	}
	return out.Plans, nil
}

func (c *Client) ReportPlanResult(ctx context.Context, result executor.PlanResult) error {
	return c.postJSON(ctx, "/v1/executions/result", result, nil)
}

func (c *Client) postJSON(ctx context.Context, path string, reqBody any, out any) error {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request %s failed status=%d body=%s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request %s failed status=%d body=%s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}
