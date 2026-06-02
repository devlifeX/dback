# DBack DB Tools — WordPress Plugin Agent Guide

This document describes how the **DBack DB Tools** WordPress plugin works. It is written for AI agents and developers working in the `dback` monorepo or on a WordPress site where this plugin is installed.

---

## Purpose

The plugin exposes **pure-PHP** database operations for the **DBack** desktop app (Go) and for WordPress administrators:

- **Export** — stream a `.sql.gz` backup of the WordPress database
- **Import** — restore from a `.sql.gz` file
- **Run SQL** — execute arbitrary SQL against the WordPress database
- **Error log** — structured error logging for admin and REST clients

It replaces the legacy single-file template at `plugin_template/dback-sync.php`, which used `exec()` / shell commands (`mysqldump`, `mysql`, `gunzip`). **This plugin must never use shell commands.**

Target environment: **heavy databases** and **shared hosting** where CLI tools and long-running shell processes are unavailable or unreliable.

---

## Location in repo

```
wordpress/dback-db-tools/          ← install this folder into wp-content/plugins/
├── dback-db-tools.php             ← plugin bootstrap
├── wordpress_agent.md             ← this file
├── includes/                      ← PHP classes
├── assets/admin.js                ← admin UI (REST client)
└── vendor/ifsnop/mysqldump-php/   ← vendored dump library (no Composer at runtime)
```

Legacy reference (do not extend unless explicitly asked):

```
plugin_template/dback-sync.php     ← old exec()-based stub; same REST namespace
```

The Go desktop app currently **does not** actively use WordPress profiles (`ConnectionType "WordPress"` is filtered out in `internal/store/store.go`). The plugin is built for future or manual integration via REST.

---

## Hard constraints (never violate)

1. **No shell execution** — forbidden: `exec`, `shell_exec`, `system`, `passthru`, `proc_open`, `popen`, backticks, or invoking `mysqldump` / `mysql` / `gunzip` CLI.
2. **Pure PHP only** — use PDO, `$wpdb`/mysqli, zlib (`deflate_*`, `gzopen` for reads), and PHP streams.
3. **Streaming for export** — do not buffer the entire dump in memory or on disk before sending; write to `php://output` incrementally to avoid HTTP timeouts on large DBs.
4. **REST namespace** — keep `dback/v1` and header `X-DBACK-KEY` for compatibility with the legacy template and DBack clients.
5. **Security** — all routes require `X-DBACK-KEY` **or** logged-in user with `manage_options`. SQL execution is full database access; do not weaken auth.

---

## Architecture overview

```mermaid
flowchart TB
    subgraph clients [Clients]
        DBackGo[DBack Go app]
        AdminUI[WP Admin Tools page]
    end

    subgraph rest [REST API dback/v1]
        Export[/export GET]
        Import[/import POST]
        Query[/query POST]
        Logs[/logs GET DELETE]
    end

    subgraph core [Plugin core]
        RestCtrl[DBack_Rest_Controller]
        Exporter[DBack_Exporter]
        ExporterMysqli[DBack_Exporter_Mysqli]
        Importer[DBack_Importer]
        QueryRunner[DBack_Query_Runner]
        ErrorLog[DBack_Error_Logger]
        Database[DBack_Database]
        GzipStream[DBack_Gzip_Stream]
    end

    subgraph drivers [DB drivers]
        PDO[mysqldump-php via PDO]
        WPDB[wpdb mysqli fallback]
    end

    DBackGo --> rest
    AdminUI --> rest
    rest --> RestCtrl
    RestCtrl --> Exporter
    RestCtrl --> Importer
    RestCtrl --> QueryRunner
    RestCtrl --> ErrorLog
    Exporter -->|pdo_mysql available| PDO
    Exporter -->|fallback| ExporterMysqli
    ExporterMysqli --> GzipStream
    ExporterMysqli --> WPDB
    Importer --> Database
    QueryRunner --> Database
    Database --> PDO
    Database --> WPDB
```

---

## File map

| File | Responsibility |
|------|----------------|
| `dback-db-tools.php` | Constants, requires, hooks, singleton init |
| `class-dback-api-key.php` | API key in option `dback_api_key`; generate on activation |
| `class-dback-database.php` | Credentials, DSN, PDO/wpdb abstraction, temp paths |
| `class-dback-exporter.php` | Export entry: PDO path or delegate to mysqli fallback |
| `class-dback-exporter-mysqli.php` | Pure `$wpdb`/mysqli dump when PDO unavailable |
| `class-dback-gzip-stream.php` | Gzip write to `php://output` via `fopen` + `deflate_*` |
| `class-dback-importer.php` | Gzip import: temp file → line parser → `DBack_Database::exec` |
| `class-dback-query-runner.php` | SQL runner; returns rows or affected count |
| `class-dback-rest-controller.php` | REST routes, auth, error wrapping |
| `class-dback-error-logger.php` | Log to option + JSONL file; build `WP_Error` |
| `class-dback-admin-page.php` | Tools → DBack DB Tools admin page |
| `assets/admin.js` | Admin forms calling REST API |

---

## Constants

| Constant | Value |
|----------|-------|
| `DBACK_DB_TOOLS_VERSION` | Plugin version string |
| `DBACK_DB_TOOLS_REST_NAMESPACE` | `dback/v1` |
| `DBack_Api_Key::OPTION_NAME` | `dback_api_key` |
| `DBack_Error_Logger::OPTION_KEY` | `dback_error_log` |
| Temp directory | `{uploads}/dback-db-tools/` |
| Error log file | `{uploads}/dback-db-tools/dback-errors.log` |

---

## Authentication

### External client (DBack app)

```http
X-DBACK-KEY: {32-char key from wp option dback_api_key}
```

Key is shown in **Tools → DBack DB Tools** and can be regenerated there.

### WordPress admin UI

Uses same REST routes with:

```http
X-WP-Nonce: {wp_create_nonce('wp_rest')}
Cookie: {wordpress logged-in session}
```

User must have `manage_options`.

### Permission matrix

| Route | API key | Admin + nonce |
|-------|---------|---------------|
| `GET /export` | yes | yes |
| `POST /import` | yes | yes |
| `POST /query` | yes | yes |
| `GET /logs` | yes | yes |
| `DELETE /logs` | no | yes only |

---

## REST API reference

Base URL: `https://{site}/wp-json/dback/v1`

### Export

```http
GET /export
X-DBACK-KEY: {key}
```

**Success:** raw binary body, `Content-Type: application/gzip`, `Content-Disposition: attachment`.

**Failure:** JSON `WP_Error` (export may fail before stream starts; once streaming begins, errors become connection aborts).

Client example (Go / curl):

```bash
curl -H "X-DBACK-KEY: YOUR_KEY" \
  "https://example.com/wp-json/dback/v1/export" \
  -o backup.sql.gz
```

Read the response body as a **stream** (`http.Response.Body` in Go). Chunks keep the HTTP connection alive and prevent gateway timeouts.

### Import

```http
POST /import
X-DBACK-KEY: {key}
Content-Type: application/gzip
Body: raw .sql.gz bytes
```

**Success:**

```json
{
  "success": true,
  "message": "Database imported successfully.",
  "statements_executed": 42
}
```

Import flow:

1. Read raw body from `php://input`
2. Save to temp file under uploads (`import-{random}.sql.gz`)
3. `gzopen` + read line-by-line
4. Accumulate SQL until line ends with `;`
5. Execute via `DBack_Database::exec()`
6. Delete temp file

### Query

```http
POST /query
X-DBACK-KEY: {key}
Content-Type: application/json

{"sql": "SHOW TABLES"}
```

**SELECT / SHOW / DESCRIBE / EXPLAIN / WITH:**

```json
{
  "success": true,
  "type": "result",
  "query_type": "SHOW",
  "columns": ["Tables_in_db"],
  "rows": [{"Tables_in_db": "wp_posts"}],
  "row_count": 1,
  "driver": "wpdb"
}
```

**Other statements (INSERT, UPDATE, DELETE, etc.):**

```json
{
  "success": true,
  "type": "command",
  "query_type": "UPDATE",
  "affected_rows": 3,
  "driver": "pdo"
}
```

### Error log

```http
GET /logs?limit=50
DELETE /logs   (admin only)
```

---

## Error response format

All handled failures go through `DBack_Error_Logger::to_wp_error()`:

```json
{
  "code": "dback_export_failed",
  "message": "Human-readable exception message",
  "data": {
    "status": 500,
    "error_id": "kQ3U8JHl",
    "operation": "export",
    "logged_at": "2026-06-02T08:09:29+00:00",
    "details": {
      "exception": "RuntimeException"
    }
  }
}
```

Error codes:

| Code | Operation |
|------|-----------|
| `dback_export_failed` | export |
| `dback_import_failed` | import |
| `dback_query_failed` | query |
| `dback_forbidden` | auth failure (403) |

Logs are stored in:

- WordPress option `dback_error_log` (last 100 entries, newest first)
- File `{uploads}/dback-db-tools/dback-errors.log` (JSONL, one JSON object per line)

When debugging, search logs by `error_id`.

---

## Database driver selection

`DBack_Database::driver()` returns `pdo` or `wpdb`.

| Feature | `pdo_mysql` available | Fallback (`wpdb`/mysqli) |
|---------|----------------------|---------------------------|
| Export | `ifsnop/mysqldump-php` with `GZIPSTREAM` | `DBack_Exporter_Mysqli` |
| Import | PDO `exec()` | `$wpdb->query()` |
| Query | PDO | `$wpdb->get_results()` / `$wpdb->query()` |

**Important:** WordPress runs fine with only **mysqli**. Many Docker/minimal PHP images lack **pdo_mysql**. The plugin must work without it.

`DBack_Database::has_pdo_mysql()` checks:

```php
class_exists('PDO') && in_array('mysql', PDO::getAvailableDrivers(), true)
```

Never reference `PDO::MYSQL_ATTR_USE_BUFFERED_QUERY` without confirming PDO mysql driver exists.

---

## Export internals

### Path A — PDO + mysqldump-php (preferred)

Used when `has_pdo_mysql()` is true.

- Library: `vendor/ifsnop/mysqldump-php`
- Compression: `Mysqldump::GZIPSTREAM` (not `GZIP`) — streams to `php://output`
- Settings: utf8mb4, add-drop-table, single-transaction, lock-tables off, hex-blob, extended-insert
- PDO: unbuffered queries for dump (`MYSQL_ATTR_USE_BUFFERED_QUERY => false`)
- Ends with `exit` after streaming (bypasses REST JSON wrapper)

### Path B — mysqli fallback

Used when PDO mysql is unavailable.

- Class: `DBack_Exporter_Mysqli`
- Uses global `$wpdb` and native `mysqli` unbuffered reads (`MYSQLI_USE_RESULT`) for large tables
- Dumps: tables (CREATE + INSERT batches), views (CREATE VIEW)
- Batch size: 100 rows per INSERT
- Output via `DBack_Gzip_Stream`

Does **not** dump triggers, routines, or events (mysqldump-php path may support more when PDO is available).

### Gzip streaming (`DBack_Gzip_Stream`)

**Do not use `gzopen('php://output')`** — it fails on many hosts and inside WP REST.

Correct pattern (same as mysqldump-php `GZIPSTREAM`):

```php
$out = fopen('php://output', 'wb');
$ctx = deflate_init(ZLIB_ENCODING_GZIP, ['level' => 9]);
fwrite($out, deflate_add($ctx, $data, ZLIB_NO_FLUSH));
// on close:
fwrite($out, deflate_add($ctx, '', ZLIB_FINISH));
```

Before streaming:

- Clear output buffers (`ob_end_clean`)
- Disable `zlib.output_compression`
- Set download headers manually
- Call `fflush` + `flush()` periodically to keep connection alive

---

## Import internals

- Accepts **gzip-compressed SQL** only (`.sql.gz`)
- Parser is line-based; skips `--` and `#` comments
- Splits on semicolon at end of line
- Not a full SQL parser — complex dumps with semicolons inside strings may break
- Compatible with dumps produced by this plugin and standard mysqldump output

For very large imports: `set_time_limit(0)`, temp file on disk, statement-by-statement execution (low memory).

---

## Admin UI

Location: **Tools → DBack DB Tools** (`tools_page_dback-db-tools`)

Sections:

1. API key display + regenerate
2. Export button
3. Import file picker
4. SQL textarea + Run Query
5. Error log table (auto-refresh on failure)

Uses native WordPress admin styles (`wrap`, `button`, `notice`, `widefat`). No custom CSS theme.

JavaScript: `assets/admin.js` — plain `fetch()` against REST, not `wp.apiFetch`.

---

## PHP requirements

| Extension | Required | Notes |
|-----------|----------|-------|
| mysqli | de facto yes | WordPress needs it |
| zlib (`deflate_init`) | yes | export gzip |
| `gzopen` | yes | import gzip |
| pdo_mysql | optional | enables full mysqldump-php export |

PHP >= 7.4, WordPress >= 5.8.

---

## Common failures and fixes

| Symptom | Cause | Fix |
|---------|-------|-----|
| `Undefined constant PDO::MYSQL_ATTR_USE_BUFFERED_QUERY` | pdo_mysql not loaded; constant used unconditionally | Guard with `has_pdo_mysql()` before using PDO MySQL constants |
| `Unable to open gzip output stream` | `gzopen('php://output')` unsupported | Use `DBack_Gzip_Stream` (`fopen` + `deflate_*`) |
| `pdo_mysql is required` | Old code path hard-required PDO | Use mysqli fallback exporter |
| Export timeout (504) | No data sent for 30s+ | Ensure streaming + flush; client must read body as stream |
| Import fails mid-file | SQL syntax or semicolon in string | Check error log; may need smarter parser |
| REST returns HTML instead of gzip | Auth failed or PHP fatal before headers | Check `/logs`, enable WP debug log |

---

## DBack Go client integration notes

When wiring the Go app to this plugin:

1. Base URL: `{wp_url}/wp-json/dback/v1`
2. Auth header: `X-DBACK-KEY`
3. **Export:** `GET /export`, stream `resp.Body` to local `.sql.gz` file
4. **Import:** `POST /import`, `Content-Type: application/gzip`, body = file bytes
5. **Query:** `POST /query`, JSON body `{"sql":"..."}`
6. On error, parse JSON for `code`, `message`, `data.error_id`
7. Use HTTP streaming (chunked read) for export — do not wait for full body in memory on huge DBs

Legacy Go fields: `TransferSettings.WPUrl`, `TransferSettings.WPKey` exist in models but WordPress connection type is currently disabled in the UI.

---

## Modification guidelines for AI agents

### Safe changes

- Improve mysqli fallback dump (triggers, better escaping)
- Better SQL import parser
- Admin UX improvements using WP core styles
- More structured error context in logs
- Go client integration in `backend/` or `internal/app/`

### Requires care

- Changing REST routes or auth — breaks DBack clients and legacy template contract
- Switching gzip implementation — test on Docker WordPress without pdo_mysql
- Adding Composer dependency at runtime — shared hosts may not have it; vendor libraries inline instead

### Do not do

- Add shell/exec-based backup
- Buffer full database in PHP memory
- Remove mysqli fallback in favor of PDO-only
- Store API key in plugin source code (use WP option)
- Commit real API keys or SQL dumps

### Testing checklist

1. Export with only mysqli (no pdo_mysql) — must produce valid `.sql.gz`
2. Export with pdo_mysql — mysqldump-php path works
3. Import exported file back — round-trip
4. `POST /query` with `SHOW TABLES`
5. Invalid API key → 403 JSON
6. Induce error → appears in `/logs` with `error_id`
7. Admin UI export/import/query from **Tools → DBack DB Tools**

### Load order

New classes must be `require_once` in `dback-db-tools.php` **before** classes that depend on them. Hooks register in `DBack_DB_Tools_Plugin::__construct()`.

---

## Versioning

Plugin header version: `1.0.0` (`DBACK_DB_TOOLS_VERSION`).

When making breaking REST changes, bump version and document migration. Prefer backward-compatible additions (new optional JSON fields, new routes) over breaking existing `dback/v1` contract.

---

## Quick reference — curl

```bash
# Export
curl -H "X-DBACK-KEY: KEY" "https://site/wp-json/dback/v1/export" -o dump.sql.gz

# Import
curl -X POST -H "X-DBACK-KEY: KEY" -H "Content-Type: application/gzip" \
  --data-binary @dump.sql.gz "https://site/wp-json/dback/v1/import"

# Query
curl -X POST -H "X-DBACK-KEY: KEY" -H "Content-Type: application/json" \
  -d '{"sql":"SELECT COUNT(*) AS c FROM wp_posts"}' \
  "https://site/wp-json/dback/v1/query"

# Logs
curl -H "X-DBACK-KEY: KEY" "https://site/wp-json/dback/v1/logs"
```
