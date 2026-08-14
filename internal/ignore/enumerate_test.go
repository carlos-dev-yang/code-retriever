package ignore

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFilterExcludesIgnoredSymlinkUnsupportedAndOversizeInputs(t *testing.T) {
	root := t.TempDir()
	mustFile(t, filepath.Join(root, "keep.go"), "package p")
	mustFile(t, filepath.Join(root, "skip.txt"), "text")
	mustFile(t, filepath.Join(root, "dist", "built.ts"), "export const x=1")
	mustFile(t, filepath.Join(root, "large.go"), "this is too large")
	if runtime.GOOS != "windows" {
		if err := os.Symlink(filepath.Join(root, "keep.go"), filepath.Join(root, "link.go")); err != nil {
			t.Fatal(err)
		}
	}
	paths := []byte("keep.go\x00skip.txt\x00dist/built.ts\x00large.go\x00")
	if runtime.GOOS != "windows" {
		paths = append(paths, []byte("link.go\x00")...)
	}
	got, err := Filter(root, paths, 10)
	if err != nil {
		t.Fatal(err)
	}
	reasons := map[string]string{}
	for _, v := range got {
		reasons[v.Path] = v.Exclusion
	}
	if reasons["keep.go"] != "" || reasons["dist/built.ts"] == "" || reasons["large.go"] != "oversize" {
		t.Fatalf("filter=%#v", got)
	}
	if runtime.GOOS != "windows" && reasons["link.go"] != "symlink" {
		t.Fatalf("symlink reason=%q", reasons["link.go"])
	}
}
func TestFilterRejectsRootEscape(t *testing.T) {
	if _, err := Filter(t.TempDir(), []byte("../outside.go\x00"), 10); err == nil {
		t.Fatal("root escape accepted")
	}
}
func mustFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}
