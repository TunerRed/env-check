#!/usr/bin/env bash
set -euo pipefail

OUTDIR=dist
mkdir -p "$OUTDIR"

echo "Building linux/amd64..."
env GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o "$OUTDIR/env-check-linux-amd64" .
chmod +x "$OUTDIR/env-check-linux-amd64"

echo "Building linux/arm64..."
env GOOS=linux GOARCH=arm64 go build -ldflags='-s -w' -o "$OUTDIR/env-check-linux-arm64" .
chmod +x "$OUTDIR/env-check-linux-arm64"

#echo "Packing tar.gz files..."
#pushd "$OUTDIR" >/dev/null
#tar -czf env-check-linux-amd64.tar.gz env-check-linux-amd64
#tar -czf env-check-linux-arm64.tar.gz env-check-linux-arm64
#popd >/dev/null

echo "Build and packaging complete. Files in $OUTDIR"
