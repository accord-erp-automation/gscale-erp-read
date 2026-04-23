# gscale-erp-read

## Abstract
`gscale-erp-read` is the read-only ERP catalog service used by the GScale system. It is responsible for exposing a narrow, stable HTTP interface for item and warehouse discovery without granting write access to ERP business documents.

This repository is one of three companion repositories:

- [`gscale-platform`](https://github.com/accord-erp-automation/gscale-platform): orchestration, mobile API, scale worker, simulator, and print-request flow.
- [`gscale-erp-read`](https://github.com/accord-erp-automation/gscale-erp-read): ERP-side read service implemented here.
- [`gscale-mobile-app`](https://github.com/WIKKIwk/gscale-mobile-app): operator-facing Flutter client.

If `gscale-platform` is the runtime coordinator, this repository is the catalog intelligence layer.

## Role in the Three-Repository Architecture

The central design decision of the GScale system is the separation of ERP writes from ERP reads.

This repository exists so that:

- item search remains fast and controlled,
- warehouse-related lookup logic can be specialized,
- the mobile client does not need direct awareness of ERP schema details,
- the main runtime can write ERP documents without also becoming the canonical read-model implementation.

In short, this service is the repository's way of saying to the other two repositories: "I am your ERP catalog specialist."

## Architectural Relationship

```text
gscale-mobile-app
        |
        v
   gscale-platform/mobileapi
        |
        v
     gscale-erp-read
        |
        v
     ERPNext DB
```

The mobile application never calls this service directly in the normal user flow. Instead, `gscale-platform` calls it and re-exposes catalog operations through mobile-facing endpoints.

## Responsibilities

This repository is intentionally limited to read-only concerns:

- item search,
- item detail lookup,
- item-to-warehouse shortlist lookup,
- warehouse detail lookup,
- warehouse-aware item filtering for default-warehouse workflows.

It deliberately does **not**:

- create ERP drafts,
- submit ERP documents,
- coordinate print requests,
- interact with Zebra printers,
- maintain batch transaction state.

Those responsibilities belong to `gscale-platform`.

## API Surface

### Health

- `GET /healthz`
- `GET /v1/handshake`

### Catalog Endpoints

- `GET /v1/items?query=...&limit=...&warehouse=...`
- `GET /v1/items/{item_code}`
- `GET /v1/items/{item_code}/warehouses?query=...&limit=...`
- `GET /v1/warehouses/{warehouse}`

### Important Semantics

`warehouse` on `/v1/items` is not cosmetic. It acts as a real filter when the caller wants the item picker constrained to a default warehouse. This behavior is important for the mobile workflow implemented in `gscale-platform`.

## Why This Service Exists Instead of Using ERP REST Directly

Using ERP resource endpoints directly from the runtime layer would be possible, but it would push catalog policy into the wrong place. This repository provides:

- a narrower contract,
- lower coupling to ERP internals,
- easier search policy tuning,
- easier warehouse-aware filtering,
- cleaner testing boundaries.

It therefore serves as the domain-specific read adapter for the broader GScale system.

## Data Source Strategy

The service loads ERP connection metadata from the ERP bench and site configuration:

- `ERP_BENCH_ROOT`
- `ERP_SITE_NAME`
- `ERP_SITE_CONFIG`

When deployed beside the ERP installation, the service can read trusted site configuration and connect directly to MariaDB using the site database credentials.

## Run

From the bench root, or by pointing at it explicitly:

```bash
go run ./cmd/gscale-erp-read
```

Typical development invocation:

```bash
ERP_BENCH_ROOT=/path/to/erp/bench \
ERP_SITE_NAME=erp.localhost \
ERP_READ_ADDR=127.0.0.1:8090 \
go run ./cmd/gscale-erp-read
```

Optional environment variables:

- `ERP_READ_ADDR`
- `ERP_BENCH_ROOT`
- `ERP_SITE_NAME`
- `ERP_SITE_CONFIG`
- `ERP_DB_HOST`
- `ERP_DB_PORT`
- `ERP_DB_USER`

## Interaction With `gscale-platform`

`gscale-platform` depends on this service for:

- mobile item picker results,
- mobile default-warehouse filtered item lists,
- item stock UOM lookup before draft creation,
- warehouse company lookup before material receipt creation,
- item-specific warehouse shortlist generation.

That means any change in search behavior here should be considered a public contract change for `gscale-platform`.

## Interaction With `gscale-mobile-app`

The Flutter mobile client does not usually call this repository directly. However, the user experience in the app is strongly shaped by this service because:

- item picker behavior depends on item search behavior here,
- default warehouse selection only becomes trustworthy when warehouse-aware item filtering is correct here.

In effect, the mobile app's catalog UX is downstream from this service.

## Testing Philosophy

This repository should be validated in two modes:

1. isolated package tests:

```bash
GOWORK=off go test ./...
```

2. integrated tests through `gscale-platform`:

- start `gscale-erp-read`,
- start `gscale-platform/mobileapi`,
- verify `/v1/mobile/items`,
- verify default-warehouse filtering,
- verify item-to-warehouse shortlist results.

The second mode matters because this repository is only one part of the system contract.

## Recommended Companion Reading

To understand how this repository participates in the full system, read these next:

1. [`gscale-platform`](https://github.com/accord-erp-automation/gscale-platform)
2. [`gscale-mobile-app`](https://github.com/WIKKIwk/gscale-mobile-app)

Those repositories are the operational and UI counterparts of this service.
