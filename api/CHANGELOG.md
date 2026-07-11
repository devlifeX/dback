# DBack API Changelog

All notable changes to the `/api/v1` contract are documented here.

## [1.0.0] - 2026-07

### Added

- Initial stable v1 REST API under `/api/v1/`
- Bearer token authentication (`DBACK_API_TOKEN` / `DBACK_API_TOKEN_FILE`)
- Operations: list, create, get, cancel, retry, logs, SSE stream
- Tasks: CRUD, run, list runs
- Hosts (profiles): CRUD, connection test (secret-redacted responses)
- Templates, backups, remote destinations, notifications, logs, sync
- OpenAPI spec at `GET /api/v1/openapi.json`
- Vault ETag/`If-Match` via `W/"<revision>"` on mutating endpoints
- Go SDK at `sdk/go/dback/`
