#!/bin/bash
VERSION="v0.0.9"
REPO="harborscale/harbor-lighthouse"

# --- 🗑️ UNINSTALL MODE ---
if [ "$1" == "--uninstall" ]; then
    echo "🧹 Uninstalling Harbor Lighthouse..."

    # 1. Ask binary to remove the Service
    if command -v lighthouse &> /dev/null; then
        lighthouse --uninstall
    fi

    # 2. Remove the binary file
    if [ -f "/usr/local/bin/lighthouse" ]; then
        sudo rm /usr/local/bin/lighthouse
        echo "✅ Binary removed from /usr/local/bin"
    else
        echo "ℹ️  Binary not found (already removed?)"
    fi

    echo "✅ Uninstallation complete."
    exit 0
fi
# -------------------------

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

URL="https://github.com/${REPO}/releases/download/${VERSION}/lighthouse-${BIN_OS}-${BIN_ARCH}"

echo "⬇️  Downloading..."
curl -L -o lighthouse "$URL"
chmod +x lighthouse
sudo mv lighthouse /usr/local/bin/lighthouse

# --- ⚡ INSTALL SERVICE NOW ---
# We always try to install/update the service here.
echo "⚙️  Registering System Service..."
sudo lighthouse --install

echo "✅ Installed & Running (Idle)"
echo "👉 Now configure it: lighthouse --add --name 'server-1' --harbor-id '123'"
