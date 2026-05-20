![DBack](dback.png)

# DB Sync Manager

![DB Sync Manager Screenshot](desgin/app.png)

A Linux desktop GUI application built with Go and [Gio](https://gioui.org) for managing MySQL/MariaDB backups and restores. DBack connects to remote Linux servers via SSH or Jump Host, WordPress sites via a generated plugin, or databases running inside Docker containers, then streams backups to local files and restores them to any saved host.

**Repository:** [https://github.com/devlifeX/dback/](https://github.com/devlifeX/dback/)

## Features

### 🔌 Connectivity
*   **SSH / Jump Host:** Connect to remote Linux servers using password or private key authentication.
*   **WordPress:** Direct integration with WordPress sites via a secure, auto-generated plugin (no SSH required).
*   **Docker:** Support for MySQL/MariaDB running inside Docker containers.
*   **Databases:** **MySQL** and **MariaDB** only.

### 🚀 Core Functions
*   **Backup:** Stream large database dumps (5GB+) with on-the-fly compression.
    *   **Smart transfer:** Streaming first, with automatic fallback to remote tmp-file on retryable SSH errors.
    *   **Smart compression:** Uses `zstd` when available, falling back to `gzip`.
*   **Restore:** Stream uploads or tmp-file import to remote servers; choose any host as the destination.
*   **Backup Center:** Backup and restore jobs appear in the **Backups** screen with progress and cancel controls.
*   **Clickable Backup History:** Select a saved backup record and restore it to any destination host.
*   **Preflight checks:** Linux OS, database tools, compression, disk space, and Docker/container validation before each job.
*   **Secure:** Credentials are shell-escaped; secrets are never written to logs.
*   **Reliable:** Uses `pipefail` so backup failures are caught even if compression succeeds.

### 👤 Host Management
*   **Hosts:** Each host is an independent profile with one connection, backup folder, and pre/post import queries.
*   **Backup:** Uses the selected host's own settings.
*   **Restore:** Pick a backup, then pick the destination host.
*   **Profile transfer:** Export/import hosts from **Settings**. Without a master password, secrets are excluded; with secrets enabled, bundles are encrypted (Argon2id + AES-256-GCM).
*   **Non-destructive import:** Imported hosts merge with existing ones by ID or name.
*   **Filename formatting:** Backups are named with host, database, and timestamp.

### 📋 SQL Templates
*   **Template manager:** Create, edit, and delete reusable SQL snippets.
*   **Append to queries:** Append templates to pre/post import queries without overwriting existing SQL.
*   **Placeholders:** `{databasename}`, `{host}`, `{profile}`, `{dbuser}`.

### 🛠️ Diagnostics
*   **Test connectivity:** Verify server (SSH/HTTP) and database connections before heavy operations.
*   **Structured logs:** Operation ID, phase, strategy, attempts, and masked commands in `logs.json`.

### ⚡ Why is it so fast?
*   **Direct streaming:** Data flows from the database to the destination file without stopping.
*   **Native tools:** Uses `mysqldump` / `mariadb-dump` already on the server.
*   **Smart compression:** Prefers **Zstandard (zstd)** when available.
*   **No temp files by default:** Streaming avoids filling server disk; tmp-file fallback only when needed.

## Installation & Running

### Run via Script (Linux)
This script handles dependency checks and runs the application.

```bash
./run.sh
```

**Debug logging (stderr):** pass `--debug` or set `DBACK_DEBUG=1`:

```bash
./run.sh --debug
DBACK_DEBUG=1 ./run.sh
./dist/dback-linux --debug
```

*On Debian/Ubuntu, install Gio build dependencies if prompted:*

```bash
sudo apt-get update && sudo apt-get install -y \
  build-essential pkg-config libvulkan-dev xorg-dev libwayland-dev \
  libxkbcommon-dev libxkbcommon-x11-dev libx11-xcb-dev libxcursor-dev \
  libxfixes-dev libegl-dev
```

### Build (Linux only)

```bash
./build.sh
# or directly:
./build.sh linux
```

Artifact: `dist/dback-linux`

## Build Requirements

*   **Go 1.21+**
*   **Linux:** `gcc`, `pkg-config`, `libvulkan-dev`, `xorg-dev`, `libwayland-dev`, `libxkbcommon-dev`, `libxkbcommon-x11-dev`, `libx11-xcb-dev`, `libxcursor-dev`, `libxfixes-dev`, `libegl-dev`

### Docker Alternative
If you have issues with system dependencies, you can run the app in a container:

```bash
./docker-run.sh
```

## Default Paths

*   **App config:** `~/.config/dback`
*   **Default backups:** `~/dback/backups` (per-host destination can be customized)

## WordPress Integration Guide

1.  Open or create a host from **Hosts**.
2.  Select **Type: WordPress**.
3.  Click **Generate WordPress Plugin** and save the `dback-sync-plugin.zip`.
4.  Install the plugin on your WordPress site (Plugins > Add New > Upload).
5.  Enter your WordPress **URL**; the **API Key** is filled automatically.
6.  Test connectivity or start a backup from the host card.

## About

Created by **dariush vesal**.

*   Email: `dariush.vesal@gmail.com`
*   GitHub: [https://github.com/devlifeX/dback](https://github.com/devlifeX/dback)

## FAQ

### Why does this app require Vulkan/X11 libraries on Linux?
DBack uses **Gio**, a GPU-accelerated GUI toolkit. On Linux, Gio renders through Vulkan and creates windows through X11/Wayland, which requires the corresponding development headers at build time.

### Which platforms are supported?
DBack targets **Linux desktop** only. macOS, Windows, and Android builds are not part of this release.
