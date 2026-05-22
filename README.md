![DBack](dback.png)

# DBack — DB Sync Manager

**Linux desktop GUI for MySQL/MariaDB backup and restore over SSH, Jump Host, or Docker.**

DBack connects to remote Linux servers, streams database dumps to local files with compression, and restores backups to any saved host. Hosts, templates, history, and logs live in an encrypted local vault. Built with Go and [Gio](https://gioui.org).

**Repository:** [github.com/devlifeX/dback](https://github.com/devlifeX/dback/)

## Highlights

- **Streaming backups** — large dumps (5GB+) with on-the-fly `zstd`/`gzip` compression
- **Smart fallback** — retries with a remote tmp-file when SSH streams fail; supports resume and checksum validation
- **Unified hosts** — one connection, backup folder, and import queries per host
- **Encrypted vault** — profiles, templates, history, and logs stored in `app_data.vault.json`
- **Remote sync** — push/pull encrypted app settings to S3-compatible storage (AWS S3, MinIO, etc.)
- **SQL templates** — reusable snippets with placeholders for pre/post import queries
- **Modern UI** — dark GitHub-style desktop interface with sidebar navigation

## Screenshots

### Hosts
Manage remote servers, filter by group, search, and run backup actions from host cards.

![Hosts screen](screenshot/1.png)

### SQL Templates
Create reusable SQL snippets with placeholders for pre/post import queries.

![SQL template editor](screenshot/2.png)

### Host profile — import queries
Configure pre-import SQL, append templates, and test queries before restore.

![Host profile queries](screenshot/3.png)

## Features

### Connectivity
- **SSH / Jump Host** — password or private key authentication
- **Docker** — MySQL/MariaDB inside containers (`docker exec`)
- **Databases** — MySQL and MariaDB only

### Backup & Restore
- **Preflight checks** — Linux OS, dump/client tools, compression, disk space, writable tmp paths, Docker container status
- **Restore flow** — select a backup, pick a destination host, run import with that host's pre/post queries
- **Job center** — progress and cancel controls on the Backups screen

### Host Management
- **Duplicate** — clone a host with one click (`Production 1`, `Production 2`, …)
- **Groups & search** — filter by group, search by name
- **Legacy migration** — old export/import profiles are flattened on load

### App Data Transfer
Settings has two tabs: **Export** and **Sync**.

**Export / Import (local file)**
- Export/import hosts, templates, backup history metadata, and activity logs
- Backup `.sql.gz` files are **not** included in the bundle
- Encrypted export with an export password (Argon2id + AES-256-GCM)
- Non-destructive merge — conflicts prompt before overwrite

**Remote Sync (S3-compatible)**
- Push or pull the same encrypted app-data bundle to `{bucket}/dback/app-data.json`
- Works with AWS S3, MinIO, and other S3-compatible endpoints
- Configure endpoint, region, bucket, access key, secret key, and SSL
- **Test Connection** verifies bucket access and read/write permissions under `dback/`
- **Push** / **Pull** appear after a successful connection test
- Encrypted with your **vault master key** (the same key used to unlock DBack), not the export password
- Sync settings (S3 credentials) are included in the remote bundle
- Pull downloads the bundle, asks for confirmation, then merges like a local import
- **Sync log** (local only, not uploaded): last push and last pull timestamps

### SQL Templates
- Create, edit, and delete reusable SQL snippets
- Append template text to pre/post import queries
- Placeholders: `{databasename}`, `{host}`, `{profile}`, `{dbuser}`

### Diagnostics & Logging
- Test SSH and database connectivity from the host editor
- Structured activity logs (operation ID, phase, strategy, attempt, masked commands)
- Debug mode mirrors logs to stderr (`--debug` or `DBACK_DEBUG=1`)

## Quick Start

### Run from source

```bash
./run.sh
```

With debug logging:

```bash
./run.sh --debug
# or
DBACK_DEBUG=1 ./run.sh
```

### Build (Linux)

```bash
./build.sh
# or
./build.sh linux
```

Output: `dist/dback-linux`

Set the app version at build time:

```bash
APP_VERSION=1.0.0 ./build.sh linux
```

Version appears in **About** inside the app.

### Build manually

Requirements: **Go 1.21+** and Gio development libraries.

On Debian/Ubuntu:

```bash
sudo apt-get update && sudo apt-get install -y \
  build-essential pkg-config libvulkan-dev xorg-dev libwayland-dev \
  libxkbcommon-dev libxkbcommon-x11-dev libx11-xcb-dev libxcursor-dev \
  libxfixes-dev libegl-dev
```

Then:

```bash
go build -ldflags "-X main.appVersion=1.0.0" -o dist/dback-linux .
```

### Docker alternative

If system dependencies are problematic:

```bash
./docker-run.sh
```

## Default Paths

| Purpose | Path |
|---------|------|
| App config & data | `~/.config/dback` |
| Default backups | `~/dback/backups` |
| Encrypted vault | `~/.config/dback/app_data.vault.json` |
| Remote sync object | `{bucket}/dback/app-data.json` |
| SSH known hosts | `~/.config/dback/ssh_known_hosts` |

Hosts, templates, backup history, and activity logs are stored inside the encrypted vault. Each host can override the backup destination folder.

## Security

- On first launch, DBack prompts for a **master key** (minimum 8 characters; confirmation required when creating a new vault).
- Legacy plaintext files are removed after successful migration into the vault.
- **App Data Transfer** exports require an export password. **Remote Sync** uses the vault master key instead. Metadata in history and logs may still contain host names and paths.
- S3 credentials are stored in the encrypted vault and included in sync bundles — use bucket policies and least-privilege IAM where possible.
- SSH uses **TOFU host key verification** — unknown keys are saved on first connect; mismatches are rejected.
- Remote database commands validate and shell-escape profile fields to reduce injection risk.
- Password fields are masked in the UI. Error messages are sanitized unless debug mode is enabled.

## Usage

### Hosts
1. Open **Hosts** → **+ Host**
2. Configure connection (SSH, Jump Host, or Docker)
3. Set database credentials and backup destination
4. Optional: configure pre/post import queries on the **Queries** tab
5. Use **Backup** on the host card, or **Duplicate** to clone settings

### Restore
1. Open **Backups** → filter by host if needed
2. Select a backup file
3. Choose a destination host
4. Click **Import to Selected Host**

### App data export/import
1. Open **Settings** → **Export** tab
2. **Export App Data** — enter an export password
3. **Import App Data** — enter the export password for encrypted bundles

### Remote sync (S3)
1. Open **Settings** → **Sync** tab
2. Enter S3 endpoint, bucket, and credentials → **Save**
3. Click **Test Connection** — on success, **Push** and **Pull** appear
4. **Push** — uploads encrypted app data to `dback/app-data.json`
5. **Pull** — downloads the remote bundle, confirms, then merges into this device
6. Scroll to **Sync log** at the bottom to see last push/pull times (stored locally only)

## Why is it fast?

- **Direct streaming** — data flows from database to file without intermediate storage
- **Native tools** — uses `mysqldump` / `mariadb-dump` on the server
- **Smart compression** — prefers Zstandard (`zstd`) when available
- **No temp files by default** — tmp-file fallback only when streaming fails

## About

Created by **dariush vesal**.

- Email: `dariush.vesal@gmail.com`
- GitHub: [github.com/devlifeX/dback](https://github.com/devlifeX/dback/)

## FAQ

### Why does DBack require Vulkan/X11 libraries?
DBack uses **Gio**, a GPU-accelerated GUI toolkit. On Linux, Gio renders through Vulkan and creates windows via X11/Wayland, which requires development headers at build time.

### Which platforms are supported?
**Linux desktop only.** macOS, Windows, and Android are not supported in this release.

### What happened to PostgreSQL and CouchDB?
Removed. DBack now supports **MySQL and MariaDB** only.

### What happened to separate Export/Import settings per profile?
Profiles are now independent **hosts** with a single connection. Legacy dual settings are migrated automatically on first load.

### What is included in remote sync?
The same data as **Export App Data**: hosts, templates, backup history metadata, activity logs, and sync settings. Backup `.sql.gz` files and local sync timestamps (last push/pull) are **not** uploaded.

### Can I use MinIO or a self-hosted S3 endpoint?
Yes. Enter the host (with or without `https://`) and port, set **Use SSL** as needed, and provide bucket credentials. The remote object path is always `dback/app-data.json` inside your bucket.
