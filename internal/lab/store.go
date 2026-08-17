package lab

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"cidx/internal/sourcebank"
)

// Store owns vector-free evaluation metadata. sources is a separately opened
// product source bank used only by development capture/materialization callers.
type Store struct {
	db        *sql.DB
	stateRoot string
	sources   *sourcebank.Store
	stagedMu  sync.Mutex
	staged    map[int64][]MaterializedVariant
}

type DocumentRaw = sourcebank.DocumentSource
type RawEmbeddingKey = sourcebank.Key
type RawEmbeddingRecord = sourcebank.Record

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

type EvaluationRunRecord struct {
	RunID, RepositoryIdentity, CorpusID, CorpusManifestSHA256, PinnedCommit, ContentSHA256 string
	Generation                                                                             int64
	IndexManifestSHA256, QueryManifestSHA256, CandidateProfile                             string
	SourceProfile, VectorSpaceProfile                                                      string
	QueryCount, RawDocumentInputs                                                          int
	LogicalQueryOperations, ProviderAttempts, ValidatedResponses, FailedAttempts, Retries  int
	ObservedTotalTokens                                                                    *int
	TokenObservedAttempts                                                                  int
	TokenAccountingComplete                                                                bool
	ArtifactReference, ArtifactChecksum                                                    string
}

func (s *Store) CanonicalRoot(context.Context) (string, error) {
	if s == nil || s.stateRoot == "" {
		return "", fmt.Errorf("lab state root is unavailable")
	}
	return s.stateRoot, nil
}

func (s *Store) EvaluationArtifactsRoot(ctx context.Context) (string, error) {
	root, err := s.CanonicalRoot(ctx)
	if err != nil {
		return "", err
	}
	return secureDirectoryUnderRoot(root, "evaluations")
}

func (s *Store) RecordEvaluationRun(ctx context.Context, record EvaluationRunRecord) (int64, error) {
	if record.RunID == "" || record.RepositoryIdentity == "" || record.CorpusID == "" || !labSHA256(record.CorpusManifestSHA256) || record.PinnedCommit == "" || !labSHA256(record.ContentSHA256) || record.Generation < 0 || !labSHA256(record.IndexManifestSHA256) || !labSHA256(record.QueryManifestSHA256) || record.CandidateProfile == "" || record.SourceProfile == "" || record.VectorSpaceProfile == "" || record.QueryCount <= 0 || record.RawDocumentInputs <= 0 || record.LogicalQueryOperations != record.QueryCount || record.ProviderAttempts < record.LogicalQueryOperations || record.ValidatedResponses < 0 || record.ValidatedResponses > record.LogicalQueryOperations || record.FailedAttempts != record.ProviderAttempts-record.ValidatedResponses || record.Retries != record.ProviderAttempts-record.LogicalQueryOperations || record.TokenObservedAttempts != record.ValidatedResponses || (record.ObservedTotalTokens == nil && (record.TokenObservedAttempts != 0 || record.TokenAccountingComplete)) || (record.ObservedTotalTokens != nil && (*record.ObservedTotalTokens < 0 || record.TokenObservedAttempts == 0)) || (record.TokenAccountingComplete && (record.ValidatedResponses != record.LogicalQueryOperations || record.FailedAttempts != 0)) || !validArtifactReference(record.ArtifactReference) || !labSHA256(record.ArtifactChecksum) {
		return 0, fmt.Errorf("invalid evaluation run record")
	}
	complete := 0
	if record.TokenAccountingComplete {
		complete = 1
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO evaluation_runs(run_id,repository_identity,corpus_id,corpus_manifest_sha256,pinned_commit,content_sha256,generation,index_manifest_sha256,query_manifest_sha256,query_count,candidate_profile,source_profile,vector_space_profile,raw_document_inputs,logical_query_operations,provider_attempts,validated_responses,failed_attempts,retries,observed_total_tokens,token_observed_attempts,token_accounting_complete,artifact_reference,artifact_checksum,status) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?, 'complete')`, record.RunID, record.RepositoryIdentity, record.CorpusID, record.CorpusManifestSHA256, record.PinnedCommit, record.ContentSHA256, record.Generation, record.IndexManifestSHA256, record.QueryManifestSHA256, record.QueryCount, record.CandidateProfile, record.SourceProfile, record.VectorSpaceProfile, record.RawDocumentInputs, record.LogicalQueryOperations, record.ProviderAttempts, record.ValidatedResponses, record.FailedAttempts, record.Retries, record.ObservedTotalTokens, record.TokenObservedAttempts, complete, record.ArtifactReference, record.ArtifactChecksum)
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
	return openLabDatabaseMode(ctx, path, false)
}

func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	var values []error
	s.stagedMu.Lock()
	s.staged = nil
	s.stagedMu.Unlock()
	if s.db != nil {
		values = append(values, s.db.Close())
	}
	if s.sources != nil {
		values = append(values, s.sources.Close())
	}
	return errors.Join(values...)
}

// PutInput validates the request-local capture input. Canonical bytes are not
// durable lab data; the source bank is keyed only after a validated response.
func (s *Store) PutInput(_ context.Context, input InputRecord) error {
	if !labSHA256(input.InputHash) || len(input.CanonicalBytes) == 0 {
		return fmt.Errorf("input hash and canonical bytes are required")
	}
	return nil
}

func (s *Store) ExistingKeys(ctx context.Context, sourceProfile string, hashes []string) (map[string]bool, error) {
	if s == nil || s.sources == nil {
		return nil, fmt.Errorf("source bank is unavailable")
	}
	return s.sources.ExistingKeys(ctx, sourceProfile, hashes)
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
		for index, hash := range hashes[start:end] {
			marks[index] = "?"
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
	if err != nil || changed != 1 {
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

func (s *Store) PutDocumentSource(ctx context.Context, source DocumentRaw, sourceDimensions int) error {
	if s == nil || s.sources == nil {
		return fmt.Errorf("source bank is unavailable")
	}
	return s.sources.PutDocumentSource(ctx, source, sourceDimensions)
}

func (s *Store) PutDocumentSources(ctx context.Context, sources []DocumentRaw, sourceDimensions int) error {
	if s == nil || s.sources == nil {
		return fmt.Errorf("source bank is unavailable")
	}
	return s.sources.PutDocumentSources(ctx, sources, sourceDimensions)
}

func (s *Store) GetDocumentSource(ctx context.Context, sourceProfile, inputHash string) (F32Vector, error) {
	if s == nil || s.sources == nil {
		return F32Vector{}, fmt.Errorf("source bank is unavailable")
	}
	return s.sources.GetDocumentSource(ctx, sourceProfile, inputHash)
}

func (s *Store) GetRawDocument(ctx context.Context, key RawEmbeddingKey) (RawEmbeddingRecord, error) {
	if s == nil || s.sources == nil {
		return RawEmbeddingRecord{}, fmt.Errorf("source bank is unavailable")
	}
	return s.sources.GetDocument(ctx, key)
}

func (s *Store) RawDocuments(ctx context.Context, sourceProfile string, hashes []string) (map[string]RawEmbeddingRecord, error) {
	if s == nil || s.sources == nil {
		return nil, fmt.Errorf("source bank is unavailable")
	}
	return s.sources.Documents(ctx, sourceProfile, hashes)
}
