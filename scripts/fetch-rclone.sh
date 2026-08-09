#!/usr/bin/env bash
# Download a pinned rclone binary into DEST_DIR.
# Usage: fetch-rclone.sh <linux|windows|macos> <dest_dir>
# Env: RCLONE_VERSION (default 1.75.0), RCLONE_ARCH (macos only: arm64|amd64|x86_64)
set -euo pipefail

OS_TARGET="${1:?usage: fetch-rclone.sh <linux|windows|macos> <dest_dir>}"
DEST_DIR="${2:?usage: fetch-rclone.sh <linux|windows|macos> <dest_dir>}"
RCLONE_VERSION="${RCLONE_VERSION:-1.75.0}"

mkdir -p "$DEST_DIR"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

case "$OS_TARGET" in
  linux)
    ARCHIVE="rclone-v${RCLONE_VERSION}-linux-amd64.zip"
    BIN_SRC="rclone-v${RCLONE_VERSION}-linux-amd64/rclone"
    BIN_DST="$DEST_DIR/rclone"
    ;;
  windows)
    ARCHIVE="rclone-v${RCLONE_VERSION}-windows-amd64.zip"
    BIN_SRC="rclone-v${RCLONE_VERSION}-windows-amd64/rclone.exe"
    BIN_DST="$DEST_DIR/rclone.exe"
    ;;
  macos|darwin|osx)
    arch="${RCLONE_ARCH:-$(uname -m)}"
    case "$arch" in
      arm64|aarch64) rarch=arm64 ;;
      x86_64|amd64) rarch=amd64 ;;
      *)
        echo "unsupported macOS arch: $arch (want arm64 or amd64)" >&2
        exit 2
        ;;
    esac
    ARCHIVE="rclone-v${RCLONE_VERSION}-osx-${rarch}.zip"
    BIN_SRC="rclone-v${RCLONE_VERSION}-osx-${rarch}/rclone"
    BIN_DST="$DEST_DIR/rclone"
    ;;
  *)
    echo "unsupported OS target: $OS_TARGET (want linux|windows|macos)" >&2
    exit 2
    ;;
esac

URL="https://downloads.rclone.org/v${RCLONE_VERSION}/${ARCHIVE}"
echo "Fetching rclone v${RCLONE_VERSION} ($OS_TARGET)…"
curl -fsSL -o "$TMP/$ARCHIVE" "$URL"
unzip -q -o "$TMP/$ARCHIVE" -d "$TMP"
install -m 755 "$TMP/$BIN_SRC" "$BIN_DST"
echo "Bundled $BIN_DST"
