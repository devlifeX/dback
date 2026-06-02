#!/usr/bin/env bash
BINARY="/usr/lib/dback/dback"

# Detect NVIDIA driver/library version mismatch (common after updates without reboot).
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
