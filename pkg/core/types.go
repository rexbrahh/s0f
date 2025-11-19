package core

// NodeKind enumerates supported node types.
type NodeKind string

const (
    KindFolder   NodeKind = "folder"
    KindBookmark NodeKind = "bookmark"
)

// Node represents a folder or bookmark in the tree.
type Node struct {
    ID        string   `json:"id"`
    Kind      NodeKind `json:"kind"`
    Title     string   `json:"title"`
    URL       *string  `json:"url,omitempty"`
    ParentID  *string  `json:"parentId"`
    Ord       float64  `json:"ord"`
    CreatedAt int64    `json:"createdAt"`
    UpdatedAt int64    `json:"updatedAt"`
}

// Tree contains a snapshot of the bookmark forest.
type Tree struct {
    Version  string            `json:"version"`
    RootID   string            `json:"rootId"`
    Nodes    map[string]Node   `json:"nodes"`
    Children map[string][]string `json:"children,omitempty"`
}

// Op represents a mutation that can be applied to the tree.
type Op interface {
    isOp()
}

// AddFolderOp creates a folder under ParentID.
type AddFolderOp struct {
    ParentID string
    Title    string
    Index    *int
}

func (AddFolderOp) isOp() {}

// AddBookmarkOp creates a bookmark under ParentID.
type AddBookmarkOp struct {
    ParentID string
    Title    string
    URL      string
    Index    *int
}

func (AddBookmarkOp) isOp() {}

// RenameNodeOp renames an existing node.
type RenameNodeOp struct {
    NodeID string
    Title  string
}

func (RenameNodeOp) isOp() {}

// MoveNodeOp moves a node to a new parent/index.
type MoveNodeOp struct {
    NodeID      string
    NewParentID string
    NewIndex    *int
}

func (MoveNodeOp) isOp() {}

// DeleteNodeOp removes a node (optionally recursive for folders).
type DeleteNodeOp struct {
    NodeID    string
    Recursive bool
}

func (DeleteNodeOp) isOp() {}

// UpdateBookmarkOp updates bookmark metadata.
type UpdateBookmarkOp struct {
    NodeID string
    Title  *string
    URL    *string
}

func (UpdateBookmarkOp) isOp() {}

// SaveSessionOp creates a folder with tab captures.
type SaveSessionOp struct {
    ParentID string
    Title    string
    Tabs     []Tab
    Index    *int
}

func (SaveSessionOp) isOp() {}

// Tab represents a browser tab capture.
type Tab struct {
    Title string `json:"title"`
    URL   string `json:"url"`
}
