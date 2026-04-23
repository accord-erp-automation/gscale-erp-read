package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"unicode"
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

func (s *Store) SearchItems(ctx context.Context, query string, limit int, warehouse string) ([]Item, error) {
	query = strings.TrimSpace(query)
	warehouse = strings.TrimSpace(warehouse)
	terms := searchTerms(query)
	args := make([]any, 0, 16)

	sqlText := `
SELECT
	name,
	COALESCE(NULLIF(item_code, ''), name) AS item_code,
	COALESCE(NULLIF(item_name, ''), NULLIF(item_code, ''), name) AS item_name
FROM tabItem
`
	whereAdded := false
	if warehouse != "" {
		sqlText += `
WHERE EXISTS (
	SELECT 1
	FROM ` + "`tabItem Default`" + ` item_default
	WHERE item_default.parent = tabItem.name
		AND item_default.default_warehouse = ?
)
`
		args = append(args, warehouse)
		whereAdded = true
	}
	if len(terms) > 0 {
		if whereAdded {
			sqlText += `
AND (
`
		} else {
			sqlText += `
WHERE (
`
			whereAdded = true
		}
		filterAdded := false
		for _, term := range terms {
			term = normalizedSearchText(term)
			if term == "" {
				continue
			}
			compact := compactField(term)
			if compact == "" {
				continue
			}
			if filterAdded {
				sqlText += `
	OR
`
			}
			sqlText += `(
		LOWER(COALESCE(NULLIF(tabItem.item_code, ''), tabItem.name)) LIKE ?
		OR LOWER(COALESCE(NULLIF(tabItem.item_name, ''), COALESCE(NULLIF(tabItem.item_code, ''), tabItem.name))) LIKE ?
		OR LOWER(tabItem.name) LIKE ?
		OR REPLACE(REPLACE(REPLACE(LOWER(COALESCE(NULLIF(tabItem.item_code, ''), tabItem.name)), ' ', ''), '-', ''), '_', '') LIKE ?
		OR REPLACE(REPLACE(REPLACE(LOWER(COALESCE(NULLIF(tabItem.item_name, ''), COALESCE(NULLIF(tabItem.item_code, ''), tabItem.name))), ' ', ''), '-', ''), '_', '') LIKE ?
		OR REPLACE(REPLACE(REPLACE(LOWER(tabItem.name), ' ', ''), '-', ''), '_', '') LIKE ?
	)`
			args = append(args,
				"%"+term+"%",
				"%"+term+"%",
				"%"+term+"%",
				"%"+compact+"%",
				"%"+compact+"%",
				"%"+compact+"%",
			)
			filterAdded = true
		}
		sqlText += `
)
`
	}
	sqlText += `
ORDER BY modified DESC
`
	if limit > 0 {
		sqlText += `LIMIT ?
`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("search items query: %w", err)
	}
	defer rows.Close()

	items := make([]Item, 0, 128)
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
	if len(terms) > 0 {
		items = rankItems(items, terms)
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}
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
SELECT warehouse, MAX(actual_qty) AS actual_qty
FROM (
	SELECT warehouse, actual_qty
	FROM tabBin
	WHERE item_code = ? AND actual_qty > 0

	UNION ALL

	SELECT DISTINCT
		item_default.default_warehouse AS warehouse,
		0 AS actual_qty
	FROM tabItem
	INNER JOIN ` + "`tabItem Default`" + ` item_default
		ON item_default.parent = tabItem.name
	WHERE (tabItem.item_code = ? OR tabItem.name = ?)
		AND COALESCE(NULLIF(item_default.default_warehouse, ''), '') <> ''
) warehouse_options
WHERE 1 = 1
`
	args := make([]any, 0, 6)
	args = append(args, itemCode, itemCode, itemCode)

	if query != "" {
		sqlText += `AND warehouse LIKE ?
`
		args = append(args, "%"+query+"%")
	}

	sqlText += `
GROUP BY warehouse
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

func searchTerms(query string) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 12)
	add := func(value string) {
		value = normalizedSearchText(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	addWithVariants := func(value string) {
		value = normalizedSearchText(value)
		if value == "" {
			return
		}
		queue := []string{value}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if _, ok := seen[current]; ok {
				continue
			}
			add(current)
			for _, variant := range searchAliasVariants(current) {
				variant = normalizedSearchText(variant)
				if variant == "" {
					continue
				}
				if _, ok := seen[variant]; ok {
					continue
				}
				queue = append(queue, variant)
			}
		}
	}
	addWithVariants(query)
	addWithVariants(transliterateUzbek(query))
	return out
}

func searchAliasVariants(value string) []string {
	value = normalizedSearchText(value)
	if value == "" {
		return nil
	}

	tokens := strings.Fields(value)
	if len(tokens) == 0 {
		return nil
	}

	phrases := []string{""}
	for _, token := range tokens {
		variants := tokenAliasVariants(token)
		next := make([]string, 0, len(phrases)*len(variants))
		for _, phrase := range phrases {
			for _, variant := range variants {
				combined := strings.TrimSpace(strings.Join([]string{phrase, variant}, " "))
				if combined == "" {
					continue
				}
				next = appendUniqueSearchValue(next, combined)
				if len(next) >= 16 {
					break
				}
			}
			if len(next) >= 16 {
				break
			}
		}
		phrases = next
		if len(phrases) == 0 {
			break
		}
	}

	out := make([]string, 0, len(phrases)*2)
	for _, phrase := range phrases {
		out = appendUniqueSearchValue(out, phrase)
		out = appendUniqueSearchValue(out, compactField(phrase))
	}
	return out
}

func tokenAliasVariants(token string) []string {
	token = normalizedSearchText(token)
	if token == "" {
		return nil
	}

	out := []string{token}
	add := func(value string) {
		value = normalizedSearchText(value)
		if value == "" {
			return
		}
		out = appendUniqueSearchValue(out, value)
	}

	if strings.HasPrefix(token, "x") && len(token) > 1 {
		add("h" + token[1:])
	}
	if strings.HasPrefix(token, "h") && len(token) > 1 {
		add("x" + token[1:])
	}

	type aliasRule struct {
		from string
		to   string
	}
	rules := []aliasRule{
		{from: "lanch", to: "lunch"},
		{from: "lunch", to: "lanch"},
		{from: "launch", to: "lanch"},
		{from: "launch", to: "lunch"},
	}
	for _, rule := range rules {
		if strings.Contains(token, rule.from) {
			add(strings.ReplaceAll(token, rule.from, rule.to))
		}
	}

	return out
}

func appendUniqueSearchValue(values []string, value string) []string {
	value = normalizedSearchText(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func normalizedSearchText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	lastSpace := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastSpace = false
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '\'' || r == '`' || r == '’':
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func transliterateUzbek(value string) string {
	replacer := strings.NewReplacer(
		"o'", "o",
		"g'", "g",
		"sh", "s",
		"ch", "c",
		"yo", "io",
		"yu", "iu",
		"ya", "ia",
		"ё", "yo",
		"ю", "yu",
		"я", "ya",
		"ш", "sh",
		"ч", "ch",
		"ғ", "g",
		"ў", "o",
		"қ", "q",
		"ҳ", "h",
		"й", "y",
		"ц", "s",
		"ы", "i",
		"э", "e",
		"ъ", "",
		"ь", "",
		"а", "a",
		"б", "b",
		"в", "v",
		"г", "g",
		"д", "d",
		"е", "e",
		"ж", "j",
		"з", "z",
		"и", "i",
		"к", "k",
		"л", "l",
		"м", "m",
		"н", "n",
		"о", "o",
		"п", "p",
		"р", "r",
		"с", "s",
		"т", "t",
		"у", "u",
		"ф", "f",
		"х", "x",
		"ь", "",
		"й", "y",
	)
	return replacer.Replace(strings.ToLower(value))
}

func rankItems(items []Item, terms []string) []Item {
	type scoredItem struct {
		item  Item
		score int
	}
	scored := make([]scoredItem, 0, len(items))
	for _, item := range items {
		score := scoreItemMatch(item, terms)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredItem{item: item, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].item.ItemCode < scored[j].item.ItemCode
	})
	out := make([]Item, 0, len(scored))
	for _, entry := range scored {
		out = append(out, entry.item)
	}
	return out
}

func scoreItemMatch(item Item, terms []string) int {
	fields := []string{
		normalizedSearchText(item.ItemCode),
		normalizedSearchText(item.ItemName),
		normalizedSearchText(item.Name),
		normalizedSearchText(transliterateUzbek(item.ItemCode)),
		normalizedSearchText(transliterateUzbek(item.ItemName)),
		normalizedSearchText(transliterateUzbek(item.Name)),
	}
	score := 0
	for _, term := range terms {
		best := 0
		for _, field := range fields {
			best = max(best, fuzzyFieldScore(field, term))
		}
		score += best
	}
	return score
}

func fuzzyFieldScore(field, term string) int {
	field = normalizedSearchText(field)
	term = normalizedSearchText(term)
	if field == "" || term == "" {
		return 0
	}
	fieldCompact := compactField(field)
	termCompact := compactField(term)
	shortTerm := len([]rune(termCompact)) <= 3
	switch {
	case field == term:
		return 120
	case strings.HasPrefix(field, term):
		return 100
	case fieldCompact == termCompact:
		return 99
	case strings.HasPrefix(fieldCompact, termCompact):
		return 98
	case strings.Contains(field, " "+term):
		return 90
	case !shortTerm && strings.Contains(field, term):
		return 75
	case !shortTerm && strings.Contains(fieldCompact, termCompact):
		return 72
	case !shortTerm && tokenTypoScore(field, term) > 0:
		return tokenTypoScore(field, term)
	case !shortTerm && subsequenceMatch(fieldCompact, termCompact):
		return 55
	case !shortTerm && levenshteinDistance(field, term) <= 1:
		return 45
	case !shortTerm && levenshteinDistance(fieldCompact, termCompact) <= 1:
		return 44
	case !shortTerm && levenshteinDistance(firstToken(field), term) <= 1:
		return 40
	case !shortTerm && levenshteinDistance(firstToken(fieldCompact), termCompact) <= 1:
		return 39
	default:
		return 0
	}
}

func compactField(value string) string {
	return strings.ReplaceAll(normalizedSearchText(value), " ", "")
}

func firstToken(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func subsequenceMatch(field, term string) bool {
	if len(term) < 3 {
		return false
	}
	target := []rune(term)
	idx := 0
	for _, r := range field {
		if idx < len(target) && r == target[idx] {
			idx++
			if idx == len(target) {
				return true
			}
		}
	}
	return false
}

func tokenTypoScore(field, term string) int {
	if term == "" {
		return 0
	}
	best := 0
	for _, token := range strings.Fields(field) {
		token = compactField(token)
		if token == "" {
			continue
		}
		if token == term {
			return 110
		}
		if strings.HasPrefix(token, term) {
			best = max(best, 97)
			continue
		}
		if strings.Contains(token, term) {
			best = max(best, 74)
			continue
		}
		if levenshteinDistance(token, term) <= 1 {
			best = max(best, 68)
			continue
		}
		if len(term) >= 4 && len(token) >= 4 && subsequenceMatch(token, term) {
			best = max(best, 58)
		}
	}
	return best
}

func levenshteinDistance(left, right string) int {
	if left == right {
		return 0
	}
	if left == "" {
		return len([]rune(right))
	}
	if right == "" {
		return len([]rune(left))
	}
	lr := []rune(left)
	rr := []rune(right)
	prev := make([]int, len(rr)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(lr); i++ {
		cur := make([]int, len(rr)+1)
		cur[0] = i
		for j := 1; j <= len(rr); j++ {
			cost := 0
			if lr[i-1] != rr[j-1] {
				cost = 1
			}
			cur[j] = min3(
				prev[j]+1,
				cur[j-1]+1,
				prev[j-1]+cost,
			)
		}
		prev = cur
	}
	return prev[len(rr)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
