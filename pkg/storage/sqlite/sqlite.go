package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/rexliu/s0f/pkg/core"
)

// Store owns the SQLite database for a profile.
type Store struct {
	db   *sql.DB
	path string
}

// Path returns the underlying SQLite file path.
func (s *Store) Path() string {
	return s.path
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return core.Tree{}, err
	}
	for _, op := range ops {
		switch v := op.(type) {
		case core.AddFolderOp:
			if err := s.applyAddFolder(ctx, tx, v); err != nil {
				tx.Rollback()
				return core.Tree{}, err
			}
		case core.AddBookmarkOp:
			if err := s.applyAddBookmark(ctx, tx, v); err != nil {
				tx.Rollback()
				return core.Tree{}, err
			}
		case core.RenameNodeOp:
			if err := s.applyRename(ctx, tx, v); err != nil {
				tx.Rollback()
				return core.Tree{}, err
			}
		case core.MoveNodeOp:
			if err := s.applyMove(ctx, tx, v); err != nil {
				tx.Rollback()
				return core.Tree{}, err
			}
		case core.DeleteNodeOp:
			if err := s.applyDelete(ctx, tx, v.NodeID); err != nil {
				tx.Rollback()
				return core.Tree{}, err
			}
		case core.UpdateBookmarkOp:
			if err := s.applyUpdateBookmark(ctx, tx, v); err != nil {
				tx.Rollback()
				return core.Tree{}, err
			}
		case core.SaveSessionOp:
			if err := s.applySaveSession(ctx, tx, v); err != nil {
				tx.Rollback()
				return core.Tree{}, err
			}
		default:
			tx.Rollback()
			return core.Tree{}, fmt.Errorf("unsupported op %T", op)
		}
	}
	if err := tx.Commit(); err != nil {
		return core.Tree{}, err
	}
	return s.LoadTree(ctx)
}

func (s *Store) applyAddFolder(ctx context.Context, tx *sql.Tx, op core.AddFolderOp) error {
	ord, err := s.calcOrd(ctx, tx, op.ParentID, op.Index)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	id := core.NewNodeID()
	_, err = tx.ExecContext(ctx, `INSERT INTO nodes(id, parent_id, kind, title, ord, created_at, updated_at) VALUES(?,?,?,?,?,?,?)`,
		id, op.ParentID, string(core.KindFolder), op.Title, ord, now, now)
	return err
}

func (s *Store) applyAddBookmark(ctx context.Context, tx *sql.Tx, op core.AddBookmarkOp) error {
	ord, err := s.calcOrd(ctx, tx, op.ParentID, op.Index)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	id := core.NewNodeID()
	_, err = tx.ExecContext(ctx, `INSERT INTO nodes(id, parent_id, kind, title, url, ord, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		id, op.ParentID, string(core.KindBookmark), op.Title, op.URL, ord, now, now)
	return err
}

func (s *Store) applyRename(ctx context.Context, tx *sql.Tx, op core.RenameNodeOp) error {
	res, err := tx.ExecContext(ctx, `UPDATE nodes SET title = ?, updated_at = ? WHERE id = ?`, op.Title, time.Now().UnixMilli(), op.NodeID)
	return wrapRowsAffected(res, err)
}

func (s *Store) applyMove(ctx context.Context, tx *sql.Tx, op core.MoveNodeOp) error {
	ord, err := s.calcOrd(ctx, tx, op.NewParentID, op.NewIndex)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE nodes SET parent_id = ?, ord = ?, updated_at = ? WHERE id = ?`, op.NewParentID, ord, time.Now().UnixMilli(), op.NodeID)
	return wrapRowsAffected(res, err)
}

func (s *Store) applyDelete(ctx context.Context, tx *sql.Tx, id string) error {
	res, err := tx.ExecContext(ctx, `DELETE FROM nodes WHERE id = ?`, id)
	return wrapRowsAffected(res, err)
}

func (s *Store) applyUpdateBookmark(ctx context.Context, tx *sql.Tx, op core.UpdateBookmarkOp) error {
	setClauses := make([]string, 0, 3)
	args := make([]any, 0, 4)
	if op.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *op.Title)
	}
	if op.URL != nil {
		setClauses = append(setClauses, "url = ?")
		args = append(args, *op.URL)
	}
	if len(setClauses) == 0 {
		return nil
	}
	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now().UnixMilli(), op.NodeID)
	stmt := fmt.Sprintf("UPDATE nodes SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	res, err := tx.ExecContext(ctx, stmt, args...)
	return wrapRowsAffected(res, err)
}

func (s *Store) applySaveSession(ctx context.Context, tx *sql.Tx, op core.SaveSessionOp) error {
	ord, err := s.calcOrd(ctx, tx, op.ParentID, op.Index)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	folderID := core.NewNodeID()
	if _, err := tx.ExecContext(ctx, `INSERT INTO nodes(id, parent_id, kind, title, ord, created_at, updated_at) VALUES(?,?,?,?,?,?,?)`,
		folderID, op.ParentID, string(core.KindFolder), op.Title, ord, now, now); err != nil {
		return err
	}
	for idx, tab := range op.Tabs {
		childOrd := float64(idx)
		if _, err := tx.ExecContext(ctx, `INSERT INTO nodes(id, parent_id, kind, title, url, ord, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?)`,
			core.NewNodeID(), folderID, string(core.KindBookmark), tab.Title, tab.URL, childOrd, now, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) calcOrd(ctx context.Context, tx *sql.Tx, parentID string, index *int) (float64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT ord FROM nodes WHERE parent_id = ? ORDER BY ord ASC`, parentID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var ords []float64
	for rows.Next() {
		var ord float64
		if err := rows.Scan(&ord); err != nil {
			return 0, err
		}
		ords = append(ords, ord)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	pos := len(ords)
	if index != nil {
		pos = *index
		if pos < 0 {
			pos = 0
		}
		if pos > len(ords) {
			pos = len(ords)
		}
	}
	switch {
	case len(ords) == 0:
		return 0, nil
	case pos == 0:
		return ords[0] - 1, nil
	case pos == len(ords):
		return ords[len(ords)-1] + 1, nil
	default:
		return (ords[pos-1] + ords[pos]) / 2, nil
	}
}

func wrapRowsAffected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no rows affected")
	}
	return nil
}
