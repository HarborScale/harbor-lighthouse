#!/bin/bash
set -e

VERSION="v0.1.8"
REPO="harborscale/harbor-lighthouse"
BINARY_NAME="lighthouse"
INSTALL_PATH="/usr/local/bin/$BINARY_NAME"

# --- 🗑️ UNINSTALL MODE ---
if [ "$1" == "--uninstall" ]; then
    echo "🧹 Uninstalling Harbor Lighthouse..."

    # 1. Ask binary to remove the Service
    if command -v $BINARY_NAME &> /dev/null; then
        echo "   Stopping service..."
        sudo $BINARY_NAME --uninstall || true
    fi

    # 2. Remove the binary file
    if [ -f "$INSTALL_PATH" ]; then
        sudo rm "$INSTALL_PATH"
        echo "✅ Binary removed from $INSTALL_PATH"
    else
        echo "ℹ️  Binary not found (already removed?)"
    fi

    # 3. Optional: Remove Config (User might want to keep data, but here is how to purge)
    # sudo rm -rf /etc/harbor-lighthouse

    echo "✅ Uninstallation complete."
    exit 0
fi

# --- 🚢 INSTALL MODE ---
echo "🚢 Installing Harbor Lighthouse $VERSION..."

OS="$(uname -s)"
ARCH="$(uname -m)"

# Map Architecture
case "$ARCH" in
    x86_64) BIN_ARCH="amd64" ;;
    aarch64|arm64) BIN_ARCH="arm64" ;;
    *) echo "❌ Unsupported Arch: $ARCH"; exit 1 ;;
esac

# Map OS
case "$OS" in
    Linux) BIN_OS="linux" ;;
    Darwin) BIN_OS="darwin" ;;
    *) echo "❌ Unsupported OS: $OS"; exit 1 ;;
esac

# Construct Download URL (Expects raw binary assets like 'lighthouse-linux-amd64')
FILENAME="lighthouse-${BIN_OS}-${BIN_ARCH}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

echo "⬇️  Downloading $FILENAME..."
# Download to /tmp first to avoid partial writes to bin
curl -L -o /tmp/lighthouse "$URL"

# Make executable
chmod +x /tmp/lighthouse

# Move to final destination (Requires Sudo)
echo "📦 Installing to $INSTALL_PATH..."
sudo mv /tmp/lighthouse "$INSTALL_PATH"

# --- ⚡ INSTALL SERVICE ---
echo "⚙️  Registering System Service..."
# We run the install command as root
sudo $BINARY_NAME --install

echo "✅ Installed & Running (Idle)"
echo "👉 Now configure it: sudo lighthouse --add --name 'server-1' --harbor-id '123'"
