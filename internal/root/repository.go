package root

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRoot finds and canonicalizes the worktree root containing requested. It
// intentionally has no cidx configuration dependency so init can be invoked
// from any directory inside a new repository.
func GitRoot(ctx context.Context, requested string) (string, error) {
	if requested == "" {
		return "", fmt.Errorf("repository root is required")
	}
	abs, err := filepath.Abs(requested)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(canonical)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("repository root is not a directory")
	}
	command := exec.CommandContext(ctx, "git", "-C", canonical, "rev-parse", "--show-toplevel")
	out, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("validate Git repository: %w", err)
	}
	gitRoot, err := filepath.EvalSymlinks(strings.TrimSpace(string(out)))
	if err != nil {
		return "", err
	}
	return gitRoot, nil
}

// Repository canonicalizes an explicit root and proves it is the configured
// Git worktree being indexed. Git is invoked with fixed arguments only.
func Repository(ctx context.Context, requested string) (string, error) {
	if requested == "" {
		return "", fmt.Errorf("repository root is required")
	}
	abs, err := filepath.Abs(requested)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	gitRoot, err := GitRoot(ctx, canonical)
	if err != nil {
		return "", err
	}
	if gitRoot != canonical {
		return "", fmt.Errorf("requested root is not Git worktree root")
	}
	if _, err := os.Stat(filepath.Join(canonical, ".cidx", "config.json")); err != nil {
		return "", fmt.Errorf("repository config is required: %w", err)
	}
	return canonical, nil
}
