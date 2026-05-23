#!/usr/bin/env bash

set -u

APP_NAME="dback"
DIST_DIR="dist"
ICON_PATH="logo.png"
APP_VERSION="${APP_VERSION:-3.2.0}"

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

build_linux() {
    echo ""
    echo "Building for Linux..."
    ensure_dist_dir
    ensure_icon || return 1
    ensure_linux_deps || return 1

    ldflags="-X main.appVersion=${APP_VERSION}"
    if CGO_ENABLED=1 go build -ldflags "$ldflags" -o "$DIST_DIR/${APP_NAME}-linux" .; then
        print_success "Linux build: ./$DIST_DIR/${APP_NAME}-linux"

        # Create a launcher wrapper that handles EGL/GPU driver issues automatically.
        local launcher="$DIST_DIR/${APP_NAME}"
        cat > "$launcher" << 'LAUNCHER_EOF'
#!/usr/bin/env bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="$SCRIPT_DIR/dback-linux"

# Detect NVIDIA driver/library version mismatch (common after updates without reboot).
# If detected, force Mesa EGL to avoid eglInitialize failures.
if [ -z "${__EGL_VENDOR_LIBRARY_FILENAMES}" ]; then
    if [ -f "/usr/share/glvnd/egl_vendor.d/10_nvidia.json" ]; then
        if ! nvidia-smi >/dev/null 2>&1; then
            MESA_VENDOR="/usr/share/glvnd/egl_vendor.d/50_mesa.json"
            if [ -f "$MESA_VENDOR" ]; then
                echo "WARNING: NVIDIA driver mismatch — using Mesa EGL. Reboot for full GPU acceleration." >&2
                export __EGL_VENDOR_LIBRARY_FILENAMES="$MESA_VENDOR"
            fi
        fi
    fi
fi

exec "$BINARY" "$@"
LAUNCHER_EOF
        chmod +x "$launcher"
        print_success "Launcher script:  ./$DIST_DIR/${APP_NAME}"
        echo "  Run the app with: ./$DIST_DIR/${APP_NAME}"
        return 0
    fi

    print_error "Linux build failed."
    return 1
}

run_choice() {
    case "$1" in
        1|linux|Linux|"")
            build_linux
            ;;
        q|Q|quit|exit)
            echo "Canceled."
            return 0
            ;;
        *)
            print_error "Unknown option: $1 (Linux-only build)"
            return 1
            ;;
    esac
}

if [ $# -gt 0 ]; then
    run_choice "$1"
    exit $?
fi

echo "DBack Linux build"
echo ""
run_choice linux
