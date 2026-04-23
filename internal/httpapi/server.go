package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/WIKKIwk/erp_scz_db_reader/internal/store"
)

type searcher interface {
	SearchItems(ctx context.Context, query string, limit int, warehouse string) ([]store.Item, error)
	SearchItemWarehouses(ctx context.Context, itemCode, query string, limit int) ([]store.WarehouseStock, error)
	GetItem(ctx context.Context, itemCode string) (store.ItemDetail, error)
	GetWarehouse(ctx context.Context, warehouse string) (store.Warehouse, error)
}

func NewHandler(s searcher) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("GET /v1/handshake", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"service": "gscale_erp_read",
		})
	})
	mux.HandleFunc("GET /v1/items", func(w http.ResponseWriter, r *http.Request) {
		items, err := s.SearchItems(r.Context(), r.URL.Query().Get("query"), parseLimit(r), r.URL.Query().Get("warehouse"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": items})
	})
	mux.HandleFunc("GET /v1/items/{item_code}", func(w http.ResponseWriter, r *http.Request) {
		itemCode := strings.TrimSpace(r.PathValue("item_code"))
		if itemCode == "" {
			writeError(w, http.StatusBadRequest, "item_code is required")
			return
		}
		item, err := s.GetItem(r.Context(), itemCode)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(strings.ToLower(err.Error()), "item topilmadi") {
				status = http.StatusNotFound
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": item})
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
	mux.HandleFunc("GET /v1/warehouses/{warehouse}", func(w http.ResponseWriter, r *http.Request) {
		warehouse := strings.TrimSpace(r.PathValue("warehouse"))
		if warehouse == "" {
			writeError(w, http.StatusBadRequest, "warehouse is required")
			return
		}
		out, err := s.GetWarehouse(r.Context(), warehouse)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(strings.ToLower(err.Error()), "warehouse topilmadi") {
				status = http.StatusNotFound
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": out})
	})
	return mux
}

func parseLimit(r *http.Request) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 0
	}
	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	if limit <= 0 {
		return 0
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
