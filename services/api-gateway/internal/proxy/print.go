package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PrintClient proxies api-gateway → print-svc over HTTP.
type PrintClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPrintClient(baseURL string) *PrintClient {
	return &PrintClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PrintLabelData mirrors print-svc's pdf.LabelData.
type PrintLabelData struct {
	ProductName  string `json:"product_name"`
	Manufacturer string `json:"manufacturer"`
	Quantity     string `json:"quantity"`
	ExpiryDate   string `json:"expiry_date"`
	Ingredients  string `json:"ingredients"`
	LotNumber    string `json:"lot_number"`
	Country      string `json:"country"`
	Warnings     string `json:"warnings"`
	Category     string `json:"category"`
}

// CreatePrintJobRequest is sent to POST /jobs on print-svc.
type CreatePrintJobRequest struct {
	LabelID     string         `json:"label_id"`
	WorkspaceID string         `json:"workspace_id"`
	UserID      string         `json:"user_id"`
	Format      string         `json:"format"`
	Size        string         `json:"size"`
	Copies      int            `json:"copies"`
	PrinterID   string         `json:"printer_id"`
	LabelData   PrintLabelData `json:"label_data"`
}

// CreatePrintJobResponse is the response from POST /jobs.
type CreatePrintJobResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// PrintJobStatus is the response from GET /jobs/{id}.
type PrintJobStatus struct {
	JobID     string    `json:"job_id"`
	LabelID   string    `json:"label_id"`
	Status    string    `json:"status"`
	Format    string    `json:"format"`
	Size      string    `json:"size"`
	Copies    int       `json:"copies"`
	PDFURL    string    `json:"pdf_url,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// CreatePrintJob enqueues a PDF generation job in print-svc.
func (c *PrintClient) CreatePrintJob(ctx context.Context, req CreatePrintJobRequest) (*CreatePrintJobResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/jobs", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("print-svc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("print-svc returned %d: %s", resp.StatusCode, errBody["error"])
	}

	var result CreatePrintJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetReprintURL fetches a fresh presigned PDF URL for a completed job.
func (c *PrintClient) GetReprintURL(ctx context.Context, jobID, workspaceID string) (string, error) {
	url := fmt.Sprintf("%s/jobs/%s/pdf-url?workspace_id=%s", c.baseURL, jobID, workspaceID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("print-svc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("not found")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("print-svc returned %d", resp.StatusCode)
	}

	var result struct {
		PDFURL string `json:"pdf_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.PDFURL, nil
}

// GetPrintJob fetches the status of a print job.
func (c *PrintClient) GetPrintJob(ctx context.Context, jobID, workspaceID string) (*PrintJobStatus, error) {
	url := fmt.Sprintf("%s/jobs/%s?workspace_id=%s", c.baseURL, jobID, workspaceID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("print-svc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("print-svc returned %d", resp.StatusCode)
	}

	var result PrintJobStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
