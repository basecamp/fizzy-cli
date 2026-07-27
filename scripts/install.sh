#!/usr/bin/env bash
set -euo pipefail

REPO="basecamp/fizzy-cli"
INSTALL_DIR="${FIZZY_BIN_DIR:-$HOME/.local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  mingw*|msys*|cygwin*) OS="windows" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

if [ "$OS" = "windows" ] && [ "$ARCH" = "arm64" ]; then
  echo "Windows ARM64 is not currently supported. See https://github.com/basecamp/fizzy-cli/releases for available builds."
  exit 1
fi

# Fetch latest version
echo "Fetching latest version..."
VERSION=$(curl -sI "https://github.com/$REPO/releases/latest" | grep -i '^location:' | sed 's/.*tag\///' | tr -d '\r\n' || true)
if [ -z "$VERSION" ]; then
  echo "Failed to determine latest version"
  exit 1
fi
echo "Latest version: $VERSION"

# Download release archive
EXT="tar.gz"
if [ "$OS" = "windows" ]; then
  EXT="zip"
fi
ARCHIVE="fizzy_${VERSION#v}_${OS}_${ARCH}.${EXT}"

DOWNLOAD_URL="https://github.com/$REPO/releases/download/${VERSION}/${ARCHIVE}"
CHECKSUMS_URL="https://github.com/$REPO/releases/download/${VERSION}/checksums.txt"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading $ARCHIVE..."
curl -fsSL "$DOWNLOAD_URL" -o "$TMPDIR/$ARCHIVE"
curl -fsSL "$CHECKSUMS_URL" -o "$TMPDIR/checksums.txt"

# Verify SHA256
echo "Verifying checksum..."
cd "$TMPDIR"
EXPECTED=$(awk -v f="$ARCHIVE" '$2 == f || $2 == "*" f {print $1}' checksums.txt)
if [ -z "$EXPECTED" ]; then
  echo "ERROR: Checksum for $ARCHIVE not found in checksums.txt"
  exit 1
fi
ACTUAL=$(sha256sum "$ARCHIVE" 2>/dev/null || shasum -a 256 "$ARCHIVE" | awk '{print $1}')
ACTUAL=$(echo "$ACTUAL" | awk '{print $1}')
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "ERROR: Checksum mismatch!"
  echo "  Expected: $EXPECTED"
  echo "  Actual:   $ACTUAL"
  exit 1
fi
echo "Checksum verified."

# Extract
BINARY="fizzy"
if [ "$OS" = "windows" ]; then
  BINARY="fizzy.exe"
fi
if [ "$EXT" = "zip" ]; then
  if command -v unzip > /dev/null; then
    unzip -q "$ARCHIVE" "$BINARY"
  elif tar -xf "$ARCHIVE" "$BINARY" 2>/dev/null && [ -f "$BINARY" ]; then
    :  # bsdtar (Windows 10+ tar.exe) reads zip archives
  else
    echo "ERROR: Could not extract $ARCHIVE — install unzip and re-run"
    exit 1
  fi
else
  tar -xzf "$ARCHIVE" "$BINARY"
fi

# Install
mkdir -p "$INSTALL_DIR"
cp "$BINARY" "$INSTALL_DIR/${BINARY}"
chmod +x "$INSTALL_DIR/${BINARY}"

echo ""
echo "fizzy ${VERSION} installed to $INSTALL_DIR/${BINARY}"

# Check if install dir is in PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo ""
  echo "Add $INSTALL_DIR to your PATH:"
  SHELL_NAME=$(basename "${SHELL:-bash}")
  case "$SHELL_NAME" in
    zsh)  echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.zshrc && source ~/.zshrc" ;;
    fish) echo "  fish_add_path $INSTALL_DIR" ;;
    *)    echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.bashrc && source ~/.bashrc" ;;
  esac
fi

echo ""
echo "Run '$INSTALL_DIR/${BINARY} setup' to get started."
