# Remote Sync Workflows (Draft)

This document collects the proposed flows for configuring remotes, pushing/pulling, and handling fast-forward errors. Once stabilized it will be folded into the design doc and ops playbook.

## CLI Commands

### `s0f remote set --url <git-url> [--credential <ref>]`
- Updates `<profile>/config.toml` with `vcs.remote.url` and optional `credentialRef`.
- Enables VCS in config automatically.
- Future work: prompt for credential storage via Keychain/Credential Manager.

### `s0f remote show [--profile <dir>]`
- Prints the current remote URL/credential ref, or indicates that none is configured.

### `s0f vcs push`
- Requires a configured remote.
- Daemon ensures `origin` remote exists and rejects non-fast-forward pushes with `VCS_NOT_FAST_FORWARD`.

### `s0f vcs pull`
- Requires remote config.
- Daemon fetches, checks whether local commits are ahead, and returns `VCS_LOCAL_CHANGES_PRESENT` instead of stashing automatically.

## Open Questions
- Should `s0f remote set` prompt for credential storage (Keychain, etc.) or accept manual refs only?
- Should `vcs pull` offer an option to pull onto a clean tree when local commits exist (e.g., rebase flag)?
- Expose `s0f vcs status` for quick summary of local commit hash vs remote.

