#!/bin/bash
set -e

# Clean previous builds
rm -rf delivery
mkdir -p delivery

# Build for Linux
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags="-linkmode external -extldflags '-static'" -o go-fileserver main.go
zip -r delivery/go-fileserver-linux-amd64.zip go-fileserver static

# Build for Windows
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -o go-fileserver.exe main.go
zip -r delivery/go-fileserver-windows-amd64.zip go-fileserver.exe static

# Clean up binaries
rm -f go-fileserver go-fileserver.exe

echo "Build complete. Artifacts in delivery/ folder:"
ls -lh delivery/
