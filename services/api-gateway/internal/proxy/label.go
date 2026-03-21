package proxy

import (
	"context"

	"google.golang.org/grpc"

	labelv1 "github.com/dgmmarin/etiketai/gen/label/v1"
)

// LabelClient proxies api-gateway → label-svc over gRPC.
type LabelClient struct {
	client labelv1.LabelServiceClient
}

func NewLabelClient(conn *grpc.ClientConn) *LabelClient {
	return &LabelClient{client: labelv1.NewLabelServiceClient(conn)}
}

type UploadRequest struct {
	WorkspaceID string
	UserID      string
	ImageS3Key  string
	MIMEType    string
}

type ListRequest struct {
	WorkspaceID string
	Status      string
	Category    string
	Q           string
	Page        int32
	PerPage     int32
}

func (c *LabelClient) Upload(ctx context.Context, req UploadRequest) (any, error) {
	return c.client.UploadLabel(ctx, &labelv1.UploadLabelRequest{
		WorkspaceId: req.WorkspaceID,
		UserId:      req.UserID,
		ImageS3Key:  req.ImageS3Key,
		MimeType:    req.MIMEType,
	})
}

func (c *LabelClient) GetStatus(ctx context.Context, labelID, workspaceID string) (any, error) {
	return c.client.GetLabelStatus(ctx, &labelv1.LabelStatusRequest{
		LabelId:     labelID,
		WorkspaceId: workspaceID,
	})
}

func (c *LabelClient) UpdateFields(ctx context.Context, labelID, workspaceID, userID string, fields map[string]string, isDraft bool) (any, error) {
	return c.client.UpdateLabelFields(ctx, &labelv1.UpdateFieldsRequest{
		LabelId:     labelID,
		WorkspaceId: workspaceID,
		UserId:      userID,
		Fields:      fields,
		IsDraft:     isDraft,
	})
}

func (c *LabelClient) Confirm(ctx context.Context, labelID, workspaceID, userID string) (any, error) {
	return c.client.ConfirmLabel(ctx, &labelv1.ConfirmLabelRequest{
		LabelId:     labelID,
		WorkspaceId: workspaceID,
		UserId:      userID,
	})
}

func (c *LabelClient) List(ctx context.Context, req ListRequest) (any, error) {
	return c.client.ListLabels(ctx, &labelv1.ListLabelsRequest{
		WorkspaceId: req.WorkspaceID,
		Status:      req.Status,
		Category:    req.Category,
		Q:           req.Q,
		Page:        req.Page,
		PerPage:     req.PerPage,
	})
}

func (c *LabelClient) Delete(ctx context.Context, labelID, workspaceID string) error {
	_, err := c.client.DeleteLabel(ctx, &labelv1.DeleteLabelRequest{
		LabelId:     labelID,
		WorkspaceId: workspaceID,
	})
	return err
}
