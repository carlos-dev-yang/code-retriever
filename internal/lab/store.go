package lab

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Store is a development-only f32 artifact store. It has a distinct factory,
// file path, and schema from production store and only admits document-role
// source vectors.
type Store struct{ db *sql.DB }

func (s *Store) CanonicalRoot(ctx context.Context) (string, error) {
	var root string
	err := s.db.QueryRowContext(ctx, `SELECT canonical_root FROM lab_meta WHERE id=1`).Scan(&root)
	return root, err
}

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
	Checksum                                 uint32
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

// EvaluationRunRecord is intentionally vector-free. Query text, query
// vectors, source bodies, raw document bytes, and local checkout paths are
// excluded from the only durable evaluation reference.
type EvaluationRunRecord struct {
	RunID, RepositoryIdentity, CorpusID, CorpusManifestSHA256, PinnedCommit, ContentSHA256 string
	Generation                                                                             int64
	IndexManifestSHA256, QueryManifestSHA256, CandidateProfile                             string
	SourceProfile, VectorSpaceProfile                                                      string
	QueryCount, RawDocumentInputs, QueryProviderCalls, QueryTokens                         int
	ArtifactReference, ArtifactChecksum                                                    string
}

func (s *Store) EvaluationArtifactsRoot(ctx context.Context) (string, error) {
	root, err := s.CanonicalRoot(ctx)
	if err != nil {
		return "", err
	}
	return secureDirectoryUnderRoot(root, ".cidx", "lab", "evaluations")
}

func (s *Store) RecordEvaluationRun(ctx context.Context, record EvaluationRunRecord) (int64, error) {
	if record.RunID == "" || record.RepositoryIdentity == "" || record.CorpusID == "" || !labSHA256(record.CorpusManifestSHA256) || record.PinnedCommit == "" || !labSHA256(record.ContentSHA256) || record.Generation < 0 || !labSHA256(record.IndexManifestSHA256) || !labSHA256(record.QueryManifestSHA256) || record.CandidateProfile == "" || record.SourceProfile == "" || record.VectorSpaceProfile == "" || record.QueryCount <= 0 || record.RawDocumentInputs <= 0 || record.QueryProviderCalls < 0 || record.QueryTokens < 0 || !validArtifactReference(record.ArtifactReference) || !labSHA256(record.ArtifactChecksum) {
		return 0, fmt.Errorf("invalid evaluation run record")
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO evaluation_runs(run_id,repository_identity,corpus_id,corpus_manifest_sha256,pinned_commit,content_sha256,generation,index_manifest_sha256,query_manifest_sha256,query_count,candidate_profile,source_profile,vector_space_profile,raw_document_inputs,query_provider_calls,query_tokens,artifact_reference,artifact_checksum,status) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?, 'complete')`, record.RunID, record.RepositoryIdentity, record.CorpusID, record.CorpusManifestSHA256, record.PinnedCommit, record.ContentSHA256, record.Generation, record.IndexManifestSHA256, record.QueryManifestSHA256, record.QueryCount, record.CandidateProfile, record.SourceProfile, record.VectorSpaceProfile, record.RawDocumentInputs, record.QueryProviderCalls, record.QueryTokens, record.ArtifactReference, record.ArtifactChecksum)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) EvaluationRunCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM evaluation_runs WHERE status='complete'`).Scan(&count)
	return count, err
}

func labSHA256(value string) bool {
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

func validArtifactReference(value string) bool {
	return strings.HasPrefix(value, "evaluations/") && !strings.Contains(value, "..") && filepath.ToSlash(value) == value
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
	record.Checksum = checksum
	return record, nil
}

// RawDocuments reads and fully validates a bounded batch of immutable raw
// inputs. It is deliberately a lab-only API; production never opens this DB.
func (s *Store) RawDocuments(ctx context.Context, sourceProfile string, hashes []string) (map[string]RawEmbeddingRecord, error) {
	if sourceProfile == "" {
		return nil, fmt.Errorf("source profile is required")
	}
	output := make(map[string]RawEmbeddingRecord, len(hashes))
	for start := 0; start < len(hashes); start += sqliteVariableBatch {
		end := start + sqliteVariableBatch
		if end > len(hashes) {
			end = len(hashes)
		}
		marks := make([]string, end-start)
		args := make([]any, 0, end-start+1)
		args = append(args, sourceProfile)
		for i, hash := range hashes[start:end] {
			marks[i] = "?"
			args = append(args, hash)
		}
		rows, err := s.db.QueryContext(ctx, `SELECT canonical_input_sha256,dimensions,checksum,blob,vector_sha256,requested_model,response_model,request_id FROM raw_document_embeddings WHERE source_profile=? AND canonical_input_sha256 IN (`+strings.Join(marks, ",")+`)`, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var record RawEmbeddingRecord
			record.Key.SourceProfile = sourceProfile
			if err := rows.Scan(&record.Key.InputHash, &record.Dimensions, &record.Checksum, &record.VectorF32LE, &record.VectorSHA256, &record.RequestedModel, &record.ResponseModel, &record.RequestID); err != nil {
				rows.Close()
				return nil, err
			}
			if _, err := DecodeF32(record.VectorF32LE, record.Dimensions, record.Checksum); err != nil {
				rows.Close()
				return nil, err
			}
			if record.VectorSHA256 != VectorSHA256(record.VectorF32LE) {
				rows.Close()
				return nil, fmt.Errorf("raw vector SHA-256 mismatch")
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
