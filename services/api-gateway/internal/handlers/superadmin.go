package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/api-gateway/internal/middleware"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/proxy"
)

// SuperAdminHandler handles platform-level superadmin routes.
// All methods require is_superadmin=true in the JWT (enforced by RequireSuperAdmin middleware).
type SuperAdminHandler struct {
	billing *proxy.BillingClient
	logger  *zap.Logger
}

func NewSuperAdminHandler(billing *proxy.BillingClient, logger *zap.Logger) *SuperAdminHandler {
	return &SuperAdminHandler{billing: billing, logger: logger}
}

// GET /v1/superadmin/workspaces?limit=50&offset=0
func (h *SuperAdminHandler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.billing.ListWorkspaces(r.Context(), limit, offset)
	if err != nil {
		h.logger.Error("superadmin: list workspaces", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

// GET /v1/superadmin/workspaces/{id}
func (h *SuperAdminHandler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	result, err := h.billing.GetWorkspace(r.Context(), id)
	if err != nil {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "workspace not found", "code": "NOT_FOUND"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}
