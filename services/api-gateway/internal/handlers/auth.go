package handlers

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/api-gateway/internal/middleware"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/proxy"
)

type AuthHandler struct {
	auth      *proxy.AuthClient
	workspace *proxy.WorkspaceClient
	logger    *zap.Logger
}

func NewAuthHandler(auth *proxy.AuthClient, workspace *proxy.WorkspaceClient, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, workspace: workspace, logger: logger}
}

// POST /v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email         string `json:"email"`
		Password      string `json:"password"`
		WorkspaceName string `json:"workspace_name"`
		CUI           string `json:"cui"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body", "code": "BAD_REQUEST"})
		return
	}

	if req.Email == "" || req.Password == "" || req.WorkspaceName == "" {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "email, password and workspace_name are required", "code": "VALIDATION_ERROR"})
		return
	}

	result, err := h.auth.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.Error("register failed", zap.Error(err))
		middleware.WriteJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "code": "REGISTER_ERROR"})
		return
	}

	// Create workspace via workspace-svc
	workspaceID, err := h.workspace.CreateWorkspace(r.Context(), result.UserID, req.Email, req.WorkspaceName, req.CUI)
	if err != nil {
		h.logger.Error("create workspace failed", zap.Error(err))
		// User created but workspace failed — log and return partial success
	}

	middleware.WriteJSON(w, http.StatusCreated, map[string]any{
		"user_id":      result.UserID,
		"workspace_id": workspaceID,
		"message":      "Verification email sent",
	})
}

// POST /v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body", "code": "BAD_REQUEST"})
		return
	}

	// Get workspace & role for this user
	wsInfo, err := h.workspace.GetUserWorkspace(r.Context(), req.Email)
	if err != nil {
		h.logger.Warn("workspace lookup failed", zap.Error(err))
	}

	result, err := h.auth.Login(r.Context(), req.Email, req.Password, wsInfo.WorkspaceID, wsInfo.Role)
	if err != nil {
		h.logger.Info("login failed", zap.String("email", req.Email), zap.Error(err))
		middleware.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error(), "code": "INVALID_CREDENTIALS"})
		return
	}

	// Auth-svc JWT doesn't embed workspace (it's resolved at gateway level).
	// Use the workspace info we already fetched.
	if result.WorkspaceID == "" {
		result.WorkspaceID = wsInfo.WorkspaceID
		result.Role = wsInfo.Role
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
		"user": map[string]string{
			"id":           result.UserID,
			"email":        result.Email,
			"workspace_id": result.WorkspaceID,
			"role":         result.Role,
		},
	})
}

// POST /v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "refresh_token is required", "code": "BAD_REQUEST"})
		return
	}

	result, err := h.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		middleware.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid refresh token", "code": "INVALID_TOKEN"})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token": result.AccessToken,
		"expires_in":   result.ExpiresIn,
	})
}

// POST /v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "refresh_token is required", "code": "BAD_REQUEST"})
		return
	}

	if err := h.auth.Logout(r.Context(), req.RefreshToken); err != nil {
		h.logger.Warn("logout error", zap.Error(err))
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GET /v1/auth/verify-email?token=...
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "token is required", "code": "BAD_REQUEST"})
		return
	}

	if err := h.auth.VerifyEmail(r.Context(), token); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or expired token", "code": "INVALID_TOKEN"})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// POST /v1/auth/oauth/google
func (h *AuthHandler) OAuthGoogle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IDToken == "" {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "id_token is required", "code": "BAD_REQUEST"})
		return
	}

	result, err := h.auth.OAuthGoogle(r.Context(), req.IDToken)
	if err != nil {
		h.logger.Warn("google oauth failed", zap.Error(err))
		middleware.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid google token", "code": "UNAUTHORIZED"})
		return
	}

	// Resolve workspace for this user (same as login flow).
	if result.WorkspaceID == "" {
		if info, wsErr := h.workspace.GetUserWorkspace(r.Context(), result.Email); wsErr == nil {
			result.WorkspaceID = info.WorkspaceID
			result.Role = info.Role
		}
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
		"user": map[string]string{
			"id":           result.UserID,
			"email":        result.Email,
			"workspace_id": result.WorkspaceID,
			"role":         result.Role,
		},
	})
}
