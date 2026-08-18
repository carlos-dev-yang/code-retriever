package relationdiag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

func createSchema(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("relation database is required")
	}
	statements := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA foreign_keys=ON`,
		`CREATE TABLE graph_meta (key TEXT PRIMARY KEY NOT NULL, value TEXT NOT NULL)`,
		`CREATE TABLE semantic_parents (
			parent_id TEXT PRIMARY KEY NOT NULL,
			path TEXT NOT NULL, indexed_sha256 TEXT NOT NULL, language TEXT NOT NULL,
			kind TEXT NOT NULL, symbol TEXT NOT NULL, qualified_symbol TEXT NOT NULL,
			start_byte INTEGER NOT NULL, end_byte INTEGER NOT NULL,
			UNIQUE(path,indexed_sha256,language,kind,qualified_symbol,start_byte,end_byte)
		)`,
		`CREATE TABLE parent_traits (
			parent_id TEXT PRIMARY KEY NOT NULL,
			file_role TEXT NOT NULL,
			deprecated INTEGER NOT NULL,
			FOREIGN KEY(parent_id) REFERENCES semantic_parents(parent_id),
			CHECK(file_role IN ('PRODUCTION','TEST','EXAMPLE','BENCHMARK')),
			CHECK(deprecated IN (0,1))
		)`,
		`CREATE TABLE relation_occurrences (
			relation_id TEXT PRIMARY KEY NOT NULL,
			source_parent_id TEXT,
			target_parent_id TEXT,
			path TEXT NOT NULL, language TEXT NOT NULL, relation_kind TEXT NOT NULL,
			start_byte INTEGER NOT NULL, end_byte INTEGER NOT NULL,
			outcome TEXT NOT NULL, resolver TEXT NOT NULL,
			occurrence_zone TEXT NOT NULL DEFAULT 'BODY', occurrence_role TEXT NOT NULL DEFAULT 'TYPE_OTHER',
			flow_role TEXT NOT NULL DEFAULT 'NONE', file_role TEXT NOT NULL DEFAULT 'PRODUCTION',
			execution_mode TEXT NOT NULL DEFAULT 'DIRECT', control_role TEXT NOT NULL DEFAULT 'NONE',
			context_identifiers TEXT NOT NULL DEFAULT '[]', source_ordinal INTEGER NOT NULL DEFAULT 1,
			FOREIGN KEY(source_parent_id) REFERENCES semantic_parents(parent_id),
			FOREIGN KEY(target_parent_id) REFERENCES semantic_parents(parent_id),
			CHECK(relation_kind IN ('CALLS','TYPE_REF','MEMBER_OF')),
			CHECK(outcome IN ('RESOLVED_UNIQUE','UNRESOLVED','AMBIGUOUS','OUT_OF_CORPUS','OUT_OF_RESOLVER_SCOPE','PARENT_MAPPING_FAILED','NO_ENCLOSING_PARENT')),
			CHECK(occurrence_zone IN ('SIGNATURE','BODY','TYPE_BODY','INITIALIZER')),
			CHECK(occurrence_role IN ('CALL_FREE_FUNCTION','CALL_METHOD','CALLABLE_VALUE','TYPE_PARAMETER','TYPE_RETURN','TYPE_FIELD','TYPE_ALIAS','TYPE_HERITAGE','TYPE_ARGUMENT','TYPE_LOCAL','TYPE_OTHER','MEMBER_RECEIVER','MEMBER_DECLARATION')),
			CHECK(flow_role IN ('NONE','RETURN','ASSIGNMENT','CONDITION','ARGUMENT','DECLARATION')),
			CHECK(file_role IN ('PRODUCTION','TEST','EXAMPLE','BENCHMARK')),
			CHECK(execution_mode IN ('DIRECT','DEFERRED','CONCURRENT','AWAITED')),
			CHECK(control_role IN ('NONE','BRANCH','LOOP','SWITCH','TRY_CATCH')),
			CHECK(source_ordinal >= 1),
			CHECK((outcome='RESOLVED_UNIQUE' AND source_parent_id IS NOT NULL AND target_parent_id IS NOT NULL) OR (outcome!='RESOLVED_UNIQUE' AND target_parent_id IS NULL))
		)`,
		`CREATE TABLE file_resolution (path TEXT PRIMARY KEY NOT NULL, language TEXT NOT NULL, outcome TEXT NOT NULL, detail TEXT NOT NULL DEFAULT '')`,
		`CREATE INDEX relation_occurrences_source ON relation_occurrences(source_parent_id, outcome, relation_kind, start_byte, relation_id)`,
		`CREATE INDEX relation_occurrences_target ON relation_occurrences(target_parent_id, outcome, relation_kind, start_byte, relation_id)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func insertGraph(ctx context.Context, db *sql.DB, metadata map[string]string, parents []Parent, occurrences []Occurrence, files []FileResolution) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for key, value := range metadata {
		if key == "" || value == "" {
			return fmt.Errorf("invalid graph metadata")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO graph_meta(key,value) VALUES(?,?)`, key, value); err != nil {
			return err
		}
	}
	for _, parent := range parents {
		if _, err := tx.ExecContext(ctx, `INSERT INTO semantic_parents(parent_id,path,indexed_sha256,language,kind,symbol,qualified_symbol,start_byte,end_byte) VALUES(?,?,?,?,?,?,?,?,?)`, parent.ID, parent.Path, parent.IndexedSHA256, parent.Language, parent.Kind, parent.Symbol, parent.QualifiedSymbol, parent.StartByte, parent.EndByte); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO parent_traits(parent_id,file_role,deprecated) VALUES(?,?,?)`, parent.ID, parent.FileRole, boolInt(parent.Deprecated)); err != nil {
			return err
		}
	}
	for _, occurrence := range occurrences {
		if err := occurrence.Validate(); err != nil {
			return err
		}
		var source, target any
		if occurrence.SourceParentID != "" {
			source = occurrence.SourceParentID
		}
		if occurrence.TargetParentID != "" {
			target = occurrence.TargetParentID
		}
		contextJSON, err := json.Marshal(occurrence.Metadata.ContextIdentifiers)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO relation_occurrences(relation_id,source_parent_id,target_parent_id,path,language,relation_kind,start_byte,end_byte,outcome,resolver,occurrence_zone,occurrence_role,flow_role,file_role,execution_mode,control_role,context_identifiers,source_ordinal) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, occurrence.ID, source, target, occurrence.Path, occurrence.Language, occurrence.Kind, occurrence.StartByte, occurrence.EndByte, occurrence.Outcome, occurrence.Resolver, occurrence.Metadata.Zone, occurrence.Metadata.Role, occurrence.Metadata.Flow, occurrence.Metadata.FileRole, occurrence.Metadata.Execution, occurrence.Metadata.Control, string(contextJSON), occurrence.Metadata.SourceOrdinal); err != nil {
			return err
		}
	}
	for _, file := range files {
		if !validRelative(file.Path) || file.Language == "" || file.Outcome == "" {
			return fmt.Errorf("invalid file resolution")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO file_resolution(path,language,outcome,detail) VALUES(?,?,?,?)`, file.Path, file.Language, file.Outcome, file.Detail); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func graphIntegrity(ctx context.Context, db *sql.DB) error {
	for _, query := range []string{`PRAGMA foreign_key_check`, `PRAGMA integrity_check`} {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				rows.Close()
				return err
			}
			if query == `PRAGMA integrity_check` && value == "ok" {
				continue
			}
			rows.Close()
			return fmt.Errorf("relation graph integrity failure: %s", value)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}
	return nil
}
