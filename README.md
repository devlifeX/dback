![DBack](dback.png)

# DB Sync Manager

![DB Sync Manager Screenshot](desgin/app.png)

A cross-platform desktop GUI application built with Go and [Gio](https://gioui.org) for managing database backups and restores. DBack can connect to remote Linux servers via SSH, WordPress sites via a generated plugin, or databases running inside Docker containers, then stream backups directly to local files and restore them to another profile.

**Repository:** [https://github.com/devlifeX/dback/](https://github.com/devlifeX/dback/)

## Features

### 🔌 Connectivity
*   **SSH:** Connect to any remote Linux server using Password or Private Key authentication.
*   **WordPress:** Direct integration with WordPress sites via a secure, auto-generated plugin (no SSH required).
*   **Docker:** Seamless support for databases running inside Docker containers.
*   **Databases:** Support for **MySQL**, **MariaDB**, **PostgreSQL**, and **CouchDB**.

### 🚀 Core Functions
*   **Export (Backup):** Stream large database dumps (5GB+) with on-the-fly compression.
    *   **Smart Compression:** Automatically detects and uses `zstd` if available for faster compression, falling back to `gzip`.
*   **Import (Restore):** Stream uploads and restores to remote servers or local instances.
*   **Backup Center:** Backup and import jobs appear in the **Backups** screen with progress and cancel controls.
*   **Clickable Backup History:** Select a saved backup record and import it into any destination profile.
*   **Secure:** All database credentials are shell-escaped to prevent command injection.
*   **Reliable:** Uses `pipefail` to ensure backup failures are caught even if compression succeeds.

### 👤 Profile Management
*   **Hosts:** Create, edit, group, search, and delete saved host profiles.
*   **Separate Export/Import Settings:** Each profile can keep different settings for source backup and destination restore.
*   **Copy Settings:** Copy Export settings to Import, or Import settings to Export, to avoid duplicate typing.
*   **Profile Transfer:** Export and import all profiles from **Settings**. Passwords/API keys are excluded unless explicitly included.
*   **Filename Formatting:** Exports are named with the profile, database name, and timestamp for easy organization.

### 📱 UI and Mobile
*   **Modern Desktop UI:** Dark-themed Gio layout with sidebar navigation, cards, and tabbed profile editing.
*   **Mobile-Ready Architecture:** UI state, navigation, and platform services are structured so Android support can be added later without another full rewrite.
*   **Conditional Fields:** SSH and WordPress fields are shown only when relevant.
*   **About Screen:** Includes project and author information inside the app.

### 🛠️ Diagnostics
*   **Test Connectivity:** Built-in tools to verify Server (SSH/HTTP) and Database connections before running heavy operations.
*   **Comprehensive Logs:** detailed error logs capture remote command output for debugging.

### ⚡ Why is it so fast?
This app is designed for speed by removing common bottlenecks:
*   **Direct Streaming:** It operates like a pipeline. Data flows directly from the database to the destination file without stopping. It doesn't wait to "download" the file before saving it; it saves it *while* it downloads.
*   **No Middleman:** It uses the official, high-speed tools already installed on your server (like `mysqldump`, `pg_dump`, and `tar`) instead of trying to process the data itself.
*   **Smart Compression:** It automatically uses **Zstandard (zstd)** if available. Zstd is a modern compression tool that is significantly faster than traditional methods like ZIP or GZIP, meaning less waiting for files to compress.
*   **No Temporary Files:** By streaming data directly over the network (SSH), it avoids creating massive temporary backup files that fill up your server's disk space and slow things down.

## Installation & Running

### Run via Script (Linux)
This script handles dependency checks and runs the application.

```bash
./run.sh
```

**Debug logging (stderr):** pass `--debug` or set `DBACK_DEBUG=1` to mirror activity and errors to the terminal (useful when running from a TTY):

```bash
./run.sh --debug
DBACK_DEBUG=1 ./run.sh
./dist/dback-linux --debug
```

*Note: You may need to install Gio build dependencies if prompted. On Debian/Ubuntu:*

```bash
sudo apt-get update && sudo apt-get install -y \
  build-essential pkg-config libvulkan-dev xorg-dev libwayland-dev \
  libxkbcommon-dev libxkbcommon-x11-dev libx11-xcb-dev libxcursor-dev \
  libxfixes-dev libegl-dev
```

### Build Binaries
Run the interactive build script and choose a target:

```bash
./build.sh
```

You can also build a target directly:

```bash
./build.sh linux
./build.sh windows
./build.sh macos
./build.sh all
```

Artifacts are written to `dist/`:

*   `dist/dback-linux`
*   `dist/dback-windows.exe`
*   `dist/dback-macos`

Android builds are intentionally deferred until Gio mobile packaging is added.

## Build Requirements

To build the application from source, you need **Go 1.21+** and the following platform-specific dependencies:

| Platform | Requirements |
| :--- | :--- |
| **Linux** | `gcc`, `pkg-config`, `libvulkan-dev`, `xorg-dev`, `libwayland-dev`, `libxkbcommon-dev`, `libxkbcommon-x11-dev`, `libx11-xcb-dev`, `libxcursor-dev`, `libxfixes-dev`, `libegl-dev` |
| **Windows** | `gcc` (MinGW-w64 or TDM-GCC) |
| **macOS** | Xcode Command Line Tools (`xcode-select --install`) |
| **Cross-Compile (Linux -> Windows)** | `mingw-w64` (`gcc-mingw-w64`) |
| **Cross-Compile (Linux -> macOS)** | `zig` or `osxcross` |
| **Android** | Planned for a future Gio mobile release |

### Docker Alternative
If you have issues with system dependencies, you can run the app in a container:

```bash
./docker-run.sh
```

## WordPress Integration Guide

1.  Open or create a host profile from **Hosts**.
2.  In the Export or Import settings, select **Type: WordPress**.
3.  Click **Generate WordPress Plugin** and save the `dback-sync-plugin.zip`.
4.  **Install** this plugin on your WordPress site (Plugins > Add New > Upload).
5.  Copy your WordPress **URL** into the app.
6.  The **API Key** is automatically filled (it matches the key embedded in the plugin).
7.  Click **Test Export Connection**, **Test Import Connection**, or start a backup from the host card.

## About

Created by **dariush vesal**.

*   Email: `dariush.vesal@gmail.com`
*   GitHub: [https://github.com/devlifeX/dback](https://github.com/devlifeX/dback)

## FAQ

### Why does this app require Vulkan/X11 libraries on Linux?
DBack uses **Gio**, a GPU-accelerated GUI toolkit for Go. On Linux, Gio renders through Vulkan and creates windows through X11/Wayland, which requires the corresponding development headers at build time.

### Will Android be supported?
The UI is structured for future Android support, but desktop is the current target. Android packaging will be added in a later release using Gio mobile tooling.
