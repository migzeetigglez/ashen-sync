#!/usr/bin/env bash
# Build a macOS .app so the dock uses assets/icon.icns (bare binaries show "exec").
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f assets/icon.icns ]]; then
  echo "missing assets/icon.icns — generate it first" >&2
  exit 1
fi

mkdir -p dist
LDFLAGS="$("$ROOT/scripts/version-ldflags.sh" || true)"
LDFLAG_ARGS=()
if [[ -n "$LDFLAGS" ]]; then
  LDFLAG_ARGS=(-ldflags "$LDFLAGS")
  echo "Stamping version via ldflags: $LDFLAGS"
fi
go build "${LDFLAG_ARGS[@]}" -o /tmp/save-channeler-bin ./cmd/save-channeler

# Remove previous bundle names
rm -rf "dist/Ashen Sync.app" "dist/Save Channeler.app"

APP="dist/Save Channeler.app"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp /tmp/save-channeler-bin "$APP/Contents/MacOS/Save Channeler"
chmod +x "$APP/Contents/MacOS/Save Channeler"
cp assets/icon.icns "$APP/Contents/Resources/AppIcon.icns"

cat > "$APP/Contents/Info.plist" <<'PLIST'
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
	<string>0.1.0</string>
	<key>CFBundleShortVersionString</key>
	<string>0.1.0</string>
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
echo "Built $APP"
echo "Quit any old app in the dock, then: open \"$APP\""
