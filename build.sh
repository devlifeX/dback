#!/usr/bin/env bash

set -u

APP_NAME="dback"
DIST_DIR="dist"
ICON_PATH="logo.png"
APP_VERSION="${APP_VERSION:-3.8.3}"
NFPM_VERSION="${NFPM_VERSION:-v2.44.1}"

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

ensure_nfpm() {
    if command -v nfpm >/dev/null 2>&1; then
        return 0
    fi

    local nfpm_version="$NFPM_VERSION"
    local nfpm_arch
    case "$(uname -m)" in
        x86_64|amd64) nfpm_arch="x86_64" ;;
        aarch64|arm64) nfpm_arch="arm64" ;;
        *)
            print_error "Unsupported architecture for nfpm bootstrap: $(uname -m)"
            return 1
            ;;
    esac

    local nfpm_url="https://github.com/goreleaser/nfpm/releases/download/${nfpm_version}/nfpm_${nfpm_version#v}_Linux_${nfpm_arch}.tar.gz"
    local tmp_dir
    tmp_dir="$(mktemp -d)"

    echo "Installing nfpm ${nfpm_version} (${nfpm_arch})..."
    if ! curl -fsSL "$nfpm_url" | tar -xz -C "$tmp_dir" nfpm; then
        print_error "Failed to download nfpm from ${nfpm_url}"
        rm -rf "$tmp_dir"
        return 1
    fi

    mkdir -p "${HOME}/.local/bin"
    mv "$tmp_dir/nfpm" "${HOME}/.local/bin/nfpm"
    chmod +x "${HOME}/.local/bin/nfpm"
    export PATH="${HOME}/.local/bin:${PATH}"
    rm -rf "$tmp_dir"

    if ! command -v nfpm >/dev/null 2>&1; then
        print_error "nfpm install failed."
        return 1
    fi
}

prepare_packaging_icons() {
    ensure_icon || return 1
    mkdir -p packaging/icons/hicolor/256x256/apps packaging/icons/hicolor/48x48/apps
    cp "$ICON_PATH" packaging/icons/hicolor/256x256/apps/dback.png
    cp "$ICON_PATH" packaging/icons/hicolor/48x48/apps/dback.png
}

pack_deb() {
    if [ ! -f "$DIST_DIR/${APP_NAME}-linux" ]; then
        print_error "Missing Linux binary; build it first."
        return 1
    fi

    prepare_packaging_icons || return 1
    ensure_nfpm || return 1

    echo ""
    echo "Building Debian package..."
    rm -f "$DIST_DIR"/dback_*_amd64.deb
    export APP_VERSION
    if APP_VERSION="$APP_VERSION" nfpm package -f packaging/nfpm.yaml -p deb --target "$DIST_DIR"; then
        local deb_file="$DIST_DIR/dback_${APP_VERSION}_amd64.deb"
        if [ -f "$deb_file" ]; then
            print_success "Debian package: ./$deb_file"
            return 0
        fi
        local found
        found="$(find "$DIST_DIR" -maxdepth 1 -name 'dback_*_amd64.deb' | head -n 1)"
        if [ -n "$found" ]; then
            print_success "Debian package: ./$found"
            return 0
        fi
    fi

    print_error "Debian package build failed."
    return 1
}

print_release_hint() {
    echo ""
    echo "Release tag for GitHub (version ${APP_VERSION}):"
    echo "  git tag v${APP_VERSION}"
    echo "  git push origin v${APP_VERSION}"
    echo ""
    echo "CI will publish: dback-linux, dback-windows.exe, dback_${APP_VERSION}_amd64.deb"
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
        pack_deb || return 1
        print_release_hint
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
