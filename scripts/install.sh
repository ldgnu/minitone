#!/usr/bin/env bash
# minitone installer — cross-platform one-liner downloader.
#   curl -fsSL https://raw.githubusercontent.com/ldgnu/minitone/master/scripts/install.sh | sh
# Detects OS/arch, downloads the matching GitHub release tarball and installs
# the binary. Set MINITONE_VERSION to pin a version, or pass it as $1.
set -euo pipefail

REPO="ldgnu/minitone"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BIN="minitone"

err() { echo "error: $*" >&2; exit 1; }

# --- detect OS ---------------------------------------------------------------
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *) err "unsupported OS: $(uname -s) (supported: linux, darwin)" ;;
esac

# --- detect arch -------------------------------------------------------------
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) err "unsupported arch: $ARCH (supported: amd64, arm64)" ;;
esac

# --- version -----------------------------------------------------------------
VERSION="${MINITONE_VERSION:-${1:-}}"
if [[ -z "$VERSION" ]]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
  [[ -n "$VERSION" ]] || err "could not determine latest version (rate limited? set MINITONE_VERSION)"
fi
VERSION="${VERSION#v}"

ASSET="minitone-${VERSION}-${OS}-${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ASSET}"

# --- download ----------------------------------------------------------------
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "→ downloading ${ASSET} (v${VERSION})"
curl -fsSL "$URL" -o "$TMP/$ASSET" || err "download failed: $URL"

tar -xzf "$TMP/$ASSET" -C "$TMP"
[[ -f "$TMP/$BIN" ]] || err "binary not found in archive"

# --- install -----------------------------------------------------------------
if [[ ! -d "$INSTALL_DIR" ]]; then
  mkdir -p "$INSTALL_DIR" 2>/dev/null || true
fi
if [[ -w "$INSTALL_DIR" ]]; then
  install -m 0755 "$TMP/$BIN" "$INSTALL_DIR/$BIN"
else
  echo "→ $INSTALL_DIR not writable, installing to ~/.local/bin (add to PATH)"
  mkdir -p ~/.local/bin
  install -m 0755 "$TMP/$BIN" ~/.local/bin/$BIN
  INSTALL_DIR="$HOME/.local/bin"
fi

echo "✓ minitone v${VERSION} installed to ${INSTALL_DIR}/${BIN}"
echo "  requires: mpv  (yt-dlp for YouTube). run: minitone --help"
