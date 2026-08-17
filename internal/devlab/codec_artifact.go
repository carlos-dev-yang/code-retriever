package devlab

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cidx/internal/evalcontract"
	"cidx/internal/vector"
)

type codecComparisonManifest struct {
	SchemaVersion               int                             `json:"schema_version"`
	Kind                        string                          `json:"kind"`
	RunID                       string                          `json:"run_id"`
	CreatedAt                   string                          `json:"created_at"`
	CorpusID                    string                          `json:"corpus_id"`
	CorpusManifestSHA256        string                          `json:"corpus_manifest_sha256"`
	PinnedCommit                string                          `json:"pinned_commit"`
	ContentSHA256               string                          `json:"content_sha256"`
	DatasetSHA256               string                          `json:"dataset_sha256"`
	IndexGeneration             int64                           `json:"index_generation"`
	IndexManifestSHA256         string                          `json:"index_manifest_sha256"`
	IndexProfile                string                          `json:"index_profile"`
	SourceProfile               string                          `json:"source_profile"`
	VectorSpaceProfile          string                          `json:"vector_space_profile"`
	ActiveVectorStorageProfile  string                          `json:"active_vector_storage_profile"`
	DocumentBankSHA256          string                          `json:"document_bank_sha256"`
	RawDocumentInputs           int                             `json:"raw_document_inputs"`
	LocallyEncodedInt8Documents int                             `json:"locally_encoded_int8_documents"`
	SourceDimensions            int                             `json:"source_dimensions"`
	ServingDimensions           int                             `json:"serving_dimensions"`
	ProductionStorageCodec      string                          `json:"production_storage_codec"`
	ActiveBinaryCodecID         string                          `json:"active_binary_codec_id"`
	CandidateInt8CodecID        string                          `json:"candidate_int8_codec_id"`
	Depth                       int                             `json:"depth"`
	Ks                          []int                           `json:"ks"`
	UsesFTS                     bool                            `json:"uses_fts"`
	UsesRRF                     bool                            `json:"uses_rrf"`
	QueryVectorReuseArms        int                             `json:"query_vector_reuse_arms"`
	QueryVectorPersisted        bool                            `json:"query_vector_persisted"`
	DocumentProviderOperations  int                             `json:"document_provider_operations"`
	ProviderUsage               retrievalProviderUsageAggregate `json:"provider_usage"`
	ExperimentSeriesID          string                          `json:"experiment_series_id"`
	SeriesQueryOperations       int                             `json:"series_query_operations_planned"`
	AuthorizationReference      string                          `json:"authorization_reference"`
	USDCap                      float64                         `json:"usd_cap"`
	PricingTableIdentity        string                          `json:"pricing_table_identity"`
	USDPerMillionTokens         float64                         `json:"usd_per_million_tokens"`
	PlannedMaximumCostUSD       float64                         `json:"planned_maximum_cost_usd"`
	ActualAccountedCostUSD      float64                         `json:"actual_accounted_cost_usd"`
	TokenAccountingComplete     bool                            `json:"token_accounting_complete"`
	CodeCommit                  string                          `json:"code_commit"`
	SourceModified              string                          `json:"source_modified"`
	EvaluationExecutableSHA256  string                          `json:"evaluation_executable_sha256"`
	EvidenceClass               string                          `json:"evidence_class"`
	PromotionEligible           bool                            `json:"promotion_eligible"`
}

func publishCodecComparisonArtifact(ctx context.Context, prepared codecComparisonPrepared, run CodecComparisonRun, usage retrievalProviderUsage) (RetrievalArtifactReference, error) {
	if err := validateCodecComparisonRun(prepared, run); err != nil {
		return RetrievalArtifactReference{}, err
	}
	if err := usage.ValidateOperations(prepared.base, prepared.base.dataset); err != nil {
		return RetrievalArtifactReference{}, err
	}
	runID, err := newCodecComparisonRunID()
	if err != nil {
		return RetrievalArtifactReference{}, err
	}
	root, err := prepared.base.raw.EvaluationArtifactsRoot(ctx)
	if err != nil {
		return RetrievalArtifactReference{}, err
	}
	target := filepath.Join(root, runID)
	if _, err := os.Lstat(target); err == nil {
		return RetrievalArtifactReference{}, fmt.Errorf("codec comparison artifact already exists")
	} else if !os.IsNotExist(err) {
		return RetrievalArtifactReference{}, err
	}
	temporary, err := os.MkdirTemp(root, ".codec-comparison-")
	if err != nil {
		return RetrievalArtifactReference{}, err
	}
	defer os.RemoveAll(temporary)
	actualCost := 0.0
	if usage.Aggregate.ObservedTotalTokens != nil {
		actualCost = float64(*usage.Aggregate.ObservedTotalTokens) * prepared.plan.USDPerMillionTokens / 1_000_000
	}
	resolved := prepared.base.application.Resolved
	manifest := codecComparisonManifest{
		SchemaVersion: evalcontract.SchemaVersion, Kind: "cidx.codec_comparison.v1", RunID: runID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), CorpusID: prepared.plan.CorpusID,
		CorpusManifestSHA256: prepared.plan.CorpusManifestSHA256, PinnedCommit: prepared.plan.PinnedCommit,
		ContentSHA256: prepared.plan.ContentSHA256, DatasetSHA256: prepared.plan.DatasetSHA256,
		IndexGeneration: prepared.plan.IndexGeneration, IndexManifestSHA256: prepared.plan.IndexManifestSHA256,
		IndexProfile: string(resolved.Profiles.Fingerprints.Index), SourceProfile: string(resolved.Profiles.Fingerprints.Source),
		VectorSpaceProfile: string(resolved.Profiles.Fingerprints.VectorSpace), ActiveVectorStorageProfile: string(resolved.Profiles.Fingerprints.VectorStorage),
		DocumentBankSHA256: prepared.base.documentBankFingerprint, RawDocumentInputs: prepared.plan.RawDocumentInputs,
		LocallyEncodedInt8Documents: prepared.plan.RawDocumentInputs,
		SourceDimensions:            prepared.plan.SourceDimensions, ServingDimensions: prepared.plan.ServingDimensions,
		ProductionStorageCodec: prepared.plan.ProductionStorageCodec, ActiveBinaryCodecID: vector.BinaryCodecID,
		CandidateInt8CodecID: vector.Int8CodecID, Depth: prepared.plan.Depth, Ks: append([]int(nil), prepared.plan.Ks...),
		UsesFTS: false, UsesRRF: false, QueryVectorReuseArms: 3, QueryVectorPersisted: false,
		DocumentProviderOperations: 0, ProviderUsage: usage.Aggregate,
		ExperimentSeriesID: prepared.plan.ExperimentSeriesID, SeriesQueryOperations: prepared.plan.SeriesQueryOperationsPlanned,
		AuthorizationReference: prepared.plan.AuthorizationReference, USDCap: prepared.plan.USDCap,
		PricingTableIdentity: prepared.plan.PricingTableIdentity, USDPerMillionTokens: prepared.plan.USDPerMillionTokens,
		PlannedMaximumCostUSD: prepared.plan.PlannedMaximumCostUSD, ActualAccountedCostUSD: actualCost,
		TokenAccountingComplete: usage.Aggregate.TokenAccountingComplete, CodeCommit: prepared.plan.CodeCommit,
		SourceModified: prepared.plan.SourceModified, EvaluationExecutableSHA256: prepared.plan.EvaluationExecutableSHA256,
		EvidenceClass: "CALIBRATION_CODEC_DIAGNOSTIC", PromotionEligible: false,
	}
	if err := writeJSON(filepath.Join(temporary, "run-manifest.json"), manifest); err != nil {
		return RetrievalArtifactReference{}, err
	}
	if err := writeCodecComparisonLines(temporary, run); err != nil {
		return RetrievalArtifactReference{}, err
	}
	if err := writeJSON(filepath.Join(temporary, "aggregate-metrics.json"), run.AggregateMetrics); err != nil {
		return RetrievalArtifactReference{}, err
	}
	cohorts := map[string]any{}
	for representation, summary := range run.AggregateMetrics {
		cohorts[representation] = map[string]any{"by_language": summary.ByLanguage, "by_cohort": summary.ByCohort}
	}
	if err := writeJSON(filepath.Join(temporary, "cohort-language-report.json"), cohorts); err != nil {
		return RetrievalArtifactReference{}, err
	}
	if err := writeJSON(filepath.Join(temporary, "fidelity-summary.json"), run.FidelitySummaries); err != nil {
		return RetrievalArtifactReference{}, err
	}
	if err := writeJSON(filepath.Join(temporary, "provider-usage.json"), usage); err != nil {
		return RetrievalArtifactReference{}, err
	}
	report := fmt.Sprintf("# Isolated codec comparison %s\n\nQueries: %d\nDepth: %d\nRepresentations: target f32, active binary, candidate int8\nProvider attempts: %d\nFTS: false\nRRF: false\n\nThis is a calibration diagnostic, not promotion evidence.\n", runID, prepared.plan.QueryCount, prepared.plan.Depth, usage.Aggregate.ProviderAttempts)
	if err := os.WriteFile(filepath.Join(temporary, "report.md"), []byte(report), 0o600); err != nil {
		return RetrievalArtifactReference{}, err
	}
	entries, err := codecComparisonArtifactEntries(temporary)
	if err != nil {
		return RetrievalArtifactReference{}, err
	}
	checksum, err := evalcontract.ArtifactChecksum(entries)
	if err != nil {
		return RetrievalArtifactReference{}, err
	}
	checksums := retrievalArtifactChecksums{SchemaVersion: evalcontract.SchemaVersion, Kind: "codec_comparison_artifact", Complete: true, PromotionEvidenceComplete: false, Entries: entries, EntriesChecksum: checksum}
	if err := writeJSON(filepath.Join(temporary, "artifact-checksums.json"), checksums); err != nil {
		return RetrievalArtifactReference{}, err
	}
	if err := os.Rename(temporary, target); err != nil {
		return RetrievalArtifactReference{}, err
	}
	return RetrievalArtifactReference{RunID: runID, Reference: filepath.ToSlash(filepath.Join("evaluations", runID)), Checksum: checksum}, nil
}

func writeCodecComparisonLines(root string, run CodecComparisonRun) error {
	parents, segments, metrics, fidelity := []any{}, []any{}, []any{}, []any{}
	for _, item := range run.Cases {
		for _, value := range item.Rankings {
			parents = append(parents, map[string]any{"query_id": item.QueryID, "query_vector_sha256": item.QueryVectorSHA256, "variant": value.Variant, "hits": value.Hits})
		}
		for _, value := range item.Segments {
			segments = append(segments, map[string]any{"query_id": item.QueryID, "query_vector_sha256": item.QueryVectorSHA256, "representation": value.Representation, "segments": value.Segments})
		}
		for _, value := range item.Metrics {
			metrics = append(metrics, map[string]any{"query_id": item.QueryID, "representation": value.Representation, "metrics": value.Metrics})
		}
		fidelity = append(fidelity, map[string]any{"query_id": item.QueryID, "binary": item.BinaryFidelity, "int8": item.Int8Fidelity})
	}
	for name, values := range map[string][]any{
		"parent-rankings.jsonl": parents, "segment-rankings.jsonl": segments,
		"per-query-metrics.jsonl": metrics, "codec-fidelity.jsonl": fidelity,
	} {
		if err := writeJSONLines(filepath.Join(root, name), values); err != nil {
			return err
		}
	}
	return nil
}

func codecComparisonArtifactEntries(root string) ([]evalcontract.ArtifactEntry, error) {
	names := []string{"run-manifest.json", "parent-rankings.jsonl", "segment-rankings.jsonl", "per-query-metrics.jsonl", "codec-fidelity.jsonl", "aggregate-metrics.json", "cohort-language-report.json", "fidelity-summary.json", "provider-usage.json", "report.md"}
	entries := make([]evalcontract.ArtifactEntry, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		mediaType := "application/json"
		if strings.HasSuffix(name, ".jsonl") {
			mediaType = "application/x-ndjson"
		} else if strings.HasSuffix(name, ".md") {
			mediaType = "text/markdown"
		}
		entries = append(entries, evalcontract.ArtifactEntry{Path: name, MediaType: mediaType, ByteSize: int64(len(data)), SHA256: hex.EncodeToString(sum[:])})
	}
	return entries, nil
}

func newCodecComparisonRunID() (string, error) {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "codec-" + hex.EncodeToString(value), nil
}

func validCodecComparisonRunID(value string) bool {
	if !strings.HasPrefix(value, "codec-") || len(value) != len("codec-")+24 {
		return false
	}
	for _, character := range value[len("codec-"):] {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func removeCodecComparisonArtifact(ctx context.Context, prepared codecComparisonPrepared, artifact RetrievalArtifactReference) error {
	if !validCodecComparisonRunID(artifact.RunID) || artifact.Reference != filepath.ToSlash(filepath.Join("evaluations", artifact.RunID)) {
		return fmt.Errorf("invalid codec comparison compensation target")
	}
	root, err := prepared.base.raw.EvaluationArtifactsRoot(ctx)
	if err != nil {
		return err
	}
	target := filepath.Join(root, artifact.RunID)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative != artifact.RunID || filepath.Dir(target) != filepath.Clean(root) {
		return fmt.Errorf("invalid codec comparison compensation target")
	}
	return os.RemoveAll(target)
}
