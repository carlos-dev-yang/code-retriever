package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cidx/internal/root"
)

type Layout struct {
	SourceRoot string
	StateRoot  string
}

func Default(ctx context.Context, requestedSource string) (Layout, error) {
	source, err := root.SourceRepository(ctx, requestedSource)
	if err != nil {
		return Layout{}, err
	}
	return Layout{SourceRoot: source, StateRoot: filepath.Join(source, ".cidx")}, nil
}

// Test resolves an explicitly isolated development workspace. Both arguments
// are relative to the controlling Git worktree, and state is confined to the
// code-owned .cidx/test/states namespace.
func Test(ctx context.Context, requestedSource, requestedState string) (Layout, error) {
	controller, err := root.GitRoot(ctx, ".")
	if err != nil {
		return Layout{}, err
	}
	source, err := controlledPath(controller, requestedSource, ".cidx/test/corpora")
	if err != nil {
		return Layout{}, fmt.Errorf("source directory: %w", err)
	}
	source, err = root.SourceRepository(ctx, source)
	if err != nil {
		return Layout{}, err
	}
	state, err := controlledPath(controller, requestedState, ".cidx/test/states")
	if err != nil {
		return Layout{}, fmt.Errorf("state directory: %w", err)
	}
	return Layout{SourceRoot: source, StateRoot: state}, nil
}

func controlledPath(controller, requested, namespace string) (string, error) {
	if requested == "" || filepath.IsAbs(requested) {
		return "", fmt.Errorf("project-relative path is required")
	}
	clean := filepath.Clean(requested)
	if clean == "." || clean == ".." || strings.HasPrefix(filepath.ToSlash(clean), "../") {
		return "", fmt.Errorf("path escapes the controlling project")
	}
	portable := filepath.ToSlash(clean)
	if !strings.HasPrefix(portable, namespace+"/") || strings.TrimPrefix(portable, namespace+"/") == "" {
		return "", fmt.Errorf("path must be below %s/<name>", namespace)
	}
	full := filepath.Join(controller, clean)
	rel, err := filepath.Rel(controller, full)
	if err != nil || rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
		return "", fmt.Errorf("path escapes the controlling project")
	}
	current := controller
	for _, component := range strings.Split(filepath.ToSlash(rel), "/") {
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("path contains a symlink")
		}
	}
	return full, nil
}
