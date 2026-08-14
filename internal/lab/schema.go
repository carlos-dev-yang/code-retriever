package lab

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const SchemaVersion = 1

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
	if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 1`); err != nil {
		return err
	}
	return tx.Commit()
}

var labSchemaStatements = []string{
	`CREATE TABLE lab_meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=1), canonical_root TEXT NOT NULL)`,
	`CREATE TABLE lab_inputs (canonical_input_sha256 TEXT PRIMARY KEY, canonical_bytes BLOB, snapshot_reference TEXT, CHECK(canonical_bytes IS NOT NULL OR snapshot_reference IS NOT NULL))`,
	`CREATE TABLE raw_document_embeddings (source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL CHECK(dimensions=1024), checksum INTEGER NOT NULL, blob BLOB NOT NULL, response_model TEXT NOT NULL, created_at TEXT NOT NULL, PRIMARY KEY(source_profile,canonical_input_sha256))`,
	`CREATE TABLE capture_runs (id INTEGER PRIMARY KEY, generation INTEGER NOT NULL, source_profile TEXT NOT NULL, requested_count INTEGER NOT NULL, hit_count INTEGER NOT NULL, miss_count INTEGER NOT NULL, success_count INTEGER NOT NULL, failure_count INTEGER NOT NULL)`,
	`CREATE TABLE materialization_runs (id INTEGER PRIMARY KEY, vector_space_profile TEXT NOT NULL, storage_profile TEXT NOT NULL, raw_coverage REAL NOT NULL, output_checksum TEXT NOT NULL, evaluation_run_ref TEXT)`,
	`CREATE TABLE evaluation_runs (id INTEGER PRIMARY KEY, repository_identity TEXT NOT NULL, generation INTEGER NOT NULL, query_manifest_sha256 TEXT NOT NULL, candidate_profile TEXT NOT NULL, artifact_reference TEXT NOT NULL)`,
}

func requireExpectedSchema(ctx context.Context, db *sql.DB) error {
	for _, table := range []string{"lab_meta", "lab_inputs", "raw_document_embeddings", "capture_runs", "materialization_runs", "evaluation_runs"} {
		var found int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found); err != nil || found != 1 {
			if err != nil {
				return err
			}
			return fmt.Errorf("lab schema version %d missing %s", SchemaVersion, table)
		}
	}
	return nil
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
