package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/api-gateway/internal/middleware"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/proxy"
)

type WorkspaceHandler struct {
	workspace *proxy.WorkspaceClient
	logger    *zap.Logger
}

func NewWorkspaceHandler(workspace *proxy.WorkspaceClient, logger *zap.Logger) *WorkspaceHandler {
	return &WorkspaceHandler{workspace: workspace, logger: logger}
}

// GET /v1/workspace
func (h *WorkspaceHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	wid := middleware.WorkspaceIDFromCtx(r.Context())
	ws, err := h.workspace.GetWorkspace(r.Context(), wid)
	if err != nil {
		h.logger.Error("get workspace", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load workspace", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, ws)
}

// PUT /v1/workspace/profile
func (h *WorkspaceHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	wid := middleware.WorkspaceIDFromCtx(r.Context())
	var req struct {
		Name    string `json:"name"`
		CUI     string `json:"cui"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body", "code": "BAD_REQUEST"})
		return
	}
	ws, err := h.workspace.UpdateProfile(r.Context(), wid, req.Name, req.CUI, req.Address, req.Phone)
	if err != nil {
		h.logger.Error("update profile", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, ws)
}

// GET /v1/workspace/subscription
func (h *WorkspaceHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	wid := middleware.WorkspaceIDFromCtx(r.Context())
	sub, err := h.workspace.GetSubscription(r.Context(), wid)
	if err != nil {
		h.logger.Error("get subscription", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load subscription", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, sub)
}

// POST /v1/workspace/invite
func (h *WorkspaceHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
	wid := middleware.WorkspaceIDFromCtx(r.Context())
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "email is required", "code": "BAD_REQUEST"})
		return
	}
	if req.Role == "" {
		req.Role = "operator"
	}
	token, err := h.workspace.InviteMember(r.Context(), wid, req.Email, req.Role)
	if err != nil {
		h.logger.Error("invite member", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "invite failed", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, map[string]string{"invite_token": token, "status": "invited"})
}

// GET /v1/workspace/invite/{token}
func (h *WorkspaceHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	userID := middleware.UserIDFromCtx(r.Context())
	resp, err := h.workspace.AcceptInvitation(r.Context(), token, userID)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or expired invite", "code": "INVALID_TOKEN"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"workspace_id": resp.WorkspaceId,
		"role":         resp.Role,
	})
}

// GET /v1/workspace/members
func (h *WorkspaceHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	wid := middleware.WorkspaceIDFromCtx(r.Context())
	members, err := h.workspace.ListMembers(r.Context(), wid)
	if err != nil {
		h.logger.Error("list members", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list members", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"data": members})
}

// DELETE /v1/workspace/members/{id}
func (h *WorkspaceHandler) RevokeMember(w http.ResponseWriter, r *http.Request) {
	wid := middleware.WorkspaceIDFromCtx(r.Context())
	memberID := chi.URLParam(r, "id")
	if err := h.workspace.RevokeMember(r.Context(), wid, memberID); err != nil {
		h.logger.Error("revoke member", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "revoke failed", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}
