package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rexliu/s0f/pkg/core"
	"github.com/rexliu/s0f/pkg/ipc"
	gitvcs "github.com/rexliu/s0f/pkg/vcs/git"
)

func (d *daemon) registerHandlers(srv *ipc.Server) {
	srv.Register("ping", pingHandler(d.logger))
	srv.Register("get_tree", d.handleGetTree)
	srv.Register("apply_ops", d.handleApplyOps)
	srv.Register("vcs_push", d.handleVCSPush)
	srv.Register("vcs_pull", d.handleVCSPull)
	srv.Register("search", d.handleSearch)
}

func (d *daemon) handleGetTree(ctx context.Context, params json.RawMessage) (any, *ipc.Error) {
	tree, err := d.store.LoadTree(ctx)
	if err != nil {
		return nil, ipc.Errorf("STORAGE_ERROR", err.Error(), nil)
	}
	return map[string]any{"tree": tree}, nil
}

func (d *daemon) handleApplyOps(ctx context.Context, params json.RawMessage) (any, *ipc.Error) {
	var payload applyOpsParams
	if err := json.Unmarshal(params, &payload); err != nil {
		return nil, ipc.Errorf("INVALID_REQUEST", "invalid params", nil)
	}
	if len(payload.Ops) == 0 {
		return nil, ipc.Errorf("INVALID_REQUEST", "ops required", nil)
	}
	tree, err := d.store.LoadTree(ctx)
	if err != nil {
		return nil, ipc.Errorf("STORAGE_ERROR", err.Error(), nil)
	}
	ops, err := payload.toCoreOps()
	if err != nil {
		return nil, ipc.Errorf("INVALID_REQUEST", err.Error(), nil)
	}
	if err := core.ValidateOps(tree, ops); err != nil {
		return nil, ipc.Errorf("VALIDATION_FAILED", err.Error(), nil)
	}
	updated, err := d.store.ApplyOps(ctx, ops)
	if err != nil {
		return nil, ipc.Errorf("STORAGE_ERROR", err.Error(), nil)
	}
	status := vcsStatus{Pending: true}
	if err := writeSnapshot(d.profileDir, updated); err != nil {
		d.logger.Printf("snapshot write failed: %v", err)
	} else if d.repo != nil {
		files := []string{d.store.Path(), filepath.Join(d.profileDir, "snapshot.json")}
		message := fmt.Sprintf("apply %d ops: %s", len(ops), payload.firstOpType())
		gstatus, err := d.repo.Commit(ctx, message, files)
		if err != nil {
			d.logger.Printf("commit failed: %v", err)
		} else {
			status = fromGitStatus(gstatus)
			if gstatus.Hash != "" {
				updated.Version = gstatus.Hash
			}
		}
	}
	resp := map[string]any{
		"tree":      updated,
		"vcsStatus": status,
	}
	d.broadcastTreeChanged(updated)
	return resp, nil
}

func (d *daemon) handleVCSPush(ctx context.Context, params json.RawMessage) (any, *ipc.Error) {
	if d.repo == nil {
		return nil, ipc.Errorf("VCS_ERROR", "git repo unavailable", nil)
	}
	if err := d.repo.Push(ctx); err != nil {
		return nil, ipc.Errorf("VCS_ERROR", err.Error(), nil)
	}
	return map[string]any{"status": "ok"}, nil
}

func (d *daemon) handleVCSPull(ctx context.Context, params json.RawMessage) (any, *ipc.Error) {
	if d.repo == nil {
		return nil, ipc.Errorf("VCS_ERROR", "git repo unavailable", nil)
	}
	if err := d.repo.Pull(ctx); err != nil {
		return nil, ipc.Errorf("VCS_ERROR", err.Error(), nil)
	}
	tree, err := d.store.LoadTree(ctx)
	if err != nil {
		return nil, ipc.Errorf("STORAGE_ERROR", err.Error(), nil)
	}
	return map[string]any{"tree": tree}, nil
}

type vcsStatus struct {
	Committed bool   `json:"committed"`
	Pending   bool   `json:"pending"`
	Hash      string `json:"hash"`
}

func fromGitStatus(status gitvcs.Status) vcsStatus {
	return vcsStatus{
		Committed: status.Committed,
		Pending:   status.Pending,
		Hash:      status.Hash,
	}
}

type applyOpsParams struct {
	Ops []rpcOp `json:"ops"`
}

func (p applyOpsParams) toCoreOps() ([]core.Op, error) {
	ops := make([]core.Op, 0, len(p.Ops))
	for _, raw := range p.Ops {
		op, err := raw.toCoreOp()
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}

type rpcOp struct {
	Type        string     `json:"type"`
	ParentID    string     `json:"parentId"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Index       *int       `json:"index"`
	NodeID      string     `json:"nodeId"`
	NewParentID string     `json:"newParentId"`
	NewIndex    *int       `json:"newIndex"`
	Recursive   bool       `json:"recursive"`
	Tabs        []core.Tab `json:"tabs"`
}

func (op rpcOp) toCoreOp() (core.Op, error) {
	switch op.Type {
	case "add_folder":
		if op.ParentID == "" {
			return nil, fmt.Errorf("parentId required for add_folder")
		}
		return core.AddFolderOp{ParentID: op.ParentID, Title: op.Title, Index: op.Index}, nil
	case "add_bookmark":
		if op.ParentID == "" || op.URL == "" {
			return nil, fmt.Errorf("parentId and url required for add_bookmark")
		}
		return core.AddBookmarkOp{ParentID: op.ParentID, Title: op.Title, URL: op.URL, Index: op.Index}, nil
	case "rename_node":
		if op.NodeID == "" {
			return nil, fmt.Errorf("nodeId required for rename_node")
		}
		return core.RenameNodeOp{NodeID: op.NodeID, Title: op.Title}, nil
	case "move_node":
		if op.NodeID == "" || op.NewParentID == "" {
			return nil, fmt.Errorf("nodeId and newParentId required for move_node")
		}
		return core.MoveNodeOp{NodeID: op.NodeID, NewParentID: op.NewParentID, NewIndex: op.NewIndex}, nil
	case "delete_node":
		if op.NodeID == "" {
			return nil, fmt.Errorf("nodeId required for delete_node")
		}
		return core.DeleteNodeOp{NodeID: op.NodeID, Recursive: op.Recursive}, nil
	case "update_bookmark":
		if op.NodeID == "" {
			return nil, fmt.Errorf("nodeId required for update_bookmark")
		}
		return core.UpdateBookmarkOp{NodeID: op.NodeID, Title: optStr(op.Title), URL: optStr(op.URL)}, nil
	case "save_session":
		if op.ParentID == "" {
			return nil, fmt.Errorf("parentId required for save_session")
		}
		return core.SaveSessionOp{ParentID: op.ParentID, Title: op.Title, Tabs: op.Tabs, Index: op.Index}, nil
	default:
		return nil, fmt.Errorf("unknown op type %s", op.Type)
	}
}

func optStr(val string) *string {
	if val == "" {
		return nil
	}
	s := val
	return &s
}

func (p applyOpsParams) firstOpType() string {
	if len(p.Ops) == 0 {
		return "unknown"
	}
	return p.Ops[0].Type
}
func (d *daemon) handleSearch(ctx context.Context, params json.RawMessage) (any, *ipc.Error) {
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, ipc.Errorf("INVALID_REQUEST", "invalid search params", nil)
	}
	if req.Limit <= 0 || req.Limit > 500 {
		req.Limit = 50
	}
	tree, err := d.store.LoadTree(ctx)
	if err != nil {
		return nil, ipc.Errorf("STORAGE_ERROR", err.Error(), nil)
	}
	query := strings.ToLower(req.Query)
	results := make([]map[string]any, 0)
	for _, node := range tree.Nodes {
		if len(results) >= req.Limit {
			break
		}
		if query == "" || strings.Contains(strings.ToLower(node.Title), query) || (node.URL != nil && strings.Contains(strings.ToLower(*node.URL), query)) {
			results = append(results, map[string]any{
				"id":    node.ID,
				"title": node.Title,
				"url":   node.URL,
				"kind":  node.Kind,
			})
		}
	}
	return map[string]any{"matches": results}, nil
}

func (d *daemon) handleSubscribeEvents(ctx context.Context) (<-chan []byte, *ipc.Error) {
	if d.eventHub == nil {
		return nil, ipc.Errorf("INTERNAL", "event hub unavailable", nil)
	}
	client := d.eventHub.register()
	go func() {
		<-ctx.Done()
		d.eventHub.unregister(client)
	}()
	return client.send, nil
}
