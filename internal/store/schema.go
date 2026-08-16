package store

import (
	"context"
	"database/sql"
	"fmt"
)

const ProductionSchemaVersion = 4

func migrateProduction(ctx context.Context, database *sql.DB) error {
	version, err := productionSchemaVersion(ctx, database)
	if err != nil {
		return err
	}
	if version > ProductionSchemaVersion {
		return fmt.Errorf("production schema version %d is newer than supported", version)
	}
	if version == ProductionSchemaVersion {
		return requireProductionV4Schema(ctx, database)
	}
	if version == 3 {
		return migrateProductionV3ToV4(ctx, database)
	}
	if version == 2 {
		if err := migrateProductionV2ToV3(ctx, database); err != nil {
			return err
		}
		return migrateProductionV3ToV4(ctx, database)
	}
	if version == 1 {
		if err := migrateProductionV1ToV2(ctx, database); err != nil {
			return err
		}
		if err := migrateProductionV2ToV3(ctx, database); err != nil {
			return err
		}
		return migrateProductionV3ToV4(ctx, database)
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
		`CREATE TABLE meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=4), active_generation INTEGER NOT NULL CHECK(active_generation>=0), manifest_sha256 TEXT NOT NULL, index_profile TEXT NOT NULL, index_profile_json BLOB NOT NULL, canonical_text_profile TEXT NOT NULL, canonical_text_profile_json BLOB NOT NULL, source_profile TEXT NOT NULL, source_profile_json BLOB NOT NULL, vector_space_profile TEXT NOT NULL, vector_space_profile_json BLOB NOT NULL, vector_storage_profile TEXT NOT NULL, vector_storage_profile_json BLOB NOT NULL, active_serving_profile TEXT NOT NULL, index_attempted_at TEXT NOT NULL, index_succeeded_at TEXT NOT NULL, embed_attempted_at TEXT NOT NULL, embed_succeeded_at TEXT NOT NULL, observed_git_commit TEXT NOT NULL, observed_git_dirty INTEGER NOT NULL CHECK(observed_git_dirty IN (0,1)))`,
		`CREATE TABLE IF NOT EXISTS files (id INTEGER PRIMARY KEY, path TEXT NOT NULL UNIQUE, language TEXT NOT NULL, indexed_sha256 TEXT NOT NULL, observed_mtime_ns INTEGER NOT NULL, observed_size INTEGER NOT NULL)`,
		`CREATE TABLE chunks (id INTEGER PRIMARY KEY, file_id INTEGER NOT NULL REFERENCES files(id), kind TEXT NOT NULL, symbol TEXT NOT NULL, qualified_symbol TEXT NOT NULL, signature TEXT NOT NULL, start_byte INTEGER NOT NULL CHECK(start_byte>=0), end_byte INTEGER NOT NULL CHECK(end_byte>=start_byte), start_line INTEGER NOT NULL CHECK(start_line>0), end_line INTEGER NOT NULL CHECK(end_line>=start_line), source_body BLOB NOT NULL)`,
		`CREATE TABLE chunk_projections (chunk_id INTEGER NOT NULL REFERENCES chunks(id), projection_kind TEXT NOT NULL CHECK(projection_kind IN ('signature','body')), ordinal INTEGER NOT NULL CHECK(ordinal>=0), start_byte INTEGER NOT NULL CHECK(start_byte>=0), end_byte INTEGER NOT NULL CHECK(end_byte>=start_byte), PRIMARY KEY(chunk_id, projection_kind, ordinal))`,
		`CREATE TABLE IF NOT EXISTS symbols (chunk_id INTEGER NOT NULL REFERENCES chunks(id), original_name TEXT NOT NULL, normalized_name TEXT NOT NULL, PRIMARY KEY(chunk_id, original_name))`,
		`CREATE VIRTUAL TABLE chunk_fts USING fts5(symbols, body, content='')`,
		`CREATE TABLE embedding_segments (id INTEGER PRIMARY KEY, chunk_id INTEGER NOT NULL REFERENCES chunks(id), segment_number INTEGER NOT NULL CHECK(segment_number>=0), canonical_input_sha256 TEXT NOT NULL, canonical_text_profile TEXT NOT NULL, serving_profile TEXT NOT NULL, display_start_byte INTEGER NOT NULL CHECK(display_start_byte>=0), display_end_byte INTEGER NOT NULL CHECK(display_end_byte>=display_start_byte), UNIQUE(chunk_id, segment_number))`,
		`CREATE TABLE segment_projections (segment_id INTEGER NOT NULL REFERENCES embedding_segments(id), ordinal INTEGER NOT NULL CHECK(ordinal>=0), start_byte INTEGER NOT NULL CHECK(start_byte>=0), end_byte INTEGER NOT NULL CHECK(end_byte>=start_byte), PRIMARY KEY(segment_id, ordinal))`,
		// Production intentionally has no f32/f16 column. Blob metadata validates
		// only the selected binary or int8 representation.
		`CREATE TABLE vector_cache (serving_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL CHECK(dimensions>0), codec_id TEXT NOT NULL CHECK(codec_id IN ('cidx-binary-sign-lsb-v1','cidx-int8-symmetric-v1')), codec_version INTEGER NOT NULL CHECK(codec_version=1), blob BLOB NOT NULL CHECK(length(blob)>0), scale REAL, norm REAL, materialization_fingerprint TEXT NOT NULL, source_profile TEXT NOT NULL DEFAULT '', vector_space_profile TEXT NOT NULL DEFAULT '', raw_vector_sha256 TEXT NOT NULL DEFAULT '', materialized_at TEXT NOT NULL DEFAULT '', PRIMARY KEY(serving_profile, canonical_input_sha256))`,
		`CREATE TABLE embedding_failures (id INTEGER PRIMARY KEY, source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, classification TEXT NOT NULL CHECK(classification IN ('terminal','retryable')), attempts INTEGER NOT NULL CHECK(attempts>0), error_class TEXT NOT NULL, last_error TEXT NOT NULL, last_attempted_at TEXT NOT NULL)`,
		`CREATE INDEX embedding_failures_latest_by_key ON embedding_failures(source_profile,canonical_input_sha256,id DESC)`,
		`CREATE TABLE embedding_runs (id INTEGER PRIMARY KEY, generation INTEGER NOT NULL, manifest_sha256 TEXT NOT NULL, source_profile TEXT NOT NULL, vector_space_profile TEXT NOT NULL, storage_profile TEXT NOT NULL, planned_count INTEGER NOT NULL, ready_count INTEGER NOT NULL, skipped_count INTEGER NOT NULL, requested_count INTEGER NOT NULL, succeeded_count INTEGER NOT NULL, failed_count INTEGER NOT NULL, discarded_count INTEGER NOT NULL, estimated_tokens INTEGER NOT NULL, actual_tokens INTEGER NOT NULL, actual_cost_usd REAL, status TEXT NOT NULL CHECK(status IN ('planned','running','partially_succeeded','succeeded','failed','cancelled')), started_at TEXT NOT NULL, finished_at TEXT)`,
		`CREATE TABLE index_runs (id INTEGER PRIMARY KEY, phase TEXT NOT NULL, state TEXT NOT NULL, reason TEXT NOT NULL, started_at TEXT NOT NULL, ended_at TEXT)`,
		`CREATE TABLE index_run_files (run_id INTEGER NOT NULL REFERENCES index_runs(id), path TEXT NOT NULL, planned_action TEXT NOT NULL, outcome TEXT NOT NULL, error TEXT, PRIMARY KEY(run_id,path))`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 4`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return requireProductionV4Schema(ctx, database)
}

// v2 adds only provenance required to prove that a serving row came from the
// currently selected source and vector space. Existing rows are retained but
// have empty lineage and therefore fail the v2 readiness check until rebuilt.
func migrateProductionV1ToV2(ctx context.Context, database *sql.DB) error {
	if err := requireProductionV1Schema(ctx, database); err != nil {
		return err
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range []string{
		`ALTER TABLE meta RENAME TO meta_v1`,
		`CREATE TABLE meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=2), canonical_root TEXT NOT NULL, active_generation INTEGER NOT NULL CHECK(active_generation>=0), manifest_sha256 TEXT NOT NULL, index_profile TEXT NOT NULL, index_profile_json BLOB NOT NULL, canonical_text_profile TEXT NOT NULL, canonical_text_profile_json BLOB NOT NULL, source_profile TEXT NOT NULL, source_profile_json BLOB NOT NULL, vector_space_profile TEXT NOT NULL, vector_space_profile_json BLOB NOT NULL, vector_storage_profile TEXT NOT NULL, vector_storage_profile_json BLOB NOT NULL, active_serving_profile TEXT NOT NULL, index_attempted_at TEXT NOT NULL, index_succeeded_at TEXT NOT NULL, embed_attempted_at TEXT NOT NULL, embed_succeeded_at TEXT NOT NULL, observed_git_commit TEXT NOT NULL, observed_git_dirty INTEGER NOT NULL CHECK(observed_git_dirty IN (0,1)))`,
		`INSERT INTO meta(id,schema_version,canonical_root,active_generation,manifest_sha256,index_profile,index_profile_json,canonical_text_profile,canonical_text_profile_json,source_profile,source_profile_json,vector_space_profile,vector_space_profile_json,vector_storage_profile,vector_storage_profile_json,active_serving_profile,index_attempted_at,index_succeeded_at,embed_attempted_at,embed_succeeded_at,observed_git_commit,observed_git_dirty) SELECT id,2,canonical_root,active_generation,manifest_sha256,index_profile,index_profile_json,canonical_text_profile,canonical_text_profile_json,source_profile,source_profile_json,vector_space_profile,vector_space_profile_json,vector_storage_profile,vector_storage_profile_json,active_serving_profile,index_attempted_at,index_succeeded_at,embed_attempted_at,embed_succeeded_at,observed_git_commit,observed_git_dirty FROM meta_v1`,
		`DROP TABLE meta_v1`,
		`ALTER TABLE vector_cache ADD COLUMN source_profile TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE vector_cache ADD COLUMN vector_space_profile TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE vector_cache ADD COLUMN raw_vector_sha256 TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE vector_cache ADD COLUMN materialized_at TEXT NOT NULL DEFAULT ''`,
		`PRAGMA user_version = 2`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return requireProductionV2Schema(ctx, database)
}

// v3 makes failure state historical so "latest terminal failure" can be
// derived, and records public embedding orchestration runs. The v2 table had
// no retry classification; preserving its rows as terminal retains the old
// visible failed state while allowing new retryable attempts to supersede it.
func migrateProductionV2ToV3(ctx context.Context, database *sql.DB) error {
	if err := requireProductionV2Schema(ctx, database); err != nil {
		return err
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range []string{
		`ALTER TABLE meta RENAME TO meta_v2`,
		`CREATE TABLE meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=3), canonical_root TEXT NOT NULL, active_generation INTEGER NOT NULL CHECK(active_generation>=0), manifest_sha256 TEXT NOT NULL, index_profile TEXT NOT NULL, index_profile_json BLOB NOT NULL, canonical_text_profile TEXT NOT NULL, canonical_text_profile_json BLOB NOT NULL, source_profile TEXT NOT NULL, source_profile_json BLOB NOT NULL, vector_space_profile TEXT NOT NULL, vector_space_profile_json BLOB NOT NULL, vector_storage_profile TEXT NOT NULL, vector_storage_profile_json BLOB NOT NULL, active_serving_profile TEXT NOT NULL, index_attempted_at TEXT NOT NULL, index_succeeded_at TEXT NOT NULL, embed_attempted_at TEXT NOT NULL, embed_succeeded_at TEXT NOT NULL, observed_git_commit TEXT NOT NULL, observed_git_dirty INTEGER NOT NULL CHECK(observed_git_dirty IN (0,1)))`,
		`INSERT INTO meta(id,schema_version,canonical_root,active_generation,manifest_sha256,index_profile,index_profile_json,canonical_text_profile,canonical_text_profile_json,source_profile,source_profile_json,vector_space_profile,vector_space_profile_json,vector_storage_profile,vector_storage_profile_json,active_serving_profile,index_attempted_at,index_succeeded_at,embed_attempted_at,embed_succeeded_at,observed_git_commit,observed_git_dirty) SELECT id,3,canonical_root,active_generation,manifest_sha256,index_profile,index_profile_json,canonical_text_profile,canonical_text_profile_json,source_profile,source_profile_json,vector_space_profile,vector_space_profile_json,vector_storage_profile,vector_storage_profile_json,active_serving_profile,index_attempted_at,index_succeeded_at,embed_attempted_at,embed_succeeded_at,observed_git_commit,observed_git_dirty FROM meta_v2`,
		`DROP TABLE meta_v2`,
		`ALTER TABLE embedding_failures RENAME TO embedding_failures_v2`,
		`CREATE TABLE embedding_failures (id INTEGER PRIMARY KEY, source_profile TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, classification TEXT NOT NULL CHECK(classification IN ('terminal','retryable')), attempts INTEGER NOT NULL CHECK(attempts>0), error_class TEXT NOT NULL, last_error TEXT NOT NULL, last_attempted_at TEXT NOT NULL)`,
		`INSERT INTO embedding_failures(source_profile,canonical_input_sha256,classification,attempts,error_class,last_error,last_attempted_at) SELECT source_profile,canonical_input_sha256,'terminal',attempts,error_class,last_error,last_attempted_at FROM embedding_failures_v2`,
		`DROP TABLE embedding_failures_v2`,
		`CREATE INDEX embedding_failures_latest_by_key ON embedding_failures(source_profile,canonical_input_sha256,id DESC)`,
		`CREATE TABLE embedding_runs (id INTEGER PRIMARY KEY, generation INTEGER NOT NULL, manifest_sha256 TEXT NOT NULL, source_profile TEXT NOT NULL, vector_space_profile TEXT NOT NULL, storage_profile TEXT NOT NULL, planned_count INTEGER NOT NULL, ready_count INTEGER NOT NULL, skipped_count INTEGER NOT NULL, requested_count INTEGER NOT NULL, succeeded_count INTEGER NOT NULL, failed_count INTEGER NOT NULL, discarded_count INTEGER NOT NULL, estimated_tokens INTEGER NOT NULL, actual_tokens INTEGER NOT NULL, actual_cost_usd REAL, status TEXT NOT NULL CHECK(status IN ('planned','running','partially_succeeded','succeeded','failed','cancelled')), started_at TEXT NOT NULL, finished_at TEXT)`,
		`PRAGMA user_version = 3`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return requireProductionV3Schema(ctx, database)
}

// v4 removes the machine-specific canonical checkout path. Source and state
// roots are runtime inputs; portable manifests, profiles, and content hashes
// remain the persistent compatibility authority.
func migrateProductionV3ToV4(ctx context.Context, database *sql.DB) error {
	if err := requireProductionV3Schema(ctx, database); err != nil {
		return err
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range []string{
		`ALTER TABLE meta RENAME TO meta_v3`,
		`CREATE TABLE meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=4), active_generation INTEGER NOT NULL CHECK(active_generation>=0), manifest_sha256 TEXT NOT NULL, index_profile TEXT NOT NULL, index_profile_json BLOB NOT NULL, canonical_text_profile TEXT NOT NULL, canonical_text_profile_json BLOB NOT NULL, source_profile TEXT NOT NULL, source_profile_json BLOB NOT NULL, vector_space_profile TEXT NOT NULL, vector_space_profile_json BLOB NOT NULL, vector_storage_profile TEXT NOT NULL, vector_storage_profile_json BLOB NOT NULL, active_serving_profile TEXT NOT NULL, index_attempted_at TEXT NOT NULL, index_succeeded_at TEXT NOT NULL, embed_attempted_at TEXT NOT NULL, embed_succeeded_at TEXT NOT NULL, observed_git_commit TEXT NOT NULL, observed_git_dirty INTEGER NOT NULL CHECK(observed_git_dirty IN (0,1)))`,
		`INSERT INTO meta(id,schema_version,active_generation,manifest_sha256,index_profile,index_profile_json,canonical_text_profile,canonical_text_profile_json,source_profile,source_profile_json,vector_space_profile,vector_space_profile_json,vector_storage_profile,vector_storage_profile_json,active_serving_profile,index_attempted_at,index_succeeded_at,embed_attempted_at,embed_succeeded_at,observed_git_commit,observed_git_dirty) SELECT id,4,active_generation,manifest_sha256,index_profile,index_profile_json,canonical_text_profile,canonical_text_profile_json,source_profile,source_profile_json,vector_space_profile,vector_space_profile_json,vector_storage_profile,vector_storage_profile_json,active_serving_profile,index_attempted_at,index_succeeded_at,embed_attempted_at,embed_succeeded_at,observed_git_commit,observed_git_dirty FROM meta_v3`,
		`DROP TABLE meta_v3`,
		`PRAGMA user_version = 4`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return requireProductionV4Schema(ctx, database)
}

func requireProductionV1Schema(ctx context.Context, database *sql.DB) error {
	if err := requireProductionSchema(ctx, database); err != nil {
		return err
	}
	if err := requireProductionColumnSet(ctx, database, "meta", []string{"id", "schema_version", "canonical_root", "active_generation", "manifest_sha256", "index_profile", "index_profile_json", "canonical_text_profile", "canonical_text_profile_json", "source_profile", "source_profile_json", "vector_space_profile", "vector_space_profile_json", "vector_storage_profile", "vector_storage_profile_json", "active_serving_profile", "index_attempted_at", "index_succeeded_at", "embed_attempted_at", "embed_succeeded_at", "observed_git_commit", "observed_git_dirty"}); err != nil {
		return err
	}
	if err := requireProductionColumnSet(ctx, database, "vector_cache", []string{"serving_profile", "canonical_input_sha256", "dimensions", "codec_id", "codec_version", "blob", "scale", "norm", "materialization_fingerprint"}); err != nil {
		return err
	}
	for _, column := range []string{"source_profile", "vector_space_profile", "raw_vector_sha256", "materialized_at"} {
		var count int
		if err := database.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('vector_cache') WHERE name=?`, column).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("production v1 vector cache unexpectedly has %s", column)
		}
	}
	var metaSQL string
	if err := database.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='meta'`).Scan(&metaSQL); err != nil {
		return err
	}
	if !containsSchemaVersionCheck(metaSQL, 1) {
		return fmt.Errorf("production v1 meta schema is not recognized")
	}
	return nil
}

func requireProductionColumnSet(ctx context.Context, database *sql.DB, table string, columns []string) error {
	for _, column := range columns {
		var count int
		if err := database.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info(?) WHERE name=?`, table, column).Scan(&count); err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("production %s missing %s", table, column)
		}
	}
	return nil
}

func requireProductionExactColumns(ctx context.Context, database *sql.DB, table string, expected []string) error {
	rows, err := database.QueryContext(ctx, `SELECT name FROM pragma_table_info(?) ORDER BY cid`, table)
	if err != nil {
		return err
	}
	defer rows.Close()
	var actual []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		actual = append(actual, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(actual) != len(expected) {
		return fmt.Errorf("production %s column count mismatch", table)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			return fmt.Errorf("production %s column %d mismatch", table, i)
		}
	}
	return nil
}

func containsSchemaVersionCheck(definition string, version int) bool {
	needle := fmt.Sprintf("schema_version=%d", version)
	return containsNormalizedSQL(definition, needle)
}

func containsNormalizedSQL(value, needle string) bool {
	compact := ""
	for _, r := range value {
		if r != ' ' && r != '\n' && r != '\t' {
			compact += string(r)
		}
	}
	return len(compact) >= len(needle) && containsSubstring(compact, needle)
}

func containsSubstring(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
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
	return requireProductionV4Schema(ctx, database)
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

func requireProductionV2Schema(ctx context.Context, database *sql.DB) error {
	if err := requireProductionSchema(ctx, database); err != nil {
		return err
	}
	for _, column := range []string{"source_profile", "vector_space_profile", "raw_vector_sha256", "materialized_at"} {
		var count int
		if err := database.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('vector_cache') WHERE name=?`, column).Scan(&count); err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("production v2 vector cache missing %s", column)
		}
	}
	var metaSQL string
	if err := database.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='meta'`).Scan(&metaSQL); err != nil {
		return err
	}
	if !containsSchemaVersionCheck(metaSQL, 2) {
		return fmt.Errorf("production v2 meta schema is not recognized")
	}
	return nil
}

func requireProductionV3Schema(ctx context.Context, database *sql.DB) error {
	if err := requireProductionSchema(ctx, database); err != nil {
		return err
	}
	for _, table := range []string{"embedding_runs"} {
		var found int
		if err := database.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found); err != nil {
			return err
		}
		if found != 1 {
			return fmt.Errorf("production v3 missing %s", table)
		}
	}
	if err := requireProductionExactColumns(ctx, database, "embedding_failures", []string{"id", "source_profile", "canonical_input_sha256", "classification", "attempts", "error_class", "last_error", "last_attempted_at"}); err != nil {
		return err
	}
	if err := requireProductionExactColumns(ctx, database, "embedding_runs", []string{"id", "generation", "manifest_sha256", "source_profile", "vector_space_profile", "storage_profile", "planned_count", "ready_count", "skipped_count", "requested_count", "succeeded_count", "failed_count", "discarded_count", "estimated_tokens", "actual_tokens", "actual_cost_usd", "status", "started_at", "finished_at"}); err != nil {
		return err
	}
	var failureIndex int
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM pragma_index_list('embedding_failures') WHERE name='embedding_failures_latest_by_key'`).Scan(&failureIndex); err != nil {
		return err
	}
	if failureIndex != 1 {
		return fmt.Errorf("production v3 missing latest failure index")
	}
	rows, err := database.QueryContext(ctx, `SELECT name,desc FROM pragma_index_xinfo('embedding_failures_latest_by_key') WHERE key=1 ORDER BY seqno`)
	if err != nil {
		return err
	}
	defer rows.Close()
	expectedIndex := []struct {
		name string
		desc int
	}{{"source_profile", 0}, {"canonical_input_sha256", 0}, {"id", 1}}
	count := 0
	for rows.Next() {
		if count >= len(expectedIndex) {
			return fmt.Errorf("production v3 latest failure index has extra key")
		}
		var name string
		var desc int
		if err := rows.Scan(&name, &desc); err != nil {
			return err
		}
		if name != expectedIndex[count].name || desc != expectedIndex[count].desc {
			return fmt.Errorf("production v3 latest failure index mismatch")
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count != len(expectedIndex) {
		return fmt.Errorf("production v3 latest failure index key count mismatch")
	}
	var metaSQL string
	if err := database.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='meta'`).Scan(&metaSQL); err != nil {
		return err
	}
	if !containsSchemaVersionCheck(metaSQL, 3) {
		return fmt.Errorf("production v3 meta schema is not recognized")
	}
	return nil
}

func requireProductionV4Schema(ctx context.Context, database *sql.DB) error {
	if err := requireProductionSchema(ctx, database); err != nil {
		return err
	}
	if err := requireProductionExactColumns(ctx, database, "meta", []string{"id", "schema_version", "active_generation", "manifest_sha256", "index_profile", "index_profile_json", "canonical_text_profile", "canonical_text_profile_json", "source_profile", "source_profile_json", "vector_space_profile", "vector_space_profile_json", "vector_storage_profile", "vector_storage_profile_json", "active_serving_profile", "index_attempted_at", "index_succeeded_at", "embed_attempted_at", "embed_succeeded_at", "observed_git_commit", "observed_git_dirty"}); err != nil {
		return err
	}
	if err := requireProductionExactColumns(ctx, database, "embedding_failures", []string{"id", "source_profile", "canonical_input_sha256", "classification", "attempts", "error_class", "last_error", "last_attempted_at"}); err != nil {
		return err
	}
	if err := requireProductionExactColumns(ctx, database, "embedding_runs", []string{"id", "generation", "manifest_sha256", "source_profile", "vector_space_profile", "storage_profile", "planned_count", "ready_count", "skipped_count", "requested_count", "succeeded_count", "failed_count", "discarded_count", "estimated_tokens", "actual_tokens", "actual_cost_usd", "status", "started_at", "finished_at"}); err != nil {
		return err
	}
	var failureIndex int
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM pragma_index_list('embedding_failures') WHERE name='embedding_failures_latest_by_key'`).Scan(&failureIndex); err != nil {
		return err
	}
	if failureIndex != 1 {
		return fmt.Errorf("production v4 missing latest failure index")
	}
	var metaSQL string
	if err := database.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='meta'`).Scan(&metaSQL); err != nil {
		return err
	}
	if !containsSchemaVersionCheck(metaSQL, 4) {
		return fmt.Errorf("production v4 meta schema is not recognized")
	}
	return nil
}
