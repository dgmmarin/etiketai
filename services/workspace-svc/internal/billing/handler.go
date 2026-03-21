// Package billing provides an internal HTTP server for Stripe checkout and webhooks.
package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/workspace-svc/internal/repo"
	stripe "github.com/dgmmarin/etiketai/services/workspace-svc/internal/stripe"
)

// Handler is the billing HTTP handler.
type Handler struct {
	stripe *stripe.Client
	ws     *repo.WorkspaceRepo
	logger *zap.Logger
}

// New creates a billing Handler.
func New(s *stripe.Client, ws *repo.WorkspaceRepo, logger *zap.Logger) *Handler {
	return &Handler{stripe: s, ws: ws, logger: logger}
}

// RegisterMux wires billing and superadmin routes into mux.
func RegisterMux(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("POST /checkout", h.CreateCheckout)
	mux.HandleFunc("POST /webhook", h.Webhook)
	// Superadmin — internal only, called by api-gateway with RequireSuperAdmin guard
	mux.HandleFunc("GET /admin/workspaces", h.ListWorkspaces)
	mux.HandleFunc("GET /admin/workspaces/{id}", h.GetWorkspace)
}

// ─── POST /checkout ────────────────────────────────────────────────────────────

// CreateCheckout creates a Stripe Checkout session.
// Expects headers: X-Workspace-ID, X-Email, X-Name
// Body: {"plan":"starter","success_url":"...","cancel_url":"..."}
func (h *Handler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	if !h.stripe.IsConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Stripe not configured", "code": "NOT_CONFIGURED",
		})
		return
	}

	workspaceID := r.Header.Get("X-Workspace-ID")
	email := r.Header.Get("X-Email")
	name := r.Header.Get("X-Name")

	var body struct {
		Plan       string `json:"plan"`
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Plan == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "plan is required", "code": "BAD_REQUEST"})
		return
	}

	priceID, err := h.stripe.PriceIDForPlan(body.Plan)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "INVALID_PLAN"})
		return
	}

	// Get or update Stripe customer ID.
	ws, err := h.ws.GetByID(r.Context(), workspaceID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "workspace not found", "code": "NOT_FOUND"})
		return
	}

	customerID := ws.StripeCustomerID
	if customerID == "" {
		cust, err := h.stripe.CreateOrGetCustomer(r.Context(), email, name, workspaceID)
		if err != nil {
			h.logger.Error("create stripe customer", zap.Error(err))
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stripe error", "code": "INTERNAL_ERROR"})
			return
		}
		customerID = cust.ID
		if err := h.ws.SetStripeCustomerID(r.Context(), workspaceID, customerID); err != nil {
			h.logger.Warn("save customer id", zap.Error(err))
		}
	}

	sess, err := h.stripe.CreateCheckoutSession(r.Context(), customerID, priceID, body.SuccessURL, body.CancelURL)
	if err != nil {
		h.logger.Error("create checkout session", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stripe error", "code": "INTERNAL_ERROR"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": sess.URL, "session_id": sess.ID})
}

// ─── POST /webhook ─────────────────────────────────────────────────────────────

func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !h.verifySignature(r.Header.Get("Stripe-Signature"), body) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var event stripe.Event
	if err := json.Unmarshal(body, &event); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		h.handleCheckoutCompleted(r, event)
	case "customer.subscription.updated", "customer.subscription.deleted":
		h.handleSubscriptionUpdated(r, event)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleCheckoutCompleted(r *http.Request, event stripe.Event) {
	var data stripe.SubscriptionData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		h.logger.Warn("parse checkout.session.completed", zap.Error(err))
		return
	}
	obj := data.Object
	workspaceID := obj.Metadata["workspace_id"]
	if workspaceID == "" {
		return
	}
	var priceID string
	if len(obj.Items.Data) > 0 {
		priceID = obj.Items.Data[0].Price.ID
	}
	plan := h.planFromPriceID(priceID)
	if err := h.ws.SetSubscription(r.Context(), workspaceID, obj.ID, plan, obj.CurrentPeriodEnd); err != nil {
		h.logger.Error("set subscription", zap.String("workspace", workspaceID), zap.Error(err))
	}
}

func (h *Handler) handleSubscriptionUpdated(r *http.Request, event stripe.Event) {
	var data stripe.SubscriptionData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		h.logger.Warn("parse subscription event", zap.Error(err))
		return
	}
	obj := data.Object
	workspaceID := obj.Metadata["workspace_id"]
	if workspaceID == "" {
		return
	}
	var priceID string
	if len(obj.Items.Data) > 0 {
		priceID = obj.Items.Data[0].Price.ID
	}
	plan := h.planFromPriceID(priceID)
	if event.Type == "customer.subscription.deleted" {
		plan = "free"
	}
	if err := h.ws.SetSubscription(r.Context(), workspaceID, obj.ID, plan, obj.CurrentPeriodEnd); err != nil {
		h.logger.Error("update subscription", zap.String("workspace", workspaceID), zap.Error(err))
	}
}

// planFromPriceID resolves a plan name from a Stripe price ID.
func (h *Handler) planFromPriceID(priceID string) string {
	for _, plan := range []string{"starter", "business", "enterprise"} {
		if pid, err := h.stripe.PriceIDForPlan(plan); err == nil && pid == priceID {
			return plan
		}
	}
	return "free"
}

// verifySignature verifies the Stripe-Signature header using HMAC-SHA256.
func (h *Handler) verifySignature(sigHeader string, body []byte) bool {
	secret := h.stripe.WebhookSecret()
	if secret == "" {
		return true // skip in dev
	}

	// Parse t=...,v1=...
	var ts string
	var v1 string
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts = kv[1]
		case "v1":
			v1 = kv[1]
		}
	}
	if ts == "" || v1 == "" {
		return false
	}

	// Reject timestamps older than 5 minutes.
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || time.Now().Unix()-tsInt > 300 {
		return false
	}

	payload := fmt.Sprintf("%s.%s", ts, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(v1))
}

// ─── GET /admin/workspaces ─────────────────────────────────────────────────────

// ListWorkspaces returns all workspaces (superadmin only, called from api-gateway).
func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	limit, offset := 50, 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	workspaces, total, err := h.ws.ListAll(r.Context(), limit, offset)
	if err != nil {
		h.logger.Error("list workspaces", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	type wsItem struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		CUI             string `json:"cui,omitempty"`
		Plan            string `json:"plan"`
		LabelQuota      int    `json:"label_quota"`
		LabelsUsed      int    `json:"labels_used"`
		CreatedAt       string `json:"created_at"`
	}
	items := make([]wsItem, 0, len(workspaces))
	for _, ws := range workspaces {
		items = append(items, wsItem{
			ID:         ws.ID,
			Name:       ws.Name,
			CUI:        ws.CUI,
			Plan:       ws.Plan,
			LabelQuota: ws.LabelQuotaMonthly,
			LabelsUsed: ws.LabelQuotaUsed,
			CreatedAt:  ws.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": items, "total": total})
}

// ─── GET /admin/workspaces/{id} ────────────────────────────────────────────────

// GetWorkspace returns a single workspace by ID (superadmin only).
func (h *Handler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ws, err := h.ws.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found", "code": "NOT_FOUND"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           ws.ID,
		"name":         ws.Name,
		"cui":          ws.CUI,
		"address":      ws.Address,
		"phone":        ws.Phone,
		"plan":         ws.Plan,
		"label_quota":  ws.LabelQuotaMonthly,
		"labels_used":  ws.LabelQuotaUsed,
		"created_at":   ws.CreatedAt.Format(time.RFC3339),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
