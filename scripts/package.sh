#!/bin/bash
set -e

# Parse arguments
VERSION=""
BINARY="loadshow"

while [[ $# -gt 0 ]]; do
    case $1 in
        --version) VERSION="$2"; shift 2 ;;
        --binary) BINARY="$2"; shift 2 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

if [ -z "$VERSION" ]; then
    echo "Error: --version is required"
    exit 1
fi

# Detect OS and architecture
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "unknown" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "unknown" ;;
    esac
}

OS=$(detect_os)
ARCH=$(detect_arch)

echo "Packaging for: ${OS}_${ARCH}"
echo "Version: $VERSION"

# GoReleaser compatible naming: {name}_{version}_{os}_{arch}
ARCHIVE_NAME="loadshow_${VERSION}_${OS}_${ARCH}"

# Create archive
if [ "$OS" = "windows" ]; then
    # Windows: use zip
    BINARY_FILE="${BINARY}"
    if [[ ! "$BINARY_FILE" =~ \.exe$ ]]; then
        BINARY_FILE="${BINARY}.exe"
    fi

    if [ ! -f "$BINARY_FILE" ]; then
        echo "Error: Binary not found: $BINARY_FILE"
        exit 1
    fi

    # PowerShell for zip (works in both MSYS2 and native Windows)
    powershell -Command "Compress-Archive -Path '${BINARY_FILE}' -DestinationPath '${ARCHIVE_NAME}.zip' -Force"
    echo "Created: ${ARCHIVE_NAME}.zip"
else
    # Unix: use tar.gz
    if [ ! -f "$BINARY" ]; then
        echo "Error: Binary not found: $BINARY"
        exit 1
    fi

    tar -czvf "${ARCHIVE_NAME}.tar.gz" "$BINARY"
    echo "Created: ${ARCHIVE_NAME}.tar.gz"
fi

echo "Packaging complete!"
