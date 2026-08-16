package lab

import (
	"context"
	"database/sql"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestF32LittleEndianRoundTripAndIntegrityRejection(t *testing.T) {
	original, err := NewF32Vector([]float32{1.25, -2.5}, 2)
	if err != nil {
		t.Fatal(err)
	}
	blob := EncodeF32(original.Values)
	decoded, err := DecodeF32(blob, original.Dimensions, original.Checksum)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Values[0] != original.Values[0] || decoded.Values[1] != original.Values[1] {
		t.Fatalf("round trip mismatch: %#v", decoded.Values)
	}
	if _, err := DecodeF32(blob[:len(blob)-1], original.Dimensions, original.Checksum); err == nil {
		t.Fatal("expected length rejection")
	}
	if _, err := NewF32Vector([]float32{float32(math.NaN())}, 1); err == nil {
		t.Fatal("expected NaN rejection")
	}
}

func TestLabStoreIsolatedDocumentOnlyF32Artifact(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := OpenStore(ctx, Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	values := make([]float32, 1024)
	values[0], values[1] = 1, 2
	vector, err := NewF32Vector(values, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutDocumentSource(ctx, DocumentRaw{SourceProfile: "source-profile", InputHash: "input-hash", ResponseModel: "voyage-code-4", Vector: vector}, 1024); err != nil {
		t.Fatal(err)
	}
	stored, err := store.GetDocumentSource(ctx, "source-profile", "input-hash")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Checksum != vector.Checksum || stored.Values[1] != 2 {
		t.Fatalf("unexpected stored lab vector: %#v", stored)
	}
	if err := store.PutDocumentSource(ctx, DocumentRaw{SourceProfile: "source-profile", InputHash: "input-hash", ResponseModel: "voyage-code-4", Vector: vector}, 1024); err != nil {
		t.Fatalf("identical raw write is not idempotent: %v", err)
	}
	differentValues := append([]float32(nil), vector.Values...)
	differentValues[1] = 3
	different, err := NewF32Vector(differentValues, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutDocumentSource(ctx, DocumentRaw{SourceProfile: "source-profile", InputHash: "input-hash", ResponseModel: "voyage-code-4", Vector: different}, 1024); err == nil {
		t.Fatal("conflicting raw write replaced immutable content")
	}
	preserved, err := store.GetDocumentSource(ctx, "source-profile", "input-hash")
	if err != nil || preserved.Values[1] != 2 {
		t.Fatalf("conflicting raw write changed original: %#v %v", preserved, err)
	}
}

func TestLabIdenticalRawWritesAreConcurrentIdempotent(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	values := make([]float32, 1024)
	values[0], values[1] = 1, 2
	vector, err := NewF32Vector(values, 1024)
	if err != nil {
		t.Fatal(err)
	}
	raw := DocumentRaw{SourceProfile: "source", InputHash: "input", ResponseModel: "voyage-code-4", Vector: vector}
	var group sync.WaitGroup
	errs := make(chan error, 8)
	for range cap(errs) {
		group.Add(1)
		go func() {
			defer group.Done()
			errs <- store.PutDocumentSource(ctx, raw, 1024)
		}()
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent identical write: %v", err)
		}
	}
}

func TestFormalLabStoreUsesSeparateDerivedPathAndSchema(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := OpenStore(ctx, Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	version, err := store.InspectSchemaVersion(ctx)
	if err != nil || version != SchemaVersion {
		t.Fatalf("formal lab schema = %d, %v", version, err)
	}
	path, err := (Options{Root: root}).Path()
	if err != nil || path == "" {
		t.Fatalf("formal lab path = %q, %v", path, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("lab db permissions=%#o err=%v", info.Mode().Perm(), err)
		}
	}
}

func TestLabMigrationIsAtomicAndFailsClosed(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lab.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil || version != SchemaVersion {
		t.Fatalf("version=%d err=%v", version, err)
	}
	if err := migrate(ctx, db); err != nil {
		t.Fatalf("current schema did not validate: %v", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA user_version=7`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, db); err == nil {
		t.Fatal("newer schema accepted")
	}
	unknown := filepath.Join(t.TempDir(), "unknown.db")
	unknownDB, err := sql.Open("sqlite", unknown)
	if err != nil {
		t.Fatal(err)
	}
	defer unknownDB.Close()
	if _, err := unknownDB.ExecContext(ctx, `CREATE TABLE stray(id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, unknownDB); err == nil {
		t.Fatal("unknown unversioned schema accepted")
	}
	var unknownVersion int
	if err := unknownDB.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&unknownVersion); err != nil || unknownVersion != 0 {
		t.Fatalf("failed migration stamped version=%d err=%v", unknownVersion, err)
	}
}

func TestV4ToV5EvaluationUsageMigrationPreservesLegacyAccounting(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "v4.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`ALTER TABLE lab_meta RENAME TO lab_meta_v5`,
		labMetaV4TableStatement,
		`INSERT INTO lab_meta(id,schema_version,canonical_root,created_at,last_successful_collection_at) SELECT id,4,'legacy-root',created_at,last_successful_collection_at FROM lab_meta_v5`,
		`DROP TABLE lab_meta_v5`,
		`ALTER TABLE evaluation_runs RENAME TO evaluation_runs_v5`,
		evaluationRunsV4TableStatement,
		`INSERT INTO evaluation_runs(id,run_id,repository_identity,corpus_id,corpus_manifest_sha256,pinned_commit,content_sha256,generation,index_manifest_sha256,query_manifest_sha256,query_count,candidate_profile,source_profile,vector_space_profile,raw_document_inputs,query_provider_calls,query_tokens,artifact_reference,artifact_checksum,status,created_at) VALUES(9,'run','identity','corpus','manifest','commit','content',7,'index','queries',2,'candidate','source','space',3,4,0,'evaluations/run','checksum','complete','then')`,
		`DROP TABLE evaluation_runs_v5`,
		`PRAGMA user_version=4`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var status string
	var legacyCalls, legacyTokens int
	var logical, observed any
	var created string
	if err := db.QueryRowContext(ctx, `SELECT status,legacy_query_provider_calls,legacy_query_tokens,logical_query_operations,observed_total_tokens,created_at FROM evaluation_runs WHERE id=9`).Scan(&status, &legacyCalls, &legacyTokens, &logical, &observed, &created); err != nil {
		t.Fatal(err)
	}
	if status != "legacy" || legacyCalls != 4 || legacyTokens != 0 || logical != nil || observed != nil || created != "then" {
		t.Fatalf("v4 preservation status=%q calls=%d tokens=%d logical=%v observed=%v created=%q", status, legacyCalls, legacyTokens, logical, observed, created)
	}
}

func TestRecordEvaluationRunV5AccountingAndSQLChecks(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	zero := 0
	hash := strings.Repeat("a", 64)
	record := EvaluationRunRecord{RunID: "run", RepositoryIdentity: "identity", CorpusID: "corpus", CorpusManifestSHA256: hash, PinnedCommit: "commit", ContentSHA256: hash, Generation: 1, IndexManifestSHA256: hash, QueryManifestSHA256: hash, QueryCount: 1, CandidateProfile: "candidate", SourceProfile: "source", VectorSpaceProfile: "space", RawDocumentInputs: 1, LogicalQueryOperations: 1, ProviderAttempts: 1, ValidatedResponses: 1, FailedAttempts: 0, Retries: 0, ObservedTotalTokens: &zero, TokenObservedAttempts: 1, TokenAccountingComplete: true, ArtifactReference: "evaluations/run", ArtifactChecksum: hash}
	if _, err := store.RecordEvaluationRun(ctx, record); err != nil {
		t.Fatal(err)
	}
	var observed int
	var complete int
	if err := store.db.QueryRowContext(ctx, `SELECT observed_total_tokens,token_accounting_complete FROM evaluation_runs WHERE run_id='run'`).Scan(&observed, &complete); err != nil || observed != 0 || complete != 1 {
		t.Fatalf("v5 row observed=%d complete=%d err=%v", observed, complete, err)
	}
	record.RunID, record.ArtifactReference = "bad", "evaluations/bad"
	record.ObservedTotalTokens = nil
	if _, err := store.RecordEvaluationRun(ctx, record); err == nil {
		t.Fatal("incomplete token accounting accepted as complete")
	}
}

func TestV1MigrationPreservesRawBytesHashAndSnapshotReference(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "v1.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE lab_meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=1), canonical_root TEXT NOT NULL)`,
		`CREATE TABLE lab_inputs (canonical_input_sha256 TEXT PRIMARY KEY, canonical_bytes BLOB, snapshot_reference TEXT, CHECK(canonical_bytes IS NOT NULL OR snapshot_reference IS NOT NULL))`,
		`CREATE TABLE raw_document_embeddings (source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL CHECK(dimensions=1024), checksum INTEGER NOT NULL, blob BLOB NOT NULL, response_model TEXT NOT NULL, created_at TEXT NOT NULL, PRIMARY KEY(source_profile,canonical_input_sha256))`,
		`CREATE TABLE capture_runs (id INTEGER PRIMARY KEY, generation INTEGER NOT NULL, source_profile TEXT NOT NULL, requested_count INTEGER NOT NULL, hit_count INTEGER NOT NULL, miss_count INTEGER NOT NULL, success_count INTEGER NOT NULL, failure_count INTEGER NOT NULL)`,
		`CREATE TABLE materialization_runs (id INTEGER PRIMARY KEY, vector_space_profile TEXT NOT NULL, storage_profile TEXT NOT NULL, raw_coverage REAL NOT NULL, output_checksum TEXT NOT NULL, evaluation_run_ref TEXT)`,
		`CREATE TABLE evaluation_runs (id INTEGER PRIMARY KEY, repository_identity TEXT NOT NULL, generation INTEGER NOT NULL, query_manifest_sha256 TEXT NOT NULL, candidate_profile TEXT NOT NULL, artifact_reference TEXT NOT NULL)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	values := make([]float32, 1024)
	values[0] = 1
	vector, err := NewF32Vector(values, 1024)
	if err != nil {
		t.Fatal(err)
	}
	blob := EncodeF32(vector.Values)
	if _, err := db.ExecContext(ctx, `INSERT INTO lab_meta VALUES(1,1,'root'); INSERT INTO lab_inputs VALUES('input',NULL,'snapshot'); INSERT INTO raw_document_embeddings VALUES('source','input',1024,?,'x'||'', 'voyage-code-4','now')`, vector.Checksum); err == nil { /* placeholder below avoids multi-statement blob ambiguity */
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM raw_document_embeddings; INSERT INTO raw_document_embeddings VALUES(?,?,?,?,?,?,?)`, "source", "input", 1024, vector.Checksum, blob, "voyage-code-4", "now"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA user_version=1`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var digest, reference string
	if err := db.QueryRowContext(ctx, `SELECT vector_sha256 FROM raw_document_embeddings WHERE source_profile='source'`).Scan(&digest); err != nil || digest != VectorSHA256(blob) {
		t.Fatalf("vector hash=%q err=%v", digest, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT snapshot_reference FROM lab_inputs WHERE canonical_input_sha256='input'`).Scan(&reference); err != nil || reference != "snapshot" {
		t.Fatalf("snapshot=%q err=%v", reference, err)
	}
}

func TestV2ToV3MaterializationMigrationPreservesRawAndCaptureRows(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "v2.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`ALTER TABLE lab_meta RENAME TO lab_meta_v3`,
		`CREATE TABLE lab_meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=2), canonical_root TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), last_successful_collection_at TEXT NOT NULL DEFAULT '')`,
		`DROP TABLE lab_meta_v3`,
		`ALTER TABLE materialization_runs RENAME TO materialization_runs_v3`,
		`CREATE TABLE materialization_runs (id INTEGER PRIMARY KEY, vector_space_profile TEXT NOT NULL, storage_profile TEXT NOT NULL, raw_coverage REAL NOT NULL, output_checksum TEXT NOT NULL, evaluation_run_ref TEXT)`,
		`DROP TABLE materialization_runs_v3`,
		`DROP TABLE materialized_variants`,
		`ALTER TABLE evaluation_runs RENAME TO evaluation_runs_v3`,
		`CREATE TABLE evaluation_runs (id INTEGER PRIMARY KEY, repository_identity TEXT NOT NULL, generation INTEGER NOT NULL, query_manifest_sha256 TEXT NOT NULL, candidate_profile TEXT NOT NULL, artifact_reference TEXT NOT NULL)`,
		`DROP TABLE evaluation_runs_v3`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO lab_meta VALUES(1,2,'root','then',''); INSERT INTO raw_document_embeddings VALUES('source','input',1024,1,x'00000000','sha','voyage-code-4','voyage-code-4','','cidx-lab-f32-le-v1','then'); INSERT INTO capture_runs(generation,manifest_sha256,source_profile,planned_count,requested_count,hit_count,miss_count,success_count,failure_count,estimated_tokens,actual_tokens,status) VALUES(1,'manifest','source',1,1,0,1,0,0,0,0,'complete'); INSERT INTO materialization_runs VALUES(9,'space','storage',1.0,'sum','ref'); INSERT INTO evaluation_runs VALUES(4,'root',1,'query','profile','artifact'); PRAGMA user_version=2`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var rawCount, captureCount, variantTable, evaluationCount, materializationCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM raw_document_embeddings`).Scan(&rawCount); err != nil || rawCount != 1 {
		t.Fatalf("raw preservation=%d %v", rawCount, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM capture_runs`).Scan(&captureCount); err != nil || captureCount != 1 {
		t.Fatalf("capture preservation=%d %v", captureCount, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM evaluation_runs`).Scan(&evaluationCount); err != nil || evaluationCount != 1 {
		t.Fatalf("evaluation preservation=%d %v", evaluationCount, err)
	}
	var legacyRunID, evaluationStatus string
	if err := db.QueryRowContext(ctx, `SELECT run_id,status FROM evaluation_runs WHERE id=4`).Scan(&legacyRunID, &evaluationStatus); err != nil || legacyRunID != "legacy-4" || evaluationStatus != "legacy" {
		t.Fatalf("evaluation v4 migration=%q/%q err=%v", legacyRunID, evaluationStatus, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM materialization_runs WHERE build_id='legacy-9'`).Scan(&materializationCount); err != nil || materializationCount != 1 {
		t.Fatalf("materialization preservation=%d %v", materializationCount, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='materialized_variants'`).Scan(&variantTable); err != nil || variantTable != 1 {
		t.Fatalf("variant table=%d %v", variantTable, err)
	}
}

func TestLabRootUsesCanonicalSymlinkIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	ctx := context.Background()
	root := t.TempDir()
	link := filepath.Join(t.TempDir(), "root-link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	first, err := OpenStore(ctx, Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	_ = first.Close()
	second, err := OpenStore(ctx, Options{Root: link})
	if err != nil {
		t.Fatalf("symlink-equivalent lab root rejected: %v", err)
	}
	_ = second.Close()
}

func TestOpenExistingStoreWritableRejectsMissingStateWithoutCreatingIt(t *testing.T) {
	root := t.TempDir()
	if _, err := OpenExistingStoreWritable(context.Background(), Options{Root: root}); err == nil {
		t.Fatal("missing lab state accepted")
	}
	if _, err := os.Stat(filepath.Join(root, ".cidx")); !os.IsNotExist(err) {
		t.Fatalf("writable existing open created lab state: %v", err)
	}
}

func TestLabRejectsSymlinkedStateComponents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	ctx := context.Background()
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".cidx")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStore(ctx, Options{Root: root}); err == nil {
		t.Fatal("lab followed symlinked .cidx")
	}
	root = t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".cidx"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".cidx", "raw")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStore(ctx, Options{Root: root}); err == nil {
		t.Fatal("lab followed symlinked .cidx/raw")
	}
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".cidx", "raw"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "embeddings.db"), filepath.Join(root, ".cidx", "raw", "embeddings.db")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStore(ctx, Options{Root: root}); err == nil {
		t.Fatal("lab followed symlinked database")
	}
}
