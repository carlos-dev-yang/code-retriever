package lab

import (
	"fmt"
	"path/filepath"
)

type Options struct {
	// StateRoot is the explicit state namespace. Root remains a compatibility
	// input for ordinary project-local development and resolves to Root/.cidx.
	StateRoot string
	Root      string
}

func (options Options) ResolvedStateRoot() (string, error) {
	root := options.StateRoot
	if root == "" && options.Root != "" {
		root = filepath.Join(options.Root, ".cidx")
	}
	if root == "" {
		return "", fmt.Errorf("state root is required")
	}
	return filepath.Abs(root)
}

func (options Options) Path() (string, error) {
	root, err := options.ResolvedStateRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "raw", "embeddings.db"), nil
}

func canonicalRoot(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}
