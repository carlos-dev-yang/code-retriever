package devlab

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cidx/internal/eval"
	"cidx/internal/evalcontract"
)

// RetrievalArtifactReference is the durable, vector-free identity returned
// after an explicitly applied evaluation has been atomically published.
type RetrievalArtifactReference struct {
	RunID, Reference, Checksum string
}

type retrievalProviderUsage struct {
	QueryProviderCalls int `json:"query_provider_calls"`
	SuccessfulCalls    int `json:"successful_calls"`
	FailedCalls        int `json:"failed_calls"`
	QueryTokens        int `json:"query_tokens"`
}

type retrievalArtifactManifest struct {
	SchemaVersion          int                    `json:"schema_version"`
	RunID                  string                 `json:"run_id"`
	CreatedAt              string                 `json:"created_at"`
	CorpusID               string                 `json:"corpus_id"`
	CorpusManifestSHA256   string                 `json:"corpus_manifest_sha256"`
	PinnedCommit           string                 `json:"pinned_commit"`
	ContentSHA256          string                 `json:"content_sha256"`
	IndexGeneration        int64                  `json:"index_generation"`
	IndexManifestSHA256    string                 `json:"index_manifest_sha256"`
	DatasetSHA256          string                 `json:"dataset_sha256"`
	QueryIDs               []string               `json:"query_ids"`
	ServingProfile         string                 `json:"serving_profile"`
	SourceProfile          string                 `json:"source_profile"`
	VectorSpaceProfile     string                 `json:"vector_space_profile"`
	RawDocumentInputs      int                    `json:"raw_document_inputs"`
	CandidatePolicy        string                 `json:"candidate_policy"`
	BodyBudget             int                    `json:"body_budget"`
	PromotionEvidenceState string                 `json:"promotion_evidence_state"`
	ProviderUsage          retrievalProviderUsage `json:"provider_usage"`
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
	sort.Strings(queryIDs)
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
		ProviderUsage:          usage,
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
	report := fmt.Sprintf("# Retrieval evaluation %s\n\nQueries: %d\nProvider calls: %d\n\nThis artifact is implementation evidence only and is not promotion-ready.\n", runID, prepared.plan.QueryCount, usage.QueryProviderCalls)
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
