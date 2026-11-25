#!/bin/bash
# Builds build/ghm.app: the SwiftUI shell, the ghm CLI it talks to, Sparkle.
set -euo pipefail

cd "$(dirname "$0")/.."
CONFIG="${1:-release}"
APP="build/ghm.app"
BIN_DIR="$(swift build -c "$CONFIG" --product GhmApp --show-bin-path)"
VERSION="${GHM_VERSION:-0.8.2}"
ROOT="$(cd .. && pwd)"

swift build -c "$CONFIG" --product GhmApp

rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" "$APP/Contents/Frameworks"

cp "$BIN_DIR/GhmApp" "$APP/Contents/MacOS/ghm-app"
cp Resources/AppIcon.icns "$APP/Contents/Resources/AppIcon.icns"

# Bundle the backend so the app works when launched from Finder, where the
# user's shell PATH (and ~/.local/bin) is not available.
GHM_SRC="${GHM_SRC:-$HOME/.local/bin/ghm}"
if [ -x "$GHM_SRC" ]; then
  cp "$GHM_SRC" "$APP/Contents/MacOS/ghm"
  echo "bundled backend: $GHM_SRC"
else
  echo "warning: $GHM_SRC not found; the app will search PATH for ghm" >&2
fi

# Sparkle ships as a framework; it has to live in Contents/Frameworks and be
# found through an rpath relative to the executable.
if [ -d "$BIN_DIR/Sparkle.framework" ]; then
  cp -R "$BIN_DIR/Sparkle.framework" "$APP/Contents/Frameworks/Sparkle.framework"
else
  echo "error: Sparkle.framework not found in $BIN_DIR" >&2
  exit 1
fi
if ! otool -l "$APP/Contents/MacOS/ghm-app" | grep -q '@executable_path/../Frameworks'; then
  install_name_tool -add_rpath "@executable_path/../Frameworks" "$APP/Contents/MacOS/ghm-app"
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
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
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
    <string>26.0</string>
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
    <key>SUFeedURL</key>
    <string>${GHM_FEED_URL:-https://github.com/thaoanhhaa1/ghm/releases/latest/download/appcast.xml}</string>
    <key>SUPublicEDKey</key>
    <string>eV0UJtx24DTCO8I7ZWQiQFmdqGdLvKsVM+4xrhRzewo=</string>
    <key>SUEnableAutomaticChecks</key>
    <true/>
    <key>SUAllowsAutomaticUpdates</key>
    <true/>
    <key>SUAutomaticallyUpdate</key>
    <true/>
    <key>SUScheduledCheckInterval</key>
    <integer>86400</integer>
    <key>SUEnableInstallerLauncherService</key>
    <true/>
</dict>
</plist>
PLIST

# Sign inside-out: Sparkle's helpers first, then the app.
codesign --force --deep --sign - "$APP/Contents/Frameworks/Sparkle.framework" >/dev/null
codesign --force --deep --sign - "$APP" >/dev/null
codesign --verify --deep --strict "$APP"

echo "built $APP (version ${VERSION})"
echo "icon: $(sips -g pixelWidth Resources/AppIcon.icns 2>/dev/null | tail -1 | tr -s ' ')"

# Release helper: zip + signed appcast for Sparkle, ready for a GitHub release.
if [ "${GHM_MAKE_RELEASE:-0}" = "1" ]; then
  DIST="$ROOT/dist"
  mkdir -p "$DIST"
  ZIP="$DIST/ghm-$VERSION.zip"
  rm -f "$ZIP"
  ditto -c -k --sequesterRsrc --keepParent "$APP" "$ZIP"
  SPARKLE_BIN="${SPARKLE_BIN:-.build/artifacts/sparkle/Sparkle/bin}"
  # Stage only this archive: generate_appcast rewrites every item it finds to
  # the given URL prefix, so an older zip left in dist/ would get a v$VERSION
  # URL it does not live at.
  STAGE="$DIST/.stage"
  rm -rf "$STAGE"
  mkdir -p "$STAGE"
  cp "$ZIP" "$STAGE/"
  # CI has no login keychain, so it passes the exported private key in a file.
  generate() {
    "$SPARKLE_BIN/generate_appcast" "$STAGE" "$@" \
      --download-url-prefix "https://github.com/thaoanhhaa1/ghm/releases/download/v$VERSION/" \
      --link "https://github.com/thaoanhhaa1/ghm" \
      --maximum-deltas 0
  }
  if [ -n "${GHM_SPARKLE_KEY_FILE:-}" ]; then
    generate --ed-key-file "$GHM_SPARKLE_KEY_FILE"
  else
    generate
  fi
  mv "$STAGE/appcast.xml" "$DIST/appcast.xml"
  rm -rf "$STAGE"
  echo "release assets in $DIST:"
  ls -la "$DIST"
fi

echo "run: open $APP"
