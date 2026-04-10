package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gscale_erp_read/internal/store"
)

type fakeSearcher struct {
	items  []store.Item
	stocks []store.WarehouseStock
}

func (f fakeSearcher) SearchItems(ctx context.Context, query string, limit int) ([]store.Item, error) {
	return f.items, nil
}

func (f fakeSearcher) SearchItemWarehouses(ctx context.Context, itemCode, query string, limit int) ([]store.WarehouseStock, error) {
	return f.stocks, nil
}

func TestItemsEndpoint(t *testing.T) {
	h := NewHandler(fakeSearcher{
		items: []store.Item{{ItemCode: "ITM-001", ItemName: "Item 1", Name: "ITM-001"}},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/items?query=itm&limit=10", nil)
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
