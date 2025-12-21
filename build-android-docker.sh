#!/bin/bash
# Build Android APK using Docker (if Android SDK setup fails)

set -e

echo "🐳 Building Android APK using Docker..."

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker or set up Android SDK manually."
    exit 1
fi

# Build using a Docker image with Android SDK
docker run --rm -it \
    -v "$(pwd):/workspace" \
    -w /workspace \
    -e ANDROID_HOME=/opt/android-sdk \
    -e ANDROID_NDK_HOME=/opt/android-sdk/ndk-bundle \
    -e PATH=$PATH:/opt/android-sdk/cmdline-tools/latest/bin \
    golang:1.21 \
    bash -c "
        apt-get update -qq && \
        apt-get install -y -qq wget unzip && \
        export PATH=\$PATH:\$(go env GOPATH)/bin && \
        go install fyne.io/tools/cmd/fyne@latest && \
        fyne package --target android --app-id com.dbsync.manager --name 'DB Sync Manager' --app-version 1.5 --icon desgin/app.png --release
    "

if [ $? -eq 0 ]; then
    echo "✅ APK built successfully!"
else
    echo "❌ Build failed"
    exit 1
fi

