package service

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/label-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/label-svc/internal/worker"
)

// LabelService orchestrates label operations.
type LabelService struct {
	labels *repo.LabelRepo
	queue  *asynq.Client
	logger *zap.Logger
}

func NewLabelService(labels *repo.LabelRepo, queue *asynq.Client, logger *zap.Logger) *LabelService {
	return &LabelService{labels: labels, queue: queue, logger: logger}
}

// Upload creates a label record and enqueues async AI processing.
func (s *LabelService) Upload(ctx context.Context, workspaceID, userID, imageS3Key string) (*repo.Label, error) {
	if imageS3Key == "" {
		return nil, fmt.Errorf("image_s3_key is required")
	}
	label, err := s.labels.Create(ctx, workspaceID, userID, imageS3Key)
	if err != nil {
		s.logger.Error("create label failed", zap.Error(err))
		return nil, fmt.Errorf("create label: %w", err)
	}
	s.logger.Info("label created", zap.String("label_id", label.ID))

	// Enqueue async processing (vision → translation → validation)
	if s.queue != nil {
		task, err := worker.NewProcessLabelTask(label.ID, workspaceID, imageS3Key)
		if err == nil {
			if _, err = s.queue.EnqueueContext(ctx, task); err != nil {
				s.logger.Warn("enqueue label processing failed", zap.Error(err))
			}
		}
	}
	return label, nil
}

// GetStatus returns the current status and fields of a label.
func (s *LabelService) GetStatus(ctx context.Context, id, workspaceID string) (*repo.Label, error) {
	label, err := s.labels.GetByID(ctx, id, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("get label: %w", err)
	}
	return label, nil
}

// SetAIResult stores the vision + translation pipeline output.
func (s *LabelService) SetAIResult(ctx context.Context, id, workspaceID, category, detectedLang string,
	aiRaw map[string]any, translated map[string]*string, score int, missing []map[string]string) error {

	if err := s.labels.SetAIResult(ctx, id, workspaceID, category, detectedLang, aiRaw, translated, score, missing); err != nil {
		s.logger.Error("set AI result failed", zap.String("label_id", id), zap.Error(err))
		return fmt.Errorf("set ai result: %w", err)
	}
	return nil
}

// UpdateFields applies operator edits to translated fields.
func (s *LabelService) UpdateFields(ctx context.Context, id, workspaceID, userID string, fields map[string]*string) error {
	if err := s.labels.UpdateFields(ctx, id, workspaceID, fields); err != nil {
		return fmt.Errorf("update fields: %w", err)
	}
	go func() {
		_ = s.labels.AppendAuditLog(context.Background(), id, userID, "field_edited", nil, nil)
	}()
	return nil
}

// Confirm marks a label as confirmed (ready for print).
func (s *LabelService) Confirm(ctx context.Context, id, workspaceID, userID string) error {
	if err := s.labels.Confirm(ctx, id, workspaceID, userID); err != nil {
		return fmt.Errorf("confirm label: %w", err)
	}
	s.logger.Info("label confirmed", zap.String("label_id", id))
	return nil
}

// List returns paginated labels for a workspace.
func (s *LabelService) List(ctx context.Context, f repo.ListFilter) ([]repo.Label, int, error) {
	return s.labels.List(ctx, f)
}

// Delete removes a non-confirmed label.
func (s *LabelService) Delete(ctx context.Context, id, workspaceID string) error {
	return s.labels.Delete(ctx, id, workspaceID)
}
