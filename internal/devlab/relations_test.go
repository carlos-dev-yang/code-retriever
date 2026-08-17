package devlab

import (
	"os"
	"path/filepath"
	"testing"
)

func TestControlledRelationInputRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := controlledRelationInput(root, "../outside.json"); err == nil {
		t.Fatal("escape accepted")
	}
	inside := filepath.Join(root, "testdata", "x.json")
	if err := os.MkdirAll(filepath.Dir(inside), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inside, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	expected, err := filepath.EvalSymlinks(inside)
	if err != nil {
		t.Fatal(err)
	}
	if value, err := controlledRelationInput(root, "testdata/x.json"); err != nil || value != expected {
		t.Fatalf("value=%q err=%v", value, err)
	}
}
