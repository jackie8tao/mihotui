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

SUITE_DIR="${SUITE_DIR:-/usr/local/mihotui}"
BIN_LINK_DIR="/usr/local/bin"
BIN_NAME="mihotui"

# Find binary to install
SRC_BIN=""
for candidate in "bin/$BIN_NAME" "$BIN_NAME" "${BIN_NAME}-${TARGET_ARCH}"; do
    if [ -f "./$candidate" ]; then
        SRC_BIN="./$candidate"
        break
    elif [ -f "./dist/$candidate" ]; then
        SRC_BIN="./dist/$candidate"
        break
    elif [ -f "./build/$candidate" ]; then
        SRC_BIN="./build/$candidate"
        break
    fi
done

CFG_EXAMPLE=""
for cfg_candidate in "./etc/mihotui/config.yaml" "./configs/config.example.yaml"; do
    if [ -f "$cfg_candidate" ]; then
        CFG_EXAMPLE="$cfg_candidate"
        break
    fi
done

# If binary not found locally, try downloading from GitHub Releases
if [ -z "$SRC_BIN" ]; then
    REPO=${GITHUB_REPO:-"jackie8tao/mihotui"}
    echo -e "${YELLOW}Notice:${NC} Local binary not found. Attempting to fetch latest release from GitHub (${REPO})..."

    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    LATEST_TAG=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$LATEST_TAG" ]; then
        LATEST_TAG="v0.1.0"
    fi

    TAR_NAME="mihotui-${LATEST_TAG}-${TARGET_ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${TAR_NAME}"

    echo -e "${BLUE}==>${NC} Downloading ${GREEN}${DOWNLOAD_URL}${NC}..."
    if ! curl -sSL -f -o "${TMP_DIR}/${TAR_NAME}" "$DOWNLOAD_URL"; then
        echo -e "${RED}Error:${NC} Failed to download release package from ${DOWNLOAD_URL}"
        exit 1
    fi

    if [ -f "${TMP_DIR}/${TAR_NAME}" ]; then
        tar -zxf "${TMP_DIR}/${TAR_NAME}" -C "$TMP_DIR"
        if [ -f "${TMP_DIR}/bin/${BIN_NAME}" ]; then
            SRC_BIN="${TMP_DIR}/bin/${BIN_NAME}"
        elif [ -f "${TMP_DIR}/${BIN_NAME}" ]; then
            SRC_BIN="${TMP_DIR}/${BIN_NAME}"
        fi
        if [ -z "$CFG_EXAMPLE" ] && [ -f "${TMP_DIR}/etc/mihotui/config.yaml" ]; then
            CFG_EXAMPLE="${TMP_DIR}/etc/mihotui/config.yaml"
        fi
    fi

    if [ -z "$SRC_BIN" ]; then
        echo -e "${RED}Error:${NC} Failed to locate or extract valid binary from release package."
        exit 1
    fi
fi

echo -e "${BLUE}==>${NC} Initializing self-contained suite directory at ${SUITE_DIR}..."
$SUDO mkdir -p "${SUITE_DIR}/bin"
$SUDO mkdir -p "${SUITE_DIR}/etc/mihotui/subscriptions"
$SUDO mkdir -p "${SUITE_DIR}/etc/mihomo"

echo -e "${BLUE}==>${NC} Installing binary to ${SUITE_DIR}/bin/${BIN_NAME}..."
$SUDO cp "$SRC_BIN" "${SUITE_DIR}/bin/${BIN_NAME}"
$SUDO chmod 755 "${SUITE_DIR}/bin/${BIN_NAME}"

echo -e "${BLUE}==>${NC} Linking ${BIN_LINK_DIR}/${BIN_NAME} -> ${SUITE_DIR}/bin/${BIN_NAME}..."
$SUDO mkdir -p "$BIN_LINK_DIR"
$SUDO ln -sf "${SUITE_DIR}/bin/${BIN_NAME}" "${BIN_LINK_DIR}/${BIN_NAME}"

SUITE_CFG="${SUITE_DIR}/etc/mihotui/config.yaml"
if [ -n "$CFG_EXAMPLE" ] && [ ! -f "$SUITE_CFG" ]; then
    echo -e "${BLUE}==>${NC} Writing initial suite config to ${SUITE_CFG}..."
    $SUDO cp "$CFG_EXAMPLE" "$SUITE_CFG"
    $SUDO chmod 644 "$SUITE_CFG"
fi

echo -e "${GREEN}==> Installation completed successfully!${NC}"
echo -e "Suite Root : ${GREEN}${SUITE_DIR}${NC}"
echo -e "Symlink    : ${GREEN}${BIN_LINK_DIR}/${BIN_NAME}${NC}"
echo -e "Usage examples:"
echo -e "  sudo mihotui install                     # Install Mihomo core & systemd service"
echo -e "  mihotui start                            # Launch interactive TUI"
echo -e "  mihotui start -c ${SUITE_CFG}"
