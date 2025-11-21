#!/bin/bash
# Build script for mcp-serve
# Builds binaries for multiple platforms

set -e

VERSION="${VERSION:-1.0.0}"
BUILD_DIR="dist"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Building mcp-serve v${VERSION}${NC}"
echo "================================"
echo ""

# Clean previous builds
echo -e "${YELLOW}Cleaning previous builds...${NC}"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Build flags
LDFLAGS="-s -w -X main.VERSION=${VERSION}"

# Build for different platforms
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"

    output_name="mcp-serve-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo -e "${YELLOW}Building for ${GOOS}/${GOARCH}...${NC}"

    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="$LDFLAGS" \
        -o "${BUILD_DIR}/${output_name}" \
        .

    echo -e "${GREEN}✓ Built: ${output_name}${NC}"
done

echo ""
echo -e "${GREEN}Build complete!${NC}"
echo ""
echo "Binaries in ${BUILD_DIR}:"
ls -lh "$BUILD_DIR"

echo ""
echo "To create a release, run:"
echo "  gh release create v${VERSION} ${BUILD_DIR}/* --title \"v${VERSION}\" --notes \"Release v${VERSION}\""
