#!/bin/bash
VERSION="v0.0.7" # UPDATE THIS
REPO="harborscale/harbor-lighthouse"

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

echo "✅ Installed! Run: lighthouse --add --name 'test' --harbor-id '123'"
