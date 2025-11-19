# s0f Runtime Architecture

## 1. Product Scope and Goals
- **Vision:** Local-first bookmark manager with Arc-like folder UX. All data lives on-device, with a single daemon exposing a small IPC API to platform clients. Version history is Git-based for private auditing and optional manual sync.
- **Primary Targets:** macOS first, Linux and Windows later. Clients: SwiftUI macOS app, Safari app extension, Chrome extension via Native Messaging bridge. No local HTTP servers and no background network listeners.
- **v1 Feature Set:** Nested folders/bookmarks with drag-and-drop ordering, add/rename/move/delete, save current browser window as a folder, case-insensitive search over title/URL, SQLite canonical store, Git history per profile, optional manual push/pull. Non-goals: tags, bookmark notes, full-text page search, multi-user, conflict-free multi-device merge, end-to-end encryption.

## 2. System Components and Deployment Model
- **Core library (`pkg/core`):** Domain models (Node/Tree/Op), validation, ULID generation, fractional ordering helpers.
- **Storage layer (`pkg/storage/sqlite`):** SQLite persistence with migrations, transactional apply, rebalance helpers when ord gaps shrink below `1e-6`.
- **VCS layer (`pkg/vcs/git`):** Profiles are git repos (`.git`, `state.db`, `snapshot.json`). Commits happen after every successful batch; remote push/pull are explicit user actions with fast-forward enforcement.
- **Daemon (`cmd/bmd`):** Owns profile lifecycle, coordinates storage + VCS + IPC, emits events. Deployment: LaunchAgent on macOS, systemd user service on Linux, Windows service/login app; always one daemon per active profile.
- **IPC transport (`pkg/ipc`):** 4-byte little-endian length prefix framing, UTF-8 JSON payloads. Unix sockets on macOS/Linux, named pipe on Windows (`\\.\pipe\BookmarkRuntime-<profile>`). Socket parent dir must be `0700`.
- **Browser bridge (`cmd/bmd-bridge`):** Native Messaging host bridging MV3 extension messages to daemon sockets.
- **Clients:** `ui/macos` SwiftUI app + Safari extension, `ui/chrome` MV3 extension. All clients speak the same IPC envelopes.

## 3. Domain Model and Operations
- **IDs:** ULIDs generated exclusively by the daemon for time-orderable, opaque identifiers.
- **Node structure:** `kind` (`folder`/`bookmark`), title, optional URL, `parentId`, floating `ord`, timestamps. Root node is immutable and undeletable.
- **Tree payload:** `version`, `rootId`, `nodes` map, optional `children` map for quick UI rendering.
- **Batched operations:** `add_folder`, `add_bookmark`, `rename_node`, `update_bookmark`, `move_node`, `delete_node`, `save_session`. Validation enforces existing folder parents, cycle prevention, URL must be http/https, `newIndex` clamped, duplicates allowed in v1.
- **Batch semantics:** Entire batch executes in a single SQLite transaction; on validation failure the batch rolls back. Clients must coalesce gestures (drag reorder, multi-tab capture) into one batch to keep commits meaningful.

## 4. Storage Design (SQLite)
- **Pragmas:** `foreign_keys=ON`, `journal_mode=DELETE`, `synchronous=FULL`, `busy_timeout=5000`.
- **Schema v1:** `meta` table (`schemaVersion` tracking) and `nodes` table with indexes on `(parent_id, ord)`, `title COLLATE NOCASE`, `url`.
- **Ordering:** Floating `ord`; insert between siblings uses midpoint. When gaps shrink below `1e-6`, rebalance a folder's children in one transaction. Root children are `parent_id = root`.
- **Lifecycle:** On first run create root node and seed ord values. Every successful batch: commit SQLite tx → export `snapshot.json` (schema version, generatedAt, nodes, children) → stage + commit DB + snapshot.
- **Migrations:** Go migration runner increments `meta.schemaVersion`, idempotent where possible.

## 5. Version Control Design (Git)
- **Repo layout:** `<profile>/repo/.git`, `state.db`, `snapshot.json` under the same directory.
- **Commit policy:** After each batch, auto commit with message `apply <n> ops: <first-op-kind>`. VCS status returned to clients when commits fail (pending state with retry worker).
- **Remote sync (optional):** Remotes stored in config. `s0f push`/`pull` are explicit and enforce fast-forward. Push fails with `VCS_NOT_FAST_FORWARD`; pull fails with `VCS_LOCAL_CHANGES_PRESENT` until the user pushes or resets manually. No automatic syncs.
- **Credentials:** Stored in platform secure stores (macOS Keychain, Windows Credential Manager, Linux libsecret/file 0600). Never exposed over IPC.

## 6. IPC Protocol
- **Transport:** Unix domain socket (`<profile>/ipc.sock`) or Windows named pipe. Directory perms must be `0700` to honor local security model.
- **Framing & envelopes:** Request `{ id, type, params }`, response `{ id, ok, result, error, traceId }`. Errors carry codes and structured details. `traceId` correlates logs and RPC responses.
- **Methods:** `get_tree`, `apply_ops`, `search`, `subscribe_events`, optional `vcs_history`, `vcs_push`, `vcs_pull`, plus `ping`. Apply path serializes via mutex; reads are concurrent.
- **Events:** `tree_changed` events contain `version` and `changedNodeIds` only; clients re-fetch when they need data. Long-lived subscriptions send keep-alive pings.
- **Limits:** Max payload 2 MB, server clamps `search.limit`≤500, serialized `apply_ops`, idle timeouts on subscriptions, optional shared secret header when `ipc.requireToken` is enabled.
- **Error codes:** `INVALID_REQUEST`, `UNSUPPORTED_VERSION`, `NOT_FOUND`, `INVALID_PARENT`, `CYCLE_DETECTED`, `ROOT_IMMUTABLE`, `VALIDATION_FAILED`, `OUT_OF_RANGE`, `STORAGE_ERROR`, `VCS_ERROR`, `VCS_NOT_FAST_FORWARD`, `VCS_LOCAL_CHANGES_PRESENT`, `PERMISSION_DENIED`.

## 7. Daemon Behavior and Data Flow
1. Client sends RPC (`apply_ops`).
2. Daemon validates ops, acquires single-writer lock, begins SQLite transaction.
3. Apply ops, clamp ord indexes, update timestamps.
4. Commit SQLite, refresh in-memory tree, export snapshot JSON.
5. Stage and commit Git artifacts; if commit fails, mark `vcsStatus.pending` and schedule retries.
6. Emit `tree_changed` with `version` + changed IDs; respond to client with tree and VCS status.
7. Background worker handles pending Git commits and remote pushes when user requests them.

## 8. Security, Performance, and Testing Targets
- **Security:** No network listeners; only local IPC endpoints. Socket dirs `0700`. Optional token enforcement via config. Remote creds always in secure stores. No telemetry by default.
- **Performance targets:** Cold start <200 ms for <10k nodes; `apply_ops` <50 ms for adds and <200 ms for moves; search <50 ms on 10k nodes; Git commit median <150 ms; memory footprint <50 MB at 50k nodes.
- **Testing:** Unit tests (core validation, storage apply, IPC framing), integration tests (daemon RPCs with temp profile and Git commit assertions), property tests (random ops maintain invariants), load tests (50k nodes), static checks (`go vet`, `staticcheck`, `golangci-lint`, SwiftLint), CLI coverage via `s0f` e2e harnesses.

## 9. Roadmap Snapshot
- **v1.0:** Daemon + storage + Git commits + IPC, SwiftUI macOS client, Safari/Chrome quick capture, manual push/pull, CLI `s0f` for scripts.
- **v1.1:** Import/export flows, richer event subscriptions, improved reorder batching.
- **v2.0:** Multi-device merge strategy, tags/notes, optional encryption at rest + secure sync envelopes.

This architecture summary mirrors the authoritative design specs while presenting each decision in a scoped, implementation-ready format for engineers onboarding to s0f.
