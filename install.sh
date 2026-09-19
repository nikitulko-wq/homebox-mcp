#!/bin/sh
# install.sh — install a prebuilt homebox-mcp binary from GitHub releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/nikitulko-wq/homebox-mcp/main/install.sh | sh
#
# Options (environment variables):
#   VERSION     version to install, e.g. v1.0.0 (default: latest release)
#   INSTALL_DIR destination directory (default: /usr/local/bin or ~/.local/bin)
set -eu

REPO="nikitulko-wq/homebox-mcp"
BIN="homebox-mcp"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-}"

OS=$(uname -s)
ARCH=$(uname -m)
case "$OS" in
  Linux)  GOOS=linux ;;
  Darwin) GOOS=darwin ;;
  *)
    echo "error: unsupported OS: $OS" >&2
    echo "Windows users: download the .zip from https://github.com/$REPO/releases" >&2
    exit 1
    ;;
esac
case "$ARCH" in
  x86_64 | amd64)  GOARCH=amd64 ;;
  arm64 | aarch64) GOARCH=arm64 ;;
  *)
    echo "error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- "$1"; }
else
  echo "error: neither curl nor wget found" >&2
  exit 1
fi

if [ "$VERSION" = "latest" ]; then
  TAG=$(fetch "https://api.github.com/repos/$REPO/releases/latest" |
    grep -m1 '"tag_name"' | cut -d'"' -f4)
else
  case "$VERSION" in
    v*) TAG="$VERSION" ;;
    *)  TAG="v$VERSION" ;;
  esac
fi
if [ -z "${TAG:-}" ]; then
  echo "error: could not determine version (no releases published yet?)" >&2
  exit 1
fi

ASSET="${BIN}_${TAG#v}_${GOOS}_${GOARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$TAG/$ASSET"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading $ASSET ..."
if ! fetch "$URL" | tar -xz -C "$TMPDIR"; then
  echo "error: download failed: $URL" >&2
  exit 1
fi

if [ -z "$INSTALL_DIR" ]; then
  if [ -w /usr/local/bin ]; then
    INSTALL_DIR=/usr/local/bin
  else
    INSTALL_DIR="$HOME/.local/bin"
  fi
fi
mkdir -p "$INSTALL_DIR"
mv "$TMPDIR/$BIN" "$INSTALL_DIR/$BIN"
chmod +x "$INSTALL_DIR/$BIN"

echo "Installed homebox-mcp $TAG -> $INSTALL_DIR/$BIN"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "note: $INSTALL_DIR is not in your PATH" >&2 ;;
esac
