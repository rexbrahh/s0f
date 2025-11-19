package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rexliu/s0f/pkg/core"
)

func writeSnapshot(profileDir string, tree core.Tree) error {
	path := filepath.Join(profileDir, "snapshot.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(tree)
}
