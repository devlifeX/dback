#!/usr/bin/env bash

set -u

APP_NAME="dback"
APP_ID="com.dbsync.manager"
DIST_DIR="dist"
ICON_PATH="logo.png"
APP_VERSION="${APP_VERSION:-1.0.0}"

export PATH="$PATH:$(go env GOPATH 2>/dev/null)/bin"

ensure_dist_dir() {
    mkdir -p "$DIST_DIR"
}

ensure_icon() {
    if [ ! -f "$ICON_PATH" ]; then
        print_error "Icon not found: $ICON_PATH"
        return 1
    fi
}

print_success() {
    echo "OK: $1"
}

print_error() {
    echo "ERROR: $1"
}

sudo_cmd() {
    if [ "$(id -u)" -eq 0 ]; then
        "$@"
    else
        sudo "$@"
    fi
}

install_apt_packages() {
    if ! command -v apt-get >/dev/null 2>&1; then
        print_error "Automatic system package installation currently supports apt-based Linux distributions."
        echo "Missing packages: $*"
        return 1
    fi

    echo "Installing system packages: $*"
    sudo_cmd apt-get update
    sudo_cmd env DEBIAN_FRONTEND=noninteractive apt-get install -y "$@"
}

ensure_go_tool() {
    if command -v "$1" >/dev/null 2>&1; then
        return 0
    fi

    echo "Installing Go tool: $2"
    go install "$2"
    export PATH="$PATH:$(go env GOPATH)/bin"
    command -v "$1" >/dev/null 2>&1
}

ensure_linux_deps() {
    local missing=()

    command -v gcc >/dev/null 2>&1 || missing+=(build-essential)
    command -v pkg-config >/dev/null 2>&1 || missing+=(pkg-config)
    [ -f "/usr/include/vulkan/vulkan.h" ] || missing+=(libvulkan-dev)
    [ -f "/usr/include/X11/Xlib.h" ] || missing+=(xorg-dev)

    local gio_pkg_modules=(
        egl wayland-egl wayland-client wayland-cursor
        x11 xkbcommon xkbcommon-x11 x11-xcb xcursor xfixes
    )
    if command -v pkg-config >/dev/null 2>&1; then
        for pkg in "${gio_pkg_modules[@]}"; do
            if ! pkg-config --exists "$pkg" 2>/dev/null; then
                case "$pkg" in
                    xkbcommon-x11) missing+=(libxkbcommon-x11-dev) ;;
                    x11-xcb) missing+=(libx11-xcb-dev) ;;
                    wayland-*) missing+=(libwayland-dev) ;;
                    xkbcommon) missing+=(libxkbcommon-dev) ;;
                    egl|wayland-egl) missing+=(libegl-dev) ;;
                    xcursor) missing+=(libxcursor-dev) ;;
                    xfixes) missing+=(libxfixes-dev) ;;
                    x11) missing+=(xorg-dev) ;;
                esac
            fi
        done
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        local unique_missing=()
        for pkg in "${missing[@]}"; do
            local seen=false
            for existing in "${unique_missing[@]}"; do
                if [ "$existing" = "$pkg" ]; then
                    seen=true
                    break
                fi
            done
            if [ "$seen" = false ]; then
                unique_missing+=("$pkg")
            fi
        done
        install_apt_packages "${unique_missing[@]}" || return 1
    fi
}

ensure_windows_deps() {
    if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
        return 0
    fi

    install_apt_packages mingw-w64
}

ensure_macos_deps() {
    if command -v zig >/dev/null 2>&1; then
        return 0
    fi

    if command -v snap >/dev/null 2>&1; then
        echo "Installing Zig with snap..."
        sudo_cmd snap install zig --classic
        return $?
    fi

    print_error "Zig is required for macOS cross-compilation and could not be installed automatically."
    echo "Install snap or Zig manually, then run this script again."
    return 1
}

build_linux() {
    echo ""
    echo "Building for Linux..."
    ensure_dist_dir
    ensure_icon || return 1
    ensure_linux_deps || return 1

    if CGO_ENABLED=1 go build -o "$DIST_DIR/${APP_NAME}-linux" .; then
        print_success "Linux build: ./$DIST_DIR/${APP_NAME}-linux"
        return 0
    fi

    print_error "Linux build failed."
    return 1
}

build_windows() {
    echo ""
    echo "Building for Windows..."
    ensure_dist_dir
    ensure_icon || return 1
    ensure_windows_deps || return 1

    if CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o "$DIST_DIR/${APP_NAME}-windows.exe" .; then
        print_success "Windows build: ./$DIST_DIR/${APP_NAME}-windows.exe"
        return 0
    fi

    print_error "Windows build failed."
    return 1
}

build_macos() {
    echo ""
    echo "Building for macOS..."
    ensure_dist_dir
    ensure_icon || return 1
    ensure_macos_deps || return 1

    echo "Using Zig for macOS cross-compilation..."
    if CC="zig cc -target x86_64-macos" CXX="zig c++ -target x86_64-macos" CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o "$DIST_DIR/${APP_NAME}-macos" .; then
        print_success "macOS build: ./$DIST_DIR/${APP_NAME}-macos"
        return 0
    fi

    print_error "macOS build failed."
    return 1
}

build_android() {
    echo ""
    echo "Android builds are not yet available with the Gio UI."
    echo "Desktop targets are supported today; Android packaging will be added in a future release."
    return 1
}

build_all() {
    local failed=0
    build_linux || failed=1
    build_windows || failed=1
    build_macos || failed=1
    return "$failed"
}

show_menu() {
    echo "Select build target:"
    echo "  1) Linux"
    echo "  2) Windows"
    echo "  3) macOS"
    echo "  4) Android (not yet supported with Gio)"
    echo "  5) All desktop targets"
    echo "  q) Quit"
}

run_choice() {
    case "$1" in
        1|linux|Linux)
            build_linux
            ;;
        2|windows|Windows|win)
            build_windows
            ;;
        3|macos|macOS|darwin|Darwin)
            build_macos
            ;;
        4|android|Android)
            build_android
            ;;
        5|all|All)
            build_all
            ;;
        q|Q|quit|exit)
            echo "Canceled."
            return 0
            ;;
        *)
            print_error "Unknown option: $1"
            return 1
            ;;
    esac
}

if [ $# -gt 0 ]; then
    run_choice "$1"
    exit $?
fi

show_menu
echo ""
read -r -p "Build target: " choice
run_choice "$choice"
