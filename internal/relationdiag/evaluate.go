package relationdiag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/bits"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cidx/internal/buildinfo"
	"cidx/internal/eval"
	"cidx/internal/evalcontract"
	"cidx/internal/search/lexical"
	"cidx/internal/store"
	"cidx/internal/symbol"
	_ "modernc.org/sqlite"
)

type EvaluationRequest struct {
	RunID, EvaluationRoot, GraphDirectory, ReplayPath, DatasetPath, ProbesPath string
	Parents                                                                    store.SemanticParentSnapshot
	SelectionPolicy                                                            string
}

type rankHit struct {
	Path            string   `json:"path"`
	IndexedSHA256   string   `json:"indexed_sha256"`
	QualifiedSymbol string   `json:"qualified_symbol"`
	StartByte       int      `json:"start_byte"`
	EndByte         int      `json:"end_byte"`
	Rank            int      `json:"rank"`
	Score           *float64 `json:"score,omitempty"`
}
type frozenReplay struct {
	SchemaVersion      int               `json:"schema_version"`
	Kind               string            `json:"kind"`
	CorpusID           string            `json:"corpus_id"`
	DatasetFingerprint string            `json:"dataset_fingerprint"`
	ReviewProtocol     string            `json:"review_protocol_version"`
	RelevanceAuthority string            `json:"relevance_authority"`
	ReviewValidation   string            `json:"review_validation"`
	SourceSHA256       map[string]string `json:"source_sha256"`
	Lanes              map[string]struct {
		Ranks map[string][]rankHit `json:"ranks"`
	} `json:"lanes"`
}
type Fact struct {
	RelationID        string             `json:"relation_id"`
	Direction         Direction          `json:"direction"`
	AnchorID          string             `json:"anchor_parent_id"`
	EndpointID        string             `json:"endpoint_parent_id"`
	Kind              RelationKind       `json:"relation_kind"`
	OccurrencePath    string             `json:"occurrence_path"`
	OccurrenceByte    int                `json:"occurrence_byte"`
	OccurrenceEndByte int                `json:"occurrence_end_byte"`
	Metadata          OccurrenceMetadata `json:"metadata"`
}
type Bundle struct {
	QueryID         string               `json:"query_id"`
	Selected        *Fact                `json:"selected,omitempty"`
	AddedParentIDs  []string             `json:"added_parent_ids"`
	SelectionPolicy string               `json:"selection_policy"`
	SelectionKey    []any                `json:"selection_key,omitempty"`
	AdmissionOrder  []AdmissionCandidate `json:"admission_order"`
}
type AdmissionCandidate struct {
	Fact         Fact  `json:"fact"`
	Prefix       []any `json:"prefix"`
	SelectionKey []any `json:"selection_key"`
	RerankKey    []any `json:"rerank_key,omitempty"`
	Admitted     bool  `json:"admitted"`
}
type queryFeatures struct {
	QueryID                      string     `json:"query_id"`
	Tokens                       []string   `json:"tokens"`
	AnchorTokens                 []string   `json:"anchor_tokens"`
	Direction                    Direction  `json:"direction"`
	SignatureIntent              bool       `json:"signature_intent"`
	ValueParameterContractIntent bool       `json:"value_parameter_contract_intent"`
	MutationIntent               bool       `json:"mutation_intent"`
	ReturnIntent                 bool       `json:"return_intent"`
	ConditionIntent              bool       `json:"condition_intent"`
	DeprecatedIntent             bool       `json:"deprecated_intent"`
	ExplicitFileRoles            []FileRole `json:"explicit_file_roles"`
}
type RelatedBody struct {
	QueryID        string `json:"query_id"`
	ParentID       string `json:"parent_id"`
	BodyBytes      int    `json:"body_bytes"`
	BodySHA256     string `json:"body_sha256,omitempty"`
	BodyComplete   bool   `json:"body_complete"`
	OmissionReason string `json:"omission_reason,omitempty"`
}
type queryTrace struct {
	QueryID           string             `json:"query_id"`
	PrimaryTop5       []rankHit          `json:"primary_top5"`
	PrimaryBodyProofs []PrimaryBodyProof `json:"primary_body_proofs"`
	StageAFacts       []Fact             `json:"stage_a_facts"`
	Bundle            Bundle             `json:"bundle"`
	Related           []RelatedBody      `json:"related_bodies"`
	Baseline          eval.CaseResult    `json:"baseline_after_primary"`
	Augmented         eval.CaseResult    `json:"augmented_after_related"`
	Attachments       []ParentAttachment `json:"attachments"`
	WalkXFFAttached   bool               `json:"walkxff_attached"`
	FirstLoss         string             `json:"first_loss"`
	AnchorEdge        *anchorEdgeTrace   `json:"anchor_edge,omitempty"`
	Frontier          *frontierTrace     `json:"frontier,omitempty"`
}

type anchorSelection struct {
	Ordinal             int      `json:"ordinal"`
	Selected            bool     `json:"selected"`
	SelectedOrdinal     int      `json:"selected_ordinal"`
	ParentID            string   `json:"parent_id"`
	DenseRank           int      `json:"dense_rank"`
	MatchedTokens       []string `json:"matched_tokens"`
	CoverageNumerator   int      `json:"coverage_numerator"`
	CoverageDenominator int      `json:"coverage_denominator"`
	QueryTokens         int      `json:"query_token_count"`
	ParentTokens        int      `json:"parent_token_count"`
}
type anchorGroup struct {
	QueryID    string            `json:"query_id"`
	Tokens     []string          `json:"query_tokens"`
	Candidates []anchorSelection `json:"candidates"`
	Anchors    []anchorSelection `json:"anchors"`
}
type anchorScore struct {
	anchorSelection
	coverageNumerator, coverageDenominator int
}
type fileRoleSlice struct {
	Occurrences    int `json:"occurrences"`
	DistinctSource int `json:"distinct_sources"`
}
type structuralTierCount struct {
	Occurrences     int                        `json:"occurrences"`
	DistinctSources int                        `json:"distinct_sources"`
	FileRoleSlices  map[FileRole]fileRoleSlice `json:"file_role_slices"`
}
type edgeStats struct {
	SourceID                             string                     `json:"source_parent_id"`
	TargetID                             string                     `json:"target_parent_id"`
	Kind                                 RelationKind               `json:"relation_kind"`
	Tier                                 StructuralTier             `json:"structural_tier"`
	EdgeOccurrences                      int                        `json:"edge_occurrences"`
	SourceStratumOccurrences             int                        `json:"source_stratum_occurrences"`
	SourceStratumDistinctTargets         int                        `json:"source_stratum_distinct_targets"`
	TargetIncomingStratumOccurrences     int                        `json:"target_incoming_stratum_occurrences"`
	TargetIncomingStratumDistinctSources int                        `json:"target_incoming_stratum_distinct_sources"`
	EdgeFileRoleSlices                   map[FileRole]fileRoleSlice `json:"edge_file_role_slices"`
	SourceStratumFileRoleSlices          map[FileRole]fileRoleSlice `json:"source_stratum_file_role_slices"`
	TargetIncomingStratumFileRoleSlices  map[FileRole]fileRoleSlice `json:"target_incoming_stratum_file_role_slices"`
	Best                                 Fact                       `json:"best_occurrence"`
}
type rankingComponent struct {
	Name        string `json:"name"`
	Order       string `json:"order"`
	Value       any    `json:"value,omitempty"`
	Numerator   *int   `json:"numerator,omitempty"`
	Denominator *int   `json:"denominator,omitempty"`
}
type anchorEdgeCandidate struct {
	Fact              Fact               `json:"fact"`
	AnchorOrdinal     int                `json:"anchor_selection_ordinal"`
	DirectionMismatch int                `json:"direction_mismatch"`
	Stats             edgeStats          `json:"stats"`
	RankingTuple      []rankingComponent `json:"ranking_tuple"`
	Rank              int                `json:"rank"`
	FirstDifference   string             `json:"first_differing_component,omitempty"`
}
type anchorEdgeDenominators struct {
	Anchors           int `json:"anchors"`
	DirectionalOneHop int `json:"directional_one_hop_facts"`
	DistinctEdges     int `json:"distinct_edge_candidates"`
	Selected          int `json:"selected"`
	AddedBeforeCap    int `json:"added_before_cap"`
	PackagedComplete  int `json:"packaged_complete"`
}
type anchorEdgeTrace struct {
	Group            anchorGroup            `json:"anchor_group"`
	DirectionalFacts []Fact                 `json:"directional_one_hop_facts"`
	Candidates       []anchorEdgeCandidate  `json:"edge_candidate_stats"`
	Denominators     anchorEdgeDenominators `json:"denominators"`
	StagePresence    map[string]bool        `json:"stage_presence"`
}
type frontierBucket struct {
	AnchorOrdinal int            `json:"anchor_ordinal"`
	Direction     Direction      `json:"direction"`
	Tier          StructuralTier `json:"structural_tier"`
}
type frontierEdge struct {
	CanonicalEdgeID string              `json:"canonical_edge_id"`
	Bucket          frontierBucket      `json:"bucket"`
	Candidate       anchorEdgeCandidate `json:"candidate"`
	DirectBridge    bool                `json:"direct_anchor_bridge"`
}
type frontierBridgeReservation struct {
	CanonicalEdgeID          string         `json:"canonical_edge_id"`
	Outcome                  string         `json:"outcome"`
	EligibleBucket           frontierBucket `json:"eligible_bucket"`
	DisplacedCanonicalEdgeID string         `json:"displaced_canonical_edge_id"`
}
type frontierBridgeDisplacement struct {
	BridgeCanonicalEdgeID    string         `json:"bridge_canonical_edge_id"`
	DisplacedCanonicalEdgeID string         `json:"displaced_canonical_edge_id"`
	Bucket                   frontierBucket `json:"bucket"`
}
type frontierCounts struct {
	RawDirectionalFacts               int `json:"raw_directional_facts"`
	SelfFactsRemoved                  int `json:"self_facts_removed"`
	NonSelfDirectionalFacts           int `json:"non_self_directional_facts"`
	BucketDistinctCanonicalViews      int `json:"bucket_distinct_canonical_views"`
	RepeatedOccurrenceCollapse        int `json:"repeated_occurrence_collapse"`
	GlobalCanonicalUniverseEdges      int `json:"global_canonical_universe_edges"`
	UniverseCrossBucketDuplicateViews int `json:"universe_cross_bucket_duplicate_views"`
	BucketTruncations                 int `json:"bucket_truncations"`
	ProvisionalRows                   int `json:"provisional_rows"`
	PostCapCrossBucketDuplicates      int `json:"post_cap_cross_bucket_duplicates"`
	FinalRetained                     int `json:"final_retained"`
	// Legacy field names retain the explicit semantics above for old readers.
	CanonicalEdges         int `json:"canonical_edges"`
	OccurrenceCollapse     int `json:"occurrence_collapse"`
	ProvisionalBucketEdges int `json:"provisional_bucket_edges"`
	CanonicalUnionEdges    int `json:"canonical_union_edges"`
	CrossBucketDuplicates  int `json:"cross_bucket_duplicates"`
	ReservedBridgeEdges    int `json:"reserved_bridge_edges"`
	RetainedFrontierEdges  int `json:"retained_frontier_edges"`
	TruncatedEdges         int `json:"truncated_edges"`
}
type frontierBucketCounts struct {
	PreCapDistinct int `json:"pre_cap_distinct"`
	Retained       int `json:"retained"`
	Truncated      int `json:"truncated"`
}
type frontierTrace struct {
	Policy              string                          `json:"policy"`
	Group               anchorGroup                     `json:"anchor_group"`
	Counts              frontierCounts                  `json:"counts"`
	Buckets             map[string][]frontierEdge       `json:"buckets"`
	BucketCounts        map[string]frontierBucketCounts `json:"bucket_counts"`
	Provisional         []frontierEdge                  `json:"provisional_top2_per_bucket"`
	CanonicalUnion      []frontierEdge                  `json:"canonical_union"`
	BridgeReservations  []frontierBridgeReservation     `json:"bridge_reservations"`
	BridgeDisplacements []frontierBridgeDisplacement    `json:"bridge_displacements"`
	FinalFrontier       []frontierEdge                  `json:"final_frontier"`
	FinalDigest         string                          `json:"final_frontier_sha256"`
	CapReached          bool                            `json:"cap_reached"`
	AbstentionReason    string                          `json:"abstention_reason,omitempty"`
	FirstLossReason     string                          `json:"first_loss_reason,omitempty"`
	Selected            *frontierEdge                   `json:"selected,omitempty"`
	GraphOnly           *frontierGraphOnlyTrace         `json:"graph_only,omitempty"`
}
type frontierParetoCandidate struct {
	CanonicalEdgeID                      string   `json:"canonical_edge_id"`
	EdgeOccurrences                      int      `json:"edge_occurrences"`
	SourceStratumOccurrences             int      `json:"source_stratum_occurrences"`
	SourceStratumDistinctTargets         int      `json:"source_stratum_distinct_targets"`
	TargetIncomingStratumDistinctSources int      `json:"target_incoming_stratum_distinct_sources"`
	Nondominated                         bool     `json:"nondominated"`
	DominatedBy                          []string `json:"dominated_by"`
}
type frontierParetoTier struct {
	Candidates []frontierParetoCandidate `json:"candidates"`
}
type frontierGraphOnlyTrace struct {
	DirectBridgeCandidates int                                   `json:"direct_bridge_candidates"`
	IncomingExcluded       int                                   `json:"incoming_excluded"`
	DenseEndpointExcluded  int                                   `json:"dense_endpoint_excluded"`
	GraphOnlyCandidates    int                                   `json:"graph_only_candidates"`
	Tiers                  map[StructuralTier]frontierParetoTier `json:"tiers"`
	UnionCount             int                                   `json:"union_count"`
	Outcome                string                                `json:"outcome"`
	Selected               *frontierEdge                         `json:"selected,omitempty"`
}
type PrimaryBodyProof struct {
	ParentID   string `json:"parent_id"`
	BodySHA256 string `json:"body_sha256"`
}
type ParentAttachment struct {
	ParentID       string `json:"parent_id"`
	Role           string `json:"role"`
	Required       bool   `json:"required"`
	Classification string `json:"classification"`
}
type DiagnosticGateReason struct {
	ParentID        string   `json:"parent_id"`
	QualifiedSymbol string   `json:"qualified_symbol"`
	Reason          string   `json:"reason"`
	QueryIDs        []string `json:"query_ids"`
}
type DiagnosticGate struct {
	Eligible bool                   `json:"eligible"`
	Reasons  []DiagnosticGateReason `json:"reasons"`
}
type EvaluationResult struct {
	RunID     string `json:"run_id"`
	Reference string `json:"reference"`
	Queries   int    `json:"queries"`
}
type evaluationBinding struct {
	GraphLogicalSHA256         string         `json:"graph_logical_sha256"`
	GraphCorpusID              string         `json:"graph_corpus_id"`
	ReplayCorpusID             string         `json:"replay_corpus_id"`
	ReplaySHA256               string         `json:"replay_sha256"`
	DatasetSHA256              string         `json:"dataset_sha256"`
	ExpectedDatasetSHA256      string         `json:"expected_dataset_sha256"`
	ExpectedDatasetFingerprint string         `json:"expected_dataset_fingerprint"`
	ProbesSHA256               string         `json:"probes_sha256"`
	DenseLane                  string         `json:"dense_lane"`
	DenseDepth                 int            `json:"dense_depth"`
	SelectionPolicy            string         `json:"selection_policy"`
	MetadataPolicy             string         `json:"metadata_policy"`
	PolicyFingerprint          string         `json:"policy_fingerprint"`
	QueryFeaturesSHA256        string         `json:"query_features_sha256"`
	Evaluator                  buildinfo.Info `json:"evaluator"`
}

// Evaluate replays only immutable dense1024/int8 ranks. It intentionally does
// all graph reachability, deterministic selection, and body-fit work before
// opening the frozen dataset labels.
func Evaluate(ctx context.Context, request EvaluationRequest) (EvaluationResult, error) {
	if err := validateEvaluationRequest(request); err != nil {
		return EvaluationResult{}, err
	}
	if !cleanKnownRelationBuild(buildinfo.Current()) {
		return EvaluationResult{}, fmt.Errorf("relation diagnostic requires a clean known build")
	}
	parents, err := ParentInventory(request.Parents.Parents)
	if err != nil {
		return EvaluationResult{}, err
	}
	if err := verifyChecksums(request.GraphDirectory, []string{"graph-manifest.json", "relations.db", "resolution-summary.json"}); err != nil {
		return EvaluationResult{}, fmt.Errorf("relation graph artifact checksum verification: %w", err)
	}
	manifest, err := loadGraphManifest(filepath.Join(request.GraphDirectory, "graph-manifest.json"))
	if err != nil {
		return EvaluationResult{}, err
	}
	if err := validateGraphBinding(request.GraphDirectory, manifest, parents, request.Parents); err != nil {
		return EvaluationResult{}, err
	}
	replay, err := loadReplay(request.ReplayPath)
	if err != nil {
		return EvaluationResult{}, err
	}
	if manifest.Corpus.CorpusID != replay.CorpusID {
		return EvaluationResult{}, fmt.Errorf("graph and replay corpus mismatch")
	}
	datasetSourceSHA, err := fileSHA256(request.DatasetPath)
	if err != nil {
		return EvaluationResult{}, err
	}
	if replay.SourceSHA256["dataset"] != datasetSourceSHA {
		return EvaluationResult{}, fmt.Errorf("frozen replay dataset digest mismatch")
	}
	replaySHA, err := fileSHA256(request.ReplayPath)
	if err != nil {
		return EvaluationResult{}, err
	}
	probesSHA, err := fileSHA256(request.ProbesPath)
	if err != nil {
		return EvaluationResult{}, err
	}
	selectionPolicy := selectedPolicy(request.SelectionPolicy)
	policyFingerprint, err := relationPolicyFingerprint(selectionPolicy)
	if err != nil {
		return EvaluationResult{}, err
	}
	binding := evaluationBinding{GraphLogicalSHA256: manifest.LogicalGraphSHA256, GraphCorpusID: manifest.Corpus.CorpusID, ReplayCorpusID: replay.CorpusID, ReplaySHA256: replaySHA, DatasetSHA256: datasetSourceSHA, ExpectedDatasetSHA256: replay.SourceSHA256["dataset"], ExpectedDatasetFingerprint: replay.DatasetFingerprint, ProbesSHA256: probesSHA, DenseLane: "dense_1024_int8", DenseDepth: MaxDenseDepth, SelectionPolicy: selectionPolicy, MetadataPolicy: MetadataPolicyID, PolicyFingerprint: policyFingerprint, Evaluator: buildinfo.Current()}
	lane, ok := replay.Lanes["dense_1024_int8"]
	if !ok || len(lane.Ranks) == 0 {
		return EvaluationResult{}, fmt.Errorf("missing frozen dense_1024_int8 ranks")
	}
	if err := validateReplayRanks(lane); err != nil {
		return EvaluationResult{}, err
	}
	if selectionPolicy == GraphFirstPolicyID {
		if err := validateGraphFirstLanes(replay, lane); err != nil {
			return EvaluationResult{}, err
		}
	}
	features, err := loadQueryFeatures(request.DatasetPath)
	if err != nil {
		return EvaluationResult{}, err
	}
	if len(features) != len(lane.Ranks) {
		return EvaluationResult{}, fmt.Errorf("query-feature/replay cardinality mismatch")
	}
	featureDigest, err := canonicalHash(features)
	if err != nil {
		return EvaluationResult{}, err
	}
	binding.QueryFeaturesSHA256 = featureDigest
	for queryID := range lane.Ranks {
		if _, ok := features[queryID]; !ok {
			return EvaluationResult{}, fmt.Errorf("replay query absent from query features")
		}
	}
	db, err := openImmutableGraph(filepath.Join(request.GraphDirectory, "relations.db"))
	if err != nil {
		return EvaluationResult{}, err
	}
	defer db.Close()
	if err := graphIntegrity(ctx, db); err != nil {
		return EvaluationResult{}, err
	}
	parents, err = loadGraphParentTraits(ctx, db, parents)
	if err != nil {
		return EvaluationResult{}, err
	}
	byID, byHit := parentMaps(parents)
	resolution, err := resolutionDenominators(ctx, db)
	if err != nil {
		return EvaluationResult{}, err
	}
	policy := selectedPolicy(request.SelectionPolicy)
	var completeStats map[string]edgeStats
	var tierCounts map[StructuralTier]structuralTierCount
	if isAnchorEdgePolicy(policy) || isFrontierPolicy(policy) {
		completeStats, tierCounts, err = completeGraphEdgeStats(ctx, db)
		if err != nil {
			return EvaluationResult{}, err
		}
	}
	traces := make([]queryTrace, 0, len(lane.Ranks))
	for _, queryID := range sortedKeys(lane.Ranks) {
		primary, primaryIDs, seedIDs, err := primaryTop5(lane.Ranks[queryID], byHit)
		if err != nil {
			return EvaluationResult{}, fmt.Errorf("%s: %w", queryID, err)
		}
		if policy == GraphFirstPolicyID {
			seedIDs, err = graphFirstSeeds(replay, queryID, byHit)
			if err != nil {
				return EvaluationResult{}, err
			}
		}
		var group anchorGroup
		if isAnchorEdgePolicy(policy) || isFrontierPolicy(policy) {
			group, err = selectAnchorGroup(queryID, features[queryID].AnchorTokens, lane.Ranks[queryID], byHit, byID)
			if err != nil {
				return EvaluationResult{}, err
			}
			seedIDs = make([]string, 0, len(group.Anchors))
			for _, anchor := range group.Anchors {
				seedIDs = append(seedIDs, anchor.ParentID)
			}
		}
		facts, err := reachableFacts(ctx, db, seedIDs)
		if err != nil {
			return EvaluationResult{}, err
		}
		ranks := rankPositions(lane.Ranks[queryID], byHit)
		var bundle Bundle
		var anchorTrace *anchorEdgeTrace
		var frontierTrace *frontierTrace
		if isFrontierPolicy(policy) {
			bundle, frontierTrace, err = selectFrontierBundle(queryID, features[queryID], group, facts, completeStats, ranks, byID, primaryIDs, policy)
			if err != nil {
				return EvaluationResult{}, err
			}
		} else if isAnchorEdgePolicy(policy) {
			bundle, anchorTrace, err = selectAnchorEdgeBundle(queryID, features[queryID], group, facts, completeStats, ranks, byID, primaryIDs, policy)
			if err != nil {
				return EvaluationResult{}, err
			}
		} else {
			bundle = selectBundleWithPolicy(queryID, features[queryID], facts, ranks, byID, primaryIDs, policy)
		}
		related := packageRelated(queryID, bundle, byID, primaryIDs)
		if anchorTrace != nil {
			anchorTrace.Denominators.AddedBeforeCap = len(bundle.AddedParentIDs)
			for _, body := range related {
				if body.BodyComplete {
					anchorTrace.Denominators.PackagedComplete++
				}
			}
			anchorTrace.StagePresence["cap"] = len(bundle.AddedParentIDs) > 0
			anchorTrace.StagePresence["packaging"] = anchorTrace.Denominators.PackagedComplete > 0
		}
		proofs := make([]PrimaryBodyProof, 0, len(primaryIDs))
		for _, id := range primaryIDs {
			parent := byID[id]
			proofs = append(proofs, PrimaryBodyProof{ParentID: id, BodySHA256: sha256Hex([]byte(parent.SourceBody))})
		}
		traces = append(traces, queryTrace{QueryID: queryID, PrimaryTop5: primary, PrimaryBodyProofs: proofs, StageAFacts: facts, Bundle: bundle, Related: related, AnchorEdge: anchorTrace, Frontier: frontierTrace})
	}
	// Labels are deliberately unavailable until the previous loop has finished.
	datasetBytes, err := os.ReadFile(request.DatasetPath)
	if err != nil {
		return EvaluationResult{}, err
	}
	dataset, err := eval.LoadDataset(datasetBytes)
	if err != nil {
		return EvaluationResult{}, err
	}
	if dataset.CorpusID != replay.CorpusID || dataset.CorpusID != manifest.Corpus.CorpusID {
		return EvaluationResult{}, fmt.Errorf("dataset and replay corpus mismatch")
	}
	datasetFingerprint, err := dataset.Fingerprint()
	if err != nil {
		return EvaluationResult{}, err
	}
	if datasetFingerprint != replay.DatasetFingerprint {
		return EvaluationResult{}, fmt.Errorf("frozen replay dataset fingerprint mismatch")
	}
	caseByID := map[string]int{}
	for index, item := range dataset.Cases {
		caseByID[item.ID] = index
	}
	if len(caseByID) != len(lane.Ranks) {
		return EvaluationResult{}, fmt.Errorf("dataset/replay query-set cardinality mismatch")
	}
	for queryID := range lane.Ranks {
		if _, ok := caseByID[queryID]; !ok {
			return EvaluationResult{}, fmt.Errorf("replay query absent from dataset")
		}
	}
	for index := range traces {
		trace := &traces[index]
		caseIndex, ok := caseByID[trace.QueryID]
		if !ok {
			return EvaluationResult{}, fmt.Errorf("replay query absent from dataset")
		}
		item := dataset.Cases[caseIndex]
		baselineHits := toLexical(trace.PrimaryTop5)
		baseline, err := eval.EvaluateCase(item, baselineHits, []int{len(baselineHits)}, nil)
		if err != nil {
			return EvaluationResult{}, err
		}
		combined := append([]lexical.Hit(nil), baselineHits...)
		for _, body := range trace.Related {
			if body.BodyComplete {
				combined = append(combined, parentHit(byID[body.ParentID]))
			}
		}
		augmented, err := eval.EvaluateCase(item, combined, []int{len(combined)}, nil)
		if err != nil {
			return EvaluationResult{}, err
		}
		trace.Baseline, trace.Augmented = baseline, augmented
		trace.Attachments = classifyAttachments(item, trace.Bundle, trace.Related, byID)
		for _, attachment := range trace.Attachments {
			if parent, ok := byID[attachment.ParentID]; ok && parent.QualifiedSymbol == "middleware.walkXFF" {
				trace.WalkXFFAttached = true
			}
		}
		trace.FirstLoss = relationFirstLoss(item, *trace, byID)
	}
	probes, err := evaluateProbes(ctx, db, request.ProbesPath, byID, manifest.Corpus.CorpusID)
	if err != nil {
		return EvaluationResult{}, err
	}
	gate := diagnosticGate(traces, byID)
	target := filepath.Join(request.EvaluationRoot, request.RunID)
	if _, err := os.Lstat(target); err == nil {
		return EvaluationResult{}, fmt.Errorf("relation diagnostic artifact already exists")
	} else if !os.IsNotExist(err) {
		return EvaluationResult{}, err
	}
	temporary, err := os.MkdirTemp(request.EvaluationRoot, ".relation-diagnostic-")
	if err != nil {
		return EvaluationResult{}, err
	}
	defer os.RemoveAll(temporary)
	if err := writeEvaluationArtifacts(temporary, traces, features, probes, resolution, gate, manifest, replay, binding, tierCounts); err != nil {
		return EvaluationResult{}, err
	}
	if err := writeChecksums(temporary); err != nil {
		return EvaluationResult{}, err
	}
	if err := os.Rename(temporary, target); err != nil {
		return EvaluationResult{}, err
	}
	return EvaluationResult{RunID: request.RunID, Reference: filepath.ToSlash(filepath.Join("evaluations", request.RunID)), Queries: len(traces)}, nil
}

func cleanKnownRelationBuild(info buildinfo.Info) bool {
	if info.SourceModified != "false" || (len(info.Commit) != 40 && len(info.Commit) != 64) {
		return false
	}
	for _, value := range info.Commit {
		if !(value >= '0' && value <= '9' || value >= 'a' && value <= 'f') {
			return false
		}
	}
	return true
}

func selectedPolicy(value string) string {
	if value == "" {
		return DenseFirstPolicyID
	}
	return value
}

func relationPolicyFingerprint(policy string) (string, error) {
	return canonicalHash(relationPolicySpec(policy))
}

// relationPolicySpec keeps every pre-existing policy serialization immutable.
// The graph-only experiment carries its own additive spec only when selected.
func relationPolicySpec(policies ...string) map[string]any {
	policy := DenseFirstPolicyID
	if len(policies) > 0 {
		policy = policies[0]
	}
	spec := map[string]any{
		"metadata_policy":                   MetadataPolicyID,
		"selection_policies":                []string{DenseFirstPolicyID, ValueParameterDenseFirstPolicyID, GraphFirstPolicyID, AnchorEdgeRawFrequencyPolicyID, AnchorEdgeSourceNormalizedPolicyID, AnchorEdgeBidirectionalPolicyID, AnchorEdgeIncomingPopularityPolicyID, AnchorFrontierCapOnlyPolicyID, AnchorFrontierBridgePolicyID},
		"dense_first_tuple":                 []string{"qualifier", "negative_context_overlap", "negative_endpoint_overlap", "negative_anchor_overlap", "intent_mismatch", "same_file", "occurrence_file_role", "anchor_dense_rank", "endpoint_dense_rank", "source_ordinal", "occurrence_byte", "stable_id"},
		"value_parameter_dense_first_tuple": []string{"qualifier", "value_parameter_mismatch", "negative_context_overlap", "negative_endpoint_overlap", "negative_anchor_overlap", "intent_mismatch", "same_file", "occurrence_file_role", "anchor_dense_rank", "endpoint_dense_rank", "source_ordinal", "occurrence_byte", "stable_id"},
		"graph_first":                       map[string]any{"seed_lanes": []string{"fts", "simple_control"}, "seed_k": ProtectedPrimaryK, "admission_prefix": []string{"qualifier", "negative_context_overlap", "negative_endpoint_overlap", "negative_anchor_overlap", "intent_mismatch", "same_file", "occurrence_file_role"}, "admitted_tier": "all facts tied at best prefix", "rerank_tuple": []string{"best_dense_endpoint_ordinal", "worst_dense_endpoint_ordinal", "source_ordinal", "occurrence_byte", "stable_id"}},
		"anchor_edge": map[string]any{
			"policies":   []string{AnchorEdgeRawFrequencyPolicyID, AnchorEdgeSourceNormalizedPolicyID, AnchorEdgeBidirectionalPolicyID, AnchorEdgeIncomingPopularityPolicyID},
			"anchors":    "query_tokens=stable_distinct(symbol.ClassifyQuery IdentifierTokens then TextTokens);parent_tokens=stable_distinct(existing_identifier_normalizer(symbol+qualified_symbol));deduped_dense_top20;all_candidates_sorted_by_coverage_then_matched_count_then_dense_rank_then_parent_id;select=min(2,available);coverage=matched_query_tokens/parent_tokens desc;matched_count desc;dense_rank asc;parent_id asc;checked_cross_multiplication;zero_coverage_falls_through;anchor_limit=2",
			"facts":      "both_outgoing_and_incoming_resolved_unique_one_hop;stored_orientation_source_target_relation_kind_structural_tier;no_self_edge_no_second_hop;direction_mismatch_shared_not_filtering",
			"tiers":      []string{"DECLARATION_CONTRACT=TYPE_REF:SIGNATURE|TYPE_BODY", "EXECUTABLE_DEPENDENCY=CALLS", "BODY_REFERENCE=TYPE_REF:BODY|INITIALIZER", "DECLARATION_STRUCTURE=MEMBER_OF"},
			"statistics": "complete_non_self_graph;stratum=relation_kind+structural_tier;edge_occurrences;source_occurrences;source_distinct_targets;target_incoming_occurrences;target_incoming_distinct_sources;file_role_occurrence_and_distinct_source_slices_not_ranked;checked_integer_cross_products",
			"ranking": map[string][]string{
				AnchorEdgeRawFrequencyPolicyID:       anchorRankingComponentSequence(AnchorEdgeRawFrequencyPolicyID),
				AnchorEdgeSourceNormalizedPolicyID:   anchorRankingComponentSequence(AnchorEdgeSourceNormalizedPolicyID),
				AnchorEdgeBidirectionalPolicyID:      anchorRankingComponentSequence(AnchorEdgeBidirectionalPolicyID),
				AnchorEdgeIncomingPopularityPolicyID: anchorRankingComponentSequence(AnchorEdgeIncomingPopularityPolicyID),
			},
			"visibility": "separate_and_unused",
		},
		"anchor_frontier": map[string]any{
			"policies":           []string{AnchorFrontierCapOnlyPolicyID, AnchorFrontierBridgePolicyID},
			"shared":             "anchor_selection=anchor_edge;uncapped_resolved_non_self_one_hop_canonical_typed_tier_edges;bucket=anchor_ordinal_x_direction_x_structural_tier;bucket_order=anchor_edge_bidirectional_specificity_then_canonical_edge;provisional=top2_per_bucket;bridge_reservation_happens_within_one_eligible_bucket_before_canonical_union;canonical_union_without_backfill;global_cap=32;bundle_cap_and_packaging=anchor_edge",
			"bridge_reservation": "unique_canonical_direct_edges_between_selected_anchors;eligible_buckets_from_uncapped_universe;deterministic_bridge_and_bucket_order=anchor_edge_bidirectional_specificity_then_canonical_edge;survived_bridge_keeps_slot;reserved_bridge_replaces_worst_non_bridge_or_uses_unfilled_slot;overflow=BRIDGE_CAP_OVERFLOW;record_outcome_eligible_bucket_and_concrete_displaced_canonical_edge",
			"digest":             "canonical_ordered_final_frontier_identity_sha256",
			"observability":      "raw_input_directional_facts;self_facts_removed;non_self_directional_facts;sum_bucket_distinct_canonical_views;repeated_occurrence_collapse=non_self_minus_bucket_views;global_canonical_universe_edges;universe_cross_bucket_duplicate_views=bucket_views_minus_global_universe;per_bucket_pre_cap_retained_truncated;total_bucket_truncations;provisional_rows;post_cap_cross_bucket_duplicates;final_retained",
			"arms":               map[string]string{AnchorFrontierCapOnlyPolicyID: "select_deterministic_top_final_frontier_edge", AnchorFrontierBridgePolicyID: "select_deterministic_direct_anchor_bridge_from_final_frontier_else_NO_DIRECT_ANCHOR_BRIDGE"},
		},
		"keyword_maps":    map[string][]string{"signature": {"contract", "props", "type", "interface", "signature", "options", "schema"}, "value_parameter": {"props", "contract"}, "mutation": {"mutate", "set", "write", "assign"}, "return": {"return"}, "condition": {"condition", "when"}, "reverse": {"caller", "used-by"}, "deprecated": {"deprecated"}, "file_roles": {"test", "example", "benchmark"}},
		"caps":            map[string]int{"dense_depth": MaxDenseDepth, "protected_primary": ProtectedPrimaryK, "related_parent_limit": RelatedParentLimit, "related_body_limit": RelatedBodyLimit, "context_identifier_limit": 8},
		"qualifier_rules": []string{"deprecated_query_requires_either_endpoint_deprecated", "no_deprecated_query_penalty", "explicit_file_role_uses_occurrence_file_role", "default_production_uses_occurrence_file_role"},
		"role_rules":      []string{"resolved_distinct_endpoints_only", "selection_never_reads_source_body", "package_related_applies_complete_body_cap"},
		"occurrence_zone": []OccurrenceZone{SignatureZone, BodyZone, TypeBodyZone, InitializerZone},
		"occurrence_role": []OccurrenceRole{CallFreeFunctionRole, CallMethodRole, CallableValueRole, TypeParameterRole, TypeValueParameterRole, TypeReturnRole, TypeFieldRole, TypeAliasRole, TypeHeritageRole, TypeArgumentRole, TypeLocalRole, TypeOtherRole, MemberReceiverRole, MemberDeclarationRole},
		"flow_role":       []FlowRole{FlowNone, FlowReturn, FlowAssignment, FlowCondition, FlowArgument, FlowDeclaration},
		"file_role":       []FileRole{ProductionFileRole, TestFileRole, ExampleFileRole, BenchmarkFileRole},
		"execution_mode":  []ExecutionMode{DirectExecution, DeferredExecution, ConcurrentExecution, AwaitedExecution},
		"control_role":    []ControlRole{ControlNone, ControlBranch, ControlLoop, ControlSwitch, ControlTryCatch},
	}
	if policy == AnchorFrontierGraphOnlyParetoPolicyID {
		spec["selection_policies"] = append(spec["selection_policies"].([]string), AnchorFrontierGraphOnlyParetoPolicyID)
		spec["anchor_frontier_graph_only_pareto"] = map[string]any{
			"policy":     AnchorFrontierGraphOnlyParetoPolicyID,
			"source":     "existing_final_frontier_only",
			"admission":  "first_direct_selected_anchor_bridge_in_frontier_order_else_forward_edges_with_endpoint_absent_from_frozen_dense_top20",
			"pareto":     "within_structural_tier:max(edge_occurrences/source_stratum_occurrences)_exact_rational,min(source_stratum_distinct_targets),min(target_incoming_stratum_distinct_sources);nondominated=no_worse_all_and_one_strict;no_cross_tier_dominance",
			"outcomes":   []string{"DIRECT_BRIDGE", "NO_CANDIDATE", "ONE_WINNER", "MULTIPLE_WINNERS"},
			"artifacts":  "frontier-graph-only-pareto.jsonl;frontier-graph-only-pareto-denominators.json;per-query-relation-trace.jsonl",
			"validation": "bridge_precedence;full_final_frontier_partition;per_tier_exact_pareto;union_outcome;per_query_and_aggregate_denominators",
		}
	}
	return spec
}

func validateEvaluationRequest(v EvaluationRequest) error {
	if !strings.HasPrefix(v.RunID, "relation-diagnostic-") || !validRelative(v.RunID) || v.EvaluationRoot == "" || v.GraphDirectory == "" || v.ReplayPath == "" || v.DatasetPath == "" || v.ProbesPath == "" || !validSelectionPolicy(v.SelectionPolicy) {
		return fmt.Errorf("invalid relation diagnostic evaluation request")
	}
	for _, dir := range []string{v.EvaluationRoot, v.GraphDirectory} {
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe relation diagnostic directory")
		}
	}
	return nil
}

func validSelectionPolicy(policy string) bool {
	switch policy {
	case "", DenseFirstPolicyID, ValueParameterDenseFirstPolicyID, GraphFirstPolicyID, AnchorEdgeRawFrequencyPolicyID, AnchorEdgeSourceNormalizedPolicyID, AnchorEdgeBidirectionalPolicyID, AnchorEdgeIncomingPopularityPolicyID, AnchorFrontierCapOnlyPolicyID, AnchorFrontierBridgePolicyID, AnchorFrontierGraphOnlyParetoPolicyID:
		return true
	}
	return false
}

func isFrontierPolicy(policy string) bool {
	return policy == AnchorFrontierCapOnlyPolicyID || policy == AnchorFrontierBridgePolicyID || policy == AnchorFrontierGraphOnlyParetoPolicyID
}

func isAnchorEdgePolicy(policy string) bool {
	switch policy {
	case AnchorEdgeRawFrequencyPolicyID, AnchorEdgeSourceNormalizedPolicyID, AnchorEdgeBidirectionalPolicyID, AnchorEdgeIncomingPopularityPolicyID:
		return true
	}
	return false
}
func loadReplay(file string) (frozenReplay, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return frozenReplay{}, err
	}
	var value frozenReplay
	if err := json.Unmarshal(data, &value); err != nil {
		return frozenReplay{}, err
	}
	if value.SchemaVersion != 1 || value.Kind != "cidx.provider_free_frozen_label_replay.v1" || value.CorpusID == "" || !validDigest(value.DatasetFingerprint) || value.ReviewProtocol != "owner-adopted-dual-ai-v1" || value.RelevanceAuthority != "OWNER_ADOPTED_DUAL_AI_REVIEW" || value.ReviewValidation != "NO_INDEPENDENT_HUMAN_REVIEW" || !validDigest(value.SourceSHA256["dataset"]) {
		return frozenReplay{}, fmt.Errorf("invalid frozen replay")
	}
	return value, nil
}

func validateReplayRanks(lane struct {
	Ranks map[string][]rankHit `json:"ranks"`
}) error {
	seenQueries := map[string]bool{}
	for queryID, hits := range lane.Ranks {
		if queryID == "" || seenQueries[queryID] || len(hits) != MaxDenseDepth {
			return fmt.Errorf("invalid frozen replay query/depth")
		}
		seenQueries[queryID] = true
		seenHits := map[string]bool{}
		for index, hit := range hits {
			if hit.Rank != index+1 || !validRelative(hit.Path) || !validDigest(hit.IndexedSHA256) || hit.QualifiedSymbol == "" || hit.StartByte < 0 || hit.EndByte <= hit.StartByte {
				return fmt.Errorf("invalid frozen replay rank")
			}
			key := hitKey(hit.Path, hit.IndexedSHA256, hit.QualifiedSymbol, hit.StartByte, hit.EndByte)
			if seenHits[key] {
				return fmt.Errorf("duplicate frozen replay rank")
			}
			seenHits[key] = true
		}
	}
	return nil
}

func validateGraphFirstLanes(replay frozenReplay, dense struct {
	Ranks map[string][]rankHit `json:"ranks"`
}) error {
	for _, name := range []string{"fts", "simple_control"} {
		lane, ok := replay.Lanes[name]
		if !ok || len(lane.Ranks) != len(dense.Ranks) {
			return fmt.Errorf("invalid frozen %s graph-first lane", name)
		}
		for queryID := range dense.Ranks {
			hits, ok := lane.Ranks[queryID]
			if !ok || len(hits) < ProtectedPrimaryK {
				return fmt.Errorf("invalid frozen %s graph-first query set", name)
			}
			seen := map[string]bool{}
			for index, hit := range hits {
				if hit.Rank != index+1 || !validRelative(hit.Path) || !validDigest(hit.IndexedSHA256) || hit.QualifiedSymbol == "" || hit.StartByte < 0 || hit.EndByte <= hit.StartByte {
					return fmt.Errorf("invalid frozen %s graph-first rank", name)
				}
				key := hitKey(hit.Path, hit.IndexedSHA256, hit.QualifiedSymbol, hit.StartByte, hit.EndByte)
				if seen[key] {
					return fmt.Errorf("duplicate frozen %s graph-first rank", name)
				}
				seen[key] = true
			}
		}
	}
	return nil
}

func loadGraphParentTraits(ctx context.Context, db *sql.DB, parents []Parent) ([]Parent, error) {
	// Selection consumes only these frozen graph traits. SourceBody remains for
	// post-selection packaging and is never consulted by admission or ordering.
	traits := map[string]struct {
		role       FileRole
		deprecated bool
	}{}
	rows, err := db.QueryContext(ctx, `SELECT parent_id,file_role,deprecated FROM parent_traits ORDER BY parent_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, role string
		var deprecated int
		if err := rows.Scan(&id, &role, &deprecated); err != nil {
			return nil, err
		}
		value := FileRole(role)
		if id == "" || !value.Valid() || (deprecated != 0 && deprecated != 1) {
			return nil, fmt.Errorf("invalid graph parent trait")
		}
		if _, exists := traits[id]; exists {
			return nil, fmt.Errorf("duplicate graph parent trait")
		}
		traits[id] = struct {
			role       FileRole
			deprecated bool
		}{value, deprecated == 1}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(traits) != len(parents) {
		return nil, fmt.Errorf("graph parent trait cardinality mismatch")
	}
	for index := range parents {
		value, ok := traits[parents[index].ID]
		if !ok {
			return nil, fmt.Errorf("missing graph parent trait")
		}
		parents[index].FileRole, parents[index].Deprecated = value.role, value.deprecated
	}
	return parents, nil
}
func loadGraphManifest(file string) (GraphManifest, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return GraphManifest{}, err
	}
	var value GraphManifest
	if err := json.Unmarshal(data, &value); err != nil {
		return GraphManifest{}, err
	}
	if value.SchemaVersion != SchemaVersion || value.Kind != "cidx.relation_graph.v3" || value.Corpus.CorpusID == "" || !validDigest(value.LogicalGraphSHA256) || !validDigest(value.DatabaseSHA256) || !validDigest(value.SemanticParentInventorySHA256) || !validDigest(value.IndexedFileInventorySHA256) || value.ResolverPolicy["protocol"] != ProtocolVersion || value.ResolverPolicy["metadata"] != MetadataPolicyID {
		return GraphManifest{}, fmt.Errorf("invalid graph manifest")
	}
	return value, nil
}
func validDigest(value string) bool {
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
func validateGraphBinding(graphDir string, manifest GraphManifest, parents []Parent, snapshot store.SemanticParentSnapshot) error {
	if digest, err := fileSHA256(filepath.Join(graphDir, "relations.db")); err != nil || digest != manifest.DatabaseSHA256 {
		return fmt.Errorf("relation graph database digest mismatch")
	}
	parentHash, err := inventoryHash(parents)
	if err != nil {
		return err
	}
	if manifest.IndexGeneration != snapshot.Generation || manifest.IndexManifestSHA256 != snapshot.ManifestSHA256 || manifest.SemanticParentInventorySHA256 != parentHash {
		return fmt.Errorf("relation graph binding mismatch")
	}
	db, err := openImmutableGraph(filepath.Join(graphDir, "relations.db"))
	if err != nil {
		return err
	}
	defer db.Close()
	var logical string
	if err := db.QueryRow(`SELECT value FROM graph_meta WHERE key='logical_graph_sha256'`).Scan(&logical); err != nil || logical != manifest.LogicalGraphSHA256 {
		return fmt.Errorf("relation graph logical digest mismatch")
	}
	return nil
}
func openImmutableGraph(file string) (*sql.DB, error) {
	if _, err := os.Stat(file); err != nil {
		return nil, err
	}
	return sql.Open("sqlite", "file:"+filepath.ToSlash(file)+"?mode=ro&immutable=1")
}
func parentMaps(parents []Parent) (map[string]Parent, map[string]string) {
	byID, byHit := map[string]Parent{}, map[string]string{}
	for _, parent := range parents {
		byID[parent.ID] = parent
		byHit[hitKey(parent.Path, parent.IndexedSHA256, parent.QualifiedSymbol, parent.StartByte, parent.EndByte)] = parent.ID
	}
	return byID, byHit
}
func hitKey(path, hash, symbol string, start, end int) string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%d", path, hash, symbol, start, end)
}
func primaryTop5(hits []rankHit, byHit map[string]string) ([]rankHit, []string, []string, error) {
	if len(hits) != MaxDenseDepth {
		return nil, nil, nil, fmt.Errorf("frozen rank depth must equal 20")
	}
	result := append([]rankHit(nil), hits[:ProtectedPrimaryK]...)
	primaryIDs, seedIDs := make([]string, 0, ProtectedPrimaryK), make([]string, 0, len(hits))
	for position, hit := range hits {
		if hit.Rank != position+1 {
			return nil, nil, nil, fmt.Errorf("invalid dense rank")
		}
		id, ok := byHit[hitKey(hit.Path, hit.IndexedSHA256, hit.QualifiedSymbol, hit.StartByte, hit.EndByte)]
		if !ok {
			return nil, nil, nil, fmt.Errorf("rank hit not in semantic parent inventory")
		}
		seedIDs = append(seedIDs, id)
		if position < ProtectedPrimaryK {
			primaryIDs = append(primaryIDs, id)
		}
	}
	return result, primaryIDs, seedIDs, nil
}
func rankPositions(hits []rankHit, byHit map[string]string) map[string]int {
	values := map[string]int{}
	for _, hit := range hits {
		if id, ok := byHit[hitKey(hit.Path, hit.IndexedSHA256, hit.QualifiedSymbol, hit.StartByte, hit.EndByte)]; ok {
			if existing, seen := values[id]; !seen || hit.Rank < existing {
				values[id] = hit.Rank
			}
		}
	}
	return values
}

func selectAnchorGroup(queryID string, tokens []string, hits []rankHit, byHit map[string]string, parents map[string]Parent) (anchorGroup, error) {
	if len(hits) != MaxDenseDepth || len(tokens) == 0 {
		return anchorGroup{}, fmt.Errorf("invalid anchor selection input")
	}
	seen := map[string]bool{}
	values := make([]anchorScore, 0, MaxDenseDepth)
	for _, hit := range hits {
		id, ok := byHit[hitKey(hit.Path, hit.IndexedSHA256, hit.QualifiedSymbol, hit.StartByte, hit.EndByte)]
		if !ok {
			return anchorGroup{}, fmt.Errorf("anchor rank hit absent from parent inventory")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		parent, ok := parents[id]
		if !ok {
			return anchorGroup{}, fmt.Errorf("anchor parent absent from inventory")
		}
		parentTokens := stableUniqueStrings(strings.Fields((symbol.IdentifierNormalizer{}).Normalize(parent.Symbol + " " + parent.QualifiedSymbol)))
		matched := intersectTokens(tokens, parentTokens)
		denominator := len(parentTokens)
		if denominator == 0 {
			denominator = 1 // a missing normalized symbol has zero coverage and falls through.
		}
		values = append(values, anchorScore{anchorSelection: anchorSelection{ParentID: id, DenseRank: hit.Rank, MatchedTokens: matched, CoverageNumerator: len(matched), CoverageDenominator: denominator, QueryTokens: len(tokens), ParentTokens: len(parentTokens)}, coverageNumerator: len(matched), coverageDenominator: denominator})
	}
	if len(values) == 0 {
		return anchorGroup{}, fmt.Errorf("no available dense anchors")
	}
	if err := validateAnchorCoverageProducts(values); err != nil {
		return anchorGroup{}, err
	}
	sort.SliceStable(values, func(i, j int) bool {
		if comparison := compareFractionDesc(uint64(values[i].coverageNumerator), uint64(values[i].coverageDenominator), uint64(values[j].coverageNumerator), uint64(values[j].coverageDenominator)); comparison != 0 {
			return comparison < 0
		}
		if values[i].coverageNumerator != values[j].coverageNumerator {
			return values[i].coverageNumerator > values[j].coverageNumerator
		}
		if values[i].DenseRank != values[j].DenseRank {
			return values[i].DenseRank < values[j].DenseRank
		}
		return values[i].ParentID < values[j].ParentID
	})
	group := anchorGroup{QueryID: queryID, Tokens: append([]string(nil), tokens...)}
	for index := range values {
		if index < 2 {
			values[index].Ordinal = index + 1
			values[index].Selected = true
			values[index].SelectedOrdinal = index + 1
			group.Anchors = append(group.Anchors, values[index].anchorSelection)
		} else {
			values[index].Ordinal = 0
			values[index].Selected = false
			values[index].SelectedOrdinal = 0
		}
		group.Candidates = append(group.Candidates, values[index].anchorSelection)
	}
	return group, nil
}

func intersectTokens(left, right []string) []string {
	set := map[string]bool{}
	for _, value := range right {
		set[value] = true
	}
	var result []string
	for _, value := range left {
		if set[value] {
			result = append(result, value)
		}
	}
	return result
}

func validateAnchorCoverageProducts(values []anchorScore) error {
	var maxNumerator, maxDenominator uint64
	for _, value := range values {
		if value.coverageNumerator < 0 || value.coverageDenominator < 1 {
			return fmt.Errorf("invalid anchor coverage")
		}
		if uint64(value.coverageNumerator) > maxNumerator {
			maxNumerator = uint64(value.coverageNumerator)
		}
		if uint64(value.coverageDenominator) > maxDenominator {
			maxDenominator = uint64(value.coverageDenominator)
		}
	}
	if high, _ := bits.Mul64(maxNumerator, maxDenominator); high != 0 {
		return fmt.Errorf("anchor coverage cross product overflows")
	}
	return nil
}

func compareFractionDesc(leftNumerator, leftDenominator, rightNumerator, rightDenominator uint64) int {
	_, left := bits.Mul64(leftNumerator, rightDenominator)
	_, right := bits.Mul64(rightNumerator, leftDenominator)
	if left > right {
		return -1
	}
	if left < right {
		return 1
	}
	return 0
}

func structuralTier(kind RelationKind, metadata OccurrenceMetadata) (StructuralTier, error) {
	switch {
	case kind == TypeRef && (metadata.Zone == SignatureZone || metadata.Zone == TypeBodyZone):
		return DeclarationContractTier, nil
	case kind == Calls:
		return ExecutableDependencyTier, nil
	case kind == TypeRef && (metadata.Zone == BodyZone || metadata.Zone == InitializerZone):
		return BodyReferenceTier, nil
	case kind == MemberOf:
		return DeclarationStructureTier, nil
	}
	return "", fmt.Errorf("unmapped resolved relation kind/zone %s/%s", kind, metadata.Zone)
}

func structuralTierOrdinal(value StructuralTier) int {
	switch value {
	case DeclarationContractTier:
		return 0
	case ExecutableDependencyTier:
		return 1
	case BodyReferenceTier:
		return 2
	case DeclarationStructureTier:
		return 3
	}
	return 4
}

func storedEdgeKey(source, target string, kind RelationKind, tier StructuralTier) string {
	return source + "\x00" + target + "\x00" + string(kind) + "\x00" + string(tier)
}

func stratumKey(kind RelationKind, tier StructuralTier, parentID string) string {
	return string(kind) + "\x00" + string(tier) + "\x00" + parentID
}

func completeGraphEdgeStats(ctx context.Context, db *sql.DB) (map[string]edgeStats, map[StructuralTier]structuralTierCount, error) {
	rows, err := db.QueryContext(ctx, `SELECT relation_id,source_parent_id,target_parent_id,relation_kind,path,start_byte,end_byte,occurrence_zone,occurrence_role,flow_role,file_role,execution_mode,control_role,context_identifiers,source_ordinal FROM relation_occurrences WHERE outcome='RESOLVED_UNIQUE' ORDER BY relation_id`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	edges := map[string]edgeStats{}
	sourceOccurrences, targetOccurrences := map[string]int{}, map[string]int{}
	sourceTargets, targetSources := map[string]map[string]bool{}, map[string]map[string]bool{}
	tierSources := map[StructuralTier]map[string]bool{}
	edgeSliceCounts, sourceSliceCounts, targetSliceCounts := map[string]map[FileRole]int{}, map[string]map[FileRole]int{}, map[string]map[FileRole]int{}
	edgeSliceSources, sourceSliceSources, targetSliceSources := map[string]map[FileRole]map[string]bool{}, map[string]map[FileRole]map[string]bool{}, map[string]map[FileRole]map[string]bool{}
	tierSliceSources := map[StructuralTier]map[FileRole]map[string]bool{}
	tierCounts := map[StructuralTier]structuralTierCount{}
	for _, tier := range []StructuralTier{DeclarationContractTier, ExecutableDependencyTier, BodyReferenceTier, DeclarationStructureTier} {
		tierCounts[tier] = structuralTierCount{FileRoleSlices: completeFileRoleSlices(nil)}
	}
	for rows.Next() {
		fact, source, target, err := scanResolvedFact(rows, Forward, "")
		if err != nil {
			return nil, nil, err
		}
		tier, err := structuralTier(fact.Kind, fact.Metadata)
		if err != nil {
			return nil, nil, err
		}
		if source == target {
			continue // Self relations are outside the one-hop anchor experiment.
		}
		key := storedEdgeKey(source, target, fact.Kind, tier)
		stat := edges[key]
		if stat.EdgeOccurrences == 0 {
			stat = edgeStats{SourceID: source, TargetID: target, Kind: fact.Kind, Tier: tier, Best: fact}
		}
		stat.EdgeOccurrences++
		if lessBestOccurrence(fact, stat.Best) {
			stat.Best = fact
		}
		edges[key] = stat
		sourceKey, targetKey := stratumKey(fact.Kind, tier, source), stratumKey(fact.Kind, tier, target)
		sourceOccurrences[sourceKey]++
		targetOccurrences[targetKey]++
		if sourceTargets[sourceKey] == nil {
			sourceTargets[sourceKey] = map[string]bool{}
		}
		if targetSources[targetKey] == nil {
			targetSources[targetKey] = map[string]bool{}
		}
		sourceTargets[sourceKey][target] = true
		targetSources[targetKey][source] = true
		if tierSources[tier] == nil {
			tierSources[tier] = map[string]bool{}
		}
		tierSources[tier][source] = true
		incrementFileRoleSlice(edgeSliceCounts, edgeSliceSources, key, fact.Metadata.FileRole, source)
		incrementFileRoleSlice(sourceSliceCounts, sourceSliceSources, sourceKey, fact.Metadata.FileRole, source)
		incrementFileRoleSlice(targetSliceCounts, targetSliceSources, targetKey, fact.Metadata.FileRole, source)
		count := tierCounts[tier]
		if count.FileRoleSlices == nil {
			count.FileRoleSlices = map[FileRole]fileRoleSlice{}
		}
		count.Occurrences++
		roleCount := count.FileRoleSlices[fact.Metadata.FileRole]
		roleCount.Occurrences++
		count.FileRoleSlices[fact.Metadata.FileRole] = roleCount
		tierCounts[tier] = count
		if tierSliceSources[tier] == nil {
			tierSliceSources[tier] = map[FileRole]map[string]bool{}
		}
		if tierSliceSources[tier][fact.Metadata.FileRole] == nil {
			tierSliceSources[tier][fact.Metadata.FileRole] = map[string]bool{}
		}
		tierSliceSources[tier][fact.Metadata.FileRole][source] = true
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	for key, stat := range edges {
		sourceKey, targetKey := stratumKey(stat.Kind, stat.Tier, stat.SourceID), stratumKey(stat.Kind, stat.Tier, stat.TargetID)
		stat.SourceStratumOccurrences = sourceOccurrences[sourceKey]
		stat.SourceStratumDistinctTargets = len(sourceTargets[sourceKey])
		stat.TargetIncomingStratumOccurrences = targetOccurrences[targetKey]
		stat.TargetIncomingStratumDistinctSources = len(targetSources[targetKey])
		stat.EdgeFileRoleSlices = fileRoleSlices(edgeSliceCounts[key], edgeSliceSources[key])
		stat.SourceStratumFileRoleSlices = fileRoleSlices(sourceSliceCounts[sourceKey], sourceSliceSources[sourceKey])
		stat.TargetIncomingStratumFileRoleSlices = fileRoleSlices(targetSliceCounts[targetKey], targetSliceSources[targetKey])
		edges[key] = stat
	}
	for tier, count := range tierCounts {
		count.DistinctSources = len(tierSources[tier])
		for role, roleSources := range tierSliceSources[tier] {
			slice := count.FileRoleSlices[role]
			slice.DistinctSource = len(roleSources)
			count.FileRoleSlices[role] = slice
		}
		count.FileRoleSlices = completeFileRoleSlices(count.FileRoleSlices)
		tierCounts[tier] = count
	}
	if err := validateEdgeRatioProducts(edges); err != nil {
		return nil, nil, err
	}
	return edges, tierCounts, nil
}

func incrementFileRoleSlice(counts map[string]map[FileRole]int, sources map[string]map[FileRole]map[string]bool, key string, role FileRole, source string) {
	if counts[key] == nil {
		counts[key] = map[FileRole]int{}
	}
	counts[key][role]++
	if sources[key] == nil {
		sources[key] = map[FileRole]map[string]bool{}
	}
	if sources[key][role] == nil {
		sources[key][role] = map[string]bool{}
	}
	sources[key][role][source] = true
}

func fileRoleSlices(counts map[FileRole]int, sources map[FileRole]map[string]bool) map[FileRole]fileRoleSlice {
	values := map[FileRole]fileRoleSlice{}
	for role, occurrences := range counts {
		values[role] = fileRoleSlice{Occurrences: occurrences, DistinctSource: len(sources[role])}
	}
	return completeFileRoleSlices(values)
}

func completeFileRoleSlices(values map[FileRole]fileRoleSlice) map[FileRole]fileRoleSlice {
	if values == nil {
		values = map[FileRole]fileRoleSlice{}
	}
	for _, role := range []FileRole{ProductionFileRole, TestFileRole, ExampleFileRole, BenchmarkFileRole} {
		if _, ok := values[role]; !ok {
			values[role] = fileRoleSlice{}
		}
	}
	return values
}

func lessBestOccurrence(left, right Fact) bool {
	if left.Metadata.SourceOrdinal != right.Metadata.SourceOrdinal {
		return left.Metadata.SourceOrdinal < right.Metadata.SourceOrdinal
	}
	if left.OccurrenceByte != right.OccurrenceByte {
		return left.OccurrenceByte < right.OccurrenceByte
	}
	return left.RelationID < right.RelationID
}

func validateEdgeRatioProducts(edges map[string]edgeStats) error {
	var maxEdge, maxSource uint64
	for _, stat := range edges {
		if stat.EdgeOccurrences < 1 || stat.SourceStratumOccurrences < 1 || stat.SourceStratumDistinctTargets < 1 || stat.TargetIncomingStratumOccurrences < 1 || stat.TargetIncomingStratumDistinctSources < 1 {
			return fmt.Errorf("invalid complete graph edge statistic")
		}
		if uint64(stat.EdgeOccurrences) > maxEdge {
			maxEdge = uint64(stat.EdgeOccurrences)
		}
		if uint64(stat.SourceStratumOccurrences) > maxSource {
			maxSource = uint64(stat.SourceStratumOccurrences)
		}
	}
	if high, _ := bits.Mul64(maxEdge, maxSource); high != 0 {
		return fmt.Errorf("edge-strength ratio cross product overflows")
	}
	return nil
}

func scanResolvedFact(rows *sql.Rows, direction Direction, anchor string) (Fact, string, string, error) {
	var id, source, target, kind, path, zone, role, flow, fileRole, execution, control, contexts string
	var offset, end, ordinal int
	if err := rows.Scan(&id, &source, &target, &kind, &path, &offset, &end, &zone, &role, &flow, &fileRole, &execution, &control, &contexts, &ordinal); err != nil {
		return Fact{}, "", "", err
	}
	metadata := DefaultOccurrenceMetadata(path, ordinal)
	metadata.Zone, metadata.Role, metadata.Flow, metadata.FileRole, metadata.Execution, metadata.Control = OccurrenceZone(zone), OccurrenceRole(role), FlowRole(flow), FileRole(fileRole), ExecutionMode(execution), ControlRole(control)
	if err := json.Unmarshal([]byte(contexts), &metadata.ContextIdentifiers); err != nil || metadata.Validate() != nil {
		return Fact{}, "", "", fmt.Errorf("invalid relation metadata")
	}
	fact := Fact{RelationID: id, Direction: direction, AnchorID: source, EndpointID: target, Kind: RelationKind(kind), OccurrencePath: path, OccurrenceByte: offset, OccurrenceEndByte: end, Metadata: metadata}
	if direction == Reverse {
		fact.AnchorID, fact.EndpointID = target, source
	}
	if anchor != "" {
		fact.AnchorID = anchor
	}
	return fact, source, target, nil
}

func selectAnchorEdgeBundle(queryID string, feature queryFeatures, group anchorGroup, facts []Fact, stats map[string]edgeStats, ranks map[string]int, parents map[string]Parent, primary []string, policy string) (Bundle, *anchorEdgeTrace, error) {
	result := Bundle{QueryID: queryID, SelectionPolicy: policy}
	trace := &anchorEdgeTrace{Group: group, StagePresence: map[string]bool{"anchor": len(group.Anchors) > 0, "reachability": false, "strength": false, "cap": false, "packaging": false}}
	anchorOrdinals := map[string]int{}
	for _, anchor := range group.Anchors {
		anchorOrdinals[anchor.ParentID] = anchor.Ordinal
	}
	seen := map[string]bool{}
	for _, fact := range facts {
		source, target := fact.AnchorID, fact.EndpointID
		if fact.Direction == Reverse {
			source, target = fact.EndpointID, fact.AnchorID
		}
		tier, err := structuralTier(fact.Kind, fact.Metadata)
		if err != nil {
			return Bundle{}, nil, err
		}
		if source == target {
			continue
		}
		trace.DirectionalFacts = append(trace.DirectionalFacts, fact)
		key := storedEdgeKey(source, target, fact.Kind, tier)
		stat, ok := stats[key]
		if !ok {
			return Bundle{}, nil, fmt.Errorf("missing complete graph statistic for reachable edge")
		}
		candidateKey := fact.AnchorID + "\x00" + string(fact.Direction) + "\x00" + key
		if seen[candidateKey] {
			continue
		}
		seen[candidateKey] = true
		best := stat.Best
		best.Direction, best.AnchorID, best.EndpointID = fact.Direction, fact.AnchorID, fact.EndpointID
		candidate := anchorEdgeCandidate{Fact: best, AnchorOrdinal: anchorOrdinals[fact.AnchorID], DirectionMismatch: boolInt(feature.Direction != fact.Direction), Stats: stat}
		if candidate.AnchorOrdinal < 1 {
			return Bundle{}, nil, fmt.Errorf("reachable fact lacks selected anchor ordinal")
		}
		candidate.RankingTuple = anchorRankingTuple(candidate, ranks, policy)
		trace.Candidates = append(trace.Candidates, candidate)
	}
	trace.Denominators.Anchors, trace.Denominators.DirectionalOneHop, trace.Denominators.DistinctEdges = len(group.Anchors), len(trace.DirectionalFacts), len(trace.Candidates)
	trace.StagePresence["reachability"] = len(trace.DirectionalFacts) > 0
	if len(trace.Candidates) == 0 {
		return result, trace, nil
	}
	trace.StagePresence["strength"] = true
	sort.SliceStable(trace.Candidates, func(i, j int) bool {
		return compareAnchorEdgeCandidates(trace.Candidates[i], trace.Candidates[j], ranks, policy) < 0
	})
	for index := range trace.Candidates {
		trace.Candidates[index].Rank = index + 1
		if index > 0 {
			trace.Candidates[index].FirstDifference = firstAnchorEdgeDifference(trace.Candidates[0], trace.Candidates[index], ranks, policy)
		}
	}
	if err := validateAnchorRanking(trace.Candidates, ranks, policy); err != nil {
		return Bundle{}, nil, err
	}
	selected := trace.Candidates[0]
	result.Selected = &selected.Fact
	result.SelectionKey = rankingComponentsAny(selected.RankingTuple)
	result.AdmissionOrder = make([]AdmissionCandidate, 0, len(trace.Candidates))
	for _, candidate := range trace.Candidates {
		result.AdmissionOrder = append(result.AdmissionOrder, AdmissionCandidate{Fact: candidate.Fact, Prefix: []any{candidate.DirectionMismatch, structuralTierOrdinal(candidate.Stats.Tier)}, SelectionKey: rankingComponentsAny(candidate.RankingTuple), Admitted: true})
	}
	primarySet := map[string]bool{}
	for _, id := range primary {
		primarySet[id] = true
	}
	for _, id := range []string{selected.Stats.SourceID, selected.Stats.TargetID} {
		if _, ok := parents[id]; !ok || primarySet[id] || containsString(result.AddedParentIDs, id) {
			continue
		}
		result.AddedParentIDs = append(result.AddedParentIDs, id)
		if len(result.AddedParentIDs) == RelatedParentLimit {
			break
		}
	}
	trace.Denominators.Selected = 1
	return result, trace, nil
}

func selectFrontierBundle(queryID string, feature queryFeatures, group anchorGroup, facts []Fact, stats map[string]edgeStats, ranks map[string]int, parents map[string]Parent, primary []string, policy string) (Bundle, *frontierTrace, error) {
	trace, err := buildFrontierTrace(group, feature, facts, stats, ranks)
	if err != nil {
		return Bundle{}, nil, err
	}
	trace.Policy = policy
	result := Bundle{QueryID: queryID, SelectionPolicy: policy}
	if policy == AnchorFrontierGraphOnlyParetoPolicyID {
		if _, err := selectFrontierGraphOnlyPareto(trace, ranks); err != nil {
			return Bundle{}, nil, err
		}
	}
	if trace.AbstentionReason != "" {
		trace.FirstLossReason = trace.AbstentionReason
		return result, trace, nil
	}
	var selected *frontierEdge
	switch policy {
	case AnchorFrontierCapOnlyPolicyID:
		if len(trace.FinalFrontier) > 0 {
			value := trace.FinalFrontier[0]
			selected = &value
		} else {
			trace.AbstentionReason, trace.FirstLossReason = "NO_FRONTIER_EDGE", "NO_FRONTIER_EDGE"
		}
	case AnchorFrontierBridgePolicyID:
		for index := range trace.FinalFrontier {
			if trace.FinalFrontier[index].DirectBridge {
				value := trace.FinalFrontier[index]
				selected = &value
				break
			}
		}
		if selected == nil {
			trace.AbstentionReason, trace.FirstLossReason = "NO_DIRECT_ANCHOR_BRIDGE", "NO_DIRECT_ANCHOR_BRIDGE"
		}
	case AnchorFrontierGraphOnlyParetoPolicyID:
		selected = trace.GraphOnly.Selected
		if selected == nil {
			trace.AbstentionReason, trace.FirstLossReason = trace.GraphOnly.Outcome, trace.GraphOnly.Outcome
		}
	default:
		return Bundle{}, nil, fmt.Errorf("invalid frontier policy")
	}
	if selected == nil {
		return result, trace, nil
	}
	trace.Selected = selected
	selectedFact := selected.Candidate.Fact
	result.Selected = &selectedFact
	result.SelectionKey = rankingComponentsAny(selected.Candidate.RankingTuple)
	result.AdmissionOrder = make([]AdmissionCandidate, 0, len(trace.FinalFrontier))
	for _, edge := range trace.FinalFrontier {
		result.AdmissionOrder = append(result.AdmissionOrder, AdmissionCandidate{Fact: edge.Candidate.Fact, Prefix: []any{edge.Candidate.DirectionMismatch, structuralTierOrdinal(edge.Candidate.Stats.Tier)}, SelectionKey: rankingComponentsAny(edge.Candidate.RankingTuple), Admitted: true})
	}
	primarySet := map[string]bool{}
	for _, id := range primary {
		primarySet[id] = true
	}
	for _, id := range []string{selected.Candidate.Stats.SourceID, selected.Candidate.Stats.TargetID} {
		if _, ok := parents[id]; !ok || primarySet[id] || containsString(result.AddedParentIDs, id) {
			continue
		}
		result.AddedParentIDs = append(result.AddedParentIDs, id)
		if len(result.AddedParentIDs) == RelatedParentLimit {
			break
		}
	}
	return result, trace, nil
}

var frontierStructuralTiers = []StructuralTier{DeclarationContractTier, ExecutableDependencyTier, BodyReferenceTier, DeclarationStructureTier}

// selectFrontierGraphOnlyPareto is deliberately downstream of FinalFrontier:
// it neither expands the graph nor changes frontier construction or its digest.
func selectFrontierGraphOnlyPareto(trace *frontierTrace, ranks map[string]int) (*frontierEdge, error) {
	decision, selected, err := frontierGraphOnlyDecision(trace.FinalFrontier, ranks)
	if err != nil {
		return nil, err
	}
	trace.GraphOnly = decision
	if err := validateFrontierGraphOnlyDecision(trace.GraphOnly, trace.FinalFrontier, ranks); err != nil {
		return nil, err
	}
	return selected, nil
}

func frontierGraphOnlyDecision(final []frontierEdge, ranks map[string]int) (*frontierGraphOnlyTrace, *frontierEdge, error) {
	decision := &frontierGraphOnlyTrace{Tiers: make(map[StructuralTier]frontierParetoTier, len(frontierStructuralTiers))}
	for _, tier := range frontierStructuralTiers {
		decision.Tiers[tier] = frontierParetoTier{Candidates: []frontierParetoCandidate{}}
	}
	var firstBridge *frontierEdge
	byTier := make(map[StructuralTier][]frontierEdge, len(frontierStructuralTiers))
	for index := range final {
		edge := final[index]
		if edge.DirectBridge {
			decision.DirectBridgeCandidates++
			if firstBridge == nil {
				value := edge
				firstBridge = &value
			}
			continue
		}
		if edge.Candidate.Fact.Direction != Forward {
			decision.IncomingExcluded++
			continue
		}
		if edge.Candidate.Stats.Tier != edge.Bucket.Tier || !edge.Candidate.Stats.Tier.Valid() {
			return nil, nil, fmt.Errorf("graph-only frontier edge has invalid structural tier")
		}
		if edge.Candidate.Fact.EndpointID == "" {
			return nil, nil, fmt.Errorf("graph-only frontier edge lacks endpoint")
		}
		// Frozen dense ranks are 1..20. FinalFrontier carries the same canonical
		// endpoint identity used for anchor ranking, so no source text is consulted.
		if _, present := ranks[edge.Candidate.Fact.EndpointID]; present {
			decision.DenseEndpointExcluded++
			continue
		}
		byTier[edge.Candidate.Stats.Tier] = append(byTier[edge.Candidate.Stats.Tier], edge)
	}
	decision, selected, err := frontierGraphOnlyParetoByTier(decision, byTier)
	if err != nil || firstBridge == nil {
		return decision, selected, err
	}
	decision.Outcome, decision.Selected = "DIRECT_BRIDGE", firstBridge
	return decision, firstBridge, nil
}

func frontierGraphOnlyParetoByTier(decision *frontierGraphOnlyTrace, byTier map[StructuralTier][]frontierEdge) (*frontierGraphOnlyTrace, *frontierEdge, error) {
	survivors := make([]frontierEdge, 0)
	for _, tier := range frontierStructuralTiers {
		values := byTier[tier]
		candidates := make([]frontierParetoCandidate, len(values))
		for index := range values {
			if err := validateFrontierParetoStats(values[index]); err != nil {
				return nil, nil, err
			}
			candidates[index] = frontierParetoCandidate{
				CanonicalEdgeID:                      values[index].CanonicalEdgeID,
				EdgeOccurrences:                      values[index].Candidate.Stats.EdgeOccurrences,
				SourceStratumOccurrences:             values[index].Candidate.Stats.SourceStratumOccurrences,
				SourceStratumDistinctTargets:         values[index].Candidate.Stats.SourceStratumDistinctTargets,
				TargetIncomingStratumDistinctSources: values[index].Candidate.Stats.TargetIncomingStratumDistinctSources,
				DominatedBy:                          []string{},
			}
		}
		for index := range values {
			for other := range values {
				if index == other {
					continue
				}
				dominates, err := frontierParetoDominates(values[other], values[index])
				if err != nil {
					return nil, nil, err
				}
				if dominates {
					candidates[index].DominatedBy = append(candidates[index].DominatedBy, values[other].CanonicalEdgeID)
				}
			}
			sort.Strings(candidates[index].DominatedBy)
			candidates[index].Nondominated = len(candidates[index].DominatedBy) == 0
			if candidates[index].Nondominated {
				survivors = append(survivors, values[index])
			}
		}
		decision.GraphOnlyCandidates += len(values)
		decision.Tiers[tier] = frontierParetoTier{Candidates: candidates}
	}
	decision.UnionCount = len(survivors)
	switch len(survivors) {
	case 0:
		decision.Outcome = "NO_CANDIDATE"
	case 1:
		value := survivors[0]
		decision.Outcome, decision.Selected = "ONE_WINNER", &value
		return decision, &value, nil
	default:
		decision.Outcome = "MULTIPLE_WINNERS"
	}
	return decision, nil, nil
}

func validateFrontierParetoStats(edge frontierEdge) error {
	stats := edge.Candidate.Stats
	if edge.CanonicalEdgeID == "" || stats.EdgeOccurrences < 1 || stats.SourceStratumOccurrences < 1 || stats.SourceStratumDistinctTargets < 1 || stats.TargetIncomingStratumDistinctSources < 1 {
		return fmt.Errorf("invalid graph-only pareto statistics")
	}
	if stats.EdgeOccurrences > stats.SourceStratumOccurrences {
		return fmt.Errorf("graph-only edge occurrences exceed source stratum occurrences")
	}
	return nil
}

func frontierParetoDominates(left, right frontierEdge) (bool, error) {
	if err := validateFrontierParetoStats(left); err != nil {
		return false, err
	}
	if err := validateFrontierParetoStats(right); err != nil {
		return false, err
	}
	leftStats, rightStats := left.Candidate.Stats, right.Candidate.Stats
	leftHigh, leftLow := bits.Mul64(uint64(leftStats.EdgeOccurrences), uint64(rightStats.SourceStratumOccurrences))
	rightHigh, rightLow := bits.Mul64(uint64(rightStats.EdgeOccurrences), uint64(leftStats.SourceStratumOccurrences))
	if leftHigh != 0 || rightHigh != 0 {
		return false, fmt.Errorf("graph-only pareto ratio cross product overflows")
	}
	ratio := 0
	if leftLow > rightLow {
		ratio = -1
	} else if leftLow < rightLow {
		ratio = 1
	}
	noWorse := ratio <= 0 && leftStats.SourceStratumDistinctTargets <= rightStats.SourceStratumDistinctTargets && leftStats.TargetIncomingStratumDistinctSources <= rightStats.TargetIncomingStratumDistinctSources
	strict := ratio < 0 || leftStats.SourceStratumDistinctTargets < rightStats.SourceStratumDistinctTargets || leftStats.TargetIncomingStratumDistinctSources < rightStats.TargetIncomingStratumDistinctSources
	return noWorse && strict, nil
}

func validateFrontierGraphOnlyDecision(actual *frontierGraphOnlyTrace, final []frontierEdge, ranks map[string]int) error {
	if actual == nil {
		return fmt.Errorf("missing graph-only frontier decision")
	}
	if actual.DirectBridgeCandidates+actual.IncomingExcluded+actual.DenseEndpointExcluded+actual.GraphOnlyCandidates != len(final) {
		return fmt.Errorf("graph-only frontier partition does not cover final frontier")
	}
	var candidates, survivors int
	soleSurvivor := ""
	for _, tier := range frontierStructuralTiers {
		traceTier, ok := actual.Tiers[tier]
		if !ok {
			return fmt.Errorf("graph-only frontier trace lacks structural tier")
		}
		tierDominated, tierNondominated := 0, 0
		for _, candidate := range traceTier.Candidates {
			candidates++
			if candidate.Nondominated != (len(candidate.DominatedBy) == 0) {
				return fmt.Errorf("graph-only frontier pareto classification is inconsistent")
			}
			if candidate.Nondominated {
				tierNondominated++
				survivors++
				soleSurvivor = candidate.CanonicalEdgeID
			} else {
				tierDominated++
			}
		}
		if len(traceTier.Candidates) != tierDominated+tierNondominated {
			return fmt.Errorf("graph-only frontier tier candidate denominator is inconsistent")
		}
	}
	if actual.GraphOnlyCandidates != candidates {
		return fmt.Errorf("graph-only frontier candidate denominator is inconsistent")
	}
	if actual.UnionCount != survivors {
		return fmt.Errorf("graph-only frontier union denominator is inconsistent")
	}
	var firstBridge *frontierEdge
	for index := range final {
		if final[index].DirectBridge {
			value := final[index]
			firstBridge = &value
			break
		}
	}
	if firstBridge != nil {
		if actual.Outcome != "DIRECT_BRIDGE" || actual.Selected == nil || actual.Selected.CanonicalEdgeID != firstBridge.CanonicalEdgeID {
			return fmt.Errorf("graph-only frontier bridge outcome is inconsistent")
		}
	} else {
		switch actual.UnionCount {
		case 0:
			if actual.Outcome != "NO_CANDIDATE" || actual.Selected != nil {
				return fmt.Errorf("graph-only frontier empty outcome is inconsistent")
			}
		case 1:
			if actual.Outcome != "ONE_WINNER" || actual.Selected == nil || actual.Selected.CanonicalEdgeID != soleSurvivor {
				return fmt.Errorf("graph-only frontier singleton outcome is inconsistent")
			}
		default:
			if actual.Outcome != "MULTIPLE_WINNERS" || actual.Selected != nil {
				return fmt.Errorf("graph-only frontier multiple outcome is inconsistent")
			}
		}
	}
	expected, _, err := frontierGraphOnlyDecision(final, ranks)
	if err != nil {
		return err
	}
	expectedHash, err := canonicalHash(expected)
	if err != nil {
		return err
	}
	actualHash, err := canonicalHash(actual)
	if err != nil {
		return err
	}
	if expectedHash != actualHash {
		return fmt.Errorf("graph-only frontier denominator validation failed")
	}
	return nil
}

func buildFrontierTrace(group anchorGroup, feature queryFeatures, facts []Fact, stats map[string]edgeStats, ranks map[string]int) (*frontierTrace, error) {
	trace := &frontierTrace{Group: group, Buckets: map[string][]frontierEdge{}, BucketCounts: map[string]frontierBucketCounts{}}
	anchorOrdinals := map[string]int{}
	for _, anchor := range group.Anchors {
		anchorOrdinals[anchor.ParentID] = anchor.Ordinal
	}
	buckets := map[string]map[string]frontierEdge{}
	canonical := map[string]frontierEdge{}
	for _, fact := range facts {
		trace.Counts.RawDirectionalFacts++
		source, target := fact.AnchorID, fact.EndpointID
		if fact.Direction == Reverse {
			source, target = fact.EndpointID, fact.AnchorID
		}
		tier, err := structuralTier(fact.Kind, fact.Metadata)
		if err != nil {
			return nil, err
		}
		if source == target {
			trace.Counts.SelfFactsRemoved++
			continue
		}
		trace.Counts.NonSelfDirectionalFacts++
		stat, ok := stats[storedEdgeKey(source, target, fact.Kind, tier)]
		if !ok {
			return nil, fmt.Errorf("missing complete graph statistic for frontier edge")
		}
		ordinal := anchorOrdinals[fact.AnchorID]
		if ordinal < 1 {
			return nil, fmt.Errorf("frontier fact lacks selected anchor ordinal")
		}
		best := stat.Best
		best.Direction, best.AnchorID, best.EndpointID = fact.Direction, fact.AnchorID, fact.EndpointID
		candidate := anchorEdgeCandidate{Fact: best, AnchorOrdinal: ordinal, DirectionMismatch: boolInt(feature.Direction != fact.Direction), Stats: stat}
		candidate.RankingTuple = anchorRankingTuple(candidate, ranks, AnchorEdgeBidirectionalPolicyID)
		bucket := frontierBucket{AnchorOrdinal: ordinal, Direction: fact.Direction, Tier: tier}
		edge := frontierEdge{CanonicalEdgeID: storedEdgeKey(source, target, fact.Kind, tier), Bucket: bucket, Candidate: candidate, DirectBridge: directAnchorBridge(group.Anchors, source, target)}
		bucketKey := frontierBucketKey(bucket)
		if buckets[bucketKey] == nil {
			buckets[bucketKey] = map[string]frontierEdge{}
		}
		if existing, exists := buckets[bucketKey][edge.CanonicalEdgeID]; !exists || lessFrontierEdge(edge, existing, ranks) {
			buckets[bucketKey][edge.CanonicalEdgeID] = edge
		}
		if existing, exists := canonical[edge.CanonicalEdgeID]; !exists || lessFrontierEdge(edge, existing, ranks) {
			canonical[edge.CanonicalEdgeID] = edge
		}
	}
	eligibleBuckets := map[string][]frontierEdge{}
	provisionalBuckets := map[string][]frontierEdge{}
	for _, key := range sortedKeys(buckets) {
		values := frontierEdgeValues(buckets[key], ranks)
		trace.Buckets[key] = append([]frontierEdge(nil), values...)
		eligibleBuckets[key] = append([]frontierEdge(nil), values...)
		counts := frontierBucketCounts{PreCapDistinct: len(values)}
		trace.Counts.BucketDistinctCanonicalViews += counts.PreCapDistinct
		if len(values) > FrontierBucketLimit {
			values = values[:FrontierBucketLimit]
		}
		provisionalBuckets[key] = append([]frontierEdge(nil), values...)
		trace.BucketCounts[key] = counts
	}
	trace.Counts.RepeatedOccurrenceCollapse = trace.Counts.NonSelfDirectionalFacts - trace.Counts.BucketDistinctCanonicalViews
	trace.Counts.GlobalCanonicalUniverseEdges = len(canonical)
	trace.Counts.UniverseCrossBucketDuplicateViews = trace.Counts.BucketDistinctCanonicalViews - trace.Counts.GlobalCanonicalUniverseEdges
	trace.Counts.CanonicalEdges = trace.Counts.GlobalCanonicalUniverseEdges
	trace.Counts.OccurrenceCollapse = trace.Counts.RepeatedOccurrenceCollapse
	reservations, displacements, overflow := assignFrontierBridges(provisionalBuckets, eligibleBuckets, ranks)
	trace.BridgeReservations, trace.BridgeDisplacements = reservations, displacements
	trace.Counts.ReservedBridgeEdges = len(reservations)
	provisional, err := frontierProvisionalEdges(provisionalBuckets, ranks)
	if err != nil {
		return nil, err
	}
	trace.Provisional = append([]frontierEdge(nil), provisional...)
	trace.Counts.ProvisionalRows, trace.Counts.ProvisionalBucketEdges = len(provisional), len(provisional)
	for key, counts := range trace.BucketCounts {
		counts.Retained = len(provisionalBuckets[key])
		counts.Truncated = counts.PreCapDistinct - counts.Retained
		trace.Counts.BucketTruncations += counts.Truncated
		trace.BucketCounts[key] = counts
	}
	trace.Counts.TruncatedEdges = trace.Counts.BucketTruncations
	if overflow {
		trace.CapReached = true
		trace.AbstentionReason, trace.FirstLossReason = "BRIDGE_CAP_OVERFLOW", "BRIDGE_CAP_OVERFLOW"
		return finalizeFrontierDigest(trace)
	}
	union := map[string]frontierEdge{}
	for _, edge := range provisional {
		if existing, exists := union[edge.CanonicalEdgeID]; !exists || lessFrontierEdge(edge, existing, ranks) {
			union[edge.CanonicalEdgeID] = edge
		}
	}
	trace.Counts.CanonicalUnionEdges = len(union)
	trace.Counts.PostCapCrossBucketDuplicates = len(provisional) - len(union)
	trace.Counts.CrossBucketDuplicates = trace.Counts.PostCapCrossBucketDuplicates
	trace.CanonicalUnion = frontierEdgeValues(union, ranks)
	trace.FinalFrontier = append([]frontierEdge(nil), trace.CanonicalUnion...)
	if len(trace.FinalFrontier) > FrontierGlobalLimit {
		return nil, fmt.Errorf("frontier global cap exceeded after canonical union")
	}
	trace.Counts.FinalRetained, trace.Counts.RetainedFrontierEdges = len(trace.FinalFrontier), len(trace.FinalFrontier)
	trace.CapReached = len(trace.FinalFrontier) == FrontierGlobalLimit
	return finalizeFrontierDigest(trace)
}

func frontierProvisionalEdges(buckets map[string][]frontierEdge, ranks map[string]int) ([]frontierEdge, error) {
	var provisional []frontierEdge
	for _, key := range sortedKeys(buckets) {
		values := buckets[key]
		sort.SliceStable(values, func(i, j int) bool { return lessFrontierEdge(values[i], values[j], ranks) })
		if len(values) > FrontierBucketLimit {
			return nil, fmt.Errorf("frontier bucket exceeds cap after bridge reservation")
		}
		provisional = append(provisional, values...)
	}
	return provisional, nil
}

func assignFrontierBridges(provisional, eligible map[string][]frontierEdge, ranks map[string]int) ([]frontierBridgeReservation, []frontierBridgeDisplacement, bool) {
	bridgeChoices := map[string][]frontierEdge{}
	for _, values := range eligible {
		for _, edge := range values {
			if edge.DirectBridge {
				bridgeChoices[edge.CanonicalEdgeID] = append(bridgeChoices[edge.CanonicalEdgeID], edge)
			}
		}
	}
	bridges := make([]frontierEdge, 0, len(bridgeChoices))
	for _, values := range bridgeChoices {
		sort.SliceStable(values, func(i, j int) bool { return lessFrontierEdge(values[i], values[j], ranks) })
		bridges = append(bridges, values[0])
	}
	sort.SliceStable(bridges, func(i, j int) bool { return lessFrontierEdge(bridges[i], bridges[j], ranks) })
	reservations := make([]frontierBridgeReservation, 0, len(bridges))
	displacements := []frontierBridgeDisplacement{}
	for _, bridge := range bridges {
		choices := bridgeChoices[bridge.CanonicalEdgeID]
		sort.SliceStable(choices, func(i, j int) bool { return lessFrontierEdge(choices[i], choices[j], ranks) })
		var survived *frontierEdge
		for _, choice := range choices {
			for _, current := range provisional[frontierBucketKey(choice.Bucket)] {
				if current.CanonicalEdgeID == bridge.CanonicalEdgeID {
					value := choice
					survived = &value
					break
				}
			}
			if survived != nil {
				break
			}
		}
		if survived != nil {
			reservations = append(reservations, frontierBridgeReservation{CanonicalEdgeID: bridge.CanonicalEdgeID, Outcome: "SURVIVED", EligibleBucket: survived.Bucket})
			continue
		}
		assigned := false
		for _, choice := range choices {
			bucketKey := frontierBucketKey(choice.Bucket)
			values := provisional[bucketKey]
			if len(values) < FrontierBucketLimit {
				provisional[bucketKey] = append(values, choice)
				reservations = append(reservations, frontierBridgeReservation{CanonicalEdgeID: bridge.CanonicalEdgeID, Outcome: "RESERVED", EligibleBucket: choice.Bucket})
				assigned = true
				break
			}
			worst := -1
			for index := range values {
				if values[index].DirectBridge {
					continue
				}
				if worst == -1 || lessFrontierEdge(values[worst], values[index], ranks) {
					worst = index
				}
			}
			if worst == -1 {
				continue
			}
			displaced := values[worst]
			values[worst] = choice
			provisional[bucketKey] = values
			reservations = append(reservations, frontierBridgeReservation{CanonicalEdgeID: bridge.CanonicalEdgeID, Outcome: "RESERVED", EligibleBucket: choice.Bucket, DisplacedCanonicalEdgeID: displaced.CanonicalEdgeID})
			displacements = append(displacements, frontierBridgeDisplacement{BridgeCanonicalEdgeID: bridge.CanonicalEdgeID, DisplacedCanonicalEdgeID: displaced.CanonicalEdgeID, Bucket: choice.Bucket})
			assigned = true
			break
		}
		if !assigned {
			return reservations, displacements, true
		}
	}
	return reservations, displacements, false
}

func frontierBucketKey(bucket frontierBucket) string {
	return fmt.Sprintf("%d\x00%s\x00%s", bucket.AnchorOrdinal, bucket.Direction, bucket.Tier)
}

func directAnchorBridge(anchors []anchorSelection, source, target string) bool {
	return len(anchors) == 2 && ((source == anchors[0].ParentID && target == anchors[1].ParentID) || (source == anchors[1].ParentID && target == anchors[0].ParentID))
}

func lessFrontierEdge(left, right frontierEdge, ranks map[string]int) bool {
	if comparison := compareAnchorEdgeCandidates(left.Candidate, right.Candidate, ranks, AnchorEdgeBidirectionalPolicyID); comparison != 0 {
		return comparison < 0
	}
	return left.CanonicalEdgeID < right.CanonicalEdgeID
}

func frontierEdgeValues(values map[string]frontierEdge, ranks map[string]int) []frontierEdge {
	result := make([]frontierEdge, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool { return lessFrontierEdge(result[i], result[j], ranks) })
	return result
}

func finalizeFrontierDigest(trace *frontierTrace) (*frontierTrace, error) {
	type digestEntry struct {
		CanonicalEdgeID, AnchorID, EndpointID, RelationID string
		Direction                                         Direction
	}
	entries := make([]digestEntry, 0, len(trace.FinalFrontier))
	for _, edge := range trace.FinalFrontier {
		entries = append(entries, digestEntry{CanonicalEdgeID: edge.CanonicalEdgeID, AnchorID: edge.Candidate.Fact.AnchorID, EndpointID: edge.Candidate.Fact.EndpointID, RelationID: edge.Candidate.Fact.RelationID, Direction: edge.Candidate.Fact.Direction})
	}
	digest, err := canonicalHash(entries)
	if err != nil {
		return nil, err
	}
	trace.FinalDigest = digest
	return trace, nil
}

func anchorEndpointRank(candidate anchorEdgeCandidate, ranks map[string]int) int {
	if rank, ok := ranks[candidate.Fact.EndpointID]; ok {
		return rank
	}
	return MaxDenseDepth + 1
}

func anchorRankingTuple(candidate anchorEdgeCandidate, ranks map[string]int, policy string) []rankingComponent {
	stats := candidate.Stats
	base := []rankingComponent{rankingValue("direction_mismatch", "asc", candidate.DirectionMismatch), rankingValue("structural_tier", "asc", structuralTierOrdinal(stats.Tier))}
	switch policy {
	case AnchorEdgeRawFrequencyPolicyID:
		base = append(base, rankingValue("edge_occurrences", "desc", stats.EdgeOccurrences))
	case AnchorEdgeSourceNormalizedPolicyID:
		base = append(base, rankingRatio("edge_occurrences_over_source_stratum_occurrences", stats.EdgeOccurrences, stats.SourceStratumOccurrences), rankingValue("source_stratum_distinct_targets", "asc", stats.SourceStratumDistinctTargets))
	case AnchorEdgeBidirectionalPolicyID:
		base = append(base, rankingRatio("edge_occurrences_over_source_stratum_occurrences", stats.EdgeOccurrences, stats.SourceStratumOccurrences), rankingValue("source_stratum_distinct_targets", "asc", stats.SourceStratumDistinctTargets), rankingValue("target_incoming_stratum_distinct_sources", "asc", stats.TargetIncomingStratumDistinctSources), rankingValue("target_incoming_stratum_occurrences", "asc", stats.TargetIncomingStratumOccurrences))
	case AnchorEdgeIncomingPopularityPolicyID:
		base = append(base, rankingValue("target_incoming_stratum_distinct_sources", "desc", stats.TargetIncomingStratumDistinctSources), rankingValue("target_incoming_stratum_occurrences", "desc", stats.TargetIncomingStratumOccurrences), rankingValue("edge_occurrences", "desc", stats.EdgeOccurrences))
	}
	return append(base,
		rankingValue("anchor_selection_ordinal", "asc", candidate.AnchorOrdinal),
		rankingValue("endpoint_dense_rank", "asc", anchorEndpointRank(candidate, ranks)),
		rankingValue("best_occurrence_ordinal", "asc", stats.Best.Metadata.SourceOrdinal),
		rankingValue("best_occurrence_byte", "asc", stats.Best.OccurrenceByte),
		rankingValue("source_id", "asc", stats.SourceID),
		rankingValue("target_id", "asc", stats.TargetID),
		rankingValue("relation_kind", "asc", string(stats.Kind)),
		rankingValue("stable_relation_id", "asc", stats.Best.RelationID),
	)
}

func rankingValue(name, order string, value any) rankingComponent {
	return rankingComponent{Name: name, Order: order, Value: value}
}

func rankingRatio(name string, numerator, denominator int) rankingComponent {
	return rankingComponent{Name: name, Order: "desc_rational", Numerator: &numerator, Denominator: &denominator}
}

func anchorRankingComponentDefinitions(policy string) []rankingComponent {
	values := []rankingComponent{{Name: "direction_mismatch", Order: "asc"}, {Name: "structural_tier", Order: "asc"}}
	switch policy {
	case AnchorEdgeRawFrequencyPolicyID:
		values = append(values, rankingComponent{Name: "edge_occurrences", Order: "desc"})
	case AnchorEdgeSourceNormalizedPolicyID:
		values = append(values, rankingComponent{Name: "edge_occurrences_over_source_stratum_occurrences", Order: "desc_rational"}, rankingComponent{Name: "source_stratum_distinct_targets", Order: "asc"})
	case AnchorEdgeBidirectionalPolicyID:
		values = append(values, rankingComponent{Name: "edge_occurrences_over_source_stratum_occurrences", Order: "desc_rational"}, rankingComponent{Name: "source_stratum_distinct_targets", Order: "asc"}, rankingComponent{Name: "target_incoming_stratum_distinct_sources", Order: "asc"}, rankingComponent{Name: "target_incoming_stratum_occurrences", Order: "asc"})
	case AnchorEdgeIncomingPopularityPolicyID:
		values = append(values, rankingComponent{Name: "target_incoming_stratum_distinct_sources", Order: "desc"}, rankingComponent{Name: "target_incoming_stratum_occurrences", Order: "desc"}, rankingComponent{Name: "edge_occurrences", Order: "desc"})
	}
	return append(values,
		rankingComponent{Name: "anchor_selection_ordinal", Order: "asc"}, rankingComponent{Name: "endpoint_dense_rank", Order: "asc"}, rankingComponent{Name: "best_occurrence_ordinal", Order: "asc"}, rankingComponent{Name: "best_occurrence_byte", Order: "asc"}, rankingComponent{Name: "source_id", Order: "asc"}, rankingComponent{Name: "target_id", Order: "asc"}, rankingComponent{Name: "relation_kind", Order: "asc"}, rankingComponent{Name: "stable_relation_id", Order: "asc"},
	)
}

func anchorRankingComponentSequence(policy string) []string {
	definitions := anchorRankingComponentDefinitions(policy)
	values := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		values = append(values, definition.Name+":"+definition.Order)
	}
	return values
}

func rankingComponentsAny(values []rankingComponent) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func validateAnchorRanking(values []anchorEdgeCandidate, ranks map[string]int, policy string) error {
	definitions := anchorRankingComponentDefinitions(policy)
	for index := range values {
		if !sameRankingComponentSequence(values[index].RankingTuple, definitions) {
			return fmt.Errorf("anchor ranking tuple does not match frozen policy component sequence")
		}
		if !sameRankingComponents(values[index].RankingTuple, anchorRankingTuple(values[index], ranks, policy)) {
			return fmt.Errorf("anchor ranking tuple is not mechanically reproducible")
		}
		if index > 0 {
			if compareAnchorEdgeCandidates(values[index-1], values[index], ranks, policy) > 0 {
				return fmt.Errorf("anchor ranking order is not monotonic")
			}
			expected := firstAnchorEdgeDifference(values[0], values[index], ranks, policy)
			if values[index].FirstDifference != expected || !rankingComponentNamed(values[index].RankingTuple, expected) {
				return fmt.Errorf("anchor ranking first difference is invalid")
			}
		}
	}
	return nil
}

func sameRankingComponentSequence(values, definitions []rankingComponent) bool {
	if len(values) != len(definitions) {
		return false
	}
	for index := range values {
		if values[index].Name != definitions[index].Name || values[index].Order != definitions[index].Order {
			return false
		}
	}
	return true
}

func sameRankingComponents(left, right []rankingComponent) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Name != right[index].Name || left[index].Order != right[index].Order {
			return false
		}
		if left[index].Numerator != nil || right[index].Numerator != nil {
			if left[index].Numerator == nil || left[index].Denominator == nil || right[index].Numerator == nil || right[index].Denominator == nil || *left[index].Numerator != *right[index].Numerator || *left[index].Denominator != *right[index].Denominator {
				return false
			}
			continue
		}
		if fmt.Sprint(left[index].Value) != fmt.Sprint(right[index].Value) {
			return false
		}
	}
	return true
}

func rankingComponentNamed(values []rankingComponent, name string) bool {
	for _, value := range values {
		if value.Name == name {
			return true
		}
	}
	return false
}

func compareAnchorEdgeCandidates(left, right anchorEdgeCandidate, ranks map[string]int, policy string) int {
	if result := compareInt(left.DirectionMismatch, right.DirectionMismatch); result != 0 {
		return result
	}
	if result := compareInt(structuralTierOrdinal(left.Stats.Tier), structuralTierOrdinal(right.Stats.Tier)); result != 0 {
		return result
	}
	switch policy {
	case AnchorEdgeRawFrequencyPolicyID:
		if result := compareIntDesc(left.Stats.EdgeOccurrences, right.Stats.EdgeOccurrences); result != 0 {
			return result
		}
	case AnchorEdgeSourceNormalizedPolicyID:
		if result := compareFractionDesc(uint64(left.Stats.EdgeOccurrences), uint64(left.Stats.SourceStratumOccurrences), uint64(right.Stats.EdgeOccurrences), uint64(right.Stats.SourceStratumOccurrences)); result != 0 {
			return result
		}
		if result := compareInt(left.Stats.SourceStratumDistinctTargets, right.Stats.SourceStratumDistinctTargets); result != 0 {
			return result
		}
	case AnchorEdgeBidirectionalPolicyID:
		if result := compareFractionDesc(uint64(left.Stats.EdgeOccurrences), uint64(left.Stats.SourceStratumOccurrences), uint64(right.Stats.EdgeOccurrences), uint64(right.Stats.SourceStratumOccurrences)); result != 0 {
			return result
		}
		if result := compareInt(left.Stats.SourceStratumDistinctTargets, right.Stats.SourceStratumDistinctTargets); result != 0 {
			return result
		}
		if result := compareInt(left.Stats.TargetIncomingStratumDistinctSources, right.Stats.TargetIncomingStratumDistinctSources); result != 0 {
			return result
		}
		if result := compareInt(left.Stats.TargetIncomingStratumOccurrences, right.Stats.TargetIncomingStratumOccurrences); result != 0 {
			return result
		}
	case AnchorEdgeIncomingPopularityPolicyID:
		if result := compareIntDesc(left.Stats.TargetIncomingStratumDistinctSources, right.Stats.TargetIncomingStratumDistinctSources); result != 0 {
			return result
		}
		if result := compareIntDesc(left.Stats.TargetIncomingStratumOccurrences, right.Stats.TargetIncomingStratumOccurrences); result != 0 {
			return result
		}
		if result := compareIntDesc(left.Stats.EdgeOccurrences, right.Stats.EdgeOccurrences); result != 0 {
			return result
		}
	}
	if result := compareInt(left.AnchorOrdinal, right.AnchorOrdinal); result != 0 {
		return result
	}
	if result := compareInt(anchorEndpointRank(left, ranks), anchorEndpointRank(right, ranks)); result != 0 {
		return result
	}
	if result := compareInt(left.Stats.Best.Metadata.SourceOrdinal, right.Stats.Best.Metadata.SourceOrdinal); result != 0 {
		return result
	}
	if result := compareInt(left.Stats.Best.OccurrenceByte, right.Stats.Best.OccurrenceByte); result != 0 {
		return result
	}
	if left.Stats.SourceID != right.Stats.SourceID {
		if left.Stats.SourceID < right.Stats.SourceID {
			return -1
		}
		return 1
	}
	if left.Stats.TargetID != right.Stats.TargetID {
		if left.Stats.TargetID < right.Stats.TargetID {
			return -1
		}
		return 1
	}
	if left.Stats.Kind != right.Stats.Kind {
		if left.Stats.Kind < right.Stats.Kind {
			return -1
		}
		return 1
	}
	if left.Stats.Best.RelationID != right.Stats.Best.RelationID {
		if left.Stats.Best.RelationID < right.Stats.Best.RelationID {
			return -1
		}
		return 1
	}
	return 0
}

func firstAnchorEdgeDifference(left, right anchorEdgeCandidate, ranks map[string]int, policy string) string {
	return firstUnequalRankingComponent(anchorRankingTuple(left, ranks, policy), anchorRankingTuple(right, ranks, policy))
}

func firstUnequalRankingComponent(left, right []rankingComponent) string {
	if len(left) != len(right) {
		return "tuple_length"
	}
	for index := range left {
		if left[index].Name != right[index].Name || left[index].Order != right[index].Order || !sameRankingComponentOrderValue(left[index], right[index]) {
			return left[index].Name
		}
	}
	return "identical"
}

func sameRankingComponentOrderValue(left, right rankingComponent) bool {
	if left.Numerator != nil || right.Numerator != nil {
		return left.Numerator != nil && left.Denominator != nil && right.Numerator != nil && right.Denominator != nil && compareFractionDesc(uint64(*left.Numerator), uint64(*left.Denominator), uint64(*right.Numerator), uint64(*right.Denominator)) == 0
	}
	switch value := left.Value.(type) {
	case int:
		rightValue, ok := right.Value.(int)
		return ok && value == rightValue
	case string:
		rightValue, ok := right.Value.(string)
		return ok && value == rightValue
	default:
		return false
	}
}

func compareInt(left, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
func compareIntDesc(left, right int) int { return compareInt(right, left) }

func reachableFacts(ctx context.Context, db *sql.DB, seeds []string) ([]Fact, error) {
	seen := map[string]bool{}
	var values []Fact
	for _, seed := range seeds {
		rows, err := db.QueryContext(ctx, `SELECT relation_id,source_parent_id,target_parent_id,relation_kind,path,start_byte,end_byte,occurrence_zone,occurrence_role,flow_role,file_role,execution_mode,control_role,context_identifiers,source_ordinal FROM relation_occurrences WHERE outcome='RESOLVED_UNIQUE' AND (source_parent_id=? OR target_parent_id=?) ORDER BY relation_id`, seed, seed)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id, source, target, kind, path, zone, role, flow, fileRole, execution, control, contexts string
			var offset, end int
			var ordinal int
			if err := rows.Scan(&id, &source, &target, &kind, &path, &offset, &end, &zone, &role, &flow, &fileRole, &execution, &control, &contexts, &ordinal); err != nil {
				rows.Close()
				return nil, err
			}
			direction, endpoint := Forward, target
			if target == seed {
				direction, endpoint = Reverse, source
			}
			key := id + "\x00" + string(direction) + "\x00" + seed
			metadata := DefaultOccurrenceMetadata(path, ordinal)
			metadata.Zone, metadata.Role, metadata.Flow, metadata.FileRole, metadata.Execution, metadata.Control = OccurrenceZone(zone), OccurrenceRole(role), FlowRole(flow), FileRole(fileRole), ExecutionMode(execution), ControlRole(control)
			if err := json.Unmarshal([]byte(contexts), &metadata.ContextIdentifiers); err != nil || metadata.Validate() != nil {
				rows.Close()
				return nil, fmt.Errorf("invalid relation metadata")
			}
			if !seen[key] {
				seen[key] = true
				values = append(values, Fact{RelationID: id, Direction: direction, AnchorID: seed, EndpointID: endpoint, Kind: RelationKind(kind), OccurrencePath: path, OccurrenceByte: offset, OccurrenceEndByte: end, Metadata: metadata})
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	sort.Slice(values, func(i, j int) bool { return factKey(values[i]) < factKey(values[j]) })
	return values, nil
}
func factKey(v Fact) string {
	return fmt.Sprintf("%s\x00%s\x00%s", v.RelationID, v.Direction, v.AnchorID)
}

// loadQueryFeatures intentionally decodes only query identity and text. The
// frozen label structure remains unopened until Stage A, ordering, admission,
// and packaging have all completed.
func loadQueryFeatures(file string) (map[string]queryFeatures, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Cases []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if len(raw.Cases) == 0 {
		return nil, fmt.Errorf("missing query features")
	}
	normalizer := symbol.IdentifierNormalizer{}
	result := map[string]queryFeatures{}
	for _, item := range raw.Cases {
		if item.ID == "" || item.Text == "" || result[item.ID].QueryID != "" {
			return nil, fmt.Errorf("invalid query feature input")
		}
		tokens := strings.Fields(normalizer.Normalize(item.Text))
		classified := symbol.ClassifyQuery(item.Text, normalizer)
		feature := queryFeatures{QueryID: item.ID, Tokens: uniqueStrings(tokens), AnchorTokens: stableUniqueStrings(append(classified.IdentifierTokens, classified.TextTokens...)), Direction: Forward}
		if len(feature.AnchorTokens) == 0 {
			return nil, fmt.Errorf("empty classified query tokens")
		}
		for _, token := range feature.Tokens {
			switch token {
			case "contract", "props", "type", "interface", "signature", "options", "schema":
				feature.SignatureIntent = true
				if token == "contract" || token == "props" {
					feature.ValueParameterContractIntent = true
				}
			case "mutate", "set", "write", "assign":
				feature.MutationIntent = true
			case "return":
				feature.ReturnIntent = true
			case "condition", "when":
				feature.ConditionIntent = true
			case "caller":
				feature.Direction = Reverse
			case "deprecated":
				feature.DeprecatedIntent = true
			case "test", "tests":
				feature.ExplicitFileRoles = append(feature.ExplicitFileRoles, TestFileRole)
			case "example", "examples":
				feature.ExplicitFileRoles = append(feature.ExplicitFileRoles, ExampleFileRole)
			case "benchmark", "benchmarks", "bench":
				feature.ExplicitFileRoles = append(feature.ExplicitFileRoles, BenchmarkFileRole)
			}
		}
		if containsString(feature.Tokens, "used") && containsString(feature.Tokens, "by") {
			feature.Direction = Reverse
		}
		feature.ExplicitFileRoles = uniqueFileRoles(feature.ExplicitFileRoles)
		result[item.ID] = feature
	}
	return result, nil
}

func graphFirstSeeds(replay frozenReplay, queryID string, byHit map[string]string) ([]string, error) {
	seen := map[string]bool{}
	var result []string
	for _, laneName := range []string{"fts", "simple_control"} {
		lane, ok := replay.Lanes[laneName]
		if !ok {
			return nil, fmt.Errorf("missing frozen %s lane for graph-first crossover", laneName)
		}
		hits := lane.Ranks[queryID]
		if len(hits) < ProtectedPrimaryK {
			return nil, fmt.Errorf("invalid frozen %s graph seeds", laneName)
		}
		for _, hit := range hits[:ProtectedPrimaryK] {
			id, ok := byHit[hitKey(hit.Path, hit.IndexedSHA256, hit.QualifiedSymbol, hit.StartByte, hit.EndByte)]
			if !ok {
				return nil, fmt.Errorf("graph seed absent from semantic parent inventory")
			}
			if !seen[id] {
				seen[id] = true
				result = append(result, id)
			}
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("empty graph-first seed union")
	}
	return result, nil
}

func selectBundleWithPolicy(queryID string, feature queryFeatures, facts []Fact, ranks map[string]int, parents map[string]Parent, primary []string, policy string) Bundle {
	result := Bundle{QueryID: queryID, SelectionPolicy: policy}
	primarySet := map[string]bool{}
	for _, id := range primary {
		primarySet[id] = true
	}
	candidates := make([]AdmissionCandidate, 0, len(facts))
	for _, fact := range facts {
		if eligibleFact(fact, parents, primarySet) {
			prefix := metadataAdmissionPrefix(fact, feature, parents, policy)
			candidate := AdmissionCandidate{Fact: fact, Prefix: prefix, SelectionKey: metadataSelectionKey(fact, feature, ranks, parents, policy)}
			if policy == GraphFirstPolicyID {
				candidate.RerankKey = graphRerankKey(fact, ranks)
			}
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return result
	}
	if policy == GraphFirstPolicyID {
		sort.SliceStable(candidates, func(i, j int) bool { return lessSelectionKey(candidates[i].Prefix, candidates[j].Prefix) })
		best := candidates[0].Prefix
		for index := range candidates {
			candidates[index].Admitted = equalSelectionKey(candidates[index].Prefix, best)
		}
		admitted := make([]AdmissionCandidate, 0, len(candidates))
		for _, candidate := range candidates {
			if candidate.Admitted {
				admitted = append(admitted, candidate)
			}
		}
		sort.SliceStable(admitted, func(i, j int) bool { return lessSelectionKey(admitted[i].RerankKey, admitted[j].RerankKey) })
		selected := admitted[0].Fact
		result.Selected, result.SelectionKey = &selected, admitted[0].SelectionKey
		sort.SliceStable(candidates, func(i, j int) bool {
			if candidates[i].Admitted != candidates[j].Admitted {
				return candidates[i].Admitted
			}
			if candidates[i].Admitted {
				return lessSelectionKey(candidates[i].RerankKey, candidates[j].RerankKey)
			}
			if lessSelectionKey(candidates[i].Prefix, candidates[j].Prefix) {
				return true
			}
			if lessSelectionKey(candidates[j].Prefix, candidates[i].Prefix) {
				return false
			}
			return lessSelectionKey(candidates[i].SelectionKey, candidates[j].SelectionKey)
		})
	} else {
		for index := range candidates {
			candidates[index].Admitted = true
		}
		sort.SliceStable(candidates, func(i, j int) bool { return lessSelectionKey(candidates[i].SelectionKey, candidates[j].SelectionKey) })
		selected := candidates[0].Fact
		result.Selected, result.SelectionKey = &selected, candidates[0].SelectionKey
	}
	result.AdmissionOrder = candidates
	selected := *result.Selected
	for _, id := range []string{selected.AnchorID, selected.EndpointID} {
		if _, ok := parents[id]; !ok || primarySet[id] || containsString(result.AddedParentIDs, id) {
			continue
		}
		result.AddedParentIDs = append(result.AddedParentIDs, id)
		if len(result.AddedParentIDs) == RelatedParentLimit {
			break
		}
	}
	return result
}

func eligibleFact(fact Fact, parents map[string]Parent, primary map[string]bool) bool {
	if fact.Metadata.Validate() != nil {
		return false
	}
	if fact.AnchorID == "" || fact.EndpointID == "" || fact.AnchorID == fact.EndpointID {
		return false
	}
	_, anchor := parents[fact.AnchorID]
	_, endpoint := parents[fact.EndpointID]
	return anchor && endpoint
}

func lessGraphRerank(a, b Fact, ranks map[string]int) bool {
	return lessSelectionKey(graphRerankKey(a, ranks), graphRerankKey(b, ranks))
}
func graphRerankKey(a Fact, ranks map[string]int) []any {
	rank := func(id string) int {
		if value, ok := ranks[id]; ok {
			return value
		}
		return MaxDenseDepth + 1
	}
	amin, amax := rank(a.AnchorID), rank(a.EndpointID)
	if amax < amin {
		amin, amax = amax, amin
	}
	return []any{amin, amax, a.Metadata.SourceOrdinal, a.OccurrenceByte, factKey(a)}
}

func metadataSelectionKey(fact Fact, feature queryFeatures, ranks map[string]int, parents map[string]Parent, policy string) []any {
	prefix := metadataAdmissionPrefix(fact, feature, parents, policy)
	rank := func(id string) int {
		if value, ok := ranks[id]; ok {
			return value
		}
		return MaxDenseDepth + 1
	}
	if policy == GraphFirstPolicyID {
		return append(prefix, fact.Metadata.SourceOrdinal, fact.OccurrenceByte, factKey(fact))
	}
	return append(prefix, rank(fact.AnchorID), rank(fact.EndpointID), fact.Metadata.SourceOrdinal, fact.OccurrenceByte, factKey(fact))
}

func metadataAdmissionPrefix(fact Fact, feature queryFeatures, parents map[string]Parent, policy string) []any {
	anchor, endpoint := parents[fact.AnchorID], parents[fact.EndpointID]
	qualifierMismatch := qualifierMismatch(feature, anchor, endpoint, fact.Metadata.FileRole)
	contextOverlap := tokenOverlap(feature.Tokens, fact.Metadata.ContextIdentifiers)
	endpointOverlap := tokenOverlap(feature.Tokens, parentTokens(endpoint))
	anchorOverlap := tokenOverlap(feature.Tokens, parentTokens(anchor))
	intentMismatch := metadataIntentMismatch(feature, fact)
	productionPenalty := 0
	if !hasExplicitRole(feature) && fact.Metadata.FileRole != ProductionFileRole {
		productionPenalty = 1
	}
	sameFilePenalty := 1
	if anchor.Path == endpoint.Path {
		sameFilePenalty = 0
	}
	prefix := []any{qualifierMismatch}
	if policy == ValueParameterDenseFirstPolicyID {
		prefix = append(prefix, valueParameterMismatch(feature, fact))
	}
	return append(prefix, -contextOverlap, -endpointOverlap, -anchorOverlap, intentMismatch, sameFilePenalty, productionPenalty)
}
func valueParameterMismatch(feature queryFeatures, fact Fact) int {
	if !feature.ValueParameterContractIntent {
		return 0
	}
	if fact.Direction == Forward && fact.Kind == TypeRef && fact.Metadata.Zone == SignatureZone && fact.Metadata.Role == TypeValueParameterRole {
		return 0
	}
	return 1
}
func qualifierMismatch(feature queryFeatures, anchor, endpoint Parent, occurrenceRole FileRole) int {
	if feature.DeprecatedIntent && !anchor.Deprecated && !endpoint.Deprecated {
		return 1
	}
	if len(feature.ExplicitFileRoles) > 0 {
		for _, role := range feature.ExplicitFileRoles {
			if role == occurrenceRole {
				return 0
			}
		}
		return 1
	}
	return 0
}
func hasExplicitRole(feature queryFeatures) bool { return len(feature.ExplicitFileRoles) > 0 }
func metadataIntentMismatch(feature queryFeatures, fact Fact) int {
	if feature.SignatureIntent && fact.Metadata.Zone != SignatureZone && fact.Metadata.Zone != TypeBodyZone {
		return 1
	}
	if feature.SignatureIntent && !(fact.Metadata.Role == TypeParameterRole || fact.Metadata.Role == TypeReturnRole || fact.Metadata.Role == TypeFieldRole || fact.Metadata.Role == TypeAliasRole || fact.Metadata.Role == TypeHeritageRole || fact.Metadata.Role == TypeArgumentRole || fact.Metadata.Role == TypeLocalRole || fact.Metadata.Role == TypeOtherRole || fact.Kind == TypeRef) {
		return 1
	}
	if feature.MutationIntent && fact.Metadata.Flow != FlowAssignment {
		return 1
	}
	if feature.ReturnIntent && fact.Metadata.Flow != FlowReturn {
		return 1
	}
	if feature.ConditionIntent && fact.Metadata.Flow != FlowCondition {
		return 1
	}
	if feature.Direction != fact.Direction {
		return 1
	}
	return 0
}
func parentTokens(parent Parent) []string {
	return strings.Fields((symbol.IdentifierNormalizer{}).Normalize(parent.Symbol + " " + parent.QualifiedSymbol))
}
func tokenOverlap(left, right []string) int {
	set := map[string]bool{}
	for _, value := range left {
		set[value] = true
	}
	total := 0
	for _, value := range right {
		if set[value] {
			total++
		}
	}
	return total
}
func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func stableUniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}
func uniqueFileRoles(values []FileRole) []FileRole {
	seen := map[FileRole]bool{}
	var result []FileRole
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
func lessSelectionKey(left, right []any) bool {
	for index := range left {
		switch a := left[index].(type) {
		case int:
			b := right[index].(int)
			if a != b {
				return a < b
			}
		case string:
			b := right[index].(string)
			if a != b {
				return a < b
			}
		default:
			panic("unsupported relation selection key")
		}
	}
	return false
}

func equalSelectionKey(left, right []any) bool {
	return !lessSelectionKey(left, right) && !lessSelectionKey(right, left)
}

func selectBundle(query string, facts []Fact, ranks map[string]int) Bundle {
	result := Bundle{QueryID: query, SelectionPolicy: SelectionPolicyID}
	if len(facts) == 0 {
		return result
	}
	sort.SliceStable(facts, func(i, j int) bool { return lessFact(facts[i], facts[j], ranks) })
	selected := facts[0]
	result.Selected = &selected
	primary := func(id string) bool { rank, ok := ranks[id]; return ok && rank <= ProtectedPrimaryK }
	for _, id := range []string{selected.AnchorID, selected.EndpointID} {
		if !primary(id) && !containsString(result.AddedParentIDs, id) && len(result.AddedParentIDs) < RelatedParentLimit {
			result.AddedParentIDs = append(result.AddedParentIDs, id)
		}
	}
	return result
}
func lessFact(a, b Fact, ranks map[string]int) bool {
	role := func(kind RelationKind) int {
		switch kind {
		case TypeRef:
			return 0
		case Calls:
			return 1
		case MemberOf:
			return 2
		default:
			return 3
		}
	}
	if role(a.Kind) != role(b.Kind) {
		return role(a.Kind) < role(b.Kind)
	}
	rank := func(id string) int {
		if value, ok := ranks[id]; ok {
			return value
		}
		return MaxDenseDepth + 1
	}
	if rank(a.AnchorID) != rank(b.AnchorID) {
		return rank(a.AnchorID) < rank(b.AnchorID)
	}
	if rank(a.EndpointID) != rank(b.EndpointID) {
		return rank(a.EndpointID) < rank(b.EndpointID)
	}
	if a.OccurrenceByte != b.OccurrenceByte {
		return a.OccurrenceByte < b.OccurrenceByte
	}
	return factKey(a) < factKey(b)
}
func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func packageRelated(query string, bundle Bundle, parents map[string]Parent, primary []string) []RelatedBody {
	primarySet := map[string]bool{}
	for _, id := range primary {
		primarySet[id] = true
	}
	result := make([]RelatedBody, 0, len(bundle.AddedParentIDs))
	for _, id := range bundle.AddedParentIDs {
		if primarySet[id] {
			continue
		}
		parent, ok := parents[id]
		if !ok {
			result = append(result, RelatedBody{QueryID: query, ParentID: id, OmissionReason: "PARENT_NOT_FOUND"})
			continue
		}
		body := RelatedBody{QueryID: query, ParentID: id, BodyBytes: len(parent.SourceBody)}
		if body.BodyBytes > RelatedBodyLimit {
			body.OmissionReason = "BODY_TOO_LARGE"
		} else {
			body.BodyComplete = true
			body.BodySHA256 = sha256Hex([]byte(parent.SourceBody))
		}
		result = append(result, body)
	}
	return result
}
func toLexical(hits []rankHit) []lexical.Hit {
	result := make([]lexical.Hit, 0, len(hits))
	for _, hit := range hits {
		result = append(result, lexical.Hit{Path: hit.Path, IndexedSHA256: hit.IndexedSHA256, QualifiedSymbol: hit.QualifiedSymbol, StartByte: hit.StartByte, EndByte: hit.EndByte})
	}
	return result
}
func parentHit(parent Parent) lexical.Hit {
	return lexical.Hit{Path: parent.Path, IndexedSHA256: parent.IndexedSHA256, QualifiedSymbol: parent.QualifiedSymbol, StartByte: parent.StartByte, EndByte: parent.EndByte}
}

func parentClassification(item evalcontract.EvaluationCase, parent Parent) string {
	for _, negative := range item.HardNegatives {
		if parentContainsSpan(parent, negative.Span.Path, negative.Span.ContentSHA256, negative.Span.QualifiedSymbol, negative.Span.StartByte, negative.Span.EndByte) {
			return "HARD_NEGATIVE"
		}
	}
	for _, judgment := range item.Judgments {
		if parentContainsSpan(parent, judgment.Span.Path, judgment.Span.ContentSHA256, judgment.Span.QualifiedSymbol, judgment.Span.StartByte, judgment.Span.EndByte) {
			return fmt.Sprintf("GRADE_%d", judgment.Grade)
		}
	}
	return "UNREVIEWED"
}
func parentRequired(item evalcontract.EvaluationCase, parent Parent) bool {
	for _, group := range item.RequiredGroups {
		for _, alternative := range group.Alternatives {
			for _, span := range alternative.Spans {
				if parentContainsSpan(parent, span.Path, span.ContentSHA256, span.QualifiedSymbol, span.StartByte, span.EndByte) {
					return true
				}
			}
		}
	}
	return false
}
func parentContainsSpan(parent Parent, path, hash, symbol string, start, end int) bool {
	return parent.Path == path && parent.IndexedSHA256 == hash && parent.QualifiedSymbol == symbol && parent.StartByte <= start && parent.EndByte >= end
}
func classifyAttachments(item evalcontract.EvaluationCase, bundle Bundle, related []RelatedBody, parents map[string]Parent) []ParentAttachment {
	var out []ParentAttachment
	seen := map[string]int{}
	add := func(id, role string) {
		if index, ok := seen[id]; ok {
			out[index].Role += "+" + role
			return
		}
		parent, ok := parents[id]
		classification, required := "UNREVIEWED", false
		if ok {
			classification = parentClassification(item, parent)
			required = parentRequired(item, parent)
		}
		seen[id] = len(out)
		out = append(out, ParentAttachment{ParentID: id, Role: role, Required: required, Classification: classification})
	}
	if bundle.Selected != nil {
		add(bundle.Selected.AnchorID, "anchor")
		add(bundle.Selected.EndpointID, "endpoint")
	}
	for _, body := range related {
		add(body.ParentID, "added")
	}
	return out
}

// diagnosticGate reports observed bad attachments after label-blind selection
// and packaging have frozen. It never changes selection: a hard negative (or
// the expressly guarded walkXFF parent) makes this diagnostic ineligible.
func diagnosticGate(traces []queryTrace, parents map[string]Parent) DiagnosticGate {
	type key struct{ parentID, reason string }
	queries := map[key]map[string]bool{}
	symbols := map[key]string{}
	for _, trace := range traces {
		for _, attachment := range trace.Attachments {
			parent, ok := parents[attachment.ParentID]
			if !ok {
				continue
			}
			reasons := []string{}
			if attachment.Classification == "HARD_NEGATIVE" {
				reasons = append(reasons, "ATTACHED_HARD_NEGATIVE")
			}
			if parent.QualifiedSymbol == "middleware.walkXFF" {
				reasons = append(reasons, "WALKXFF_ATTACHED")
			}
			for _, reason := range reasons {
				item := key{parentID: parent.ID, reason: reason}
				if queries[item] == nil {
					queries[item] = map[string]bool{}
					symbols[item] = parent.QualifiedSymbol
				}
				queries[item][trace.QueryID] = true
			}
		}
	}
	gate := DiagnosticGate{Eligible: len(queries) == 0}
	for item, querySet := range queries {
		queryIDs := sortedKeys(querySet)
		gate.Reasons = append(gate.Reasons, DiagnosticGateReason{ParentID: item.parentID, QualifiedSymbol: symbols[item], Reason: item.reason, QueryIDs: queryIDs})
	}
	sort.Slice(gate.Reasons, func(i, j int) bool {
		if gate.Reasons[i].ParentID != gate.Reasons[j].ParentID {
			return gate.Reasons[i].ParentID < gate.Reasons[j].ParentID
		}
		return gate.Reasons[i].Reason < gate.Reasons[j].Reason
	})
	return gate
}

func relationFirstLoss(item evalcontract.EvaluationCase, value queryTrace, parents map[string]Parent) string {
	completed := false
	for _, value := range value.Augmented.CompleteRequirementHitAt {
		completed = value
	}
	if completed {
		return "NO_LOSS"
	}
	stage := map[string]bool{}
	for _, fact := range value.StageAFacts {
		stage[fact.AnchorID] = true
		stage[fact.EndpointID] = true
	}
	selected := map[string]bool{}
	if value.Bundle.Selected != nil {
		selected[value.Bundle.Selected.AnchorID] = true
		selected[value.Bundle.Selected.EndpointID] = true
	}
	added := map[string]bool{}
	packaged := map[string]bool{}
	for _, body := range value.Related {
		added[body.ParentID] = true
		if body.BodyComplete {
			packaged[body.ParentID] = true
		}
	}
	best := 0
	for _, group := range item.RequiredGroups {
		groupBest := 0
		for _, alternative := range group.Alternatives {
			alternativeStage := 7 // completed is handled above, but retain a total order.
			for _, span := range alternative.Spans {
				mapped := []string{}
				for id, parent := range parents {
					if parentContainsSpan(parent, span.Path, span.ContentSHA256, span.QualifiedSymbol, span.StartByte, span.EndByte) {
						mapped = append(mapped, id)
					}
				}
				stageForSpan := 0 // target parent mapping
				for _, id := range mapped {
					parent := parents[id]
					if isPrimaryParent(value.PrimaryTop5, parent) {
						stageForSpan = 7
						break
					}
					if stage[id] && stageForSpan < 1 {
						stageForSpan = 1 // relation reachability
					}
					if selected[id] && stageForSpan < 2 {
						stageForSpan = 2 // relation admission
					}
					if added[id] && stageForSpan < 3 {
						stageForSpan = 3 // parent cap
					}
					if packaged[id] && stageForSpan < 4 {
						stageForSpan = 4 // packaging
					}
				}
				if stageForSpan < alternativeStage {
					alternativeStage = stageForSpan
				}
			}
			if alternativeStage > groupBest {
				groupBest = alternativeStage
			}
		}
		if groupBest < best || best == 0 {
			best = groupBest
		}
	}
	switch best {
	case 0:
		// The required parent exists in the bound inventory, but no verified
		// one-hop fact reached it from the fixed seed pool. Global resolver
		// denominators cannot safely attribute this query to one earlier stage.
		return "RELATION_REACHABILITY"
	case 1:
		return "RELATION_ADMISSION"
	case 2:
		return "BUNDLE_PARENT_CAP"
	case 3:
		return "RELATED_BODY_PACKAGING"
	case 4:
		return "REQUIRED_GROUP_COMPLETENESS"
	}
	return "REQUIRED_GROUP_COMPLETENESS"
}
func isPrimaryParent(hits []rankHit, parent Parent) bool {
	for _, hit := range hits {
		if hitKey(hit.Path, hit.IndexedSHA256, hit.QualifiedSymbol, hit.StartByte, hit.EndByte) == hitKey(parent.Path, parent.IndexedSHA256, parent.QualifiedSymbol, parent.StartByte, parent.EndByte) {
			return true
		}
	}
	return false
}

type probeFile struct {
	SchemaVersion int     `json:"schema_version"`
	Probes        []probe `json:"probes"`
}
type probeParent struct {
	Path            string `json:"path"`
	IndexedSHA256   string `json:"indexed_sha256"`
	QualifiedSymbol string `json:"qualified_symbol"`
	StartByte       int    `json:"start_byte"`
	EndByte         int    `json:"end_byte"`
}
type probeOccurrence struct {
	Path      string `json:"path"`
	StartByte int    `json:"start_byte"`
	EndByte   int    `json:"end_byte"`
}
type probe struct {
	ID                  string            `json:"id"`
	CorpusID            string            `json:"corpus_id"`
	Source              probeParent       `json:"source_parent"`
	Target              probeParent       `json:"target_parent"`
	Kind                RelationKind      `json:"relation_kind"`
	Direction           Direction         `json:"direction"`
	ExpectedCardinality int               `json:"expected_cardinality"`
	ExpectedOccurrences []probeOccurrence `json:"expected_occurrences"`
}
type probeResult struct {
	ID          string    `json:"id"`
	Direction   Direction `json:"direction"`
	Passed      bool      `json:"passed"`
	Matches     int       `json:"matches"`
	Occurrences []Fact    `json:"occurrences"`
}

func evaluateProbes(ctx context.Context, db *sql.DB, file string, parents map[string]Parent, corpusID string) ([]probeResult, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var values probeFile
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	if values.SchemaVersion != ProbeSchemaVersion || len(values.Probes) == 0 {
		return nil, fmt.Errorf("invalid relation probes")
	}
	var result []probeResult
	for _, probe := range values.Probes {
		if probe.ID == "" || probe.CorpusID == "" || !probe.Kind.Valid() || (probe.Direction != Forward && probe.Direction != Reverse) || probe.ExpectedCardinality <= 0 || len(probe.ExpectedOccurrences) != probe.ExpectedCardinality {
			return nil, fmt.Errorf("invalid relation probe")
		}
		if probe.CorpusID != corpusID {
			continue
		}
		source, err := exactProbeParent(probe.Source, parents)
		if err != nil {
			return nil, fmt.Errorf("%s source: %w", probe.ID, err)
		}
		target, err := exactProbeParent(probe.Target, parents)
		if err != nil {
			return nil, fmt.Errorf("%s target: %w", probe.ID, err)
		}
		expected := map[string]bool{}
		for _, occurrence := range probe.ExpectedOccurrences {
			if !validRelative(occurrence.Path) || occurrence.StartByte < 0 || occurrence.EndByte <= occurrence.StartByte {
				return nil, fmt.Errorf("%s has invalid expected occurrence", probe.ID)
			}
			key := probeOccurrenceKey(occurrence.Path, occurrence.StartByte, occurrence.EndByte)
			if expected[key] {
				return nil, fmt.Errorf("%s has duplicate expected occurrence", probe.ID)
			}
			expected[key] = true
		}
		querySource, queryTarget := source, target
		if probe.Direction == Reverse {
			querySource, queryTarget = target, source
		}
		rows, err := db.QueryContext(ctx, `SELECT relation_id,path,start_byte,end_byte FROM relation_occurrences WHERE source_parent_id=? AND target_parent_id=? AND relation_kind=? AND outcome='RESOLVED_UNIQUE' ORDER BY path,start_byte,end_byte,relation_id`, querySource, queryTarget, probe.Kind)
		if err != nil {
			return nil, err
		}
		facts := []Fact{}
		for rows.Next() {
			var relationID, path string
			var start, end int
			if err := rows.Scan(&relationID, &path, &start, &end); err != nil {
				rows.Close()
				return nil, err
			}
			if !expected[probeOccurrenceKey(path, start, end)] {
				rows.Close()
				return nil, fmt.Errorf("%s returned an unexpected occurrence span", probe.ID)
			}
			facts = append(facts, Fact{RelationID: relationID, Direction: probe.Direction, AnchorID: source, EndpointID: target, Kind: probe.Kind, OccurrencePath: path, OccurrenceByte: start, OccurrenceEndByte: end})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		if len(facts) != probe.ExpectedCardinality {
			return nil, fmt.Errorf("%s expected %d exact occurrences, got %d", probe.ID, probe.ExpectedCardinality, len(facts))
		}
		result = append(result, probeResult{ID: probe.ID, Direction: probe.Direction, Passed: true, Matches: len(facts), Occurrences: facts})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no relation probes for corpus %q", corpusID)
	}
	return result, nil
}

func exactProbeParent(expected probeParent, parents map[string]Parent) (string, error) {
	if !validRelative(expected.Path) || !validDigest(expected.IndexedSHA256) || expected.QualifiedSymbol == "" || expected.StartByte < 0 || expected.EndByte <= expected.StartByte {
		return "", fmt.Errorf("invalid immutable parent span")
	}
	var matches []string
	for id, parent := range parents {
		if parent.Path == expected.Path && parent.IndexedSHA256 == expected.IndexedSHA256 && parent.QualifiedSymbol == expected.QualifiedSymbol && parent.StartByte == expected.StartByte && parent.EndByte == expected.EndByte {
			matches = append(matches, id)
		}
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("expected exactly one immutable parent, got %d", len(matches))
	}
	return matches[0], nil
}
func probeOccurrenceKey(path string, start, end int) string {
	return fmt.Sprintf("%s\x00%d\x00%d", path, start, end)
}

func resolutionDenominators(ctx context.Context, db *sql.DB) (map[string]int, error) {
	rows, err := db.QueryContext(ctx, `SELECT outcome,COUNT(*) FROM relation_occurrences GROUP BY outcome ORDER BY outcome`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := map[string]int{}
	for rows.Next() {
		var outcome string
		var count int
		if err := rows.Scan(&outcome, &count); err != nil {
			return nil, err
		}
		values[outcome] = count
	}
	return values, rows.Err()
}

func writeEvaluationArtifacts(root string, traces []queryTrace, features map[string]queryFeatures, probes []probeResult, resolution map[string]int, gate DiagnosticGate, graph GraphManifest, replay frozenReplay, binding evaluationBinding, tierCounts map[StructuralTier]structuralTierCount) error {
	primary, stageA, bundles, bodies, complete := make([]any, 0, len(traces)), []any{}, make([]any, 0, len(traces)), []any{}, make([]any, 0, len(traces))
	for _, trace := range traces {
		digest, err := canonicalHash(trace.PrimaryTop5)
		if err != nil {
			return err
		}
		proofDigest, err := canonicalHash(trace.PrimaryBodyProofs)
		if err != nil {
			return err
		}
		primary = append(primary, map[string]any{"query_id": trace.QueryID, "primary_top5": trace.PrimaryTop5, "primary_top5_sha256": digest, "primary_body_hashes": trace.PrimaryBodyProofs, "primary_body_hashes_sha256": proofDigest, "identity_order_score_preserved": true})
		for _, fact := range trace.StageAFacts {
			stageA = append(stageA, map[string]any{"query_id": trace.QueryID, "fact": fact})
		}
		bundles = append(bundles, trace.Bundle)
		for _, body := range trace.Related {
			bodies = append(bodies, body)
		}
		complete = append(complete, trace)
	}
	featureRows := make([]any, 0, len(features))
	for _, queryID := range sortedKeys(features) {
		featureRows = append(featureRows, features[queryID])
	}
	admission := make([]any, 0, len(traces))
	for _, trace := range traces {
		admission = append(admission, map[string]any{"query_id": trace.QueryID, "selection_policy": trace.Bundle.SelectionPolicy, "admission_order": trace.Bundle.AdmissionOrder, "selected": trace.Bundle.Selected, "selection_key": trace.Bundle.SelectionKey, "added_parent_ids": trace.Bundle.AddedParentIDs})
	}
	for _, file := range []struct {
		name string
		rows []any
	}{
		{"query-features.jsonl", featureRows}, {"primary-top5-proof.jsonl", primary}, {"stage-a-reachability.jsonl", stageA}, {"stage-b-admission-order.jsonl", admission}, {"stage-b-bundles.jsonl", bundles}, {"related-body-packages.jsonl", bodies}, {"per-query-relation-trace.jsonl", complete},
	} {
		if err := writeJSONL(filepath.Join(root, file.name), file.rows); err != nil {
			return err
		}
	}
	if isAnchorEdgePolicy(binding.SelectionPolicy) {
		anchorGroups, directionalFacts, candidateStats, rankings := make([]any, 0, len(traces)), []any{}, []any{}, make([]any, 0, len(traces))
		denominatorRows := make([]any, 0, len(traces))
		for _, trace := range traces {
			if trace.AnchorEdge == nil {
				return fmt.Errorf("anchor-edge policy is missing per-query trace")
			}
			anchorGroups = append(anchorGroups, trace.AnchorEdge.Group)
			for _, fact := range trace.AnchorEdge.DirectionalFacts {
				directionalFacts = append(directionalFacts, map[string]any{"query_id": trace.QueryID, "fact": fact})
			}
			for _, candidate := range trace.AnchorEdge.Candidates {
				candidateStats = append(candidateStats, map[string]any{"query_id": trace.QueryID, "candidate": candidate})
			}
			rankings = append(rankings, map[string]any{"query_id": trace.QueryID, "policy": binding.SelectionPolicy, "selected": firstAnchorCandidate(trace.AnchorEdge.Candidates), "runner_up": secondAnchorCandidate(trace.AnchorEdge.Candidates), "selected_to_runner_up_first_differing_component": selectedRunnerDifference(trace.AnchorEdge.Candidates), "ranked_candidates": trace.AnchorEdge.Candidates})
			denominatorRows = append(denominatorRows, map[string]any{"query_id": trace.QueryID, "denominators": trace.AnchorEdge.Denominators, "stage_presence": trace.AnchorEdge.StagePresence})
		}
		for _, file := range []struct {
			name string
			rows []any
		}{
			{"anchor-groups.jsonl", anchorGroups}, {"directional-one-hop-facts.jsonl", directionalFacts}, {"edge-candidate-stats.jsonl", candidateStats}, {"policy-ranking.jsonl", rankings},
		} {
			if err := writeJSONL(filepath.Join(root, file.name), file.rows); err != nil {
				return err
			}
		}
		if err := writePortableJSON(filepath.Join(root, "structural-tier-counts.json"), tierCounts, ""); err != nil {
			return err
		}
		if err := writePortableJSON(filepath.Join(root, "denominator-report.json"), map[string]any{"policy": binding.SelectionPolicy, "queries": denominatorRows, "stages": []string{"anchor", "reachability", "strength", "cap", "packaging"}}, ""); err != nil {
			return err
		}
	}
	if isFrontierPolicy(binding.SelectionPolicy) {
		rows := make([]any, 0, len(traces))
		for _, trace := range traces {
			if trace.Frontier == nil {
				return fmt.Errorf("frontier policy is missing per-query trace")
			}
			rows = append(rows, map[string]any{"query_id": trace.QueryID, "frontier": trace.Frontier})
		}
		if err := writeJSONL(filepath.Join(root, "frontier-cap-diagnostic.jsonl"), rows); err != nil {
			return err
		}
	}
	if binding.SelectionPolicy == AnchorFrontierGraphOnlyParetoPolicyID {
		rows := make([]any, 0, len(traces))
		denominatorRows := make([]any, 0, len(traces))
		totals := map[string]int{"queries": len(traces)}
		outcomes := map[string]int{}
		for _, trace := range traces {
			if trace.Frontier == nil || trace.Frontier.GraphOnly == nil {
				return fmt.Errorf("graph-only frontier policy is missing per-query decision")
			}
			rows = append(rows, map[string]any{"query_id": trace.QueryID, "final_frontier_sha256": trace.Frontier.FinalDigest, "graph_only": trace.Frontier.GraphOnly})
			decision := trace.Frontier.GraphOnly
			finalCount := len(trace.Frontier.FinalFrontier)
			denominatorRows = append(denominatorRows, map[string]any{"query_id": trace.QueryID, "final_frontier_edges": finalCount, "direct_bridge_candidates": decision.DirectBridgeCandidates, "incoming_excluded": decision.IncomingExcluded, "dense_endpoint_excluded": decision.DenseEndpointExcluded, "graph_only_candidates": decision.GraphOnlyCandidates, "union_count": decision.UnionCount, "outcome": decision.Outcome})
			totals["final_frontier_edges"] += finalCount
			totals["direct_bridge_candidates"] += decision.DirectBridgeCandidates
			totals["incoming_excluded"] += decision.IncomingExcluded
			totals["dense_endpoint_excluded"] += decision.DenseEndpointExcluded
			totals["graph_only_candidates"] += decision.GraphOnlyCandidates
			totals["union_count"] += decision.UnionCount
			outcomes[decision.Outcome]++
		}
		if err := writeJSONL(filepath.Join(root, "frontier-graph-only-pareto.jsonl"), rows); err != nil {
			return err
		}
		if err := writePortableJSON(filepath.Join(root, "frontier-graph-only-pareto-denominators.json"), map[string]any{"policy": binding.SelectionPolicy, "queries": denominatorRows, "totals": totals, "outcomes": outcomes}, ""); err != nil {
			return err
		}
	}
	aggregate := aggregateMetrics(traces)
	aggregate["schema_version"], aggregate["selection_policy"], aggregate["metadata_policy"], aggregate["body_policy"], aggregate["primary_top5_protected"] = SchemaVersion, binding.SelectionPolicy, MetadataPolicyID, BodyPolicyID, true
	aggregate["policy_fingerprint"] = binding.PolicyFingerprint
	aggregate["resolution_outcomes"] = resolution
	aggregate["diagnostic_eligible"] = gate.Eligible
	aggregate["diagnostic_gate_reasons"] = gate.Reasons
	if err := writePortableJSON(filepath.Join(root, "aggregate-relation-metrics.json"), aggregate, ""); err != nil {
		return err
	}
	if err := writePortableJSON(filepath.Join(root, "probe-results.json"), probes, ""); err != nil {
		return err
	}
	if binding.GraphLogicalSHA256 != graph.LogicalGraphSHA256 || binding.GraphCorpusID != graph.Corpus.CorpusID || binding.ReplayCorpusID != replay.CorpusID || binding.ExpectedDatasetSHA256 != replay.SourceSHA256["dataset"] || binding.ExpectedDatasetFingerprint != replay.DatasetFingerprint || binding.SelectionPolicy == "" || binding.MetadataPolicy != MetadataPolicyID || !validDigest(binding.PolicyFingerprint) || !validDigest(binding.QueryFeaturesSHA256) {
		return fmt.Errorf("evaluation binding changed after selection")
	}
	manifest := map[string]any{"schema_version": SchemaVersion, "kind": "cidx.relation_diagnostic.v3", "created_at": time.Now().UTC().Format(time.RFC3339Nano), "queries": len(traces), "selection_policy": binding.SelectionPolicy, "metadata_policy": MetadataPolicyID, "policy_spec": relationPolicySpec(binding.SelectionPolicy), "policy_fingerprint": binding.PolicyFingerprint, "label_loading": "after_query_features_facts_order_selection_and_package_freeze", "zero_provider_operations": true, "calibration_only": true, "binding_verified_before_selection": true, "frozen_binding": binding, "resolution_outcomes": resolution, "diagnostic_eligible": gate.Eligible, "diagnostic_gate_reasons": gate.Reasons}
	if err := writePortableJSON(filepath.Join(root, "run-manifest.json"), manifest, ""); err != nil {
		return err
	}
	gateSummary := "eligible"
	if !gate.Eligible {
		gateSummary = fmt.Sprintf("ineligible: %d diagnostic gate reason(s)", len(gate.Reasons))
	}
	return os.WriteFile(filepath.Join(root, "report.md"), []byte(fmt.Sprintf("# Relation diagnostic\n\nQueries: %d\n\nDiagnostic eligibility: %s\n\nEvaluation-only diagnostic; not promotion evidence.\n", len(traces), gateSummary)), 0o600)
}

func firstAnchorCandidate(values []anchorEdgeCandidate) *anchorEdgeCandidate {
	if len(values) == 0 {
		return nil
	}
	value := values[0]
	return &value
}

func secondAnchorCandidate(values []anchorEdgeCandidate) *anchorEdgeCandidate {
	if len(values) < 2 {
		return nil
	}
	value := values[1]
	return &value
}

func selectedRunnerDifference(values []anchorEdgeCandidate) string {
	if len(values) < 2 {
		return ""
	}
	return values[1].FirstDifference
}
func writeJSONL(file string, rows []any) error {
	var data []byte
	for _, row := range rows {
		line, err := json.Marshal(row)
		if err != nil {
			return err
		}
		data = append(data, line...)
		data = append(data, '\n')
	}
	if !portableNoAbsolute(data) {
		return fmt.Errorf("unsafe relation JSONL output")
	}
	return os.WriteFile(file, data, 0o600)
}

func aggregateMetrics(traces []queryTrace) map[string]any {
	result := map[string]any{"queries": len(traces), "baseline_complete": 0, "augmented_complete": 0, "by_language": map[string]map[string]int{}, "by_cohort": map[string]map[string]int{}, "attachment_classification": map[string]int{}, "attachment_required": 0, "attachment_grade_2": 0, "attachment_support_grade_1": 0, "attachment_grade_0": 0, "attachment_hard_negative": 0, "attachment_unreviewed": 0, "first_loss": map[string]int{}, "related_body_bytes": 0, "related_omissions": map[string]int{}, "selected_bundles": 0, "walkxff_attachments": 0}
	languages := result["by_language"].(map[string]map[string]int)
	cohorts := result["by_cohort"].(map[string]map[string]int)
	reviews := result["attachment_classification"].(map[string]int)
	losses := result["first_loss"].(map[string]int)
	omissions := result["related_omissions"].(map[string]int)
	for _, trace := range traces {
		baseline := false
		for _, value := range trace.Baseline.CompleteRequirementHitAt {
			baseline = value
		}
		augmented := false
		for _, value := range trace.Augmented.CompleteRequirementHitAt {
			augmented = value
		}
		if baseline {
			result["baseline_complete"] = result["baseline_complete"].(int) + 1
		}
		if augmented {
			result["augmented_complete"] = result["augmented_complete"].(int) + 1
		}
		if trace.Bundle.Selected != nil {
			result["selected_bundles"] = result["selected_bundles"].(int) + 1
		}
		for _, attachment := range trace.Attachments {
			reviews[attachment.Classification]++
			if attachment.Required {
				result["attachment_required"] = result["attachment_required"].(int) + 1
			}
			switch attachment.Classification {
			case "GRADE_2":
				result["attachment_grade_2"] = result["attachment_grade_2"].(int) + 1
			case "GRADE_1":
				result["attachment_support_grade_1"] = result["attachment_support_grade_1"].(int) + 1
			case "GRADE_0":
				result["attachment_grade_0"] = result["attachment_grade_0"].(int) + 1
			case "HARD_NEGATIVE":
				result["attachment_hard_negative"] = result["attachment_hard_negative"].(int) + 1
			case "UNREVIEWED":
				result["attachment_unreviewed"] = result["attachment_unreviewed"].(int) + 1
			}
		}
		if trace.WalkXFFAttached {
			result["walkxff_attachments"] = result["walkxff_attachments"].(int) + 1
		}
		losses[trace.FirstLoss]++
		language := string(trace.Baseline.Language)
		if languages[language] == nil {
			languages[language] = map[string]int{}
		}
		languages[language]["queries"]++
		if baseline {
			languages[language]["baseline_complete"]++
		}
		if augmented {
			languages[language]["augmented_complete"]++
		}
		for _, cohort := range trace.Baseline.Cohorts {
			if cohorts[cohort] == nil {
				cohorts[cohort] = map[string]int{}
			}
			cohorts[cohort]["queries"]++
			if baseline {
				cohorts[cohort]["baseline_complete"]++
			}
			if augmented {
				cohorts[cohort]["augmented_complete"]++
			}
		}
		for _, body := range trace.Related {
			result["related_body_bytes"] = result["related_body_bytes"].(int) + body.BodyBytes
			if body.OmissionReason != "" {
				omissions[body.OmissionReason]++
			}
		}
	}
	return result
}
