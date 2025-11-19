package git

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	ggit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// Status describes VCS status for the current profile.
type Status struct {
	Committed bool
	Pending   bool
	Hash      string
}

// StatusInfo captures local vs remote comparison details.
type StatusInfo struct {
	LocalHash        string
	RemoteHash       string
	RemoteConfigured bool
	Ahead            bool
	Behind           bool
}

// Repo wraps a go-git repository rooted at the profile directory.
type Repo struct {
	root string
	repo *ggit.Repository
}

var (
	ErrRemoteNotConfigured = errors.New("remote not configured")
	ErrNonFastForward      = errors.New("non fast-forward")
	ErrLocalCommitsPresent = errors.New("local commits present")
)

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
func (r *Repo) ensureRemote() error {
	if r == nil || r.repo == nil {
		return fmt.Errorf("nil repo")
	}
	if _, err := r.repo.Remote("origin"); err == ggit.ErrRemoteNotFound {
		return ErrRemoteNotConfigured
	} else if err != nil {
		return err
	}
	return nil
}

func (r *Repo) Push(ctx context.Context) error {
	if err := r.ensureRemote(); err != nil {
		return err
	}
	err := r.repo.PushContext(ctx, &ggit.PushOptions{RemoteName: "origin"})
	if errors.Is(err, ggit.NoErrAlreadyUpToDate) || err == nil {
		return nil
	}
	if errors.Is(err, ggit.ErrNonFastForwardUpdate) {
		return ErrNonFastForward
	}
	return err
}

// Pull performs a fast-forward pull from origin/<branch>.
func (r *Repo) Pull(ctx context.Context, branch string) error {
	if err := r.ensureRemote(); err != nil {
		return err
	}
	if err := r.repo.FetchContext(ctx, &ggit.FetchOptions{RemoteName: "origin"}); err != nil && !errors.Is(err, ggit.NoErrAlreadyUpToDate) {
		return err
	}
	_, _, ahead, _, err := r.compareRemote(branch)
	if err != nil {
		return err
	}
	if ahead {
		return ErrLocalCommitsPresent
	}
	wt, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	err = wt.PullContext(ctx, &ggit.PullOptions{RemoteName: "origin"})
	if errors.Is(err, ggit.NoErrAlreadyUpToDate) || err == nil {
		return nil
	}
	if errors.Is(err, ggit.ErrNonFastForwardUpdate) {
		return ErrNonFastForward
	}
	return err
}

// EnsureRemote sets a remote URL if not present.
func (r *Repo) EnsureRemote(name, url string) error {
	if r == nil || r.repo == nil {
		return fmt.Errorf("nil repo")
	}
	remote, err := r.repo.Remote(name)
	if err == ggit.ErrRemoteNotFound {
		_, err = r.repo.CreateRemote(&config.RemoteConfig{Name: name, URLs: []string{url}})
		return err
	} else if err != nil {
		return err
	}
	cfg := remote.Config()
	if len(cfg.URLs) == 0 || cfg.URLs[0] != url {
		if err := r.repo.DeleteRemote(name); err != nil {
			return err
		}
		_, err = r.repo.CreateRemote(&config.RemoteConfig{Name: name, URLs: []string{url}})
	}
	return err
}

// SnapshotPath returns the snapshot.json location relative to root.
func (r *Repo) SnapshotPath() string {
	return filepath.Join(r.root, "snapshot.json")
}

func (r *Repo) isAheadOfRemote(branch string) (bool, error) {
	_, _, ahead, _, err := r.compareRemote(branch)
	return ahead, err
}

// StatusInfo returns commit hashes and ahead/behind info.
func (r *Repo) Status(branch string) (StatusInfo, error) {
	var info StatusInfo
	head, remote, ahead, behind, err := r.compareRemote(branch)
	if err != nil {
		return info, err
	}
	info.LocalHash = head
	info.RemoteHash = remote
	info.Ahead = ahead
	info.Behind = behind
	info.RemoteConfigured = remote != ""
	return info, nil
}

func (r *Repo) compareRemote(branch string) (localHash string, remoteHash string, ahead bool, behind bool, err error) {
	headRef, err := r.repo.Head()
	if err != nil {
		return "", "", false, false, err
	}
	localHash = headRef.Hash().String()
	remoteRef, err := r.repo.Reference(plumbing.ReferenceName("refs/remotes/origin/"+branch), true)
	if err == plumbing.ErrReferenceNotFound {
		return localHash, "", false, false, nil
	} else if err != nil {
		return "", "", false, false, err
	}
	remoteHash = remoteRef.Hash().String()
	if headRef.Hash() == remoteRef.Hash() {
		return localHash, remoteHash, false, false, nil
	}
	headCommit, err := r.repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", "", false, false, err
	}
	remoteCommit, err := r.repo.CommitObject(remoteRef.Hash())
	if err != nil {
		return "", "", false, false, err
	}
	ahead = commitContains(headCommit, remoteRef.Hash()) == false
	behind = commitContains(remoteCommit, headRef.Hash()) == false
	return localHash, remoteHash, ahead, behind, nil
}

func commitContains(start *object.Commit, target plumbing.Hash) bool {
	if start.Hash == target {
		return true
	}
	iter := object.NewCommitPreorderIter(start, nil, nil)
	found := false
	iter.ForEach(func(c *object.Commit) error {
		if c.Hash == target {
			found = true
			return storer.ErrStop
		}
		return nil
	})
	return found
}
