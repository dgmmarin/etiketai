package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/print-svc/internal/pdf"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/service"
)

// PrintHandler exposes HTTP endpoints for print job management.
type PrintHandler struct {
	svc    *service.PrintService
	logger *zap.Logger
}

func NewPrintHandler(svc *service.PrintService, logger *zap.Logger) *PrintHandler {
	return &PrintHandler{svc: svc, logger: logger}
}

// CreateJobRequest is the JSON body for POST /jobs.
type CreateJobRequest struct {
	LabelID     string        `json:"label_id"`
	WorkspaceID string        `json:"workspace_id"`
	UserID      string        `json:"user_id"`
	ProductID   string        `json:"product_id"`
	Format      string        `json:"format"`
	Size        string        `json:"size"`
	Copies      int           `json:"copies"`
	PrinterID   string        `json:"printer_id"`
	LabelData   pdf.LabelData `json:"label_data"`
}

// POST /jobs
func (h *PrintHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.LabelID == "" || req.WorkspaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request", "code": "BAD_REQUEST"})
		return
	}

	job, err := h.svc.CreateJob(r.Context(),
		req.WorkspaceID, req.LabelID, req.UserID, req.ProductID,
		req.Format, req.Size, req.Copies, req.PrinterID,
		req.LabelData,
	)
	if err != nil {
		h.logger.Error("create print job", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create job", "code": "INTERNAL_ERROR"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"job_id": job.ID,
		"status": job.Status,
	})
}

// GET /jobs/{id}?workspace_id=xxx
func (h *PrintHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	workspaceID := r.URL.Query().Get("workspace_id")
	if jobID == "" || workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "job_id and workspace_id required", "code": "BAD_REQUEST"})
		return
	}

	job, signedURL, err := h.svc.GetJob(r.Context(), jobID, workspaceID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found", "code": "NOT_FOUND"})
			return
		}
		h.logger.Error("get print job", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	resp := map[string]any{
		"job_id":     job.ID,
		"label_id":   job.LabelID,
		"status":     job.Status,
		"format":     job.Format,
		"size":       job.Size,
		"copies":     job.Copies,
		"created_at": job.CreatedAt,
	}
	if signedURL != "" {
		resp["pdf_url"] = signedURL
	}
	if job.ZPLPayload != "" {
		resp["zpl_payload"] = job.ZPLPayload
	}
	if job.ErrorMsg != "" {
		resp["error"] = job.ErrorMsg
	}

	writeJSON(w, http.StatusOK, resp)
}

// GET /jobs/{id}/pdf-url?workspace_id=xxx
// Returns a fresh presigned S3 URL for a completed PDF job.
func (h *PrintHandler) GetReprintURL(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	workspaceID := r.URL.Query().Get("workspace_id")
	if jobID == "" || workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "job_id and workspace_id required", "code": "BAD_REQUEST"})
		return
	}

	url, err := h.svc.ReprintURL(r.Context(), jobID, workspaceID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found", "code": "NOT_FOUND"})
			return
		}
		h.logger.Error("reprint url", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"job_id": jobID, "pdf_url": url})
}

// GET /health
func Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "print-svc"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
