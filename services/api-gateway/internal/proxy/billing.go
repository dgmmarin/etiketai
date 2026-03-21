package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BillingClient proxies api-gateway → workspace-svc billing HTTP server.
type BillingClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewBillingClient(baseURL string) *BillingClient {
	return &BillingClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// ListWorkspaces returns all workspaces from workspace-svc (superadmin only).
func (c *BillingClient) ListWorkspaces(ctx context.Context, limit, offset int) (map[string]any, error) {
	url := fmt.Sprintf("%s/admin/workspaces?limit=%d&offset=%d", c.baseURL, limit, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("billing: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("billing %d: %s", resp.StatusCode, string(raw))
	}
	var result map[string]any
	_ = json.Unmarshal(raw, &result)
	return result, nil
}

// GetWorkspace returns a single workspace by ID from workspace-svc (superadmin only).
func (c *BillingClient) GetWorkspace(ctx context.Context, id string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/admin/workspaces/"+id, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("billing: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("billing %d: %s", resp.StatusCode, string(raw))
	}
	var result map[string]any
	_ = json.Unmarshal(raw, &result)
	return result, nil
}

// CreateCheckout forwards a checkout creation request to workspace-svc.
// workspaceID, email, name are passed as headers.
func (c *BillingClient) CreateCheckout(ctx context.Context, workspaceID, email, name string, body map[string]any) (map[string]any, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/checkout", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-ID", workspaceID)
	req.Header.Set("X-Email", email)
	req.Header.Set("X-Name", name)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("billing: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("billing %d: %s", resp.StatusCode, string(raw))
	}
	var result map[string]any
	_ = json.Unmarshal(raw, &result)
	return result, nil
}
