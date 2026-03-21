// Package email provides a minimal Resend API client for transactional email.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const resendBaseURL = "https://api.resend.com"

// Client sends emails via the Resend REST API.
type Client struct {
	apiKey     string
	from       string
	httpClient *http.Client
}

// NewClient creates a Resend email client.
// from should be "EtiketAI <noreply@etiketai.ro>" format.
func NewClient(apiKey, from string) *Client {
	return &Client{
		apiKey: apiKey,
		from:   from,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SendParams holds the data for a single email.
type SendParams struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

type resendResponse struct {
	ID    string `json:"id"`
	Error string `json:"message,omitempty"`
}

// Send delivers a single email. Returns the Resend message ID on success.
func (c *Client) Send(ctx context.Context, p SendParams) (string, error) {
	body, err := json.Marshal(resendRequest{
		From:    c.from,
		To:      []string{p.To},
		Subject: p.Subject,
		HTML:    p.HTML,
		Text:    p.Text,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendBaseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("resend http: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result resendResponse
	_ = json.Unmarshal(raw, &result)

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("resend %d: %s", resp.StatusCode, result.Error)
	}
	return result.ID, nil
}

// IsConfigured returns true when an API key has been provided.
func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}
