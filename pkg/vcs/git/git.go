package git

import "context"

// Status represents Git state following a commit attempt.
type Status struct {
    Committed bool
    Pending   bool
    Hash      string
}

// Repo describes the operations needed by the daemon.
type Repo interface {
    Init(ctx context.Context) error
    Commit(ctx context.Context, message string) (Status, error)
    Push(ctx context.Context) error
    Pull(ctx context.Context) error
}

// FilesystemRepo is a placeholder implementation backed by system git.
type FilesystemRepo struct {
    Path string
}

// Init initializes a repo at Path.
func (r *FilesystemRepo) Init(ctx context.Context) error {
    _ = ctx
    // TODO: shell out to git or embed go-git.
    return nil
}

// Commit records a snapshot.
func (r *FilesystemRepo) Commit(ctx context.Context, message string) (Status, error) {
    _ = ctx
    _ = message
    // TODO: stage state.db + snapshot.json and commit with message.
    return Status{Committed: false, Pending: true}, nil
}

// Push pushes to configured origin.
func (r *FilesystemRepo) Push(ctx context.Context) error {
    _ = ctx
    return nil
}

// Pull fetches and fast-forward merges.
func (r *FilesystemRepo) Pull(ctx context.Context) error {
    _ = ctx
    return nil
}
