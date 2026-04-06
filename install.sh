#!/bin/bash
# AniClew installer — downloads the latest release for your platform
set -e

REPO="Dannykkh/Ani-Clew"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo "AniClew Installer"
echo "=================="

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)             echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

BINARY="aniclew-${OS}-${ARCH}"
if [ "$OS" = "windows" ] || [ "$OS" = "mingw"* ]; then
  BINARY="${BINARY}.exe"
fi

echo "Platform: $OS/$ARCH"
echo "Binary: $BINARY"

# Get latest release
echo "Fetching latest release..."
DOWNLOAD_URL=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep "browser_download_url.*${OS}-${ARCH}" | head -1 | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
  echo ""
  echo "No pre-built binary found. Building from source..."
  echo ""
  echo "  git clone https://github.com/$REPO.git && cd Ani-Clew"
  echo "  cd web && npm install && npm run build && cd .."
  echo "  cp -r web/dist/* internal/server/webdist/"
  echo "  go build -o aniclew ./cmd/proxy"
  echo ""
  exit 1
fi

echo "Downloading: $DOWNLOAD_URL"
curl -sL "$DOWNLOAD_URL" -o aniclew
chmod +x aniclew

if [ -w "$INSTALL_DIR" ]; then
  mv aniclew "$INSTALL_DIR/aniclew"
else
  sudo mv aniclew "$INSTALL_DIR/aniclew"
fi

echo ""
echo "Installed: $INSTALL_DIR/aniclew"
echo ""
echo "Quick start:"
echo "  aniclew                              # interactive provider select"
echo "  aniclew -provider ollama -model qwen3:14b  # direct start"
echo ""
echo "Web UI: http://localhost:4000/app"
echo ""
echo "Connect CLI tools:"
echo "  ANTHROPIC_BASE_URL=http://localhost:4000 claude"
echo "  OPENAI_BASE_URL=http://localhost:4000 codex"
