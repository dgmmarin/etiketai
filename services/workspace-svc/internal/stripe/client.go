// Package stripe provides a minimal Stripe API client for workspace billing.
// Uses net/http directly — no SDK dependency needed for the subset of APIs used.
package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const stripeBase = "https://api.stripe.com/v1"

// Client wraps the Stripe REST API.
type Client struct {
	secretKey      string
	webhookSecret  string
	priceStarter   string
	priceBusiness  string
	priceEnterprise string
	httpClient     *http.Client
}

// Config holds Stripe configuration.
type Config struct {
	SecretKey       string
	WebhookSecret   string
	PriceStarter    string
	PriceBusiness   string
	PriceEnterprise string
}

func NewClient(cfg Config) *Client {
	return &Client{
		secretKey:       cfg.SecretKey,
		webhookSecret:   cfg.WebhookSecret,
		priceStarter:    cfg.PriceStarter,
		priceBusiness:   cfg.PriceBusiness,
		priceEnterprise: cfg.PriceEnterprise,
		httpClient:      &http.Client{Timeout: 15 * time.Second},
	}
}

// IsConfigured returns true when a secret key is set.
func (c *Client) IsConfigured() bool { return c.secretKey != "" }

// WebhookSecret returns the webhook signing secret for signature verification.
func (c *Client) WebhookSecret() string { return c.webhookSecret }

// PriceIDForPlan maps a plan name to a Stripe price ID.
func (c *Client) PriceIDForPlan(plan string) (string, error) {
	switch plan {
	case "starter":
		return c.priceStarter, nil
	case "business":
		return c.priceBusiness, nil
	case "enterprise":
		return c.priceEnterprise, nil
	}
	return "", fmt.Errorf("unknown plan: %s", plan)
}

// ─── Checkout ─────────────────────────────────────────────────────────────────

// CheckoutSession is the Stripe response after creating a session.
type CheckoutSession struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// CreateCheckoutSession creates a Stripe Checkout session for the given workspace + plan.
func (c *Client) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (*CheckoutSession, error) {
	params := url.Values{}
	params.Set("mode", "subscription")
	params.Set("line_items[0][price]", priceID)
	params.Set("line_items[0][quantity]", "1")
	params.Set("success_url", successURL)
	params.Set("cancel_url", cancelURL)
	if customerID != "" {
		params.Set("customer", customerID)
	}
	params.Set("allow_promotion_codes", "true")

	var sess CheckoutSession
	if err := c.post(ctx, "/checkout/sessions", params, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// ─── Customer ─────────────────────────────────────────────────────────────────

// Customer is a minimal Stripe customer response.
type Customer struct {
	ID string `json:"id"`
}

// CreateOrGetCustomer creates a Stripe customer for the workspace (idempotent by email).
func (c *Client) CreateOrGetCustomer(ctx context.Context, email, name, workspaceID string) (*Customer, error) {
	params := url.Values{}
	params.Set("email", email)
	params.Set("name", name)
	params.Set("metadata[workspace_id]", workspaceID)

	var cust Customer
	if err := c.post(ctx, "/customers", params, &cust); err != nil {
		return nil, err
	}
	return &cust, nil
}

// ─── Webhook event ────────────────────────────────────────────────────────────

// Event is a minimal Stripe webhook event.
type Event struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// SubscriptionData is the nested object inside checkout.session.completed events.
type SubscriptionData struct {
	Object struct {
		ID             string `json:"id"`
		Customer       string `json:"customer"`
		Status         string `json:"status"`
		CurrentPeriodEnd int64 `json:"current_period_end"`
		Metadata       map[string]string `json:"metadata"`
		Items          struct {
			Data []struct {
				Price struct {
					ID string `json:"id"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
	} `json:"object"`
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

func (c *Client) post(ctx context.Context, path string, params url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		stripeBase+path, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Stripe-Version", "2024-06-20")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("stripe: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var stripeErr struct {
			Error struct{ Message string `json:"message"` } `json:"error"`
		}
		_ = json.Unmarshal(body, &stripeErr)
		return fmt.Errorf("stripe %d: %s", resp.StatusCode, stripeErr.Error.Message)
	}
	return json.Unmarshal(body, out)
}
