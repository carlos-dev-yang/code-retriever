package store

import (
	"context"
	"database/sql"
	"fmt"
)

const ProductionSchemaVersion = 1

func migrateProduction(ctx context.Context, database *sql.DB) error {
	version, err := productionSchemaVersion(ctx, database)
	if err != nil {
		return err
	}
	if version > ProductionSchemaVersion {
		return fmt.Errorf("production schema version %d is newer than supported", version)
	}
	if version == ProductionSchemaVersion {
		return requireProductionSchema(ctx, database)
	}
	var existing int
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table'`).Scan(&existing); err != nil {
		return err
	}
	if existing != 0 {
		return fmt.Errorf("refuse to stamp unknown unversioned production database")
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	statements := []string{
		`CREATE TABLE meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=1), canonical_root TEXT NOT NULL, active_generation INTEGER NOT NULL CHECK(active_generation>=0), manifest_sha256 TEXT NOT NULL, index_profile TEXT NOT NULL, index_profile_json BLOB NOT NULL, canonical_text_profile TEXT NOT NULL, canonical_text_profile_json BLOB NOT NULL, source_profile TEXT NOT NULL, source_profile_json BLOB NOT NULL, vector_space_profile TEXT NOT NULL, vector_space_profile_json BLOB NOT NULL, vector_storage_profile TEXT NOT NULL, vector_storage_profile_json BLOB NOT NULL, active_serving_profile TEXT NOT NULL, index_attempted_at TEXT NOT NULL, index_succeeded_at TEXT NOT NULL, embed_attempted_at TEXT NOT NULL, embed_succeeded_at TEXT NOT NULL, observed_git_commit TEXT NOT NULL, observed_git_dirty INTEGER NOT NULL CHECK(observed_git_dirty IN (0,1)))`,
		`CREATE TABLE IF NOT EXISTS files (id INTEGER PRIMARY KEY, path TEXT NOT NULL UNIQUE, language TEXT NOT NULL, indexed_sha256 TEXT NOT NULL, observed_mtime_ns INTEGER NOT NULL, observed_size INTEGER NOT NULL)`,
		`CREATE TABLE chunks (id INTEGER PRIMARY KEY, file_id INTEGER NOT NULL REFERENCES files(id), kind TEXT NOT NULL, symbol TEXT NOT NULL, qualified_symbol TEXT NOT NULL, signature TEXT NOT NULL, start_byte INTEGER NOT NULL CHECK(start_byte>=0), end_byte INTEGER NOT NULL CHECK(end_byte>=start_byte), start_line INTEGER NOT NULL CHECK(start_line>0), end_line INTEGER NOT NULL CHECK(end_line>=start_line), source_body BLOB NOT NULL)`,
		`CREATE TABLE chunk_projections (chunk_id INTEGER NOT NULL REFERENCES chunks(id), projection_kind TEXT NOT NULL CHECK(projection_kind IN ('signature','body')), ordinal INTEGER NOT NULL CHECK(ordinal>=0), start_byte INTEGER NOT NULL CHECK(start_byte>=0), end_byte INTEGER NOT NULL CHECK(end_byte>=start_byte), PRIMARY KEY(chunk_id, projection_kind, ordinal))`,
		`CREATE TABLE IF NOT EXISTS symbols (chunk_id INTEGER NOT NULL REFERENCES chunks(id), original_name TEXT NOT NULL, normalized_name TEXT NOT NULL, PRIMARY KEY(chunk_id, original_name))`,
		`CREATE VIRTUAL TABLE chunk_fts USING fts5(symbols, body, content='')`,
		`CREATE TABLE embedding_segments (id INTEGER PRIMARY KEY, chunk_id INTEGER NOT NULL REFERENCES chunks(id), segment_number INTEGER NOT NULL CHECK(segment_number>=0), canonical_input_sha256 TEXT NOT NULL, canonical_text_profile TEXT NOT NULL, serving_profile TEXT NOT NULL, display_start_byte INTEGER NOT NULL CHECK(display_start_byte>=0), display_end_byte INTEGER NOT NULL CHECK(display_end_byte>=display_start_byte), UNIQUE(chunk_id, segment_number))`,
		`CREATE TABLE segment_projections (segment_id INTEGER NOT NULL REFERENCES embedding_segments(id), ordinal INTEGER NOT NULL CHECK(ordinal>=0), start_byte INTEGER NOT NULL CHECK(start_byte>=0), end_byte INTEGER NOT NULL CHECK(end_byte>=start_byte), PRIMARY KEY(segment_id, ordinal))`,
		// Production intentionally has no f32/f16 column. Blob metadata validates
		// only the selected binary or int8 representation.
		`CREATE TABLE vector_cache (serving_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL CHECK(dimensions>0), codec_id TEXT NOT NULL CHECK(codec_id IN ('cidx-binary-sign-lsb-v1','cidx-int8-symmetric-v1')), codec_version INTEGER NOT NULL CHECK(codec_version=1), blob BLOB NOT NULL CHECK(length(blob)>0), scale REAL, norm REAL, materialization_fingerprint TEXT NOT NULL, PRIMARY KEY(serving_profile, canonical_input_sha256))`,
		`CREATE TABLE embedding_failures (source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, attempts INTEGER NOT NULL CHECK(attempts>0), error_class TEXT NOT NULL, last_error TEXT NOT NULL, last_attempted_at TEXT NOT NULL, PRIMARY KEY(source_profile, canonical_input_sha256))`,
		`CREATE TABLE index_runs (id INTEGER PRIMARY KEY, phase TEXT NOT NULL, state TEXT NOT NULL, reason TEXT NOT NULL, started_at TEXT NOT NULL, ended_at TEXT)`,
		`CREATE TABLE index_run_files (run_id INTEGER NOT NULL REFERENCES index_runs(id), path TEXT NOT NULL, planned_action TEXT NOT NULL, outcome TEXT NOT NULL, error TEXT, PRIMARY KEY(run_id,path))`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 1`); err != nil {
		return err
	}
	return tx.Commit()
}

func productionSchemaVersion(ctx context.Context, database *sql.DB) (int, error) {
	var version int
	err := database.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version)
	return version, err
}
func requireProductionVersion(ctx context.Context, database *sql.DB) error {
	version, err := productionSchemaVersion(ctx, database)
	if err != nil {
		return err
	}
	if version != ProductionSchemaVersion {
		return fmt.Errorf("production schema version %d requires migration", version)
	}
	return requireProductionSchema(ctx, database)
}

func requireProductionSchema(ctx context.Context, database *sql.DB) error {
	for _, table := range []string{"meta", "files", "chunks", "chunk_projections", "symbols", "chunk_fts", "embedding_segments", "segment_projections", "vector_cache", "embedding_failures", "index_runs", "index_run_files"} {
		var found int
		if err := database.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE (type='table' OR type='view') AND name=?`, table).Scan(&found); err != nil {
			return err
		}
		if found != 1 {
			return fmt.Errorf("production schema version %d missing %s", ProductionSchemaVersion, table)
		}
	}
	return nil
}
