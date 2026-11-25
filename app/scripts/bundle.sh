#!/bin/bash
# Builds build/ghm.app: the SwiftUI shell plus the ghm CLI it talks to.
set -euo pipefail

cd "$(dirname "$0")/.."
CONFIG="${1:-release}"
APP="build/ghm.app"
BIN_DIR="$(swift build -c "$CONFIG" --product GhmApp --show-bin-path)"
VERSION="${GHM_VERSION:-0.6.0}"

swift build -c "$CONFIG" --product GhmApp

rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

cp "$BIN_DIR/GhmApp" "$APP/Contents/MacOS/ghm-app"

# Bundle the backend so the app works when launched from Finder, where the
# user's shell PATH (and ~/.local/bin) is not available.
GHM_SRC="${GHM_SRC:-$HOME/.local/bin/ghm}"
if [ -x "$GHM_SRC" ]; then
  cp "$GHM_SRC" "$APP/Contents/MacOS/ghm"
  echo "bundled backend: $GHM_SRC"
else
  echo "warning: $GHM_SRC not found; the app will search PATH for ghm" >&2
fi

cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleExecutable</key>
    <string>ghm-app</string>
    <key>CFBundleIdentifier</key>
    <string>dev.ghm.app</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>ghm</string>
    <key>CFBundleDisplayName</key>
    <string>ghm</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>LSMinimumSystemVersion</key>
    <string>13.0</string>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.developer-tools</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSDocumentsFolderUsageDescription</key>
    <string>ghm scans these folders for git repositories so they appear in the dashboard.</string>
    <key>NSDesktopFolderUsageDescription</key>
    <string>ghm scans these folders for git repositories so they appear in the dashboard.</string>
    <key>NSHumanReadableCopyright</key>
    <string>MIT · github.com/thaoanhhaa1/ghm</string>
</dict>
</plist>
PLIST

codesign --force --deep --sign - "$APP" >/dev/null
echo "built $APP (version ${VERSION})"
echo "run: open $APP"
