package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/print-svc/internal/pdf"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/storage"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/zpl"
)

const TaskPrintPDF = "print:pdf"

// PrintPayload is the task body enqueued by the service layer.
type PrintPayload struct {
	JobID          string        `json:"job_id"`
	Format         string        `json:"format"` // "pdf" | "zpl"
	LabelData      pdf.LabelData `json:"label_data"`
	Size           string        `json:"size"`
	DPI            int           `json:"dpi"`            // ZPL only: 203 or 300
	ProductID      string        `json:"product_id"`     // optional — triggers print counter
	WorkspaceID    string        `json:"workspace_id"`   // needed for counter call
	ProductSvcURL  string        `json:"product_svc_url"` // label-svc product HTTP base URL
}

// NewPrintTask builds an Asynq task for PDF/ZPL generation.
func NewPrintTask(p PrintPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskPrintPDF, data, asynq.MaxRetry(3)), nil
}

// Handler processes print jobs.
type Handler struct {
	jobs   *repo.PrintJobRepo
	s3     *storage.S3Client
	logger *zap.Logger
}

func NewHandler(jobs *repo.PrintJobRepo, s3 *storage.S3Client, logger *zap.Logger) *Handler {
	return &Handler{jobs: jobs, s3: s3, logger: logger}
}

// ProcessPDF implements asynq.HandlerFunc for TaskPrintPDF (handles both PDF and ZPL).
func (h *Handler) ProcessPDF(ctx context.Context, t *asynq.Task) error {
	var p PrintPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	h.logger.Info("print job started", zap.String("job_id", p.JobID), zap.String("format", p.Format))

	if err := h.jobs.SetProcessing(ctx, p.JobID); err != nil {
		h.logger.Warn("set processing status", zap.Error(err))
	}

	size := resolveSize(p.Size)

	if p.Format == "zpl" {
		return h.processZPL(ctx, p, size)
	}
	return h.processPDF(ctx, p, size)
}

func (h *Handler) processPDF(ctx context.Context, p PrintPayload, size pdf.SizeMM) error {
	pdfBytes, err := pdf.Generate(p.LabelData, size)
	if err != nil {
		_ = h.jobs.SetFailed(ctx, p.JobID, err.Error())
		return fmt.Errorf("pdf generate: %w", err)
	}

	s3Key := fmt.Sprintf("prints/%s/%d.pdf", p.JobID, time.Now().UnixNano())
	if err := h.s3.Upload(ctx, s3Key, pdfBytes); err != nil {
		_ = h.jobs.SetFailed(ctx, p.JobID, err.Error())
		return fmt.Errorf("s3 upload: %w", err)
	}

	if err := h.jobs.SetDone(ctx, p.JobID, s3Key); err != nil {
		return fmt.Errorf("set done: %w", err)
	}
	h.logger.Info("PDF job done", zap.String("job_id", p.JobID), zap.String("s3_key", s3Key))
	h.incrementProductPrint(ctx, p)
	return nil
}

// incrementProductPrint fires a POST to label-svc to bump the product's print counter.
func (h *Handler) incrementProductPrint(ctx context.Context, p PrintPayload) {
	if p.ProductID == "" || p.ProductSvcURL == "" {
		return
	}
	url := fmt.Sprintf("%s/products/%s/print", p.ProductSvcURL, p.ProductID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		h.logger.Warn("increment print: build request", zap.Error(err))
		return
	}
	req.Header.Set("X-Workspace-ID", p.WorkspaceID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.logger.Warn("increment print: call failed", zap.Error(err))
		return
	}
	resp.Body.Close()
}

func (h *Handler) processZPL(ctx context.Context, p PrintPayload, size pdf.SizeMM) error {
	dpi := p.DPI
	if dpi == 0 {
		dpi = zpl.DPI203
	}

	zplStr := zpl.Generate(p.LabelData, size, dpi)

	if err := h.jobs.SetZPLDone(ctx, p.JobID, zplStr); err != nil {
		return fmt.Errorf("set zpl done: %w", err)
	}
	h.logger.Info("ZPL job done", zap.String("job_id", p.JobID))
	h.incrementProductPrint(ctx, p)
	return nil
}

// RegisterMux registers the task handler into an Asynq mux.
func RegisterMux(mux *asynq.ServeMux, h *Handler) {
	mux.HandleFunc(TaskPrintPDF, h.ProcessPDF)
}

// NewAsynqServer builds an Asynq server from a Redis URL.
func NewAsynqServer(redisURL string) *asynq.Server {
	opt, _ := asynq.ParseRedisURI(redisURL)
	return asynq.NewServer(opt, asynq.Config{
		Concurrency: 4,
		Queues:      map[string]int{"default": 1},
	})
}

func resolveSize(s string) pdf.SizeMM {
	switch s {
	case "62x100":
		return pdf.Size62x100
	case "a4":
		return pdf.SizeA4
	default:
		return pdf.Size62x29
	}
}
