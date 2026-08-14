package lab

import (
	"fmt"
	"path/filepath"
)

type Options struct{ Root string }

func (options Options) Path() (string, error) {
	if options.Root == "" {
		return "", fmt.Errorf("repository root is required")
	}
	root, err := canonicalRoot(options.Root)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".cidx", "lab", "embeddings.db"), nil
}

func canonicalRoot(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}
