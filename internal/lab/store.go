package lab

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "modernc.org/sqlite"
)

// Store is a development-only f32 artifact store. It has a distinct factory,
// file path, and schema from production store and only admits document-role
// source vectors.
type Store struct{ db *sql.DB }

type DocumentRaw struct {
	SourceProfile  string
	InputHash      string
	RequestedModel string
	ResponseModel  string
	RequestID      string
	Vector         F32Vector
}

type RawEmbeddingKey struct{ SourceProfile, InputHash string }

// RawEmbeddingRecord is the immutable, document-only handoff shape for later
// lab materialization. It is never accepted by production serving storage.
type RawEmbeddingRecord struct {
	Key                                      RawEmbeddingKey
	Dimensions                               int
	VectorF32LE                              []byte
	VectorSHA256                             string
	RequestedModel, ResponseModel, RequestID string
}
type InputRecord struct {
	InputHash, CanonicalTextProfile string
	CanonicalBytes                  []byte
	Generation                      int64
	ManifestSHA256                  string
	SegmentID                       int64
}
type CaptureRun struct {
	Generation                                                                         int64
	ManifestSHA256, SourceProfile                                                      string
	Planned, Requested, Hits, Misses, Persisted, Failed, EstimatedTokens, ActualTokens int
}

func openLabDatabase(ctx context.Context, path string) (*Store, error) {
	uri := &url.URL{Scheme: "file", Path: path}
	q := uri.Query()
	q.Add("_pragma", "foreign_keys(1)")
	uri.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON`); err != nil {
		_ = db.Close()
		return nil, err
	}
	var enabled int
	if err := db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&enabled); err != nil || enabled != 1 {
		_ = db.Close()
		return nil, fmt.Errorf("lab foreign keys unavailable")
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) PutInput(ctx context.Context, input InputRecord) error {
	if input.InputHash == "" || len(input.CanonicalBytes) == 0 {
		return fmt.Errorf("input hash and canonical bytes are required")
	}
	result, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO lab_inputs(canonical_input_sha256,canonical_text_profile,canonical_bytes,captured_generation,manifest_sha256,source_segment_id) VALUES(?,?,?,?,?,?)`, input.InputHash, input.CanonicalTextProfile, input.CanonicalBytes, input.Generation, input.ManifestSHA256, input.SegmentID)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil || inserted == 1 {
		return err
	}
	var existing []byte
	if err := s.db.QueryRowContext(ctx, `SELECT canonical_bytes FROM lab_inputs WHERE canonical_input_sha256=?`, input.InputHash).Scan(&existing); err != nil {
		return err
	}
	if existing == nil {
		_, err := s.db.ExecContext(ctx, `UPDATE lab_inputs SET canonical_bytes=?,canonical_text_profile=?,captured_generation=?,manifest_sha256=?,source_segment_id=? WHERE canonical_input_sha256=? AND canonical_bytes IS NULL`, input.CanonicalBytes, input.CanonicalTextProfile, input.Generation, input.ManifestSHA256, input.SegmentID, input.InputHash)
		return err
	}
	if !bytes.Equal(existing, input.CanonicalBytes) {
		return fmt.Errorf("immutable lab input conflicts with existing canonical hash")
	}
	return nil
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
		for i, hash := range hashes[start:end] {
			placeholders[i] = "?"
			args = append(args, hash)
		}
		rows, err := s.db.QueryContext(ctx, `SELECT canonical_input_sha256 FROM raw_document_embeddings WHERE source_profile=? AND canonical_input_sha256 IN (`+strings.Join(placeholders, ",")+`)`, args...)
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

func (s *Store) TerminalFailures(ctx context.Context, source string, hashes []string) (map[string]bool, error) {
	out := map[string]bool{}
	for start := 0; start < len(hashes); start += sqliteVariableBatch {
		end := start + sqliteVariableBatch
		if end > len(hashes) {
			end = len(hashes)
		}
		marks := make([]string, end-start)
		args := make([]any, 0, end-start+1)
		args = append(args, source)
		for i, hash := range hashes[start:end] {
			marks[i] = "?"
			args = append(args, hash)
		}
		query := `SELECT f.canonical_input_sha256,f.classification FROM capture_failures f JOIN (SELECT canonical_input_sha256,max(id) id FROM capture_failures WHERE source_profile=? AND canonical_input_sha256 IN (` + strings.Join(marks, ",") + `) GROUP BY canonical_input_sha256) newest ON newest.id=f.id`
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var hash, class string
			if err := rows.Scan(&hash, &class); err != nil {
				rows.Close()
				return nil, err
			}
			out[hash] = class == "terminal"
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Store) StartCapture(ctx context.Context, run CaptureRun) (int64, error) {
	result, err := s.db.ExecContext(ctx, `INSERT INTO capture_runs(generation,manifest_sha256,source_profile,planned_count,requested_count,hit_count,miss_count,success_count,failure_count,estimated_tokens,actual_tokens,status) VALUES(?,?,?,?,?,?,?,?,?,?,?,'running')`, run.Generation, run.ManifestSHA256, run.SourceProfile, run.Planned, run.Requested, run.Hits, run.Misses, run.Persisted, run.Failed, run.EstimatedTokens, run.ActualTokens)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
func (s *Store) FinishCapture(ctx context.Context, id int64, run CaptureRun, status string) error {
	if id <= 0 || (status != "complete" && status != "failed") {
		return fmt.Errorf("invalid capture completion")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE capture_runs SET requested_count=?,hit_count=?,miss_count=?,success_count=?,failure_count=?,actual_tokens=?,ended_at=strftime('%Y-%m-%dT%H:%M:%fZ','now'),status=? WHERE id=?`, run.Requested, run.Hits, run.Misses, run.Persisted, run.Failed, run.ActualTokens, status, id)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("capture run not found")
	}
	if status == "complete" {
		if _, err := tx.ExecContext(ctx, `UPDATE lab_meta SET last_successful_collection_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=1`); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) RecordFailure(ctx context.Context, runID int64, key RawEmbeddingKey, classification, class, message string, attempts int) error {
	if runID <= 0 || key.SourceProfile == "" || key.InputHash == "" || (classification != "terminal" && classification != "retryable") || class == "" || message == "" || attempts <= 0 {
		return fmt.Errorf("invalid capture failure")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO capture_failures(run_id,source_profile,canonical_input_sha256,classification,error_class,message,attempts,last_attempted_at) VALUES(?,?,?,?,?,?,?,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`, runID, key.SourceProfile, key.InputHash, classification, class, message, attempts)
	return err
}

func (s *Store) PutDocumentSource(ctx context.Context, raw DocumentRaw, sourceDimensions int) error {
	return s.PutDocumentSources(ctx, []DocumentRaw{raw}, sourceDimensions)
}

// PutDocumentSources publishes a fully validated provider batch atomically.
func (s *Store) PutDocumentSources(ctx context.Context, raws []DocumentRaw, sourceDimensions int) error {
	if len(raws) == 0 {
		return fmt.Errorf("raw embedding batch is required")
	}
	for _, raw := range raws {
		if raw.SourceProfile == "" || raw.InputHash == "" || raw.ResponseModel == "" || sourceDimensions <= 0 || raw.Vector.Dimensions != sourceDimensions {
			return fmt.Errorf("invalid raw embedding batch")
		}
		validated, err := NewF32Vector(raw.Vector.Values, raw.Vector.Dimensions)
		if err != nil || validated.Checksum != raw.Vector.Checksum {
			return fmt.Errorf("raw vector validation failed")
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, raw := range raws {
		blob := EncodeF32(raw.Vector.Values)
		requested := raw.RequestedModel
		if requested == "" {
			requested = raw.ResponseModel
		}
		result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO raw_document_embeddings (source_profile, canonical_input_sha256, dimensions, checksum, blob, vector_sha256, requested_model, response_model, request_id, encoding, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))`, raw.SourceProfile, raw.InputHash, raw.Vector.Dimensions, raw.Vector.Checksum, blob, VectorSHA256(blob), requested, raw.ResponseModel, raw.RequestID, F32CodecID)
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
			var sha, requestedExisting, response, encoding string
			if err := tx.QueryRowContext(ctx, `SELECT dimensions,checksum,blob,vector_sha256,requested_model,response_model,encoding FROM raw_document_embeddings WHERE source_profile=? AND canonical_input_sha256=?`, raw.SourceProfile, raw.InputHash).Scan(&dimensions, &checksum, &existing, &sha, &requestedExisting, &response, &encoding); err != nil || dimensions != raw.Vector.Dimensions || checksum != raw.Vector.Checksum || !bytes.Equal(existing, blob) || sha != VectorSHA256(blob) || requestedExisting != requested || response != raw.ResponseModel || encoding != F32CodecID {
				return fmt.Errorf("immutable lab raw embedding conflicts with existing source/input key")
			}
		}
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

func (s *Store) GetRawDocument(ctx context.Context, key RawEmbeddingKey) (RawEmbeddingRecord, error) {
	var record RawEmbeddingRecord
	record.Key = key
	var checksum uint32
	if err := s.db.QueryRowContext(ctx, `SELECT dimensions,checksum,blob,vector_sha256,requested_model,response_model,request_id FROM raw_document_embeddings WHERE source_profile=? AND canonical_input_sha256=?`, key.SourceProfile, key.InputHash).Scan(&record.Dimensions, &checksum, &record.VectorF32LE, &record.VectorSHA256, &record.RequestedModel, &record.ResponseModel, &record.RequestID); err != nil {
		return RawEmbeddingRecord{}, err
	}
	if _, err := DecodeF32(record.VectorF32LE, record.Dimensions, checksum); err != nil {
		return RawEmbeddingRecord{}, err
	}
	if record.VectorSHA256 != VectorSHA256(record.VectorF32LE) {
		return RawEmbeddingRecord{}, fmt.Errorf("raw vector SHA-256 mismatch")
	}
	return record, nil
}
