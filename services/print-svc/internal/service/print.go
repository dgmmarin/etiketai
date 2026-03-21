package service

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/print-svc/internal/pdf"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/storage"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/worker"
)

// PrintService handles print job creation and status queries.
type PrintService struct {
	jobs          *repo.PrintJobRepo
	queue         *asynq.Client
	s3            *storage.S3Client
	productSvcURL string
	logger        *zap.Logger
}

func NewPrintService(jobs *repo.PrintJobRepo, queue *asynq.Client, s3 *storage.S3Client, productSvcURL string, logger *zap.Logger) *PrintService {
	return &PrintService{jobs: jobs, queue: queue, s3: s3, productSvcURL: productSvcURL, logger: logger}
}

// CreateJob persists a print job and enqueues async PDF generation.
func (s *PrintService) CreateJob(ctx context.Context, workspaceID, labelID, userID, productID, format, size string, copies int, printerID string, labelData pdf.LabelData) (*repo.PrintJob, error) {
	if format == "" {
		format = "pdf"
	}
	if size == "" {
		size = "62x29"
	}
	if copies < 1 {
		copies = 1
	}

	job, err := s.jobs.Create(ctx, workspaceID, labelID, userID, format, size, copies, printerID)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	task, err := worker.NewPrintTask(worker.PrintPayload{
		JobID:         job.ID,
		Format:        format,
		LabelData:     labelData,
		Size:          size,
		ProductID:     productID,
		WorkspaceID:   workspaceID,
		ProductSvcURL: s.productSvcURL,
	})
	if err != nil {
		return nil, fmt.Errorf("build task: %w", err)
	}

	if _, err = s.queue.EnqueueContext(ctx, task); err != nil {
		s.logger.Error("enqueue print task", zap.String("job_id", job.ID), zap.Error(err))
		// Non-fatal: job is created, worker will be retried manually or via retry policy
	}

	return job, nil
}

// ReprintURL generates a fresh presigned URL for a completed PDF print job.
func (s *PrintService) ReprintURL(ctx context.Context, jobID, workspaceID string) (string, error) {
	job, err := s.jobs.GetByID(ctx, jobID, workspaceID)
	if err != nil {
		return "", err
	}
	if job.Status != repo.StatusDone {
		return "", fmt.Errorf("job %s is not completed (status: %s)", jobID, job.Status)
	}
	if job.PDFS3Key == "" {
		return "", fmt.Errorf("job %s has no PDF (format: %s)", jobID, job.Format)
	}
	if s.s3 == nil {
		return "", fmt.Errorf("S3 not configured")
	}
	return s.s3.PresignGet(ctx, job.PDFS3Key, 15*time.Minute)
}

// GetJob returns a job with an optional pre-signed PDF URL when done.
func (s *PrintService) GetJob(ctx context.Context, jobID, workspaceID string) (*repo.PrintJob, string, error) {
	job, err := s.jobs.GetByID(ctx, jobID, workspaceID)
	if err != nil {
		return nil, "", err
	}

	var signedURL string
	if job.Status == repo.StatusDone && job.PDFS3Key != "" && s.s3 != nil {
		signedURL, err = s.s3.PresignGet(ctx, job.PDFS3Key, 15*time.Minute)
		if err != nil {
			s.logger.Warn("presign pdf url", zap.Error(err))
		}
	}

	return job, signedURL, nil
}
