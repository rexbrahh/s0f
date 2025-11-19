package core

import "testing"

func TestValidateOps(t *testing.T) {
	tree := newTestTree()

	t.Run("invalid parent on add", func(t *testing.T) {
		err := ValidateOps(tree, []Op{AddFolderOp{ParentID: "missing", Title: "X"}})
		if err != ErrInvalidParent {
			t.Fatalf("expected ErrInvalidParent, got %v", err)
		}
	})

	t.Run("rename root", func(t *testing.T) {
		err := ValidateOps(tree, []Op{RenameNodeOp{NodeID: "root", Title: "new"}})
		if err != ErrRootImmutable {
			t.Fatalf("expected ErrRootImmutable, got %v", err)
		}
	})

	t.Run("move creates cycle", func(t *testing.T) {
		err := ValidateOps(tree, []Op{MoveNodeOp{NodeID: "fld", NewParentID: "childFolder"}})
		if err != ErrCycleDetected {
			t.Fatalf("expected ErrCycleDetected, got %v", err)
		}
	})

	t.Run("bookmark URL validation", func(t *testing.T) {
		err := ValidateOps(tree, []Op{AddBookmarkOp{ParentID: "root", Title: "bad", URL: "ftp://example.com"}})
		if err != ErrInvalidURL {
			t.Fatalf("expected ErrInvalidURL, got %v", err)
		}
	})

	t.Run("update bookmark wrong target", func(t *testing.T) {
		err := ValidateOps(tree, []Op{UpdateBookmarkOp{NodeID: "fld", Title: strPtr("x")}})
		if err != ErrInvalidNode {
			t.Fatalf("expected ErrInvalidNode, got %v", err)
		}
	})

	t.Run("update bookmark success validation", func(t *testing.T) {
		err := ValidateOps(tree, []Op{UpdateBookmarkOp{NodeID: "bookmark", URL: strPtr("https://valid.example")}})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("delete root forbidden", func(t *testing.T) {
		err := ValidateOps(tree, []Op{DeleteNodeOp{NodeID: "root"}})
		if err != ErrRootImmutable {
			t.Fatalf("expected ErrRootImmutable, got %v", err)
		}
	})
}

func newTestTree() Tree {
	root := Node{ID: "root", Kind: KindFolder, Title: "Root"}
	fldParent := "root"
	fld := Node{ID: "fld", Kind: KindFolder, Title: "Folder", ParentID: &fldParent}
	childParent := "fld"
	childFolder := Node{ID: "childFolder", Kind: KindFolder, Title: "ChildFolder", ParentID: &childParent}
	bookmarkParent := "root"
	bookmark := Node{ID: "bookmark", Kind: KindBookmark, Title: "Bookmark", URL: strPtr("https://example.com"), ParentID: &bookmarkParent}
	return Tree{
		Version: "v",
		RootID:  "root",
		Nodes: map[string]Node{
			"root":        root,
			"fld":         fld,
			"childFolder": childFolder,
			"bookmark":    bookmark,
		},
	}
}

func strPtr(s string) *string {
	return &s
}
