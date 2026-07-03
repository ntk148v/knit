#!/usr/bin/env bash
set -euo pipefail

REPO="ntk148v/knit"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# --- step 1: ensure npx (Node.js) ---
if ! command -v npx &>/dev/null; then
  echo "Error: npx not found — install Node.js from https://nodejs.org" >&2
  exit 1
fi

# --- step 2: cache npx skills ---
echo "==> Ensuring npx skills is available..."
npx skills --version 2>/dev/null || echo "    (skills will be fetched on first use)"

# --- step 3: detect platform ---
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux|darwin) ;;
  *) echo "Error: unsupported OS: $OS" >&2; exit 1 ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Error: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# --- step 4: resolve latest release tag ---
echo "==> Fetching latest release..."
VERSION="$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' \
  | sed 's/.*"tag_name": "\(.*\)",.*/\1/')"

if [ -z "$VERSION" ]; then
  echo "Error: could not determine latest release" >&2
  exit 1
fi

# --- step 5: download & install ---
ARCHIVE="knit_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$ARCHIVE"

echo "==> Downloading knit $VERSION for $OS/$ARCH..."
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "$TMP/$ARCHIVE"
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"

echo "==> Installing knit to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR" 2>/dev/null || sudo mkdir -p "$INSTALL_DIR"
cp "$TMP/knit" "$INSTALL_DIR/knit" 2>/dev/null || sudo cp "$TMP/knit" "$INSTALL_DIR/knit"
chmod +x "$INSTALL_DIR/knit" 2>/dev/null || sudo chmod +x "$INSTALL_DIR/knit"

echo "==> Done! Run 'knit' to start."
