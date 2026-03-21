package grpchandler

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	labelv1 "github.com/dgmmarin/etiketai/gen/label/v1"
	"github.com/dgmmarin/etiketai/services/label-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/label-svc/internal/service"
)

// LabelHandler implements labelv1.LabelServiceServer.
type LabelHandler struct {
	labelv1.UnimplementedLabelServiceServer
	svc *service.LabelService
}

func NewLabelHandler(svc *service.LabelService) *LabelHandler {
	return &LabelHandler{svc: svc}
}

func Register(srv *grpc.Server, svc *service.LabelService) {
	labelv1.RegisterLabelServiceServer(srv, NewLabelHandler(svc))
}

func (h *LabelHandler) UploadLabel(ctx context.Context, req *labelv1.UploadLabelRequest) (*labelv1.UploadLabelResponse, error) {
	label, err := h.svc.Upload(ctx, req.WorkspaceId, req.UserId, req.ImageS3Key)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &labelv1.UploadLabelResponse{
		LabelId: label.ID,
		Status:  label.Status,
	}, nil
}

func (h *LabelHandler) GetLabelStatus(ctx context.Context, req *labelv1.LabelStatusRequest) (*labelv1.LabelStatusResponse, error) {
	label, err := h.svc.GetStatus(ctx, req.LabelId, req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%s", err.Error())
	}

	// Build fields map with confidence scores from ai_raw_json.
	confidence := extractConfidenceMap(label.AIRawJSON)
	fields := make(map[string]*labelv1.FieldValue, len(label.FieldsTranslated))
	for k, v := range label.FieldsTranslated {
		fv := &labelv1.FieldValue{}
		if v != nil {
			fv.Value = *v
		}
		if c, ok := confidence[k]; ok {
			fv.Confidence = c
		}
	 	fields[k] = fv
	}

	// Build compliance info.
	var complianceInfo *labelv1.ComplianceInfo
	if label.ComplianceScore > 0 || len(label.MissingFields) > 0 {
		var missing []*labelv1.MissingField
		for _, m := range label.MissingFields {
			missing = append(missing, &labelv1.MissingField{
				Field:    m["field"],
				Severity: m["severity"],
			})
		}
		complianceInfo = &labelv1.ComplianceInfo{
			Score:   int32(label.ComplianceScore),
			Missing: missing,
		}
	}

	progress := statusToProgress(label.Status)

	return &labelv1.LabelStatusResponse{
		LabelId:    label.ID,
		Status:     label.Status,
		Progress:   progress,
		Fields:     fields,
		Compliance: complianceInfo,
	}, nil
}

// extractConfidenceMap pulls the per-field confidence scores from ai_raw_json.
func extractConfidenceMap(aiRaw map[string]any) map[string]float32 {
	out := make(map[string]float32)
	if aiRaw == nil {
		return out
	}
	conf, ok := aiRaw["confidence"]
	if !ok {
		return out
	}
	switch v := conf.(type) {
	case map[string]any:
		for field, score := range v {
			switch s := score.(type) {
			case float64:
				out[field] = float32(s)
			case float32:
				out[field] = s
			}
		}
	}
	return out
}

// statusToProgress maps label status to a 0–100 progress indicator.
func statusToProgress(s string) int32 {
	switch s {
	case "draft":
		return 0
	case "processing":
		return 50
	case "ready":
		return 90
	case "confirmed":
		return 100
	default:
		return 0
	}
}

func (h *LabelHandler) UpdateLabelFields(ctx context.Context, req *labelv1.UpdateFieldsRequest) (*labelv1.UpdateFieldsResponse, error) {
	// Convert map[string]string → map[string]*string
	fields := make(map[string]*string, len(req.Fields))
	for k, v := range req.Fields {
		s := v
		fields[k] = &s
	}
	if err := h.svc.UpdateFields(ctx, req.LabelId, req.WorkspaceId, req.UserId, fields); err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &labelv1.UpdateFieldsResponse{LabelId: req.LabelId}, nil
}

func (h *LabelHandler) ConfirmLabel(ctx context.Context, req *labelv1.ConfirmLabelRequest) (*labelv1.ConfirmLabelResponse, error) {
	if err := h.svc.Confirm(ctx, req.LabelId, req.WorkspaceId, req.UserId); err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &labelv1.ConfirmLabelResponse{LabelId: req.LabelId, Status: "confirmed"}, nil
}

func (h *LabelHandler) DeleteLabel(ctx context.Context, req *labelv1.DeleteLabelRequest) (*labelv1.DeleteLabelResponse, error) {
	if err := h.svc.Delete(ctx, req.LabelId, req.WorkspaceId); err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &labelv1.DeleteLabelResponse{Success: true}, nil
}

func (h *LabelHandler) ListLabels(ctx context.Context, req *labelv1.ListLabelsRequest) (*labelv1.ListLabelsResponse, error) {
	filter := repo.ListFilter{
		WorkspaceID: req.WorkspaceId,
		Status:      req.Status,
		Category:    req.Category,
		Query:       req.Q,
		Page:        int(req.Page),
		PerPage:     int(req.PerPage),
	}
	labels, total, err := h.svc.List(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	var items []*labelv1.LabelSummary
	for _, l := range labels {
		items = append(items, &labelv1.LabelSummary{
			Id:        l.ID,
			Status:    l.Status,
			Category:  l.Category,
			CreatedAt: l.CreatedAt.String(),
		})
	}
	return &labelv1.ListLabelsResponse{
		Data: items,
		Pagination: &labelv1.PaginationInfo{
			Total:   int32(total),
			Page:    req.Page,
			PerPage: req.PerPage,
		},
	}, nil
}

func (h *LabelHandler) SetAIResult(ctx context.Context, req *labelv1.SetAIResultRequest) (*labelv1.SetAIResultResponse, error) {
	var aiRaw map[string]any
	if req.AiRawJson != "" {
		_ = json.Unmarshal([]byte(req.AiRawJson), &aiRaw)
	}

	var translated map[string]*string
	if req.FieldsTranslated != "" {
		_ = json.Unmarshal([]byte(req.FieldsTranslated), &translated)
	}

	var missing []map[string]string
	if req.MissingFields != "" {
		_ = json.Unmarshal([]byte(req.MissingFields), &missing)
	}

	err := h.svc.SetAIResult(ctx,
		req.LabelId, "",
		req.Category, req.DetectedLanguage,
		aiRaw, translated, int(req.ComplianceScore), missing,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &labelv1.SetAIResultResponse{Success: true}, nil
}
