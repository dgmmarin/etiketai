package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/grpc"

	agentv1 "github.com/dgmmarin/etiketai/gen/agent/v1"
)

// AgentClient proxies api-gateway → agent-svc over gRPC + internal HTTP.
type AgentClient struct {
	client     agentv1.AgentServiceClient
	adminURL   string
	httpClient *http.Client
}

func NewAgentClient(conn *grpc.ClientConn) *AgentClient {
	return NewAgentClientWithAdmin(conn, "")
}

func NewAgentClientWithAdmin(conn *grpc.ClientConn, adminURL string) *AgentClient {
	return &AgentClient{
		client:   agentv1.NewAgentServiceClient(conn),
		adminURL: adminURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *AgentClient) GetAgentConfig(ctx context.Context, workspaceID string) (*agentv1.AgentConfig, error) {
	return c.client.GetAgentConfig(ctx, &agentv1.ConfigRequest{WorkspaceId: workspaceID})
}

func (c *AgentClient) UpdateAgentConfig(ctx context.Context, req *agentv1.UpdateConfigRequest) error {
	_, err := c.client.UpdateAgentConfig(ctx, req)
	return err
}

func (c *AgentClient) TestAgentConfig(ctx context.Context, workspaceID, agentType string) (*agentv1.TestConfigResponse, error) {
	return c.client.TestAgentConfig(ctx, &agentv1.TestConfigRequest{
		WorkspaceId: workspaceID,
		AgentType:   agentType,
	})
}

// GetCallLogs fetches the last N agent call log entries via the admin HTTP API.
func (c *AgentClient) GetCallLogs(ctx context.Context, workspaceID string, limit int) (map[string]any, error) {
	return c.adminGet(ctx, fmt.Sprintf("/internal/call-logs?workspace_id=%s&limit=%d", workspaceID, limit))
}

// GetMetrics fetches aggregated usage metrics via the admin HTTP API.
func (c *AgentClient) GetMetrics(ctx context.Context, workspaceID string) (map[string]any, error) {
	return c.adminGet(ctx, fmt.Sprintf("/internal/metrics?workspace_id=%s", workspaceID))
}

// SetRateLimits updates per-workspace processing rate limits via the admin HTTP API.
func (c *AgentClient) SetRateLimits(ctx context.Context, workspaceID string, body map[string]any) (map[string]any, error) {
	return c.adminPut(ctx, fmt.Sprintf("/internal/workspaces/%s/rate-limits", workspaceID), body)
}

// GetRateLimits fetches per-workspace processing rate limits via the admin HTTP API.
func (c *AgentClient) GetRateLimits(ctx context.Context, workspaceID string) (map[string]any, error) {
	return c.adminGet(ctx, fmt.Sprintf("/internal/workspaces/%s/rate-limits", workspaceID))
}

func (c *AgentClient) adminPut(ctx context.Context, path string, body map[string]any) (map[string]any, error) {
	if c.adminURL == "" {
		return nil, fmt.Errorf("agent admin URL not configured")
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.adminURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agent admin: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent admin returned %d", resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *AgentClient) adminGet(ctx context.Context, path string) (map[string]any, error) {
	if c.adminURL == "" {
		return nil, fmt.Errorf("agent admin URL not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminURL+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agent admin: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent admin returned %d", resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
