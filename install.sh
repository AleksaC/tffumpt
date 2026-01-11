#!/bin/sh

set -e

REPO_OWNER="AleksaC"
REPO_NAME="tffumpt"
REPO="${REPO_OWNER}/${REPO_NAME}"

INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
VERSION="${VERSION:-latest}"
DRY_RUN="${DRY_RUN:-false}"

http_curl() {
    local url="$1"
    local output="$2"
    shift 2
    curl -fsSL "$@" "$url" -o "$output"
}

http_wget() {
    local url="$1"
    local output="$2"
    shift 2
    wget -q "$@" "$url" -O "$output"
}

# http_req url output [args...], use - to output to stdout
http_req() {
    if command -v curl >/dev/null 2>&1; then
        http_curl "$@"
    elif command -v wget >/dev/null 2>&1; then
        http_wget "$@"
    else
        echo "Error: Neither curl nor wget is available. Please install one of them."
        exit 1
    fi
}

## Platform detection

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    msys*) OS="windows" ;;
    mingw*) OS="windows" ;;
    cygwin*) OS="windows" ;;
    win*) OS="windows" ;;
    linux) OS="linux" ;;
    darwin) OS="darwin" ;;
    *) echo "Unsupported operating system: $OS"; exit 1 ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

## Installation directory setup

if [ "$DRY_RUN" = "false" ]; then
    mkdir -p "${INSTALL_DIR}"
fi

INSTALL_DIR="$(cd "${INSTALL_DIR}" && pwd)"

if ! echo "$PATH" | grep -q "${INSTALL_DIR}:"; then
    echo "Warning: ${INSTALL_DIR} is not in your PATH."
    echo ""
    echo "You may need to add it to your PATH to use the tffumpt command or change the installation directory by setting the INSTALL_DIR environment variable."
    echo ""
    echo "To add it to your PATH, run one of the following commands:"
    echo ""
    echo "For bash/zsh (add to ~/.bashrc, ~/.zshrc, or ~/.profile):"
    echo "  export PATH=\"\${PATH}:${INSTALL_DIR}\""
    echo ""
    echo "For fish (add to ~/.config/fish/config.fish):"
    echo "  set -gx PATH \$PATH ${INSTALL_DIR}"
    echo ""
fi

## Version detection

if [ "$VERSION" = "latest" ]; then
    # parsing with grep and cut isn't reliable, but can't assume jq is available
    VERSION=$(
        http_req "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest" - \
        | grep -oE '"tag_name": \".*\"' \
        | cut -d':' -f 2 \
        | cut -d'"' -f 2
    )
    if [ "$VERSION" = "" ]; then
        echo "Error: Failed to fetch latest version from GitHub API"
        exit 1
    fi
fi

# Add v prefix if missing
VERSION="v${VERSION#v}"

## Archive download and extraction

ARCHIVE_NAME="tffumpt-${VERSION}-${OS}-${ARCH}.zip"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"

echo "Downloading from: ${DOWNLOAD_URL}"
if [ "$DRY_RUN" = "false" ]; then
    tmpdir=$(dirname "$(mktemp -ut tmp.XXXXXXXXXX)")
    cd "$tmpdir"
    http_req "${DOWNLOAD_URL}" "${ARCHIVE_NAME}"
fi

echo "Installing to: ${INSTALL_DIR}"
if [ "$DRY_RUN" = "false" ]; then
    unzip -oqj "${ARCHIVE_NAME}" -x "LICENSE*" "README*"
    mv tffumpt "${INSTALL_DIR}/"
fi

echo "Installation complete! The binary has been installed to: ${INSTALL_DIR}/tffumpt"
