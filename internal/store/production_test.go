package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cidx/internal/config"
	"cidx/internal/vector"
)

func TestProductionMigrationFailsClosedAndUsesOwnerPermissions(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "index.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateProduction(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := migrateProduction(ctx, db); err != nil {
		t.Fatalf("current production schema failed validation: %v", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA user_version=2`); err != nil {
		t.Fatal(err)
	}
	if err := migrateProduction(ctx, db); err == nil {
		t.Fatal("newer production schema accepted")
	}
	_ = db.Close()
	unknown := filepath.Join(t.TempDir(), "unknown.db")
	unknownDB, err := sql.Open("sqlite", unknown)
	if err != nil {
		t.Fatal(err)
	}
	defer unknownDB.Close()
	if _, err := unknownDB.ExecContext(ctx, `CREATE TABLE unknown_table(id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if err := migrateProduction(ctx, unknownDB); err == nil {
		t.Fatal("unknown unversioned production schema accepted")
	}
	var version int
	if err := unknownDB.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil || version != 0 {
		t.Fatalf("unknown migration version=%d err=%v", version, err)
	}

	root := t.TempDir()
	production, err := OpenProduction(ctx, root, testResolvedConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	applied, err := production.AppliedProfiles(ctx)
	if err != nil || applied.SchemaVersion != ProductionSchemaVersion || applied.ActiveGeneration != 0 || applied.ManifestSHA256 != "" || applied.ActiveServingProfile != testResolvedConfig(t).Profiles.Fingerprints.VectorStorage {
		t.Fatalf("applied profile handoff = %#v, %v", applied, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(root, ".cidx", "index.db"))
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("production db permissions=%#o err=%v", info.Mode().Perm(), err)
		}
		dir, err := os.Stat(filepath.Join(root, ".cidx"))
		if err != nil || dir.Mode().Perm() != 0o700 {
			t.Fatalf("production dir permissions=%#o err=%v", dir.Mode().Perm(), err)
		}
	}
}

func TestProductionRootUsesCanonicalSymlinkIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	ctx := context.Background()
	root := t.TempDir()
	link := filepath.Join(t.TempDir(), "root-link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	first, err := OpenProduction(ctx, root, testResolvedConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	_ = first.Close()
	second, err := OpenProduction(ctx, link, testResolvedConfig(t))
	if err != nil {
		t.Fatalf("symlink-equivalent production root rejected: %v", err)
	}
	_ = second.Close()
}

func TestProductionRejectsSymlinkedStateComponents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".cidx")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProduction(context.Background(), root, testResolvedConfig(t)); err == nil {
		t.Fatal("production followed symlinked .cidx")
	}
	root = t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".cidx"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "index.db"), filepath.Join(root, ".cidx", "index.db")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProduction(context.Background(), root, testResolvedConfig(t)); err == nil {
		t.Fatal("production followed symlinked database")
	}
}

func TestProductionAndLabSchemasStaySeparateAndValidateActiveCodec(t *testing.T) {
	ctx := context.Background()
	resolved, err := config.Resolve(config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: 10, MaxChunkBytes: 10, MaxSegmentInputBytes: 10}, Embedding: config.RawEmbedding{TargetDimensions: intPointer(256), Batch: config.RawBatch{MaxInputs: 1, MaxInputTokens: 1, RequestTimeoutMS: 1}}, Search: config.RawSearch{}, MCP: config.RawMCP{HardMaxInlineBytes: 1, MaxReadSpanLines: 1}})
	if err != nil {
		t.Fatal(err)
	}
	production, err := OpenProduction(ctx, t.TempDir(), resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	var definition string
	if err := production.Read.db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='vector_cache'`).Scan(&definition); err != nil {
		t.Fatal(err)
	}
	if containsFloatStorage(definition) {
		t.Fatalf("production vector schema leaks raw floats: %s", definition)
	}
	values := make([]float32, resolved.Embedding.TargetDimensions)
	values[0] = 1
	stored, err := vector.EncodeBinary(values)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateServingVector(resolved, stored); err != nil {
		t.Fatal(err)
	}
	stored.Dimensions = 512
	if err := ValidateServingVector(resolved, stored); err == nil {
		t.Fatal("wrong active dimensions accepted")
	}
}

func TestActiveEmbeddingStatesRejectStaleOrInvalidRowsAndSuccessClearsFailure(t *testing.T) {
	ctx := context.Background()
	resolved := testResolvedConfig(t)
	production, err := OpenProduction(ctx, t.TempDir(), resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	active := string(resolved.Profiles.Fingerprints.VectorStorage)
	if _, err := production.Write.db.ExecContext(ctx, `INSERT INTO files(id,path,language,indexed_sha256,observed_mtime_ns,observed_size) VALUES(1,'a.go','go','hash',0,1); INSERT INTO chunks(id,file_id,kind,symbol,qualified_symbol,signature,start_byte,end_byte,start_line,end_line,source_body) VALUES(1,1,'function','F','F','',0,1,1,1,x'78')`); err != nil {
		t.Fatal(err)
	}
	insertSegment := func(id int, hash, profile string) {
		if _, err := production.Write.db.ExecContext(ctx, `INSERT INTO embedding_segments(id,chunk_id,segment_number,canonical_input_sha256,canonical_text_profile,serving_profile,display_start_byte,display_end_byte) VALUES(?,?,?,?,?,?,0,1)`, id, 1, id, hash, "canonical", profile); err != nil {
			t.Fatal(err)
		}
	}
	insertSegment(1, "pending", active)
	insertSegment(2, "failed", active)
	insertSegment(3, "ready", active)
	insertSegment(4, "invalid", active)
	insertSegment(5, "wrong-codec", active)
	insertSegment(6, "bad-blob", active)
	insertSegment(7, "stale", "old-profile")
	if err := production.RecordEmbeddingFailure(ctx, resolved, "failed", "network", "retry later"); err != nil {
		t.Fatal(err)
	}
	if err := production.RecordEmbeddingFailure(ctx, resolved, "ready", "network", "old failure"); err != nil {
		t.Fatal(err)
	}
	valid := validBinary(t, resolved.Embedding.TargetDimensions)
	if err := production.UpsertServingVector(ctx, resolved, "ready", "materialization", valid); err != nil {
		t.Fatal(err)
	}
	invalid := valid.Clone()
	invalid.Dimensions = 512
	if _, err := production.Write.db.ExecContext(ctx, `INSERT INTO vector_cache(serving_profile,canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,materialization_fingerprint) VALUES(?,?,?,?,?,?,?,?,?)`, active, "invalid", invalid.Dimensions, invalid.CodecID, invalid.CodecVersion, invalid.Blob, nil, nil, "bad"); err != nil {
		t.Fatal(err)
	}
	wrongCodec, err := vector.EncodeInt8(makeUnitValues(resolved.Embedding.TargetDimensions))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := production.Write.db.ExecContext(ctx, `INSERT INTO vector_cache(serving_profile,canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,materialization_fingerprint) VALUES(?,?,?,?,?,?,?,?,?)`, active, "wrong-codec", wrongCodec.Dimensions, wrongCodec.CodecID, wrongCodec.CodecVersion, wrongCodec.Blob, wrongCodec.Scale, wrongCodec.Norm, "bad"); err != nil {
		t.Fatal(err)
	}
	if _, err := production.Write.db.ExecContext(ctx, `INSERT INTO vector_cache(serving_profile,canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,materialization_fingerprint) VALUES(?,?,?,?,?,?,?,?,?)`, active, "bad-blob", valid.Dimensions, valid.CodecID, valid.CodecVersion, valid.Blob[:1], nil, nil, "bad"); err != nil {
		t.Fatal(err)
	}
	states, err := production.ActiveSegmentStates(ctx, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 6 {
		t.Fatalf("active state count = %d", len(states))
	}
	got := map[int64]EmbeddingState{}
	for _, state := range states {
		got[state.SegmentID] = state.State
	}
	if got[1] != EmbeddingPending || got[2] != EmbeddingFailed || got[3] != EmbeddingReady || got[4] != EmbeddingPending || got[5] != EmbeddingPending || got[6] != EmbeddingPending {
		t.Fatalf("states = %#v", got)
	}
	ready, total, err := production.ActiveCoverage(ctx, resolved)
	if err != nil || ready != 1 || total != 6 {
		t.Fatalf("coverage = %d/%d %v", ready, total, err)
	}
	var failures int
	if err := production.Read.db.QueryRowContext(ctx, `SELECT count(*) FROM embedding_failures WHERE canonical_input_sha256='ready'`).Scan(&failures); err != nil || failures != 0 {
		t.Fatalf("successful upsert did not clear failure: %d %v", failures, err)
	}
	if _, err := production.Write.db.ExecContext(ctx, `UPDATE meta SET active_serving_profile='changed' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err := production.UpsertServingVector(ctx, resolved, "after-change", "materialization", valid); err == nil {
		t.Fatal("write proceeded after active profile changed")
	}
	if err := production.RecordEmbeddingFailure(ctx, resolved, "after-change", "network", "must fail"); err == nil {
		t.Fatal("failure write proceeded after active profile changed")
	}
}

func testResolvedConfig(t *testing.T) config.ResolvedConfig {
	t.Helper()
	resolved, err := config.Resolve(config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: 512, MaxChunkBytes: 512, MaxSegmentInputBytes: 256}, Embedding: config.RawEmbedding{TargetDimensions: intPointer(256), Batch: config.RawBatch{MaxInputs: 1, MaxInputTokens: 1, RequestTimeoutMS: 1}}, MCP: config.RawMCP{HardMaxInlineBytes: 1, MaxReadSpanLines: 1}})
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func validBinary(t *testing.T, dimensions int) vector.StoredVector {
	t.Helper()
	values := make([]float32, dimensions)
	values[0] = 1
	stored, err := vector.EncodeBinary(values)
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

func makeUnitValues(dimensions int) []float32 {
	values := make([]float32, dimensions)
	values[0] = 1
	return values
}
func intPointer(value int) *int { return &value }
func containsFloatStorage(value string) bool {
	return stringContains(value, "f32") || stringContains(value, "f16") || stringContains(value, "float32")
}
func stringContains(value, needle string) bool {
	for index := 0; index+len(needle) <= len(value); index++ {
		if value[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}
