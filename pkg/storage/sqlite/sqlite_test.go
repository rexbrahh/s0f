package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rexliu/s0f/pkg/core"
)

func TestStoreApplyOps(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	if err := store.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}

	ops := []core.Op{
		core.AddFolderOp{ParentID: "root", Title: "Projects"},
		core.AddBookmarkOp{ParentID: "root", Title: "Example", URL: "https://example.com"},
	}
	tree, err := store.ApplyOps(ctx, ops)
	if err != nil {
		t.Fatalf("apply ops: %v", err)
	}
	if len(tree.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(tree.Nodes))
	}

	folderID, bookmarkID := findByTitle(tree, "Projects"), findByTitle(tree, "Example")
	if folderID == "" || bookmarkID == "" {
		t.Fatal("missing inserted nodes")
	}

	moveTree, err := store.ApplyOps(ctx, []core.Op{
		core.MoveNodeOp{NodeID: bookmarkID, NewParentID: folderID},
	})
	if err != nil {
		t.Fatalf("move apply: %v", err)
	}
	parent := moveTree.Nodes[bookmarkID].ParentID
	if parent == nil || *parent != folderID {
		t.Fatalf("expected bookmark parent %s, got %v", folderID, parent)
	}
}

func findByTitle(tree core.Tree, title string) string {
	for id, node := range tree.Nodes {
		if node.Title == title {
			return id
		}
	}
	return ""
}
