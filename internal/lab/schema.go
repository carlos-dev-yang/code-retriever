package lab

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
)

const SchemaVersion = 5

// OpenStore is the formal lab-only factory. It creates a distinct path/schema
// and never opens or attaches production index.db.
func OpenStore(ctx context.Context, options Options) (*Store, error) {
	path, err := options.Path()
	if err != nil {
		return nil, err
	}
	root, err := canonicalRoot(options.Root)
	if err != nil {
		return nil, err
	}
	cidxDir, err := secureDirectoryUnderRoot(root, ".cidx")
	if err != nil {
		return nil, err
	}
	labDir, err := secureDirectoryUnderRoot(cidxDir, "lab")
	if err != nil {
		return nil, err
	}
	path = filepath.Join(labDir, "embeddings.db")
	if err := ensureOwnerDatabaseFile(path); err != nil {
		return nil, err
	}
	store, err := openLabDatabase(ctx, path)
	if err != nil {
		return nil, err
	}
	if err := migrate(ctx, store.db); err != nil {
		_ = store.Close()
		return nil, err
	}
	root, err = canonicalRoot(options.Root)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	var existingRoot string
	err = store.db.QueryRowContext(ctx, `SELECT canonical_root FROM lab_meta WHERE id=1`).Scan(&existingRoot)
	if err == sql.ErrNoRows {
		_, err = store.db.ExecContext(ctx, `INSERT INTO lab_meta(id,schema_version,canonical_root) VALUES(1,?,?)`, SchemaVersion, root)
	}
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	if existingRoot != "" && existingRoot != root {
		_ = store.Close()
		return nil, fmt.Errorf("lab database belongs to different root")
	}
	if err := ensureOwnerFile(path); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

// OpenExistingStore opens an already-initialized lab database without creating
// directories, a database file, migrations, or metadata. Evaluation planning
// uses this path so a missing raw bank remains a read-only preflight failure.
func OpenExistingStore(ctx context.Context, options Options) (*Store, error) {
	path, err := options.Path()
	if err != nil {
		return nil, err
	}
	root, err := canonicalRoot(options.Root)
	if err != nil {
		return nil, err
	}
	for _, value := range []string{filepath.Join(root, ".cidx"), filepath.Join(root, ".cidx", "lab")} {
		info, err := os.Lstat(value)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("existing lab state directory is required")
		}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
	}
	uri := &url.URL{Scheme: "file", Path: path}
	query := uri.Query()
	query.Set("mode", "ro")
	query.Add("_pragma", "foreign_keys(1)")
	uri.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &Store{db: db}
	if err := store.RequireSchemaVersion(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	var existingRoot string
	if err := db.QueryRowContext(ctx, `SELECT canonical_root FROM lab_meta WHERE id=1`).Scan(&existingRoot); err != nil || existingRoot != root {
		_ = db.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("lab database belongs to different root")
	}
	return store, nil
}

// OpenExistingStoreWritable opens and migrates an existing lab database, but
// never creates lab directories, a database file, or initial metadata. It is
// reserved for explicitly applied operations that must persist provenance.
func OpenExistingStoreWritable(ctx context.Context, options Options) (*Store, error) {
	path, err := options.Path()
	if err != nil {
		return nil, err
	}
	root, err := canonicalRoot(options.Root)
	if err != nil {
		return nil, err
	}
	for _, value := range []string{filepath.Join(root, ".cidx"), filepath.Join(root, ".cidx", "lab")} {
		info, err := os.Lstat(value)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("existing lab state directory is required")
		}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
	}
	store, err := openLabDatabase(ctx, path)
	if err != nil {
		return nil, err
	}
	if err := migrate(ctx, store.db); err != nil {
		_ = store.Close()
		return nil, err
	}
	var existingRoot string
	if err := store.db.QueryRowContext(ctx, `SELECT canonical_root FROM lab_meta WHERE id=1`).Scan(&existingRoot); err != nil || existingRoot != root {
		_ = store.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("lab database belongs to different root")
	}
	return store, nil
}

func (store *Store) InspectSchemaVersion(ctx context.Context) (int, error) {
	var version int
	err := store.db.QueryRowContext(ctx, `SELECT schema_version FROM lab_meta WHERE id=1`).Scan(&version)
	return version, err
}

func migrate(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version > SchemaVersion {
		return fmt.Errorf("lab schema version %d is newer than supported", version)
	}
	if version == SchemaVersion {
		return requireExpectedSchema(ctx, db)
	}
	if version == 4 {
		if err := migrateV4ToV5(ctx, db); err != nil {
			return err
		}
		return requireExpectedSchema(ctx, db)
	}
	if version == 3 {
		if err := migrateV3ToV4(ctx, db); err != nil {
			return err
		}
		if err := migrateV4ToV5(ctx, db); err != nil {
			return err
		}
		return requireExpectedSchema(ctx, db)
	}
	if version == 2 {
		if err := migrateV2ToV3(ctx, db); err != nil {
			return err
		}
		if err := migrateV3ToV4(ctx, db); err != nil {
			return err
		}
		if err := migrateV4ToV5(ctx, db); err != nil {
			return err
		}
		return requireExpectedSchema(ctx, db)
	}
	if version == 1 {
		if err := migrateV1ToV3(ctx, db); err != nil {
			return err
		}
		if err := migrateV3ToV4(ctx, db); err != nil {
			return err
		}
		if err := migrateV4ToV5(ctx, db); err != nil {
			return err
		}
		return requireExpectedSchema(ctx, db)
	}
	var tables int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table'`).Scan(&tables); err != nil {
		return err
	}
	if tables != 0 {
		return fmt.Errorf("refuse unknown unversioned lab database")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range labSchemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 5`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return requireExpectedSchema(ctx, db)
}

const labMetaV3TableStatement = `CREATE TABLE lab_meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=3), canonical_root TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), last_successful_collection_at TEXT NOT NULL DEFAULT '')`
const labMetaV4TableStatement = `CREATE TABLE lab_meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=4), canonical_root TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), last_successful_collection_at TEXT NOT NULL DEFAULT '')`
const labMetaTableStatement = `CREATE TABLE lab_meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=5), canonical_root TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), last_successful_collection_at TEXT NOT NULL DEFAULT '')`
const materializationRunsTableStatement = `CREATE TABLE materialization_runs (id INTEGER PRIMARY KEY, build_id TEXT NOT NULL UNIQUE, generation INTEGER NOT NULL, manifest_sha256 TEXT NOT NULL, source_profile TEXT NOT NULL, vector_space_profile TEXT NOT NULL, storage_profile TEXT NOT NULL, planned_count INTEGER NOT NULL DEFAULT 0, staged_count INTEGER NOT NULL DEFAULT 0, missing_count INTEGER NOT NULL DEFAULT 0, rejected_count INTEGER NOT NULL DEFAULT 0, raw_coverage REAL NOT NULL DEFAULT 0, output_checksum TEXT NOT NULL DEFAULT '', evaluation_run_ref TEXT, status TEXT NOT NULL CHECK(status IN ('planned','building','ready','published','aborted','failed')), started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), ended_at TEXT NOT NULL DEFAULT '', error TEXT NOT NULL DEFAULT '')`
const materializedVariantsTableStatement = `CREATE TABLE materialized_variants (materialization_id INTEGER NOT NULL REFERENCES materialization_runs(id), canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL, codec_id TEXT NOT NULL, codec_version INTEGER NOT NULL, blob BLOB NOT NULL, scale REAL, norm REAL, raw_vector_sha256 TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), PRIMARY KEY(materialization_id,canonical_input_sha256))`
const evaluationRunsV3TableStatement = `CREATE TABLE evaluation_runs (id INTEGER PRIMARY KEY, repository_identity TEXT NOT NULL, generation INTEGER NOT NULL, query_manifest_sha256 TEXT NOT NULL, candidate_profile TEXT NOT NULL, artifact_reference TEXT NOT NULL)`
const evaluationRunsV4TableStatement = `CREATE TABLE evaluation_runs (id INTEGER PRIMARY KEY, run_id TEXT NOT NULL UNIQUE, repository_identity TEXT NOT NULL, corpus_id TEXT NOT NULL, corpus_manifest_sha256 TEXT NOT NULL, pinned_commit TEXT NOT NULL, content_sha256 TEXT NOT NULL, generation INTEGER NOT NULL, index_manifest_sha256 TEXT NOT NULL, query_manifest_sha256 TEXT NOT NULL, query_count INTEGER NOT NULL, candidate_profile TEXT NOT NULL, source_profile TEXT NOT NULL, vector_space_profile TEXT NOT NULL, raw_document_inputs INTEGER NOT NULL, query_provider_calls INTEGER NOT NULL, query_tokens INTEGER NOT NULL, artifact_reference TEXT NOT NULL UNIQUE, artifact_checksum TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('complete','failed','legacy')), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')))`
const evaluationRunsTableStatement = `CREATE TABLE evaluation_runs (id INTEGER PRIMARY KEY, run_id TEXT NOT NULL UNIQUE, repository_identity TEXT NOT NULL, corpus_id TEXT NOT NULL, corpus_manifest_sha256 TEXT NOT NULL, pinned_commit TEXT NOT NULL, content_sha256 TEXT NOT NULL, generation INTEGER NOT NULL, index_manifest_sha256 TEXT NOT NULL, query_manifest_sha256 TEXT NOT NULL, query_count INTEGER NOT NULL, candidate_profile TEXT NOT NULL, source_profile TEXT NOT NULL, vector_space_profile TEXT NOT NULL, raw_document_inputs INTEGER NOT NULL, legacy_query_provider_calls INTEGER, legacy_query_tokens INTEGER, logical_query_operations INTEGER, provider_attempts INTEGER, validated_responses INTEGER, failed_attempts INTEGER, retries INTEGER, observed_total_tokens INTEGER, token_observed_attempts INTEGER, token_accounting_complete INTEGER, artifact_reference TEXT NOT NULL UNIQUE, artifact_checksum TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('complete','failed','legacy')), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), CHECK((status='legacy' AND legacy_query_provider_calls IS NOT NULL AND legacy_query_tokens IS NOT NULL AND logical_query_operations IS NULL AND provider_attempts IS NULL AND validated_responses IS NULL AND failed_attempts IS NULL AND retries IS NULL AND observed_total_tokens IS NULL AND token_observed_attempts IS NULL AND token_accounting_complete IS NULL) OR (status IN ('complete','failed') AND legacy_query_provider_calls IS NULL AND legacy_query_tokens IS NULL AND logical_query_operations=query_count AND logical_query_operations>0 AND provider_attempts>=logical_query_operations AND validated_responses>=0 AND validated_responses<=logical_query_operations AND failed_attempts=provider_attempts-validated_responses AND retries=provider_attempts-logical_query_operations AND token_observed_attempts=validated_responses AND token_accounting_complete IN (0,1) AND ((observed_total_tokens IS NULL AND token_observed_attempts=0 AND token_accounting_complete=0) OR (observed_total_tokens>=0 AND token_observed_attempts>0)) AND (token_accounting_complete=0 OR (validated_responses=logical_query_operations AND failed_attempts=0)))))`

var labV3SchemaStatements = []string{
	labMetaV3TableStatement,
	`CREATE TABLE lab_inputs (canonical_input_sha256 TEXT PRIMARY KEY, canonical_text_profile TEXT NOT NULL DEFAULT '', canonical_bytes BLOB, snapshot_reference TEXT, captured_generation INTEGER NOT NULL DEFAULT 0, manifest_sha256 TEXT NOT NULL DEFAULT '', source_segment_id INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), CHECK(canonical_bytes IS NOT NULL OR snapshot_reference IS NOT NULL))`,
	`CREATE TABLE raw_document_embeddings (source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL CHECK(dimensions=1024), checksum INTEGER NOT NULL, blob BLOB NOT NULL, vector_sha256 TEXT NOT NULL, requested_model TEXT NOT NULL, response_model TEXT NOT NULL, request_id TEXT NOT NULL DEFAULT '', encoding TEXT NOT NULL CHECK(encoding='cidx-lab-f32-le-v1'), created_at TEXT NOT NULL, PRIMARY KEY(source_profile,canonical_input_sha256))`,
	`CREATE TABLE capture_runs (id INTEGER PRIMARY KEY, generation INTEGER NOT NULL, manifest_sha256 TEXT NOT NULL DEFAULT '', source_profile TEXT NOT NULL, planned_count INTEGER NOT NULL DEFAULT 0, requested_count INTEGER NOT NULL, hit_count INTEGER NOT NULL, miss_count INTEGER NOT NULL, success_count INTEGER NOT NULL, failure_count INTEGER NOT NULL, estimated_tokens INTEGER NOT NULL DEFAULT 0, actual_tokens INTEGER NOT NULL DEFAULT 0, started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), ended_at TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'running')`,
	`CREATE TABLE capture_failures (id INTEGER PRIMARY KEY, run_id INTEGER NOT NULL, source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, classification TEXT NOT NULL CHECK(classification IN ('terminal','retryable')), error_class TEXT NOT NULL, message TEXT NOT NULL, attempts INTEGER NOT NULL, last_attempted_at TEXT NOT NULL, FOREIGN KEY(run_id) REFERENCES capture_runs(id))`,
	`CREATE INDEX capture_failures_latest_by_key ON capture_failures(source_profile,canonical_input_sha256,id DESC)`,
	materializationRunsTableStatement,
	materializedVariantsTableStatement,
	evaluationRunsV3TableStatement,
}

var labSchemaStatements = []string{
	labMetaTableStatement,
	`CREATE TABLE lab_inputs (canonical_input_sha256 TEXT PRIMARY KEY, canonical_text_profile TEXT NOT NULL DEFAULT '', canonical_bytes BLOB, snapshot_reference TEXT, captured_generation INTEGER NOT NULL DEFAULT 0, manifest_sha256 TEXT NOT NULL DEFAULT '', source_segment_id INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), CHECK(canonical_bytes IS NOT NULL OR snapshot_reference IS NOT NULL))`,
	`CREATE TABLE raw_document_embeddings (source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL CHECK(dimensions=1024), checksum INTEGER NOT NULL, blob BLOB NOT NULL, vector_sha256 TEXT NOT NULL, requested_model TEXT NOT NULL, response_model TEXT NOT NULL, request_id TEXT NOT NULL DEFAULT '', encoding TEXT NOT NULL CHECK(encoding='cidx-lab-f32-le-v1'), created_at TEXT NOT NULL, PRIMARY KEY(source_profile,canonical_input_sha256))`,
	`CREATE TABLE capture_runs (id INTEGER PRIMARY KEY, generation INTEGER NOT NULL, manifest_sha256 TEXT NOT NULL DEFAULT '', source_profile TEXT NOT NULL, planned_count INTEGER NOT NULL DEFAULT 0, requested_count INTEGER NOT NULL, hit_count INTEGER NOT NULL, miss_count INTEGER NOT NULL, success_count INTEGER NOT NULL, failure_count INTEGER NOT NULL, estimated_tokens INTEGER NOT NULL DEFAULT 0, actual_tokens INTEGER NOT NULL DEFAULT 0, started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), ended_at TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'running')`,
	`CREATE TABLE capture_failures (id INTEGER PRIMARY KEY, run_id INTEGER NOT NULL, source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, classification TEXT NOT NULL CHECK(classification IN ('terminal','retryable')), error_class TEXT NOT NULL, message TEXT NOT NULL, attempts INTEGER NOT NULL, last_attempted_at TEXT NOT NULL, FOREIGN KEY(run_id) REFERENCES capture_runs(id))`,
	`CREATE INDEX capture_failures_latest_by_key ON capture_failures(source_profile,canonical_input_sha256,id DESC)`,
	materializationRunsTableStatement,
	materializedVariantsTableStatement,
	evaluationRunsTableStatement,
}

func requireExpectedSchema(ctx context.Context, db *sql.DB) error {
	for _, table := range []string{"lab_meta", "lab_inputs", "raw_document_embeddings", "capture_runs", "capture_failures", "materialization_runs", "materialized_variants", "evaluation_runs"} {
		var found int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found); err != nil || found != 1 {
			if err != nil {
				return err
			}
			return fmt.Errorf("lab schema version %d missing %s", SchemaVersion, table)
		}
	}
	var index int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='index' AND name='capture_failures_latest_by_key'`).Scan(&index); err != nil || index != 1 {
		return fmt.Errorf("lab schema missing latest failure index")
	}
	if err := requireColumnSet(ctx, db, "materialization_runs", []string{"id", "build_id", "generation", "manifest_sha256", "source_profile", "vector_space_profile", "storage_profile", "planned_count", "staged_count", "missing_count", "rejected_count", "raw_coverage", "output_checksum", "status", "started_at", "ended_at", "error"}); err != nil {
		return err
	}
	if err := requireColumnSet(ctx, db, "materialized_variants", []string{"materialization_id", "canonical_input_sha256", "dimensions", "codec_id", "codec_version", "blob", "scale", "norm", "raw_vector_sha256", "created_at"}); err != nil {
		return err
	}
	if err := requireColumnSet(ctx, db, "evaluation_runs", []string{"id", "run_id", "repository_identity", "corpus_id", "corpus_manifest_sha256", "pinned_commit", "content_sha256", "generation", "index_manifest_sha256", "query_manifest_sha256", "query_count", "candidate_profile", "source_profile", "vector_space_profile", "raw_document_inputs", "legacy_query_provider_calls", "legacy_query_tokens", "logical_query_operations", "provider_attempts", "validated_responses", "failed_attempts", "retries", "observed_total_tokens", "token_observed_attempts", "token_accounting_complete", "artifact_reference", "artifact_checksum", "status", "created_at"}); err != nil {
		return err
	}
	var metaSQL string
	if err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='lab_meta'`).Scan(&metaSQL); err != nil {
		return err
	}
	if !containsSchemaVersionCheck(metaSQL, 5) {
		return fmt.Errorf("lab v5 meta schema is not recognized")
	}
	return nil
}

func requireV3Schema(ctx context.Context, db *sql.DB) error {
	if err := requireColumnSet(ctx, db, "evaluation_runs", []string{"id", "repository_identity", "generation", "query_manifest_sha256", "candidate_profile", "artifact_reference"}); err != nil {
		return err
	}
	var metaSQL string
	if err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='lab_meta'`).Scan(&metaSQL); err != nil {
		return err
	}
	if !containsSchemaVersionCheck(metaSQL, 3) {
		return fmt.Errorf("lab v3 meta schema is not recognized")
	}
	return nil
}

func requireV4Schema(ctx context.Context, db *sql.DB) error {
	for _, table := range []string{"lab_meta", "lab_inputs", "raw_document_embeddings", "capture_runs", "capture_failures", "materialization_runs", "materialized_variants", "evaluation_runs"} {
		var found int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found); err != nil || found != 1 {
			if err != nil {
				return err
			}
			return fmt.Errorf("lab v4 schema missing %s", table)
		}
	}
	var index int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='index' AND name='capture_failures_latest_by_key'`).Scan(&index); err != nil || index != 1 {
		return fmt.Errorf("lab v4 schema missing latest failure index")
	}
	if err := requireColumnSet(ctx, db, "evaluation_runs", []string{"id", "run_id", "repository_identity", "corpus_id", "corpus_manifest_sha256", "pinned_commit", "content_sha256", "generation", "index_manifest_sha256", "query_manifest_sha256", "query_count", "candidate_profile", "source_profile", "vector_space_profile", "raw_document_inputs", "query_provider_calls", "query_tokens", "artifact_reference", "artifact_checksum", "status", "created_at"}); err != nil {
		return err
	}
	var metaSQL string
	if err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='lab_meta'`).Scan(&metaSQL); err != nil {
		return err
	}
	if !containsSchemaVersionCheck(metaSQL, 4) {
		return fmt.Errorf("lab v4 meta schema is not recognized")
	}
	return nil
}

func requireV2Schema(ctx context.Context, db *sql.DB) error {
	for _, table := range []string{"lab_meta", "lab_inputs", "raw_document_embeddings", "capture_runs", "capture_failures", "materialization_runs", "evaluation_runs"} {
		var found int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found); err != nil {
			return err
		}
		if found != 1 {
			return fmt.Errorf("lab v2 schema missing %s", table)
		}
	}
	var variants int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='materialized_variants'`).Scan(&variants); err != nil {
		return err
	}
	if variants != 0 {
		return fmt.Errorf("lab v2 schema unexpectedly has materialized variants")
	}
	var index int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='index' AND name='capture_failures_latest_by_key'`).Scan(&index); err != nil {
		return err
	}
	if index != 1 {
		return fmt.Errorf("lab v2 schema missing latest failure index")
	}
	if err := requireColumnSet(ctx, db, "lab_inputs", []string{"canonical_input_sha256", "canonical_text_profile", "canonical_bytes", "snapshot_reference", "captured_generation", "manifest_sha256", "source_segment_id"}); err != nil {
		return err
	}
	if err := requireColumnSet(ctx, db, "raw_document_embeddings", []string{"source_profile", "canonical_input_sha256", "dimensions", "checksum", "blob", "vector_sha256", "requested_model", "response_model", "request_id", "encoding"}); err != nil {
		return err
	}
	if err := requireColumnSet(ctx, db, "capture_runs", []string{"id", "generation", "manifest_sha256", "source_profile", "planned_count", "requested_count", "hit_count", "miss_count", "success_count", "failure_count", "status"}); err != nil {
		return err
	}
	if err := requireColumnSet(ctx, db, "capture_failures", []string{"id", "run_id", "source_profile", "canonical_input_sha256", "classification", "error_class", "message", "attempts"}); err != nil {
		return err
	}
	if err := requireColumnSet(ctx, db, "evaluation_runs", []string{"id", "repository_identity", "generation", "query_manifest_sha256", "candidate_profile", "artifact_reference"}); err != nil {
		return err
	}
	if err := requireColumnSet(ctx, db, "materialization_runs", []string{"id", "vector_space_profile", "storage_profile", "raw_coverage", "output_checksum", "evaluation_run_ref"}); err != nil {
		return err
	}
	var metaSQL string
	if err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='lab_meta'`).Scan(&metaSQL); err != nil {
		return err
	}
	if !containsSchemaVersionCheck(metaSQL, 2) {
		return fmt.Errorf("lab v2 meta schema is not recognized")
	}
	return nil
}

func requireColumnSet(ctx context.Context, db *sql.DB, table string, columns []string) error {
	for _, column := range columns {
		var found int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info(?) WHERE name=?`, table, column).Scan(&found); err != nil {
			return err
		}
		if found != 1 {
			return fmt.Errorf("%s missing %s", table, column)
		}
	}
	return nil
}

func containsSchemaVersionCheck(definition string, version int) bool {
	needle := fmt.Sprintf("schema_version=%d", version)
	compact := ""
	for _, r := range definition {
		if r != ' ' && r != '\n' && r != '\t' {
			compact += string(r)
		}
	}
	return containsSubstring(compact, needle)
}

func containsSubstring(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// v3 adds an isolated, search-invisible materialization work area. Raw and
// capture data are never rewritten; the legacy run table is retained as the
// immutable v3 run record with explicit defaults for its old fields.
func migrateV2ToV3(ctx context.Context, db *sql.DB) error {
	if err := requireV2Schema(ctx, db); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `ALTER TABLE lab_meta RENAME TO lab_meta_v2`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE materialization_runs RENAME TO materialization_runs_v2`); err != nil {
		return err
	}
	for _, statement := range []string{labMetaV3TableStatement, materializationRunsTableStatement, materializedVariantsTableStatement} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO lab_meta(id,schema_version,canonical_root,created_at,last_successful_collection_at) SELECT id,3,canonical_root,created_at,last_successful_collection_at FROM lab_meta_v2`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO materialization_runs(id,build_id,generation,manifest_sha256,source_profile,vector_space_profile,storage_profile,raw_coverage,output_checksum,status,evaluation_run_ref) SELECT id,'legacy-' || id,0,'','',vector_space_profile,storage_profile,raw_coverage,output_checksum,'published',evaluation_run_ref FROM materialization_runs_v2`); err != nil {
		return err
	}
	for _, name := range []string{"lab_meta_v2", "materialization_runs_v2"} {
		if _, err := tx.ExecContext(ctx, `DROP TABLE `+name); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 3`); err != nil {
		return err
	}
	return tx.Commit()
}

// migrateV3ToV4 adds vector-free retrieval-run provenance and artifact
// checksums. It copies every historical row without rewriting raw vectors or
// capture/materialization history; legacy rows are explicitly marked rather
// than being treated as current complete evaluation evidence.
func migrateV3ToV4(ctx context.Context, db *sql.DB) error {
	if err := requireV3Schema(ctx, db); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `ALTER TABLE lab_meta RENAME TO lab_meta_v3`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE evaluation_runs RENAME TO evaluation_runs_v3`); err != nil {
		return err
	}
	for _, statement := range []string{labMetaV4TableStatement, evaluationRunsV4TableStatement} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO lab_meta(id,schema_version,canonical_root,created_at,last_successful_collection_at) SELECT id,4,canonical_root,created_at,last_successful_collection_at FROM lab_meta_v3`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO evaluation_runs(id,run_id,repository_identity,corpus_id,corpus_manifest_sha256,pinned_commit,content_sha256,generation,index_manifest_sha256,query_manifest_sha256,query_count,candidate_profile,source_profile,vector_space_profile,raw_document_inputs,query_provider_calls,query_tokens,artifact_reference,artifact_checksum,status) SELECT id,'legacy-' || id,repository_identity,'','','','',generation,'',query_manifest_sha256,0,candidate_profile,'','',0,0,0,artifact_reference,'','legacy' FROM evaluation_runs_v3`); err != nil {
		return err
	}
	for _, name := range []string{"lab_meta_v3", "evaluation_runs_v3"} {
		if _, err := tx.ExecContext(ctx, `DROP TABLE `+name); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 4`); err != nil {
		return err
	}
	return tx.Commit()
}

// migrateV4ToV5 replaces ambiguous flat provider call/token totals with
// nullable legacy fields and the two-layer logical-query accounting columns.
// Historical values remain intact but are explicitly not interpreted as
// observed token usage under the current contract.
func migrateV4ToV5(ctx context.Context, db *sql.DB) error {
	if err := requireV4Schema(ctx, db); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `ALTER TABLE lab_meta RENAME TO lab_meta_v4`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE evaluation_runs RENAME TO evaluation_runs_v4`); err != nil {
		return err
	}
	for _, statement := range []string{labMetaTableStatement, evaluationRunsTableStatement} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO lab_meta(id,schema_version,canonical_root,created_at,last_successful_collection_at) SELECT id,5,canonical_root,created_at,last_successful_collection_at FROM lab_meta_v4`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO evaluation_runs(id,run_id,repository_identity,corpus_id,corpus_manifest_sha256,pinned_commit,content_sha256,generation,index_manifest_sha256,query_manifest_sha256,query_count,candidate_profile,source_profile,vector_space_profile,raw_document_inputs,legacy_query_provider_calls,legacy_query_tokens,artifact_reference,artifact_checksum,status,created_at) SELECT id,run_id,repository_identity,corpus_id,corpus_manifest_sha256,pinned_commit,content_sha256,generation,index_manifest_sha256,query_manifest_sha256,query_count,candidate_profile,source_profile,vector_space_profile,raw_document_inputs,query_provider_calls,query_tokens,artifact_reference,artifact_checksum,'legacy',created_at FROM evaluation_runs_v4`); err != nil {
		return err
	}
	for _, name := range []string{"lab_meta_v4", "evaluation_runs_v4"} {
		if _, err := tx.ExecContext(ctx, `DROP TABLE `+name); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 5`); err != nil {
		return err
	}
	return tx.Commit()
}

// migrateV1ToV3 is additive at the data boundary: immutable raw rows and
// capture history are copied byte-for-byte while v3 adds run provenance and
// search-invisible materialization variants.
func migrateV1ToV3(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	statements := []string{
		`ALTER TABLE lab_meta RENAME TO lab_meta_v1`,
		`ALTER TABLE lab_inputs RENAME TO lab_inputs_v1`,
		`ALTER TABLE raw_document_embeddings RENAME TO raw_document_embeddings_v1`,
		`ALTER TABLE capture_runs RENAME TO capture_runs_v1`,
		`ALTER TABLE materialization_runs RENAME TO materialization_runs_v1`,
		`ALTER TABLE evaluation_runs RENAME TO evaluation_runs_v1`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	for _, statement := range labV3SchemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO lab_meta(id,schema_version,canonical_root) SELECT id,3,canonical_root FROM lab_meta_v1`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO lab_inputs(canonical_input_sha256,canonical_bytes,snapshot_reference) SELECT canonical_input_sha256,canonical_bytes,snapshot_reference FROM lab_inputs_v1`); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT source_profile,canonical_input_sha256,dimensions,checksum,blob,response_model,created_at FROM raw_document_embeddings_v1`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var source, input, model, created string
		var dimensions int
		var checksum uint32
		var blob []byte
		if err := rows.Scan(&source, &input, &dimensions, &checksum, &blob, &model, &created); err != nil {
			_ = rows.Close()
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO raw_document_embeddings(source_profile,canonical_input_sha256,dimensions,checksum,blob,vector_sha256,requested_model,response_model,encoding,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, source, input, dimensions, checksum, blob, VectorSHA256(blob), "voyage-code-4", model, F32CodecID, created); err != nil {
			_ = rows.Close()
			return err
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO capture_runs(id,generation,source_profile,requested_count,hit_count,miss_count,success_count,failure_count,status) SELECT id,generation,source_profile,requested_count,hit_count,miss_count,success_count,failure_count,'complete' FROM capture_runs_v1`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO materialization_runs(id,build_id,generation,manifest_sha256,source_profile,vector_space_profile,storage_profile,raw_coverage,output_checksum,status,evaluation_run_ref) SELECT id,'legacy-' || id,0,'','',vector_space_profile,storage_profile,raw_coverage,output_checksum,'published',evaluation_run_ref FROM materialization_runs_v1`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO evaluation_runs(id,repository_identity,generation,query_manifest_sha256,candidate_profile,artifact_reference) SELECT id,repository_identity,generation,query_manifest_sha256,candidate_profile,artifact_reference FROM evaluation_runs_v1`); err != nil {
		return err
	}
	for _, name := range []string{"lab_meta_v1", "lab_inputs_v1", "raw_document_embeddings_v1", "capture_runs_v1", "materialization_runs_v1", "evaluation_runs_v1"} {
		if _, err := tx.ExecContext(ctx, `DROP TABLE `+name); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 3`); err != nil {
		return err
	}
	return tx.Commit()
}

func ensureOwnerDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		return os.Chmod(path, 0o700)
	}
	return nil
}

func ensureOwnerFile(path string) error {
	if runtime.GOOS != "windows" {
		return os.Chmod(path, 0o600)
	}
	return nil
}

func ensureOwnerDatabaseFile(path string) error {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("database path must not be a symlink")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return ensureOwnerFile(path)
}

func secureDirectoryUnderRoot(root string, components ...string) (string, error) {
	current := root
	for _, component := range components {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("lab state path %s must be a real directory", component)
			}
		} else if os.IsNotExist(err) {
			if err := os.Mkdir(current, 0o700); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
		if err := ensureOwnerDirectory(current); err != nil {
			return "", err
		}
	}
	return current, nil
}
func (store *Store) RequireSchemaVersion(ctx context.Context) error {
	version, err := store.InspectSchemaVersion(ctx)
	if err != nil {
		return err
	}
	if version != SchemaVersion {
		return fmt.Errorf("lab schema version %d requires migration", version)
	}
	return nil
}
