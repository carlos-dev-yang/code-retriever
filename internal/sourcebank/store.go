package sourcebank

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Store struct {
	db        *sql.DB
	stateRoot string
}

type Key struct{ SourceProfile, InputHash string }

type DocumentSource struct {
	SourceProfile  string
	InputHash      string
	RequestedModel string
	ResponseModel  string
	RequestID      string
	Vector         F32Vector
}

type Record struct {
	Key                                      Key
	Dimensions                               int
	Checksum                                 uint32
	VectorF32LE                              []byte
	VectorSHA256                             string
	RequestedModel, ResponseModel, RequestID string
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) InspectSchemaVersion(ctx context.Context) (int, error) {
	var version int
	err := s.db.QueryRowContext(ctx, `SELECT schema_version FROM source_meta WHERE id=1`).Scan(&version)
	return version, err
}

func (s *Store) RequireSchemaVersion(ctx context.Context) error {
	version, err := s.InspectSchemaVersion(ctx)
	if err != nil {
		return err
	}
	if version != SchemaVersion {
		return fmt.Errorf("source-bank schema version %d requires migration", version)
	}
	return requireExpectedSchema(ctx, s.db)
}

func (s *Store) ExistingKeys(ctx context.Context, sourceProfile string, hashes []string) (map[string]bool, error) {
	result := make(map[string]bool, len(hashes))
	for start := 0; start < len(hashes); start += sqliteVariableBatch {
		end := start + sqliteVariableBatch
		if end > len(hashes) {
			end = len(hashes)
		}
		placeholders := make([]string, end-start)
		args := make([]any, 0, end-start+1)
		args = append(args, sourceProfile)
		for index, hash := range hashes[start:end] {
			placeholders[index] = "?"
			args = append(args, hash)
		}
		rows, err := s.db.QueryContext(ctx, `SELECT canonical_input_sha256 FROM document_source_embeddings WHERE source_profile_fingerprint=? AND canonical_input_sha256 IN (`+strings.Join(placeholders, ",")+`)`, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var hash string
			if err := rows.Scan(&hash); err != nil {
				rows.Close()
				return nil, err
			}
			result[hash] = true
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

const sqliteVariableBatch = 900

func (s *Store) PutDocumentSource(ctx context.Context, source DocumentSource, sourceDimensions int) error {
	return s.PutDocumentSources(ctx, []DocumentSource{source}, sourceDimensions)
}

func (s *Store) PutDocumentSources(ctx context.Context, sources []DocumentSource, sourceDimensions int) error {
	if len(sources) == 0 || sourceDimensions != 1024 {
		return fmt.Errorf("document source batch is required")
	}
	for _, source := range sources {
		if !digest(source.SourceProfile) || !digest(source.InputHash) || source.ResponseModel == "" || source.Vector.Dimensions != sourceDimensions {
			return fmt.Errorf("invalid document source batch")
		}
		validated, err := NewF32Vector(source.Vector.Values, source.Vector.Dimensions)
		if err != nil || validated.Checksum != source.Vector.Checksum {
			return fmt.Errorf("document source vector validation failed")
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, source := range sources {
		blob := EncodeF32(source.Vector.Values)
		requested := source.RequestedModel
		if requested == "" {
			requested = source.ResponseModel
		}
		result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO document_source_embeddings(source_profile_fingerprint,canonical_input_sha256,dimensions,checksum,vector_f32_le,vector_sha256,requested_model,response_model,request_id,encoding) VALUES(?,?,?,?,?,?,?,?,?,?)`, source.SourceProfile, source.InputHash, source.Vector.Dimensions, source.Vector.Checksum, blob, VectorSHA256(blob), requested, source.ResponseModel, source.RequestID, EncodingID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			var dimensions int
			var checksum uint32
			var existing []byte
			var sha, requestedExisting, response, requestID, encoding string
			if err := tx.QueryRowContext(ctx, `SELECT dimensions,checksum,vector_f32_le,vector_sha256,requested_model,response_model,request_id,encoding FROM document_source_embeddings WHERE source_profile_fingerprint=? AND canonical_input_sha256=?`, source.SourceProfile, source.InputHash).Scan(&dimensions, &checksum, &existing, &sha, &requestedExisting, &response, &requestID, &encoding); err != nil || dimensions != source.Vector.Dimensions || checksum != source.Vector.Checksum || !bytes.Equal(existing, blob) || sha != VectorSHA256(blob) || requestedExisting != requested || response != source.ResponseModel || requestID != source.RequestID || encoding != EncodingID {
				return fmt.Errorf("immutable document source conflicts with existing source/input key")
			}
		}
	}
	return tx.Commit()
}

func (s *Store) GetDocumentSource(ctx context.Context, sourceProfile, inputHash string) (F32Vector, error) {
	var dimensions int
	var checksum uint32
	var blob []byte
	err := s.db.QueryRowContext(ctx, `SELECT dimensions,checksum,vector_f32_le FROM document_source_embeddings WHERE source_profile_fingerprint=? AND canonical_input_sha256=?`, sourceProfile, inputHash).Scan(&dimensions, &checksum, &blob)
	if err != nil {
		return F32Vector{}, err
	}
	return DecodeF32(blob, dimensions, checksum)
}

func (s *Store) GetDocument(ctx context.Context, key Key) (Record, error) {
	var record Record
	record.Key = key
	if err := s.db.QueryRowContext(ctx, `SELECT dimensions,checksum,vector_f32_le,vector_sha256,requested_model,response_model,request_id FROM document_source_embeddings WHERE source_profile_fingerprint=? AND canonical_input_sha256=?`, key.SourceProfile, key.InputHash).Scan(&record.Dimensions, &record.Checksum, &record.VectorF32LE, &record.VectorSHA256, &record.RequestedModel, &record.ResponseModel, &record.RequestID); err != nil {
		return Record{}, err
	}
	if _, err := DecodeF32(record.VectorF32LE, record.Dimensions, record.Checksum); err != nil || record.VectorSHA256 != VectorSHA256(record.VectorF32LE) {
		return Record{}, fmt.Errorf("document source integrity mismatch")
	}
	return record, nil
}

func (s *Store) Documents(ctx context.Context, sourceProfile string, hashes []string) (map[string]Record, error) {
	if !digest(sourceProfile) {
		return nil, fmt.Errorf("source profile is required")
	}
	output := make(map[string]Record, len(hashes))
	for start := 0; start < len(hashes); start += sqliteVariableBatch {
		end := start + sqliteVariableBatch
		if end > len(hashes) {
			end = len(hashes)
		}
		marks := make([]string, end-start)
		args := make([]any, 0, end-start+1)
		args = append(args, sourceProfile)
		for index, hash := range hashes[start:end] {
			marks[index] = "?"
			args = append(args, hash)
		}
		rows, err := s.db.QueryContext(ctx, `SELECT canonical_input_sha256,dimensions,checksum,vector_f32_le,vector_sha256,requested_model,response_model,request_id FROM document_source_embeddings WHERE source_profile_fingerprint=? AND canonical_input_sha256 IN (`+strings.Join(marks, ",")+`)`, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var record Record
			record.Key.SourceProfile = sourceProfile
			if err := rows.Scan(&record.Key.InputHash, &record.Dimensions, &record.Checksum, &record.VectorF32LE, &record.VectorSHA256, &record.RequestedModel, &record.ResponseModel, &record.RequestID); err != nil {
				rows.Close()
				return nil, err
			}
			if _, err := DecodeF32(record.VectorF32LE, record.Dimensions, record.Checksum); err != nil || record.VectorSHA256 != VectorSHA256(record.VectorF32LE) {
				rows.Close()
				return nil, fmt.Errorf("document source integrity mismatch")
			}
			record.VectorF32LE = append([]byte(nil), record.VectorF32LE...)
			output[record.Key.InputHash] = record
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return output, nil
}

func importLegacyDocumentSources(ctx context.Context, destination *Store, options Options) error {
	legacyPath, err := options.LegacyPath()
	if err != nil {
		return err
	}
	info, err := os.Lstat(legacyPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("legacy source database path is unsafe")
	}
	uri := &url.URL{Scheme: "file", Path: legacyPath}
	query := uri.Query()
	query.Set("mode", "ro")
	uri.RawQuery = query.Encode()
	legacy, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return err
	}
	defer legacy.Close()
	var found int
	if err := legacy.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='raw_document_embeddings'`).Scan(&found); err != nil || found == 0 {
		return err
	}
	rows, err := legacy.QueryContext(ctx, `SELECT source_profile,canonical_input_sha256,dimensions,checksum,blob,vector_sha256,requested_model,response_model,request_id,encoding FROM raw_document_embeddings ORDER BY source_profile,canonical_input_sha256`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var source DocumentSource
		var dimensions int
		var checksum uint32
		var blob []byte
		var sha, encoding string
		if err := rows.Scan(&source.SourceProfile, &source.InputHash, &dimensions, &checksum, &blob, &sha, &source.RequestedModel, &source.ResponseModel, &source.RequestID, &encoding); err != nil {
			return err
		}
		if dimensions != 1024 || encoding != "cidx-lab-f32-le-v1" || sha != VectorSHA256(blob) || !digest(source.SourceProfile) || !digest(source.InputHash) {
			continue
		}
		vector, err := DecodeF32(blob, dimensions, checksum)
		if err != nil {
			continue
		}
		source.Vector = vector
		if err := destination.PutDocumentSource(ctx, source, 1024); err != nil {
			return err
		}
	}
	return rows.Err()
}

func digest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}
