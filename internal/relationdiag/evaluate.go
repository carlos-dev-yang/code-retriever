package relationdiag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	QueryID           string     `json:"query_id"`
	Tokens            []string   `json:"tokens"`
	Direction         Direction  `json:"direction"`
	SignatureIntent   bool       `json:"signature_intent"`
	MutationIntent    bool       `json:"mutation_intent"`
	ReturnIntent      bool       `json:"return_intent"`
	ConditionIntent   bool       `json:"condition_intent"`
	DeprecatedIntent  bool       `json:"deprecated_intent"`
	ExplicitFileRoles []FileRole `json:"explicit_file_roles"`
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
	policyFingerprint, err := relationPolicyFingerprint()
	if err != nil {
		return EvaluationResult{}, err
	}
	binding := evaluationBinding{GraphLogicalSHA256: manifest.LogicalGraphSHA256, GraphCorpusID: manifest.Corpus.CorpusID, ReplayCorpusID: replay.CorpusID, ReplaySHA256: replaySHA, DatasetSHA256: datasetSourceSHA, ExpectedDatasetSHA256: replay.SourceSHA256["dataset"], ExpectedDatasetFingerprint: replay.DatasetFingerprint, ProbesSHA256: probesSHA, DenseLane: "dense_1024_int8", DenseDepth: MaxDenseDepth, SelectionPolicy: selectedPolicy(request.SelectionPolicy), MetadataPolicy: MetadataPolicyID, PolicyFingerprint: policyFingerprint, Evaluator: buildinfo.Current()}
	lane, ok := replay.Lanes["dense_1024_int8"]
	if !ok || len(lane.Ranks) == 0 {
		return EvaluationResult{}, fmt.Errorf("missing frozen dense_1024_int8 ranks")
	}
	if err := validateReplayRanks(lane); err != nil {
		return EvaluationResult{}, err
	}
	if selectedPolicy(request.SelectionPolicy) == GraphFirstPolicyID {
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
		facts, err := reachableFacts(ctx, db, seedIDs)
		if err != nil {
			return EvaluationResult{}, err
		}
		bundle := selectBundleWithPolicy(queryID, features[queryID], facts, rankPositions(lane.Ranks[queryID], byHit), byID, primaryIDs, policy)
		related := packageRelated(queryID, bundle, byID, primaryIDs)
		proofs := make([]PrimaryBodyProof, 0, len(primaryIDs))
		for _, id := range primaryIDs {
			parent := byID[id]
			proofs = append(proofs, PrimaryBodyProof{ParentID: id, BodySHA256: sha256Hex([]byte(parent.SourceBody))})
		}
		traces = append(traces, queryTrace{QueryID: queryID, PrimaryTop5: primary, PrimaryBodyProofs: proofs, StageAFacts: facts, Bundle: bundle, Related: related})
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
	if err := writeEvaluationArtifacts(temporary, traces, features, probes, resolution, gate, manifest, replay, binding); err != nil {
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

func relationPolicyFingerprint() (string, error) {
	return canonicalHash(relationPolicySpec())
}

func relationPolicySpec() map[string]any {
	return map[string]any{
		"metadata_policy":    MetadataPolicyID,
		"selection_policies": []string{DenseFirstPolicyID, GraphFirstPolicyID},
		"dense_first_tuple":  []string{"qualifier", "negative_context_overlap", "negative_endpoint_overlap", "negative_anchor_overlap", "intent_mismatch", "same_file", "occurrence_file_role", "anchor_dense_rank", "endpoint_dense_rank", "source_ordinal", "occurrence_byte", "stable_id"},
		"graph_first":        map[string]any{"seed_lanes": []string{"fts", "simple_control"}, "seed_k": ProtectedPrimaryK, "admission_prefix": []string{"qualifier", "negative_context_overlap", "negative_endpoint_overlap", "negative_anchor_overlap", "intent_mismatch", "same_file", "occurrence_file_role"}, "admitted_tier": "all facts tied at best prefix", "rerank_tuple": []string{"best_dense_endpoint_ordinal", "worst_dense_endpoint_ordinal", "source_ordinal", "occurrence_byte", "stable_id"}},
		"keyword_maps":       map[string][]string{"signature": {"contract", "props", "type", "interface", "signature", "options", "schema"}, "mutation": {"mutate", "set", "write", "assign"}, "return": {"return"}, "condition": {"condition", "when"}, "reverse": {"caller", "used-by"}, "deprecated": {"deprecated"}, "file_roles": {"test", "example", "benchmark"}},
		"caps":               map[string]int{"dense_depth": MaxDenseDepth, "protected_primary": ProtectedPrimaryK, "related_parent_limit": RelatedParentLimit, "related_body_limit": RelatedBodyLimit, "context_identifier_limit": 8},
		"qualifier_rules":    []string{"deprecated_query_requires_either_endpoint_deprecated", "no_deprecated_query_penalty", "explicit_file_role_uses_occurrence_file_role", "default_production_uses_occurrence_file_role"},
		"role_rules":         []string{"resolved_distinct_endpoints_only", "selection_never_reads_source_body", "package_related_applies_complete_body_cap"},
		"occurrence_zone":    []OccurrenceZone{SignatureZone, BodyZone, TypeBodyZone, InitializerZone},
		"occurrence_role":    []OccurrenceRole{CallFreeFunctionRole, CallMethodRole, CallableValueRole, TypeParameterRole, TypeReturnRole, TypeFieldRole, TypeAliasRole, TypeHeritageRole, TypeArgumentRole, TypeLocalRole, TypeOtherRole, MemberReceiverRole, MemberDeclarationRole},
		"flow_role":          []FlowRole{FlowNone, FlowReturn, FlowAssignment, FlowCondition, FlowArgument, FlowDeclaration},
		"file_role":          []FileRole{ProductionFileRole, TestFileRole, ExampleFileRole, BenchmarkFileRole},
		"execution_mode":     []ExecutionMode{DirectExecution, DeferredExecution, ConcurrentExecution, AwaitedExecution},
		"control_role":       []ControlRole{ControlNone, ControlBranch, ControlLoop, ControlSwitch, ControlTryCatch},
	}
}

func validateEvaluationRequest(v EvaluationRequest) error {
	if !strings.HasPrefix(v.RunID, "relation-diagnostic-") || !validRelative(v.RunID) || v.EvaluationRoot == "" || v.GraphDirectory == "" || v.ReplayPath == "" || v.DatasetPath == "" || v.ProbesPath == "" || (v.SelectionPolicy != "" && v.SelectionPolicy != DenseFirstPolicyID && v.SelectionPolicy != GraphFirstPolicyID) {
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
	if value.SchemaVersion != SchemaVersion || value.Kind != "cidx.relation_graph.v2" || value.Corpus.CorpusID == "" || !validDigest(value.LogicalGraphSHA256) || !validDigest(value.DatabaseSHA256) || !validDigest(value.SemanticParentInventorySHA256) || !validDigest(value.IndexedFileInventorySHA256) || value.ResolverPolicy["protocol"] != ProtocolVersion || value.ResolverPolicy["metadata"] != MetadataPolicyID {
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
			values[id] = hit.Rank
		}
	}
	return values
}

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
		feature := queryFeatures{QueryID: item.ID, Tokens: uniqueStrings(tokens), Direction: Forward}
		for _, token := range feature.Tokens {
			switch token {
			case "contract", "props", "type", "interface", "signature", "options", "schema":
				feature.SignatureIntent = true
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
			prefix := metadataAdmissionPrefix(fact, feature, parents)
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
	prefix := metadataAdmissionPrefix(fact, feature, parents)
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

func metadataAdmissionPrefix(fact Fact, feature queryFeatures, parents map[string]Parent) []any {
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
	return []any{qualifierMismatch, -contextOverlap, -endpointOverlap, -anchorOverlap, intentMismatch, sameFilePenalty, productionPenalty}
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

func writeEvaluationArtifacts(root string, traces []queryTrace, features map[string]queryFeatures, probes []probeResult, resolution map[string]int, gate DiagnosticGate, graph GraphManifest, replay frozenReplay, binding evaluationBinding) error {
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
	manifest := map[string]any{"schema_version": SchemaVersion, "kind": "cidx.relation_diagnostic.v2", "created_at": time.Now().UTC().Format(time.RFC3339Nano), "queries": len(traces), "selection_policy": binding.SelectionPolicy, "metadata_policy": MetadataPolicyID, "policy_spec": relationPolicySpec(), "policy_fingerprint": binding.PolicyFingerprint, "label_loading": "after_query_features_facts_order_selection_and_package_freeze", "zero_provider_operations": true, "binding_verified_before_selection": true, "frozen_binding": binding, "resolution_outcomes": resolution, "diagnostic_eligible": gate.Eligible, "diagnostic_gate_reasons": gate.Reasons}
	if err := writePortableJSON(filepath.Join(root, "run-manifest.json"), manifest, ""); err != nil {
		return err
	}
	gateSummary := "eligible"
	if !gate.Eligible {
		gateSummary = fmt.Sprintf("ineligible: %d diagnostic gate reason(s)", len(gate.Reasons))
	}
	return os.WriteFile(filepath.Join(root, "report.md"), []byte(fmt.Sprintf("# Relation diagnostic\n\nQueries: %d\n\nDiagnostic eligibility: %s\n\nEvaluation-only diagnostic; not promotion evidence.\n", len(traces), gateSummary)), 0o600)
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
