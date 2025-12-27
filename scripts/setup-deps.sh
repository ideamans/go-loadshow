#!/bin/bash
set -e

# Configuration
LIBAOM_VERSION="3.8.0"
LIBAOM_TARBALL="https://storage.googleapis.com/aom-releases/libaom-${LIBAOM_VERSION}.tar.gz"

# Parse arguments
STATIC=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --static) STATIC=true; shift ;;
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
echo "Detected OS: $OS"
echo "Static build: $STATIC"

# Build libaom from source (for static linking)
build_libaom_static() {
    local install_prefix=$1
    local cc=${2:-gcc}
    local extra_cmake_args=${3:-}

    echo "Building libaom ${LIBAOM_VERSION} from source..."

    # Download and extract if not exists
    if [ ! -d "aom" ]; then
        echo "Downloading libaom ${LIBAOM_VERSION}..."
        curl -L "${LIBAOM_TARBALL}" -o libaom.tar.gz
        tar -xzf libaom.tar.gz
        mv "libaom-${LIBAOM_VERSION}" aom
        rm libaom.tar.gz
    fi

    cd aom
    # Use _build directory to avoid conflicting with source's build/cmake directory
    rm -rf _build && mkdir _build && cd _build

    CC=$cc cmake .. -G Ninja \
        -DCMAKE_BUILD_TYPE=Release \
        -DENABLE_SHARED=OFF \
        -DCONFIG_AV1_ENCODER=1 \
        -DCONFIG_AV1_DECODER=1 \
        -DENABLE_EXAMPLES=OFF \
        -DENABLE_TESTS=OFF \
        -DENABLE_TOOLS=OFF \
        -DENABLE_DOCS=OFF \
        -DCMAKE_INSTALL_PREFIX="${install_prefix}" \
        $extra_cmake_args

    ninja

    if [ "$OS" = "windows" ]; then
        ninja install
    else
        sudo ninja install
    fi

    cd ../..
    echo "libaom installed to ${install_prefix}"
}

# Linux setup
setup_linux() {
    echo "Setting up dependencies for Linux..."
    sudo apt-get update
    sudo apt-get install -y cmake ninja-build pkg-config git nasm ffmpeg

    # Always build from source for static linking
    build_libaom_static "/usr/local" "gcc"
}

# macOS setup
setup_darwin() {
    echo "Setting up dependencies for macOS..."
    brew install cmake ninja pkg-config nasm libvmaf

    # Detect architecture for correct install path
    local install_prefix="/usr/local"
    if [ "$(uname -m)" = "arm64" ]; then
        install_prefix="/opt/homebrew"
    fi

    # Always build from source to avoid VMAF dependency
    # Note: libvmaf is installed above for compatibility with Homebrew's aom
    # (in case someone later uses `brew install aom` instead of source build)
    build_libaom_static "$install_prefix"
}

# Windows (MSYS2) setup - Note: CI uses vcpkg instead
setup_windows() {
    echo "Setting up dependencies for Windows (MSYS2)..."

    if [ "$STATIC" = true ]; then
        pacman -S --noconfirm --needed \
            mingw-w64-ucrt-x86_64-cmake \
            mingw-w64-ucrt-x86_64-ninja \
            mingw-w64-ucrt-x86_64-gcc \
            mingw-w64-ucrt-x86_64-pkg-config \
            mingw-w64-ucrt-x86_64-curl
        # Use generic CPU to avoid nasm version issues on Windows
        build_libaom_static "/ucrt64" "gcc" "-DAOM_TARGET_CPU=generic"
    else
        pacman -S --noconfirm --needed \
            mingw-w64-ucrt-x86_64-gcc \
            mingw-w64-ucrt-x86_64-pkg-config \
            mingw-w64-ucrt-x86_64-aom
    fi
}

# Main
case $OS in
    linux)   setup_linux ;;
    darwin)  setup_darwin ;;
    windows) setup_windows ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Dependencies setup complete!"
