#!/bin/bash
set -e

if [ -d delivery ]; then
  rm -rf delivery
fi
# Build Go binary
GOOS=linux GOARCH=amd64 go build -ldflags="-linkmode external -extldflags '-static'" -o go-fileserver main.go 

mkdir -p delivery
# Create zip

zip ./delivery/go-fileserver.zip ../go-fileserver/go-fileserver ../go-fileserver/static/index.html

echo "Copied go-fileserver.zip to delivery/ folder."
