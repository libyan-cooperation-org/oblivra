#!/bin/bash
set -e

# Target version (default to dev)
VERSION=${1:-"dev"}
OUTPUT_DIR="builds"
APP_NAME="oblivrashell"

mkdir -p "$OUTPUT_DIR"

echo "Building version: $VERSION"

build() {
    local GOOS=$1
    local GOARCH=$2
    local EXT=$3

    echo "Building for $GOOS/$GOARCH..."
    
    local BIN_NAME="$APP_NAME"
    if [ "$GOOS" = "windows" ]; then
        BIN_NAME="${APP_NAME}.exe"
    fi

    # Build the binary
    # We use Wails CLI to build properly for the target OS
    # wails build -platform $GOOS/$GOARCH -o $BIN_NAME
    # For now (as wails build can be complex in generic scripts), we just cross-compile backend if possible
    # In a real scenario, this would ideally run `wails build` for all targets on a MacOS/Linux hybrid CI runner
    
    # Simulating standard go build for backend testing
    # go build -ldflags "-X main.Version=$VERSION" -o "$OUTPUT_DIR/$BIN_NAME" ./main.go

    # Using Wails to build. Note: Windows -> Windows is fine. Cross compiling CGO (needed for Wails) is tricky.
    wails build -platform "$GOOS/$GOARCH" -v 2 -s -o "$BIN_NAME"

    # Package
    local PKG_NAME="${APP_NAME}_${VERSION}_${GOOS}_${GOARCH}"
    cd "build/bin" # default Wails output dir
    
    if [ "$GOOS" = "windows" ]; then
        zip -q "../$OUTPUT_DIR/${PKG_NAME}.zip" "$BIN_NAME"
        echo "Created ${PKG_NAME}.zip"
    else
        tar -czf "../$OUTPUT_DIR/${PKG_NAME}.tar.gz" "$BIN_NAME"
        echo "Created ${PKG_NAME}.tar.gz"
    fi

    # SHA256 Sum
    cd "../$OUTPUT_DIR"
    if [ "$GOOS" = "windows" ]; then
        sha256sum "${PKG_NAME}.zip" >> "checksums.txt"
    else
        sha256sum "${PKG_NAME}.tar.gz" >> "checksums.txt"
    fi
    cd - > /dev/null
}

# In a realistic environment, you might only build windows on windows, linux on linux
# build "windows" "amd64" ".exe"
# build "linux" "amd64" ""
# build "linux" "arm64" ""
# build "darwin" "arm64" ""
# build "darwin" "amd64" ""

echo "Build script template created. Review Wails cross-compilation documentation for full CGO cross-compile."
