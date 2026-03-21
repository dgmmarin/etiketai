package grpchandler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authv1 "github.com/dgmmarin/etiketai/gen/auth/v1"
	"github.com/dgmmarin/etiketai/services/auth-svc/internal/service"
)

// AuthHandler implements authv1.AuthServiceServer.
type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func Register(srv *grpc.Server, svc *service.AuthService) {
	authv1.RegisterAuthServiceServer(srv, NewAuthHandler(svc))
}

func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	result, err := h.svc.Register(ctx, req.Email, req.Password)
	if err != nil {
		return nil, status.Errorf(codes.AlreadyExists, "%s", err.Error())
	}
	return &authv1.RegisterResponse{
		UserId:  result.UserID,
		Message: "Verification email sent",
	}, nil
}

func metaValue(ctx context.Context, key string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(key); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	workspaceID := metaValue(ctx, "x-workspace-id")
	role := metaValue(ctx, "x-role")
	result, err := h.svc.Login(ctx, req.Email, req.Password, workspaceID, role)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "%s", err.Error())
	}
	return &authv1.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    int32(result.ExpiresIn),
		User: &authv1.UserInfo{
			Id:          result.UserID,
			Email:       result.Email,
			WorkspaceId: result.WorkspaceID,
			Role:        result.Role,
		},
	}, nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	result, err := h.svc.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "%s", err.Error())
	}
	return &authv1.RefreshTokenResponse{
		AccessToken: result.AccessToken,
		ExpiresIn:   int32(result.ExpiresIn),
	}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if err := h.svc.Logout(ctx, req.RefreshToken); err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &authv1.LogoutResponse{Success: true}, nil
}

func (h *AuthHandler) VerifyEmail(ctx context.Context, req *authv1.VerifyEmailRequest) (*authv1.VerifyEmailResponse, error) {
	if _, err := h.svc.VerifyEmail(ctx, req.Token); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
	}
	return &authv1.VerifyEmailResponse{Success: true}, nil
}

func (h *AuthHandler) VerifyToken(ctx context.Context, req *authv1.VerifyTokenRequest) (*authv1.VerifyTokenResponse, error) {
	claims, err := h.svc.VerifyToken(ctx, req.AccessToken)
	if err != nil {
		return &authv1.VerifyTokenResponse{Valid: false, Error: err.Error()}, nil
	}
	return &authv1.VerifyTokenResponse{
		Valid: true,
		User: &authv1.UserInfo{
			Id:          claims.Subject, // jwt.RegisteredClaims.Subject = userID
			WorkspaceId: claims.WorkspaceID,
			Role:        claims.Role,
		},
	}, nil
}

func (h *AuthHandler) OAuthGoogle(ctx context.Context, req *authv1.OAuthGoogleRequest) (*authv1.LoginResponse, error) {
	result, err := h.svc.GoogleLogin(ctx, req.IdToken, "", "")
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "%s", err.Error())
	}
	return &authv1.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    int32(result.ExpiresIn),
		User: &authv1.UserInfo{
			Id:          result.UserID,
			Email:       result.Email,
			WorkspaceId: result.WorkspaceID,
			Role:        result.Role,
		},
	}, nil
}
