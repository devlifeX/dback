# Deploying DBack Control Plane

DBack server mode runs as a headless control plane with HTTP API and optional Web UI.

## Requirements

- Linux (systemd unit provided)
- Encrypted vault passphrase (`DBACK_PASSPHRASE` or file)
- API bearer token (`DBACK_API_TOKEN` or file)
- Optional: built Web UI at `DBACK_WEB_ROOT`

## Quick start

```bash
export DBACK_DATA_DIR=/var/lib/dback
export DBACK_PASSPHRASE_FILE=/etc/dback/passphrase
export DBACK_API_TOKEN_FILE=/etc/dback/api-token
export DBACK_LISTEN=127.0.0.1:14127
export DBACK_WEB_ROOT=/usr/share/dback/web

dback serve
```

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DBACK_DATA_DIR` | `~/.config/dback` | Vault and data directory |
| `DBACK_LISTEN` | `127.0.0.1:14127` | HTTP bind address |
| `DBACK_PASSPHRASE` / `_FILE` | — | Vault unlock passphrase |
| `DBACK_API_TOKEN` / `_FILE` | — | Bearer token for `/api/v1/*` |
| `DBACK_WEB_ROOT` | — | Path to `web/dist` for SPA |
| `DBACK_QUEUE_CAPACITY` | `256` | Operation queue size |
| `DBACK_MAX_CONCURRENT` | `4` | Concurrent operation workers |
| `DBACK_SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown drain |
| `DBACK_RATE_LIMIT_RPS` | `20` | Per-IP API rate limit (avg) |
| `DBACK_RATE_LIMIT_BURST` | `40` | Rate limit burst |
| `DBACK_METRICS` | `true` | Expose `GET /metrics` (Prometheus) |
| `DBACK_AUDIT_CAP` | `500` | In-memory audit ring buffer |

## systemd

Install [`packaging/dback.service`](../packaging/dback.service) and provide `/etc/dback/dback.env`:

```ini
DBACK_DATA_DIR=/var/lib/dback
DBACK_PASSPHRASE_FILE=/etc/dback/passphrase
DBACK_API_TOKEN_FILE=/etc/dback/api-token
DBACK_LISTEN=127.0.0.1:14127
DBACK_WEB_ROOT=/usr/share/dback/web
```

```bash
sudo systemctl enable --now dback
```

## Health checks

- `GET /health/live` — process up
- `GET /health/ready` — vault unlocked
- `GET /metrics` — Prometheus metrics (restrict via firewall or reverse proxy)

## Web UI build

```bash
cd web && npm ci && npm run build
```

Serve `web/dist` via `DBACK_WEB_ROOT` or place behind reverse proxy static root.

## API

- OpenAPI: `GET /api/v1/openapi.json`
- Versioning policy: [api-versioning.md](api-versioning.md)
- Changelog: [../api/CHANGELOG.md](../api/CHANGELOG.md)

TLS termination and public exposure: use reverse proxy — see [reverse-proxy.md](reverse-proxy.md).
