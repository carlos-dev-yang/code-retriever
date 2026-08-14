package lab

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"cidx/internal/vector"

	_ "modernc.org/sqlite"
)

const LabSpikeSchemaVersion = 1

// Store is a development-only f32 artifact store. It has a distinct factory,
// file path, and schema from production store and only admits document-role
// source vectors. Phase 02 replaces this temporary schema with its migration.
type Store struct{ db *sql.DB }

type DocumentRaw struct {
	SourceProfile string
	InputHash     string
	Vector        F32Vector
}

func OpenSpikeStore(ctx context.Context, path string) (*Store, error) {
	if path == "" || strings.HasSuffix(path, "/index.db") || path == ".cidx/index.db" {
		return nil, fmt.Errorf("lab store requires an explicit non-production database path")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &Store{db: db}
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS lab_spike_meta (schema_version INTEGER NOT NULL)`,
		`INSERT INTO lab_spike_meta (schema_version) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM lab_spike_meta)`,
		`CREATE TABLE IF NOT EXISTS lab_spike_document_f32 (
			source_profile TEXT NOT NULL,
			input_hash TEXT NOT NULL,
			dimensions INTEGER NOT NULL,
			checksum INTEGER NOT NULL,
			blob BLOB NOT NULL,
			PRIMARY KEY (source_profile, input_hash)
		)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) PutDocumentSource(ctx context.Context, raw DocumentRaw) error {
	if raw.SourceProfile == "" || raw.InputHash == "" {
		return fmt.Errorf("source profile and input hash are required")
	}
	if raw.Vector.Dimensions != vector.SourceDimensions {
		return fmt.Errorf("lab source vector dimensions must be %d", vector.SourceDimensions)
	}
	validated, err := NewF32Vector(raw.Vector.Values, raw.Vector.Dimensions)
	if err != nil {
		return err
	}
	if validated.Checksum != raw.Vector.Checksum {
		return fmt.Errorf("raw vector checksum mismatch")
	}
	_, err = s.db.ExecContext(ctx, `INSERT OR REPLACE INTO lab_spike_document_f32 (source_profile, input_hash, dimensions, checksum, blob) VALUES (?, ?, ?, ?, ?)`, raw.SourceProfile, raw.InputHash, raw.Vector.Dimensions, raw.Vector.Checksum, EncodeF32(raw.Vector.Values))
	return err
}

func (s *Store) GetDocumentSource(ctx context.Context, sourceProfile, inputHash string) (F32Vector, error) {
	var dimensions int
	var checksum uint32
	var blob []byte
	err := s.db.QueryRowContext(ctx, `SELECT dimensions, checksum, blob FROM lab_spike_document_f32 WHERE source_profile = ? AND input_hash = ?`, sourceProfile, inputHash).Scan(&dimensions, &checksum, &blob)
	if err != nil {
		return F32Vector{}, err
	}
	return DecodeF32(blob, dimensions, checksum)
}
