#!/usr/bin/env bash
# Build a Windows amd64 distribution zip.
# On Windows (Git Bash / MSYS): prefers a full GUI build; falls back to -tags nogui.
# When cross-compiling (e.g. from macOS/Linux): builds CLI/watcher with -tags nogui.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ARCH="${TARGET_ARCH:-amd64}"
STAGE="dist/save-channeler-windows-${ARCH}"
ZIP="dist/save-channeler-windows-${ARCH}.zip"
EXE_NAME="save-channeler.exe"

mkdir -p dist
rm -rf "$STAGE" "$ZIP"
mkdir -p "$STAGE"

# Optional .ico for shipping alongside the exe (Explorer association / shortcuts)
if [[ ! -f assets/icon.ico ]]; then
  if command -v magick >/dev/null 2>&1 && [[ -f assets/icon.png ]]; then
    echo "Generating assets/icon.ico…"
    magick assets/icon.png -define icon:auto-resize=256,128,64,48,32,16 assets/icon.ico
  else
    echo "note: assets/icon.ico missing (install ImageMagick to generate from icon.png)"
  fi
fi

build_gui=0
# Native Windows build (MINGW / CYGWIN / Windows uname)
uname_s="$(uname -s 2>/dev/null || echo unknown)"
if [[ "$uname_s" == MINGW* || "$uname_s" == MSYS* || "$uname_s" == CYGWIN* || "$uname_s" == Windows* ]]; then
  echo "Building native Windows binary (GUI if deps available)…"
  if go build -o "$STAGE/$EXE_NAME" ./cmd/save-channeler; then
    build_gui=1
    echo "GUI build OK"
  else
    echo "GUI build failed — falling back to nogui CLI"
    go build -tags nogui -o "$STAGE/$EXE_NAME" ./cmd/save-channeler
  fi
else
  echo "Cross-compiling Windows/${ARCH} CLI (-tags nogui)…"
  GOOS=windows GOARCH="$ARCH" CGO_ENABLED=0 go build -tags nogui -o "$STAGE/$EXE_NAME" ./cmd/save-channeler
fi

if [[ -f assets/icon.ico ]]; then
  cp assets/icon.ico "$STAGE/save-channeler.ico"
fi
if [[ -f assets/icon.png ]]; then
  cp assets/icon.png "$STAGE/save-channeler.png"
fi

cat > "$STAGE/play-ds3.bat" <<'BAT'
@echo off
cd /d "%~dp0"
save-channeler.exe play ds3
if errorlevel 1 pause
BAT

cat > "$STAGE/play-ds2.bat" <<'BAT'
@echo off
cd /d "%~dp0"
save-channeler.exe play ds2
if errorlevel 1 pause
BAT

cat > "$STAGE/README.txt" <<EOF
Save Channeler for Windows
==========================

1. Install rclone and put it on your PATH:
   https://rclone.org/install/

2. Put this folder somewhere permanent (e.g. %%USERPROFILE%%\Games\SaveChanneler).

3. Open a terminal here, or double-click play-ds3.bat after setup:
   save-channeler.exe doctor
   save-channeler.exe init
   save-channeler.exe play ds3

GUI:
  - This zip build type: $([[ "$build_gui" == 1 ]] && echo "GUI+CLI — run save-channeler.exe with no args or: save-channeler.exe gui" || echo "CLI only (cross-compiled). For the desktop GUI, build on a Windows PC without -tags nogui.")

Optional: set save-channeler.ico as the icon on a shortcut to save-channeler.exe.
EOF

# Zip contents (prefer PowerShell on Windows, zip elsewhere)
if command -v zip >/dev/null 2>&1; then
  (cd dist && zip -r "$(basename "$ZIP")" "$(basename "$STAGE")")
elif command -v powershell.exe >/dev/null 2>&1; then
  powershell.exe -NoProfile -Command \
    "Compress-Archive -Path '$(cygpath -w "$STAGE" 2>/dev/null || echo "$STAGE")' -DestinationPath '$(cygpath -w "$ZIP" 2>/dev/null || echo "$ZIP")' -Force"
else
  echo "warning: no zip tool — folder left at $STAGE" >&2
  echo "Built $STAGE"
  exit 0
fi

echo "Built $STAGE"
echo "Archive $ZIP"
echo "Copy the zip to your Windows PC, extract, install rclone, run save-channeler.exe doctor"
