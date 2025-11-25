#!/bin/bash

echo "Building for Linux..."
go build -o dback-linux main.go
if [ $? -eq 0 ]; then
    echo "Linux build successful: ./dback-linux"
else
    echo "Linux build failed. Ensure you have gcc and X11 headers installed."
fi

echo ""
echo "Building for Windows..."
# Check for MinGW compiler
if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
    CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o dback-windows.exe main.go
    if [ $? -eq 0 ]; then
        echo "Windows build successful: ./dback-windows.exe"
    else
        echo "Windows build failed."
    fi
else
    echo "MinGW compiler (x86_64-w64-mingw32-gcc) not found."
    echo "Skipping Windows build."
    echo "To build for Windows on Linux, install 'mingw-w64':"
    echo "  sudo apt-get install mingw-w64"
fi

echo ""
echo "Building for macOS..."
# Check for cross-compiler (Zig is best for macOS CGO cross-compilation)
if command -v zig &> /dev/null; then
    echo "Using Zig for macOS cross-compilation..."
    CC="zig cc -target x86_64-macos" CXX="zig c++ -target x86_64-macos" CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o dback-macos main.go
    if [ $? -eq 0 ]; then
        echo "macOS build successful: ./dback-macos"
    else
        echo "macOS build failed."
    fi
else
    echo "Cross-compiler for macOS not found (Zig recommended)."
    echo "To build for macOS on Linux, install 'zig' or 'osxcross'."
    echo "Example with Zig: snap install zig --classic"
fi
