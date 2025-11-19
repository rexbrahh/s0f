#!/usr/bin/env bash
set -euo pipefail
PROFILE=${1:-$HOME/Library/Application\ Support/BookmarkRuntime/default}
LABEL="com.example.bookmarkd"
PLIST="$HOME/Library/LaunchAgents/$LABEL.plist"

cat >"$PLIST" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>$LABEL</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/bmd</string>
    <string>--profile</string>
    <string>$PROFILE</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
</dict>
</plist>
PLIST

launchctl unload "$PLIST" 2>/dev/null || true
launchctl load "$PLIST"
echo "Installed LaunchAgent $LABEL"
