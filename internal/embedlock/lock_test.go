package embedlock

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRejectsSymlinkLockLeaf(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".cidx"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "x"), filepath.Join(root, ".cidx", "embed.lock")); err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(context.Background(), root); err == nil {
		t.Fatal("symlink lock accepted")
	}
}
