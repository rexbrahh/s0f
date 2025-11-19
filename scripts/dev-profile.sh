#!/usr/bin/env bash
set -euo pipefail
PROFILE_DIR=${1:-./_dev_profile}

mkdir -p "$PROFILE_DIR"
chmod 700 "$PROFILE_DIR"
cat >"$PROFILE_DIR/config.toml" <<TOML
profileName = "dev"

[storage]
dbPath = "$PROFILE_DIR/state.db"
journalMode = "DELETE"
synchronous = "FULL"

[vcs]
enabled = false
branch = "main"
autoPush = false

[ipc]
socketPath = "$PROFILE_DIR/ipc.sock"
requireToken = false
TOML

echo "Initialized dev profile at $PROFILE_DIR"
