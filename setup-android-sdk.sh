#!/bin/bash
# Setup Android SDK and NDK for Fyne

set -e

ANDROID_SDK_DIR="$HOME/android-sdk"
export ANDROID_HOME="$ANDROID_SDK_DIR"
export ANDROID_NDK_HOME="$ANDROID_SDK_DIR/ndk-bundle"

echo "📦 Setting up Android SDK..."

mkdir -p "$ANDROID_SDK_DIR"

# Try to download NDK directly (Fyne mainly needs NDK)
echo "📥 Downloading Android NDK..."
cd "$ANDROID_SDK_DIR"

# Try multiple NDK versions
NDK_VERSIONS=("25.1.8937393" "25.2.9519653" "26.1.10909125")

for NDK_VER in "${NDK_VERSIONS[@]}"; do
    echo "Trying NDK version $NDK_VER..."
    NDK_URL="https://dl.google.com/android/repository/android-ndk-r${NDK_VER//./}-linux.zip"
    
    if curl -L -f -o "ndk.zip" "$NDK_URL" 2>/dev/null; then
        echo "✅ Downloaded NDK $NDK_VER"
        unzip -q ndk.zip
        mv android-ndk-r* ndk-bundle || mv android-ndk-* ndk-bundle
        rm -f ndk.zip
        echo "✅ NDK installed to $ANDROID_NDK_HOME"
        break
    else
        echo "❌ Failed to download NDK $NDK_VER"
        rm -f ndk.zip
    fi
done

# If NDK download failed, try alternative: download minimal SDK structure
if [ ! -d "$ANDROID_NDK_HOME" ]; then
    echo "⚠️  Direct NDK download failed. Trying alternative method..."
    
    # Create minimal structure and download NDK via sdkmanager if possible
    mkdir -p "$ANDROID_SDK_DIR/cmdline-tools/latest/bin"
    
    # Try to get sdkmanager from a working source
    echo "Attempting to get command line tools..."
    
    # Alternative: Use a known working URL format
    CMD_TOOLS_URL="https://dl.google.com/android/repository/commandlinetools-linux-9477386_latest.zip"
    
    if curl -L -f -o cmdline-tools.zip "$CMD_TOOLS_URL" 2>/dev/null && [ -f cmdline-tools.zip ] && [ $(stat -f%z cmdline-tools.zip 2>/dev/null || stat -c%s cmdline-tools.zip) -gt 1000000 ]; then
        echo "✅ Downloaded command line tools"
        unzip -q cmdline-tools.zip
        mv cmdline-tools/* cmdline-tools/latest/ 2>/dev/null || true
        rm -f cmdline-tools.zip
        
        # Install NDK via sdkmanager
        export PATH="$PATH:$ANDROID_SDK_DIR/cmdline-tools/latest/bin"
        yes | sdkmanager --licenses > /dev/null 2>&1 || true
        sdkmanager "ndk;25.1.8937393" || sdkmanager "ndk-bundle" || true
    fi
fi

# Final check
if [ -d "$ANDROID_NDK_HOME" ]; then
    echo "✅ Android NDK is ready at $ANDROID_NDK_HOME"
    echo ""
    echo "Set these environment variables:"
    echo "export ANDROID_HOME=$ANDROID_HOME"
    echo "export ANDROID_NDK_HOME=$ANDROID_NDK_HOME"
    echo "export PATH=\$PATH:\$ANDROID_HOME/cmdline-tools/latest/bin"
else
    echo "❌ Failed to install Android NDK"
    echo "Please install Android Studio or manually download NDK from:"
    echo "https://developer.android.com/ndk/downloads"
    exit 1
fi

