#!/bin/bash
# AniClew installer — downloads the latest release for your platform
set -e

REPO="Dannykkh/Ani-Clew"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  linux)   SUFFIX="linux" ;;
  darwin)  SUFFIX="mac" ;;
  *)       echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
  x86_64|amd64) SUFFIX="$SUFFIX-amd64" ;;
  aarch64|arm64) SUFFIX="$SUFFIX-arm64" ;;
  *)             echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

if [ "$OS" = "darwin" ] && [ "$ARCH" = "x86_64" ]; then
  SUFFIX="mac-intel"
fi

echo "Downloading AniClew for $OS/$ARCH..."

# Get latest release URL
LATEST=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep "browser_download_url.*$SUFFIX" | head -1 | cut -d '"' -f 4)

if [ -z "$LATEST" ]; then
  echo "No release found for $SUFFIX. Building from source..."
  echo "  git clone https://github.com/$REPO.git"
  echo "  cd Ani-Clew && go build -o aniclew ./cmd/proxy/"
  exit 1
fi

curl -sL "$LATEST" -o aniclew
chmod +x aniclew

if [ -w "$INSTALL_DIR" ]; then
  mv aniclew "$INSTALL_DIR/aniclew"
  echo "Installed to $INSTALL_DIR/aniclew"
else
  sudo mv aniclew "$INSTALL_DIR/aniclew"
  echo "Installed to $INSTALL_DIR/aniclew (sudo)"
fi

echo ""
echo "Run: aniclew"
echo "Web UI: http://localhost:4000/app"
