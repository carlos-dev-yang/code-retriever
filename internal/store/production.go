package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"cidx/internal/config"
)

type ProductionStore struct {
	*Stores
	Root      string
	StateRoot string
}

func OpenProduction(ctx context.Context, root string, resolved config.ResolvedConfig) (*ProductionStore, error) {
	if root == "" {
		return nil, fmt.Errorf("repository root is required")
	}
	canonicalRoot, err := canonicalRoot(root)
	if err != nil {
		return nil, err
	}
	cidxDir, err := secureDirectoryUnderRoot(canonicalRoot, ".cidx")
	if err != nil {
		return nil, err
	}
	if err := migrateLegacyProductionPath(cidxDir); err != nil {
		return nil, err
	}
	return openProductionAt(ctx, canonicalRoot, cidxDir, resolved)
}

// OpenProductionAt uses an explicit state root while preserving the same
// production schema and search/index services. Paths remain process-local and
// are never written into SQLite metadata.
func OpenProductionAt(ctx context.Context, sourceRoot, stateRoot string, resolved config.ResolvedConfig) (*ProductionStore, error) {
	if sourceRoot == "" || stateRoot == "" {
		return nil, fmt.Errorf("source and state roots are required")
	}
	canonicalSource, err := canonicalRoot(sourceRoot)
	if err != nil {
		return nil, err
	}
	if err := ensureOwnerDirectory(stateRoot); err != nil {
		return nil, err
	}
	canonicalState, err := canonicalRoot(stateRoot)
	if err != nil {
		return nil, err
	}
	return openProductionAt(ctx, canonicalSource, canonicalState, resolved)
}

func openProductionAt(ctx context.Context, canonicalRoot, stateRoot string, resolved config.ResolvedConfig) (*ProductionStore, error) {
	dbDir, err := secureDirectoryUnderRoot(stateRoot, "db")
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dbDir, "index.db")
	if err := ensureOwnerDirectory(filepath.Dir(path)); err != nil {
		return nil, err
	}
	if err := ensureOwnerDatabaseFile(path); err != nil {
		return nil, err
	}
	stores, _, err := OpenSQLiteStores(ctx, path)
	if err != nil {
		return nil, err
	}
	if err := migrateProduction(ctx, stores.Write.db); err != nil {
		_ = stores.Close()
		return nil, err
	}
	production := &ProductionStore{Stores: stores, Root: canonicalRoot, StateRoot: stateRoot}
	if err := production.ensureMeta(ctx, resolved); err != nil {
		_ = stores.Close()
		return nil, err
	}
	if err := ensureOwnerFile(path); err != nil {
		_ = stores.Close()
		return nil, err
	}
	return production, nil
}

func migrateLegacyProductionPath(stateRoot string) error {
	legacy := filepath.Join(stateRoot, "index.db")
	targetDir := filepath.Join(stateRoot, "db")
	target := filepath.Join(targetDir, "index.db")
	legacyInfo, legacyErr := os.Lstat(legacy)
	targetInfo, targetErr := os.Lstat(target)
	if legacyErr == nil && targetErr == nil {
		return fmt.Errorf("both legacy and current production database paths exist")
	}
	if legacyErr != nil {
		if os.IsNotExist(legacyErr) {
			return nil
		}
		return legacyErr
	}
	if !legacyInfo.Mode().IsRegular() || legacyInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("legacy production database path is unsafe")
	}
	if targetErr != nil && !os.IsNotExist(targetErr) {
		return targetErr
	}
	if targetInfo != nil {
		return fmt.Errorf("current production database path already exists")
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Lstat(legacy + suffix); err == nil {
			return fmt.Errorf("legacy production database must be closed before layout migration")
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := ensureOwnerDirectory(targetDir); err != nil {
		return err
	}
	return os.Rename(legacy, target)
}

func canonicalRoot(root string) (string, error) {
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
				return "", fmt.Errorf("repository state path %s must be a real directory", component)
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

func (store *ProductionStore) ensureMeta(ctx context.Context, resolved config.ResolvedConfig) error {
	if err := requireProductionVersion(ctx, store.Write.db); err != nil {
		return err
	}
	profiles := resolved.Profiles.Fingerprints
	indexJSON, err := config.CanonicalJSON(resolved.Profiles.Index)
	if err != nil {
		return err
	}
	canonicalJSON, err := config.CanonicalJSON(resolved.Profiles.CanonicalText)
	if err != nil {
		return err
	}
	sourceJSON, err := config.CanonicalJSON(resolved.Profiles.Source)
	if err != nil {
		return err
	}
	spaceJSON, err := config.CanonicalJSON(resolved.Profiles.VectorSpace)
	if err != nil {
		return err
	}
	storageJSON, err := config.CanonicalJSON(resolved.Profiles.VectorStorage)
	if err != nil {
		return err
	}
	var existing int
	err = store.Write.db.QueryRowContext(ctx, `SELECT id FROM meta WHERE id=1`).Scan(&existing)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = store.Write.db.ExecContext(ctx, `INSERT INTO meta(id,schema_version,active_generation,manifest_sha256,index_profile,index_profile_json,canonical_text_profile,canonical_text_profile_json,source_profile,source_profile_json,vector_space_profile,vector_space_profile_json,vector_storage_profile,vector_storage_profile_json,active_serving_profile,index_attempted_at,index_succeeded_at,embed_attempted_at,embed_succeeded_at,observed_git_commit,observed_git_dirty) VALUES(1,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, ProductionSchemaVersion, 0, "", profiles.Index, indexJSON, profiles.CanonicalText, canonicalJSON, profiles.Source, sourceJSON, profiles.VectorSpace, spaceJSON, profiles.VectorStorage, storageJSON, profiles.VectorStorage, "", "", "", "", "", 0)
	return err
}
