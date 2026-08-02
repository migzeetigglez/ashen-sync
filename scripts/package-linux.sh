#!/usr/bin/env bash
# Build a Linux amd64 distribution (Steam Deck / desktop).
# On Linux hosts: prefers a full GUI build; falls back to -tags nogui.
# When cross-compiling (e.g. from macOS): builds CLI/watcher with -tags nogui.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ARCH="${TARGET_ARCH:-amd64}"
STAGE="dist/save-channeler-linux-${ARCH}"
TARBALL="dist/save-channeler-linux-${ARCH}.tar.gz"
BIN_NAME="save-channeler"

mkdir -p dist
rm -rf "$STAGE" "$TARBALL"
mkdir -p "$STAGE/bin" \
  "$STAGE/share/applications" \
  "$STAGE/share/icons/hicolor/256x256/apps" \
  "$STAGE/share/icons/hicolor/128x128/apps" \
  "$STAGE/share/icons/hicolor/64x64/apps" \
  "$STAGE/share/doc/save-channeler"

build_gui=0
if [[ "$(uname -s)" == "Linux" ]]; then
  echo "Building native Linux binary (GUI if deps available)…"
  if CGO_ENABLED=1 go build -o "$STAGE/bin/$BIN_NAME" ./cmd/save-channeler; then
    build_gui=1
    echo "GUI build OK"
  else
    echo "GUI build failed — falling back to nogui CLI"
    CGO_ENABLED=0 go build -tags nogui -o "$STAGE/bin/$BIN_NAME" ./cmd/save-channeler
  fi
else
  echo "Cross-compiling Linux/${ARCH} CLI (-tags nogui)…"
  GOOS=linux GOARCH="$ARCH" CGO_ENABLED=0 go build -tags nogui -o "$STAGE/bin/$BIN_NAME" ./cmd/save-channeler
fi
chmod +x "$STAGE/bin/$BIN_NAME"

# Desktop entry with PATH-friendly Exec
sed 's|^Exec=.*|Exec=save-channeler gui|' assets/save-channeler.desktop \
  > "$STAGE/share/applications/save-channeler.desktop"

# Icons
if [[ -f assets/iconset/icon-256.png ]]; then
  cp assets/iconset/icon-256.png "$STAGE/share/icons/hicolor/256x256/apps/save-channeler.png"
fi
if [[ -f assets/iconset/icon-128.png ]]; then
  cp assets/iconset/icon-128.png "$STAGE/share/icons/hicolor/128x128/apps/save-channeler.png"
fi
if [[ -f assets/iconset/icon-64.png ]]; then
  cp assets/iconset/icon-64.png "$STAGE/share/icons/hicolor/64x64/apps/save-channeler.png"
fi
# Fallback if iconset missing
if [[ ! -f "$STAGE/share/icons/hicolor/256x256/apps/save-channeler.png" && -f assets/icon.png ]]; then
  cp assets/icon.png "$STAGE/share/icons/hicolor/256x256/apps/save-channeler.png"
fi

cat > "$STAGE/install.sh" <<'INSTALL'
#!/usr/bin/env bash
# Install Save Channeler for the current user (~/.local)
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
PREFIX="${PREFIX:-$HOME/.local}"

mkdir -p "$PREFIX/bin" \
  "$PREFIX/share/applications" \
  "$PREFIX/share/icons/hicolor/256x256/apps" \
  "$PREFIX/share/icons/hicolor/128x128/apps" \
  "$PREFIX/share/icons/hicolor/64x64/apps"

install -m 755 "$HERE/bin/save-channeler" "$PREFIX/bin/save-channeler"
install -m 644 "$HERE/share/applications/save-channeler.desktop" \
  "$PREFIX/share/applications/save-channeler.desktop"

for sz in 256x256 128x128 64x64; do
  src="$HERE/share/icons/hicolor/${sz}/apps/save-channeler.png"
  if [[ -f "$src" ]]; then
    install -m 644 "$src" "$PREFIX/share/icons/hicolor/${sz}/apps/save-channeler.png"
  fi
done

# Point desktop entry at the installed binary (GNU sed on Linux/SteamOS)
sed -i "s|^Exec=.*|Exec=$PREFIX/bin/save-channeler gui|" \
  "$PREFIX/share/applications/save-channeler.desktop"

if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$PREFIX/share/applications" 2>/dev/null || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -f -t "$PREFIX/share/icons/hicolor" 2>/dev/null || true
fi

echo "Installed save-channeler to $PREFIX/bin"
echo "Ensure $PREFIX/bin is on your PATH, then run: save-channeler doctor"
echo "Also install rclone: https://rclone.org/install/"
INSTALL
chmod +x "$STAGE/install.sh"

cat > "$STAGE/share/doc/save-channeler/README.txt" <<EOF
Save Channeler for Linux / Steam Deck
=====================================

Quick install (Desktop Mode on Deck is fine):
  ./install.sh

Or copy bin/save-channeler somewhere on your PATH.

Requires rclone on PATH: https://rclone.org/install/

Then:
  save-channeler doctor
  save-channeler gui          # if this build includes the GUI
  save-channeler play ds3

Build type: $([[ "$build_gui" == 1 ]] && echo "GUI+CLI" || echo "CLI/watcher (-tags nogui)")
EOF

tar -C dist -czf "$TARBALL" "$(basename "$STAGE")"
echo "Built $STAGE"
echo "Archive $TARBALL"
echo "On the Deck/Linux box: tar xzf $(basename "$TARBALL") && cd $(basename "$STAGE") && ./install.sh"
