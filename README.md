# s0f

Local-first bookmark runtime with a Go daemon, SwiftUI macOS client, and Chrome extension. This repo currently contains scaffolding for:

- Go packages (`pkg/`) that define core types, SQLite storage, Git integration, IPC, config, and logging helpers.
- Command binaries (`cmd/`) for the daemon (`bmd`), CLI (`s0f`), and Chrome Native Messaging bridge (`bmd-bridge`).
- Platform clients under `ui/` (SwiftUI macOS shell and an MV3 Chrome extension scaffold; other UIs like GTK can follow).
- Scripts for profile setup and platform service installation.
- Architecture and implementation docs in `docs/`.

Each folder includes TODOs describing the next implementation steps.

## Project Layout

- `cmd/bmd`: daemon entrypoint (loads `config.toml`, exposes IPC)
- `cmd/s0f`: CLI for profile init and sending RPCs
- `pkg/core`, `pkg/storage`, `pkg/vcs`, `pkg/ipc`: shared libraries
- `cmd/bmd-bridge` + `ui/chrome`: Native Messaging host and MV3 scaffold
- `scripts/`: helper scripts (`dev-profile.sh`, `smoke.sh`)

## Quick Start (CLI-only)

```
make build-cli build-daemon
./bin/s0f init --profile ./_dev_profile --name dev
./bin/bmd --profile ./_dev_profile &
./bin/s0f apply --profile ./_dev_profile --ops '{"ops":[{"type":"add_folder","parentId":"root","title":"Example"}]}'
./bin/s0f tree --profile ./_dev_profile
```

Run `make smoke` (or `scripts/smoke.sh`) for an end-to-end sanity check that starts the daemon, applies ops, and prints the resulting tree.
