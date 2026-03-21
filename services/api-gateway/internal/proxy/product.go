package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ProductClient proxies api-gateway → label-svc product HTTP server.
type ProductClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewProductClient(baseURL string) *ProductClient {
	return &ProductClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *ProductClient) do(ctx context.Context, method, path, workspaceID string, body any) (map[string]any, int, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = bytes.NewReader(data)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if workspaceID != "" {
		req.Header.Set("X-Workspace-ID", workspaceID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("product-svc: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result, resp.StatusCode, nil
}

func (c *ProductClient) CreateProduct(ctx context.Context, workspaceID string, body map[string]any) (map[string]any, error) {
	result, status, err := c.do(ctx, http.MethodPost, "/products", workspaceID, body)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("product-svc %d: %v", status, result["error"])
	}
	return result, nil
}

func (c *ProductClient) ListProducts(ctx context.Context, workspaceID, q, category string, page, perPage int) (map[string]any, error) {
	path := fmt.Sprintf("/products?workspace_id=%s&q=%s&category=%s&page=%d&per_page=%d",
		workspaceID, q, category, page, perPage)
	result, status, err := c.do(ctx, http.MethodGet, path, workspaceID, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("product-svc %d: %v", status, result["error"])
	}
	return result, nil
}

func (c *ProductClient) GetProduct(ctx context.Context, id, workspaceID string) (map[string]any, error) {
	result, status, err := c.do(ctx, http.MethodGet, "/products/"+id+"?workspace_id="+workspaceID, workspaceID, nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}
	if status >= 400 {
		return nil, fmt.Errorf("product-svc %d: %v", status, result["error"])
	}
	return result, nil
}

func (c *ProductClient) UpdateProduct(ctx context.Context, id, workspaceID string, body map[string]any) (map[string]any, error) {
	result, status, err := c.do(ctx, http.MethodPatch, "/products/"+id, workspaceID, body)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}
	if status >= 400 {
		return nil, fmt.Errorf("product-svc %d: %v", status, result["error"])
	}
	return result, nil
}
