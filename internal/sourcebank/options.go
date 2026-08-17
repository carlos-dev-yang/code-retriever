package sourcebank

import (
	"fmt"
	"path/filepath"
)

type Options struct {
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
	return filepath.Join(root, "db", "embeddings.db"), nil
}

func (options Options) LegacyPath() (string, error) {
	root, err := options.ResolvedStateRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "raw", "embeddings.db"), nil
}
