#!/bin/bash
# Check Android SDK/NDK setup and provide helpful guidance

echo "🔍 Checking Android SDK/NDK setup..."

# Check Fyne
if ! command -v fyne &> /dev/null; then
    echo "❌ Fyne CLI not found"
    echo "   Install: go install fyne.io/tools/cmd/fyne@latest"
    exit 1
else
    echo "✅ Fyne CLI found: $(fyne --version)"
fi

# Check ANDROID_HOME
if [ -z "$ANDROID_HOME" ]; then
    echo "⚠️  ANDROID_HOME not set"
    echo "   Common locations:"
    echo "   - ~/Android/Sdk (Android Studio default)"
    echo "   - ~/android-sdk"
    echo ""
    echo "   Set it: export ANDROID_HOME=\$HOME/Android/Sdk"
else
    echo "✅ ANDROID_HOME: $ANDROID_HOME"
    if [ ! -d "$ANDROID_HOME" ]; then
        echo "   ⚠️  Directory doesn't exist!"
    fi
fi

# Check NDK
NDK_FOUND=false
if [ -n "$ANDROID_NDK_HOME" ] && [ -d "$ANDROID_NDK_HOME" ]; then
    echo "✅ ANDROID_NDK_HOME: $ANDROID_NDK_HOME"
    if [ -f "$ANDROID_NDK_HOME/toolchains/llvm/prebuilt/linux-x86_64/bin/clang" ]; then
        echo "✅ NDK compiler found"
        NDK_FOUND=true
    else
        echo "   ⚠️  NDK compiler not found (incomplete installation)"
    fi
elif [ -n "$ANDROID_HOME" ] && [ -d "$ANDROID_HOME/ndk-bundle" ]; then
    echo "✅ Found NDK at: $ANDROID_HOME/ndk-bundle"
    if [ -f "$ANDROID_HOME/ndk-bundle/toolchains/llvm/prebuilt/linux-x86_64/bin/clang" ]; then
        echo "✅ NDK compiler found"
        NDK_FOUND=true
    fi
elif [ -n "$ANDROID_HOME" ]; then
    # Check for side-by-side NDK
    NDK_DIRS=$(find "$ANDROID_HOME/ndk" -maxdepth 1 -type d 2>/dev/null | grep -E "[0-9]+\.[0-9]+" | head -1)
    if [ -n "$NDK_DIRS" ]; then
        echo "✅ Found NDK at: $NDK_DIRS"
        if [ -f "$NDK_DIRS/toolchains/llvm/prebuilt/linux-x86_64/bin/clang" ]; then
            echo "✅ NDK compiler found"
            export ANDROID_NDK_HOME="$NDK_DIRS"
            NDK_FOUND=true
        fi
    fi
fi

if [ "$NDK_FOUND" = false ]; then
    echo ""
    echo "❌ Android NDK not found or incomplete"
    echo ""
    echo "📥 To install NDK:"
    echo ""
    echo "Option 1: Install Android Studio"
    echo "   1. Download: https://developer.android.com/studio"
    echo "   2. Install Android Studio"
    echo "   3. Open Android Studio → Tools → SDK Manager"
    echo "   4. Install 'NDK (Side by side)' - version 25.1.8937393 or newer"
    echo "   5. Set: export ANDROID_HOME=\$HOME/Android/Sdk"
    echo ""
    echo "Option 2: Manual download"
    echo "   1. Visit: https://developer.android.com/ndk/downloads"
    echo "   2. Download NDK r25c or newer for Linux"
    echo "   3. Extract to: ~/android-sdk/ndk-bundle"
    echo "   4. Set: export ANDROID_NDK_HOME=\$HOME/android-sdk/ndk-bundle"
    echo ""
    echo "Option 3: Use package manager (older versions)"
    echo "   sudo apt-get install google-android-ndk-r21c-installer"
    echo "   export ANDROID_NDK_HOME=/usr/lib/android-ndk-r21c"
    exit 1
fi

echo ""
echo "✅ Android SDK/NDK setup looks good!"
echo ""
echo "🚀 You can now build APK with:"
echo "   ./build-android.sh"

