package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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
	if _, err := db.ExecContext(ctx, `PRAGMA user_version=5`); err != nil {
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
		info, err := os.Stat(filepath.Join(root, ".cidx", "db", "index.db"))
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("production db permissions=%#o err=%v", info.Mode().Perm(), err)
		}
		dir, err := os.Stat(filepath.Join(root, ".cidx"))
		if err != nil || dir.Mode().Perm() != 0o700 {
			t.Fatalf("production dir permissions=%#o err=%v", dir.Mode().Perm(), err)
		}
	}
}

func TestProductionV1ToV3PreservesLegacyVectorButRequiresNewLineage(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "v1.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrateProduction(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`ALTER TABLE meta RENAME TO meta_v2`,
		`CREATE TABLE meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=1), canonical_root TEXT NOT NULL, active_generation INTEGER NOT NULL CHECK(active_generation>=0), manifest_sha256 TEXT NOT NULL, index_profile TEXT NOT NULL, index_profile_json BLOB NOT NULL, canonical_text_profile TEXT NOT NULL, canonical_text_profile_json BLOB NOT NULL, source_profile TEXT NOT NULL, source_profile_json BLOB NOT NULL, vector_space_profile TEXT NOT NULL, vector_space_profile_json BLOB NOT NULL, vector_storage_profile TEXT NOT NULL, vector_storage_profile_json BLOB NOT NULL, active_serving_profile TEXT NOT NULL, index_attempted_at TEXT NOT NULL, index_succeeded_at TEXT NOT NULL, embed_attempted_at TEXT NOT NULL, embed_succeeded_at TEXT NOT NULL, observed_git_commit TEXT NOT NULL, observed_git_dirty INTEGER NOT NULL CHECK(observed_git_dirty IN (0,1)))`,
		`DROP TABLE meta_v2`,
		`DROP TABLE embedding_runs`,
		`ALTER TABLE vector_cache RENAME TO vector_cache_v2`,
		`CREATE TABLE vector_cache (serving_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL CHECK(dimensions>0), codec_id TEXT NOT NULL CHECK(codec_id IN ('cidx-binary-sign-lsb-v1','cidx-int8-symmetric-v1')), codec_version INTEGER NOT NULL CHECK(codec_version=1), blob BLOB NOT NULL CHECK(length(blob)>0), scale REAL, norm REAL, materialization_fingerprint TEXT NOT NULL, PRIMARY KEY(serving_profile, canonical_input_sha256))`,
		`DROP TABLE vector_cache_v2`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO meta VALUES(1,1,'root',1,'0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef','index',x'7b7d','canonical',x'7b7d','source',x'7b7d','space',x'7b7d','storage',x'7b7d','serving','','','','','',0); INSERT INTO vector_cache VALUES('serving','input',256,'cidx-binary-sign-lsb-v1',1,x'01',NULL,NULL,'legacy'); PRAGMA user_version=1`); err != nil {
		t.Fatal(err)
	}
	if err := migrateProduction(ctx, db); err != nil {
		t.Fatal(err)
	}
	var schemaVersion int
	var manifest, source, space, storage, serving string
	var generation int64
	if err := db.QueryRowContext(ctx, `SELECT schema_version,active_generation,manifest_sha256,source_profile,vector_space_profile,vector_storage_profile,active_serving_profile FROM meta WHERE id=1`).Scan(&schemaVersion, &generation, &manifest, &source, &space, &storage, &serving); err != nil || schemaVersion != 4 || generation != 1 || manifest != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" || source != "source" || space != "space" || storage != "storage" || serving != "serving" {
		t.Fatalf("meta preservation=%d %d %q %q %q %q %q err=%v", schemaVersion, generation, manifest, source, space, storage, serving, err)
	}
	var blob []byte
	var rowSource, rowSpace, rawSHA, at string
	if err := db.QueryRowContext(ctx, `SELECT blob,source_profile,vector_space_profile,raw_vector_sha256,materialized_at FROM vector_cache WHERE canonical_input_sha256='input'`).Scan(&blob, &rowSource, &rowSpace, &rawSHA, &at); err != nil {
		t.Fatal(err)
	}
	if len(blob) != 1 || rowSource != "" || rowSpace != "" || rawSHA != "" || at != "" {
		t.Fatalf("legacy row was rewritten or trusted: blob=%x source=%q space=%q raw=%q at=%q", blob, rowSource, rowSpace, rawSHA, at)
	}
}

func TestProductionV2ToV3PreservesHistoricalFailureAndPointers(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	resolved := testResolvedConfig(t)
	production, err := OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	active := string(resolved.Profiles.Fingerprints.VectorStorage)
	valid := validBinary(t, resolved.Embedding.ServingDimensions)
	for _, statement := range []string{`UPDATE meta SET active_generation=7,manifest_sha256='0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef'`, `INSERT INTO files(id,path,language,indexed_sha256,observed_mtime_ns,observed_size) VALUES(1,'historical.go','go','file',0,1)`, `INSERT INTO chunks(id,file_id,kind,symbol,qualified_symbol,signature,start_byte,end_byte,start_line,end_line,source_body) VALUES(1,1,'function','Historical','Historical','',0,1,1,1,x'78')`, `INSERT INTO embedding_failures(source_profile,canonical_input_sha256,classification,attempts,error_class,last_error,last_attempted_at) VALUES('source','input','terminal',4,'provider','safe','2026-01-02T03:04:05Z')`} {
		if _, err := production.Write.db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := production.Write.db.ExecContext(ctx, `INSERT INTO embedding_segments(id,chunk_id,segment_number,canonical_input_sha256,canonical_text_profile,serving_profile,display_start_byte,display_end_byte) VALUES(1,1,0,?,'canonical',?,0,1)`, testRawSHA, active); err != nil {
		t.Fatal(err)
	}
	if _, err := production.Write.db.ExecContext(ctx, `INSERT INTO vector_cache(serving_profile,canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,materialization_fingerprint,source_profile,vector_space_profile,raw_vector_sha256,materialized_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, active, testRawSHA, valid.Dimensions, valid.CodecID, valid.CodecVersion, valid.Blob, nil, nil, active, string(resolved.Profiles.Fingerprints.Source), string(resolved.Profiles.Fingerprints.VectorSpace), testRawSHA, "2026-01-02T03:04:05Z"); err != nil {
		t.Fatal(err)
	}
	if err := production.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "db", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{`ALTER TABLE meta RENAME TO meta_v4`, `CREATE TABLE meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=2), canonical_root TEXT NOT NULL, active_generation INTEGER NOT NULL CHECK(active_generation>=0), manifest_sha256 TEXT NOT NULL, index_profile TEXT NOT NULL, index_profile_json BLOB NOT NULL, canonical_text_profile TEXT NOT NULL, canonical_text_profile_json BLOB NOT NULL, source_profile TEXT NOT NULL, source_profile_json BLOB NOT NULL, vector_space_profile TEXT NOT NULL, vector_space_profile_json BLOB NOT NULL, vector_storage_profile TEXT NOT NULL, vector_storage_profile_json BLOB NOT NULL, active_serving_profile TEXT NOT NULL, index_attempted_at TEXT NOT NULL, index_succeeded_at TEXT NOT NULL, embed_attempted_at TEXT NOT NULL, embed_succeeded_at TEXT NOT NULL, observed_git_commit TEXT NOT NULL, observed_git_dirty INTEGER NOT NULL CHECK(observed_git_dirty IN (0,1)))`, `INSERT INTO meta SELECT id,2,'legacy-root',active_generation,manifest_sha256,index_profile,index_profile_json,canonical_text_profile,canonical_text_profile_json,source_profile,source_profile_json,vector_space_profile,vector_space_profile_json,vector_storage_profile,vector_storage_profile_json,active_serving_profile,index_attempted_at,index_succeeded_at,embed_attempted_at,embed_succeeded_at,observed_git_commit,observed_git_dirty FROM meta_v4`, `DROP TABLE meta_v4`, `ALTER TABLE embedding_failures RENAME TO embedding_failures_v4`, `CREATE TABLE embedding_failures (source_profile TEXT NOT NULL,canonical_input_sha256 TEXT NOT NULL,attempts INTEGER NOT NULL,error_class TEXT NOT NULL,last_error TEXT NOT NULL,last_attempted_at TEXT NOT NULL,PRIMARY KEY(source_profile,canonical_input_sha256))`, `INSERT INTO embedding_failures SELECT source_profile,canonical_input_sha256,attempts,error_class,last_error,last_attempted_at FROM embedding_failures_v4`, `DROP TABLE embedding_failures_v4`, `DROP TABLE embedding_runs`, `PRAGMA user_version=2`} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateProduction(ctx, db); err != nil {
		t.Fatal(err)
	}
	var files, vectors int
	if err := db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM files),(SELECT count(*) FROM vector_cache)`).Scan(&files, &vectors); err != nil || files != 1 || vectors != 1 {
		t.Fatalf("historical rows files=%d vectors=%d err=%v", files, vectors, err)
	}
	var version, generation, attempts int
	var manifest, classification, class, at string
	if err := db.QueryRowContext(ctx, `SELECT m.schema_version,m.active_generation,m.manifest_sha256,f.classification,f.attempts,f.error_class,f.last_attempted_at FROM meta m JOIN embedding_failures f ON 1=1`).Scan(&version, &generation, &manifest, &classification, &attempts, &class, &at); err != nil || version != 4 || generation != 7 || manifest != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" || classification != "terminal" || attempts != 4 || class != "provider" || at != "2026-01-02T03:04:05Z" {
		t.Fatalf("migration=%d/%d/%q/%q/%d/%q/%q err=%v", version, generation, manifest, classification, attempts, class, at, err)
	}
	if err := migrateProduction(ctx, db); err != nil {
		t.Fatalf("v4 reopen=%v", err)
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
	if err := os.Mkdir(filepath.Join(root, ".cidx", "db"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "index.db"), filepath.Join(root, ".cidx", "db", "index.db")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProduction(context.Background(), root, testResolvedConfig(t)); err == nil {
		t.Fatal("production followed symlinked database")
	}
}

func TestProductionAndLabSchemasStaySeparateAndValidateActiveCodec(t *testing.T) {
	ctx := context.Background()
	resolved, err := config.Resolve(config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: 10, TargetSegmentBytes: 10}, Embedding: config.RawEmbedding{ServingDimensions: intPointer(256), Request: config.RawRequest{MaxInputs: 1, MaxTotalInputBytes: 1, TimeoutSeconds: 1}}, Search: config.RawSearch{}, MCP: config.RawMCP{HardMaxInlineBytes: 1}})
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
	values := make([]float32, resolved.Embedding.ServingDimensions)
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
	insertSegment(3, testRawSHA, active)
	insertSegment(4, "invalid", active)
	insertSegment(5, "wrong-codec", active)
	insertSegment(6, "bad-blob", active)
	insertSegment(7, "stale", "old-profile")
	if err := production.RecordEmbeddingFailure(ctx, resolved, "failed", "network", "retry later"); err != nil {
		t.Fatal(err)
	}
	if err := production.RecordEmbeddingFailure(ctx, resolved, testRawSHA, "network", "old failure"); err != nil {
		t.Fatal(err)
	}
	valid := validBinary(t, resolved.Embedding.ServingDimensions)
	if err := production.UpsertServingVector(ctx, resolved, testRawSHA, string(resolved.Profiles.Fingerprints.VectorStorage), testRawSHA, valid); err != nil {
		t.Fatal(err)
	}
	invalid := valid.Clone()
	invalid.Dimensions = 512
	if _, err := production.Write.db.ExecContext(ctx, `INSERT INTO vector_cache(serving_profile,canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,materialization_fingerprint) VALUES(?,?,?,?,?,?,?,?,?)`, active, "invalid", invalid.Dimensions, invalid.CodecID, invalid.CodecVersion, invalid.Blob, nil, nil, "bad"); err != nil {
		t.Fatal(err)
	}
	wrongCodec, err := vector.EncodeInt8(makeUnitValues(resolved.Embedding.ServingDimensions))
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
	if err := production.Read.db.QueryRowContext(ctx, `SELECT count(*) FROM embedding_failures WHERE canonical_input_sha256=?`, testRawSHA).Scan(&failures); err != nil || failures != 0 {
		t.Fatalf("successful upsert did not clear failure: %d %v", failures, err)
	}
	if _, err := production.Write.db.ExecContext(ctx, `UPDATE meta SET active_serving_profile='changed' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err := production.UpsertServingVector(ctx, resolved, testRawSHA, string(resolved.Profiles.Fingerprints.VectorStorage), testRawSHA, valid); err == nil {
		t.Fatal("write proceeded after active profile changed")
	}
	if err := production.RecordEmbeddingFailure(ctx, resolved, "after-change", "network", "must fail"); err == nil {
		t.Fatal("failure write proceeded after active profile changed")
	}
}

func TestDesiredEmbeddingStatesUsesInactiveDesiredProfileCache(t *testing.T) {
	ctx := context.Background()
	current := testResolvedConfig(t)
	production, err := OpenProduction(ctx, t.TempDir(), current)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	dim := 512
	desired, err := config.Resolve(config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: 512, TargetSegmentBytes: 256}, Embedding: config.RawEmbedding{ServingDimensions: &dim, Request: config.RawRequest{MaxInputs: 1, MaxTotalInputBytes: 1, TimeoutSeconds: 1}}, MCP: config.RawMCP{HardMaxInlineBytes: 1}})
	if err != nil {
		t.Fatal(err)
	}
	stored := validBinary(t, dim)
	profile := string(desired.Profiles.Fingerprints.VectorStorage)
	if _, err := production.Write.db.ExecContext(ctx, `INSERT INTO vector_cache(serving_profile,canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,materialization_fingerprint,source_profile,vector_space_profile,raw_vector_sha256,materialized_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, profile, testRawSHA, stored.Dimensions, stored.CodecID, stored.CodecVersion, stored.Blob, nil, nil, profile, string(desired.Profiles.Fingerprints.Source), string(desired.Profiles.Fingerprints.VectorSpace), testRawSHA, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	states, err := production.DesiredEmbeddingStates(ctx, desired, []string{testRawSHA, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	if err != nil {
		t.Fatal(err)
	}
	if states[testRawSHA] != EmbeddingReady || states["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"] != EmbeddingPending {
		t.Fatalf("states=%#v", states)
	}
}

func testResolvedConfig(t *testing.T) config.ResolvedConfig {
	t.Helper()
	resolved, err := config.Resolve(config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: 512, TargetSegmentBytes: 256}, Embedding: config.RawEmbedding{ServingDimensions: intPointer(256), Request: config.RawRequest{MaxInputs: 1, MaxTotalInputBytes: 1, TimeoutSeconds: 1}}, MCP: config.RawMCP{HardMaxInlineBytes: 1}})
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
