# Android SDK Setup Instructions

## Issue
Google's Android SDK/NDK download URLs require authentication or have changed. Here are alternative methods:

## Method 1: Manual Download (Recommended)

1. **Download Android Studio:**
   ```bash
   # Visit: https://developer.android.com/studio
   # Or download directly:
   wget https://redirector.gvt1.com/edgedl/android/studio/ide-zips/2023.3.1.18/android-studio-2023.3.1.18-linux.tar.gz
   ```

2. **Extract and set up:**
   ```bash
   tar -xzf android-studio-*.tar.gz
   cd android-studio/bin
   ./studio.sh
   ```
   
   In Android Studio:
   - Go to Tools → SDK Manager
   - Install Android SDK Platform-Tools
   - Install NDK (Side by side) - version 25.1.8937393 or newer
   - Note the SDK location (usually `~/Android/Sdk`)

3. **Set environment variables:**
   ```bash
   export ANDROID_HOME=$HOME/Android/Sdk
   export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/25.1.8937393
   # Or if using ndk-bundle:
   export ANDROID_NDK_HOME=$ANDROID_HOME/ndk-bundle
   export PATH=$PATH:$ANDROID_HOME/cmdline-tools/latest/bin
   ```

## Method 2: Using Package Manager (Older versions)

```bash
sudo apt-get update
sudo apt-get install -y google-android-ndk-r21c-installer
# Then set:
export ANDROID_NDK_HOME=/usr/lib/android-ndk-r21c
```

## Method 3: Docker (Alternative)

Use a Docker container with pre-installed Android SDK:
```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  -w /workspace \
  -e ANDROID_HOME=/opt/android-sdk \
  android/android-build:latest \
  fyne package --target android ...
```

## Quick Test

After setup, test with:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
fyne package --target android --app-id com.dbsync.manager --name "Test" --icon desgin/app.png
```

