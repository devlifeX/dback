#!/bin/bash

# Build Android APK for DB Sync Manager
# Requirements:
# 1. Install fyne: go install fyne.io/fyne/v2/cmd/fyne@latest
# 2. Install Android SDK and NDK
# 3. Set ANDROID_HOME environment variable

set -e

echo "🔨 Building Android APK..."

# Check if fyne is installed
if ! command -v fyne &> /dev/null; then
    echo "❌ Error: fyne command not found"
    echo "Install it with: go install fyne.io/fyne/v2/cmd/fyne@latest"
    exit 1
fi

# Check Android SDK
if [ -z "$ANDROID_HOME" ]; then
    echo "⚠️  Warning: ANDROID_HOME not set"
    echo "Set it to your Android SDK path, e.g.:"
    echo "export ANDROID_HOME=$HOME/Android/Sdk"
    exit 1
fi

# App metadata
APP_NAME="DB Sync Manager"
APP_ID="com.dbsync.manager"
APP_VERSION="1.5"
APP_ICON="desgin/app.png"

# Check if icon exists
if [ ! -f "$APP_ICON" ]; then
    echo "⚠️  Warning: Icon not found at $APP_ICON"
    echo "Creating a placeholder icon..."
    # Create a simple icon using fyne
    fyne bundle -o icon.png desgin/app.png 2>/dev/null || echo "Icon will use default"
    APP_ICON="icon.png"
fi

# Set environment variables
export ANDROID_HOME=${ANDROID_HOME:-$HOME/android-sdk}
export ANDROID_NDK_HOME=${ANDROID_NDK_HOME:-$ANDROID_HOME/ndk-bundle}
export PATH=$PATH:$(go env GOPATH)/bin

# Build APK
echo "📦 Packaging APK..."
fyne package \
    --target android \
    --app-id "$APP_ID" \
    --name "$APP_NAME" \
    --app-version "$APP_VERSION" \
    --icon "$APP_ICON" \
    --release

if [ $? -eq 0 ]; then
    echo "✅ APK built successfully!"
    echo "📱 Output: DB Sync Manager.apk"
    echo ""
    echo "To install on device:"
    echo "  adb install 'DB Sync Manager.apk'"
else
    echo "❌ Build failed"
    exit 1
fi

