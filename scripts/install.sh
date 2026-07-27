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
  linux|darwin|freebsd|openbsd) ;;
  mingw*|msys*|cygwin*) OS="windows" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Fetch latest version
echo "Fetching latest version..."
VERSION=$(curl -sI "https://github.com/$REPO/releases/latest" | grep -i '^location:' | sed 's/.*tag\///' | tr -d '\r\n' || true)
if [ -z "$VERSION" ]; then
  echo "Failed to determine latest version"
  exit 1
fi
echo "Latest version: $VERSION"

# Select the release archive for this platform
VERSION_NUMBER="${VERSION#v}"
ARCHIVE_FORMAT="tar.gz"
BINARY="fizzy"
if [ "$OS" = "windows" ]; then
  ARCHIVE_FORMAT="zip"
  BINARY="fizzy.exe"
fi
ARCHIVE_NAME="fizzy_${VERSION_NUMBER}_${OS}_${ARCH}.${ARCHIVE_FORMAT}"

DOWNLOAD_URL="https://github.com/$REPO/releases/download/${VERSION}/${ARCHIVE_NAME}"
CHECKSUMS_URL="https://github.com/$REPO/releases/download/${VERSION}/checksums.txt"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading $ARCHIVE_NAME..."
curl -fsSL "$DOWNLOAD_URL" -o "$TMPDIR/$ARCHIVE_NAME"
curl -fsSL "$CHECKSUMS_URL" -o "$TMPDIR/checksums.txt"

# Verify the release archive before extracting it
echo "Verifying checksum..."
EXPECTED=$(awk -v name="$ARCHIVE_NAME" '$2 == name { print $1; exit }' "$TMPDIR/checksums.txt")
if [ -z "$EXPECTED" ]; then
  echo "ERROR: Checksum not found for $ARCHIVE_NAME"
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL=$(sha256sum "$TMPDIR/$ARCHIVE_NAME" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL=$(shasum -a 256 "$TMPDIR/$ARCHIVE_NAME" | awk '{print $1}')
elif command -v sha256 >/dev/null 2>&1; then
  ACTUAL=$(sha256 -q "$TMPDIR/$ARCHIVE_NAME")
elif command -v openssl >/dev/null 2>&1; then
  ACTUAL=$(openssl dgst -sha256 "$TMPDIR/$ARCHIVE_NAME" | awk '{print $NF}')
else
  echo "ERROR: A SHA-256 checksum tool is required (sha256sum, shasum, sha256, or openssl)"
  exit 1
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "ERROR: Checksum mismatch!"
  echo "  Expected: $EXPECTED"
  echo "  Actual:   $ACTUAL"
  exit 1
fi
echo "Checksum verified."

# Extract and install the binary
EXTRACT_DIR="$TMPDIR/extracted"
mkdir -p "$EXTRACT_DIR"
if [ "$ARCHIVE_FORMAT" = "zip" ]; then
  if ! command -v unzip >/dev/null 2>&1; then
    echo "ERROR: unzip is required to install Fizzy on Windows"
    exit 1
  fi
  unzip -q "$TMPDIR/$ARCHIVE_NAME" -d "$EXTRACT_DIR"
else
  tar -xzf "$TMPDIR/$ARCHIVE_NAME" -C "$EXTRACT_DIR"
fi

if [ ! -f "$EXTRACT_DIR/$BINARY" ]; then
  echo "ERROR: $BINARY was not found in $ARCHIVE_NAME"
  exit 1
fi

mkdir -p "$INSTALL_DIR"
cp "$EXTRACT_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"

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
