#!/bin/sh
set -e

REPO="eduardmaghakyan/why"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

DEST="${1:-/usr/local/bin}"
URL="https://github.com/$REPO/releases/latest/download/why_${OS}_${ARCH}.tar.gz"

echo "Downloading why for ${OS}/${ARCH}..."
curl -fsSL "$URL" | tar xz -C "$DEST" why
echo "Installed to $DEST/why"
