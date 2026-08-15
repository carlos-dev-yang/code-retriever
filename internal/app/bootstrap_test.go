package app

import (
	"cidx/internal/config"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenProductionAssemblyDoesNotCreateLabState(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	dim := 256
	raw := config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: 4096, TargetSegmentBytes: 2048}, Embedding: config.RawEmbedding{ServingDimensions: &dim, Request: config.RawRequest{MaxInputs: 1, MaxTotalInputBytes: 8192, TimeoutSeconds: 1}}, MCP: config.RawMCP{HardMaxInlineBytes: 1024}}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), string(data))
	application, err := Open(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()
	if _, err := os.Stat(filepath.Join(root, ".cidx", "lab")); !os.IsNotExist(err) {
		t.Fatalf("serve assembly created lab state: %v", err)
	}
}
