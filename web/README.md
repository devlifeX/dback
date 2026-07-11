# DBack Web UI

Primary management interface for the DBack Control Plane (React 19 + Vite + TypeScript).

## Development

```bash
# Terminal 1 — API server
export DBACK_API_TOKEN=dev-token
export DBACK_PASSPHRASE=your-vault-pass
dback serve

# Terminal 2 — Vite dev server (proxies /api to :14127)
cd web && npm install && npm run dev
```

Open http://localhost:5173 and enter the API token when prompted.

## Production

```bash
cd web && npm run build
export DBACK_WEB_ROOT=/path/to/dback/web/dist
export DBACK_API_TOKEN=...
dback serve
```

The server serves the SPA from `/` and API from `/api/v1/`.

## Structure

Feature-based layout under `src/features/` with API adapters in `src/api/`.
