package grpchandler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	workspacev1 "github.com/dgmmarin/etiketai/gen/workspace/v1"
	"github.com/dgmmarin/etiketai/services/workspace-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/workspace-svc/internal/service"
)

// WorkspaceHandler implements workspacev1.WorkspaceServiceServer.
type WorkspaceHandler struct {
	workspacev1.UnimplementedWorkspaceServiceServer
	svc *service.WorkspaceService
}

func NewWorkspaceHandler(svc *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc}
}

func Register(srv *grpc.Server, svc *service.WorkspaceService) {
	workspacev1.RegisterWorkspaceServiceServer(srv, NewWorkspaceHandler(svc))
}

func (h *WorkspaceHandler) CreateWorkspace(ctx context.Context, req *workspacev1.CreateWorkspaceRequest) (*workspacev1.CreateWorkspaceResponse, error) {
	wsID, err := h.svc.CreateWorkspace(ctx, req.OwnerUserId, req.OwnerEmail, req.Name, req.Cui)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &workspacev1.CreateWorkspaceResponse{WorkspaceId: wsID}, nil
}

func (h *WorkspaceHandler) GetMemberByEmail(ctx context.Context, req *workspacev1.GetMemberByEmailRequest) (*workspacev1.GetMemberByEmailResponse, error) {
	wsID, role, err := h.svc.GetUserWorkspace(ctx, req.Email)
	if err != nil {
		// Not found is not an error for login flow — caller handles empty wsID
		return &workspacev1.GetMemberByEmailResponse{Found: false}, nil
	}
	return &workspacev1.GetMemberByEmailResponse{
		WorkspaceId: wsID,
		Role:        role,
		Found:       true,
	}, nil
}

func (h *WorkspaceHandler) CheckQuota(ctx context.Context, req *workspacev1.CheckQuotaRequest) (*workspacev1.QuotaResponse, error) {
	allowed, used, err := h.svc.CheckQuota(ctx, req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &workspacev1.QuotaResponse{Allowed: allowed, QuotaUsed: int32(used)}, nil
}

func (h *WorkspaceHandler) IncrementQuota(ctx context.Context, req *workspacev1.IncrementQuotaRequest) (*workspacev1.QuotaResponse, error) {
	allowed, err := h.svc.IncrementQuota(ctx, req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &workspacev1.QuotaResponse{Allowed: allowed}, nil
}

func (h *WorkspaceHandler) InviteMember(ctx context.Context, req *workspacev1.InviteMemberRequest) (*workspacev1.InviteMemberResponse, error) {
	token, err := h.svc.InviteMember(ctx, req.WorkspaceId, req.Email, req.Role, req.InvitedBy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &workspacev1.InviteMemberResponse{InviteToken: token}, nil
}

func (h *WorkspaceHandler) AcceptInvitation(ctx context.Context, req *workspacev1.AcceptInvitationRequest) (*workspacev1.AcceptInvitationResponse, error) {
	wsID, role, err := h.svc.AcceptInvite(ctx, req.Token, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%s", err.Error())
	}
	return &workspacev1.AcceptInvitationResponse{Success: true, WorkspaceId: wsID, Role: role}, nil
}

func (h *WorkspaceHandler) ListMembers(ctx context.Context, req *workspacev1.ListMembersRequest) (*workspacev1.ListMembersResponse, error) {
	members, err := h.svc.ListMembers(ctx, req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	var out []*workspacev1.Member
	for _, m := range members {
		out = append(out, &workspacev1.Member{
			Id:     m.ID,
			UserId: m.UserID,
			Email:  m.Email,
			Role:   m.Role,
		})
	}
	return &workspacev1.ListMembersResponse{Members: out}, nil
}

func (h *WorkspaceHandler) RevokeMember(ctx context.Context, req *workspacev1.RevokeMemberRequest) (*workspacev1.RevokeMemberResponse, error) {
	if err := h.svc.RevokeMember(ctx, req.WorkspaceId, req.MemberId); err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &workspacev1.RevokeMemberResponse{Success: true}, nil
}

func (h *WorkspaceHandler) GetWorkspace(ctx context.Context, req *workspacev1.GetWorkspaceRequest) (*workspacev1.Workspace, error) {
	ws, err := h.svc.GetWorkspace(ctx, req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workspace not found")
	}
	return repoWorkspaceToProto(ws), nil
}

func (h *WorkspaceHandler) UpdateProfile(ctx context.Context, req *workspacev1.UpdateProfileRequest) (*workspacev1.Workspace, error) {
	ws, err := h.svc.UpdateProfile(ctx, req.WorkspaceId, req.Name, req.Cui, req.Address, req.Phone, req.LogoS3Key)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return repoWorkspaceToProto(ws), nil
}

func (h *WorkspaceHandler) GetSubscription(ctx context.Context, req *workspacev1.GetSubscriptionRequest) (*workspacev1.Subscription, error) {
	ws, err := h.svc.GetSubscription(ctx, req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workspace not found")
	}
	sub := &workspacev1.Subscription{
		Plan:              ws.Plan,
		LabelQuotaMonthly: int32(ws.LabelQuotaMonthly),
		LabelQuotaUsed:    int32(ws.LabelQuotaUsed),
	}
	if ws.SubscriptionPeriodEnd != nil {
		sub.SubscriptionExpires = ws.SubscriptionPeriodEnd.UTC().Format("2006-01-02T15:04:05Z")
	}
	return sub, nil
}

func repoWorkspaceToProto(ws *repo.Workspace) *workspacev1.Workspace {
	return &workspacev1.Workspace{
		Id:                ws.ID,
		Name:              ws.Name,
		Cui:               ws.CUI,
		Plan:              ws.Plan,
		LabelQuotaMonthly: int32(ws.LabelQuotaMonthly),
		LabelQuotaUsed:    int32(ws.LabelQuotaUsed),
		LogoUrl:           ws.LogoS3Key,
		CreatedAt:         ws.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
