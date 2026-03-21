package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/api-gateway/internal/middleware"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/proxy"
)

type ProductHandler struct {
	product *proxy.ProductClient
	logger  *zap.Logger
}

func NewProductHandler(product *proxy.ProductClient, logger *zap.Logger) *ProductHandler {
	return &ProductHandler{product: product, logger: logger}
}

// POST /v1/products
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body == nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body", "code": "BAD_REQUEST"})
		return
	}
	result, err := h.product.CreateProduct(r.Context(), workspaceID, body)
	if err != nil {
		h.logger.Error("create product", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, result)
}

// GET /v1/products
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())
	q := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	result, err := h.product.ListProducts(r.Context(), workspaceID, q, category, page, perPage)
	if err != nil {
		h.logger.Error("list products", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

// GET /v1/products/{id}
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())
	result, err := h.product.GetProduct(r.Context(), id, workspaceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found", "code": "NOT_FOUND"})
			return
		}
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

// PATCH /v1/products/{id}
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := middleware.WorkspaceIDFromCtx(r.Context())
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body", "code": "BAD_REQUEST"})
		return
	}
	result, err := h.product.UpdateProduct(r.Context(), id, workspaceID, body)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found", "code": "NOT_FOUND"})
			return
		}
		h.logger.Error("update product", zap.Error(err))
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed", "code": "INTERNAL_ERROR"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}
