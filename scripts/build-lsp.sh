#!/bin/bash
set -e

# Build script for unqueryvet-lsp
# Builds binaries for all supported platforms

VERSION=${1:-dev}
OUTPUT_DIR=${2:-dist}

echo "Building unqueryvet-lsp version: $VERSION"
echo "Output directory: $OUTPUT_DIR"

# Clean and create output directory
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Platforms to build for
PLATFORMS=(
    "windows/amd64"
    "windows/arm64"
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

# Build for each platform
for PLATFORM in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a PARTS <<< "$PLATFORM"
    GOOS="${PARTS[0]}"
    GOARCH="${PARTS[1]}"

    OUTPUT_NAME="unqueryvet-lsp-${GOOS}-${GOARCH}"

    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    OUTPUT_PATH="$OUTPUT_DIR/$OUTPUT_NAME"

    echo "Building for $GOOS/$GOARCH..."

    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-s -w -X main.version=$VERSION" \
        -o "$OUTPUT_PATH" \
        ./cmd/unqueryvet-lsp

    if [ $? -eq 0 ]; then
        SIZE=$(ls -lh "$OUTPUT_PATH" | awk '{print $5}')
        echo "  ✓ Built: $OUTPUT_NAME ($SIZE)"
    else
        echo "  ✗ Failed to build for $GOOS/$GOARCH"
        exit 1
    fi
done

echo ""
echo "Build complete! Binaries in: $OUTPUT_DIR"
echo ""
ls -lh "$OUTPUT_DIR"
