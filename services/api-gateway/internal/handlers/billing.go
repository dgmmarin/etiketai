package handlers

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/api-gateway/internal/middleware"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/proxy"
)

type BillingHandler struct {
	billing *proxy.BillingClient
	logger  *zap.Logger
}

func NewBillingHandler(billing *proxy.BillingClient, logger *zap.Logger) *BillingHandler {
	return &BillingHandler{billing: billing, logger: logger}
}

// POST /v1/billing/create-checkout
func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	wid := middleware.WorkspaceIDFromCtx(r.Context())
	email := middleware.EmailFromCtx(r.Context())
	name := email // workspace name not in JWT; billing svc will fetch from DB

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body == nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body", "code": "BAD_REQUEST"})
		return
	}

	result, err := h.billing.CreateCheckout(r.Context(), wid, email, name, body)
	if err != nil {
		h.logger.Error("create checkout", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "billing error", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}
