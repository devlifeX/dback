# DBack DB Tools — WordPress Plugin Agent Guide

This document describes how the **DBack DB Tools** WordPress plugin works. It is written for AI agents and developers working in the `dback` monorepo or on a WordPress site where this plugin is installed.

---

## Purpose

The plugin exposes **pure-PHP** database operations for the **DBack** desktop app (Go) and for WordPress administrators:

- **Export** — stream a `.sql.gz` backup of the WordPress database
- **Import** — restore from a `.sql.gz` file (optional target database)
- **Run SQL** — execute arbitrary SQL against the WordPress or target database
- **Ping / preflight** — connection test and environment checks for DBack
- **Error log** — structured error logging for admin and REST clients

It replaces the legacy single-file template at `plugin_template/dback-sync.php`, which used `exec()` / shell commands (`mysqldump`, `mysql`, `gunzip`). **This plugin must never use shell commands.**

Target environment: **heavy databases** and **shared hosting** where CLI tools and long-running shell processes are unavailable or unreliable.

---

## Location in repo

```
wordpress/dback-db-tools/          ← plugin source (also embedded in Go binary)
├── dback-db-tools.php             ← plugin bootstrap; {{DBACK_API_KEY}} placeholder for zip build
├── embed.go                       ← //go:embed for Go app (not loaded by WordPress)
├── wordpress_agent.md             ← this file
├── includes/                      ← PHP classes
├── assets/admin.js                ← admin UI (REST client)
└── vendor/ifsnop/mysqldump-php/   ← vendored dump library (no Composer at runtime)
```

Go app integration (same monorepo):

| Item | Path |
|------|------|
| REST client | `backend/wordpress/client.go` |
| Plugin zip build | `backend/wordpress/pluginzip.go` — `BuildPluginZip(siteURL, apiKey)` |
| Embed | `wordpress/dback-db-tools/embed.go` |
| Backup / restore | `backend/transfer/transfer_wordpress.go` |
| Host UI | `ui/settings_form.go` — Generate Token, Download Plugin |

Legacy reference (do not extend unless explicitly asked):

```
plugin_template/dback-sync.php     ← old exec()-based stub; same REST namespace
```

The Go desktop app **actively supports** WordPress as a Host type (`ConnectionTypeWordPress`). Users download a site-specific zip from DBack; backup, import, query, connection test, and preflight all go through this plugin’s REST API.

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
        Ping[/ping GET]
        Preflight[/preflight GET]
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
| `class-dback-api-key.php` | Hardcoded key (`DBACK_HARDCODED_API_KEY`) + option `dback_api_key` fallback |
| `class-dback-preflight.php` | Environment checks for `/preflight` (PHP, zlib, DB, uploads, temp dir) |
| `class-dback-database.php` | Credentials, DSN, PDO/wpdb abstraction, temp paths |
| `class-dback-exporter.php` | Export entry: PDO path or delegate to mysqli fallback |
| `class-dback-exporter-mysqli.php` | Pure `$wpdb`/mysqli dump when PDO unavailable |
| `class-dback-gzip-stream.php` | Gzip write to `php://output` via `fopen` + `deflate_*` |
| `class-dback-importer.php` | Gzip import: temp file → line parser → `DBack_Database::exec` |
| `class-dback-query-runner.php` | SQL runner; returns rows or affected count |
| `class-dback-rest-controller.php` | REST routes, auth, error wrapping |
| `class-dback-diagnostics.php` | Admin diagnostics, route checks, plugin list links |
| `class-dback-error-logger.php` | Log to option + JSONL file; build `WP_Error` |
| `class-dback-admin-page.php` | Tools → DBack DB Tools admin page |
| `assets/admin.js` | Admin forms calling REST API |

---

## Constants

| Constant | Value |
|----------|-------|
| `DBACK_DB_TOOLS_VERSION` | Plugin version string |
| `DBACK_DB_TOOLS_REST_NAMESPACE` | `dback/v1` |
| `DBACK_HARDCODED_API_KEY` | Site token from DBack download; placeholder `{{DBACK_API_KEY}}` in source |
| `DBack_Api_Key::OPTION_NAME` | `dback_api_key` |
| `DBack_Api_Key::PLACEHOLDER` | `{{DBACK_API_KEY}}` — treated as unset |
| `DBack_Error_Logger::OPTION_KEY` | `dback_error_log` |
| Temp directory | `{uploads}/dback-db-tools/` |
| Error log file | `{uploads}/dback-db-tools/dback-errors.log` |

---

## Authentication

### External client (DBack app)

```http
X-DBACK-KEY: {site token}
```

Validation order in `DBack_Api_Key::is_valid()`:

1. **Hardcoded key** — `DBACK_HARDCODED_API_KEY` in `dback-db-tools.php` (replaced when DBack builds the download zip). Primary path for DBack Host profiles.
2. **WordPress option** — `dback_api_key` (32-char random string). Generated on activation; shown in **Tools → DBack DB Tools** and regeneratable there. Fallback for manual installs or admin UI.

Both keys are checked with `hash_equals()`. Either match grants access.

Optional target database for import and query:

```http
X-DBACK-DATABASE: {database_name}
```

Also accepted as JSON/query param `database` on `/import` and `/query`. Empty → WordPress default `DB_NAME`. Non-empty → create DB if missing, then select it for the operation.

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
| `GET /ping` | yes | yes |
| `GET /preflight` | yes | yes |
| `GET /export` | yes | yes |
| `POST /import` | yes | yes |
| `POST /query` | yes | yes |
| `GET /logs` | yes | yes |
| `DELETE /logs` | no | yes only |

---

## REST API reference

Base URL: `https://{site}/wp-json/dback/v1`

### Ping (connection test)

```http
GET /ping
X-DBACK-KEY: {key}
```

**Success:**

```json
{
  "success": true,
  "message": "pong",
  "plugin_version": "1.0.0",
  "site_url": "https://example.com",
  "driver": "wpdb"
}
```

Used by DBack **Test Connection** for WordPress hosts.

### Preflight

```http
GET /preflight
X-DBACK-KEY: {key}
```

**Success** (all checks pass):

```json
{
  "success": true,
  "plugin_version": "1.0.0",
  "php_version": "8.2.0",
  "wordpress_version": "6.5",
  "site_url": "https://example.com",
  "driver": "wpdb",
  "db_version": "SELECT 1 ok",
  "checks": [
    {"name": "php_version", "status": "ok", "details": "PHP 8.2.0"},
    {"name": "zlib", "status": "ok", "details": "gzip available"},
    {"name": "pdo_mysql", "status": "ok", "details": "wpdb"},
    {"name": "uploads_writable", "status": "ok", "details": "uploads writable"},
    {"name": "temp_dir", "status": "ok", "details": "temp dir ready"},
    {"name": "database", "status": "ok", "details": "SELECT 1 ok"}
  ]
}
```

When any check fails, `success` is `false` and failed entries have `"status": "fail"`. DBack surfaces this before backup/import.

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
X-DBACK-DATABASE: {optional_target_db}
Content-Type: application/gzip
Body: raw .sql.gz bytes
```

**Success:**

```json
{
  "success": true,
  "message": "Database imported successfully.",
  "statements_executed": 42,
  "bytes_received": 1048576,
  "database": "my_restore_db"
}
```

Import flow:

1. Read gzip body from `$request->get_body()` — **not** `php://input` (WP REST consumes the input stream before the callback)
2. Resolve target DB from `X-DBACK-DATABASE` header or `database` param; empty uses `DB_NAME`
3. `DBack_Database::prepare_import_target()` — create/select non-default DB if needed
4. Save body to temp file under uploads (`import-{random}.sql.gz`)
5. `gzopen` + read line-by-line
6. Accumulate SQL until line ends with `;`
7. Execute via `DBack_Database::exec()` against active target
8. Delete temp file; reset import target connection state

### Query

```http
POST /query
X-DBACK-KEY: {key}
X-DBACK-DATABASE: {optional_target_db}
Content-Type: application/json

{"sql": "SHOW TABLES", "database": "optional_alt_to_header"}
```

**Pre/post import queries from DBack use the same target as restore when connected to the import database (`connectDB=true`). Before-import queries run without selecting `TargetDBName` so statements like `DROP DATABASE` / `CREATE DATABASE` work.**

Multi-statement scripts (semicolon-separated) are split and executed sequentially. Batch response:

```json
{
  "success": true,
  "type": "batch",
  "statements_executed": 2,
  "statements": [ ... ],
  "driver": "wpdb",
  "database": "wordpress"
}
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

### Target database selection

`DBack_Database::with_target_database()` and `prepare_import_target()` handle import and query against a non-default database:

| Request value | Behavior |
|---------------|----------|
| Empty / omitted | Use WordPress `DB_NAME` |
| Valid name (`^[A-Za-z0-9_]{1,64}$`) | `CREATE DATABASE IF NOT EXISTS`, then `USE` / `$wpdb->select()` |

**wpdb note:** On modern WordPress, `$wpdb->select()` returns `null` on success. Verify selection with `$wpdb->ready` and `SELECT DATABASE()`, not the return value of `select()`.

After import/query, `reset_import_target()` restores the default WordPress connection.

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

Also linked from **Plugins → DBack DB Tools → Status & Logs / Diagnostics**.

Sections:

1. **Status & Diagnostics** — REST availability, registered routes, internal ping test, endpoint URLs, permalink warning, auth mode, active plugin list (for conflict diagnostics)
2. API key display + regenerate (hardcoded DBack token status shown separately)
3. Export button
4. Import file picker
5. SQL textarea + Run Query
6. Debug log table (errors, warnings, info — auto-refresh on failure)

Uses native WordPress admin styles (`wrap`, `button`, `notice`, `widefat`). No custom CSS theme.

JavaScript: `assets/admin.js` — plain `fetch()` against REST, not `wp.apiFetch`. Shows extra hint when `rest_no_route` is returned.

### Diagnostics (`DBack_Diagnostics`)

Rendered server-side on page load (works even when browser REST calls fail):

- Plugin active / folder / version
- Hardcoded token configured vs WordPress option key
- REST index URL and `dback/v1` namespace URL
- Pretty permalinks enabled or plain
- List of registered `/dback/v1/*` routes
- List of active plugins with version and plugin file path (including network-active plugins on multisite)
- Internal `rest_do_request()` tests for `/ping` and `/preflight`
  - `rest_no_route` → routes not registered (plugin inactive or bootstrap failed)
  - `dback_forbidden` → routes OK, auth required (expected without key in internal test)
- Snapshot written to debug log on each diagnostics page view

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
| Import succeeds but DB unchanged | Body read from `php://input` in REST context | Use `$request->get_body()` in `DBack_Importer::import_rest_request()` |
| `Unable to select database X` | `$wpdb->select()` return value trusted | Verify with `$wpdb->ready` + `SELECT DATABASE()` |
| Plugin not recognized after upload | Extra folder nesting in zip | Use DBack-generated zip; see **Plugin distribution** below |
| `rest_no_route` from DBack | Plugin inactive, wrong Site URL, or routes not registered | Open **Plugins → Status & Diagnostics**; activate plugin; match DBack Site URL to WordPress **Settings → General** |
| Import/query hits wrong database | Missing target header | Send `X-DBACK-DATABASE` when `TargetDBName` is set in DBack profile |

---

## Plugin distribution (DBack download)

The Go app embeds plugin files and builds a per-site zip when the user clicks **Download Plugin** in the WordPress Host form.

**Build flow** (`backend/wordpress/pluginzip.go`):

1. Walk embedded files from `wordpress/dback-db-tools/embed.go`
2. Skip paths rejected by `release_zip.go` — `IncludeInReleaseZip()` (see **Release zip exclusions** below)
3. Replace `{{DBACK_API_KEY}}` in `dback-db-tools.php` with the host’s generated token
4. Pack into zip with a **single stable top-level folder** named `dback-db-tools`

### Release zip exclusions

Defined in `wordpress/dback-db-tools/release_zip.go`. These files stay in the repo (and may be embedded for tooling) but are **not** shipped in the user download zip:

| Rule | Examples |
|------|----------|
| Exact names (`ReleaseZipExcludeNames`) | `embed.go`, `release_zip.go`, `wordpress_agent.md` |
| Extensions (`ReleaseZipExcludeExtensions`) | `.md`, `.go` |

Add new dev-only files to `ReleaseZipExcludeNames` or extend the extension list when introducing docs or Go sources under the plugin tree. Runtime PHP/JS/assets under `includes/`, `assets/`, and `vendor/` (except excluded types) are included.

**Output filename:** `dback-{hostname}-{pluginVersion}.zip`  
Example: `dback-florancewatch.com-1.0.0.zip`

**Zip internal layout** (WordPress-compatible):

```
dback-db-tools/
├── dback-db-tools.php      ← main plugin file at folder root
├── includes/
├── assets/
├── vendor/
└── index.php
```

The zip filename keeps the host/version pattern, but the internal root folder stays `dback-db-tools/` so extracting a newer zip overwrites the previous plugin folder instead of creating versioned directories. Upload the `.zip` directly in **Plugins → Add New → Upload Plugin** — do not re-zip an extracted folder.

Manual dev install: copy the `wordpress/dback-db-tools/` folder to `wp-content/plugins/dback-db-tools/` (no token replacement; uses `dback_api_key` option instead).

### Stable API token (DBack Host)

The token is stored in the DBack host profile (`WPKey`) and embedded in the downloaded plugin as `DBACK_HARDCODED_API_KEY`.

- **Re-downloading the plugin does not change the token** — the same profile key is embedded each time.
- A new token is created only when:
  - The user clicks **Generate Token** in DBack, or
  - The host is saved with an empty key (auto-generated once on save).
- After downloading, **Save Host** in DBack so the token persists in the vault.
- On plugin activation, if a hardcoded key exists it is copied to the `dback_api_key` option for display in admin.

---

## DBack Go client integration

WordPress is a first-class Host type in the Go app. See also [`agent.md`](../../agent.md) (repo root).

| Concern | Go location |
|---------|-------------|
| Profile fields | `models.Profile` — `WPUrl`, `WPKey`, `ConnectionTypeWordPress` |
| REST client | `backend/wordpress/client.go` — `Ping`, `Preflight`, `Export`, `Import`, `Query` |
| Backup | `backend/transfer/transfer_wordpress.go` — `BackupWordPress` |
| Restore | `transfer.RestoreWordPress` after pre-import query in `internal/app/restore_queries.go` |
| Connection test | `internal/app/connectiontest.go` → `GET /ping` |
| Preflight step | `GET /preflight` before backup/import |
| Plugin zip | `internal/app/pluginzip.go` → `BuildWordPressPluginZip` |

**Restore order (Go `App.Restore`):**

1. Pre-import query via `POST /query` when `PreImportQuery` is non-empty (no `X-DBACK-DATABASE` — server default connection)
2. Abort restore if pre-import fails — **import is not started**
3. Plugin preflight + `POST /import`
4. Post-import query (optional, with `X-DBACK-DATABASE` when target DB is set)

**Client contract:**

1. Base URL: `{wp_url}/wp-json/dback/v1`
2. Auth: `X-DBACK-KEY` from profile `WPKey` (must match hardcoded key in downloaded plugin)
3. Optional: `X-DBACK-DATABASE` from `Profile.TargetDBName` on import and query (empty → site default DB)
4. **Ping:** `GET /ping` — connection test
5. **Preflight:** `GET /preflight` — environment checks before ops
6. **Export:** `GET /export` — stream `resp.Body` to local `.sql.gz` (chunked read; do not buffer huge DBs)
7. **Import:** `POST /import`, `Content-Type: application/gzip`, body = file bytes; validate `statements_executed > 0`
8. **Query:** `POST /query`, JSON `{"sql":"..."}`; parse `type` (`result` vs `command`)
9. On error: parse JSON `code`, `message`, `data.error_id`; correlate with `GET /logs`

---

## Modification guidelines for AI agents

### Safe changes

- Improve mysqli fallback dump (triggers, better escaping)
- Better SQL import parser
- Admin UX improvements using WP core styles
- More structured error context in logs
- Extend preflight checks or ping payload
- Go client changes in `backend/wordpress/` (keep REST contract stable)

### Requires care

- Changing REST routes or auth — breaks DBack clients and legacy template contract
- Switching gzip implementation — test on Docker WordPress without pdo_mysql
- Adding Composer dependency at runtime — shared hosts may not have it; vendor libraries inline instead

### Do not do

- Add shell/exec-based backup
- Buffer full database in PHP memory
- Remove mysqli fallback in favor of PDO-only
- Commit real API keys or production tokens in git (use `{{DBACK_API_KEY}}` placeholder only)
- Remove hardcoded key support without updating DBack zip download flow
- Commit real API keys or SQL dumps
- Ship plugin changes without bumping `DBACK_DB_TOOLS_VERSION` (see **Versioning**)

### Testing checklist

1. `GET /ping` with valid hardcoded key and with option-only key
2. `GET /preflight` — all checks `ok` on healthy site
3. Export with only mysqli (no pdo_mysql) — must produce valid `.sql.gz`
4. Export with pdo_mysql — mysqldump-php path works
5. Import exported file back — round-trip; `statements_executed > 0`
6. Import with `X-DBACK-DATABASE` to non-default DB — creates/selects DB
7. `POST /query` with `SHOW TABLES` on default and target DB
8. Invalid API key → 403 JSON
9. Induce error → appears in `/logs` with `error_id`
10. Admin UI export/import/query from **Tools → DBack DB Tools**
11. DBack **Download Plugin** zip — upload in WP admin without re-zipping; plugin activates
12. **Status & Diagnostics** shows registered routes and internal ping test
13. Re-download plugin zip — token in file unchanged when profile key unchanged

### Load order

New classes must be `require_once` in `dback-db-tools.php` **before** classes that depend on them. Hooks register in `DBack_DB_Tools_Plugin::__construct()`.

---

## Versioning

Plugin header version and `DBACK_DB_TOOLS_VERSION` must stay in sync (currently **1.1.3**).

### Required on every plugin change

When you modify anything under `wordpress/dback-db-tools/`:

1. **Bump the version** in both places:
   - Plugin header `Version:` in `dback-db-tools.php`
   - `define('DBACK_DB_TOOLS_VERSION', '…')` in the same file
2. Rebuild/re-embed the Go app if shipping the desktop binary (embedded plugin template).
3. Document notable changes in this file if behavior or REST contract changed.

Use semantic-ish increments:

| Change type | Example bump |
|-------------|--------------|
| Bug fix, diagnostics, internal refactor | 1.1.1 → 1.1.2 |
| New REST field or admin feature (backward compatible) | 1.1.1 → 1.2.0 |
| Breaking REST or auth change | 2.0.0 |

When making breaking REST changes, document migration. Prefer backward-compatible additions (new optional JSON fields, new routes) over breaking existing `dback/v1` contract.

**Last aligned with:** v1.1.3 — diagnostics now include active plugin inventory (name/version/file) to speed up REST route conflict debugging.

---

## Quick reference — curl

```bash
# Ping
curl -H "X-DBACK-KEY: KEY" "https://site/wp-json/dback/v1/ping"

# Preflight
curl -H "X-DBACK-KEY: KEY" "https://site/wp-json/dback/v1/preflight"

# Export
curl -H "X-DBACK-KEY: KEY" "https://site/wp-json/dback/v1/export" -o dump.sql.gz

# Import (optional target database)
curl -X POST -H "X-DBACK-KEY: KEY" -H "X-DBACK-DATABASE: my_db" \
  -H "Content-Type: application/gzip" \
  --data-binary @dump.sql.gz "https://site/wp-json/dback/v1/import"

# Query (optional target database)
curl -X POST -H "X-DBACK-KEY: KEY" -H "X-DBACK-DATABASE: my_db" \
  -H "Content-Type: application/json" \
  -d '{"sql":"SELECT COUNT(*) AS c FROM wp_posts"}' \
  "https://site/wp-json/dback/v1/query"

# Logs
curl -H "X-DBACK-KEY: KEY" "https://site/wp-json/dback/v1/logs"
```
