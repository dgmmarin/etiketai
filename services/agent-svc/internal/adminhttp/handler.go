// Package adminhttp exposes internal admin HTTP endpoints for agent-svc.
// These are NOT behind JWT — they are on a separate internal port, firewalled
// from the public internet. The api-gateway calls them from inside the cluster.
package adminhttp

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/agent-svc/internal/repo"
)

// RateLimit holds per-workspace processing rate limits.
type RateLimit struct {
	MaxLabelsPerHour int `json:"max_labels_per_hour"`
	MaxTokensPerHour int `json:"max_tokens_per_hour"`
}

// Handler serves internal admin HTTP routes.
type Handler struct {
	logs        *repo.CallLogRepo
	rateLimits  sync.Map // workspace_id → RateLimit
	logger      *zap.Logger
}

func NewHandler(logs *repo.CallLogRepo, logger *zap.Logger) *Handler {
	return &Handler{logs: logs, logger: logger}
}

// NewMux builds the HTTP mux for the internal admin server.
func NewMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /internal/call-logs", h.GetCallLogs)
	mux.HandleFunc("GET /internal/metrics", h.GetMetrics)
	mux.HandleFunc("PUT /internal/workspaces/{id}/rate-limits", h.SetRateLimits)
	mux.HandleFunc("GET /internal/workspaces/{id}/rate-limits", h.GetRateLimits)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}

// GET /internal/call-logs?workspace_id=xxx&limit=100
func (h *Handler) GetCallLogs(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace_id required"})
		return
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	entries, err := h.logs.ListByWorkspace(r.Context(), workspaceID, limit)
	if err != nil {
		h.logger.Error("list call logs", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if entries == nil {
		entries = []repo.CallLogEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"workspace_id": workspaceID,
		"total":        len(entries),
		"logs":         entries,
	})
}

// GET /internal/metrics?workspace_id=xxx
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace_id required"})
		return
	}

	entries, err := h.logs.ListByWorkspace(r.Context(), workspaceID, 1000)
	if err != nil {
		h.logger.Error("aggregate metrics", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Aggregate in Go — avoids complex SQL for now
	var totalCost float64
	var totalTokens int
	var totalCalls, successCalls int
	byType := map[string]int{}
	byProvider := map[string]int{}

	for _, e := range entries {
		totalCalls++
		if e.Success {
			successCalls++
		}
		totalCost += e.CostUSD
		totalTokens += e.TokensInput + e.TokensOutput
		byType[e.AgentType]++
		byProvider[e.Provider]++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"workspace_id":   workspaceID,
		"total_calls":    totalCalls,
		"success_calls":  successCalls,
		"total_cost_usd": totalCost,
		"total_tokens":   totalTokens,
		"calls_by_type":  byType,
		"calls_by_provider": byProvider,
	})
}

// PUT /internal/workspaces/{id}/rate-limits
func (h *Handler) SetRateLimits(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("id")
	if workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace id required"})
		return
	}
	var rl RateLimit
	if err := json.NewDecoder(r.Body).Decode(&rl); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if rl.MaxLabelsPerHour < 0 || rl.MaxTokensPerHour < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limits must be non-negative"})
		return
	}
	h.rateLimits.Store(workspaceID, rl)
	h.logger.Info("rate limits updated",
		zap.String("workspace_id", workspaceID),
		zap.Int("max_labels_per_hour", rl.MaxLabelsPerHour),
		zap.Int("max_tokens_per_hour", rl.MaxTokensPerHour),
	)
	writeJSON(w, http.StatusOK, map[string]any{
		"workspace_id": workspaceID,
		"rate_limits":  rl,
	})
}

// GET /internal/workspaces/{id}/rate-limits
func (h *Handler) GetRateLimits(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("id")
	if workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace id required"})
		return
	}
	rl := RateLimit{}
	if v, ok := h.rateLimits.Load(workspaceID); ok {
		rl = v.(RateLimit)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"workspace_id": workspaceID,
		"rate_limits":  rl,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
