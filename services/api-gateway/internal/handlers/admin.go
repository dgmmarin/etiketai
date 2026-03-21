package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	agentv1 "github.com/dgmmarin/etiketai/gen/agent/v1"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/middleware"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/proxy"
)

type AdminHandler struct {
	agent  *proxy.AgentClient
	logger *zap.Logger
}

func NewAdminHandler(agent *proxy.AgentClient, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{agent: agent, logger: logger}
}

// GET /v1/admin/workspaces/{id}/agent-config
func (h *AdminHandler) GetAgentConfig(w http.ResponseWriter, r *http.Request) {
	wid := chi.URLParam(r, "id")
	cfg, err := h.agent.GetAgentConfig(r.Context(), wid)
	if err != nil {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "config not found", "code": "NOT_FOUND"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, cfg)
}

// PUT /v1/admin/workspaces/{id}/agent-config
func (h *AdminHandler) UpdateAgentConfig(w http.ResponseWriter, r *http.Request) {
	wid := chi.URLParam(r, "id")
	updatedBy := middleware.UserIDFromCtx(r.Context())

	var req struct {
		Vision     *struct{ Provider, Model string } `json:"vision"`
		Translation *struct{ Provider, Model string } `json:"translation"`
		Validation  *struct{ Provider, Model string } `json:"validation"`
		Fallback    *struct{ Provider, Model string } `json:"fallback"`
		OllamaURL   string                            `json:"ollama_url"`
		APIKey      string                            `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body", "code": "BAD_REQUEST"})
		return
	}

	grpcReq := &agentv1.UpdateConfigRequest{
		WorkspaceId: wid,
		OllamaUrl:   req.OllamaURL,
		ApiKey:      req.APIKey,
		UpdatedBy:   updatedBy,
	}
	if req.Vision != nil {
		grpcReq.Vision = &agentv1.ProviderConfig{Provider: req.Vision.Provider, Model: req.Vision.Model}
	}
	if req.Translation != nil {
		grpcReq.Translation = &agentv1.ProviderConfig{Provider: req.Translation.Provider, Model: req.Translation.Model}
	}
	if req.Validation != nil {
		grpcReq.Validation = &agentv1.ProviderConfig{Provider: req.Validation.Provider, Model: req.Validation.Model}
	}
	if req.Fallback != nil {
		grpcReq.Fallback = &agentv1.ProviderConfig{Provider: req.Fallback.Provider, Model: req.Fallback.Model}
	}

	if err := h.agent.UpdateAgentConfig(r.Context(), grpcReq); err != nil {
		h.logger.Error("update agent config", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// POST /v1/admin/workspaces/{id}/agent-config/test
func (h *AdminHandler) TestAgentConfig(w http.ResponseWriter, r *http.Request) {
	wid := chi.URLParam(r, "id")
	agentType := r.URL.Query().Get("type")
	if agentType == "" {
		agentType = "vision"
	}
	result, err := h.agent.TestAgentConfig(r.Context(), wid, agentType)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "TEST_FAILED"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

// GET /v1/admin/workspaces/{id}/agent-logs?limit=100
func (h *AdminHandler) GetAgentLogs(w http.ResponseWriter, r *http.Request) {
	wid := chi.URLParam(r, "id")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := parseInt(l); err == nil && n > 0 {
			limit = n
		}
	}
	result, err := h.agent.GetCallLogs(r.Context(), wid, limit)
	if err != nil {
		h.logger.Error("get agent logs", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch logs", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

// GET /v1/admin/workspaces/{id}/metrics
func (h *AdminHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	wid := chi.URLParam(r, "id")
	result, err := h.agent.GetMetrics(r.Context(), wid)
	if err != nil {
		h.logger.Error("get metrics", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch metrics", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

// PUT /v1/admin/workspaces/{id}/rate-limits
func (h *AdminHandler) SetRateLimits(w http.ResponseWriter, r *http.Request) {
	wid := chi.URLParam(r, "id")
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body", "code": "BAD_REQUEST"})
		return
	}
	result, err := h.agent.SetRateLimits(r.Context(), wid, body)
	if err != nil {
		h.logger.Error("set rate limits", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

// GET /v1/admin/workspaces/{id}/rate-limits
func (h *AdminHandler) GetRateLimits(w http.ResponseWriter, r *http.Request) {
	wid := chi.URLParam(r, "id")
	result, err := h.agent.GetRateLimits(r.Context(), wid)
	if err != nil {
		h.logger.Error("get rate limits", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
