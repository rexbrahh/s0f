package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/rexliu/s0f/pkg/core"
)

const snapshotSchemaVersion = 1

type snapshotPayload struct {
	SchemaVersion int                  `json:"schemaVersion"`
	GeneratedAt   int64                `json:"generatedAt"`
	Version       string               `json:"version"`
	RootID        string               `json:"rootId"`
	Nodes         map[string]core.Node `json:"nodes"`
	Children      map[string][]string  `json:"children"`
}

func writeSnapshot(profileDir string, tree core.Tree) error {
	path := filepath.Join(profileDir, "snapshot.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	payload := snapshotPayload{
		SchemaVersion: snapshotSchemaVersion,
		GeneratedAt:   time.Now().UnixMilli(),
		Version:       tree.Version,
		RootID:        tree.RootID,
		Nodes:         tree.Nodes,
		Children:      normalizeChildren(tree),
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func normalizeChildren(tree core.Tree) map[string][]string {
	children := make(map[string][]string, len(tree.Nodes))
	for id := range tree.Nodes {
		if slice, ok := tree.Children[id]; ok {
			cp := append([]string(nil), slice...)
			children[id] = cp
		} else {
			children[id] = []string{}
		}
	}
	return children
}
