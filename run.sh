#!/bin/bash
echo "Checking system dependencies..."

missing=()
command -v gcc >/dev/null 2>&1 || missing+=(build-essential)
command -v pkg-config >/dev/null 2>&1 || missing+=(pkg-config)
[ -f "/usr/include/vulkan/vulkan.h" ] || missing+=(libvulkan-dev)
[ -f "/usr/include/X11/Xlib.h" ] || missing+=(xorg-dev)

gio_pkg_modules=(
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
	# Deduplicate package names while preserving order.
	unique_missing=()
	for pkg in "${missing[@]}"; do
		seen=false
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

	echo "ERROR: Missing development libraries: ${unique_missing[*]}"
	echo "Please run:"
	echo ""
	echo "    sudo apt-get update && sudo apt-get install -y build-essential pkg-config libvulkan-dev xorg-dev libwayland-dev libxkbcommon-dev libxkbcommon-x11-dev libx11-xcb-dev libxcursor-dev libxfixes-dev libegl-dev"
	echo ""
	exit 1
fi

echo "Setting up environment..."
export CGO_ENABLED=1

echo "Tidying modules..."
go mod tidy

ARGS=()
if [[ "${DBACK_DEBUG}" == "1" || "${DBACK_DEBUG}" == "true" ]]; then
    ARGS+=(--debug)
fi

echo "Running DBack..."
go run . "${ARGS[@]}" "$@"
