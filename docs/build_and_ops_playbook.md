# s0f Build, Tooling, and Ops Playbook

## 1. Repository Scaffold
```
.
├─ cmd/
│  ├─ bmd/            # daemon
│  ├─ s0f/            # CLI client
│  └─ bmd-bridge/     # Native Messaging host
├─ pkg/
│  ├─ core/
│  ├─ storage/sqlite/
│  ├─ vcs/git/
│  ├─ ipc/
│  ├─ config/
│  └─ logging/
├─ ui/
│  ├─ macos/
│  │  └─ extension/
│  └─ chrome/
├─ scripts/
│  ├─ dev-profile.sh
│  ├─ install-launchagent.sh
│  └─ install-native-messaging.sh
└─ docs/
```
- Root `go.mod` (Go 1.22+), `golangci.yml` for lint config, Swift workspace under `ui/macos`, MV3 project under `ui/chrome`.

## 2. Build and Test Commands
- `go build ./cmd/bmd`: compile the daemon for the host platform; use `GOOS/GOARCH` overrides for cross builds.
- `PROFILE=dev go run ./cmd/bmd --ipc unix`: run daemon bound to `<profile>/ipc.sock` for ad-hoc testing.
- `go build ./cmd/s0f`: build the CLI. User-facing subcommands **must** call the daemon over IPC to share validations and Git behavior; direct SQLite/Git access is reserved for `s0f diag`, `s0f db check`, etc.
- `go build ./cmd/bmd-bridge`: build the Native Messaging host that proxies Chrome extension calls to the daemon.
- `go test ./... -coverprofile=coverage.out`: run all Go tests; maintain ≥80% coverage in `pkg` packages.
- `xcodebuild -scheme BookmarkRuntime -destination 'platform=macOS'`: build the SwiftUI host. `xcodebuild test -scheme BookmarkRuntimeTests` runs UI/unit suites.
- `npm install && npm run dev --prefix ui/chrome`: run the MV3 extension with live reload targeting `bmd-bridge`. `npm run test --prefix ui/chrome` executes Playwright specs.
- `make build|test|run-daemon|s0f|bridge`: convenience targets orchestrate the Go builds and local profile wiring.

## 3. Platform Packaging & Integration
- **macOS LaunchAgent:** `scripts/install-launchagent.sh` writes `~/Library/LaunchAgents/com.example.bookmarkd.plist` launching `/usr/local/bin/bmd --profile ~/Library/Application Support/BookmarkRuntime/default`, `RunAtLoad=true`, `KeepAlive=true`, optional `GODEBUG` env.
- **systemd user unit:** `~/.config/systemd/user/bookmarkd.service` with `ExecStart=%h/bin/bmd --profile %h/.local/share/BookmarkRuntime/default`, `Restart=on-failure`, `WantedBy=default.target`.
- **Native Messaging manifest:** `~/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.example.bmd.json` referencing `/usr/local/bin/bmd-bridge` and restricting `allowed_origins` to the extension ID. Windows uses registry keys; Linux uses `~/.config/google-chrome/NativeMessagingHosts/`.
- **Swift workspace:** `ui/macos` includes the Swift IPC client package, login item helper for App Store builds, and Safari App Extension target for capture.

## 4. CLI Usage Patterns
```
s0f init --profile ./_dev_profile --name dev
s0f ping --profile ./_dev_profile
s0f tree --profile ./_dev_profile
s0f apply --profile ./_dev_profile --ops '{"ops":[{"type":"add_folder","parentId":"root","title":"Example"}]}'
s0f search --profile ./_dev_profile --query example
s0f watch --profile ./_dev_profile
s0f diag --profile ./_dev_profile
s0f vcs push --profile ./_dev_profile
s0f vcs pull --profile ./_dev_profile
```
- User-facing verbs (init/ping/tree/apply, later add/move/delete/service/vcs) always route over IPC so behavior mirrors GUI clients.
- Diagnostics (`s0f diag`, `s0f db check`, `s0f vcs retry`) may touch SQLite or Git directly but must call out that they bypass the daemon API surface.

## 5. Config (TOML) and Tokens
Profiles now use `config.toml`. Example:

```toml
profileName = "dev"

[storage]
dbPath = "/Users/alice/.s0f/dev/state.db"
journalMode = "DELETE"
synchronous = "FULL"

[vcs]
enabled = false
branch = "main"
autoPush = false

[ipc]
socketPath = "/Users/alice/.s0f/dev/ipc.sock"
requireToken = false

[logging]
level = "info"
filePath = "/Users/alice/.s0f/dev/logs/daemon.log"
fileMaxSizeMB = 10
```

- Add tables such as `[logging]` or `[vcs.remote]` as needed. `ipc.requireToken` defaults to `false`; when enabled you must configure `tokenRef` (and clients must send the shared secret before the daemon accepts a connection).
- `[logging]` controls daemon output; set `filePath` to enable log files with simple size-based rotation, or leave blank to stay on stdout.

## 6. Ops Runbook
1. **First install:** `s0f init --profile <dir>` ensures directory perms (0700), boots daemon once, creates SQLite DB + Git repo, and prints socket path/profile ID.
2. **Log rotation:** Daemon logs live under `<profile>/log/` (rotating file size/backups per config). `s0f diag` tails the last 200 log lines plus 10 Git commits.
3. **Backup & restore:** Backups are simple `git clone` of the repo folder. Restore by cloning into a new profile and running `s0f migrate` if schema version differs.
4. **Remote setup:** `s0f remote set <url>` stores remote metadata; creds go in platform secure storage. `s0f push` enforces fast-forward and prints upstream hash. `s0f pull` is manual and returns `VCS_LOCAL_CHANGES_PRESENT` if local commits exist until the user pushes or resets. No daemon-side background sync jobs exist.
5. **Incident checklist:**
   - `s0f diag` output + commit log
   - Verify socket perms remain `0700`
   - `s0f db check` to enforce foreign keys and counts
   - Inspect `s0f vcs status`; run `s0f vcs retry` if commits are pending
   - Confirm LaunchAgent/systemd service is active; restart via platform tooling if needed

## 7. CI, Tooling, and Release
- **CI expectations:** GitHub Actions builds all Go targets, runs Go tests/linters, Xcode tests, and MV3 lint/tests. Artifacts uploaded for daemon, CLI, bridge, and notarized macOS app bundles.
- **Service install automation:** `s0f service install` writes LaunchAgent/systemd definitions; `s0f service start|stop|status` proxies to platform commands.
- **Packaging:** Codesign macOS binaries, produce `.pkg` or `.dmg` for distribution, notarize apps, and generate signed Native Messaging manifests for Chrome/Edge.
- **Release checklist:** bump schema version if migrations changed, update docs, regenerate `snapshot.json` schema samples, verify `s0f` CLI help includes new verbs, and run smoke tests (`s0f init/add/list`, Swift UI reorder, Chrome capture) before tagging.

This playbook translates the original design specs into actionable build, tooling, and operational briefs so engineers and operators can stand up s0f confidently.
