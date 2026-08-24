#!/bin/bash
set -e

PLATFORMS=("windows/amd64" "windows/arm64" "linux/amd64" "linux/arm64" "android/arm64" "darwin/amd64" "darwin/arm64")
OUTDIR="bin"

mkdir -p "$OUTDIR"

for platform in "${PLATFORMS[@]}"; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    output="$OUTDIR/boi-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output="${output}.exe"
    fi
    echo "Building $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o "$output" ./cmd/boi
done

echo ""
echo "Build complete. Binaries:"
ls -lh "$OUTDIR"/boi-*
