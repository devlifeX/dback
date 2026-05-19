#!/usr/bin/env bash

set -u

APP_NAME="dback"
APP_ID="com.dbsync.manager"
DIST_DIR="dist"
ICON_PATH="logo.png"
APP_VERSION="${APP_VERSION:-1.0.0}"
ANDROID_API_LEVEL="35"
ANDROID_BUILD_TOOLS="35.0.0"
ANDROID_NDK_VERSION="26.3.11579264"
ANDROID_SDK_DIR="${ANDROID_SDK_ROOT:-${ANDROID_HOME:-$HOME/.android-sdk}}"

export PATH="$PATH:$(go env GOPATH 2>/dev/null)/bin:$ANDROID_SDK_DIR/cmdline-tools/latest/bin:$ANDROID_SDK_DIR/platform-tools"

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
    [ -f "/usr/include/X11/Xlib.h" ] || missing+=(xorg-dev)
    [ -f "/usr/include/GL/gl.h" ] || missing+=(libgl1-mesa-dev)

    if [ ${#missing[@]} -gt 0 ]; then
        install_apt_packages "${missing[@]}" || return 1
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

ensure_android_sdk() {
    install_apt_packages openjdk-17-jdk unzip wget ca-certificates || return 1
    ensure_go_tool fyne fyne.io/tools/cmd/fyne@latest || return 1

    mkdir -p "$ANDROID_SDK_DIR/cmdline-tools"

    if [ ! -x "$ANDROID_SDK_DIR/cmdline-tools/latest/bin/sdkmanager" ]; then
        echo "Installing Android command-line tools into $ANDROID_SDK_DIR..."
        local tmp_dir
        tmp_dir="$(mktemp -d)"
        local zip_file="$tmp_dir/commandlinetools.zip"

        wget -q -O "$zip_file" "https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip" || {
            rm -rf "$tmp_dir"
            print_error "Failed to download Android command-line tools."
            return 1
        }

        unzip -q "$zip_file" -d "$tmp_dir"
        rm -rf "$ANDROID_SDK_DIR/cmdline-tools/latest"
        mkdir -p "$ANDROID_SDK_DIR/cmdline-tools/latest"
        mv "$tmp_dir/cmdline-tools/"* "$ANDROID_SDK_DIR/cmdline-tools/latest/"
        rm -rf "$tmp_dir"
    fi

    export ANDROID_HOME="$ANDROID_SDK_DIR"
    export ANDROID_SDK_ROOT="$ANDROID_SDK_DIR"
    export PATH="$PATH:$ANDROID_SDK_DIR/cmdline-tools/latest/bin:$ANDROID_SDK_DIR/platform-tools"

    echo "Accepting Android SDK licenses..."
    yes | sdkmanager --licenses >/dev/null || true

    echo "Installing Android SDK packages..."
    sdkmanager \
        "platform-tools" \
        "platforms;android-$ANDROID_API_LEVEL" \
        "build-tools;$ANDROID_BUILD_TOOLS" \
        "ndk;$ANDROID_NDK_VERSION" || return 1

    export ANDROID_NDK_HOME="$ANDROID_SDK_DIR/ndk/$ANDROID_NDK_VERSION"
    export ANDROID_NDK_ROOT="$ANDROID_NDK_HOME"
    print_success "Android SDK ready: $ANDROID_SDK_DIR"
}

build_linux() {
    echo ""
    echo "Building for Linux..."
    ensure_dist_dir
    ensure_icon || return 1
    ensure_linux_deps || return 1

    if CGO_ENABLED=1 go build -o "$DIST_DIR/${APP_NAME}-linux" main.go; then
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

    if CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o "$DIST_DIR/${APP_NAME}-windows.exe" main.go; then
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
    if CC="zig cc -target x86_64-macos" CXX="zig c++ -target x86_64-macos" CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o "$DIST_DIR/${APP_NAME}-macos" main.go; then
        print_success "macOS build: ./$DIST_DIR/${APP_NAME}-macos"
        return 0
    fi

    print_error "macOS build failed."
    return 1
}

build_android() {
    echo ""
    echo "Building for Android..."
    ensure_dist_dir
    ensure_icon || return 1
    ensure_android_sdk || return 1

    if fyne package \
        --target android \
        --app-id "$APP_ID" \
        --app-version "$APP_VERSION" \
        --name "$APP_NAME" \
        --icon "$ICON_PATH"; then
        mv "${APP_NAME}.apk" "$DIST_DIR/${APP_NAME}-android.apk" 2>/dev/null || true
        print_success "Android build: ./$DIST_DIR/${APP_NAME}-android.apk"
        return 0
    fi

    print_error "Android build failed. Check Android SDK/NDK and Fyne mobile build requirements."
    return 1
}

build_all() {
    local failed=0
    build_linux || failed=1
    build_windows || failed=1
    build_macos || failed=1
    build_android || failed=1
    return "$failed"
}

show_menu() {
    echo "Select build target:"
    echo "  1) Linux"
    echo "  2) Windows"
    echo "  3) macOS"
    echo "  4) Android"
    echo "  5) All"
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
