# DBack API Versioning

Public HTTP APIs live under `/api/v1/`. This document defines compatibility rules for v1.

## URL versioning

- All public endpoints use the `/api/v1/` prefix.
- A new major version (`v2`) is required only for breaking changes, not for additive features.

## Authentication

- v1 uses `Authorization: Bearer <token>` only.
- Configure `DBACK_API_TOKEN_FILE` (preferred) or `DBACK_API_TOKEN`.
- Health probes (`/health/live`, `/health/ready`) and `GET /api/v1/version` do not require auth.
- All other `/api/v1/*` routes require a valid bearer token.

## Backward compatibility (within v1)

| Allowed | Breaking (requires v2) |
|---------|------------------------|
| Optional JSON fields | Remove or rename fields |
| New endpoints | Change semantics of existing fields |
| New enum values | Change required ↔ optional |
| New SSE event types | Change paths or HTTP methods |

**Client rule:** ignore unknown JSON fields and tolerate unknown enum values.

## Vault concurrency

- Mutating vault-backed resources support optimistic concurrency.
- Responses include `ETag: W/"<revision>"` where revision is the vault data revision.
- Send `If-Match` on PUT/POST/DELETE mutations to avoid lost updates.

## OpenAPI

- Source of truth: `internal/api/v1/openapi.yaml` (embedded, served as JSON).
- Regenerate clients from the committed spec when the API changes.

## Deprecation

1. Mark deprecated endpoints in OpenAPI and response headers.
2. Maintain overlap for at least 90 days before removal.
3. Record changes in `api/CHANGELOG.md`.
