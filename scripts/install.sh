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

# Download binary
BINARY_NAME="fizzy-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  BINARY_NAME="fizzy-${OS}-${ARCH}.exe"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/${VERSION}/${BINARY_NAME}"
CHECKSUMS_URL="https://github.com/$REPO/releases/download/${VERSION}/SHA256SUMS-${OS}-${ARCH}.txt"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading $BINARY_NAME..."
curl -fsSL "$DOWNLOAD_URL" -o "$TMPDIR/$BINARY_NAME"
curl -fsSL "$CHECKSUMS_URL" -o "$TMPDIR/checksums.txt"

# Verify SHA256
echo "Verifying checksum..."
cd "$TMPDIR"
EXPECTED=$(awk '{print $1}' checksums.txt)
if [ -z "$EXPECTED" ]; then
  echo "ERROR: Checksum not found"
  exit 1
fi
ACTUAL=$(sha256sum "$BINARY_NAME" 2>/dev/null || shasum -a 256 "$BINARY_NAME" | awk '{print $1}')
ACTUAL=$(echo "$ACTUAL" | awk '{print $1}')
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "ERROR: Checksum mismatch!"
  echo "  Expected: $EXPECTED"
  echo "  Actual:   $ACTUAL"
  exit 1
fi
echo "Checksum verified."

# Install
mkdir -p "$INSTALL_DIR"
BINARY="fizzy"
if [ "$OS" = "windows" ]; then
  BINARY="fizzy.exe"
fi
cp "$BINARY_NAME" "$INSTALL_DIR/${BINARY}"
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
"$INSTALL_DIR/${BINARY}" setup || echo "Run '$INSTALL_DIR/${BINARY} setup' to get started."
