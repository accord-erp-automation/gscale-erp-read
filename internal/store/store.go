package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Item struct {
	Name     string `json:"name"`
	ItemCode string `json:"item_code"`
	ItemName string `json:"item_name"`
}

type ItemDetail struct {
	Name     string `json:"name"`
	ItemCode string `json:"item_code"`
	ItemName string `json:"item_name"`
	StockUOM string `json:"stock_uom"`
}

type WarehouseStock struct {
	Warehouse string  `json:"warehouse"`
	ActualQty float64 `json:"actual_qty"`
}

type Warehouse struct {
	Name    string `json:"name"`
	Company string `json:"company"`
}

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) SearchItems(ctx context.Context, query string, limit int) ([]Item, error) {
	limit = normalizeLimit(limit)
	query = strings.TrimSpace(query)

	sqlText := `
SELECT
	name,
	COALESCE(NULLIF(item_code, ''), name) AS item_code,
	COALESCE(NULLIF(item_name, ''), NULLIF(item_code, ''), name) AS item_name
FROM tabItem
`
	args := make([]any, 0, 5)
	if query != "" {
		pattern := "%" + query + "%"
		sqlText += `
WHERE item_code LIKE ? OR item_name LIKE ? OR name LIKE ?
`
		args = append(args, pattern, pattern, pattern)
	}
	sqlText += `
ORDER BY modified DESC
LIMIT ?
`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("search items query: %w", err)
	}
	defer rows.Close()

	items := make([]Item, 0, limit)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.Name, &item.ItemCode, &item.ItemName); err != nil {
			return nil, fmt.Errorf("search items scan: %w", err)
		}
		item.Name = strings.TrimSpace(item.Name)
		item.ItemCode = strings.TrimSpace(item.ItemCode)
		item.ItemName = strings.TrimSpace(item.ItemName)
		if item.ItemCode == "" {
			continue
		}
		if item.ItemName == "" {
			item.ItemName = item.ItemCode
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search items rows: %w", err)
	}
	return items, nil
}

func (s *Store) GetItem(ctx context.Context, itemCode string) (ItemDetail, error) {
	itemCode = strings.TrimSpace(itemCode)
	if itemCode == "" {
		return ItemDetail{}, fmt.Errorf("item code is empty")
	}

	const sqlText = `
SELECT
	name,
	COALESCE(NULLIF(item_code, ''), name) AS item_code,
	COALESCE(NULLIF(item_name, ''), NULLIF(item_code, ''), name) AS item_name,
	COALESCE(NULLIF(stock_uom, ''), '') AS stock_uom
FROM tabItem
WHERE item_code = ? OR name = ?
LIMIT 1
`

	row := s.db.QueryRowContext(ctx, sqlText, itemCode, itemCode)
	var item ItemDetail
	if err := row.Scan(&item.Name, &item.ItemCode, &item.ItemName, &item.StockUOM); err != nil {
		if err == sql.ErrNoRows {
			return ItemDetail{}, fmt.Errorf("item topilmadi: %s", itemCode)
		}
		return ItemDetail{}, fmt.Errorf("get item query: %w", err)
	}
	item.Name = strings.TrimSpace(item.Name)
	item.ItemCode = strings.TrimSpace(item.ItemCode)
	item.ItemName = strings.TrimSpace(item.ItemName)
	item.StockUOM = strings.TrimSpace(item.StockUOM)
	if item.ItemName == "" {
		item.ItemName = item.ItemCode
	}
	return item, nil
}

func (s *Store) SearchItemWarehouses(ctx context.Context, itemCode, query string, limit int) ([]WarehouseStock, error) {
	itemCode = strings.TrimSpace(itemCode)
	if itemCode == "" {
		return nil, fmt.Errorf("item code is empty")
	}

	limit = normalizeLimit(limit)
	query = strings.TrimSpace(query)

	sqlText := `
SELECT warehouse, actual_qty
FROM tabBin
WHERE item_code = ? AND actual_qty > 0
`
	args := make([]any, 0, 3)
	args = append(args, itemCode)

	if query != "" {
		sqlText += `AND warehouse LIKE ?
`
		args = append(args, "%"+query+"%")
	}

	sqlText += `
ORDER BY actual_qty DESC, warehouse ASC
LIMIT ?
`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("search warehouses query: %w", err)
	}
	defer rows.Close()

	stocks := make([]WarehouseStock, 0, limit)
	for rows.Next() {
		var stock WarehouseStock
		if err := rows.Scan(&stock.Warehouse, &stock.ActualQty); err != nil {
			return nil, fmt.Errorf("search warehouses scan: %w", err)
		}
		stock.Warehouse = strings.TrimSpace(stock.Warehouse)
		if stock.Warehouse == "" {
			continue
		}
		stocks = append(stocks, stock)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search warehouses rows: %w", err)
	}
	return stocks, nil
}

func (s *Store) GetWarehouse(ctx context.Context, warehouse string) (Warehouse, error) {
	warehouse = strings.TrimSpace(warehouse)
	if warehouse == "" {
		return Warehouse{}, fmt.Errorf("warehouse is empty")
	}

	const sqlText = `
SELECT
	name,
	COALESCE(NULLIF(company, ''), '') AS company
FROM tabWarehouse
WHERE name = ?
LIMIT 1
`

	row := s.db.QueryRowContext(ctx, sqlText, warehouse)
	var out Warehouse
	if err := row.Scan(&out.Name, &out.Company); err != nil {
		if err == sql.ErrNoRows {
			return Warehouse{}, fmt.Errorf("warehouse topilmadi: %s", warehouse)
		}
		return Warehouse{}, fmt.Errorf("get warehouse query: %w", err)
	}
	out.Name = strings.TrimSpace(out.Name)
	out.Company = strings.TrimSpace(out.Company)
	return out, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 50 {
		return 50
	}
	return limit
}
