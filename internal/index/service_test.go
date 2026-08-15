package index

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"cidx/internal/config"
	"cidx/internal/store"
)

func TestLiveWorktreeIndexesTrackedAndUntrackedAndPublishesAtomically(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runGit(t, root, "init")
	mustWrite(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWrite(t, filepath.Join(root, "tracked.go"), "package p\nfunc Tracked() {}\n")
	runGit(t, root, "add", "tracked.go")
	mustWrite(t, filepath.Join(root, "new.ts"), "export function NewThing() {}\n")
	mustWrite(t, filepath.Join(root, "stay.ts"), "export function StayThing() {}\n")
	mustWrite(t, filepath.Join(root, "ignored.go"), "package p\nfunc Ignored() {}\n")
	mustWrite(t, filepath.Join(root, ".gitignore"), "ignored.go\n")
	mustWrite(t, filepath.Join(root, "node_modules", "skip.ts"), "export function Skip() {}\n")
	resolved := testConfig(t)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	service := New(production)
	result, err := service.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: resolved})
	if err != nil {
		t.Fatal(err)
	}
	if result.ActivatedGeneration != 1 || result.Updated != 3 {
		t.Fatalf("first result=%#v", result)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var files, chunks, segments int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM files`).Scan(&files); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM chunks`).Scan(&chunks); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM embedding_segments`).Scan(&segments); err != nil {
		t.Fatal(err)
	}
	if files != 3 || chunks != 3 || segments != 3 {
		t.Fatalf("persisted files/chunks/segments=%d/%d/%d", files, chunks, segments)
	}
	var hits int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM chunk_fts WHERE chunk_fts MATCH 'Tracked'`).Scan(&hits); err != nil || hits != 1 {
		t.Fatalf("initial FTS=%d %v", hits, err)
	}
	second, err := service.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: resolved})
	if err != nil {
		t.Fatal(err)
	}
	if second.ActivatedGeneration != 0 || second.Reused != 3 {
		t.Fatalf("unchanged result=%#v", second)
	}
	mustWrite(t, filepath.Join(root, "tracked.go"), "package p\nfunc Changed() {}\n")
	if err := os.Remove(filepath.Join(root, "new.ts")); err != nil {
		t.Fatal(err)
	}
	third, err := service.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: resolved})
	if err != nil {
		t.Fatal(err)
	}
	if third.ActivatedGeneration != 2 || third.Updated != 1 || third.Deleted != 1 {
		t.Fatalf("incremental result=%#v", third)
	}
	var symbol string
	if err := db.QueryRowContext(ctx, `SELECT c.symbol FROM chunks c JOIN files f ON f.id=c.file_id WHERE f.path='tracked.go'`).Scan(&symbol); err != nil {
		t.Fatal(err)
	}
	if symbol != "Changed" {
		t.Fatalf("chunk not replaced: %q", symbol)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM chunk_fts WHERE chunk_fts MATCH 'Tracked'`).Scan(&hits); err != nil || hits != 0 {
		t.Fatalf("old FTS remains=%d %v", hits, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM chunk_fts WHERE chunk_fts MATCH 'Changed'`).Scan(&hits); err != nil || hits != 1 {
		t.Fatalf("new FTS=%d %v", hits, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM chunk_fts WHERE chunk_fts MATCH 'StayThing'`).Scan(&hits); err != nil || hits != 1 {
		t.Fatalf("untouched FTS=%d %v", hits, err)
	}
}

func TestDryRunPlansNewEmbeddingWithoutPublishing(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWrite(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWrite(t, filepath.Join(root, "first.go"), "package p\nfunc First() {}\n")
	runGit(t, root, "add", "first.go")
	resolved := testConfig(t)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	service := New(production)
	if _, err := service.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: resolved}); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "second.go"), "package p\nfunc Second() {}\n")
	preview, err := service.Execute(ctx, Request{Root: root, DryRun: true, Reason: ReasonManual, Config: resolved})
	if err != nil {
		t.Fatal(err)
	}
	if preview.PlannedEmbeddingsPending != 2 || preview.PlannedEmbeddingsReused != 0 {
		t.Fatalf("preview=%#v", preview)
	}
	snapshot, err := production.IndexSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Applied.ActiveGeneration != 1 || len(snapshot.Files) != 1 {
		t.Fatalf("dry run published snapshot=%#v", snapshot)
	}
}

func TestDryRunUnchangedReportsCurrentManifestAndEmbeddingPlan(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWrite(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWrite(t, filepath.Join(root, "only.go"), "package p\nfunc Only() {}\n")
	runGit(t, root, "add", "only.go")
	resolved := testConfig(t)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	service := New(production)
	indexed, err := service.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: resolved})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := service.Execute(ctx, Request{Root: root, DryRun: true, Reason: ReasonManual, Config: resolved})
	if err != nil {
		t.Fatal(err)
	}
	if preview.ManifestSHA256 != indexed.ManifestSHA256 || preview.PlannedEmbeddingsPending != 1 || preview.PlannedEmbeddingsReused != 0 {
		t.Fatalf("preview=%#v indexed=%#v", preview, indexed)
	}
	snapshot, err := production.IndexSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Applied.ActiveGeneration != 1 || snapshot.Applied.ManifestSHA256 != indexed.ManifestSHA256 {
		t.Fatalf("dry run published snapshot=%#v", snapshot)
	}
}

func TestDryRunProfileChangeUsesOnlyDesiredFutureKeys(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWrite(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWrite(t, filepath.Join(root, "only.go"), "package p\nfunc Only() {}\n")
	runGit(t, root, "add", "only.go")
	current := testConfig(t)
	production, err := store.OpenProduction(ctx, root, current)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	service := New(production)
	if _, err = service.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: current}); err != nil {
		t.Fatal(err)
	}
	dim := 512
	desired, err := config.Resolve(config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go", "typescript", "tsx"}, MaxSourceFileBytes: 1024 * 1024, TargetSegmentBytes: 1024 * 1024}, Embedding: config.RawEmbedding{ServingDimensions: &dim, Request: config.RawRequest{MaxInputs: 8, MaxTotalInputBytes: 8192, TimeoutSeconds: 1}}, MCP: config.RawMCP{HardMaxInlineBytes: 1024}})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := service.Execute(ctx, Request{Root: root, DryRun: true, Reason: ReasonManual, Config: desired})
	if err != nil {
		t.Fatal(err)
	}
	if preview.PlannedEmbeddingsPending != 1 || preview.PlannedEmbeddingsReused != 0 {
		t.Fatalf("preview=%#v", preview)
	}
}

func TestDryRunCanonicalReconciliationReplacesOldKeyWithoutPublish(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWrite(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWrite(t, filepath.Join(root, "only.go"), "package p\nfunc Only() {}\n")
	runGit(t, root, "add", "only.go")
	resolved := testConfig(t)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	service := New(production)
	if _, err = service.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: resolved}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	old := strings.Repeat("a", 64)
	if _, err := db.ExecContext(ctx, `UPDATE embedding_segments SET canonical_input_sha256=?; UPDATE meta SET canonical_text_profile='stale'`, old); err != nil {
		t.Fatal(err)
	}
	preview, err := service.Execute(ctx, Request{Root: root, DryRun: true, Reason: ReasonManual, Config: resolved})
	if err != nil {
		t.Fatal(err)
	}
	if preview.PlannedEmbeddingsPending != 1 || preview.PlannedEmbeddingsReused != 0 {
		t.Fatalf("preview=%#v", preview)
	}
	var hash string
	var generation int64
	if err := db.QueryRowContext(ctx, `SELECT canonical_input_sha256 FROM embedding_segments LIMIT 1`).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT active_generation FROM meta WHERE id=1`).Scan(&generation); err != nil {
		t.Fatal(err)
	}
	if hash != old || generation != 1 {
		t.Fatalf("dry run published hash=%s generation=%d", hash, generation)
	}
}
func TestPrepareRejectsUnsafeDiagnosticsAndCancellationLeavesGeneration(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runGit(t, root, "init")
	mustWrite(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWrite(t, filepath.Join(root, "bad.go"), "package p\nfunc Bad(\xff) {}\n")
	runGit(t, root, "add", "bad.go")
	resolved := testConfig(t)
	p, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	s := New(p)
	if _, err := s.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: resolved}); err == nil {
		t.Fatal("unsafe source accepted")
	}
	applied, err := p.AppliedProfiles(ctx)
	if err != nil || applied.ActiveGeneration != 0 {
		t.Fatalf("failed run published: %#v %v", applied, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.Execute(cancelled, Request{Root: root, Reason: ReasonManual, Config: resolved}); err == nil {
		t.Fatal("cancelled run accepted")
	}
}
func TestReadOneRejectsSymlinkedParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary")
	}
	root, outside := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(outside, "escape.go"), "package p")
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readOne(root, "linked/escape.go", 1024); err == nil {
		t.Fatal("parent symlink escape accepted")
	}
}
func TestReadOneBoundsReadAtConfiguredSourceCap(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "large.go"), "package p\n")
	if _, _, err := readOne(root, "large.go", 4); err == nil {
		t.Fatal("oversize source was read without configured cap")
	}
}
func TestGitObservationExcludesOnlyGeneratedCidxState(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.invalid")
	runGit(t, root, "config", "user.name", "Test")
	mustWrite(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWrite(t, filepath.Join(root, "a.go"), "package p\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	p, err := store.OpenProduction(ctx, root, testConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	mustWrite(t, filepath.Join(root, ".cidx", "index.lock"), "")
	_, dirty, err := gitObservation(ctx, root)
	if err != nil || dirty {
		t.Fatalf("runtime state dirty=%v err=%v", dirty, err)
	}
	mustWrite(t, filepath.Join(root, "a.go"), "package p\n// edit\n")
	_, dirty, err = gitObservation(ctx, root)
	if err != nil || !dirty {
		t.Fatalf("source dirty=%v err=%v", dirty, err)
	}
	runGit(t, root, "checkout", "--", "a.go")
	mustWrite(t, filepath.Join(root, ".cidx", "config.json"), "{\"changed\":true}")
	_, dirty, err = gitObservation(ctx, root)
	if err != nil || !dirty {
		t.Fatalf("config dirty=%v err=%v", dirty, err)
	}
	runGit(t, root, "checkout", "--", ".cidx/config.json")
	mustWrite(t, filepath.Join(root, ".cidx", "lab", "note"), "must remain visible")
	_, dirty, err = gitObservation(ctx, root)
	if err != nil || !dirty {
		t.Fatalf("lab dirty=%v err=%v", dirty, err)
	}
}
func TestProfileReconciliationRepublishesServingSegments(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runGit(t, root, "init")
	mustWrite(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWrite(t, filepath.Join(root, "a.go"), "package p\nfunc A() {}\n")
	runGit(t, root, "add", "a.go")
	first := testConfig(t)
	p, err := store.OpenProduction(ctx, root, first)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	s := New(p)
	if _, err := s.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: first}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var chunkID, segmentID, ftsID int64
	if err := db.QueryRow(`SELECT id FROM chunks`).Scan(&chunkID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM embedding_segments`).Scan(&segmentID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT rowid FROM chunk_fts`).Scan(&ftsID); err != nil {
		t.Fatal(err)
	}
	second := testConfigDim(t, 512)
	out, err := s.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: second})
	if err != nil {
		t.Fatal(err)
	}
	if out.ActivatedGeneration != 2 {
		t.Fatalf("result=%#v", out)
	}
	var profile string
	if err := db.QueryRow(`SELECT active_serving_profile FROM meta`).Scan(&profile); err != nil || profile != string(second.Profiles.Fingerprints.VectorStorage) {
		t.Fatalf("meta=%q %v", profile, err)
	}
	if err := db.QueryRow(`SELECT serving_profile FROM embedding_segments`).Scan(&profile); err != nil || profile != string(second.Profiles.Fingerprints.VectorStorage) {
		t.Fatalf("segment=%q %v", profile, err)
	}
	var afterChunk, afterSegment, afterFTS int64
	if err := db.QueryRow(`SELECT id FROM chunks`).Scan(&afterChunk); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM embedding_segments`).Scan(&afterSegment); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT rowid FROM chunk_fts`).Scan(&afterFTS); err != nil {
		t.Fatal(err)
	}
	if afterChunk != chunkID || afterSegment != segmentID || afterFTS != ftsID {
		t.Fatalf("reconciliation replaced IDs: %d/%d/%d -> %d/%d/%d", chunkID, segmentID, ftsID, afterChunk, afterSegment, afterFTS)
	}
}
func TestCanonicalOnlyReconciliationPreservesASTAndFTS(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runGit(t, root, "init")
	mustWrite(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWrite(t, filepath.Join(root, "a.go"), "package p\nfunc A() {}\n")
	runGit(t, root, "add", "a.go")
	resolved := testConfig(t)
	p, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	s := New(p)
	if _, err := s.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: resolved}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var chunkID, segmentID, ftsID int64
	if err := db.QueryRow(`SELECT c.id,s.id FROM chunks c JOIN embedding_segments s ON s.chunk_id=c.id`).Scan(&chunkID, &segmentID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT rowid FROM chunk_fts`).Scan(&ftsID); err != nil {
		t.Fatal(err)
	}
	segments, err := p.ReconciliationSegments(ctx, 1)
	if err != nil || len(segments) != 1 {
		t.Fatalf("reconciliation inputs=%#v %v", segments, err)
	}
	expected, err := canonicalStored(segments[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE meta SET canonical_text_profile=? WHERE id=1`, strings.Repeat("f", 64)); err != nil {
		t.Fatal(err)
	}
	out, err := s.Execute(ctx, Request{Root: root, Reason: ReasonManual, Config: resolved})
	if err != nil {
		t.Fatal(err)
	}
	if out.ActivatedGeneration != 2 {
		t.Fatalf("result=%#v", out)
	}
	var afterChunk, afterSegment, afterFTS int64
	var canonical, hash string
	if err := db.QueryRow(`SELECT c.id,s.id,m.canonical_text_profile,s.canonical_input_sha256 FROM chunks c JOIN embedding_segments s ON s.chunk_id=c.id JOIN meta m ON m.id=1`).Scan(&afterChunk, &afterSegment, &canonical, &hash); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT rowid FROM chunk_fts`).Scan(&afterFTS); err != nil {
		t.Fatal(err)
	}
	if afterChunk != chunkID || afterSegment != segmentID || afterFTS != ftsID {
		t.Fatal("canonical reconciliation replaced AST or FTS")
	}
	if canonical != string(resolved.Profiles.Fingerprints.CanonicalText) || hash != expected {
		t.Fatalf("canonical repair=%q hash=%q", canonical, hash)
	}
}
func testConfig(t *testing.T) config.ResolvedConfig {
	return testConfigDim(t, 256)
}
func testConfigDim(t *testing.T, d int) config.ResolvedConfig {
	t.Helper()
	v := 1
	resolved, err := config.Resolve(config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go", "typescript", "tsx"}, MaxSourceFileBytes: 1024 * 1024, TargetSegmentBytes: 1024 * 1024}, Embedding: config.RawEmbedding{ServingDimensions: &d, Request: config.RawRequest{MaxInputs: v, MaxTotalInputBytes: v, TimeoutSeconds: v}}, MCP: config.RawMCP{HardMaxInlineBytes: v}})
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}
func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
func mustWrite(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
