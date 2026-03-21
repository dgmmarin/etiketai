package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	labelv1 "github.com/dgmmarin/etiketai/gen/label/v1"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/middleware"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/proxy"
)

type PrintHandler struct {
	print  *proxy.PrintClient
	label  *proxy.LabelClient
	logger *zap.Logger
}

func NewPrintHandler(print *proxy.PrintClient, label *proxy.LabelClient, logger *zap.Logger) *PrintHandler {
	return &PrintHandler{print: print, label: label, logger: logger}
}

// POST /v1/labels/{id}/print/pdf
func (h *PrintHandler) CreatePrintJob(w http.ResponseWriter, r *http.Request) {
	labelID := chi.URLParam(r, "id")
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())
	userID := middleware.UserIDFromCtx(r.Context())

	var req struct {
		Format    string `json:"format"`
		Size      string `json:"size"`
		Copies    int    `json:"copies"`
		PrinterID string `json:"printer_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Fetch label fields to populate PDF content
	statusAny, err := h.label.GetStatus(r.Context(), labelID, workspaceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "label not found", "code": "NOT_FOUND"})
			return
		}
		h.logger.Error("fetch label for print", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch label", "code": "INTERNAL_ERROR"})
		return
	}

	copies := req.Copies
	if copies < 1 {
		copies = 1
	}

	result, err := h.print.CreatePrintJob(r.Context(), proxy.CreatePrintJobRequest{
		LabelID:     labelID,
		WorkspaceID: workspaceID,
		UserID:      userID,
		Format:      req.Format,
		Size:        req.Size,
		Copies:      copies,
		PrinterID:   req.PrinterID,
		LabelData:   extractLabelData(statusAny),
	})
	if err != nil {
		h.logger.Error("create print job", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "print job creation failed", "code": "PRINT_ERROR"})
		return
	}

	middleware.WriteJSON(w, http.StatusAccepted, map[string]string{
		"job_id": result.JobID,
		"status": result.Status,
	})
}

// GET /v1/labels/{id}/print/pdf/{job_id}
func (h *PrintHandler) GetPrintJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "job_id")
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())

	job, err := h.print.GetPrintJob(r.Context(), jobID, workspaceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "job not found", "code": "NOT_FOUND"})
			return
		}
		h.logger.Error("get print job", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, job)
}

// GET /v1/labels/{id}/print/pdf/{job_id}/url
func (h *PrintHandler) GetReprintURL(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "job_id")
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())

	url, err := h.print.GetReprintURL(r.Context(), jobID, workspaceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "job not found", "code": "NOT_FOUND"})
			return
		}
		h.logger.Error("get reprint url", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]string{"job_id": jobID, "pdf_url": url})
}

// GET /v1/labels/{id}/print/pdf/{job_id}/stream
// Server-Sent Events stream for real-time print job status updates (T-1303).
// The client receives events until the job reaches a terminal state (ready/failed).
func (h *PrintHandler) StreamPrintStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "job_id")
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())

	flusher, ok := w.(http.Flusher)
	if !ok {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	send := func(event string, data any) {
		b, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
		flusher.Flush()
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			job, err := h.print.GetPrintJob(r.Context(), jobID, workspaceID)
			if err != nil {
				send("error", map[string]string{"error": err.Error()})
				return
			}
			send("status", job)
			// Terminal states — stop streaming.
			if job.Status == "ready" || job.Status == "failed" || job.Status == "printed" {
				return
			}
		}
	}
}

// extractLabelData pulls translated field values from a LabelStatusResponse.
func extractLabelData(statusAny any) proxy.PrintLabelData {
	resp, ok := statusAny.(*labelv1.LabelStatusResponse)
	if !ok || resp == nil {
		return proxy.PrintLabelData{}
	}
	get := func(key string) string {
		if fv, ok := resp.Fields[key]; ok && fv != nil {
			return fv.Value
		}
		return ""
	}
	return proxy.PrintLabelData{
		ProductName:  get("product_name"),
		Manufacturer: get("manufacturer"),
		Quantity:     get("quantity"),
		ExpiryDate:   get("expiry_date"),
		Ingredients:  get("ingredients"),
		LotNumber:    get("lot_number"),
		Country:      get("country_of_origin"),
		Warnings:     get("warnings"),
	}
}
