package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	agentv1 "github.com/dgmmarin/etiketai/gen/agent/v1"
	"github.com/dgmmarin/etiketai/services/label-svc/internal/repo"
)

const TaskProcessLabel = "label:process"

// ProcessLabelPayload is the Asynq task payload.
type ProcessLabelPayload struct {
	LabelID     string `json:"label_id"`
	WorkspaceID string `json:"workspace_id"`
	ImageS3Key  string `json:"image_s3_key"`
}

// NewProcessLabelTask creates an Asynq task for async label processing.
func NewProcessLabelTask(labelID, workspaceID, imageS3Key string) (*asynq.Task, error) {
	payload, err := json.Marshal(ProcessLabelPayload{
		LabelID:     labelID,
		WorkspaceID: workspaceID,
		ImageS3Key:  imageS3Key,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskProcessLabel, payload, asynq.MaxRetry(3)), nil
}

// Handler processes label:process tasks.
type Handler struct {
	labels      *repo.LabelRepo
	agentClient agentv1.AgentServiceClient
	logger      *zap.Logger
}

func NewHandler(labels *repo.LabelRepo, agentAddr string, logger *zap.Logger) (*Handler, error) {
	conn, err := grpc.NewClient(agentAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial agent-svc: %w", err)
	}
	return &Handler{
		labels:      labels,
		agentClient: agentv1.NewAgentServiceClient(conn),
		logger:      logger,
	}, nil
}

// ProcessLabel is the Asynq task handler.
func (h *Handler) ProcessLabel(ctx context.Context, t *asynq.Task) error {
	var p ProcessLabelPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	h.logger.Info("processing label", zap.String("label_id", p.LabelID))

	// 1. Mark as processing
	if err := h.labels.SetStatus(ctx, p.LabelID, p.WorkspaceID, "processing"); err != nil {
		return fmt.Errorf("set status processing: %w", err)
	}

	// 2. Vision: extract fields from image
	visionResp, err := h.agentClient.ProcessVision(ctx, &agentv1.VisionRequest{
		LabelId:        p.LabelID,
		ImageS3Key:     p.ImageS3Key,
		WorkspaceId:    p.WorkspaceID,
		TargetLanguage: "ro",
	})
	if err != nil {
		_ = h.labels.SetStatus(ctx, p.LabelID, p.WorkspaceID, "error")
		return fmt.Errorf("vision agent: %w", err)
	}

	// 3. Translation: translate extracted fields to Romanian
	translResp, err := h.agentClient.ProcessTranslation(ctx, &agentv1.TranslRequest{
		LabelId:     p.LabelID,
		WorkspaceId: p.WorkspaceID,
		FieldsJson:  visionResp.RawJson,
		Category:    extractCategory(visionResp.RawJson),
		SourceLang:  visionResp.DetectedLang,
	})
	if err != nil {
		_ = h.labels.SetStatus(ctx, p.LabelID, p.WorkspaceID, "error")
		return fmt.Errorf("translation agent: %w", err)
	}

	// 4. Validation: compliance score
	validResp, err := h.agentClient.ProcessValidation(ctx, &agentv1.ValidRequest{
		LabelId:       p.LabelID,
		WorkspaceId:   p.WorkspaceID,
		TranslatedJson: translResp.TranslatedJson,
		Category:      extractCategory(visionResp.RawJson),
	})
	if err != nil {
		h.logger.Warn("validation agent failed (non-fatal)", zap.Error(err))
		validResp = &agentv1.ValidResponse{ComplianceScore: 0}
	}

	// 5. Store results
	var aiRaw map[string]any
	_ = json.Unmarshal([]byte(visionResp.RawJson), &aiRaw)

	var translated map[string]*string
	_ = json.Unmarshal([]byte(translResp.TranslatedJson), &translated)

	var missing []map[string]string
	for _, m := range validResp.Missing {
		missing = append(missing, map[string]string{
			"field":    m.Field,
			"severity": m.Severity,
			"message":  m.Message,
		})
	}

	category := extractCategory(visionResp.RawJson)
	if err := h.labels.SetAIResult(ctx,
		p.LabelID, p.WorkspaceID,
		category, visionResp.DetectedLang,
		aiRaw, translated,
		int(validResp.ComplianceScore), missing,
	); err != nil {
		return fmt.Errorf("set ai result: %w", err)
	}

	h.logger.Info("label processed",
		zap.String("label_id", p.LabelID),
		zap.Int32("compliance", validResp.ComplianceScore),
	)
	return nil
}

// extractCategory pulls the category field from the vision raw JSON.
func extractCategory(rawJSON string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &m); err != nil {
		return "other"
	}
	if cat, ok := m["category"].(string); ok && cat != "" {
		return cat
	}
	return "other"
}
