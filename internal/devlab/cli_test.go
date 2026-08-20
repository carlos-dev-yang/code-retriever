package devlab

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevelopmentCommandsRejectPositionalArguments(t *testing.T) {
	for _, command := range [][]string{
		{"embeddings", "capture", "extra"},
		{"embeddings", "materialize", "extra"},
		{"retrieval", "evaluate", "extra"},
		{"relations", "packaging", "extra"},
	} {
		err := (CLI{}).Run(context.Background(), command, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "positional") {
			t.Fatalf("command=%v err=%v", command, err)
		}
	}
}

func TestRetrievalPreflightRejectsBeforeCreatingRuntimeState(t *testing.T) {
	root := t.TempDir()
	command := exec.Command("git", "init")
	command.Dir = root
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	manifest := filepath.Join(root, "manifest.json")
	dataset := filepath.Join(root, "dataset.json")
	if err := os.WriteFile(manifest, []byte(`{"corpus_id":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataset, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	err := (CLI{}).Run(context.Background(), []string{"retrieval", "evaluate", "--root", root, "--corpus-manifest", manifest, "--dataset", dataset}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("invalid manifest accepted")
	}
	if _, err := os.Stat(filepath.Join(root, ".cidx")); !os.IsNotExist(err) {
		t.Fatalf("preflight created runtime state: %v", err)
	}
}
