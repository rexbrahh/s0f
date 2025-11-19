package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/rexliu/s0f/pkg/core"
)

// Store owns the SQLite database for a profile.
type Store struct {
	db   *sql.DB
	path string
}

// Open initializes a SQLite database at path.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	return &Store{db: db, path: path}, nil
}

// Close releases database resources.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Init ensures pragmas and schema are configured, and root node exists.
func (s *Store) Init(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("nil store")
	}
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = DELETE;",
		"PRAGMA synchronous = FULL;",
		"PRAGMA busy_timeout = 5000;",
	}
	for _, stmt := range pragmas {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply pragma %q: %w", stmt, err)
		}
	}
	if err := s.applySchema(ctx); err != nil {
		return err
	}
	return s.ensureRoot(ctx)
}

func (s *Store) applySchema(ctx context.Context) error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);`,
		`INSERT OR IGNORE INTO meta(key,value) VALUES ('schemaVersion','1');`,
		`CREATE TABLE IF NOT EXISTS nodes (
			id TEXT PRIMARY KEY,
			parent_id TEXT REFERENCES nodes(id) ON DELETE CASCADE,
			kind TEXT NOT NULL CHECK (kind IN ('folder','bookmark')),
			title TEXT NOT NULL,
			url TEXT,
			ord REAL NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_nodes_parent_ord ON nodes(parent_id, ord);`,
		`CREATE INDEX IF NOT EXISTS idx_nodes_title_nocase ON nodes(title COLLATE NOCASE);`,
		`CREATE INDEX IF NOT EXISTS idx_nodes_url ON nodes(url);`,
	}
	for _, stmt := range ddl {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	}
	return nil
}

func (s *Store) ensureRoot(ctx context.Context) error {
	now := time.Now().UnixMilli()
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO nodes(id, parent_id, kind, title, ord, created_at, updated_at)
		VALUES ('root', NULL, 'folder', 'Root', 0, ?, ?);
	`, now, now)
	return err
}

// LoadTree returns the canonical tree snapshot.
func (s *Store) LoadTree(ctx context.Context) (core.Tree, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, parent_id, kind, title, url, ord, created_at, updated_at
		FROM nodes
		ORDER BY parent_id IS NOT NULL, parent_id, ord;
	`)
	if err != nil {
		return core.Tree{}, err
	}
	defer rows.Close()

	nodes := make(map[string]core.Node)
	children := make(map[string][]string)
	for rows.Next() {
		var (
			id       string
			parentID *string
			kind     string
			title    string
			url      *string
			ord      float64
			created  int64
			updated  int64
		)
		if err := rows.Scan(&id, &parentID, &kind, &title, &url, &ord, &created, &updated); err != nil {
			return core.Tree{}, err
		}
		node := core.Node{
			ID:        id,
			Kind:      core.NodeKind(kind),
			Title:     title,
			URL:       url,
			ParentID:  parentID,
			Ord:       ord,
			CreatedAt: created,
			UpdatedAt: updated,
		}
		nodes[id] = node
		if parentID != nil {
			children[*parentID] = append(children[*parentID], id)
		}
	}
	if err := rows.Err(); err != nil {
		return core.Tree{}, err
	}
	tree := core.Tree{
		Version:  "uninitialized",
		RootID:   "root",
		Nodes:    nodes,
		Children: children,
	}
	return tree, nil
}

// ApplyOps applies a batch atomically.
func (s *Store) ApplyOps(ctx context.Context, ops []core.Op) (core.Tree, error) {
	// TODO: implement transactional batch application.
	return core.Tree{}, errors.New("ApplyOps not implemented")
}
