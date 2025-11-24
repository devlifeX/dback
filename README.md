# DB Sync Manager

A cross-platform desktop GUI application built with Go and Fyne v2 for managing database synchronizations. It allows you to export large databases from remote Linux servers via SSH and restore them to local or remote servers, supporting both native MySQL/MariaDB and Dockerized instances.

## Features

- **Export (Backup):** Connect to remote servers via SSH, dump large databases (5GB+ supported via streaming), compress (gzip), and download locally.
- **Import (Restore):** Upload local SQL dumps (gzip supported) to remote servers or restore locally, handling large files efficiently.
- **Docker Support:** Seamlessly handles databases running inside Docker containers vs native OS processes.
- **Profile Management:** Save and load connection profiles for quick access.
- **Activity Logs:** Track all operations with timestamps and status.
- **Cross-Platform:** Runs on Windows, macOS, and Linux.

## Requirements

- Go 1.21 or later
- C compiler (gcc) for Fyne (requires CGO)
- On Linux: `libgl1-mesa-dev` and `xorg-dev` packages (for Fyne)

## Installation

1. Clone the repository.
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Run the application:
   ```bash
   go run main.go
   ```

## Usage

### Export Tab
1. Enter SSH Connection details (Host, Port, User, Password/Key).
2. Configure Source Database details.
   - Check "Is Docker Container?" if applicable and provide Container Name/ID.
3. Select a Destination Folder.
4. Click "Start Backup & Download".

### Import Tab
1. Select a local `.sql.gz` (or `.sql`) file.
2. Configure Destination Server.
   - Uncheck "Restore to Localhost?" to stream to a remote server.
3. Configure Destination Database details.
4. Click "Start Upload & Restore".

### Activity Logs
View a history of all operations.

## FAQ

### Why does this app require X11/GL libraries?
This application is built using **Fyne**, a high-performance GUI toolkit for Go. Fyne uses **OpenGL** to render its graphics (GPU acceleration).
On Linux, interfacing with OpenGL and creating windows requires the **X11** and **OpenGL** development headers (C libraries).
These are standard requirements for building almost any native GUI application on Linux from source.
