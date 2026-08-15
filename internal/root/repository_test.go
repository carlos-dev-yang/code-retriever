package root

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitRootFindsContainingWorktreeWithoutConfig(t *testing.T) {
	ctx, worktree := context.Background(), t.TempDir()
	gitRootCommand(t, worktree, "init")
	nested := filepath.Join(worktree, "nested", "directory")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := GitRoot(ctx, nested)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(worktree)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("GitRoot()=%q want %q", got, want)
	}
	if _, err := Repository(ctx, nested); err == nil {
		t.Fatal("Repository accepted a subdirectory without preserving root semantics")
	}
}

func gitRootCommand(t *testing.T, worktree string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", worktree}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
