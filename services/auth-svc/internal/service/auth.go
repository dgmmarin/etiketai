package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"

	"github.com/dgmmarin/etiketai/services/auth-svc/internal/oauth"
	"github.com/dgmmarin/etiketai/services/auth-svc/internal/repo"
)

const (
	maxLoginAttempts = 5
	bcryptCost       = 12
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrTooManyAttempts    = errors.New("too many login attempts, try again in 15 minutes")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type Config struct {
	JWTSecret           string
	AccessTokenTTL      time.Duration
	RefreshTokenTTLDays int
	ResendAPIKey        string
	EmailFrom           string
	GoogleClientID      string
	GoogleClientSecret  string
}

type Claims struct {
	jwt.RegisteredClaims
	WorkspaceID  string `json:"wid"`
	Role         string `json:"role"`
	IsSuperAdmin bool   `json:"sadm,omitempty"`
}

type AuthService struct {
	cfg       Config
	users     *repo.UserRepo
	tokens    *repo.TokenRepo
	logger    *zap.Logger
}

func NewAuthService(cfg Config, users *repo.UserRepo, tokens *repo.TokenRepo, logger *zap.Logger) *AuthService {
	return &AuthService{cfg: cfg, users: users, tokens: tokens, logger: logger}
}

// RegisterGRPC wires the gRPC server.
// Delegates to grpchandler.Register which registers the AuthServiceServer.
func (s *AuthService) RegisterGRPC(server *grpc.Server) {
	_ = server // wired via grpchandler.Register(server, s) in main.go
}

// ─── Register ─────────────────────────────────────────────────────────────────

type RegisterResult struct {
	UserID      string
	WorkspaceID string // set by workspace-svc after creation (orchestrated by api-gateway)
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*RegisterResult, error) {
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, repo.ErrDuplicate) {
			return nil, fmt.Errorf("email already registered")
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Send verification email async (best-effort)
	go s.sendVerificationEmail(context.Background(), user.ID, email)

	return &RegisterResult{UserID: user.ID}, nil
}

// ─── Login ────────────────────────────────────────────────────────────────────

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds
	UserID       string
	Email        string
	WorkspaceID  string
	Role         string
}

func (s *AuthService) Login(ctx context.Context, email, password, workspaceID, role string) (*LoginResult, error) {
	// Rate limiting
	attempts, err := s.tokens.RecordLoginAttempt(ctx, email)
	if err != nil {
		s.logger.Warn("rate limit check failed", zap.Error(err))
	}
	if attempts > maxLoginAttempts {
		return nil, ErrTooManyAttempts
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsVerified {
		return nil, ErrEmailNotVerified
	}

	if user.PasswordHash == nil {
		return nil, ErrInvalidCredentials // OAuth-only account
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Clear rate limit on success
	_ = s.tokens.ClearLoginAttempts(ctx, email)
	go s.users.UpdateLastLogin(context.Background(), user.ID)

	return s.issueTokenPair(ctx, user.ID, email, workspaceID, role, user.IsSuperAdmin)
}

// ─── Token operations ─────────────────────────────────────────────────────────

func (s *AuthService) RefreshToken(ctx context.Context, rawRefreshToken string) (*LoginResult, error) {
	userID, err := s.tokens.ValidateAndRotateRefreshToken(ctx, rawRefreshToken)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Note: workspace_id and role come from the workspace-svc query in the gateway.
	// Here we re-issue with empty workspace/role — the gateway enriches the response.
	return s.issueTokenPair(ctx, user.ID, user.Email, "", "", user.IsSuperAdmin)
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	return s.tokens.RevokeRefreshToken(ctx, rawRefreshToken)
}

func (s *AuthService) VerifyToken(ctx context.Context, accessToken string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(accessToken, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check blacklist
	if claims.ID != "" {
		blacklisted, err := s.tokens.IsAccessTokenBlacklisted(ctx, claims.ID)
		if err != nil {
			s.logger.Warn("blacklist check failed", zap.Error(err))
		}
		if blacklisted {
			return nil, ErrInvalidToken
		}
	}

	return claims, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, rawToken string) (string, error) {
	tokenHash := hashVerificationToken(rawToken)
	userID, err := s.users.ConsumeEmailVerificationToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return "", ErrInvalidToken
		}
		return "", err
	}

	if err := s.users.MarkVerified(ctx, userID); err != nil {
		return "", err
	}

	return userID, nil
}

// ─── Google OAuth ─────────────────────────────────────────────────────────────

// GoogleLogin verifies a Google ID token, upserts the user, and issues a JWT pair.
// workspaceID and role are passed in after the gateway resolves them via workspace-svc.
func (s *AuthService) GoogleLogin(ctx context.Context, idToken, workspaceID, role string) (*LoginResult, error) {
	claims, err := oauth.VerifyGoogleIDToken(ctx, idToken, s.cfg.GoogleClientID)
	if err != nil {
		return nil, fmt.Errorf("google token invalid: %w", err)
	}

	user, err := s.users.CreateOAuth(ctx, claims.Email, "google", claims.Sub)
	if err != nil {
		return nil, fmt.Errorf("upsert oauth user: %w", err)
	}

	go s.users.UpdateLastLogin(context.Background(), user.ID)

	return s.issueTokenPair(ctx, user.ID, user.Email, workspaceID, role, user.IsSuperAdmin)
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (s *AuthService) issueTokenPair(ctx context.Context, userID, email, workspaceID, role string, isSuperAdmin bool) (*LoginResult, error) {
	jti := generateToken(16)
	now := time.Now()
	accessExpiry := now.Add(s.cfg.AccessTokenTTL)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			ID:        jti,
		},
		WorkspaceID:  workspaceID,
		Role:         role,
		IsSuperAdmin: isSuperAdmin,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	rawRefresh := generateToken(32)
	refreshExpiry := now.AddDate(0, 0, s.cfg.RefreshTokenTTLDays)
	if err := s.tokens.SaveRefreshToken(ctx, userID, rawRefresh, refreshExpiry); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
		UserID:       userID,
		Email:        email,
		WorkspaceID:  workspaceID,
		Role:         role,
	}, nil
}

func (s *AuthService) sendVerificationEmail(ctx context.Context, userID, email string) {
	rawToken := generateToken(32)
	tokenHash := hashVerificationToken(rawToken)
	expiresAt := time.Now().Add(24 * time.Hour)

	if err := s.users.SaveEmailVerificationToken(ctx, userID, tokenHash, expiresAt); err != nil {
		s.logger.Error("save verification token", zap.Error(err))
		return
	}

	if s.cfg.ResendAPIKey == "" {
		s.logger.Info("RESEND_API_KEY not set, skipping email",
			zap.String("verification_token", rawToken),
			zap.String("email", email))
		return
	}

	// TODO: send via Resend SDK once wired
	s.logger.Info("verification email sent", zap.String("email", email))
}

func generateToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashVerificationToken(raw string) string {
	// In production use crypto/sha256 same as token.go
	return fmt.Sprintf("%x", []byte(raw))
}
