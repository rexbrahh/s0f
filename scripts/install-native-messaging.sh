#!/usr/bin/env bash
set -euo pipefail
HOST_NAME="com.example.bmd"
TARGET=${1:-/usr/local/bin/bmd-bridge}
MANIFEST_DIR="$HOME/Library/Application Support/Google/Chrome/NativeMessagingHosts"
MANIFEST_PATH="$MANIFEST_DIR/$HOST_NAME.json"
EXTENSION_ID=${2:-exampleextensionid1234567890}

mkdir -p "$MANIFEST_DIR"
cat >"$MANIFEST_PATH" <<JSON
{
  "name": "$HOST_NAME",
  "description": "Bridge to s0f daemon",
  "path": "$TARGET",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://$EXTENSION_ID/"
  ]
}
JSON

echo "Native Messaging manifest written to $MANIFEST_PATH"
