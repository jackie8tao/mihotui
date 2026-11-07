#!/usr/bin/env bash
# ========================================================
# Mihomo TUI Linux Installer
# ========================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}==>${NC} Installing Mihomo TUI..."

# Check root privilege
SUDO=""
if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        echo -e "${RED}Error:${NC} Root permissions required to install to /usr/local/bin."
        exit 1
    fi
fi

# Detect Architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        TARGET_ARCH="linux-amd64"
        ;;
    aarch64|arm64)
        TARGET_ARCH="linux-arm64"
        ;;
    armv7*|armhf)
        TARGET_ARCH="linux-armv7"
        ;;
    i386|i686)
        TARGET_ARCH="linux-386"
        ;;
    *)
        echo -e "${RED}Error:${NC} Unsupported CPU architecture: $ARCH"
        exit 1
        ;;
esac

echo -e "${BLUE}==>${NC} Detected architecture: ${GREEN}${TARGET_ARCH}${NC}"

INSTALL_DIR="/usr/local/bin"
BIN_NAME="mihomo-tui"

# Find binary to install
SRC_BIN=""
if [ -f "./$BIN_NAME" ]; then
    SRC_BIN="./$BIN_NAME"
elif [ -f "./${BIN_NAME}-${TARGET_ARCH}" ]; then
    SRC_BIN="./${BIN_NAME}-${TARGET_ARCH}"
elif [ -f "./dist/${BIN_NAME}-${TARGET_ARCH}" ]; then
    SRC_BIN="./dist/${BIN_NAME}-${TARGET_ARCH}"
elif [ -f "./build/${BIN_NAME}-${TARGET_ARCH}" ]; then
    SRC_BIN="./build/${BIN_NAME}-${TARGET_ARCH}"
fi

# If binary not found locally, try downloading from GitHub Releases
if [ -z "$SRC_BIN" ]; then
    REPO=${GITHUB_REPO:-"mihomo-tui/mihomo-tui"}
    echo -e "${YELLOW}Notice:${NC} Local binary not found. Attempting to fetch latest release from GitHub (${REPO})..."

    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    LATEST_TAG=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$LATEST_TAG" ]; then
        LATEST_TAG="v0.1.0"
    fi

    TAR_NAME="mihomo-tui-${LATEST_TAG}-${TARGET_ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${TAR_NAME}"

    echo -e "${BLUE}==>${NC} Downloading ${GREEN}${DOWNLOAD_URL}${NC}..."
    if curl -sSL -f -o "${TMP_DIR}/${TAR_NAME}" "$DOWNLOAD_URL"; then
        tar -zxf "${TMP_DIR}/${TAR_NAME}" -C "$TMP_DIR"
        SRC_BIN="${TMP_DIR}/${BIN_NAME}"
    else
        echo -e "${RED}Error:${NC} Failed to download release package. Please compile locally or check network."
        exit 1
    fi
fi

echo -e "${BLUE}==>${NC} Installing binary to ${INSTALL_DIR}/${BIN_NAME}..."
$SUDO cp "$SRC_BIN" "${INSTALL_DIR}/${BIN_NAME}"
$SUDO chmod +x "${INSTALL_DIR}/${BIN_NAME}"

# Ensure default directories exist
CONFIG_DIR="/etc/mihomo"
if [ ! -d "$CONFIG_DIR" ]; then
    echo -e "${BLUE}==>${NC} Creating Mihomo default config directory ${CONFIG_DIR}..."
    $SUDO mkdir -p "$CONFIG_DIR"
    $SUDO chmod 755 "$CONFIG_DIR"
fi

USER_CONFIG_DIR="${HOME}/.config/mihomo-tui"
mkdir -p "$USER_CONFIG_DIR"

CFG_EXAMPLE=""
if [ -f "./configs/config.example.yaml" ]; then
    CFG_EXAMPLE="./configs/config.example.yaml"
elif [ -f "./config.example.yaml" ]; then
    CFG_EXAMPLE="./config.example.yaml"
fi

if [ -n "$CFG_EXAMPLE" ] && [ ! -f "${USER_CONFIG_DIR}/config.yaml" ]; then
    echo -e "${BLUE}==>${NC} Setting up initial config in ${USER_CONFIG_DIR}/config.yaml..."
    cp "$CFG_EXAMPLE" "${USER_CONFIG_DIR}/config.yaml"
fi

echo -e "${GREEN}==> Installation completed successfully!${NC}"
echo -e "You can now run ${GREEN}mihomo-tui${NC} anywhere in your terminal."
echo -e "Usage examples:"
echo -e "  mihomo-tui                               # Connect to 127.0.0.1:9090"
echo -e "  mihomo-tui -u http://127.0.0.1:9090 -s secret -d /etc/mihomo"
