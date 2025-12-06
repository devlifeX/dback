# DB Sync Manager

![DB Sync Manager Screenshot](desgin/app.png)

A cross-platform desktop GUI application built with Go and Fyne v2 for managing database synchronizations. It allows you to export large databases from remote Linux servers via SSH, WordPress sites, or Docker containers, and restore them efficiently.

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
*   **Secure:** All database credentials are shell-escaped to prevent command injection.
*   **Reliable:** Uses `pipefail` to ensure backup failures are caught even if compression succeeds.

### 👤 Profile Management
*   **Profiles:** Create, Save, Clone (Duplicate), and Delete connection profiles.
*   **Smart History:** Remembers your last destination folder per profile.
*   **Filename Formatting:** Exports are named with the profile, database name, and timestamp for easy organization.

### 📊 Activity & History
*   **History Tab:** A persistent data grid view of all past operations including file sizes, status, and paths.
*   **Context Actions:** Import or delete files directly from the History list.
*   **Persistence:** All logs are saved locally to `logs.json`.

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
*Note: You may need to install `gcc`, `libgl1-mesa-dev`, and `xorg-dev` if prompted.*

### Build Binaries
To generate standalone executables for Linux, Windows, and macOS:

```bash
./build.sh
```
*Artifacts will be created as `dback-linux`, `dback-windows.exe`, and `dback-macos`.*

## Build Requirements

To build the application from source, you need **Go 1.21+** and the following platform-specific dependencies:

| Platform | Requirements |
| :--- | :--- |
| **Linux** | `gcc`, `libgl1-mesa-dev`, `xorg-dev` |
| **Windows** | `gcc` (MinGW-w64 or TDM-GCC) |
| **macOS** | Xcode Command Line Tools (`xcode-select --install`) |
| **Cross-Compile (Linux -> Windows)** | `mingw-w64` (`gcc-mingw-w64`) |
| **Cross-Compile (Linux -> macOS)** | `zig` or `osxcross` |

### Docker Alternative
If you have issues with system dependencies, you can run the app in a container:

```bash
./docker-run.sh
```

## WordPress Integration Guide

1.  Open the **Export** tab.
2.  Select **Type: WordPress**.
3.  Click **Generate Plugin** and save the `dback-sync-plugin.zip`.
4.  **Install** this plugin on your WordPress site (Plugins > Add New > Upload).
5.  Copy your WordPress **URL** into the app.
6.  The **API Key** is automatically filled (it matches the key embedded in the plugin).
7.  Click **Test Connectivity** or **Start Backup**.

## FAQ

### Why does this app require X11/GL libraries?
This application is built using **Fyne**, a high-performance GUI toolkit for Go. Fyne uses **OpenGL** to render its graphics (GPU acceleration). On Linux, interfacing with OpenGL and creating windows requires the **X11** and **OpenGL** development headers.
