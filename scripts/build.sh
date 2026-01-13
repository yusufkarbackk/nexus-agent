#!/bin/bash
# ============================================
# Nexus Agent - Multi-Platform Build Script
# ============================================
# Builds nexus-agent for Windows, Linux, and macOS

set -e

VERSION=${1:-"v1.0.0"}
OUTPUT_DIR="dist"

echo "🔨 Building Nexus Agent ${VERSION}..."

# Clean and create output directory
rm -rf $OUTPUT_DIR
mkdir -p $OUTPUT_DIR

# Build for each platform
platforms=(
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "darwin/amd64"
    "darwin/arm64"
)

for platform in "${platforms[@]}"; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    
    output_name="nexus-agent-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    echo "  📦 Building ${output_name}..."
    
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o "${OUTPUT_DIR}/${output_name}" \
        ./cmd/agent
done

# Create checksums
echo "🔐 Generating checksums..."
cd $OUTPUT_DIR
sha256sum * > checksums.txt
cd ..

echo ""
echo "✅ Build complete! Binaries in ${OUTPUT_DIR}/"
ls -la $OUTPUT_DIR
