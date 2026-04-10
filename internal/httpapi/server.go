package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gscale_erp_read/internal/store"
)

type searcher interface {
	SearchItems(ctx context.Context, query string, limit int) ([]store.Item, error)
	SearchItemWarehouses(ctx context.Context, itemCode, query string, limit int) ([]store.WarehouseStock, error)
}

func NewHandler(s searcher) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("GET /v1/items", func(w http.ResponseWriter, r *http.Request) {
		items, err := s.SearchItems(r.Context(), r.URL.Query().Get("query"), parseLimit(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": items})
	})
	mux.HandleFunc("GET /v1/items/{item_code}/warehouses", func(w http.ResponseWriter, r *http.Request) {
		itemCode := strings.TrimSpace(r.PathValue("item_code"))
		if itemCode == "" {
			writeError(w, http.StatusBadRequest, "item_code is required")
			return
		}
		stocks, err := s.SearchItemWarehouses(r.Context(), itemCode, r.URL.Query().Get("query"), parseLimit(r))
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(strings.ToLower(err.Error()), "item code is empty") {
				status = http.StatusBadRequest
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": stocks})
	})
	return mux
}

func parseLimit(r *http.Request) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 20
	}
	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 20
	}
	if limit <= 0 {
		return 20
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": strings.TrimSpace(message)})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
