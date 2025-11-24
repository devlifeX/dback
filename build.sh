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
