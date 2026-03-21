package handlers

import (
	"net/http"
	"time"

	"github.com/dgmmarin/etiketai/services/api-gateway/internal/middleware"
)

func Health(w http.ResponseWriter, r *http.Request) {
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}
