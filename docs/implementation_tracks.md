# s0f Implementation Tracks and Milestones

This backlog mirrors the comprehensive design specs while breaking execution into discrete tracks with deliverables and acceptance criteria.

## Track Overview
- **Track A – Core & Storage (Go):** Domain types, validation, SQLite persistence, ordering.
- **Track B – VCS & Snapshot (Go):** Git lifecycle, commit sequencing, optional remotes.
- **Track C – IPC Server (Go):** Transport, request routing, RPC handlers, events.
- **Track D – Daemon & Services (Go):** Process management, profile lifecycle, service installers.
- **Track E – Chrome Bridge & Extension (Go + TS):** Native Messaging bridge, MV3 UI flows.
- **Track F – macOS App & Safari Extension (Swift):** Swift IPC client, SwiftUI tree UX, capture extension.
- **Track G – Tooling, CI, Packaging:** CLI, scripts, automation, release infrastructure.

## Milestone 1 – Local Core, Storage, CLI (Tracks A & D)
### A1. Core types and validation
- Deliverables: `pkg/core` containing Node/Tree/Op definitions, validation helpers, ULID generator shared across daemon and clients.
- Acceptance: Table-driven tests cover root invariants, cycle detection, URL validation, ord calculations.

### A2. SQLite schema and migrations
- Deliverables: `pkg/storage/sqlite` schema matching runtime architecture (meta + nodes, pragmas, indexes), migration runner seeding `schemaVersion=1`.
- Acceptance: `go test ./pkg/storage/...` passes; `s0f init` writes `state.db` with root node.

### A3. Fractional ordering & rebalance
- Deliverables: Ord insertion helpers plus rebalance routine triggered when sibling gaps drop below `1e-6`.
- Acceptance: Reordering 1000 siblings, persisting, restarting, and verifying order stability. Rebalance keeps ord monotonic per folder.

### A4. CLI foundation (`cmd/s0f`)
- Deliverables: `init`, `list`, `add`, `move`, `delete`, `dump-json` subcommands via IPC; `diag`/`db check` diagnostics allowed direct storage access.
- Acceptance: Manual script exercises add/move/delete and prints valid `snapshot.json` data; CLI shares validation/Git commit path with GUI clients.

## Milestone 2 – Git Integration & Snapshot (Tracks B & D)
### B1. Repo lifecycle
- Deliverables: `pkg/vcs/git` for repo init, stage, commit, status, pending retry markers.
- Acceptance: Fresh profile yields `.git`, `state.db`, `snapshot.json`, first commit with `apply ...` message.

### B2. Commit on apply
- Deliverables: Daemon transaction pipeline hooking core ops → storage → snapshot export → Git commit. Serialized critical section prevents concurrent writers.
- Acceptance: After each `s0f add/move`, Git history shows a new commit with auto-derived summary; failed commits surface `vcsStatus.pending` and retry worker restages successfully.

### B3. Optional remote push/pull
- Deliverables: `vcs_push`/`vcs_pull` RPCs enforcing fast-forward and clean working tree.
- Acceptance: Push fails with `VCS_NOT_FAST_FORWARD` when upstream ahead; pull fails with `VCS_LOCAL_CHANGES_PRESENT` when local commits exist (user resolves manually). No background sync.

## Milestone 3 – IPC Server (Track C)
### C1. Transport layer
- Deliverables: Unix socket binding, Windows named pipe binding, 4-byte LE framing, permission checks on socket dirs.
- Acceptance: Test client exchanges `ping` with length-prefixed payload; binding fails when dir perms exceed `0700` (unless override).

### C2. RPC routing & error envelopes
- Deliverables: Request/response structs with `traceId`, centralized handler registry, structured error codes.
- Acceptance: Bad parents yield `VALIDATION_FAILED` with details + traceId; logs correlate via traceId.

### C3. Method surface
- Deliverables: `get_tree`, `apply_ops`, `search`, `subscribe_events`; optional `vcs_history`, `vcs_push`, `vcs_pull` wrappers.
- Acceptance: Apply ops serialized, reads concurrent; `tree_changed` fires with `version` + IDs after commits; subscriptions receive periodic keep-alive pings.

## Milestone 4 – Chrome Bridge & Extension (Track E)
### E1. Native Messaging host
- Deliverables: `cmd/bmd-bridge` connecting stdin/stdout to daemon socket, manifest generator specifying `allowed_origins`.
- Acceptance: Local echo test: Chrome background script → bridge → daemon (ping) → response.

### E2. MV3 extension
- Deliverables: Action UI listing top-level folders, ability to add current tab or save full session by batching `apply_ops` (use `save_session`).
- Acceptance: Clicking "Add current tab" inserts bookmark, daemon emits `tree_changed`, Git commit recorded; multi-tab capture produces single batch + single commit.

## Milestone 5 – macOS App & Safari Extension (Track F)
### F1. Swift IPC client
- Deliverables: Async Swift wrapper handling socket framing, requests, event streams, token headers when required.
- Acceptance: `get_tree` and `apply_ops` round trips succeed; IPC errors surface typed Swift error enums with trace IDs.

### F2. SwiftUI tree + drag & drop
- Deliverables: Sidebar tree view showing folders/bookmarks, DnD reorder/move queuing into coalesced batches.
- Acceptance: Reorder gesture results in exactly one `apply_ops` batch and one Git commit; UI updates from events stay in sync without refetch storms.

### F3. Safari extension (optional for v1)
- Deliverables: App extension hooking Safari share button to send capture requests via host app.
- Acceptance: Selected folder receives bookmark; event surfaces in UI and `s0f list`.

## Milestone 6 – Ops, Tooling, Packaging (Track G)
### G1. Service install scripts
- Deliverables: `s0f service install|start|stop` wiring LaunchAgent plist, systemd unit, Windows service instructions.
- Acceptance: Running install + start launches daemon, verifies status, and logs path to socket.

### G2. CI and release automation
- Deliverables: GitHub Actions workflow running Go/Swift/TS tests, producing artifacts (daemon, CLI, bridge, notarized app). Release scripts package Native Messaging manifest, LaunchAgent, systemd unit.
- Acceptance: PR builds pass across platforms; tagged release uploads artifacts and documentation updates automatically.

## Cross-Cutting Acceptance
- Snapshot JSON always matches schema (ordered `children` arrays) and includes schema version/time metadata.
- Schema migrations upgrade cleanly from empty profile to latest version.
- Logs always include `traceId` for errors and commit summaries for Git actions.
- Security posture enforced: profile dirs `0700`, optional IPC token respected, no network listeners.
- CLI/GUI/extension share the same IPC endpoints, preventing divergent logic paths.

These milestones provide a readable, scoped guide that teams can execute independently while adhering to the architectural requirements in `design_doc_v1.md` and `supplementary_design_doc.md`.
