package core

import (
	"errors"
	"net/url"
)

var (
	// ErrInvalidParent indicates a parent that does not exist or is not a folder.
	ErrInvalidParent = errors.New("invalid parent")
	// ErrCycleDetected indicates a move that would introduce a cycle.
	ErrCycleDetected = errors.New("cycle detected")
	// ErrRootImmutable indicates an operation touched the root node.
	ErrRootImmutable = errors.New("root immutable")
	// ErrInvalidNode indicates the referenced node does not exist or is wrong type.
	ErrInvalidNode = errors.New("invalid node")
	// ErrInvalidIndex indicates a provided index is out of range.
	ErrInvalidIndex = errors.New("invalid index")
	// ErrInvalidURL indicates URL validation failure.
	ErrInvalidURL = errors.New("invalid url")
)

// ValidateOps performs basic syntactic validation of a batch before hitting storage.
func ValidateOps(tree Tree, ops []Op) error {
	state := newTreeState(tree)
	for _, op := range ops {
		switch v := op.(type) {
		case AddFolderOp:
			if err := state.requireParentFolder(v.ParentID); err != nil {
				return err
			}
			if err := validateIndex(v.Index, len(state.children[v.ParentID])); err != nil {
				return err
			}
		case AddBookmarkOp:
			if err := state.requireParentFolder(v.ParentID); err != nil {
				return err
			}
			if err := validateIndex(v.Index, len(state.children[v.ParentID])); err != nil {
				return err
			}
			if err := validateURL(v.URL); err != nil {
				return err
			}
		case RenameNodeOp:
			node, err := state.requireNode(v.NodeID)
			if err != nil {
				return err
			}
			if node.ID == "root" {
				return ErrRootImmutable
			}
		case MoveNodeOp:
			node, err := state.requireNode(v.NodeID)
			if err != nil {
				return err
			}
			if node.ID == "root" {
				return ErrRootImmutable
			}
			if err := state.requireParentFolder(v.NewParentID); err != nil {
				return err
			}
			if v.NewParentID == node.ID {
				return ErrCycleDetected
			}
			if state.isDescendant(v.NewParentID, node.ID) {
				return ErrCycleDetected
			}
			if err := validateIndex(v.NewIndex, len(state.children[v.NewParentID])); err != nil {
				return err
			}
			state.moveNode(node.ID, v.NewParentID)
		case DeleteNodeOp:
			node, err := state.requireNode(v.NodeID)
			if err != nil {
				return err
			}
			if node.ID == "root" {
				return ErrRootImmutable
			}
			state.deleteNode(node.ID)
		case UpdateBookmarkOp:
			node, err := state.requireNode(v.NodeID)
			if err != nil {
				return err
			}
			if node.Kind != KindBookmark {
				return ErrInvalidNode
			}
			if v.URL != nil {
				if err := validateURL(*v.URL); err != nil {
					return err
				}
			}
		case SaveSessionOp:
			if err := state.requireParentFolder(v.ParentID); err != nil {
				return err
			}
			if err := validateIndex(v.Index, len(state.children[v.ParentID])); err != nil {
				return err
			}
			for _, tab := range v.Tabs {
				if err := validateURL(tab.URL); err != nil {
					return err
				}
			}
		default:
			return errors.New("unsupported op")
		}
	}
	return nil
}

func validateIndex(idx *int, length int) error {
	if idx == nil {
		return nil
	}
	if *idx < 0 || *idx > length {
		return ErrInvalidIndex
	}
	return nil
}

func validateURL(raw string) error {
	if raw == "" {
		return ErrInvalidURL
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}
	return nil
}

type treeState struct {
	nodes    map[string]*Node
	children map[string][]string
}

func newTreeState(tree Tree) *treeState {
	nodes := make(map[string]*Node, len(tree.Nodes))
	children := make(map[string][]string)
	for id, node := range tree.Nodes {
		n := node
		nodes[id] = &n
		children[id] = nil
	}
	for _, node := range nodes {
		if node.ParentID == nil {
			continue
		}
		parent := *node.ParentID
		children[parent] = append(children[parent], node.ID)
	}
	return &treeState{nodes: nodes, children: children}
}

func (s *treeState) requireParentFolder(id string) error {
	node, ok := s.nodes[id]
	if !ok {
		return ErrInvalidParent
	}
	if node.Kind != KindFolder {
		return ErrInvalidParent
	}
	return nil
}

func (s *treeState) requireNode(id string) (*Node, error) {
	node, ok := s.nodes[id]
	if !ok {
		return nil, ErrInvalidNode
	}
	return node, nil
}

func (s *treeState) isDescendant(candidate, ancestor string) bool {
	if candidate == ancestor {
		return true
	}
	for {
		node, ok := s.nodes[candidate]
		if !ok || node.ParentID == nil {
			return false
		}
		if *node.ParentID == ancestor {
			return true
		}
		candidate = *node.ParentID
	}
}

func (s *treeState) moveNode(id, newParent string) {
	node := s.nodes[id]
	if node.ParentID != nil {
		parentID := *node.ParentID
		children := s.children[parentID]
		for i, child := range children {
			if child == id {
				s.children[parentID] = append(children[:i], children[i+1:]...)
				break
			}
		}
	}
	node.ParentID = &newParent
	s.children[newParent] = append(s.children[newParent], id)
}

func (s *treeState) deleteNode(id string) {
	node := s.nodes[id]
	if node == nil {
		return
	}
	if node.ParentID != nil {
		parentID := *node.ParentID
		children := s.children[parentID]
		for i, child := range children {
			if child == id {
				s.children[parentID] = append(children[:i], children[i+1:]...)
				break
			}
		}
	}
	delete(s.nodes, id)
}
