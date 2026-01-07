#!/bin/sh
set -e

# SuperRalph installer
# Usage: curl -fsSL https://raw.githubusercontent.com/MPJHorner/SuperRalph/main/install.sh | sh

REPO="MPJHorner/SuperRalph"
BINARY_NAME="superralph"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    printf "${GREEN}==>${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}Warning:${NC} %s\n" "$1"
}

error() {
    printf "${RED}Error:${NC} %s\n" "$1"
    exit 1
}

# Detect OS
detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Darwin) OS="darwin" ;;
        Linux) OS="linux" ;;
        *) error "Unsupported operating system: $OS" ;;
    esac
}

# Detect architecture
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
}

# Get latest release version
get_latest_version() {
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        error "Could not determine latest version. Please check https://github.com/${REPO}/releases"
    fi
}

# Download and install
install() {
    detect_os
    detect_arch
    get_latest_version

    ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"

    info "Detected: ${OS}/${ARCH}"
    info "Latest version: ${VERSION}"
    info "Downloading ${ASSET_NAME}..."

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Download binary
    if ! curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${BINARY_NAME}"; then
        error "Failed to download ${DOWNLOAD_URL}"
    fi

    # Make executable
    chmod +x "${TMP_DIR}/${BINARY_NAME}"

    # Install
    info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
    
    if [ -w "$INSTALL_DIR" ]; then
        mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        warn "Need sudo to install to ${INSTALL_DIR}"
        sudo mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    # Verify installation
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        info "Successfully installed ${BINARY_NAME} ${VERSION}"
        info "Run 'superralph --help' to get started"
    else
        warn "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
        warn "Make sure ${INSTALL_DIR} is in your PATH"
    fi
}

# Allow custom install directory
if [ -n "$1" ]; then
    INSTALL_DIR="$1"
fi

install
