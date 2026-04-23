package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/WIKKIwk/erp_scz_db_reader/internal/store"
)

type fakeSearcher struct {
	items  []store.Item
	stocks []store.WarehouseStock
	item   store.ItemDetail
	wh     store.Warehouse
	search func(context.Context, string, int, string) ([]store.Item, error)
}

func (f fakeSearcher) SearchItems(ctx context.Context, query string, limit int, warehouse string) ([]store.Item, error) {
	if f.search != nil {
		return f.search(ctx, query, limit, warehouse)
	}
	return f.items, nil
}

func (f fakeSearcher) SearchItemWarehouses(ctx context.Context, itemCode, query string, limit int) ([]store.WarehouseStock, error) {
	return f.stocks, nil
}

func (f fakeSearcher) GetItem(ctx context.Context, itemCode string) (store.ItemDetail, error) {
	return f.item, nil
}

func (f fakeSearcher) GetWarehouse(ctx context.Context, warehouse string) (store.Warehouse, error) {
	return f.wh, nil
}

func TestItemsEndpoint(t *testing.T) {
	h := NewHandler(fakeSearcher{
		search: func(_ context.Context, query string, limit int, warehouse string) ([]store.Item, error) {
			if query != "itm" {
				t.Fatalf("query = %q", query)
			}
			if limit != 10 {
				t.Fatalf("limit = %d", limit)
			}
			if warehouse != "Stores - A" {
				t.Fatalf("warehouse = %q", warehouse)
			}
			return []store.Item{{ItemCode: "ITM-001", ItemName: "Item 1", Name: "ITM-001"}}, nil
		},
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/items?query=itm&limit=10&warehouse="+url.QueryEscape("Stores - A"),
		nil,
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var payload struct {
		Data []store.Item `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].ItemCode != "ITM-001" {
		t.Fatalf("unexpected payload: %+v", payload.Data)
	}
}

func TestHandshakeEndpoint(t *testing.T) {
	h := NewHandler(fakeSearcher{})

	req := httptest.NewRequest(http.MethodGet, "/v1/handshake", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload["service"] != "gscale_erp_read" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestWarehousesEndpoint(t *testing.T) {
	h := NewHandler(fakeSearcher{
		stocks: []store.WarehouseStock{{Warehouse: "Stores - A", ActualQty: 12.5}},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/items/ITM-001/warehouses", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var payload struct {
		Data []store.WarehouseStock `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].Warehouse != "Stores - A" {
		t.Fatalf("unexpected payload: %+v", payload.Data)
	}
}

func TestItemDetailEndpoint(t *testing.T) {
	h := NewHandler(fakeSearcher{
		item: store.ItemDetail{
			Name:     "ITM-001",
			ItemCode: "ITM-001",
			ItemName: "Item 1",
			StockUOM: "Kg",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/items/ITM-001", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWarehouseDetailEndpoint(t *testing.T) {
	h := NewHandler(fakeSearcher{
		wh: store.Warehouse{
			Name:    "Stores - A",
			Company: "A Company",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/warehouses/Stores%20-%20A", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
