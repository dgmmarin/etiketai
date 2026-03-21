// Package producthttp exposes an internal HTTP API for the product library.
// The api-gateway calls these endpoints on behalf of authenticated users.
package producthttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/label-svc/internal/repo"
)

type Handler struct {
	products *repo.ProductRepo
	logger   *zap.Logger
}

func NewHandler(products *repo.ProductRepo, logger *zap.Logger) *Handler {
	return &Handler{products: products, logger: logger}
}

// NewMux builds the HTTP mux for the product HTTP server.
func NewMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /products", h.Create)
	mux.HandleFunc("GET /products", h.List)
	mux.HandleFunc("GET /products/{id}", h.Get)
	mux.HandleFunc("PATCH /products/{id}", h.Update)
	mux.HandleFunc("POST /products/{id}/print", h.IncrementPrint)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}

// POST /products
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, errResp("X-Workspace-ID header required"))
		return
	}

	var req struct {
		SKU           string             `json:"sku"`
		Name          string             `json:"name"`
		Category      string             `json:"category"`
		DefaultFields map[string]*string `json:"default_fields"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, errResp("name is required"))
		return
	}

	p, err := h.products.Create(r.Context(), workspaceID, req.SKU, req.Name, req.Category, req.DefaultFields)
	if err != nil {
		h.logger.Error("create product", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, errResp("internal error"))
		return
	}
	writeJSON(w, http.StatusCreated, productJSON(p))
}

// GET /products?q=&category=&page=&per_page=&workspace_id=
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		workspaceID = r.URL.Query().Get("workspace_id")
	}
	if workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, errResp("workspace_id required"))
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	products, total, err := h.products.List(r.Context(), repo.ProductFilter{
		WorkspaceID: workspaceID,
		Query:       r.URL.Query().Get("q"),
		Category:    r.URL.Query().Get("category"),
		Page:        page,
		PerPage:     perPage,
	})
	if err != nil {
		h.logger.Error("list products", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, errResp("internal error"))
		return
	}

	items := make([]map[string]any, len(products))
	for i := range products {
		items[i] = productJSON(&products[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total":    total,
		"products": items,
	})
}

// GET /products/{id}?workspace_id=
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		workspaceID = r.URL.Query().Get("workspace_id")
	}

	p, err := h.products.GetByID(r.Context(), id, workspaceID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errResp("not found"))
			return
		}
		writeJSON(w, http.StatusInternalServerError, errResp("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, productJSON(p))
}

// PATCH /products/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, errResp("X-Workspace-ID header required"))
		return
	}

	var req struct {
		Name          string             `json:"name"`
		Category      string             `json:"category"`
		DefaultFields map[string]*string `json:"default_fields"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid body"))
		return
	}

	if err := h.products.UpdateFields(r.Context(), id, workspaceID, req.DefaultFields, req.Name, req.Category); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errResp("not found"))
			return
		}
		h.logger.Error("update product", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, errResp("internal error"))
		return
	}

	p, _ := h.products.GetByID(r.Context(), id, workspaceID)
	if p != nil {
		writeJSON(w, http.StatusOK, productJSON(p))
	} else {
		writeJSON(w, http.StatusOK, map[string]bool{"updated": true})
	}
}

// POST /products/{id}/print — increment print counter (called by print-svc on job completion)
func (h *Handler) IncrementPrint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	workspaceID := r.Header.Get("X-Workspace-ID")
	if id == "" || workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, errResp("id and X-Workspace-ID required"))
		return
	}
	if err := h.products.IncrementPrintCount(r.Context(), id); err != nil {
		h.logger.Error("increment print count", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, errResp("failed"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func productJSON(p *repo.Product) map[string]any {
	return map[string]any{
		"id":             p.ID,
		"workspace_id":   p.WorkspaceID,
		"sku":            p.SKU,
		"name":           p.Name,
		"category":       p.Category,
		"default_fields": p.DefaultFields,
		"print_count":    p.PrintCount,
		"created_at":     p.CreatedAt,
		"updated_at":     p.UpdatedAt,
	}
}

func errResp(msg string) map[string]string {
	return map[string]string{"error": msg}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
