package proxy

import (
	"context"

	"google.golang.org/grpc"

	workspacev1 "github.com/dgmmarin/etiketai/gen/workspace/v1"
)

// WorkspaceClient proxies api-gateway → workspace-svc over gRPC.
type WorkspaceClient struct {
	client workspacev1.WorkspaceServiceClient
}

func NewWorkspaceClient(conn *grpc.ClientConn) *WorkspaceClient {
	return &WorkspaceClient{client: workspacev1.NewWorkspaceServiceClient(conn)}
}

type UserWorkspaceInfo struct {
	WorkspaceID string
	Role        string
}

func (c *WorkspaceClient) CreateWorkspace(ctx context.Context, userID, email, name, cui string) (string, error) {
	resp, err := c.client.CreateWorkspace(ctx, &workspacev1.CreateWorkspaceRequest{
		OwnerUserId: userID,
		OwnerEmail:  email,
		Name:        name,
		Cui:         cui,
	})
	if err != nil {
		return "", err
	}
	return resp.WorkspaceId, nil
}

func (c *WorkspaceClient) GetUserWorkspace(ctx context.Context, email string) (UserWorkspaceInfo, error) {
	resp, err := c.client.GetMemberByEmail(ctx, &workspacev1.GetMemberByEmailRequest{Email: email})
	if err != nil || !resp.Found {
		return UserWorkspaceInfo{}, nil
	}
	return UserWorkspaceInfo{WorkspaceID: resp.WorkspaceId, Role: resp.Role}, nil
}

func (c *WorkspaceClient) CheckQuota(ctx context.Context, workspaceID string) (bool, int32, error) {
	resp, err := c.client.CheckQuota(ctx, &workspacev1.CheckQuotaRequest{WorkspaceId: workspaceID})
	if err != nil {
		return false, 0, err
	}
	return resp.Allowed, resp.QuotaUsed, nil
}

func (c *WorkspaceClient) IncrementQuota(ctx context.Context, workspaceID string) error {
	_, err := c.client.IncrementQuota(ctx, &workspacev1.IncrementQuotaRequest{WorkspaceId: workspaceID})
	return err
}

func (c *WorkspaceClient) GetWorkspace(ctx context.Context, workspaceID string) (*workspacev1.Workspace, error) {
	return c.client.GetWorkspace(ctx, &workspacev1.GetWorkspaceRequest{WorkspaceId: workspaceID})
}

func (c *WorkspaceClient) UpdateProfile(ctx context.Context, workspaceID, name, cui, address, phone string) (*workspacev1.Workspace, error) {
	return c.client.UpdateProfile(ctx, &workspacev1.UpdateProfileRequest{
		WorkspaceId: workspaceID,
		Name:        name,
		Cui:         cui,
		Address:     address,
		Phone:       phone,
	})
}

func (c *WorkspaceClient) GetSubscription(ctx context.Context, workspaceID string) (*workspacev1.Subscription, error) {
	return c.client.GetSubscription(ctx, &workspacev1.GetSubscriptionRequest{WorkspaceId: workspaceID})
}

func (c *WorkspaceClient) InviteMember(ctx context.Context, workspaceID, email, role string) (string, error) {
	resp, err := c.client.InviteMember(ctx, &workspacev1.InviteMemberRequest{
		WorkspaceId: workspaceID,
		Email:       email,
		Role:        role,
	})
	if err != nil {
		return "", err
	}
	return resp.InviteToken, nil
}

func (c *WorkspaceClient) AcceptInvitation(ctx context.Context, token, userID string) (*workspacev1.AcceptInvitationResponse, error) {
	return c.client.AcceptInvitation(ctx, &workspacev1.AcceptInvitationRequest{
		Token:  token,
		UserId: userID,
	})
}

func (c *WorkspaceClient) ListMembers(ctx context.Context, workspaceID string) ([]*workspacev1.Member, error) {
	resp, err := c.client.ListMembers(ctx, &workspacev1.ListMembersRequest{WorkspaceId: workspaceID})
	if err != nil {
		return nil, err
	}
	return resp.Members, nil
}

func (c *WorkspaceClient) RevokeMember(ctx context.Context, workspaceID, memberID string) error {
	_, err := c.client.RevokeMember(ctx, &workspacev1.RevokeMemberRequest{
		WorkspaceId: workspaceID,
		MemberId:    memberID,
	})
	return err
}
