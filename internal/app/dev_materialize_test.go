package app

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cidx/internal/config"
	"cidx/internal/index"
	"cidx/internal/lab"
	"cidx/internal/store"
)

func TestDevMaterializePlansStagesPublishesAndRechecksActiveKeys(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWriteFile(t, filepath.Join(root, "a.go"), "package p\nfunc Indexed() int { return 1 }\n")
	runGit(t, root, "add", "a.go")
	resolved := materializeResolved(t)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
		t.Fatal(err)
	}
	raw, err := lab.OpenStore(ctx, lab.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	service := DevMaterialize{Production: production, Lab: raw, Resolved: resolved}
	foreignLab, err := lab.OpenStore(ctx, lab.Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (DevMaterialize{Production: production, Lab: foreignLab, Resolved: resolved}).Plan(ctx); err == nil {
		t.Fatal("foreign lab root accepted")
	}
	if err := foreignLab.Close(); err != nil {
		t.Fatal(err)
	}
	missing, err := service.Plan(ctx)
	if err != nil || missing.RequiredRaw == 0 || missing.MissingRaw != missing.RequiredRaw {
		t.Fatalf("missing plan=%#v err=%v", missing, err)
	}
	if _, err := service.Activate(ctx, missing); err == nil {
		t.Fatal("missing raw activation succeeded")
	}
	values := make([]float32, 1024)
	for i := range values {
		values[i] = float32(i + 1)
	}
	f32, err := lab.NewF32Vector(values, 1024)
	if err != nil {
		t.Fatal(err)
	}
	for _, hash := range missing.required {
		if err := raw.PutDocumentSource(ctx, lab.DocumentRaw{SourceProfile: string(resolved.Profiles.Fingerprints.Source), InputHash: hash, RequestedModel: "voyage-code-4", ResponseModel: "voyage-code-4", Vector: f32}, 1024); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := service.Plan(ctx)
	if err != nil || plan.MissingRaw != 0 {
		t.Fatalf("ready plan=%#v err=%v", plan, err)
	}
	result, err := service.Activate(ctx, plan)
	if err != nil || !result.Published || result.Staged != plan.RequiredRaw {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var blob []byte
	if err := db.QueryRowContext(ctx, `SELECT blob FROM vector_cache LIMIT 1`).Scan(&blob); err != nil || len(blob) != 32 {
		t.Fatalf("binary serving blob=%d err=%v", len(blob), err)
	}
	var definition string
	if err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='vector_cache'`).Scan(&definition); err != nil || strings.Contains(strings.ToLower(definition), "f32") {
		t.Fatalf("production f32 schema=%q err=%v", definition, err)
	}
	stale, err := service.Plan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE embedding_segments SET serving_profile='stale'`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Activate(ctx, stale); err == nil {
		t.Fatal("active-key change accepted")
	}
}

func materializeResolved(t *testing.T) config.ResolvedConfig {
	return materializeResolvedWithBatch(t, 8)
}

func materializeResolvedWithBatch(t *testing.T, maxInputs int) config.ResolvedConfig {
	t.Helper()
	dimensions := 256
	value, err := config.Resolve(config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: 4096, MaxChunkBytes: 4096, MaxSegmentInputBytes: 2048}, Embedding: config.RawEmbedding{TargetDimensions: &dimensions, Batch: config.RawBatch{MaxInputs: maxInputs, MaxInputTokens: 8192, RequestTimeoutMS: 100}}, MCP: config.RawMCP{HardMaxInlineBytes: 1024, MaxReadSpanLines: 10}})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
