#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
PROFILE_DIR=$(mktemp -d)
trap 'rm -rf "$PROFILE_DIR"' EXIT

BIN_S0F="$ROOT/bin/s0f"
BIN_BMD="$ROOT/bin/bmd"

if [[ ! -x "$BIN_S0F" ]]; then
  (cd "$ROOT" && make build-cli >/dev/null)
fi
if [[ ! -x "$BIN_BMD" ]]; then
  (cd "$ROOT" && make build-daemon >/dev/null)
fi

"$BIN_S0F" init --profile "$PROFILE_DIR" --name smoke >/dev/null

"$BIN_BMD" --profile "$PROFILE_DIR" >/dev/null 2>&1 &
DAEMON_PID=$!
trap 'kill $DAEMON_PID >/dev/null 2>&1 || true; rm -rf "$PROFILE_DIR"' EXIT

for i in {1..20}; do
  [[ -S "$PROFILE_DIR/ipc.sock" ]] && break
  sleep 0.25
done

PAYLOAD='{"ops":[{"type":"add_folder","parentId":"root","title":"Smoke"}]}'
"$BIN_S0F" apply --profile "$PROFILE_DIR" --ops "$PAYLOAD" >/dev/null
TREE_OUTPUT=$("$BIN_S0F" tree --profile "$PROFILE_DIR")

echo "[smoke] tree snapshot:\n$TREE_OUTPUT"
kill $DAEMON_PID >/dev/null 2>&1 || true
