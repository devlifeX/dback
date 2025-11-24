#!/bin/bash
echo "Checking system dependencies..."

# Check for X11 headers
if [ ! -f "/usr/include/X11/Xlib.h" ]; then
    echo "ERROR: X11 headers not found!"
    echo "The application requires X11 development libraries."
    echo "Please run the following command to install them:"
    echo ""
    echo "    sudo apt-get update && sudo apt-get install -y libgl1-mesa-dev xorg-dev"
    echo ""
    exit 1
fi

echo "Setting up environment..."
export CGO_ENABLED=1

echo "Tidying modules..."
go mod tidy

echo "Running DB Sync Manager..."
go run main.go
