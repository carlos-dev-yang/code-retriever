package lab

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Store is a development-only f32 artifact store. It has a distinct factory,
// file path, and schema from production store and only admits document-role
// source vectors.
type Store struct{ db *sql.DB }

type DocumentRaw struct {
	SourceProfile string
	InputHash     string
	ResponseModel string
	Vector        F32Vector
}

func openLabDatabase(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) PutDocumentSource(ctx context.Context, raw DocumentRaw, sourceDimensions int) error {
	if raw.SourceProfile == "" || raw.InputHash == "" || raw.ResponseModel == "" {
		return fmt.Errorf("source profile, input hash, and response model are required")
	}
	if sourceDimensions <= 0 || raw.Vector.Dimensions != sourceDimensions {
		return fmt.Errorf("lab source vector dimensions do not match the injected source profile")
	}
	validated, err := NewF32Vector(raw.Vector.Values, raw.Vector.Dimensions)
	if err != nil {
		return err
	}
	if validated.Checksum != raw.Vector.Checksum {
		return fmt.Errorf("raw vector checksum mismatch")
	}
	blob := EncodeF32(raw.Vector.Values)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO raw_document_embeddings (source_profile, canonical_input_sha256, dimensions, checksum, blob, response_model, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`, raw.SourceProfile, raw.InputHash, raw.Vector.Dimensions, raw.Vector.Checksum, blob, raw.ResponseModel)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 1 {
		return tx.Commit()
	}
	var existingDimensions int
	var existingChecksum uint32
	var existingBlob []byte
	var existingModel string
	if err := tx.QueryRowContext(ctx, `SELECT dimensions, checksum, blob, response_model FROM raw_document_embeddings WHERE source_profile=? AND canonical_input_sha256=?`, raw.SourceProfile, raw.InputHash).Scan(&existingDimensions, &existingChecksum, &existingBlob, &existingModel); err != nil {
		return err
	}
	if existingDimensions != raw.Vector.Dimensions || existingChecksum != raw.Vector.Checksum || existingModel != raw.ResponseModel || !bytes.Equal(existingBlob, blob) {
		return fmt.Errorf("immutable lab raw embedding conflicts with existing source/input key")
	}
	return tx.Commit()
}

func (s *Store) GetDocumentSource(ctx context.Context, sourceProfile, inputHash string) (F32Vector, error) {
	var dimensions int
	var checksum uint32
	var blob []byte
	err := s.db.QueryRowContext(ctx, `SELECT dimensions, checksum, blob FROM raw_document_embeddings WHERE source_profile = ? AND canonical_input_sha256 = ?`, sourceProfile, inputHash).Scan(&dimensions, &checksum, &blob)
	if err != nil {
		return F32Vector{}, err
	}
	return DecodeF32(blob, dimensions, checksum)
}
