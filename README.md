# gscale_erp_read

Standalone read-only ERP service for item search and warehouse shortlist.

## Endpoints

- `GET /healthz`
- `GET /v1/items?query=...&limit=...`
- `GET /v1/items/{item_code}`
- `GET /v1/items/{item_code}/warehouses?query=...&limit=...`
- `GET /v1/warehouses/{warehouse}`

The service is shaped around what the bot currently reads from ERPNext:

- item search for inline item selection
- warehouse shortlist for the chosen item
- item `stock_uom` for stock-entry draft creation
- warehouse `company` for stock-entry draft creation

## Run

From the bench root or with `ERP_BENCH_ROOT` pointing at it:

```bash
cd gscale_erp_read
go run ./cmd/gscale-erp-read
```

Optional env vars:

- `ERP_READ_ADDR`
- `ERP_BENCH_ROOT`
- `ERP_SITE_NAME`
- `ERP_SITE_CONFIG`
- `ERP_DB_HOST`
- `ERP_DB_PORT`
- `ERP_DB_USER`
