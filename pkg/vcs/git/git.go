package git

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	ggit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Status describes VCS status for the current profile.
type Status struct {
	Committed bool
	Pending   bool
	Hash      string
}

// Repo wraps a go-git repository rooted at the profile directory.
type Repo struct {
	root string
	repo *ggit.Repository
}

// Init opens or initializes a Git repository at root.
func Init(root string) (*Repo, error) {
	repo, err := ggit.PlainOpen(root)
	if err == ggit.ErrRepositoryNotExists {
		repo, err = ggit.PlainInit(root, false)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return &Repo{root: root, repo: repo}, nil
}

// Commit stages the provided paths and creates a commit.
func (r *Repo) Commit(ctx context.Context, message string, paths []string) (Status, error) {
	if r == nil || r.repo == nil {
		return Status{Pending: true}, fmt.Errorf("nil repo")
	}
	wt, err := r.repo.Worktree()
	if err != nil {
		return Status{Pending: true}, err
	}
	for _, p := range paths {
		rel, err := filepath.Rel(r.root, p)
		if err != nil {
			return Status{Pending: true}, err
		}
		if _, err := wt.Add(rel); err != nil {
			return Status{Pending: true}, err
		}
	}
	hash, err := wt.Commit(message, &ggit.CommitOptions{
		Author: &object.Signature{
			Name:  "s0f",
			Email: "s0f@local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return Status{Pending: true}, err
	}
	return Status{Committed: true, Hash: hash.String()}, nil
}

// Push performs a push with the default remote.
func (r *Repo) Push(ctx context.Context) error {
	if r == nil || r.repo == nil {
		return fmt.Errorf("nil repo")
	}
	return r.repo.PushContext(ctx, &ggit.PushOptions{})
}

// Pull performs a fast-forward pull from origin/main.
func (r *Repo) Pull(ctx context.Context) error {
	if r == nil || r.repo == nil {
		return fmt.Errorf("nil repo")
	}
	wt, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	return wt.PullContext(ctx, &ggit.PullOptions{RemoteName: "origin"})
}

// EnsureRemote sets a remote URL if not present.
func (r *Repo) EnsureRemote(name, url string) error {
	if r == nil || r.repo == nil {
		return fmt.Errorf("nil repo")
	}
	_, err := r.repo.Remote(name)
	if err == ggit.ErrRemoteNotFound {
		_, err = r.repo.CreateRemote(&config.RemoteConfig{Name: name, URLs: []string{url}})
	}
	return err
}

// SnapshotPath returns the snapshot.json location relative to root.
func (r *Repo) SnapshotPath() string {
	return filepath.Join(r.root, "snapshot.json")
}
