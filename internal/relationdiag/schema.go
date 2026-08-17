package relationdiag

import (
	"context"
	"database/sql"
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
		`CREATE TABLE relation_occurrences (
			relation_id TEXT PRIMARY KEY NOT NULL,
			source_parent_id TEXT,
			target_parent_id TEXT,
			path TEXT NOT NULL, language TEXT NOT NULL, relation_kind TEXT NOT NULL,
			start_byte INTEGER NOT NULL, end_byte INTEGER NOT NULL,
			outcome TEXT NOT NULL, resolver TEXT NOT NULL,
			FOREIGN KEY(source_parent_id) REFERENCES semantic_parents(parent_id),
			FOREIGN KEY(target_parent_id) REFERENCES semantic_parents(parent_id),
			CHECK(relation_kind IN ('CALLS','TYPE_REF','MEMBER_OF')),
			CHECK(outcome IN ('RESOLVED_UNIQUE','UNRESOLVED','AMBIGUOUS','OUT_OF_CORPUS','OUT_OF_RESOLVER_SCOPE','PARENT_MAPPING_FAILED','NO_ENCLOSING_PARENT')),
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
		if _, err := tx.ExecContext(ctx, `INSERT INTO relation_occurrences(relation_id,source_parent_id,target_parent_id,path,language,relation_kind,start_byte,end_byte,outcome,resolver) VALUES(?,?,?,?,?,?,?,?,?,?)`, occurrence.ID, source, target, occurrence.Path, occurrence.Language, occurrence.Kind, occurrence.StartByte, occurrence.EndByte, occurrence.Outcome, occurrence.Resolver); err != nil {
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
