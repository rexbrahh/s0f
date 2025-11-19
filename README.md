# s0f

Local-first bookmark runtime with a Go daemon, SwiftUI macOS client, and Chrome extension. This repo currently contains scaffolding for:

- Go packages (`pkg/`) that define core types, SQLite storage, Git integration, IPC, config, and logging helpers.
- Command binaries (`cmd/`) for the daemon (`bmd`), CLI (`s0f`), and Chrome Native Messaging bridge (`bmd-bridge`).
- Platform clients under `ui/` (SwiftUI macOS shell today; additional clients like Chrome or GTK will arrive later).
- Scripts for profile setup and platform service installation.
- Architecture and implementation docs in `docs/`.

Each folder includes TODOs describing the next implementation steps.
