package devlab

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cidx/internal/embedclient"
	"cidx/internal/eval"
	"cidx/internal/evalcontract"
	"cidx/internal/search"
)

// RetrievalArtifactReference is the durable, vector-free identity returned
// after an explicitly applied evaluation has been atomically published.
type RetrievalArtifactReference struct {
	RunID, Reference, Checksum string
}

const (
	retrievalProviderUsageKind        = "cidx.phase12.provider_usage.v2"
	retrievalRetryPolicyIdentity      = "cidx.shared_query_embedding_executor.v1"
	providerUsageTerminalValidated    = "VALIDATED"
	providerUsageTerminalFailed       = "FAILED"
	generatedResponseTokensNotApplied = "NOT_APPLICABLE"
)

// retrievalProviderUsage records one logical dataset query and the actual
// synchronous provider attempts beneath it. It deliberately has no attempt
// timings, provider status, Retry-After, error lineage, or input/output split.
type retrievalProviderUsage struct {
	SchemaVersion              int                               `json:"schema_version"`
	Kind                       string                            `json:"kind"`
	RetryPolicyIdentity        string                            `json:"retry_policy_identity"`
	MaxAttempts                int                               `json:"max_attempts"`
	ConfiguredRetryWaitSeconds []int                             `json:"configured_retry_wait_seconds"`
	AttemptTimeoutSeconds      int                               `json:"attempt_timeout_seconds"`
	SourceProfile              string                            `json:"source_profile"`
	Provider                   string                            `json:"provider"`
	Model                      string                            `json:"model"`
	AdapterVersion             int                               `json:"adapter_version"`
	QueryOperations            []retrievalProviderQueryOperation `json:"query_operations"`
	Aggregate                  retrievalProviderUsageAggregate   `json:"aggregate"`
}

type retrievalProviderQueryOperation struct {
	LogicalOperationID      int    `json:"logical_operation_id"`
	QueryID                 string `json:"query_id"`
	LogicalQueryOperations  int    `json:"logical_query_operations"`
	ProviderAttempts        int    `json:"provider_attempts"`
	Retries                 int    `json:"retries"`
	ValidatedResponses      int    `json:"validated_responses"`
	FailedAttempts          int    `json:"failed_attempts"`
	TerminalStatus          string `json:"terminal_status"`
	ObservedTotalTokens     *int   `json:"observed_total_tokens"`
	TokenObservedAttempts   int    `json:"token_observed_attempts"`
	TokenAccountingComplete bool   `json:"token_accounting_complete"`
}

type retrievalProviderUsageAggregate struct {
	LogicalQueryOperations  int    `json:"logical_query_operations"`
	ProviderAttempts        int    `json:"provider_attempts"`
	ValidatedResponses      int    `json:"validated_responses"`
	FailedAttempts          int    `json:"failed_attempts"`
	Retries                 int    `json:"retries"`
	ObservedTotalTokens     *int   `json:"observed_total_tokens"`
	TokenObservedAttempts   int    `json:"token_observed_attempts"`
	TokenAccountingComplete bool   `json:"token_accounting_complete"`
	GeneratedResponseTokens string `json:"generated_response_tokens"`
}

// retrievalProviderUsageRecorder is an operation-scoped transparent client
// factory. Phase 11 remains responsible for retry, validation, transform,
// error classification, and cancellation behavior.
type retrievalProviderUsageRecorder struct {
	delegate embedclient.EmbeddingClient
	usage    retrievalProviderUsage
	mu       sync.Mutex
	started  map[string]bool
	finished map[string]bool
}

func newRetrievalProviderUsage(prepared retrievalPrepared, delegate embedclient.EmbeddingClient) *retrievalProviderUsageRecorder {
	resolved := prepared.application.Resolved
	return &retrievalProviderUsageRecorder{
		delegate: delegate,
		usage:    retrievalProviderUsage{SchemaVersion: evalcontract.SchemaVersion, Kind: retrievalProviderUsageKind, RetryPolicyIdentity: retrievalRetryPolicyIdentity, MaxAttempts: resolved.Embedding.Retry.MaxRetries + 1, ConfiguredRetryWaitSeconds: append([]int(nil), resolved.Embedding.Retry.WaitSeconds...), AttemptTimeoutSeconds: resolved.Embedding.Request.TimeoutSeconds, SourceProfile: string(resolved.Profiles.Fingerprints.Source), Provider: resolved.Embedding.Model.Provider, Model: resolved.Embedding.Model.Model, AdapterVersion: resolved.Embedding.Model.AdapterVersion},
		started:  map[string]bool{}, finished: map[string]bool{},
	}
}

func (value *retrievalProviderUsageRecorder) Start(queryID string) (*operationRecordingClient, error) {
	value.mu.Lock()
	defer value.mu.Unlock()
	if queryID == "" || value.started[queryID] || value.delegate == nil {
		return nil, fmt.Errorf("invalid or duplicate provider usage operation")
	}
	value.started[queryID] = true
	return &operationRecordingClient{delegate: value.delegate, queryID: queryID}, nil
}

func (value *retrievalProviderUsageRecorder) Finish(ctx context.Context, client *operationRecordingClient, embedErr error) error {
	if client == nil {
		return fmt.Errorf("provider usage operation is required")
	}
	operation, err := client.finalize(ctx, embedErr)
	if err != nil {
		return err
	}
	value.mu.Lock()
	defer value.mu.Unlock()
	if value.finished[operation.QueryID] {
		return fmt.Errorf("duplicate finalized provider usage operation")
	}
	operation.LogicalOperationID = len(value.usage.QueryOperations) + 1
	value.finished[operation.QueryID] = true
	value.usage.QueryOperations = append(value.usage.QueryOperations, operation)
	return nil
}

func (value *retrievalProviderUsageRecorder) Finalize(prepared retrievalPrepared, dataset eval.EvaluationDataset, run eval.RetrievalEvaluationRun) (retrievalProviderUsage, error) {
	value.mu.Lock()
	defer value.mu.Unlock()
	if len(value.usage.QueryOperations) != len(dataset.Cases) || len(value.finished) != len(dataset.Cases) {
		return retrievalProviderUsage{}, fmt.Errorf("provider usage operations are incomplete")
	}
	value.usage.Aggregate = deriveProviderUsageAggregate(value.usage.QueryOperations)
	if err := value.usage.Validate(prepared, dataset, run); err != nil {
		return retrievalProviderUsage{}, err
	}
	return value.usage, nil
}

type operationRecordingClient struct {
	delegate  embedclient.EmbeddingClient
	queryID   string
	mu        sync.Mutex
	attempts  int
	success   bool
	tokens    int
	violation bool
}

func (value *operationRecordingClient) Embed(ctx context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	value.mu.Lock()
	if value.success {
		value.violation = true
		value.mu.Unlock()
		return embedclient.EmbeddingResponse{}, fmt.Errorf("provider attempt after successful response")
	}
	value.attempts++
	value.mu.Unlock()
	response, err := value.delegate.Embed(ctx, request)
	if err != nil {
		return embedclient.EmbeddingResponse{}, err
	}
	value.mu.Lock()
	defer value.mu.Unlock()
	if value.success || response.TotalTokens < 0 {
		value.violation = true
		return embedclient.EmbeddingResponse{}, fmt.Errorf("invalid successful provider response accounting")
	}
	value.success, value.tokens = true, response.TotalTokens
	return response, nil
}

func (value *operationRecordingClient) finalize(ctx context.Context, embedErr error) (retrievalProviderQueryOperation, error) {
	value.mu.Lock()
	defer value.mu.Unlock()
	if value.violation || value.attempts == 0 {
		return retrievalProviderQueryOperation{}, fmt.Errorf("invalid provider usage attempt sequence")
	}
	// Only the caller's context aborts publication. A Phase 11 provider
	// failure can wrap a per-attempt deadline, so errors.Is(embedErr,
	// context.DeadlineExceeded) would incorrectly erase required operational
	// denominator evidence.
	if err := ctx.Err(); err != nil {
		return retrievalProviderQueryOperation{}, err
	}
	operation := retrievalProviderQueryOperation{QueryID: value.queryID, LogicalQueryOperations: 1, ProviderAttempts: value.attempts, Retries: value.attempts - 1}
	if value.success {
		if embedErr != nil {
			// A successful HTTP response that Phase 11 rejects is a local
			// invariant failure, never an operational denominator observation.
			return retrievalProviderQueryOperation{}, embedErr
		}
		tokens := value.tokens
		operation.ValidatedResponses, operation.FailedAttempts = 1, value.attempts-1
		operation.TerminalStatus = providerUsageTerminalValidated
		operation.ObservedTotalTokens, operation.TokenObservedAttempts = &tokens, 1
		operation.TokenAccountingComplete = value.attempts == 1
		return operation, nil
	}
	var providerFailure search.QueryEmbeddingProviderError
	if embedErr == nil || !errors.As(embedErr, &providerFailure) {
		return retrievalProviderQueryOperation{}, fmt.Errorf("unclassified query embedding failure: %w", embedErr)
	}
	operation.FailedAttempts, operation.TerminalStatus = value.attempts, providerUsageTerminalFailed
	return operation, nil
}

func deriveProviderUsageAggregate(operations []retrievalProviderQueryOperation) retrievalProviderUsageAggregate {
	result := retrievalProviderUsageAggregate{GeneratedResponseTokens: generatedResponseTokensNotApplied}
	var observed int
	for _, operation := range operations {
		result.LogicalQueryOperations += operation.LogicalQueryOperations
		result.ProviderAttempts += operation.ProviderAttempts
		result.ValidatedResponses += operation.ValidatedResponses
		result.FailedAttempts += operation.FailedAttempts
		result.Retries += operation.Retries
		result.TokenObservedAttempts += operation.TokenObservedAttempts
		if operation.ObservedTotalTokens != nil {
			observed += *operation.ObservedTotalTokens
		}
	}
	if result.TokenObservedAttempts > 0 {
		result.ObservedTotalTokens = &observed
	}
	result.TokenAccountingComplete = result.ValidatedResponses == result.LogicalQueryOperations && result.FailedAttempts == 0
	return result
}

func (value retrievalProviderUsage) Validate(prepared retrievalPrepared, dataset eval.EvaluationDataset, run eval.RetrievalEvaluationRun) error {
	if value.SchemaVersion != evalcontract.SchemaVersion || value.Kind != retrievalProviderUsageKind || value.RetryPolicyIdentity != retrievalRetryPolicyIdentity || value.MaxAttempts <= 0 || value.AttemptTimeoutSeconds <= 0 || value.SourceProfile == "" || value.Provider == "" || value.Model == "" || value.AdapterVersion <= 0 || len(value.ConfiguredRetryWaitSeconds) != value.MaxAttempts-1 || len(value.QueryOperations) != len(dataset.Cases) || len(run.Cases) != len(dataset.Cases) {
		return fmt.Errorf("invalid provider usage wire")
	}
	if prepared.application != nil {
		resolved := prepared.application.Resolved
		if value.MaxAttempts != resolved.Embedding.Retry.MaxRetries+1 || !equalRetryWaits(value.ConfiguredRetryWaitSeconds, resolved.Embedding.Retry.WaitSeconds) || value.AttemptTimeoutSeconds != resolved.Embedding.Request.TimeoutSeconds || value.SourceProfile != string(resolved.Profiles.Fingerprints.Source) || value.Provider != resolved.Embedding.Model.Provider || value.Model != resolved.Embedding.Model.Model || value.AdapterVersion != resolved.Embedding.Model.AdapterVersion {
			return fmt.Errorf("provider usage policy does not match resolved configuration")
		}
	}
	for index, wait := range value.ConfiguredRetryWaitSeconds {
		if wait <= 0 || (index > 0 && wait <= value.ConfiguredRetryWaitSeconds[index-1]) {
			return fmt.Errorf("invalid configured retry waits")
		}
	}
	for index, operation := range value.QueryOperations {
		if operation.LogicalOperationID != index+1 || operation.QueryID != dataset.Cases[index].ID || operation.LogicalQueryOperations != 1 || operation.ProviderAttempts < 1 || operation.ProviderAttempts > value.MaxAttempts || operation.Retries != operation.ProviderAttempts-1 || operation.ValidatedResponses < 0 || operation.ValidatedResponses > 1 || operation.FailedAttempts != operation.ProviderAttempts-operation.ValidatedResponses {
			return fmt.Errorf("invalid provider usage operation")
		}
		if operation.ValidatedResponses == 1 {
			if operation.TerminalStatus != providerUsageTerminalValidated || operation.ObservedTotalTokens == nil || *operation.ObservedTotalTokens < 0 || operation.TokenObservedAttempts != 1 || operation.TokenAccountingComplete != (operation.ProviderAttempts == 1) {
				return fmt.Errorf("invalid validated provider usage operation")
			}
		} else if operation.TerminalStatus != providerUsageTerminalFailed || operation.ObservedTotalTokens != nil || operation.TokenObservedAttempts != 0 || operation.TokenAccountingComplete {
			return fmt.Errorf("invalid failed provider usage operation")
		}
		if run.Cases[index].Case.QueryID != operation.QueryID || !providerUsageMatchesRun(operation, run.Cases[index]) {
			return fmt.Errorf("provider usage does not match evaluation run")
		}
	}
	if expected := deriveProviderUsageAggregate(value.QueryOperations); !equalProviderUsageAggregate(expected, value.Aggregate) {
		return fmt.Errorf("provider usage aggregate is not derived")
	}
	return nil
}

func equalRetryWaits(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalProviderUsageAggregate(left, right retrievalProviderUsageAggregate) bool {
	if left.LogicalQueryOperations != right.LogicalQueryOperations || left.ProviderAttempts != right.ProviderAttempts || left.ValidatedResponses != right.ValidatedResponses || left.FailedAttempts != right.FailedAttempts || left.Retries != right.Retries || left.TokenObservedAttempts != right.TokenObservedAttempts || left.TokenAccountingComplete != right.TokenAccountingComplete || left.GeneratedResponseTokens != right.GeneratedResponseTokens || (left.ObservedTotalTokens == nil) != (right.ObservedTotalTokens == nil) {
		return false
	}
	return left.ObservedTotalTokens == nil || *left.ObservedTotalTokens == *right.ObservedTotalTokens
}

func providerUsageMatchesRun(operation retrievalProviderQueryOperation, evidence eval.RetrievalCaseEvidence) bool {
	seenTarget := false
	for _, arm := range evidence.Arms {
		vectorBearing := arm.Ranking.Variant != eval.VariantFTS && arm.Ranking.Variant != eval.VariantHybridWithoutDense
		if operation.TerminalStatus == providerUsageTerminalFailed {
			if vectorBearing && arm.FailureStage != evalcontract.FailureStage(evalcontract.StageOperational) {
				return false
			}
			if !vectorBearing && arm.FailureStage != "" {
				return false
			}
		} else if arm.FailureStage != "" {
			return false
		}
		if arm.Ranking.Variant == eval.VariantTargetF32 {
			seenTarget = true
		}
	}
	return seenTarget
}

type retrievalArtifactManifest struct {
	SchemaVersion          int                             `json:"schema_version"`
	RunID                  string                          `json:"run_id"`
	CreatedAt              string                          `json:"created_at"`
	CorpusID               string                          `json:"corpus_id"`
	CorpusManifestSHA256   string                          `json:"corpus_manifest_sha256"`
	PinnedCommit           string                          `json:"pinned_commit"`
	ContentSHA256          string                          `json:"content_sha256"`
	IndexGeneration        int64                           `json:"index_generation"`
	IndexManifestSHA256    string                          `json:"index_manifest_sha256"`
	DatasetSHA256          string                          `json:"dataset_sha256"`
	QueryIDs               []string                        `json:"query_ids"`
	ServingProfile         string                          `json:"serving_profile"`
	SourceProfile          string                          `json:"source_profile"`
	VectorSpaceProfile     string                          `json:"vector_space_profile"`
	RawDocumentInputs      int                             `json:"raw_document_inputs"`
	CandidatePolicy        string                          `json:"candidate_policy"`
	BodyBudget             int                             `json:"body_budget"`
	PromotionEvidenceState string                          `json:"promotion_evidence_state"`
	ProviderUsage          retrievalProviderUsageAggregate `json:"provider_usage"`
}

type retrievalArtifactChecksums struct {
	SchemaVersion             int                          `json:"schema_version"`
	Kind                      string                       `json:"kind"`
	Complete                  bool                         `json:"complete"`
	PromotionEvidenceComplete bool                         `json:"promotion_evidence_complete"`
	Entries                   []evalcontract.ArtifactEntry `json:"entries"`
	EntriesChecksum           string                       `json:"entries_checksum"`
}

// incompletePromotionArtifact occupies a required artifact filename without
// claiming to be a valid strict evalcontract promotion object.
type incompletePromotionArtifact struct {
	SchemaVersion             int                         `json:"schema_version"`
	Kind                      string                      `json:"kind"`
	Scope                     evalcontract.PromotionScope `json:"scope"`
	PromotionEvidenceComplete bool                        `json:"promotion_evidence_complete"`
	Reason                    string                      `json:"reason"`
}

// publishRetrievalArtifact creates the Phase 12 immutable, portable artifact
// directory. It intentionally serializes only identities, hashes, ranks,
// ranges, score diagnostics, and body digests; no query text/vector, source
// body, raw vector, credential, or absolute checkout path enters the files.
func publishRetrievalArtifact(ctx context.Context, prepared retrievalPrepared, run eval.RetrievalEvaluationRun, usage retrievalProviderUsage) (RetrievalArtifactReference, error) {
	if err := run.Validate(prepared.dataset); err != nil {
		return RetrievalArtifactReference{}, err
	}
	if err := usage.Validate(prepared, prepared.dataset, run); err != nil {
		return RetrievalArtifactReference{}, err
	}
	runID, err := newRetrievalRunID()
	if err != nil {
		return RetrievalArtifactReference{}, err
	}
	root, err := prepared.raw.EvaluationArtifactsRoot(ctx)
	if err != nil {
		return RetrievalArtifactReference{}, err
	}
	target := filepath.Join(root, runID)
	if _, err := os.Lstat(target); err == nil {
		return RetrievalArtifactReference{}, fmt.Errorf("retrieval artifact already exists")
	} else if !os.IsNotExist(err) {
		return RetrievalArtifactReference{}, err
	}
	temporary, err := os.MkdirTemp(root, ".retrieval-")
	if err != nil {
		return RetrievalArtifactReference{}, err
	}
	defer os.RemoveAll(temporary)

	queryIDs := make([]string, 0, len(prepared.dataset.Cases))
	for _, item := range prepared.dataset.Cases {
		queryIDs = append(queryIDs, item.ID)
	}
	manifest := retrievalArtifactManifest{
		SchemaVersion:          evalcontract.SchemaVersion,
		RunID:                  runID,
		CreatedAt:              time.Now().UTC().Format(time.RFC3339Nano),
		CorpusID:               prepared.plan.CorpusID,
		CorpusManifestSHA256:   prepared.plan.CorpusManifestSHA256,
		PinnedCommit:           prepared.plan.PinnedCommit,
		ContentSHA256:          prepared.plan.ContentSHA256,
		IndexGeneration:        prepared.plan.IndexGeneration,
		IndexManifestSHA256:    prepared.plan.IndexManifestSHA256,
		DatasetSHA256:          prepared.plan.DatasetSHA256,
		QueryIDs:               queryIDs,
		ServingProfile:         prepared.plan.ServingProfile,
		SourceProfile:          string(prepared.application.Resolved.Profiles.Fingerprints.Source),
		VectorSpaceProfile:     string(prepared.application.Resolved.Profiles.Fingerprints.VectorSpace),
		RawDocumentInputs:      prepared.plan.RawDocumentInputs,
		CandidatePolicy:        fmt.Sprintf("candidate-k=%d;return-k=%d;rrf-k=%d", prepared.application.Resolved.Search.CandidateK, prepared.application.Resolved.Search.ReturnK, prepared.application.Resolved.Search.RRFK),
		BodyBudget:             prepared.application.Resolved.MCP.HardMaxInlineBytes,
		PromotionEvidenceState: string(evalcontract.NotPromotionReady),
		ProviderUsage:          usage.Aggregate,
	}
	if err := writeJSON(filepath.Join(temporary, "run-manifest.json"), manifest); err != nil {
		return RetrievalArtifactReference{}, err
	}
	if err := writeRetrievalLines(temporary, prepared.dataset, run); err != nil {
		return RetrievalArtifactReference{}, err
	}
	if err := writeJSON(filepath.Join(temporary, "provider-usage.json"), usage); err != nil {
		return RetrievalArtifactReference{}, err
	}
	audit := map[string]any{"schema_version": evalcontract.SchemaVersion, "provider_union": "parent-level diagnostic FTS-plus-collapsed-dense union", "promotion_eligible": false, "reason": "official corpus approval, reviewed confirmation evidence, and frozen promotion contract are not part of this run"}
	if err := writeJSON(filepath.Join(temporary, "implementation-audit.json"), audit); err != nil {
		return RetrievalArtifactReference{}, err
	}
	// These required filenames record that this is an execution artifact, not
	// a CorePromotionEvidence wire object. The official contract is absent.
	if err := writeJSON(filepath.Join(temporary, "promotion-contract.json"), incompletePromotionArtifact{SchemaVersion: evalcontract.SchemaVersion, Kind: "incomplete_promotion_contract", Scope: evalcontract.CoreRetrieval, PromotionEvidenceComplete: false, Reason: "official frozen promotion contract was not supplied"}); err != nil {
		return RetrievalArtifactReference{}, err
	}
	if err := writeJSON(filepath.Join(temporary, "promotion-result.json"), incompletePromotionArtifact{SchemaVersion: evalcontract.SchemaVersion, Kind: "incomplete_promotion_result", Scope: evalcontract.CoreRetrieval, PromotionEvidenceComplete: false, Reason: "official corpus and promotion inputs were not supplied"}); err != nil {
		return RetrievalArtifactReference{}, err
	}
	report := fmt.Sprintf("# Retrieval evaluation %s\n\nQueries: %d\nProvider attempts: %d\n\nThis artifact is implementation evidence only and is not promotion-ready.\n", runID, prepared.plan.QueryCount, usage.Aggregate.ProviderAttempts)
	if err := os.WriteFile(filepath.Join(temporary, "report.md"), []byte(report), 0o600); err != nil {
		return RetrievalArtifactReference{}, err
	}
	entries, err := retrievalArtifactEntries(temporary)
	if err != nil {
		return RetrievalArtifactReference{}, err
	}
	checksum, err := evalcontract.ArtifactChecksum(entries)
	if err != nil {
		return RetrievalArtifactReference{}, err
	}
	checksums := retrievalArtifactChecksums{SchemaVersion: evalcontract.SchemaVersion, Kind: "phase12_execution_artifact", Complete: true, PromotionEvidenceComplete: false, Entries: entries, EntriesChecksum: checksum}
	if err := writeJSON(filepath.Join(temporary, "artifact-checksums.json"), checksums); err != nil {
		return RetrievalArtifactReference{}, err
	}
	if err := os.Rename(temporary, target); err != nil {
		return RetrievalArtifactReference{}, err
	}
	return RetrievalArtifactReference{RunID: runID, Reference: filepath.ToSlash(filepath.Join("evaluations", runID)), Checksum: checksum}, nil
}

func removeRetrievalArtifact(ctx context.Context, prepared retrievalPrepared, artifact RetrievalArtifactReference) error {
	if !validRetrievalRunID(artifact.RunID) || artifact.Reference != filepath.ToSlash(filepath.Join("evaluations", artifact.RunID)) {
		return fmt.Errorf("invalid retrieval artifact compensation target")
	}
	root, err := prepared.raw.EvaluationArtifactsRoot(ctx)
	if err != nil {
		return err
	}
	target := filepath.Join(root, artifact.RunID)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative != artifact.RunID || filepath.Dir(target) != filepath.Clean(root) {
		return fmt.Errorf("invalid retrieval artifact compensation target")
	}
	return os.RemoveAll(target)
}

func validRetrievalRunID(value string) bool {
	if !strings.HasPrefix(value, "retrieval-") || len(value) != len("retrieval-")+24 {
		return false
	}
	for _, character := range value[len("retrieval-"):] {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func newRetrievalRunID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "retrieval-" + hex.EncodeToString(bytes), nil
}

func writeRetrievalLines(root string, dataset eval.EvaluationDataset, run eval.RetrievalEvaluationRun) error {
	files := map[string][]any{
		"per-query-trace.jsonl":             {},
		"fts-candidates.jsonl":              {},
		"dense-segment-candidates.jsonl":    {},
		"collapsed-parent-candidates.jsonl": {},
		"rrf-results.jsonl":                 {},
		"inline-body-packages.jsonl":        {},
		"per-query-metrics.jsonl":           {},
	}
	aggregates := map[eval.RetrievalVariant][]eval.CaseResult{}
	byID := map[string]evalcontract.EvaluationCase{}
	for _, item := range dataset.Cases {
		byID[item.ID] = item
	}
	for _, item := range run.Cases {
		trace, err := eval.BuildRetrievalTrace(byID[item.Case.QueryID], item)
		if err != nil {
			return err
		}
		files["per-query-trace.jsonl"] = append(files["per-query-trace.jsonl"], trace)
		for _, arm := range item.Arms {
			if len(arm.Segments) > 0 {
				files["dense-segment-candidates.jsonl"] = append(files["dense-segment-candidates.jsonl"], map[string]any{"query_id": item.Case.QueryID, "variant": arm.Ranking.Variant, "segments": arm.Segments})
			}
			if arm.Ranking.Variant == eval.VariantTargetF32 || arm.Ranking.Variant == eval.VariantServingActiveCodec {
				files["collapsed-parent-candidates.jsonl"] = append(files["collapsed-parent-candidates.jsonl"], map[string]any{"query_id": item.Case.QueryID, "variant": arm.Ranking.Variant, "ranking": arm.Ranking, "failure_stage": arm.FailureStage})
			}
		}
		for _, arm := range item.Arms {
			record := map[string]any{"query_id": item.Case.QueryID, "variant": arm.Ranking.Variant, "ranking": arm.Ranking, "failure_stage": arm.FailureStage}
			switch arm.Ranking.Variant {
			case eval.VariantFTS:
				files["fts-candidates.jsonl"] = append(files["fts-candidates.jsonl"], record)
			case eval.VariantProviderUnion:
				// It is a diagnostic set operation, not an RRF result. Its
				// survival evidence is represented by the stage trace.
			case eval.VariantTargetF32, eval.VariantServingActiveCodec:
				// Parent-level entries are recorded only in the collapse artifact.
			default:
				files["rrf-results.jsonl"] = append(files["rrf-results.jsonl"], record)
			}
			if len(arm.Packaged) > 0 {
				files["inline-body-packages.jsonl"] = append(files["inline-body-packages.jsonl"], map[string]any{"query_id": item.Case.QueryID, "variant": arm.Ranking.Variant, "packages": arm.Packaged})
			}
		}
		for _, metrics := range item.Metrics {
			files["per-query-metrics.jsonl"] = append(files["per-query-metrics.jsonl"], metrics)
			aggregates[metrics.Variant] = append(aggregates[metrics.Variant], metrics.Metrics)
		}
	}
	for name, values := range files {
		if err := writeJSONLines(filepath.Join(root, name), values); err != nil {
			return err
		}
	}
	summary := map[string]eval.Summary{}
	for variant, values := range aggregates {
		summary[string(variant)] = eval.Summarize(values, run.Plan.Ks)
	}
	if err := writeJSON(filepath.Join(root, "aggregate-metrics.json"), summary); err != nil {
		return err
	}
	slices := map[string]any{}
	for variant, value := range summary {
		slices[variant] = map[string]any{"by_language": value.ByLanguage, "by_cohort": value.ByCohort}
	}
	if err := writeJSON(filepath.Join(root, "cohort-language-report.json"), slices); err != nil {
		return err
	}
	firstLoss := map[string]map[string]int{}
	for variant, values := range aggregates {
		counts := map[string]int{}
		for _, value := range values {
			counts[string(value.FirstLoss)]++
		}
		firstLoss[string(variant)] = counts
	}
	return writeJSON(filepath.Join(root, "first-loss-report.json"), firstLoss)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func writeJSONLines(path string, values []any) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, value := range values {
		if err := encoder.Encode(value); err != nil {
			return err
		}
	}
	return nil
}

func retrievalArtifactEntries(root string) ([]evalcontract.ArtifactEntry, error) {
	names := []string{"run-manifest.json", "per-query-trace.jsonl", "fts-candidates.jsonl", "dense-segment-candidates.jsonl", "collapsed-parent-candidates.jsonl", "rrf-results.jsonl", "inline-body-packages.jsonl", "per-query-metrics.jsonl", "aggregate-metrics.json", "cohort-language-report.json", "first-loss-report.json", "provider-usage.json", "implementation-audit.json", "promotion-contract.json", "promotion-result.json", "report.md"}
	entries := make([]evalcontract.ArtifactEntry, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		media := "application/json"
		if strings.HasSuffix(name, ".jsonl") {
			media = "application/x-ndjson"
		} else if strings.HasSuffix(name, ".md") {
			media = "text/markdown"
		}
		entries = append(entries, evalcontract.ArtifactEntry{Path: name, MediaType: media, ByteSize: int64(len(data)), SHA256: hex.EncodeToString(sum[:])})
	}
	return entries, nil
}
