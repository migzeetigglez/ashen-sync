#!/usr/bin/env bash
# Build a macOS .app (+ zip) so the dock uses assets/icon.icns.
# Bundles a pinned rclone next to the app binary (no PATH install needed).
# Output: dist/Save Channeler.app and dist/save-channeler-macos-<arch>.zip
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f assets/icon.icns ]]; then
  echo "missing assets/icon.icns — generate it first" >&2
  exit 1
fi

VER="${VERSION_TAG:-}"
VER="${VER#v}"
if [[ -z "$VER" ]]; then
  VER="$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || true)"
fi
if [[ -z "$VER" ]]; then
  VER="0.0.0-dev"
fi

HOST_ARCH="$(uname -m)"
case "$HOST_ARCH" in
  arm64|aarch64) ARCH_NAME=arm64 ;;
  x86_64) ARCH_NAME=amd64 ;;
  *) ARCH_NAME="$HOST_ARCH" ;;
esac

APP="dist/Save Channeler.app"
ZIP="dist/save-channeler-macos-${ARCH_NAME}.zip"
LDFLAGS="$("$ROOT/scripts/version-ldflags.sh" || true)"
LDFLAG_ARGS=()
if [[ -n "$LDFLAGS" ]]; then
  LDFLAG_ARGS=(-ldflags "$LDFLAGS")
  echo "Stamping version via ldflags: $LDFLAGS"
fi

mkdir -p dist
# Remove previous bundle / zip
rm -rf "dist/Ashen Sync.app" "$APP" "$ZIP"

go build "${LDFLAG_ARGS[@]}" -o /tmp/save-channeler-bin ./cmd/save-channeler

mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp /tmp/save-channeler-bin "$APP/Contents/MacOS/Save Channeler"
chmod +x "$APP/Contents/MacOS/Save Channeler"
cp assets/icon.icns "$APP/Contents/Resources/AppIcon.icns"

# Bundle rclone beside the executable (Contents/MacOS) so resolveRclone finds it.
RCLONE_ARCH="$HOST_ARCH" "$ROOT/scripts/fetch-rclone.sh" macos "$APP/Contents/MacOS"

cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>Save Channeler</string>
	<key>CFBundleDisplayName</key>
	<string>Save Channeler</string>
	<key>CFBundleIdentifier</key>
	<string>com.savechanneler.app</string>
	<key>CFBundleVersion</key>
	<string>${VER}</string>
	<key>CFBundleShortVersionString</key>
	<string>${VER}</string>
	<key>CFBundleExecutable</key>
	<string>Save Channeler</string>
	<key>CFBundleIconFile</key>
	<string>AppIcon</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>LSMinimumSystemVersion</key>
	<string>12.0</string>
	<key>NSHighResolutionCapable</key>
	<true/>
</dict>
</plist>
PLIST

xattr -cr "$APP" 2>/dev/null || true
codesign --force --deep --sign - "$APP" 2>/dev/null || true

# Zip for GitHub Releases (preserve .app bundle layout)
ditto -c -k --keepParent "$APP" "$ZIP"

echo "Built $APP"
echo "Archive $ZIP"
echo "Quit any old app in the dock, then: open \"$APP\""
