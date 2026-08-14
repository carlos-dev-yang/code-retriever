package lab

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const SchemaVersion = 2

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
	if version == 1 {
		return migrateV1ToV2(ctx, db)
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
	if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 2`); err != nil {
		return err
	}
	return tx.Commit()
}

var labSchemaStatements = []string{
	`CREATE TABLE lab_meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=2), canonical_root TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), last_successful_collection_at TEXT NOT NULL DEFAULT '')`,
	`CREATE TABLE lab_inputs (canonical_input_sha256 TEXT PRIMARY KEY, canonical_text_profile TEXT NOT NULL DEFAULT '', canonical_bytes BLOB, snapshot_reference TEXT, captured_generation INTEGER NOT NULL DEFAULT 0, manifest_sha256 TEXT NOT NULL DEFAULT '', source_segment_id INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), CHECK(canonical_bytes IS NOT NULL OR snapshot_reference IS NOT NULL))`,
	`CREATE TABLE raw_document_embeddings (source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL CHECK(dimensions=1024), checksum INTEGER NOT NULL, blob BLOB NOT NULL, vector_sha256 TEXT NOT NULL, requested_model TEXT NOT NULL, response_model TEXT NOT NULL, request_id TEXT NOT NULL DEFAULT '', encoding TEXT NOT NULL CHECK(encoding='cidx-lab-f32-le-v1'), created_at TEXT NOT NULL, PRIMARY KEY(source_profile,canonical_input_sha256))`,
	`CREATE TABLE capture_runs (id INTEGER PRIMARY KEY, generation INTEGER NOT NULL, manifest_sha256 TEXT NOT NULL DEFAULT '', source_profile TEXT NOT NULL, planned_count INTEGER NOT NULL DEFAULT 0, requested_count INTEGER NOT NULL, hit_count INTEGER NOT NULL, miss_count INTEGER NOT NULL, success_count INTEGER NOT NULL, failure_count INTEGER NOT NULL, estimated_tokens INTEGER NOT NULL DEFAULT 0, actual_tokens INTEGER NOT NULL DEFAULT 0, started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), ended_at TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'running')`,
	`CREATE TABLE capture_failures (id INTEGER PRIMARY KEY, run_id INTEGER NOT NULL, source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, classification TEXT NOT NULL CHECK(classification IN ('terminal','retryable')), error_class TEXT NOT NULL, message TEXT NOT NULL, attempts INTEGER NOT NULL, last_attempted_at TEXT NOT NULL, FOREIGN KEY(run_id) REFERENCES capture_runs(id))`,
	`CREATE INDEX capture_failures_latest_by_key ON capture_failures(source_profile,canonical_input_sha256,id DESC)`,
	`CREATE TABLE materialization_runs (id INTEGER PRIMARY KEY, vector_space_profile TEXT NOT NULL, storage_profile TEXT NOT NULL, raw_coverage REAL NOT NULL, output_checksum TEXT NOT NULL, evaluation_run_ref TEXT)`,
	`CREATE TABLE evaluation_runs (id INTEGER PRIMARY KEY, repository_identity TEXT NOT NULL, generation INTEGER NOT NULL, query_manifest_sha256 TEXT NOT NULL, candidate_profile TEXT NOT NULL, artifact_reference TEXT NOT NULL)`,
}

func requireExpectedSchema(ctx context.Context, db *sql.DB) error {
	for _, table := range []string{"lab_meta", "lab_inputs", "raw_document_embeddings", "capture_runs", "capture_failures", "materialization_runs", "evaluation_runs"} {
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
	return nil
}

// migrateV1ToV2 is additive at the database boundary: immutable raw rows are
// copied byte-for-byte while v2 adds capture provenance and failure records.
func migrateV1ToV2(ctx context.Context, db *sql.DB) error {
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
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	for _, statement := range labSchemaStatements {
		if strings.Contains(statement, "CREATE TABLE materialization_runs") || strings.Contains(statement, "CREATE TABLE evaluation_runs") {
			continue
		}
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO lab_meta(id,schema_version,canonical_root) SELECT id,2,canonical_root FROM lab_meta_v1`); err != nil {
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
	for _, name := range []string{"lab_meta_v1", "lab_inputs_v1", "raw_document_embeddings_v1", "capture_runs_v1"} {
		if _, err := tx.ExecContext(ctx, `DROP TABLE `+name); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 2`); err != nil {
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
