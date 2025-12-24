#!/bin/bash
set -e

# Parse arguments
STATIC=false
VERSION=""
OUTPUT="loadshow"

while [[ $# -gt 0 ]]; do
    case $1 in
        --static) STATIC=true; shift ;;
        --version) VERSION="$2"; shift 2 ;;
        --output) OUTPUT="$2"; shift 2 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "unknown" ;;
    esac
}

OS=$(detect_os)
echo "Building for OS: $OS"
echo "Static build: $STATIC"
echo "Version: ${VERSION:-dev}"
echo "Output: $OUTPUT"

# Build flags
LDFLAGS=""
if [ -n "$VERSION" ]; then
    LDFLAGS="-X main.version=${VERSION}"
fi

# Set up environment and build
build_linux() {
    if [ "$STATIC" = true ]; then
        export CC=musl-gcc
        export CGO_ENABLED=1
        export PKG_CONFIG_PATH="/usr/local/musl/lib/pkgconfig"
        export CGO_CFLAGS="-I/usr/local/musl/include"
        export CGO_LDFLAGS="-L/usr/local/musl/lib -laom -lm -static"

        go build -ldflags "${LDFLAGS} -linkmode external -extldflags '-static'" \
            -o "${OUTPUT}" ./cmd/loadshow

        echo "Verifying static binary..."
        file "${OUTPUT}"
        ldd "${OUTPUT}" 2>&1 || echo "Static binary confirmed (no dynamic dependencies)"
    else
        go build -ldflags "${LDFLAGS}" -o "${OUTPUT}" ./cmd/loadshow
    fi
}

build_darwin() {
    if [ "$STATIC" = true ]; then
        # macOS cannot be fully static, but we statically link libaom
        export PKG_CONFIG_PATH="/usr/local/lib/pkgconfig"
        export CGO_LDFLAGS="-L/usr/local/lib -laom -lm -lpthread"

        go build -ldflags "${LDFLAGS}" -o "${OUTPUT}" ./cmd/loadshow

        echo "Verifying binary (libaom should not appear in dynamic libs)..."
        otool -L "${OUTPUT}"
    else
        go build -ldflags "${LDFLAGS}" -o "${OUTPUT}" ./cmd/loadshow
    fi
}

build_windows() {
    # Add .exe extension if not present
    if [[ ! "$OUTPUT" =~ \.exe$ ]]; then
        OUTPUT="${OUTPUT}.exe"
    fi

    export CGO_ENABLED=1
    export CC=gcc

    if [ "$STATIC" = true ]; then
        export PKG_CONFIG_PATH="/ucrt64/lib/pkgconfig"
        export CGO_LDFLAGS="$(pkg-config --static --libs aom) -static"

        go build -ldflags "${LDFLAGS}" -o "${OUTPUT}" ./cmd/loadshow
    else
        export PKG_CONFIG_PATH="/ucrt64/lib/pkgconfig"
        go build -ldflags "${LDFLAGS}" -o "${OUTPUT}" ./cmd/loadshow
    fi
}

# Main
case $OS in
    linux)   build_linux ;;
    darwin)  build_darwin ;;
    windows) build_windows ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Build complete: ${OUTPUT}"
