#!/usr/bin/env bash
# BOI CLI -- Unix Installer Script (macOS / Linux)
# Usage: curl -fsSL https://boi.sh/install.sh | bash

set -euo pipefail

# ===============================================================
# Configuration
# ===============================================================

VERSION="${1:-latest}"
INSTALL_DIR="$HOME/.boi/bin"
REPO="wersoul-source/BOI-CLI"
SKIP_CHECKSUM="${BOI_SKIP_CHECKSUM:-0}"
SKIP_INIT="${BOI_SKIP_INIT:-0}"

# ===============================================================
# Colors
# ===============================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# ===============================================================
# STEP 0: Print Banner
# ===============================================================

echo ""
echo -e "${CYAN}  BOI CLI Installer${NC}"
echo -e "${GRAY}  AI Agent Runtime -- BOI Family${NC}"
echo ""

# ===============================================================
# STEP 1: Detect OS + Architecture
# ===============================================================

echo -e "${YELLOW}[1/8] Detecting environment...${NC}"

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux)   OS="Linux" ;;
    Darwin)  OS="Darwin" ;;
    *)
        echo -e "${RED}  Error: Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64)  ARCH="arm64" ;;
    *)
        echo -e "${RED}  Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "  OS:   ${GRAY}$OS${NC}"
echo -e "  Arch: ${GRAY}$ARCH${NC}"

# ===============================================================
# STEP 2: Resolve Version
# ===============================================================

echo -e "${YELLOW}[2/8] Resolving version...${NC}"

if [ "$VERSION" = "latest" ]; then
    RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest"
else
    RELEASE_URL="https://api.github.com/repos/$REPO/releases/tags/$VERSION"
fi

if command -v curl >/dev/null 2>&1; then
    RELEASE_JSON=$(curl -sSL --connect-timeout 10 "$RELEASE_URL" 2>/dev/null) || {
        echo -e "${RED}  Error: Could not fetch release info. Check your internet connection.${NC}"
        exit 1
    }
elif command -v wget >/dev/null 2>&1; then
    RELEASE_JSON=$(wget -qO- --timeout=10 "$RELEASE_URL" 2>/dev/null) || {
        echo -e "${RED}  Error: Could not fetch release info.${NC}"
        exit 1
    }
else
    echo -e "${RED}  Error: Neither curl nor wget found. Install one of them.${NC}"
    exit 1
fi

VERSION_TAG=$(echo "$RELEASE_JSON" | grep -o '"tag_name": *"[^"]*"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
echo -e "  Version: ${GRAY}$VERSION_TAG${NC}"

# ===============================================================
# STEP 3: Build Download URL
# ===============================================================

echo -e "${YELLOW}[3/8] Building download URL...${NC}"

# Normalize OS name to lowercase (Linux → linux, Darwin → darwin)
OS_LOWER=$(echo "$OS" | tr '[:upper:]' '[:lower:]')

# Strip 'v' prefix from version tag for asset naming (v0.3.0 → 0.3.0)
VER_NUM="${VERSION_TAG#v}"

BINARY_NAME="boi_${VER_NUM}_${OS_LOWER}_${ARCH}.tar.gz"
DOWNLOAD_URL=$(echo "$RELEASE_JSON" | grep -o "\"browser_download_url\": *\"[^\"]*$BINARY_NAME\"" | head -1 | sed 's/.*"browser_download_url": *"\([^"]*\)".*/\1/')

if [ -z "$DOWNLOAD_URL" ]; then
    echo -e "${RED}  Error: No asset found matching: $BINARY_NAME${NC}"
    echo -e "${GRAY}  Available assets:${NC}"
    echo "$RELEASE_JSON" | grep -o '"name": *"[^"]*"' | sed 's/"name": *"\([^"]*\)"/    \1/'
    exit 1
fi

echo -e "  URL: ${GRAY}$DOWNLOAD_URL${NC}"

# ===============================================================
# STEP 4: Create Install Directory
# ===============================================================

echo -e "${YELLOW}[4/8] Creating install directory...${NC}"

mkdir -p "$INSTALL_DIR"

BINARY_PATH="$INSTALL_DIR/boi"
BACKUP_PATH="$INSTALL_DIR/boi.old"

# Backup existing binary
if [ -f "$BINARY_PATH" ]; then
    echo -e "  ${GRAY}Backing up existing binary...${NC}"
    cp "$BINARY_PATH" "$BACKUP_PATH" 2>/dev/null || true
fi

echo -e "  Directory: ${GRAY}$INSTALL_DIR${NC}"

# ===============================================================
# STEP 5: Download Binary
# ===============================================================

echo -e "${YELLOW}[5/8] Downloading BOI CLI binary...${NC}"

TEMP_DIR="$(mktemp -d)"
TEMP_ARCHIVE="$TEMP_DIR/boi.tar.gz"

if command -v curl >/dev/null 2>&1; then
    curl -sSL --connect-timeout 30 --max-time 120 -o "$TEMP_ARCHIVE" "$DOWNLOAD_URL" || {
        echo -e "${RED}  Error: Download failed.${NC}"
        rm -rf "$TEMP_DIR"
        exit 1
    }
else
    wget -q --timeout=120 -O "$TEMP_ARCHIVE" "$DOWNLOAD_URL" || {
        echo -e "${RED}  Error: Download failed.${NC}"
        rm -rf "$TEMP_DIR"
        exit 1
    }
fi

# Extract
tar -xzf "$TEMP_ARCHIVE" -C "$TEMP_DIR"

# Find and move binary
EXTRACTED_BINARY=$(find "$TEMP_DIR" -name "boi" -type f | head -1)
if [ -z "$EXTRACTED_BINARY" ]; then
    echo -e "${RED}  Error: Could not find boi binary in archive${NC}"
    rm -rf "$TEMP_DIR"
    exit 1
fi

cp "$EXTRACTED_BINARY" "$BINARY_PATH"
chmod +x "$BINARY_PATH"

echo -e "  Installed: ${GRAY}$BINARY_PATH${NC}"

# Cleanup
rm -rf "$TEMP_DIR"

# ===============================================================
# STEP 6: Verify Checksum
# ===============================================================

echo -e "${YELLOW}[6/8] Verifying checksum...${NC}"

if [ "$SKIP_CHECKSUM" = "0" ]; then
    CHECKSUM_URL=$(echo "$RELEASE_JSON" | grep -o "\"browser_download_url\": *\"[^\"]*SHA256SUMS.txt\"" | head -1 | sed 's/.*"browser_download_url": *"\([^"]*\)".*/\1/')
    
    if [ -n "$CHECKSUM_URL" ]; then
        if command -v curl >/dev/null 2>&1; then
            CHECKSUMS=$(curl -sSL --connect-timeout 10 "$CHECKSUM_URL" 2>/dev/null) || true
        else
            CHECKSUMS=$(wget -qO- --timeout=10 "$CHECKSUM_URL" 2>/dev/null) || true
        fi
        
        EXPECTED_HASH=""
        if [ -n "$CHECKSUMS" ]; then
            EXPECTED_HASH=$(echo "$CHECKSUMS" | grep "boi_${VER_NUM}_${OS_LOWER}_${ARCH}" | awk '{print $1}' | tr '[:upper:]' '[:lower:]')
        fi
        
        if [ -n "$EXPECTED_HASH" ]; then
            if command -v shasum >/dev/null 2>&1; then
                ACTUAL_HASH=$(shasum -a 256 "$BINARY_PATH" | awk '{print $1}')
            elif command -v sha256sum >/dev/null 2>&1; then
                ACTUAL_HASH=$(sha256sum "$BINARY_PATH" | awk '{print $1}')
            else
                echo -e "  ${YELLOW}Warning: No sha256 tool found. Skipping checksum.${NC}"
                ACTUAL_HASH=""
            fi
            
            if [ -n "$ACTUAL_HASH" ]; then
                if [ "$ACTUAL_HASH" = "$EXPECTED_HASH" ]; then
                    echo -e "  ${GREEN}Checksum verified OK${NC}"
                else
                    echo -e "${RED}  WARNING: Checksum mismatch!${NC}"
                    echo -e "    Expected: ${GRAY}$EXPECTED_HASH${NC}"
                    echo -e "    Actual:   ${GRAY}$ACTUAL_HASH${NC}"
                    
                    if [ -f "$BACKUP_PATH" ]; then
                        echo -e "  ${YELLOW}Restoring previous version...${NC}"
                        cp "$BACKUP_PATH" "$BINARY_PATH"
                    fi
                    exit 1
                fi
            fi
        else
            echo -e "  ${YELLOW}Warning: Could not find checksum entry for this platform${NC}"
        fi
    else
        echo -e "  ${YELLOW}Warning: No SHA256SUMS.txt found in release${NC}"
    fi
else
    echo -e "  ${GRAY}Skipped (BOI_SKIP_CHECKSUM=1)${NC}"
fi

# ===============================================================
# STEP 7: Add to PATH
# ===============================================================

echo -e "${YELLOW}[7/8] Adding to PATH...${NC}"

SHELL_CONFIG=""

case "$SHELL" in
    */zsh)
        if [ -f "$HOME/.zshrc" ]; then
            SHELL_CONFIG="$HOME/.zshrc"
        elif [ -f "$HOME/.zprofile" ]; then
            SHELL_CONFIG="$HOME/.zprofile"
        fi
        ;;
    */bash)
        if [ -f "$HOME/.bashrc" ]; then
            SHELL_CONFIG="$HOME/.bashrc"
        elif [ -f "$HOME/.bash_profile" ]; then
            SHELL_CONFIG="$HOME/.bash_profile"
        elif [ -f "$HOME/.profile" ]; then
            SHELL_CONFIG="$HOME/.profile"
        fi
        ;;
    */fish)
        SHELL_CONFIG="$HOME/.config/fish/config.fish"
        mkdir -p "$(dirname "$SHELL_CONFIG")"
        ;;
esac

PATH_LINE='export PATH="$HOME/.boi/bin:$PATH"'

if [ -n "$SHELL_CONFIG" ]; then
    if [ -f "$SHELL_CONFIG" ]; then
        if grep -q ".boi/bin" "$SHELL_CONFIG" 2>/dev/null; then
            echo -e "  ${GRAY}Already in $SHELL_CONFIG${NC}"
        else
            echo "" >> "$SHELL_CONFIG"
            echo "# BOI CLI" >> "$SHELL_CONFIG"
            echo "$PATH_LINE" >> "$SHELL_CONFIG"
            echo -e "  ${GREEN}Added to $SHELL_CONFIG${NC}"
        fi
    fi
else
    echo -e "  ${YELLOW}Warning: Could not detect shell config file.${NC}"
    echo -e "  ${GRAY}Add this line to your profile:${NC}"
    echo -e "    ${CYAN}$PATH_LINE${NC}"
fi

# Update current session
export PATH="$HOME/.boi/bin:$PATH"

# ===============================================================
# STEP 8: Initialize
# ===============================================================

echo -e "${YELLOW}[8/8] Initializing workspace...${NC}"

if [ "$SKIP_INIT" = "0" ]; then
    if "$BINARY_PATH" init --silent 2>/dev/null; then
        echo -e "  ${GREEN}Workspace initialized${NC}"
    else
        echo -e "  ${YELLOW}Warning: Could not auto-initialize workspace${NC}"
        echo -e "  ${GRAY}Run 'boi init' manually after install${NC}"
    fi
else
    echo -e "  ${GRAY}Skipped (BOI_SKIP_INIT=1)${NC}"
fi

# ===============================================================
# SUCCESS
# ===============================================================

echo ""
echo -e "  ${GREEN}============================================${NC}"
echo -e "  ${GREEN}  BOI CLI installed successfully!${NC}"
echo -e "  ${GREEN}============================================${NC}"
echo ""
echo -e "  Version:  ${CYAN}$VERSION_TAG${NC}"
echo -e "  Binary:   ${CYAN}$BINARY_PATH${NC}"
echo -e "  Config:   ${CYAN}$HOME/.boi${NC}"
echo ""
echo -e "  ${WHITE}Quick start:${NC}"
echo -e "    ${GRAY}boi              Launch TUI${NC}"
echo -e "    ${GRAY}boi ask 'hello'  Test AI${NC}"
echo -e "    ${GRAY}boi --help        All commands${NC}"
echo ""
echo -e "  ${WHITE}Next -- Set up LLM providers:${NC}"
echo -e "    ${GRAY}cd to your project${NC}"
echo -e "    ${GRAY}cp .env.example .env${NC}"
echo -e "    ${GRAY}nano .env  # Add PSC_1_API_KEY=...${NC}"
echo ""
echo -e "  ${GRAY}Docs: https://github.com/wersoul-source/BOI-CLI${NC}"
echo ""
