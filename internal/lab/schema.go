package lab

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"cidx/internal/sourcebank"
)

const SchemaVersion = 7

const labMetaTableStatement = `CREATE TABLE lab_meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=7), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), last_successful_collection_at TEXT NOT NULL DEFAULT '')`
const captureRunsTableStatement = `CREATE TABLE capture_runs (id INTEGER PRIMARY KEY, generation INTEGER NOT NULL, manifest_sha256 TEXT NOT NULL DEFAULT '', source_profile TEXT NOT NULL, planned_count INTEGER NOT NULL DEFAULT 0, requested_count INTEGER NOT NULL, hit_count INTEGER NOT NULL, miss_count INTEGER NOT NULL, success_count INTEGER NOT NULL, failure_count INTEGER NOT NULL, estimated_tokens INTEGER NOT NULL DEFAULT 0, actual_tokens INTEGER NOT NULL DEFAULT 0, started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), ended_at TEXT NOT NULL DEFAULT '', status TEXT NOT NULL CHECK(status IN ('running','complete','failed')))`
const captureFailuresTableStatement = `CREATE TABLE capture_failures (id INTEGER PRIMARY KEY, run_id INTEGER NOT NULL REFERENCES capture_runs(id), source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, classification TEXT NOT NULL CHECK(classification IN ('terminal','retryable')), error_class TEXT NOT NULL, message TEXT NOT NULL, attempts INTEGER NOT NULL, last_attempted_at TEXT NOT NULL)`
const materializationRunsTableStatement = `CREATE TABLE materialization_runs (id INTEGER PRIMARY KEY, build_id TEXT NOT NULL UNIQUE, generation INTEGER NOT NULL, manifest_sha256 TEXT NOT NULL, source_profile TEXT NOT NULL, vector_space_profile TEXT NOT NULL, storage_profile TEXT NOT NULL, planned_count INTEGER NOT NULL DEFAULT 0, staged_count INTEGER NOT NULL DEFAULT 0, missing_count INTEGER NOT NULL DEFAULT 0, rejected_count INTEGER NOT NULL DEFAULT 0, raw_coverage REAL NOT NULL DEFAULT 0, output_checksum TEXT NOT NULL DEFAULT '', evaluation_run_ref TEXT, status TEXT NOT NULL CHECK(status IN ('planned','building','ready','published','aborted','failed')), started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), ended_at TEXT NOT NULL DEFAULT '', error TEXT NOT NULL DEFAULT '')`
const evaluationRunsTableStatement = `CREATE TABLE evaluation_runs (id INTEGER PRIMARY KEY, run_id TEXT NOT NULL UNIQUE, repository_identity TEXT NOT NULL, corpus_id TEXT NOT NULL, corpus_manifest_sha256 TEXT NOT NULL, pinned_commit TEXT NOT NULL, content_sha256 TEXT NOT NULL, generation INTEGER NOT NULL, index_manifest_sha256 TEXT NOT NULL, query_manifest_sha256 TEXT NOT NULL, query_count INTEGER NOT NULL, candidate_profile TEXT NOT NULL, source_profile TEXT NOT NULL, vector_space_profile TEXT NOT NULL, raw_document_inputs INTEGER NOT NULL, legacy_query_provider_calls INTEGER, legacy_query_tokens INTEGER, logical_query_operations INTEGER, provider_attempts INTEGER, validated_responses INTEGER, failed_attempts INTEGER, retries INTEGER, observed_total_tokens INTEGER, token_observed_attempts INTEGER, token_accounting_complete INTEGER, artifact_reference TEXT NOT NULL UNIQUE, artifact_checksum TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('complete','failed','legacy')), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), CHECK((status='legacy' AND legacy_query_provider_calls IS NOT NULL AND legacy_query_tokens IS NOT NULL AND logical_query_operations IS NULL AND provider_attempts IS NULL AND validated_responses IS NULL AND failed_attempts IS NULL AND retries IS NULL AND observed_total_tokens IS NULL AND token_observed_attempts IS NULL AND token_accounting_complete IS NULL) OR (status IN ('complete','failed') AND legacy_query_provider_calls IS NULL AND legacy_query_tokens IS NULL AND logical_query_operations=query_count AND logical_query_operations>0 AND provider_attempts>=logical_query_operations AND validated_responses>=0 AND validated_responses<=logical_query_operations AND failed_attempts=provider_attempts-validated_responses AND retries=provider_attempts-logical_query_operations AND token_observed_attempts=validated_responses AND token_accounting_complete IN (0,1) AND ((observed_total_tokens IS NULL AND token_observed_attempts=0 AND token_accounting_complete=0) OR (observed_total_tokens>=0 AND token_observed_attempts>0)) AND (token_accounting_complete=0 OR (validated_responses=logical_query_operations AND failed_attempts=0)))))`

var labSchemaStatements = []string{
	labMetaTableStatement,
	captureRunsTableStatement,
	captureFailuresTableStatement,
	`CREATE INDEX capture_failures_latest_by_key ON capture_failures(source_profile,canonical_input_sha256,id DESC)`,
	materializationRunsTableStatement,
	evaluationRunsTableStatement,
	`INSERT INTO lab_meta(id,schema_version) VALUES(1,7)`,
	`PRAGMA user_version=7`,
}

func OpenStore(ctx context.Context, options Options) (*Store, error) {
	path, stateRoot, err := prepareEvaluationPath(options)
	if err != nil {
		return nil, err
	}
	sources, err := sourcebank.Open(ctx, options.SourceBankOptions())
	if err != nil {
		return nil, err
	}
	if err := ensureOwnerDatabaseFile(path); err != nil {
		_ = sources.Close()
		return nil, err
	}
	store, err := openLabDatabase(ctx, path)
	if err != nil {
		_ = sources.Close()
		return nil, err
	}
	store.stateRoot = stateRoot
	store.sources = sources
	if err := initializeLab(ctx, store.db); err != nil {
		_ = store.Close()
		return nil, err
	}
	if err := ensureOwnerFile(path); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func OpenExistingStore(ctx context.Context, options Options) (*Store, error) {
	return openExistingStore(ctx, options, true)
}

func OpenExistingStoreWritable(ctx context.Context, options Options) (*Store, error) {
	return openExistingStore(ctx, options, false)
}

func openExistingStore(ctx context.Context, options Options, readOnly bool) (*Store, error) {
	path, err := options.Path()
	if err != nil {
		return nil, err
	}
	stateRoot, err := options.ResolvedStateRoot()
	if err != nil {
		return nil, err
	}
	stateRoot, err = canonicalRoot(stateRoot)
	if err != nil {
		return nil, err
	}
	for _, value := range []string{stateRoot, filepath.Join(stateRoot, "lab")} {
		info, err := os.Lstat(value)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("existing evaluation state directory is required")
		}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("EVALUATION_STATE_INCOMPLETE")
	}
	sources, err := sourcebank.OpenExisting(ctx, options.SourceBankOptions())
	if err != nil {
		return nil, err
	}
	store, err := openLabDatabaseMode(ctx, path, readOnly)
	if err != nil {
		_ = sources.Close()
		return nil, err
	}
	store.stateRoot = stateRoot
	store.sources = sources
	if err := store.RequireSchemaVersion(ctx); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func prepareEvaluationPath(options Options) (string, string, error) {
	stateRoot, err := options.ResolvedStateRoot()
	if err != nil {
		return "", "", err
	}
	if info, statErr := os.Lstat(stateRoot); statErr == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
		return "", "", fmt.Errorf("lab state root must be a real directory")
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return "", "", statErr
	}
	if err := ensureOwnerDirectory(stateRoot); err != nil {
		return "", "", err
	}
	stateRoot, err = canonicalRoot(stateRoot)
	if err != nil {
		return "", "", err
	}
	labDir, err := secureDirectoryUnderRoot(stateRoot, "lab")
	if err != nil {
		return "", "", err
	}
	return filepath.Join(labDir, "evaluation.db"), stateRoot, nil
}

func openLabDatabaseMode(ctx context.Context, path string, readOnly bool) (*Store, error) {
	uri := &url.URL{Scheme: "file", Path: path}
	query := uri.Query()
	if readOnly {
		query.Set("mode", "ro")
	}
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
	var enabled int
	if err := db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&enabled); err != nil || enabled != 1 {
		_ = db.Close()
		return nil, fmt.Errorf("lab foreign keys unavailable")
	}
	return &Store{db: db}, nil
}

func initializeLab(ctx context.Context, db *sql.DB) error {
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
	var tables int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table'`).Scan(&tables); err != nil {
		return err
	}
	if tables != 0 {
		return fmt.Errorf("refuse unknown unversioned evaluation database")
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
	if err := tx.Commit(); err != nil {
		return err
	}
	return requireExpectedSchema(ctx, db)
}

func requireExpectedSchema(ctx context.Context, db *sql.DB) error {
	for _, table := range []string{"lab_meta", "capture_runs", "capture_failures", "materialization_runs", "evaluation_runs"} {
		var found int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found); err != nil || found != 1 {
			return fmt.Errorf("lab schema missing %s", table)
		}
	}
	for _, retired := range []string{"lab_inputs", "raw_document_embeddings", "materialized_variants"} {
		var found int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, retired).Scan(&found); err != nil || found != 0 {
			return fmt.Errorf("lab schema contains vector-bearing table %s", retired)
		}
	}
	var version int
	if err := db.QueryRowContext(ctx, `SELECT schema_version FROM lab_meta WHERE id=1`).Scan(&version); err != nil || version != SchemaVersion {
		return fmt.Errorf("lab schema metadata is invalid")
	}
	return nil
}

func (store *Store) InspectSchemaVersion(ctx context.Context) (int, error) {
	var version int
	err := store.db.QueryRowContext(ctx, `SELECT schema_version FROM lab_meta WHERE id=1`).Scan(&version)
	return version, err
}

func (store *Store) RequireSchemaVersion(ctx context.Context) error {
	version, err := store.InspectSchemaVersion(ctx)
	if err != nil {
		return err
	}
	if version != SchemaVersion {
		return fmt.Errorf("lab schema version %d requires migration", version)
	}
	return requireExpectedSchema(ctx, store.db)
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
