package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type contextKey string

const (
	ContextUserID      contextKey = "user_id"
	ContextWorkspaceID contextKey = "workspace_id"
	ContextRole        contextKey = "role"
	ContextEmail       contextKey = "email"
	ContextSuperAdmin  contextKey = "is_superadmin"
)

type Config struct {
	JWTSecret string
}

type claims struct {
	jwt.RegisteredClaims
	WorkspaceID  string `json:"wid"`
	Role         string `json:"role"`
	Email        string `json:"email"`
	IsSuperAdmin bool   `json:"sadm,omitempty"`
}

// Auth validates the Bearer JWT in the Authorization header.
// On success, injects user_id, workspace_id and role into request context.
func Auth(cfg Config, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization header", "UNAUTHORIZED")
				return
			}

			c := &claims{}
			parsed, err := jwt.ParseWithClaims(token, c, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.JWTSecret), nil
			})
			if err != nil || !parsed.Valid {
				writeError(w, http.StatusUnauthorized, "invalid or expired token", "INVALID_TOKEN")
				return
			}

			if c.ExpiresAt != nil && c.ExpiresAt.Time.Before(time.Now()) {
				writeError(w, http.StatusUnauthorized, "token expired", "TOKEN_EXPIRED")
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextUserID, c.Subject)
			ctx = context.WithValue(ctx, ContextWorkspaceID, c.WorkspaceID)
			ctx = context.WithValue(ctx, ContextRole, c.Role)
			ctx = context.WithValue(ctx, ContextEmail, c.Email)
			ctx = context.WithValue(ctx, ContextSuperAdmin, c.IsSuperAdmin)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns a middleware that checks the user has at least the given role.
// Role hierarchy: admin > operator > viewer
func RequireRole(minRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(ContextRole).(string)
			if !hasRole(role, minRole) {
				writeError(w, http.StatusForbidden, "insufficient permissions", "FORBIDDEN")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func hasRole(userRole, minRole string) bool {
	levels := map[string]int{"viewer": 1, "operator": 2, "admin": 3}
	return levels[userRole] >= levels[minRole]
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextUserID).(string)
	return v
}

func WorkspaceIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextWorkspaceID).(string)
	return v
}

func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextRole).(string)
	return v
}

func EmailFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextEmail).(string)
	return v
}

func IsSuperAdminFromCtx(ctx context.Context) bool {
	v, _ := ctx.Value(ContextSuperAdmin).(bool)
	return v
}

// RequireSuperAdmin returns a middleware that allows only platform superadmins.
func RequireSuperAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsSuperAdminFromCtx(r.Context()) {
				writeError(w, http.StatusForbidden, "superadmin access required", "FORBIDDEN")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
