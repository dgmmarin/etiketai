package proxy

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	authv1 "github.com/dgmmarin/etiketai/gen/auth/v1"
)

type jwtClaims struct {
	jwt.RegisteredClaims
	WorkspaceID  string `json:"wid"`
	Role         string `json:"role"`
	Email        string `json:"email"`
	IsSuperAdmin bool   `json:"sadm,omitempty"`
}

// parseJWT decodes a JWT's claims without verifying the signature.
// Intentional: used only on tokens we just minted or received from our own auth-svc.
func parseJWT(accessToken string) jwtClaims {
	c := &jwtClaims{}
	_, _, _ = jwt.NewParser().ParseUnverified(accessToken, c)
	return *c
}

func extractSuperAdmin(accessToken string) bool {
	return parseJWT(accessToken).IsSuperAdmin
}

// AuthClient proxies api-gateway → auth-svc over gRPC.
type AuthClient struct {
	client authv1.AuthServiceClient
}

func NewAuthClient(conn *grpc.ClientConn) *AuthClient {
	return &AuthClient{client: authv1.NewAuthServiceClient(conn)}
}

type RegisterResult struct {
	UserID string
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	UserID       string
	Email        string
	WorkspaceID  string
	Role         string
	IsSuperAdmin bool
}

type RefreshResult struct {
	AccessToken  string
	ExpiresIn    int
	UserID       string
	Email        string
	WorkspaceID  string
	Role         string
	IsSuperAdmin bool
}

func (c *AuthClient) Register(ctx context.Context, email, password string) (*RegisterResult, error) {
	resp, err := c.client.Register(ctx, &authv1.RegisterRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	return &RegisterResult{UserID: resp.UserId}, nil
}

func (c *AuthClient) Login(ctx context.Context, email, password, workspaceID, role string) (*LoginResult, error) {
	// Pass workspace context via gRPC metadata so auth-svc can embed it in the JWT.
	md := metadata.Pairs("x-workspace-id", workspaceID, "x-role", role)
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := c.client.Login(ctx, &authv1.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	result := &LoginResult{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    int(resp.ExpiresIn),
		IsSuperAdmin: extractSuperAdmin(resp.AccessToken),
	}
	if resp.User != nil {
		result.UserID = resp.User.Id
		result.Email = resp.User.Email
		result.WorkspaceID = resp.User.WorkspaceId
		result.Role = resp.User.Role
	}
	return result, nil
}

func (c *AuthClient) Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	// Step 1: call without workspace to get the user's email from the JWT.
	resp, err := c.client.RefreshToken(ctx, &authv1.RefreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return nil, err
	}
	claims := parseJWT(resp.AccessToken)
	return &RefreshResult{
		AccessToken:  resp.AccessToken,
		ExpiresIn:    int(resp.ExpiresIn),
		UserID:       claims.Subject,
		Email:        claims.Email,
		IsSuperAdmin: claims.IsSuperAdmin,
	}, nil
}

// RefreshWithWorkspace re-issues the access token with workspace/role embedded in the JWT.
// Call this after resolving the workspace from workspace-svc using the email obtained from Refresh().
func (c *AuthClient) RefreshWithWorkspace(ctx context.Context, refreshToken, workspaceID, role string) (*RefreshResult, error) {
	md := metadata.Pairs("x-workspace-id", workspaceID, "x-role", role)
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := c.client.RefreshToken(ctx, &authv1.RefreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return nil, err
	}
	claims := parseJWT(resp.AccessToken)
	return &RefreshResult{
		AccessToken:  resp.AccessToken,
		ExpiresIn:    int(resp.ExpiresIn),
		UserID:       claims.Subject,
		Email:        claims.Email,
		WorkspaceID:  claims.WorkspaceID,
		Role:         claims.Role,
		IsSuperAdmin: claims.IsSuperAdmin,
	}, nil
}

func (c *AuthClient) Logout(ctx context.Context, refreshToken string) error {
	_, err := c.client.Logout(ctx, &authv1.LogoutRequest{RefreshToken: refreshToken})
	return err
}

func (c *AuthClient) VerifyEmail(ctx context.Context, token string) error {
	_, err := c.client.VerifyEmail(ctx, &authv1.VerifyEmailRequest{Token: token})
	return err
}

func (c *AuthClient) OAuthGoogle(ctx context.Context, idToken string) (*LoginResult, error) {
	resp, err := c.client.OAuthGoogle(ctx, &authv1.OAuthGoogleRequest{IdToken: idToken})
	if err != nil {
		return nil, err
	}
	result := &LoginResult{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    int(resp.ExpiresIn),
		IsSuperAdmin: extractSuperAdmin(resp.AccessToken),
	}
	if resp.User != nil {
		result.UserID = resp.User.Id
		result.Email = resp.User.Email
		result.WorkspaceID = resp.User.WorkspaceId
		result.Role = resp.User.Role
	}
	return result, nil
}
