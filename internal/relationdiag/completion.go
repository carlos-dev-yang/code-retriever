package relationdiag

// Stage A consumes immutable active-int8 evaluation observations. It is
// provider-free and keeps labels outside candidate construction.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cidx/internal/buildinfo"
	"cidx/internal/eval"
	"cidx/internal/evalcontract"
	"cidx/internal/search"
	"cidx/internal/store"
)

const (
	semanticCollapsePolicyID      = "max-active-int8-segment-score-per-semantic-parent-v1"
	semanticNormalizationPolicyID = "active-int8-score-span-normalization-v1"
	completionPolicyID            = "relation-evidence-completion-stage-a-v1"
	completionEvidenceClass       = "RELATION_CALIBRATION_POOL_BUILDING"
	hintDisclosureSchemaID        = "relation-hint-disclosure-v1"
)

var closureCountBudgetGrid = []int{1, 2, 4}
var closureByteBudgetGrid = []int{1024, 2048, 4096}
var hintCountBudgetGrid = []int{4, 8, 16, 32}
var hintByteBudgetGrid = []int{512, 1024, 2048, 4096}

type CompletionRequest struct {
	RunID, EvaluationRoot, GraphDirectory, RetrievalDirectory, DatasetPath string
	Parents                                                                store.SemanticParentSnapshot
	Reproof                                                                func(context.Context) (store.SemanticParentSnapshot, error)
}
type CompletionResult struct {
	RunID     string `json:"run_id"`
	Reference string `json:"reference"`
	Queries   int    `json:"queries"`
}

type retrievalCompletionManifest struct {
	SchemaVersion        int             `json:"schema_version"`
	RunID                string          `json:"run_id"`
	CreatedAt            string          `json:"created_at"`
	CorpusID             string          `json:"corpus_id"`
	CorpusManifestSHA256 string          `json:"corpus_manifest_sha256"`
	PinnedCommit         string          `json:"pinned_commit"`
	ContentSHA256        string          `json:"content_sha256"`
	IndexGeneration      int64           `json:"index_generation"`
	IndexManifestSHA256  string          `json:"index_manifest_sha256"`
	DatasetSHA256        string          `json:"dataset_sha256"`
	DatasetSourceSHA256  string          `json:"dataset_source_sha256"`
	QueryIDs             []string        `json:"query_ids"`
	ServingProfile       string          `json:"serving_profile"`
	SourceProfile        string          `json:"source_profile"`
	VectorSpaceProfile   string          `json:"vector_space_profile"`
	RawDocumentInputs    int             `json:"raw_document_inputs"`
	CandidatePolicy      string          `json:"candidate_policy"`
	BodyBudget           int             `json:"body_budget"`
	PromotionState       string          `json:"promotion_evidence_state"`
	ProviderUsage        json.RawMessage `json:"provider_usage"`
	Experiment           json.RawMessage `json:"experiment,omitempty"`
	experimentHeader     completionExperimentHeader
}
type retrievalCompletionChecksums struct {
	SchemaVersion             int                          `json:"schema_version"`
	Kind                      string                       `json:"kind"`
	Complete                  bool                         `json:"complete"`
	PromotionEvidenceComplete bool                         `json:"promotion_evidence_complete"`
	Entries                   []evalcontract.ArtifactEntry `json:"entries"`
	EntriesSum                string                       `json:"entries_checksum"`
}
type completionSegmentRow struct {
	QueryID  string                        `json:"query_id"`
	Variant  string                        `json:"variant"`
	Segments []search.EvaluationSegmentHit `json:"segments"`
}
type completionCollapsedRow struct {
	QueryID string `json:"query_id"`
	Variant string `json:"variant"`
	Ranking struct {
		QueryID           string              `json:"query_id"`
		QueryVectorSHA256 string              `json:"query_vector_sha256"`
		Hits              []eval.RetrievalHit `json:"hits"`
	} `json:"ranking"`
	FailureStage string `json:"failure_stage"`
}
type completionExperimentHeader struct {
	ExperimentSeriesID           string         `json:"experiment_series_id"`
	EvidenceClass                string         `json:"evidence_class"`
	PromotionEligible            bool           `json:"promotion_eligible"`
	LabelState                   string         `json:"label_state"`
	QueryExecutionMode           string         `json:"query_execution_mode"`
	SeriesQueryOperationsPlanned int            `json:"series_query_operations_planned"`
	CorpusCleanVerified          bool           `json:"corpus_clean_verified"`
	FTSMode                      string         `json:"fts_mode"`
	BuildInfo                    buildinfo.Info `json:"build_info"`
	EvaluationExecutableSHA256   string         `json:"evaluation_executable_sha256"`
	SourceDimensions             int            `json:"source_dimensions"`
	ServingDimensions            int            `json:"serving_dimensions"`
	StorageCodec                 string         `json:"storage_codec"`
	DocumentProviderOperations   int            `json:"document_provider_operations"`
	ReusedQueryVectors           int            `json:"reused_query_vectors"`
	ReusedDenseRankings          int            `json:"reused_dense_rankings"`
	QueryVectorPersisted         bool           `json:"query_vector_persisted"`
	QueryVectorSHA256Recorded    bool           `json:"query_vector_sha256_recorded"`
	Queries                      []struct {
		QueryID                          string `json:"query_id"`
		TextSHA256                       string `json:"text_sha256"`
		ServingActiveSegmentObservations int    `json:"serving_active_segment_observations"`
	} `json:"queries"`
}

// Persisted structures use explicit portable snake_case keys.
type semanticParentScore struct {
	QueryID                            string                      `json:"query_id"`
	ParentID                           string                      `json:"parent_id"`
	Path                               string                      `json:"path"`
	IndexedSHA256                      string                      `json:"indexed_sha256"`
	QualifiedSymbol                    string                      `json:"qualified_symbol"`
	StartByte                          int                         `json:"start_byte"`
	EndByte                            int                         `json:"end_byte"`
	GlobalRank                         int                         `json:"global_rank"`
	TieStartRank                       int                         `json:"tie_start_rank"`
	TieEndRank                         int                         `json:"tie_end_rank"`
	ParentCount                        int                         `json:"parent_count"`
	RankPercentileNumerator            int                         `json:"rank_percentile_numerator"`
	RankPercentileDenominator          int                         `json:"rank_percentile_denominator"`
	NativeScore                        float64                     `json:"native_active_int8_score"`
	WinningSegment                     search.EvaluationSegmentHit `json:"winning_segment"`
	ExactTieOrderIndependentlyVerified bool                        `json:"exact_tie_order_independently_verified"`
}
type semanticDistribution struct {
	ParentCount int     `json:"parent_count"`
	Minimum     float64 `json:"minimum_native_score"`
	Maximum     float64 `json:"maximum_native_score"`
	Mean        float64 `json:"mean_native_score"`
	StdDev      float64 `json:"population_stddev_native_score"`
	Span        float64 `json:"score_span"`
}
type semanticEndpointFeature struct {
	QueryID                       string                   `json:"query_id"`
	RelationID                    string                   `json:"representative_relation_id"`
	SupportingRelationIDs         []string                 `json:"supporting_relation_ids"`
	SupportingCanonicalEdgeIDs    []string                 `json:"supporting_canonical_edge_ids"`
	SupportingViews               []endpointSupportingView `json:"supporting_views"`
	RepresentativeRule            string                   `json:"representative_rule"`
	Direction                     Direction                `json:"direction"`
	RelationKind                  RelationKind             `json:"relation_kind"`
	StructuralTier                StructuralTier           `json:"structural_tier"`
	AnchorParentID                string                   `json:"anchor_parent_id"`
	EndpointParentID              string                   `json:"endpoint_parent_id"`
	EndpointGlobalRank            int                      `json:"endpoint_global_rank"`
	EndpointTieStartRank          int                      `json:"endpoint_tie_start_rank"`
	EndpointTieEndRank            int                      `json:"endpoint_tie_end_rank"`
	EndpointPercentileNumerator   int                      `json:"endpoint_percentile_numerator"`
	EndpointPercentileDenominator int                      `json:"endpoint_percentile_denominator"`
	EndpointPercentileStart       int                      `json:"endpoint_percentile_start_numerator"`
	EndpointPercentileEnd         int                      `json:"endpoint_percentile_end_numerator"`
	EndpointNativeScore           float64                  `json:"endpoint_native_score"`
	AnchorNativeScore             float64                  `json:"anchor_native_score"`
	EndpointAnchorGap             float64                  `json:"endpoint_minus_anchor_gap"`
	NormalizedEndpointAnchorGap   *float64                 `json:"normalized_endpoint_anchor_gap,omitempty"`
	BestCandidateScore            float64                  `json:"query_wide_best_graph_candidate_score"`
	RunnerUpCandidateScore        *float64                 `json:"query_wide_runner_up_graph_candidate_score,omitempty"`
	BestRunnerUpGap               *float64                 `json:"query_wide_best_runner_up_gap,omitempty"`
	NormalizedBestRunnerUpGap     *float64                 `json:"normalized_query_wide_best_runner_up_gap,omitempty"`
	AmbiguityDegenerate           bool                     `json:"ambiguity_degenerate"`
	NormalizationDegenerate       bool                     `json:"normalization_degenerate"`
	AlreadyPrimaryTop5            bool                     `json:"already_primary_top5"`
	AlreadyDenseTop20             bool                     `json:"already_dense_top20"`
	AbsentFromDenseTop20          bool                     `json:"absent_from_dense_top20"`
	PrimaryTop5TieStatus          string                   `json:"primary_top5_tie_status"`
	DenseTop20TieStatus           string                   `json:"dense_top20_tie_status"`
	EndpointScoreObserved         bool                     `json:"endpoint_score_observed"`
	SemanticOrdinal               int                      `json:"semantic_ordinal"`
	ScoreScale                    string                   `json:"score_scale"`
	QueryWideTopOneScore          float64                  `json:"query_wide_top_one_score"`
	QueryWideTopTwoScore          *float64                 `json:"query_wide_top_two_score,omitempty"`
	QueryWideTopOneTwoGap         *float64                 `json:"query_wide_top_one_two_gap,omitempty"`
	QueryWideTopOneTwoTied        bool                     `json:"query_wide_top_one_two_tied"`
	Distribution                  semanticDistribution     `json:"query_score_distribution"`
}
type closureCandidate struct {
	QueryID                    string         `json:"query_id"`
	PrimaryParentID            string         `json:"primary_parent_id"`
	TargetParentID             string         `json:"target_parent_id"`
	RelationID                 string         `json:"relation_id"`
	RoleClass                  string         `json:"role_class"`
	StructuralTier             StructuralTier `json:"structural_tier"`
	BodyBytes                  int            `json:"body_bytes"`
	BodyComplete               bool           `json:"body_complete"`
	OmissionReason             string         `json:"omission_reason,omitempty"`
	Ordinal                    int            `json:"ordinal"`
	CumulativeBodyBytes        int            `json:"cumulative_body_bytes"`
	CountBudgetEligible        []int          `json:"count_budget_eligible"`
	ByteBudgetEligible         []int          `json:"byte_budget_eligible"`
	PrimaryOrdinal             int            `json:"primary_ordinal"`
	PrimaryCountBudgetEligible []int          `json:"primary_count_budget_eligible"`
	RequestOrdinal             int            `json:"request_ordinal"`
	RequestCumulativeBodyBytes int            `json:"request_cumulative_body_bytes"`
	RequestCountBudgetEligible []int          `json:"request_count_budget_eligible"`
}
type relationHint struct {
	QueryID                    string         `json:"query_id"`
	RelationID                 string         `json:"representative_relation_id"`
	SupportingRelationIDs      []string       `json:"supporting_relation_ids"`
	SupportingCanonicalEdgeIDs []string       `json:"supporting_canonical_edge_ids"`
	Kind                       RelationKind   `json:"relation_kind"`
	Direction                  Direction      `json:"direction"`
	StructuralTier             StructuralTier `json:"structural_tier"`
	SourcePath                 string         `json:"source_path"`
	SourceSHA256               string         `json:"source_sha256"`
	SourceStartByte            int            `json:"source_start_byte"`
	SourceEndByte              int            `json:"source_end_byte"`
	SourceStartLine            int            `json:"source_start_line"`
	SourceEndLine              int            `json:"source_end_line"`
	TargetSymbol               string         `json:"target_symbol"`
	TargetQualified            string         `json:"target_qualified_symbol"`
	TargetPath                 string         `json:"target_path"`
	TargetSHA256               string         `json:"target_sha256"`
	TargetStartByte            int            `json:"target_start_byte"`
	TargetEndByte              int            `json:"target_end_byte"`
	TargetStartLine            int            `json:"target_start_line"`
	TargetEndLine              int            `json:"target_end_line"`
	AlreadyPrimary             bool           `json:"already_primary"`
	AlreadyDense               bool           `json:"already_dense_top20"`
	OccurrenceCount            int            `json:"underlying_occurrence_count"`
	SupportingViewCount        int            `json:"supporting_frontier_view_count"`
	SemanticOrdinal            int            `json:"semantic_ordinal"`
	SerializedBytes            int            `json:"serialized_hint_bytes"`
	DisclosureSchema           string         `json:"disclosure_schema"`
	DisclosureSHA256           string         `json:"disclosure_sha256"`
	CumulativeBytes            int            `json:"cumulative_hint_bytes"`
	CountBudgetEligible        []int          `json:"count_budget_eligible"`
	ByteBudgetEligible         []int          `json:"byte_budget_eligible"`
	OmissionStatus             string         `json:"omission_status"`
}
type endpointSupportingView struct {
	RelationID      string         `json:"relation_id"`
	CanonicalEdgeID string         `json:"canonical_edge_id"`
	AnchorParentID  string         `json:"anchor_parent_id"`
	RelationKind    RelationKind   `json:"relation_kind"`
	Direction       Direction      `json:"direction"`
	StructuralTier  StructuralTier `json:"structural_tier"`
	AnchorScore     float64        `json:"anchor_native_score"`
	EndpointGap     float64        `json:"endpoint_minus_anchor_gap"`
}

// hintDisclosure is the exact body-free object used for byte budgets. The
// evidence-only support and admission fields on relationHint are deliberately
// outside this assistant-facing payload.
type hintDisclosure struct {
	Schema          string         `json:"schema"`
	RelationID      string         `json:"relation_id"`
	Kind            RelationKind   `json:"relation_kind"`
	Direction       Direction      `json:"direction"`
	StructuralTier  StructuralTier `json:"structural_tier"`
	SourcePath      string         `json:"source_path"`
	SourceSHA256    string         `json:"source_sha256"`
	SourceStartLine int            `json:"source_start_line"`
	SourceEndLine   int            `json:"source_end_line"`
	TargetSymbol    string         `json:"target_symbol"`
	TargetQualified string         `json:"target_qualified_symbol"`
	TargetPath      string         `json:"target_path"`
	TargetSHA256    string         `json:"target_sha256"`
	TargetStartLine int            `json:"target_start_line"`
	TargetEndLine   int            `json:"target_end_line"`
}
type completionParentLines struct {
	StartLine int
	EndLine   int
}
type completionDatasetBinding struct {
	CorpusID     string
	Fingerprint  string
	SourceSHA256 string
	QueryIDs     []string
	Features     map[string]queryFeatures
	TextSHA256   map[string]string
}

func Complete(ctx context.Context, request CompletionRequest) (CompletionResult, error) {
	if err := validateCompletionRequest(request); err != nil {
		return CompletionResult{}, err
	}
	if !cleanKnownRelationBuild(buildinfo.Current()) {
		return CompletionResult{}, fmt.Errorf("relation completion requires a clean known build")
	}
	policyFingerprint, err := completionPolicyFingerprint()
	if err != nil {
		return CompletionResult{}, err
	}
	executable, err := os.Executable()
	if err != nil {
		return CompletionResult{}, err
	}
	executableSHA, err := fileSHA256(executable)
	if err != nil {
		return CompletionResult{}, err
	}
	if err := verifyChecksums(request.GraphDirectory, []string{"graph-manifest.json", "relations.db", "resolution-summary.json"}); err != nil {
		return CompletionResult{}, fmt.Errorf("relation graph artifact checksum verification: %w", err)
	}
	graph, err := loadGraphManifest(filepath.Join(request.GraphDirectory, "graph-manifest.json"))
	if err != nil {
		return CompletionResult{}, err
	}
	parents, err := ParentInventory(request.Parents.Parents)
	if err != nil {
		return CompletionResult{}, err
	}
	if err := validateGraphBinding(request.GraphDirectory, graph, parents, request.Parents); err != nil {
		return CompletionResult{}, err
	}
	retrieval, retrievalChecksum, err := loadRetrievalCompletionArtifact(request.RetrievalDirectory)
	if err != nil {
		return CompletionResult{}, err
	}
	dataset, err := loadCompletionDataset(request.DatasetPath, retrieval.DatasetSHA256)
	if err != nil {
		return CompletionResult{}, err
	}
	if err := validateCompletionBinding(graph, retrieval, dataset); err != nil {
		return CompletionResult{}, err
	}
	byID, byHit := parentMaps(parents)
	segments, collapsed, err := loadActiveInt8Scores(request.RetrievalDirectory, retrieval.QueryIDs, retrieval.RawDocumentInputs, completionSegmentCounts(retrieval.experimentHeader), byHit)
	if err != nil {
		return CompletionResult{}, err
	}
	scores, hashes := map[string][]semanticParentScore{}, map[string]string{}
	for _, queryID := range retrieval.QueryIDs {
		values, err := collapseActiveScores(queryID, segments[queryID], byHit, byID)
		if err != nil {
			return CompletionResult{}, err
		}
		if err := validateCollapsedTopK(values, collapsed[queryID], byHit); err != nil {
			return CompletionResult{}, err
		}
		scores[queryID], hashes[queryID] = values, collapsed[queryID].Ranking.QueryVectorSHA256
	}
	db, err := openImmutableGraph(filepath.Join(request.GraphDirectory, "relations.db"))
	if err != nil {
		return CompletionResult{}, err
	}
	defer db.Close()
	if err := graphIntegrity(ctx, db); err != nil {
		return CompletionResult{}, err
	}
	parents, err = loadGraphParentTraits(ctx, db, parents)
	if err != nil {
		return CompletionResult{}, err
	}
	byID, byHit = parentMaps(parents)
	_ = byID
	lines, err := completionParentLineMap(request.Parents)
	if err != nil {
		return CompletionResult{}, err
	}
	features, closures, hints, traces, err := completionEvidence(ctx, db, retrieval.QueryIDs, dataset.Features, scores, collapsed, parentsByID(parents), lines, byHit)
	if err != nil {
		return CompletionResult{}, err
	}
	target := filepath.Join(request.EvaluationRoot, request.RunID)
	if _, err := os.Lstat(target); err == nil {
		return CompletionResult{}, fmt.Errorf("relation completion artifact already exists")
	} else if !os.IsNotExist(err) {
		return CompletionResult{}, err
	}
	temporary, err := os.MkdirTemp(request.EvaluationRoot, ".relation-completion-")
	if err != nil {
		return CompletionResult{}, err
	}
	defer os.RemoveAll(temporary)
	if err := writeCompletionArtifacts(temporary, graph, retrieval, retrievalChecksum, dataset, hashes, scores, features, closures, hints, traces, policyFingerprint, executableSHA); err != nil {
		return CompletionResult{}, err
	}
	if err := writeChecksums(temporary); err != nil {
		return CompletionResult{}, err
	}
	if err := reproveCompletion(ctx, request, graph, retrieval, retrievalChecksum, dataset); err != nil {
		return CompletionResult{}, err
	}
	if err := os.Rename(temporary, target); err != nil {
		return CompletionResult{}, err
	}
	return CompletionResult{RunID: request.RunID, Reference: filepath.ToSlash(filepath.Join("evaluations", request.RunID)), Queries: len(retrieval.QueryIDs)}, nil
}

func parentsByID(parents []Parent) map[string]Parent {
	result := map[string]Parent{}
	for _, parent := range parents {
		result[parent.ID] = parent
	}
	return result
}
func completionParentLineMap(snapshot store.SemanticParentSnapshot) (map[string]completionParentLines, error) {
	result := map[string]completionParentLines{}
	for _, stored := range snapshot.Parents {
		parent, err := ParentFromStored(stored)
		if err != nil || stored.StartLine < 1 || stored.EndLine < stored.StartLine {
			return nil, fmt.Errorf("invalid completion parent line range")
		}
		result[parent.ID] = completionParentLines{StartLine: stored.StartLine, EndLine: stored.EndLine}
	}
	return result, nil
}
func validateCompletionRequest(request CompletionRequest) error {
	if !strings.HasPrefix(request.RunID, "relation-completion-") || !validRelative(request.RunID) || request.EvaluationRoot == "" || request.GraphDirectory == "" || request.RetrievalDirectory == "" || request.DatasetPath == "" || request.Parents.Generation < 1 || request.Parents.ManifestSHA256 == "" || request.Reproof == nil {
		return fmt.Errorf("invalid relation completion request")
	}
	return nil
}

func completionPolicyFingerprint() (string, error) {
	return canonicalHash(map[string]any{"policy": completionPolicyID, "metadata_policy": MetadataPolicyID, "query_feature_formula": "symbol-classify-query-normalized-identifiers-v1", "frontier_policy": AnchorFrontierCapOnlyPolicyID, "semantic_collapse": semanticCollapsePolicyID, "normalization": semanticNormalizationPolicyID, "closure_role_family": []OccurrenceRole{TypeValueParameterRole, TypeReturnRole, TypeParameterRole, TypeHeritageRole, TypeFieldRole, TypeAliasRole, TypeArgumentRole, TypeOtherRole}, "closure_count_budget_grid": closureCountBudgetGrid, "closure_byte_budget_grid": closureByteBudgetGrid, "hint_count_budget_grid": hintCountBudgetGrid, "hint_byte_budget_grid": hintByteBudgetGrid, "hint_disclosure_schema": hintDisclosureSchemaID})
}

func reproveCompletion(ctx context.Context, request CompletionRequest, originalGraph GraphManifest, originalRetrieval retrievalCompletionManifest, retrievalChecksum string, originalDataset completionDatasetBinding) error {
	refreshed, err := request.Reproof(ctx)
	if err != nil {
		return err
	}
	if refreshed.Generation != request.Parents.Generation || refreshed.ManifestSHA256 != request.Parents.ManifestSHA256 {
		return fmt.Errorf("NON_REPRODUCIBLE_RUN")
	}
	parents, err := ParentInventory(refreshed.Parents)
	if err != nil {
		return err
	}
	if err := verifyChecksums(request.GraphDirectory, []string{"graph-manifest.json", "relations.db", "resolution-summary.json"}); err != nil {
		return fmt.Errorf("relation graph artifact changed during completion: %w", err)
	}
	graph, err := loadGraphManifest(filepath.Join(request.GraphDirectory, "graph-manifest.json"))
	if err != nil {
		return err
	}
	if graph.LogicalGraphSHA256 != originalGraph.LogicalGraphSHA256 || graph.DatabaseSHA256 != originalGraph.DatabaseSHA256 || validateGraphBinding(request.GraphDirectory, graph, parents, refreshed) != nil {
		return fmt.Errorf("relation graph binding drift during completion")
	}
	retrieval, checksum, err := loadRetrievalCompletionArtifact(request.RetrievalDirectory)
	if err != nil {
		return err
	}
	if checksum != retrievalChecksum || retrieval.RunID != originalRetrieval.RunID || retrieval.DatasetSHA256 != originalRetrieval.DatasetSHA256 {
		return fmt.Errorf("retrieval artifact drift during completion")
	}
	dataset, err := loadCompletionDataset(request.DatasetPath, retrieval.DatasetSHA256)
	if err != nil {
		return err
	}
	if dataset.Fingerprint != originalDataset.Fingerprint || dataset.SourceSHA256 != originalDataset.SourceSHA256 || validateCompletionBinding(graph, retrieval, dataset) != nil {
		return fmt.Errorf("dataset binding drift during completion")
	}
	return nil
}

func loadCompletionDataset(file, canonicalFingerprint string) (completionDatasetBinding, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return completionDatasetBinding{}, err
	}
	// This deliberately projects only corpus/id/text. Labels are neither
	// decoded into typed cases nor exposed to feature construction.
	var projection struct {
		CorpusID string `json:"corpus_id"`
		Cases    []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &projection); err != nil {
		return completionDatasetBinding{}, err
	}
	if projection.CorpusID == "" || len(projection.Cases) == 0 || !validDigest(canonicalFingerprint) {
		return completionDatasetBinding{}, fmt.Errorf("invalid label-free completion dataset projection")
	}
	features, err := loadQueryFeatures(file)
	if err != nil {
		return completionDatasetBinding{}, err
	}
	ids := make([]string, 0, len(projection.Cases))
	seen := map[string]bool{}
	textSHA := map[string]string{}
	for _, item := range projection.Cases {
		if item.ID == "" || item.Text == "" || seen[item.ID] || features[item.ID].QueryID != item.ID {
			return completionDatasetBinding{}, fmt.Errorf("invalid label-free completion dataset projection")
		}
		seen[item.ID] = true
		ids = append(ids, item.ID)
		textSHA[item.ID] = sha256Hex([]byte(item.Text))
	}
	return completionDatasetBinding{CorpusID: projection.CorpusID, Fingerprint: canonicalFingerprint, SourceSHA256: sha256Hex(raw), QueryIDs: ids, Features: features, TextSHA256: textSHA}, nil
}

func loadRetrievalCompletionArtifact(root string) (retrievalCompletionManifest, string, error) {
	data, err := os.ReadFile(filepath.Join(root, "artifact-checksums.json"))
	if err != nil {
		return retrievalCompletionManifest{}, "", err
	}
	var checksums retrievalCompletionChecksums
	if err := strictJSON(data, &checksums); err != nil {
		return retrievalCompletionManifest{}, "", err
	}
	names := retrievalCompletionArtifactNames()
	if !checksums.Complete || checksums.Kind != "phase12_execution_artifact" || len(checksums.Entries) != len(names) || !validDigest(checksums.EntriesSum) {
		return retrievalCompletionManifest{}, "", fmt.Errorf("invalid retrieval artifact checksum manifest")
	}
	for index, name := range names {
		entry := checksums.Entries[index]
		data, err := os.ReadFile(filepath.Join(root, name))
		if entry.Path != name || !validDigest(entry.SHA256) || entry.ByteSize < 0 || err != nil || int64(len(data)) != entry.ByteSize || sha256Hex(data) != entry.SHA256 {
			return retrievalCompletionManifest{}, "", fmt.Errorf("retrieval artifact checksum verification failed")
		}
	}
	checksum, err := evalcontract.ArtifactChecksum(checksums.Entries)
	if err != nil || checksum != checksums.EntriesSum {
		return retrievalCompletionManifest{}, "", fmt.Errorf("retrieval artifact checksum aggregate mismatch")
	}
	data, err = os.ReadFile(filepath.Join(root, "run-manifest.json"))
	if err != nil {
		return retrievalCompletionManifest{}, "", err
	}
	var manifest retrievalCompletionManifest
	if err := strictJSON(data, &manifest); err != nil {
		return retrievalCompletionManifest{}, "", err
	}
	if manifest.SchemaVersion < 1 || manifest.RunID == "" || manifest.CorpusID == "" || !validDigest(manifest.CorpusManifestSHA256) || !validDigest(manifest.ContentSHA256) || !validDigest(manifest.IndexManifestSHA256) || !validDigest(manifest.DatasetSHA256) || !validDigest(manifest.DatasetSourceSHA256) || manifest.IndexGeneration < 1 || manifest.RawDocumentInputs < 1 || manifest.ServingProfile == "" || manifest.SourceProfile == "" || manifest.VectorSpaceProfile == "" || !strictUniqueIDs(manifest.QueryIDs) {
		return retrievalCompletionManifest{}, "", fmt.Errorf("invalid retrieval completion manifest")
	}
	if err := validateCompletionExperiment(manifest.Experiment, manifest.QueryIDs); err != nil {
		return retrievalCompletionManifest{}, "", err
	}
	if err := json.Unmarshal(manifest.Experiment, &manifest.experimentHeader); err != nil {
		return retrievalCompletionManifest{}, "", err
	}
	return manifest, checksums.EntriesSum, nil
}

func validateCompletionExperiment(raw json.RawMessage, queryIDs []string) error {
	if len(raw) == 0 || string(raw) == "null" {
		return fmt.Errorf("relation completion requires a plan-bound experiment header")
	}
	var header completionExperimentHeader
	if err := json.Unmarshal(raw, &header); err != nil {
		return fmt.Errorf("invalid relation completion experiment header: %w", err)
	}
	if header.ExperimentSeriesID == "" || header.EvidenceClass != completionEvidenceClass || header.PromotionEligible || header.LabelState != "DRAFT_TWO_PASS_PENDING" || header.QueryExecutionMode != "LIVE_ALL_QUERIES" || header.SeriesQueryOperationsPlanned < len(queryIDs) || len(header.Queries) != len(queryIDs) || !header.CorpusCleanVerified || header.FTSMode != "PRODUCTION" || !cleanKnownRelationBuild(header.BuildInfo) || !validDigest(header.EvaluationExecutableSHA256) || header.SourceDimensions != 1024 || (header.ServingDimensions != 512 && header.ServingDimensions != 1024) || header.StorageCodec != "int8" || header.DocumentProviderOperations != 0 || header.ReusedQueryVectors != 0 || header.ReusedDenseRankings != 0 || header.QueryVectorPersisted || !header.QueryVectorSHA256Recorded {
		return fmt.Errorf("relation completion experiment is not an authorized plan-bound series")
	}
	for index, query := range header.Queries {
		if query.QueryID != queryIDs[index] || !validDigest(query.TextSHA256) || query.ServingActiveSegmentObservations < 1 {
			return fmt.Errorf("relation completion experiment query binding mismatch")
		}
	}
	return nil
}
func completionSegmentCounts(header completionExperimentHeader) map[string]int {
	result := map[string]int{}
	for _, query := range header.Queries {
		result[query.QueryID] = query.ServingActiveSegmentObservations
	}
	return result
}

func retrievalCompletionArtifactNames() []string {
	return []string{"run-manifest.json", "per-query-trace.jsonl", "fts-candidates.jsonl", "dense-segment-candidates.jsonl", "collapsed-parent-candidates.jsonl", "rrf-results.jsonl", "inline-body-packages.jsonl", "per-query-metrics.jsonl", "aggregate-metrics.json", "cohort-language-report.json", "first-loss-report.json", "provider-usage.json", "implementation-audit.json", "promotion-contract.json", "promotion-result.json", "report.md"}
}
func strictJSON(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}
func strictUniqueIDs(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}
func sameIDs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
func validateCompletionBinding(graph GraphManifest, retrieval retrievalCompletionManifest, dataset completionDatasetBinding) error {
	if !cleanKnownRelationBuild(graph.Implementation) || !validDigest(graph.ResolverPolicy["build_executable_sha256"]) || graph.Corpus.CorpusID != retrieval.CorpusID || dataset.CorpusID != retrieval.CorpusID || graph.CorpusManifestFingerprint != retrieval.CorpusManifestSHA256 || graph.Corpus.ContentSHA256 != retrieval.ContentSHA256 || graph.IndexGeneration != retrieval.IndexGeneration || graph.IndexManifestSHA256 != retrieval.IndexManifestSHA256 || graph.Profiles["vector_storage"] != retrieval.ServingProfile || graph.Profiles["source"] != retrieval.SourceProfile || graph.Profiles["vector_space"] != retrieval.VectorSpaceProfile || retrieval.DatasetSHA256 != dataset.Fingerprint || retrieval.DatasetSourceSHA256 != dataset.SourceSHA256 || !sameIDs(retrieval.QueryIDs, dataset.QueryIDs) {
		return fmt.Errorf("relation completion input binding mismatch")
	}
	return validateCompletionQueryTextBinding(retrieval.experimentHeader, dataset.TextSHA256)
}

func validateCompletionQueryTextBinding(header completionExperimentHeader, textSHA256 map[string]string) error {
	for _, query := range header.Queries {
		if textSHA256[query.QueryID] != query.TextSHA256 {
			return fmt.Errorf("relation completion query text binding mismatch")
		}
	}
	return nil
}

func loadActiveInt8Scores(root string, queryIDs []string, rawDocumentInputs int, expectedCounts map[string]int, byHit map[string]string) (map[string][]search.EvaluationSegmentHit, map[string]completionCollapsedRow, error) {
	if rawDocumentInputs < 1 {
		return nil, nil, fmt.Errorf("invalid active-int8 raw document input count")
	}
	data, err := os.ReadFile(filepath.Join(root, "dense-segment-candidates.jsonl"))
	if err != nil {
		return nil, nil, err
	}
	segments, expected := map[string][]search.EvaluationSegmentHit{}, map[string]bool{}
	for _, id := range queryIDs {
		expected[id] = true
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var row completionSegmentRow
		if err := strictJSON([]byte(line), &row); err != nil {
			return nil, nil, err
		}
		if row.Variant != "serving_active_codec" {
			continue
		}
		if !expected[row.QueryID] || segments[row.QueryID] != nil || len(row.Segments) != expectedCounts[row.QueryID] {
			return nil, nil, fmt.Errorf("invalid active-int8 segment artifact row")
		}
		if err := validateCompletionCanonicalInputUniverse(row.Segments, rawDocumentInputs); err != nil {
			return nil, nil, err
		}
		observations := map[string]bool{}
		for i, segment := range row.Segments {
			if segment.Rank != i+1 || !validCompletionSegment(segment) || byHit[hitKey(segment.Path, segment.IndexedSHA256, segment.QualifiedSymbol, segment.ParentStartByte, segment.ParentEndByte)] == "" {
				return nil, nil, fmt.Errorf("active-int8 segment lacks a current semantic parent")
			}
			key := completionSegmentObservationKey(segment)
			if observations[key] {
				return nil, nil, fmt.Errorf("duplicate active-int8 segment observation")
			}
			observations[key] = true
		}
		segments[row.QueryID] = row.Segments
	}
	collapsed, err := loadActiveCollapsedRows(root, queryIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(segments) != len(queryIDs) || len(collapsed) != len(queryIDs) {
		return nil, nil, fmt.Errorf("active-int8 artifact query cardinality mismatch")
	}
	return segments, collapsed, nil
}

func validateCompletionCanonicalInputUniverse(segments []search.EvaluationSegmentHit, rawDocumentInputs int) error {
	if rawDocumentInputs < 1 {
		return fmt.Errorf("invalid active-int8 raw document input count")
	}
	inputs := map[string]bool{}
	for _, segment := range segments {
		if !validDigest(segment.CanonicalInputSHA256) {
			return fmt.Errorf("invalid active-int8 canonical input digest")
		}
		inputs[segment.CanonicalInputSHA256] = true
	}
	if len(inputs) != rawDocumentInputs {
		return fmt.Errorf("active-int8 canonical inputs do not cover the planned raw document input universe")
	}
	return nil
}
func completionSegmentObservationKey(value search.EvaluationSegmentHit) string {
	return strings.Join([]string{value.CanonicalInputSHA256, value.Path, value.IndexedSHA256, value.QualifiedSymbol, fmt.Sprintf("%d", value.ParentStartByte), fmt.Sprintf("%d", value.ParentEndByte), fmt.Sprintf("%d", value.StartByte), fmt.Sprintf("%d", value.EndByte)}, "\x00")
}
func validCompletionSegment(v search.EvaluationSegmentHit) bool {
	return validDigest(v.CanonicalInputSHA256) && validRelative(v.Path) && validDigest(v.IndexedSHA256) && v.QualifiedSymbol != "" && v.ParentStartByte >= 0 && v.ParentEndByte > v.ParentStartByte && v.StartByte >= v.ParentStartByte && v.EndByte > v.StartByte && v.EndByte <= v.ParentEndByte && !math.IsNaN(v.Score) && !math.IsInf(v.Score, 0)
}
func loadActiveCollapsedRows(root string, queryIDs []string) (map[string]completionCollapsedRow, error) {
	data, err := os.ReadFile(filepath.Join(root, "collapsed-parent-candidates.jsonl"))
	if err != nil {
		return nil, err
	}
	result, expected := map[string]completionCollapsedRow{}, map[string]bool{}
	for _, id := range queryIDs {
		expected[id] = true
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var row completionCollapsedRow
		if err := strictJSON([]byte(line), &row); err != nil {
			return nil, err
		}
		if row.Variant != "serving_active_codec" {
			continue
		}
		if !expected[row.QueryID] || row.Ranking.QueryID != row.QueryID || row.FailureStage != "" || !validDigest(row.Ranking.QueryVectorSHA256) || len(row.Ranking.Hits) == 0 || result[row.QueryID].QueryID != "" {
			return nil, fmt.Errorf("invalid active-int8 collapsed artifact row")
		}
		for i, hit := range row.Ranking.Hits {
			if hit.Rank != i+1 || hit.Score == nil || hit.Validate() != nil {
				return nil, fmt.Errorf("invalid active-int8 collapsed hit")
			}
		}
		result[row.QueryID] = row
	}
	if len(result) != len(queryIDs) {
		return nil, fmt.Errorf("active-int8 query-vector digest cardinality mismatch")
	}
	return result, nil
}

func collapseActiveScores(queryID string, segments []search.EvaluationSegmentHit, byHit map[string]string, byID map[string]Parent) ([]semanticParentScore, error) {
	best := map[string]search.EvaluationSegmentHit{}
	for _, segment := range segments {
		id := byHit[hitKey(segment.Path, segment.IndexedSHA256, segment.QualifiedSymbol, segment.ParentStartByte, segment.ParentEndByte)]
		if _, ok := byID[id]; !ok {
			return nil, fmt.Errorf("segment parent mapping changed")
		}
		if previous, ok := best[id]; !ok || completionSegmentBefore(segment, previous) {
			best[id] = segment
		}
	}
	values := make([]semanticParentScore, 0, len(best))
	for id, segment := range best {
		parent := byID[id]
		values = append(values, semanticParentScore{QueryID: queryID, ParentID: id, Path: parent.Path, IndexedSHA256: parent.IndexedSHA256, QualifiedSymbol: parent.QualifiedSymbol, StartByte: parent.StartByte, EndByte: parent.EndByte, NativeScore: segment.Score, WinningSegment: segment})
	}
	sort.Slice(values, func(i, j int) bool { return completionParentBefore(values[i], values[j]) })
	for i := range values {
		values[i].GlobalRank, values[i].ParentCount = i+1, len(values)
	}
	for start := 0; start < len(values); {
		end := start + 1
		for end < len(values) && values[end].NativeScore == values[start].NativeScore {
			end++
		}
		for i := start; i < end; i++ {
			values[i].TieStartRank, values[i].TieEndRank = start+1, end
			values[i].RankPercentileNumerator, values[i].RankPercentileDenominator = values[i].GlobalRank, len(values)
		}
		start = end
	}
	return values, nil
}
func validateCollapsedTopK(values []semanticParentScore, row completionCollapsedRow, byHit map[string]string) error {
	if len(values) < len(row.Ranking.Hits) {
		return fmt.Errorf("collapsed active top-k exceeds collapsed segment parents")
	}
	byParent := map[string]semanticParentScore{}
	for _, value := range values {
		byParent[value.ParentID] = value
	}
	seen := map[string]bool{}
	for _, hit := range row.Ranking.Hits {
		id := byHit[hitKey(hit.Path, hit.IndexedSHA256, hit.QualifiedSymbol, hit.StartByte, hit.EndByte)]
		value, ok := byParent[id]
		// The portable segment wire has no production segment IDs, so a
		// score tie may use a different stable order than production. The
		// exact active top-K identities/scores are still cross-checked and
		// the portable rank is retained as an explicit tie range.
		if id == "" || seen[id] || !ok || hit.Score == nil || value.NativeScore != *hit.Score || hit.Rank < value.TieStartRank || hit.Rank > value.TieEndRank {
			return fmt.Errorf("active-int8 collapsed parent cross-check failed")
		}
		seen[id] = true
	}
	return nil
}
func completionSegmentBefore(a, b search.EvaluationSegmentHit) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	if a.ParentStartByte != b.ParentStartByte {
		return a.ParentStartByte < b.ParentStartByte
	}
	return a.CanonicalInputSHA256 < b.CanonicalInputSHA256
}
func completionParentBefore(a, b semanticParentScore) bool {
	if a.NativeScore != b.NativeScore {
		return a.NativeScore > b.NativeScore
	}
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	if a.StartByte != b.StartByte {
		return a.StartByte < b.StartByte
	}
	return a.ParentID < b.ParentID
}
func completionRankHits(row completionCollapsedRow) ([]rankHit, error) {
	if len(row.Ranking.Hits) < MaxDenseDepth {
		return nil, fmt.Errorf("authoritative active-int8 collapsed top20 is missing")
	}
	hits := make([]rankHit, MaxDenseDepth)
	for i, value := range row.Ranking.Hits[:MaxDenseDepth] {
		if value.Rank != i+1 || value.Score == nil {
			return nil, fmt.Errorf("invalid authoritative active-int8 collapsed top20")
		}
		score := *value.Score
		hits[i] = rankHit{Path: value.Path, IndexedSHA256: value.IndexedSHA256, QualifiedSymbol: value.QualifiedSymbol, StartByte: value.StartByte, EndByte: value.EndByte, Rank: value.Rank, Score: &score}
	}
	return hits, nil
}

func completionEvidence(ctx context.Context, db *sql.DB, queryIDs []string, featuresByQuery map[string]queryFeatures, scores map[string][]semanticParentScore, collapsed map[string]completionCollapsedRow, parents map[string]Parent, lines map[string]completionParentLines, byHit map[string]string) ([]semanticEndpointFeature, []closureCandidate, []relationHint, []any, error) {
	stats, _, err := completeGraphEdgeStats(ctx, db)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	features, closures, hints := []semanticEndpointFeature{}, []closureCandidate{}, []relationHint{}
	traces := make([]any, 0, len(queryIDs))
	for _, queryID := range queryIDs {
		values, feature := scores[queryID], featuresByQuery[queryID]
		if feature.QueryID != queryID || len(values) < ProtectedPrimaryK {
			return nil, nil, nil, nil, fmt.Errorf("invalid completion query universe")
		}
		hits, err := completionRankHits(collapsed[queryID])
		if err != nil {
			return nil, nil, nil, nil, err
		}
		group, err := selectAnchorGroup(queryID, feature.AnchorTokens, hits, byHit, parents)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		anchorIDs := []string{}
		for _, anchor := range group.Anchors {
			anchorIDs = append(anchorIDs, anchor.ParentID)
		}
		frontierFacts, err := reachableFacts(ctx, db, anchorIDs)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		frontier, err := buildFrontierTrace(group, feature, frontierFacts, stats, rankPositions(hits, byHit))
		if err != nil {
			return nil, nil, nil, nil, err
		}
		primaryIDs := make([]string, ProtectedPrimaryK)
		for i, hit := range hits[:ProtectedPrimaryK] {
			primaryIDs[i] = byHit[hitKey(hit.Path, hit.IndexedSHA256, hit.QualifiedSymbol, hit.StartByte, hit.EndByte)]
			if primaryIDs[i] == "" {
				return nil, nil, nil, nil, fmt.Errorf("authoritative primary lacks current parent")
			}
		}
		byParent := map[string]semanticParentScore{}
		for _, value := range values {
			byParent[value.ParentID] = value
		}
		distribution := scoreDistribution(values)
		denseIDs := map[string]bool{}
		for _, hit := range hits {
			denseIDs[byHit[hitKey(hit.Path, hit.IndexedSHA256, hit.QualifiedSymbol, hit.StartByte, hit.EndByte)]] = true
		}
		endpointRows, err := semanticEndpointRows(queryID, frontier.FinalFrontier, primaryIDs, denseIDs, byParent, distribution)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		for i := range endpointRows {
			endpointRows[i].SemanticOrdinal = i + 1
		}
		features = append(features, endpointRows...)
		closureFacts, err := reachableFacts(ctx, db, primaryIDs)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		closureRows, err := closureRowsForQuery(queryID, closureFacts, primaryIDs, parents)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		closures = append(closures, closureRows...)
		hintRows, err := hintRowsForQuery(queryID, frontier.FinalFrontier, primaryIDs, denseIDs, byParent, parents, lines)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		hints = append(hints, hintRows...)
		traces = append(traces, map[string]any{"query_id": queryID, "primary_parent_ids": primaryIDs, "anchor_group": group, "semantic_parent_count": len(values), "semantic_distribution": distribution, "frontier_final_digest": frontier.FinalDigest, "frontier_final_edges": len(frontier.FinalFrontier), "frontier_abstention_reason": frontier.AbstentionReason, "frontier_counts": frontier.Counts, "contract_closure_one_hop_facts": len(closureFacts), "semantic_endpoint_features": len(endpointRows), "contract_closure_candidates": len(closureRows), "relation_hints": len(hintRows), "label_loading": "LABEL_FIELDS_NOT_DECODED_STAGE_A", "fts_only_relation_hints": 0})
	}
	return features, closures, hints, traces, nil
}
func scoreDistribution(values []semanticParentScore) semanticDistribution {
	result := semanticDistribution{ParentCount: len(values)}
	if len(values) == 0 {
		return result
	}
	result.Minimum, result.Maximum = values[len(values)-1].NativeScore, values[0].NativeScore
	for _, v := range values {
		result.Mean += v.NativeScore
	}
	result.Mean /= float64(len(values))
	for _, v := range values {
		d := v.NativeScore - result.Mean
		result.StdDev += d * d
	}
	result.StdDev = math.Sqrt(result.StdDev / float64(len(values)))
	result.Span = result.Maximum - result.Minimum
	return result
}

type endpointAccumulator struct {
	edge             frontierEdge
	relationIDs      map[string]bool
	canonicalEdgeIDs map[string]bool
	views            []endpointSupportingView
}

func semanticEndpointRows(queryID string, frontier []frontierEdge, primaryIDs []string, dense map[string]bool, scores map[string]semanticParentScore, distribution semanticDistribution) ([]semanticEndpointFeature, error) {
	primary, endpoints := map[string]bool{}, map[string]*endpointAccumulator{}
	for _, id := range primaryIDs {
		primary[id] = true
	}
	for _, edge := range frontier {
		fact := edge.Candidate.Fact
		if _, ok := scores[fact.AnchorID]; !ok {
			return nil, fmt.Errorf("frontier anchor lacks active semantic score")
		}
		anchor := scores[fact.AnchorID]
		value := endpoints[fact.EndpointID]
		if value == nil {
			value = &endpointAccumulator{edge: edge, relationIDs: map[string]bool{}, canonicalEdgeIDs: map[string]bool{}}
			endpoints[fact.EndpointID] = value
		}
		value.relationIDs[fact.RelationID] = true
		value.canonicalEdgeIDs[edge.CanonicalEdgeID] = true
		endpoint, observed := scores[fact.EndpointID]
		gap := 0.0
		if observed {
			gap = endpoint.NativeScore - anchor.NativeScore
		}
		value.views = append(value.views, endpointSupportingView{RelationID: fact.RelationID, CanonicalEdgeID: edge.CanonicalEdgeID, AnchorParentID: fact.AnchorID, RelationKind: fact.Kind, Direction: fact.Direction, StructuralTier: edge.Candidate.Stats.Tier, AnchorScore: anchor.NativeScore, EndpointGap: gap})
	}
	values := make([]*endpointAccumulator, 0, len(endpoints))
	for _, v := range endpoints {
		values = append(values, v)
	}
	sort.SliceStable(values, func(i, j int) bool {
		left, leftOK := scores[values[i].edge.Candidate.Fact.EndpointID]
		right, rightOK := scores[values[j].edge.Candidate.Fact.EndpointID]
		if leftOK != rightOK {
			return leftOK
		}
		if leftOK && left.NativeScore != right.NativeScore {
			return left.NativeScore > right.NativeScore
		}
		return left.ParentID < right.ParentID
	})
	if len(values) == 0 {
		return []semanticEndpointFeature{}, nil
	}
	observed := make([]*endpointAccumulator, 0, len(values))
	for _, value := range values {
		if _, ok := scores[value.edge.Candidate.Fact.EndpointID]; ok {
			observed = append(observed, value)
		}
	}
	best := 0.0
	if len(observed) > 0 {
		best = scores[observed[0].edge.Candidate.Fact.EndpointID].NativeScore
	}
	var runner, gap, normalizedGap *float64
	if len(observed) > 1 {
		v := scores[observed[1].edge.Candidate.Fact.EndpointID].NativeScore
		runner = &v
		delta := best - v
		gap = &delta
		if distribution.Span != 0 {
			n := delta / distribution.Span
			normalizedGap = &n
		}
	}
	rows := make([]semanticEndpointFeature, 0, len(values))
	for _, a := range values {
		fact := a.edge.Candidate.Fact
		anchor, endpoint := scores[fact.AnchorID], scores[fact.EndpointID]
		endpointObserved := endpoint.ParentID != ""
		row := semanticEndpointFeature{QueryID: queryID, RelationID: fact.RelationID, SupportingRelationIDs: sortedKeys(a.relationIDs), SupportingCanonicalEdgeIDs: sortedKeys(a.canonicalEdgeIDs), SupportingViews: append([]endpointSupportingView(nil), a.views...), RepresentativeRule: "first-final-frontier-view-v1", Direction: fact.Direction, RelationKind: fact.Kind, StructuralTier: a.edge.Candidate.Stats.Tier, AnchorParentID: fact.AnchorID, EndpointParentID: fact.EndpointID, EndpointGlobalRank: endpoint.GlobalRank, EndpointTieStartRank: endpoint.TieStartRank, EndpointTieEndRank: endpoint.TieEndRank, EndpointPercentileNumerator: endpoint.RankPercentileNumerator, EndpointPercentileDenominator: endpoint.RankPercentileDenominator, EndpointPercentileStart: endpoint.TieStartRank, EndpointPercentileEnd: endpoint.TieEndRank, EndpointNativeScore: endpoint.NativeScore, AnchorNativeScore: anchor.NativeScore, EndpointAnchorGap: endpoint.NativeScore - anchor.NativeScore, BestCandidateScore: best, RunnerUpCandidateScore: runner, BestRunnerUpGap: gap, NormalizedBestRunnerUpGap: normalizedGap, AmbiguityDegenerate: len(observed) < 2, NormalizationDegenerate: distribution.Span == 0, AlreadyPrimaryTop5: primary[fact.EndpointID], AlreadyDenseTop20: dense[fact.EndpointID], AbsentFromDenseTop20: !dense[fact.EndpointID], PrimaryTop5TieStatus: tieBoundaryStatus(endpoint, ProtectedPrimaryK, primary[fact.EndpointID], endpointObserved), DenseTop20TieStatus: tieBoundaryStatus(endpoint, MaxDenseDepth, dense[fact.EndpointID], endpointObserved), EndpointScoreObserved: endpointObserved, ScoreScale: "active_int8_native_similarity", QueryWideTopOneScore: best, QueryWideTopTwoScore: runner, QueryWideTopOneTwoGap: gap, QueryWideTopOneTwoTied: runner != nil && *gap == 0, Distribution: distribution}
		if endpointObserved && distribution.Span != 0 {
			n := row.EndpointAnchorGap / distribution.Span
			row.NormalizedEndpointAnchorGap = &n
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func tieBoundaryStatus(value semanticParentScore, boundary int, exactMember, observed bool) string {
	if !observed {
		return "NOT_OBSERVED"
	}
	if value.TieStartRank <= boundary && value.TieEndRank > boundary {
		return "TIE_SPANS_BOUNDARY"
	}
	if exactMember {
		return "INCLUDED_EXACT"
	}
	return "EXCLUDED_EXACT"
}

func closureRowsForQuery(queryID string, facts []Fact, primaryIDs []string, parents map[string]Parent) ([]closureCandidate, error) {
	primary := map[string]bool{}
	for _, id := range primaryIDs {
		primary[id] = true
	}
	values := []Fact{}
	for _, fact := range facts {
		if fact.Direction == Forward && primary[fact.AnchorID] && fact.Kind == TypeRef && (fact.Metadata.Zone == SignatureZone || fact.Metadata.Zone == TypeBodyZone) {
			values = append(values, fact)
		}
	}
	sort.SliceStable(values, func(i, j int) bool {
		left, right := primaryRank(values[i].AnchorID, primaryIDs), primaryRank(values[j].AnchorID, primaryIDs)
		if left != right {
			return left < right
		}
		if values[i].Metadata.SourceOrdinal != values[j].Metadata.SourceOrdinal {
			return values[i].Metadata.SourceOrdinal < values[j].Metadata.SourceOrdinal
		}
		if values[i].OccurrenceByte != values[j].OccurrenceByte {
			return values[i].OccurrenceByte < values[j].OccurrenceByte
		}
		return values[i].RelationID < values[j].RelationID
	})
	requestChosen, primaryChosen := map[string]bool{}, map[string]map[string]bool{}
	primaryOrdinals, rows, requestCumulative, requestOrdinal := map[string]int{}, []closureCandidate{}, 0, 0
	for _, fact := range values {
		role, roleOK := closureRoleClass(fact.Metadata)
		row := closureCandidate{QueryID: queryID, PrimaryParentID: fact.AnchorID, TargetParentID: fact.EndpointID, RelationID: fact.RelationID, RoleClass: role, StructuralTier: DeclarationContractTier}
		if !roleOK {
			row.RoleClass, row.OmissionReason = "UNDECLARED_SIGNATURE_ROLE", "ROLE_NOT_CONTRACT"
			rows = append(rows, row)
			continue
		}
		if fact.AnchorID == fact.EndpointID {
			row.OmissionReason = "SELF_CYCLE"
			rows = append(rows, row)
			continue
		}
		if primary[fact.EndpointID] {
			row.OmissionReason = "ALREADY_PRIMARY"
			rows = append(rows, row)
			continue
		}
		target, ok := parents[fact.EndpointID]
		if !ok {
			row.OmissionReason = "MISSING_TARGET_PARENT"
			rows = append(rows, row)
			continue
		}
		if target.FileRole != ProductionFileRole {
			row.OmissionReason = "TARGET_NOT_PRODUCTION"
			rows = append(rows, row)
			continue
		}
		row.BodyBytes, row.BodyComplete = len([]byte(target.SourceBody)), len(target.SourceBody) > 0
		if !row.BodyComplete {
			row.OmissionReason = "BODY_INCOMPLETE"
			rows = append(rows, row)
			continue
		}
		if primaryChosen[fact.AnchorID] == nil {
			primaryChosen[fact.AnchorID] = map[string]bool{}
		}
		if primaryChosen[fact.AnchorID][fact.EndpointID] {
			row.OmissionReason = "DUPLICATE_TARGET_PARENT_PRIMARY"
			rows = append(rows, row)
			continue
		}
		primaryChosen[fact.AnchorID][fact.EndpointID] = true
		primaryOrdinals[fact.AnchorID]++
		row.PrimaryOrdinal = primaryOrdinals[fact.AnchorID]
		row.PrimaryCountBudgetEligible = budgetsFor(row.PrimaryOrdinal, closureCountBudgetGrid)
		if requestChosen[fact.EndpointID] {
			row.OmissionReason = "DUPLICATE_TARGET_PARENT_REQUEST"
			rows = append(rows, row)
			continue
		}
		requestChosen[fact.EndpointID] = true
		requestOrdinal++
		requestCumulative += row.BodyBytes
		row.RequestOrdinal, row.RequestCumulativeBodyBytes = requestOrdinal, requestCumulative
		row.RequestCountBudgetEligible = budgetsFor(requestOrdinal, closureCountBudgetGrid)
		// Legacy fields remain request-global for existing portable readers.
		row.Ordinal, row.CumulativeBodyBytes = requestOrdinal, requestCumulative
		row.CountBudgetEligible, row.ByteBudgetEligible = row.RequestCountBudgetEligible, budgetsFor(requestCumulative, closureByteBudgetGrid)
		rows = append(rows, row)
	}
	return rows, nil
}
func closureRoleClass(m OccurrenceMetadata) (string, bool) {
	switch m.Role {
	case TypeValueParameterRole:
		return "VALUE_PARAMETER", true
	case TypeReturnRole:
		return "RETURN", true
	case TypeParameterRole:
		return "GENERIC_CONSTRAINT", true
	case TypeHeritageRole:
		return "HERITAGE", true
	case TypeFieldRole, TypeAliasRole, TypeArgumentRole, TypeOtherRole:
		return "SIGNATURE", true
	}
	return "", false
}
func primaryRank(id string, primary []string) int {
	for i, value := range primary {
		if value == id {
			return i + 1
		}
	}
	return len(primary) + 1
}
func budgetsFor(value int, grid []int) []int {
	result := []int{}
	for _, budget := range grid {
		if value <= budget {
			result = append(result, budget)
		}
	}
	return result
}

type hintAccumulator struct {
	edge               frontierEdge
	relationIDs        map[string]bool
	canonicalEdgeIDs   map[string]bool
	occurrences, views int
}

func hintRowsForQuery(queryID string, frontier []frontierEdge, primaryIDs []string, dense map[string]bool, scores map[string]semanticParentScore, parents map[string]Parent, lines map[string]completionParentLines) ([]relationHint, error) {
	primary, grouped := map[string]bool{}, map[string]*hintAccumulator{}
	for _, id := range primaryIDs {
		primary[id] = true
	}
	for _, edge := range frontier {
		fact, tier := edge.Candidate.Fact, edge.Candidate.Stats.Tier
		if fact.AnchorID == fact.EndpointID {
			return nil, fmt.Errorf("self edge escaped final frontier")
		}
		key := fact.EndpointID + "\x00" + string(fact.Kind) + "\x00" + string(fact.Direction) + "\x00" + string(tier)
		group := grouped[key]
		if group == nil {
			group = &hintAccumulator{edge: edge, relationIDs: map[string]bool{}, canonicalEdgeIDs: map[string]bool{}}
			grouped[key] = group
		}
		group.relationIDs[fact.RelationID] = true
		group.canonicalEdgeIDs[edge.CanonicalEdgeID] = true
		group.views++
		group.occurrences += edge.Candidate.Stats.EdgeOccurrences
	}
	values := make([]*hintAccumulator, 0, len(grouped))
	for _, v := range grouped {
		values = append(values, v)
	}
	sort.SliceStable(values, func(i, j int) bool {
		left, leftOK := scores[values[i].edge.Candidate.Fact.EndpointID]
		right, rightOK := scores[values[j].edge.Candidate.Fact.EndpointID]
		if leftOK != rightOK {
			return leftOK
		}
		if leftOK && left.NativeScore != right.NativeScore {
			return left.NativeScore > right.NativeScore
		}
		return values[i].edge.CanonicalEdgeID < values[j].edge.CanonicalEdgeID
	})
	rows, cumulative := make([]relationHint, 0, len(values)), 0
	for i, group := range values {
		fact := group.edge.Candidate.Fact
		source, sourceOK := parents[fact.AnchorID]
		target, targetOK := parents[fact.EndpointID]
		if !sourceOK || !targetOK {
			return nil, fmt.Errorf("hint parent mapping changed")
		}
		sourceLines, targetLines := lines[fact.AnchorID], lines[fact.EndpointID]
		row := relationHint{QueryID: queryID, RelationID: fact.RelationID, SupportingRelationIDs: sortedKeys(group.relationIDs), SupportingCanonicalEdgeIDs: sortedKeys(group.canonicalEdgeIDs), Kind: fact.Kind, Direction: fact.Direction, StructuralTier: group.edge.Candidate.Stats.Tier, SourcePath: source.Path, SourceSHA256: source.IndexedSHA256, SourceStartByte: source.StartByte, SourceEndByte: source.EndByte, SourceStartLine: sourceLines.StartLine, SourceEndLine: sourceLines.EndLine, TargetSymbol: target.Symbol, TargetQualified: target.QualifiedSymbol, TargetPath: target.Path, TargetSHA256: target.IndexedSHA256, TargetStartByte: target.StartByte, TargetEndByte: target.EndByte, TargetStartLine: targetLines.StartLine, TargetEndLine: targetLines.EndLine, AlreadyPrimary: primary[target.ID], AlreadyDense: dense[target.ID], OccurrenceCount: group.occurrences, SupportingViewCount: group.views, SemanticOrdinal: i + 1, OmissionStatus: "CANDIDATE", DisclosureSchema: hintDisclosureSchemaID}
		payload, err := json.Marshal(hintDisclosure{Schema: hintDisclosureSchemaID, RelationID: row.RelationID, Kind: row.Kind, Direction: row.Direction, StructuralTier: row.StructuralTier, SourcePath: row.SourcePath, SourceSHA256: row.SourceSHA256, SourceStartLine: row.SourceStartLine, SourceEndLine: row.SourceEndLine, TargetSymbol: row.TargetSymbol, TargetQualified: row.TargetQualified, TargetPath: row.TargetPath, TargetSHA256: row.TargetSHA256, TargetStartLine: row.TargetStartLine, TargetEndLine: row.TargetEndLine})
		if err != nil {
			return nil, err
		}
		row.SerializedBytes = len(payload)
		row.DisclosureSHA256 = sha256Hex(payload)
		cumulative += row.SerializedBytes
		row.CumulativeBytes = cumulative
		row.CountBudgetEligible, row.ByteBudgetEligible = budgetsFor(i+1, hintCountBudgetGrid), budgetsFor(cumulative, hintByteBudgetGrid)
		if len(row.CountBudgetEligible) == 0 || len(row.ByteBudgetEligible) == 0 {
			row.OmissionStatus = "OUTSIDE_PREDECLARED_BUDGET_GRID"
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func writeCompletionArtifacts(root string, graph GraphManifest, retrieval retrievalCompletionManifest, retrievalChecksum string, dataset completionDatasetBinding, hashes map[string]string, scores map[string][]semanticParentScore, features []semanticEndpointFeature, closures []closureCandidate, hints []relationHint, traces []any, policyFingerprint, executableSHA string) error {
	manifest := map[string]any{"schema_version": SchemaVersion, "kind": "cidx.relation_evidence_completion.stage_a.v1", "policy": completionPolicyID, "policy_fingerprint": policyFingerprint, "build_info": buildinfo.Current(), "executable_sha256": executableSHA, "graph_logical_sha256": graph.LogicalGraphSHA256, "graph_database_sha256": graph.DatabaseSHA256, "retrieval_run_id": retrieval.RunID, "retrieval_artifact_checksum": retrievalChecksum, "corpus_id": graph.Corpus.CorpusID, "corpus_manifest_sha256": graph.CorpusManifestFingerprint, "dataset_sha256": dataset.Fingerprint, "dataset_source_sha256": dataset.SourceSHA256, "index_generation": graph.IndexGeneration, "index_manifest_sha256": graph.IndexManifestSHA256, "profiles": graph.Profiles, "query_vector_sha256": hashes, "semantic_parent_collapse_policy": semanticCollapsePolicyID, "normalization_policy": semanticNormalizationPolicyID, "closure_count_budget_grid": closureCountBudgetGrid, "closure_byte_budget_grid": closureByteBudgetGrid, "hint_count_budget_grid": hintCountBudgetGrid, "hint_byte_budget_grid": hintByteBudgetGrid, "hint_disclosure_schema": hintDisclosureSchemaID, "label_loading": "LABEL_FIELDS_NOT_DECODED_STAGE_A", "relation_provider_operations": 0, "provider_usage": map[string]int{"relation_provider_operations": 0}, "query_vectors_persisted": false, "document_vectors_persisted": false, "evidence_class": completionEvidenceClass, "promotion_eligible": false, "review_validation": "NO_INDEPENDENT_HUMAN_REVIEW"}
	binding := map[string]any{"graph": map[string]any{"logical_sha256": graph.LogicalGraphSHA256, "database_sha256": graph.DatabaseSHA256, "semantic_parent_inventory_sha256": graph.SemanticParentInventorySHA256}, "retrieval": map[string]any{"run_id": retrieval.RunID, "artifact_checksum": retrievalChecksum, "dataset_sha256": retrieval.DatasetSHA256, "dataset_source_sha256": retrieval.DatasetSourceSHA256, "serving_profile": retrieval.ServingProfile}, "current": map[string]any{"index_generation": graph.IndexGeneration, "index_manifest_sha256": graph.IndexManifestSHA256, "dataset_sha256": dataset.Fingerprint, "dataset_source_sha256": dataset.SourceSHA256}}
	semanticRows := []any{}
	for _, id := range retrieval.QueryIDs {
		for _, score := range scores[id] {
			semanticRows = append(semanticRows, score)
		}
	}
	closuresResults, admissions := []any{}, []any{}
	for _, id := range retrieval.QueryIDs {
		admissions = append(admissions, map[string]any{"query_id": id, "status": "NO_SEMANTIC_GATE_SELECTED_STAGE_A"})
		closuresResults = append(closuresResults, map[string]any{"query_id": id, "status": "NO_CLOSURE_CAP_SELECTED_STAGE_A"})
	}
	aggregate := map[string]any{"queries": len(retrieval.QueryIDs), "semantic_parent_scores": len(semanticRows), "relation_endpoint_features": len(features), "contract_closure_candidates": len(closures), "relation_hints": len(hints), "fts_only_relation_hints": 0, "relation_provider_operations": 0, "label_loading": "LABEL_FIELDS_NOT_DECODED_STAGE_A"}
	for _, output := range []struct {
		name string
		rows []any
	}{{"semantic-parent-scores.jsonl", semanticRows}, {"relation-endpoint-features.jsonl", completionAny(features)}, {"contract-closure-candidates.jsonl", completionAny(closures)}, {"relation-hints.jsonl", completionAny(hints)}, {"semantic-admission-results.jsonl", admissions}, {"closure-package-results.jsonl", closuresResults}, {"per-query-relation-trace.jsonl", traces}} {
		if err := writeJSONL(filepath.Join(root, output.name), output.rows); err != nil {
			return err
		}
	}
	if err := writePortableJSON(filepath.Join(root, "run-manifest.json"), manifest, ""); err != nil {
		return err
	}
	if err := writePortableJSON(filepath.Join(root, "input-artifact-binding.json"), binding, ""); err != nil {
		return err
	}
	if err := writePortableJSON(filepath.Join(root, "aggregate-relation-metrics.json"), aggregate, ""); err != nil {
		return err
	}
	if err := writePortableJSON(filepath.Join(root, "cohort-language-report.json"), map[string]any{"status": "LABEL_FIELDS_NOT_DECODED_STAGE_A", "reason": "labels remain unavailable to completion candidate generation"}, ""); err != nil {
		return err
	}
	if err := writePortableJSON(filepath.Join(root, "first-loss-report.json"), map[string]any{"status": "NOT_EVALUATED_STAGE_A", "semantic_score_binding": len(retrieval.QueryIDs)}, ""); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "report.md"), []byte(fmt.Sprintf("# Relation evidence completion Stage A\n\nQueries: %d\n\nProvider operations: 0\n", len(retrieval.QueryIDs))), 0o600)
}
func completionAny[T any](values []T) []any {
	result := make([]any, 0, len(values))
	for _, v := range values {
		result = append(result, v)
	}
	return result
}
