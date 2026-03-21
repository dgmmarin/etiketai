package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"

	labelv1 "github.com/dgmmarin/etiketai/gen/label/v1"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/middleware"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/proxy"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/storage"
)

// S3Config holds object-storage credentials for the gateway.
type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
}

type LabelHandler struct {
	label  *proxy.LabelClient
	logger *zap.Logger
	s3cfg  S3Config
	s3     *storage.S3Client
}

func NewLabelHandler(label *proxy.LabelClient, s3cfg S3Config, logger *zap.Logger) *LabelHandler {
	s3, err := storage.NewS3Client(context.Background(), storage.Config{
		Endpoint:  s3cfg.Endpoint,
		AccessKey: s3cfg.AccessKey,
		SecretKey: s3cfg.SecretKey,
		Bucket:    s3cfg.Bucket,
		Region:    s3cfg.Region,
	})
	if err != nil {
		logger.Warn("S3 client init failed — uploads will fail", zap.Error(err))
	}
	return &LabelHandler{label: label, logger: logger, s3cfg: s3cfg, s3: s3}
}

// POST /v1/labels/upload
func (h *LabelHandler) Upload(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())
	userID := middleware.UserIDFromCtx(r.Context())

	if err := r.ParseMultipartForm(11 << 20); err != nil { // 11MB max
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large (max 10MB)", "code": "FILE_TOO_LARGE"})
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "image field is required", "code": "MISSING_FILE"})
		return
	}
	defer file.Close()

	// Validate MIME type
	mimeType := header.Header.Get("Content-Type")
	if !isAllowedMIME(mimeType) {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "only JPEG, PNG and PDF files are accepted", "code": "INVALID_FILE_TYPE"})
		return
	}

	// Upload file to S3
	s3Key := fmt.Sprintf("labels/%s/%d%s", workspaceID, time.Now().UnixNano(), filepath.Ext(header.Filename))
	if h.s3 != nil {
		if err = h.s3.Upload(r.Context(), s3Key, mimeType, io.LimitReader(file, 10<<20)); err != nil {
			h.logger.Error("s3 upload", zap.Error(err))
			middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "upload to storage failed", "code": "S3_UPLOAD_ERROR"})
			return
		}
	} else {
		// S3 unavailable — drain and continue (dev fallback)
		if _, err = io.Copy(io.Discard, io.LimitReader(file, 10<<20)); err != nil {
			h.logger.Error("read upload", zap.Error(err))
			middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read file", "code": "READ_ERROR"})
			return
		}
	}

	result, err := h.label.Upload(r.Context(), proxy.UploadRequest{
		WorkspaceID: workspaceID,
		UserID:      userID,
		ImageS3Key:  s3Key,
		MIMEType:    mimeType,
	})
	if err != nil {
		h.logger.Error("label upload", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "upload failed", "code": "UPLOAD_ERROR"})
		return
	}

	middleware.WriteJSON(w, http.StatusCreated, result)
}

// GET /v1/labels/{id}/status
func (h *LabelHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	labelID := chi.URLParam(r, "id")
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())

	result, err := h.label.GetStatus(r.Context(), labelID, workspaceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "label not found", "code": "NOT_FOUND"})
			return
		}
		h.logger.Error("get label status", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, result)
}

// PATCH /v1/labels/{id}/fields
func (h *LabelHandler) UpdateFields(w http.ResponseWriter, r *http.Request) {
	labelID := chi.URLParam(r, "id")
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())
	userID := middleware.UserIDFromCtx(r.Context())

	var fields map[string]string
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body", "code": "BAD_REQUEST"})
		return
	}

	isDraft := r.URL.Query().Get("draft") == "true"
	result, err := h.label.UpdateFields(r.Context(), labelID, workspaceID, userID, fields, isDraft)
	if err != nil {
		h.logger.Error("update label fields", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed", "code": "UPDATE_ERROR"})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, result)
}

// POST /v1/labels/{id}/confirm
func (h *LabelHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	labelID := chi.URLParam(r, "id")
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())
	userID := middleware.UserIDFromCtx(r.Context())

	result, err := h.label.Confirm(r.Context(), labelID, workspaceID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "quota") {
			middleware.WriteJSON(w, http.StatusPaymentRequired, map[string]string{"error": "monthly label quota exceeded", "code": "QUOTA_EXCEEDED"})
			return
		}
		h.logger.Error("confirm label", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "confirm failed", "code": "CONFIRM_ERROR"})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, result)
}

// GET /v1/labels
func (h *LabelHandler) List(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	if perPage < 1 {
		perPage = 50
	}

	result, err := h.label.List(r.Context(), proxy.ListRequest{
		WorkspaceID: workspaceID,
		Status:      q.Get("status"),
		Category:    q.Get("category"),
		Q:           q.Get("q"),
		Page:        int32(page),
		PerPage:     int32(perPage),
	})
	if err != nil {
		h.logger.Error("list labels", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed", "code": "LIST_ERROR"})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, result)
}

// GET /v1/labels/{id}/compliance
func (h *LabelHandler) GetCompliance(w http.ResponseWriter, r *http.Request) {
	labelID := chi.URLParam(r, "id")
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())

	raw, err := h.label.GetStatus(r.Context(), labelID, workspaceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "label not found", "code": "NOT_FOUND"})
			return
		}
		h.logger.Error("get compliance", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	result, ok := raw.(*labelv1.LabelStatusResponse)
	if !ok {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "unexpected response type", "code": "INTERNAL_ERROR"})
		return
	}

	compliance := result.GetCompliance()
	if compliance == nil {
		middleware.WriteJSON(w, http.StatusOK, map[string]any{
			"label_id": labelID,
			"score":    0,
			"missing":  []any{},
			"status":   result.GetStatus(),
		})
		return
	}

	var missing []map[string]string
	for _, m := range compliance.GetMissing() {
		missing = append(missing, map[string]string{
			"field":    m.GetField(),
			"severity": m.GetSeverity(),
		})
	}
	if missing == nil {
		missing = []map[string]string{}
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"label_id": labelID,
		"score":    compliance.GetScore(),
		"missing":  missing,
		"status":   result.GetStatus(),
	})
}

// DELETE /v1/labels/{id}
func (h *LabelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	labelID := chi.URLParam(r, "id")
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())

	if err := h.label.Delete(r.Context(), labelID, workspaceID); err != nil {
		h.logger.Error("delete label", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed", "code": "DELETE_ERROR"})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GET /v1/labels/export?format=csv
func (h *LabelHandler) Export(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())

	// Fetch up to 10 000 labels for export
	result, err := h.label.List(r.Context(), proxy.ListRequest{
		WorkspaceID: workspaceID,
		Status:      r.URL.Query().Get("status"),
		Category:    r.URL.Query().Get("category"),
		Page:        1,
		PerPage:     10000,
	})
	if err != nil {
		h.logger.Error("export labels", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "export failed", "code": "EXPORT_ERROR"})
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=labels.csv")
	w.WriteHeader(http.StatusOK)

	// Write CSV header
	fmt.Fprint(w, "id,status,category,created_at\n")

	// result is *labelv1.ListLabelsResponse — iterate Data
	type listResp interface {
		GetData() interface{ Len() int }
	}

	// Use JSON round-trip to avoid importing labelv1 here
	enc := json.NewEncoder(w)
	_ = enc

	// Type-assert to access Data field via proto reflection workaround:
	// Marshal to JSON then parse rows
	data, err := json.Marshal(result)
	if err != nil {
		return
	}
	var parsed struct {
		Data []struct {
			ID        string `json:"id"`
			Status    string `json:"status"`
			Category  string `json:"category"`
			CreatedAt string `json:"created_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return
	}
	for _, l := range parsed.Data {
		fmt.Fprintf(w, "%s,%s,%s,%s\n", l.ID, l.Status, l.Category, l.CreatedAt)
	}
}

func isAllowedMIME(mime string) bool {
	allowed := map[string]bool{
		"image/jpeg":      true,
		"image/jpg":       true,
		"image/png":       true,
		"application/pdf": true,
	}
	return allowed[strings.ToLower(mime)]
}
