package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/dgmmarin/etiketai/services/workspace-svc/internal/repo"
)

// WorkspaceService implements workspace business logic.
type WorkspaceService struct {
	workspaceRepo *repo.WorkspaceRepo
	logger        *zap.Logger
}

func NewWorkspaceService(workspaceRepo *repo.WorkspaceRepo, logger *zap.Logger) *WorkspaceService {
	return &WorkspaceService{
		workspaceRepo: workspaceRepo,
		logger:        logger,
	}
}

// RegisterGRPC registers this service with the gRPC server.
// Full implementation requires generated proto stubs (run 'task gen-proto').
func (s *WorkspaceService) RegisterGRPC(srv *grpc.Server) {
	// workspace_grpc.RegisterWorkspaceServiceServer(srv, s)
}

// CreateWorkspace creates a new workspace with the calling user as admin.
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, ownerUserID, ownerEmail, name, cui string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("workspace name is required")
	}
	ws, err := s.workspaceRepo.Create(ctx, name, cui, ownerUserID, ownerEmail)
	if err != nil {
		s.logger.Error("create workspace failed", zap.Error(err))
		return "", fmt.Errorf("create workspace: %w", err)
	}
	s.logger.Info("workspace created",
		zap.String("workspace_id", ws.ID),
		zap.String("owner", ownerUserID),
	)
	return ws.ID, nil
}

// GetUserWorkspace returns the workspace ID and role for a user identified by email.
func (s *WorkspaceService) GetUserWorkspace(ctx context.Context, email string) (workspaceID, role string, err error) {
	member, err := s.workspaceRepo.GetMemberByEmail(ctx, email)
	if err != nil {
		return "", "", err
	}
	return member.WorkspaceID, member.Role, nil
}

// CheckQuota returns whether the workspace has remaining quota.
func (s *WorkspaceService) CheckQuota(ctx context.Context, workspaceID string) (allowed bool, used int, err error) {
	return s.workspaceRepo.CheckQuota(ctx, workspaceID)
}

// IncrementQuota atomically increments the workspace label count.
func (s *WorkspaceService) IncrementQuota(ctx context.Context, workspaceID string) (bool, error) {
	allowed, _, err := s.workspaceRepo.CheckAndIncrementQuota(ctx, workspaceID)
	return allowed, err
}

// InviteMember generates an invite token and creates a pending member record.
func (s *WorkspaceService) InviteMember(ctx context.Context, workspaceID, email, role, invitedByUserID string) (string, error) {
	token, tokenHash, err := generateInviteToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	if err := s.workspaceRepo.InviteMember(ctx, workspaceID, email, role, tokenHash, invitedByUserID); err != nil {
		return "", fmt.Errorf("invite member: %w", err)
	}
	return token, nil
}

// AcceptInvite accepts an invitation using the raw token.
func (s *WorkspaceService) AcceptInvite(ctx context.Context, token, userID string) (string, string, error) {
	hash := hashToken(token)
	member, err := s.workspaceRepo.AcceptInvite(ctx, hash, userID)
	if err != nil {
		return "", "", fmt.Errorf("accept invite: %w", err)
	}
	return member.WorkspaceID, member.Role, nil
}

// ListMembers returns all active members of a workspace.
func (s *WorkspaceService) ListMembers(ctx context.Context, workspaceID string) ([]repo.WorkspaceMember, error) {
	return s.workspaceRepo.ListMembers(ctx, workspaceID)
}

// RevokeMember removes a member from a workspace.
func (s *WorkspaceService) RevokeMember(ctx context.Context, workspaceID, memberID string) error {
	return s.workspaceRepo.RevokeMember(ctx, workspaceID, memberID)
}

// GetWorkspace returns a workspace by ID.
func (s *WorkspaceService) GetWorkspace(ctx context.Context, workspaceID string) (*repo.Workspace, error) {
	return s.workspaceRepo.GetByID(ctx, workspaceID)
}

// UpdateProfile updates workspace profile fields.
func (s *WorkspaceService) UpdateProfile(ctx context.Context, workspaceID, name, cui, address, phone, logoS3Key string) (*repo.Workspace, error) {
	return s.workspaceRepo.UpdateProfile(ctx, workspaceID, name, cui, address, phone, logoS3Key)
}

// GetSubscription returns subscription info for a workspace.
func (s *WorkspaceService) GetSubscription(ctx context.Context, workspaceID string) (*repo.Workspace, error) {
	return s.workspaceRepo.GetByID(ctx, workspaceID)
}

// ─── Token helpers ────────────────────────────────────────────────────────────

func generateInviteToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	hash = hashToken(raw)
	return raw, hash, nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
