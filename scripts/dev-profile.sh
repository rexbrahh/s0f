#!/usr/bin/env bash
set -euo pipefail
PROFILE_DIR=${1:-./_dev_profile}

mkdir -p "$PROFILE_DIR"
chmod 700 "$PROFILE_DIR"
cat >"$PROFILE_DIR/config.json" <<JSON
{
  "profileName": "dev",
  "storage": {"dbPath": "$PROFILE_DIR/state.db", "journalMode": "DELETE", "synchronous": "FULL"},
  "vcs": {"enabled": false, "branch": "main"},
  "ipc": {"socketPath": "$PROFILE_DIR/ipc.sock", "requireToken": false}
}
JSON

echo "Initialized dev profile at $PROFILE_DIR"
