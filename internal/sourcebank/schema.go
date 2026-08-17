package sourcebank

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite"
)

const SchemaVersion = 1

var ErrCoverageIncomplete = errors.New("SOURCE_COVERAGE_INCOMPLETE")

const sourceMetaTable = `CREATE TABLE source_meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=1), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')))`
const documentSourcesTable = `CREATE TABLE document_source_embeddings (source_profile_fingerprint TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL CHECK(dimensions=1024), checksum INTEGER NOT NULL, vector_f32_le BLOB NOT NULL CHECK(length(vector_f32_le)=4096), vector_sha256 TEXT NOT NULL, requested_model TEXT NOT NULL, response_model TEXT NOT NULL, request_id TEXT NOT NULL DEFAULT '', encoding TEXT NOT NULL CHECK(encoding='cidx-source-f32-le-v1'), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), PRIMARY KEY(source_profile_fingerprint,canonical_input_sha256))`

func Open(ctx context.Context, options Options) (*Store, error) {
	path, stateRoot, err := preparePath(options)
	if err != nil {
		return nil, err
	}
	if err := ensureOwnerDatabaseFile(path); err != nil {
		return nil, err
	}
	store, err := openDatabase(ctx, path, stateRoot, false)
	if err != nil {
		return nil, err
	}
	if err := initialize(ctx, store.db); err != nil {
		_ = store.Close()
		return nil, err
	}
	if err := importLegacyDocumentSources(ctx, store, options); err != nil {
		_ = store.Close()
		return nil, err
	}
	if err := ensureOwnerFile(path); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func OpenExisting(ctx context.Context, options Options) (*Store, error) {
	path, err := options.Path()
	if err != nil {
		return nil, err
	}
	stateRoot, err := options.ResolvedStateRoot()
	if err != nil {
		return nil, err
	}
	stateRoot, err = canonicalStateRoot(stateRoot)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, ErrCoverageIncomplete
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("source-bank database path must be a regular file")
	}
	store, err := openDatabase(ctx, path, stateRoot, true)
	if err != nil {
		return nil, err
	}
	if err := store.RequireSchemaVersion(ctx); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func preparePath(options Options) (string, string, error) {
	stateRoot, err := options.ResolvedStateRoot()
	if err != nil {
		return "", "", err
	}
	if info, statErr := os.Lstat(stateRoot); statErr == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
		return "", "", fmt.Errorf("source-bank state root must be a real directory")
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return "", "", statErr
	}
	if err := ensureOwnerDirectory(stateRoot); err != nil {
		return "", "", err
	}
	stateRoot, err = canonicalStateRoot(stateRoot)
	if err != nil {
		return "", "", err
	}
	dbDir, err := secureDirectoryUnderRoot(stateRoot, "db")
	if err != nil {
		return "", "", err
	}
	return filepath.Join(dbDir, "embeddings.db"), stateRoot, nil
}

func openDatabase(ctx context.Context, path, stateRoot string, readOnly bool) (*Store, error) {
	uri := &url.URL{Scheme: "file", Path: path}
	query := uri.Query()
	if readOnly {
		query.Set("mode", "ro")
	}
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
	return &Store{db: db, stateRoot: stateRoot}, nil
}

func initialize(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version > SchemaVersion {
		return fmt.Errorf("source-bank schema version %d is newer than supported", version)
	}
	if version == SchemaVersion {
		return requireExpectedSchema(ctx, db)
	}
	var tables int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table'`).Scan(&tables); err != nil {
		return err
	}
	if tables != 0 {
		return fmt.Errorf("refuse unknown unversioned source bank")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range []string{sourceMetaTable, documentSourcesTable, `INSERT INTO source_meta(id,schema_version) VALUES(1,1)`, `PRAGMA user_version=1`} {
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
	for _, table := range []string{"source_meta", "document_source_embeddings"} {
		var found int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found); err != nil || found != 1 {
			return fmt.Errorf("source-bank schema missing %s", table)
		}
	}
	var version int
	if err := db.QueryRowContext(ctx, `SELECT schema_version FROM source_meta WHERE id=1`).Scan(&version); err != nil || version != SchemaVersion {
		return fmt.Errorf("source-bank schema metadata is invalid")
	}
	return nil
}

func canonicalStateRoot(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
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
		return fmt.Errorf("source-bank database path must not be a symlink")
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
				return "", fmt.Errorf("source-bank state path %s must be a real directory", component)
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
