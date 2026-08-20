package relationdiag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	PackagingProtocolVersion           = "cidx.relation-packaging.v1"
	PackagingContractKind              = "cidx.relation_packaging.experiment_contract.v1"
	PackagingPrimaryIdentityPolicyID   = "protected-dense-top5-identity-order-v1"
	PackagingSiblingEligibilityID      = "same-path-current-semantic-parent-complete-body-absent-from-primary-v1"
	PackagingSiblingBytePolicyID       = "greedy-skip-oversize-v1"
	PackagingKeepProxyID               = "target-file-is-primary-file-or-source-parent-is-primary-parent-v1"
	PackagingClusterSchemaID           = "relation-one-hop-cluster-v1"
	PackagingDecisionContinueSibling   = "CONTINUE_SIBLING_PACKAGING"
	PackagingDecisionContinueOneHop    = "CONTINUE_ONE_HOP_CLUSTERS"
	PackagingDecisionContinueBoth      = "CONTINUE_BOTH_FOR_ONE_COMBINED_TEST"
	PackagingDecisionStop              = "STOP_DEFAULT_GRAPH_PUSH"
	PackagingDecisionInconclusive      = "INCONCLUSIVE"
	PackagingMissSibling               = "SIBLING_NOT_PACKAGED"
	PackagingMissNeededFile            = "NEEDED_FILE_ABSENT"
	PackagingOmitSiblingCount          = "SIBLING_COUNT_CAP"
	PackagingOmitSiblingBytes          = "SIBLING_BYTE_CAP"
	PackagingOmitSiblingBody           = "SIBLING_MISSING_BODY"
	PackagingOmitSiblingDedupe         = "SIBLING_ALREADY_EMITTED"
	PackagingOmitClusterFiles          = "CLUSTER_FILE_CAP"
	PackagingOmitClusterBytes          = "CLUSTER_BYTE_CAP"
	PackagingOmitIsolatedHop           = "ISOLATED_HOP_OMITTED"
	packagingAcceptedImplementationSHA = "ba44fabac49d257323909ea118c66b9d8a053b9a"
	packagingFrozenDigest              = "002a30b08e137467896df63f2e5da8bf176c965f06c6a164aee7fd4db565a19b"
	packagingOverlapDiagnosticSHA      = "33a91723549c12486da93c07a638907537ff5065ad7375432109cbb19939656d"
)

type PackagingBudget struct {
	Count int `json:"count,omitempty"`
	Files int `json:"files,omitempty"`
	Bytes int `json:"bytes"`
}

type PackagingSiblingContract struct {
	Eligibility  string          `json:"eligibility"`
	BytePolicy   string          `json:"byte_policy"`
	CountGrid    []int           `json:"count_grid"`
	ByteGrid     []int           `json:"byte_grid"`
	DecisionCell PackagingBudget `json:"decision_cell"`
}

type PackagingOneHopContract struct {
	FrontierPolicy string          `json:"frontier_policy"`
	KeepProxy      string          `json:"keep_proxy"`
	FileGrid       []int           `json:"file_grid"`
	ByteGrid       []int           `json:"byte_grid"`
	DecisionCell   PackagingBudget `json:"decision_cell"`
}

type PackagingGates struct {
	SiblingMissQueryIDs     []string `json:"sibling_miss_query_ids"`
	NearbyCrossFileQueryIDs []string `json:"nearby_cross_file_query_ids"`
	LimitationQueryIDs      []string `json:"limitation_query_ids"`
	MinSiblingRecovered     int      `json:"min_sibling_recovered"`
	MinNearbyRecovered      int      `json:"min_nearby_recovered"`
}

type PackagingCompletionRef struct {
	CorpusID  string `json:"corpus_id"`
	Directory string `json:"directory"`
}

type PackagingInputs struct {
	SeriesContract string                   `json:"series_contract"`
	PreparedDir    string                   `json:"prepared_dir"`
	FrozenDir      string                   `json:"frozen_dir"`
	Completions    []PackagingCompletionRef `json:"completions"`
}

type PackagingContract struct {
	SchemaVersion                  int                      `json:"schema_version"`
	Kind                           string                   `json:"kind"`
	ExperimentID                   string                   `json:"experiment_id"`
	ProtocolVersion                string                   `json:"protocol_version"`
	AcceptedImplementationBoundary string                   `json:"accepted_implementation_boundary"`
	FrozenDigest                   string                   `json:"frozen_digest"`
	OverlapDiagnosticSHA256        string                   `json:"overlap_diagnostic_sha256"`
	ArmDAuthorized                 bool                     `json:"arm_d_authorized"`
	Arms                           []string                 `json:"arms"`
	PrimaryIdentityPolicy          string                   `json:"primary_identity_policy"`
	Sibling                        PackagingSiblingContract `json:"sibling"`
	OneHop                         PackagingOneHopContract  `json:"one_hop"`
	Gates                          PackagingGates           `json:"gates"`
	Inputs                         PackagingInputs          `json:"inputs"`
	Digest                         string                   `json:"digest,omitempty"`
}

type PackagingRequest struct {
	Contract         PackagingContract
	PreparedDir      string
	FrozenDir        string
	Completions      []PackagingCompletionRef
	OutputDir        string
	RequireCanonical bool
}

type packagingQuery struct {
	QueryID        string
	CorpusID       string
	Language       string
	Cohorts        []string
	RequiredGroups []ReviewRequiredGroup
	Primary        []string
	Scores         []semanticParentScore
	Hints          []relationHint
}

type packagingHop struct {
	QueryID, SourceParentID, TargetParentID string
}

type packagingUniverse struct {
	Queries      []packagingQuery
	Parents      map[string]Parent
	Labels       map[string]reviewLabel
	Hops         []packagingHop
	Inconclusive string
}

type packagingParentRef struct {
	ParentID        string `json:"parent_id"`
	Path            string `json:"path"`
	QualifiedSymbol string `json:"qualified_symbol"`
	StartByte       int    `json:"start_byte"`
	EndByte         int    `json:"end_byte"`
	BodyBytes       int    `json:"body_bytes"`
}

type packagingOmission struct {
	ParentID string `json:"parent_id,omitempty"`
	Path     string `json:"path,omitempty"`
	Reason   string `json:"reason"`
}

type packagingCluster struct {
	Path            string `json:"path"`
	QualifiedSymbol string `json:"qualified_symbol"`
	ParentID        string `json:"parent_id"`
	RelationKind    string `json:"relation_kind"`
	Direction       string `json:"direction"`
	StructuralTier  string `json:"structural_tier"`
	Role            string `json:"role"`
	OccurrenceCount int    `json:"occurrence_count"`
	SerializedBytes int    `json:"serialized_bytes"`
}

type packagingQueryCell struct {
	SchemaVersion         int                 `json:"schema_version"`
	Kind                  string              `json:"kind"`
	Arm                   string              `json:"arm"`
	Cell                  PackagingBudget     `json:"cell"`
	QueryID               string              `json:"query_id"`
	CorpusID              string              `json:"corpus_id"`
	Language              string              `json:"language"`
	Cohorts               []string            `json:"cohorts"`
	PrimaryParentIDs      []string            `json:"primary_parent_ids"`
	PrimaryEqual          bool                `json:"primary_equal"`
	ExtraParentIDs        []string            `json:"extra_parent_ids"`
	PrimaryCount          int                 `json:"primary_count"`
	ExtraSiblingCount     int                 `json:"extra_sibling_count"`
	ExtraSiblingBytes     int                 `json:"extra_sibling_bytes"`
	ClusterFileCount      int                 `json:"cluster_file_count"`
	ClusterBytes          int                 `json:"cluster_bytes"`
	IsolatedHopsOmitted   int                 `json:"isolated_hops_omitted"`
	LabeledIsolatedExtras int                 `json:"labeled_isolated_noise_extras"`
	LabeledIsolatedFiles  int                 `json:"labeled_isolated_noise_files"`
	UnlabeledExtras       int                 `json:"unlabeled_extras"`
	BaselineGroups        int                 `json:"baseline_groups"`
	CompleteGroups        int                 `json:"complete_groups"`
	CompleteQuery         bool                `json:"complete_query"`
	MissingGroupIDs       []string            `json:"missing_group_ids"`
	MissingGroupClasses   map[string]string   `json:"missing_group_classes"`
	MissingRequiredRanks  map[string]int      `json:"missing_required_dense_ranks"`
	Omissions             []packagingOmission `json:"omissions"`
	Clusters              []packagingCluster  `json:"clusters,omitempty"`
}

type packagingCellAggregate struct {
	SchemaVersion         int             `json:"schema_version"`
	Kind                  string          `json:"kind"`
	Arm                   string          `json:"arm"`
	Cell                  PackagingBudget `json:"cell"`
	Queries               int             `json:"queries"`
	CompleteQueries       int             `json:"complete_queries"`
	CompleteGroups        int             `json:"complete_groups"`
	RequiredGroups        int             `json:"required_groups"`
	SiblingMissRecovered  int             `json:"sibling_miss_recovered"`
	NearbyRecovered       int             `json:"nearby_cross_file_recovered"`
	ALossToWin            int             `json:"a_loss_to_win"`
	AWinToLoss            int             `json:"a_win_to_loss"`
	PrimaryEqualQueries   int             `json:"primary_equal_queries"`
	IsolatedHopsOmitted   int             `json:"isolated_hops_omitted"`
	LabeledIsolatedExtras int             `json:"labeled_isolated_noise_extras"`
	LabeledIsolatedFiles  int             `json:"labeled_isolated_noise_files"`
	SiblingNotPackaged    int             `json:"sibling_not_packaged"`
	NeededFileAbsent      int             `json:"needed_file_absent"`
}

type PackagingDecision struct {
	SchemaVersion        int    `json:"schema_version"`
	Kind                 string `json:"kind"`
	Decision             string `json:"decision"`
	ContractDigest       string `json:"contract_digest"`
	SiblingGate          bool   `json:"sibling_gate"`
	OneHopGate           bool   `json:"one_hop_gate"`
	PrimaryEqual         bool   `json:"primary_equal"`
	InconclusiveReason   string `json:"inconclusive_reason,omitempty"`
	SiblingRecovered     int    `json:"sibling_miss_recovered"`
	SiblingMisses        int    `json:"sibling_misses"`
	NearbyRecovered      int    `json:"nearby_cross_file_recovered"`
	NearbyMisses         int    `json:"nearby_cross_file_misses"`
	BaselineComplete     int    `json:"baseline_complete_queries"`
	ArmBCompleteQueries  int    `json:"arm_b_complete_queries"`
	ArmCCompleteQueries  int    `json:"arm_c_complete_queries"`
	LimitationIncomplete bool   `json:"limitation_query_still_incomplete"`
}

type packagingEvaluation struct {
	Contract   PackagingContract
	Rows       []packagingQueryCell
	Aggregates []packagingCellAggregate
	Decision   PackagingDecision
}

type packagingClusterDisclosure struct {
	Schema          string `json:"schema"`
	Path            string `json:"path"`
	QualifiedSymbol string `json:"qualified_symbol"`
	RelationKind    string `json:"relation_kind"`
	Direction       string `json:"direction"`
	StructuralTier  string `json:"structural_tier"`
	Role            string `json:"role"`
	OccurrenceCount int    `json:"occurrence_count"`
}

func canonicalPackagingContract() PackagingContract {
	return PackagingContract{
		SchemaVersion:                  1,
		Kind:                           PackagingContractKind,
		ExperimentID:                   "relation-packaging-v1",
		ProtocolVersion:                PackagingProtocolVersion,
		AcceptedImplementationBoundary: packagingAcceptedImplementationSHA,
		FrozenDigest:                   packagingFrozenDigest,
		OverlapDiagnosticSHA256:        packagingOverlapDiagnosticSHA,
		ArmDAuthorized:                 false,
		Arms:                           []string{"A", "B", "C"},
		PrimaryIdentityPolicy:          PackagingPrimaryIdentityPolicyID,
		Sibling: PackagingSiblingContract{
			Eligibility:  PackagingSiblingEligibilityID,
			BytePolicy:   PackagingSiblingBytePolicyID,
			CountGrid:    []int{2, 4, 8},
			ByteGrid:     []int{2048, 4096, 8192},
			DecisionCell: PackagingBudget{Count: 4, Bytes: 4096},
		},
		OneHop: PackagingOneHopContract{
			FrontierPolicy: AnchorFrontierCapOnlyPolicyID,
			KeepProxy:      PackagingKeepProxyID,
			FileGrid:       []int{2, 4, 8},
			ByteGrid:       []int{2048, 4096, 8192},
			DecisionCell:   PackagingBudget{Files: 4, Bytes: 4096},
		},
		Gates: PackagingGates{
			SiblingMissQueryIDs: []string{
				"me-x02-memo-editor",
				"gg-g06-commit-object",
				"me-x05-ai-provider-contract",
				"zu-t08-create-bound-contract",
				"me-x06-navigation-item",
				"me-g06-schedule-matchers",
			},
			NearbyCrossFileQueryIDs: []string{"gg-g07-diff-header-contract", "gg-g08-topological-node"},
			LimitationQueryIDs:      []string{"gg-g09-rename-change"},
			MinSiblingRecovered:     4,
			MinNearbyRecovered:      1,
		},
		Inputs: PackagingInputs{
			SeriesContract: "testdata/retrieval/relation-calibration-review-series-v1.json",
			PreparedDir:    ".cidx/test/experiments/relation-calibration-review-v1/prepared",
			FrozenDir:      ".cidx/test/experiments/relation-calibration-review-v1/frozen-ba44",
			Completions: []PackagingCompletionRef{
				{CorpusID: "go-git-go-git-v5.19.1", Directory: ".cidx/test/states/go-git-1024-int8/evaluations/relation-completion-stage-b-go-git-v2"},
				{CorpusID: "pmndrs-zustand-v5.0.14", Directory: ".cidx/test/states/zustand-1024-int8/evaluations/relation-completion-stage-b-zustand-v2"},
				{CorpusID: "usememos-memos-v0.30.0", Directory: ".cidx/test/states/memos-1024-int8/evaluations/relation-completion-stage-b-memos-v2"},
			},
		},
	}
}

func packagingContractDigest(value PackagingContract) (string, error) {
	value.Digest = ""
	return canonicalHash(value)
}

func exactCanonicalPackagingContract(value PackagingContract) bool {
	expected := canonicalPackagingContract()
	digest, err := packagingContractDigest(expected)
	if err != nil {
		return false
	}
	expected.Digest = digest
	got, err := packagingContractDigest(value)
	if err != nil {
		return false
	}
	value.Digest = got
	return expected.Kind == value.Kind && expected.ExperimentID == value.ExperimentID && expected.Digest == got && expected.FrozenDigest == value.FrozenDigest && expected.ArmDAuthorized == value.ArmDAuthorized && expected.Sibling.DecisionCell == value.Sibling.DecisionCell && expected.OneHop.DecisionCell == value.OneHop.DecisionCell
}

func validatePackagingContract(value PackagingContract) error {
	if value.SchemaVersion != 1 || value.Kind != PackagingContractKind || value.ExperimentID == "" || value.ProtocolVersion != PackagingProtocolVersion || value.PrimaryIdentityPolicy != PackagingPrimaryIdentityPolicyID || value.ArmDAuthorized || len(value.Arms) != 3 || value.Arms[0] != "A" || value.Arms[1] != "B" || value.Arms[2] != "C" {
		return fmt.Errorf("invalid packaging contract identity")
	}
	if value.Sibling.Eligibility != PackagingSiblingEligibilityID || value.Sibling.BytePolicy != PackagingSiblingBytePolicyID || !intMember(value.Sibling.DecisionCell.Count, value.Sibling.CountGrid) || !intMember(value.Sibling.DecisionCell.Bytes, value.Sibling.ByteGrid) {
		return fmt.Errorf("invalid packaging sibling contract")
	}
	if value.OneHop.FrontierPolicy != AnchorFrontierCapOnlyPolicyID || value.OneHop.KeepProxy != PackagingKeepProxyID || !intMember(value.OneHop.DecisionCell.Files, value.OneHop.FileGrid) || !intMember(value.OneHop.DecisionCell.Bytes, value.OneHop.ByteGrid) {
		return fmt.Errorf("invalid packaging one-hop contract")
	}
	if len(value.Gates.SiblingMissQueryIDs) == 0 || len(value.Gates.NearbyCrossFileQueryIDs) == 0 || value.Gates.MinSiblingRecovered < 1 || value.Gates.MinNearbyRecovered < 1 || value.Gates.MinSiblingRecovered > len(value.Gates.SiblingMissQueryIDs) || value.Gates.MinNearbyRecovered > len(value.Gates.NearbyCrossFileQueryIDs) {
		return fmt.Errorf("invalid packaging gates")
	}
	return nil
}

func Package(request PackagingRequest) (PackagingDecision, error) {
	if err := validatePackagingContract(request.Contract); err != nil {
		return PackagingDecision{}, err
	}
	digest, err := packagingContractDigest(request.Contract)
	if err != nil {
		return PackagingDecision{}, err
	}
	request.Contract.Digest = digest
	if request.RequireCanonical && !exactCanonicalPackagingContract(request.Contract) {
		return PackagingDecision{}, fmt.Errorf("packaging contract is not the frozen canonical experiment")
	}
	universe, err := loadPackagingUniverse(request)
	if err != nil {
		return PackagingDecision{}, err
	}
	evaluation, err := evaluatePackaging(request.Contract, universe)
	if err != nil {
		return PackagingDecision{}, err
	}
	if request.OutputDir != "" {
		if err := writePackagingArtifacts(request.OutputDir, evaluation); err != nil {
			return PackagingDecision{}, err
		}
	}
	return evaluation.Decision, nil
}

func loadPackagingUniverse(request PackagingRequest) (packagingUniverse, error) {
	if request.PreparedDir == "" || request.FrozenDir == "" || len(request.Completions) == 0 {
		return packagingUniverse{}, fmt.Errorf("packaging replay requires prepared, frozen, and completion inputs")
	}
	prepared, _, err := readReviewPrepared(request.PreparedDir)
	if err != nil {
		return packagingUniverse{}, err
	}
	if prepared.Digest == "" || (request.Contract.FrozenDigest != "" && request.Contract.AcceptedImplementationBoundary == packagingAcceptedImplementationSHA && prepared.Contract.QueryCount != 40) {
		return packagingUniverse{}, fmt.Errorf("prepared review is not the closed 40-query unit")
	}
	var frozen reviewFrozen
	if err = readReviewJSON(filepath.Join(request.FrozenDir, "frozen.json"), &frozen); err != nil {
		return packagingUniverse{}, err
	}
	if frozen.Digest != request.Contract.FrozenDigest {
		return packagingUniverse{}, fmt.Errorf("frozen digest does not match packaging contract")
	}
	var adoption ReviewAdoption
	if err = readReviewJSON(filepath.Join(request.FrozenDir, "owner-adoption.json"), &adoption); err != nil {
		return packagingUniverse{}, err
	}
	parentLabels, _, err := validatedReviewFrozenLabels(prepared, frozen, adoption)
	if err != nil {
		return packagingUniverse{}, err
	}
	queries := map[string]*packagingQuery{}
	order := make([]string, 0, len(prepared.Queries))
	parents := map[string]Parent{}
	for _, record := range prepared.Queries {
		query := packagingQuery{QueryID: record.Packet.QueryID, CorpusID: record.CorpusID, Language: record.Packet.Language, Cohorts: append([]string(nil), record.Cohorts...), RequiredGroups: append([]ReviewRequiredGroup(nil), record.RequiredGroups...), Primary: append([]string(nil), record.ProtectedTop5...)}
		stored := query
		queries[query.QueryID] = &stored
		order = append(order, query.QueryID)
	}
	for _, attachment := range prepared.Universe {
		parent := Parent{Path: attachment.Path, IndexedSHA256: attachment.IndexedSHA256, Language: attachment.Language, Kind: attachment.Kind, Symbol: attachment.Symbol, QualifiedSymbol: attachment.QualifiedSymbol, StartByte: attachment.StartByte, EndByte: attachment.EndByte}
		if attachment.Language != "" && attachment.Kind != "" && attachment.QualifiedSymbol != "" {
			parent.ID = ParentID(parent)
			parents[parent.ID] = parent
		}
	}
	labels := map[string]reviewLabel{}
	for _, attachment := range prepared.Universe {
		label, ok := parentLabels[attachment.AttachmentID]
		if !ok {
			continue
		}
		parent := parents[ParentID(Parent{Path: attachment.Path, IndexedSHA256: attachment.IndexedSHA256, Language: attachment.Language, Kind: attachment.Kind, QualifiedSymbol: attachment.QualifiedSymbol, StartByte: attachment.StartByte, EndByte: attachment.EndByte})]
		if parent.ID == "" {
			continue
		}
		labels[attachment.QueryID+"\x00"+parent.ID] = label
	}
	hops := make([]packagingHop, 0, len(prepared.Relations))
	attachmentParent := map[string]string{}
	for _, attachment := range prepared.Universe {
		if attachment.Language == "" || attachment.Kind == "" {
			continue
		}
		attachmentParent[attachment.AttachmentID] = ParentID(Parent{Path: attachment.Path, IndexedSHA256: attachment.IndexedSHA256, Language: attachment.Language, Kind: attachment.Kind, QualifiedSymbol: attachment.QualifiedSymbol, StartByte: attachment.StartByte, EndByte: attachment.EndByte})
	}
	for _, relation := range prepared.Relations {
		source, target := attachmentParent[relation.SourceAttachmentID], attachmentParent[relation.TargetAttachmentID]
		if source == "" || target == "" {
			continue
		}
		hops = append(hops, packagingHop{QueryID: relation.QueryID, SourceParentID: source, TargetParentID: target})
	}
	seenMembers := map[string]bool{}
	for _, ref := range request.Completions {
		features, hints, _, ids, manifest, err := loadReviewCompletion(ReviewCompletionInput{Directory: ref.Directory})
		if err != nil {
			return packagingUniverse{}, err
		}
		if ref.CorpusID != "" && manifest.CorpusID != ref.CorpusID {
			return packagingUniverse{}, fmt.Errorf("completion corpus mismatch")
		}
		if err := validateReviewMember(manifest, canonicalReviewSeriesContract().Members, len(ids), seenMembers); err != nil {
			return packagingUniverse{}, err
		}
		var scores []semanticParentScore
		if err := readReviewJSONL(filepath.Join(ref.Directory, "semantic-parent-scores.jsonl"), &scores); err != nil {
			return packagingUniverse{}, err
		}
		byQueryScores := map[string][]semanticParentScore{}
		for _, score := range scores {
			byQueryScores[score.QueryID] = append(byQueryScores[score.QueryID], score)
			if _, ok := parents[score.ParentID]; !ok {
				parents[score.ParentID] = Parent{ID: score.ParentID, Path: score.Path, IndexedSHA256: score.IndexedSHA256, QualifiedSymbol: score.QualifiedSymbol, StartByte: score.StartByte, EndByte: score.EndByte}
			}
		}
		byQueryHints := map[string][]relationHint{}
		for _, hint := range hints {
			byQueryHints[hint.QueryID] = append(byQueryHints[hint.QueryID], hint)
		}
		_ = features
		for _, id := range ids {
			query := queries[id]
			if query == nil {
				return packagingUniverse{}, fmt.Errorf("completion query missing from prepared topology")
			}
			query.Scores = byQueryScores[id]
			query.Hints = byQueryHints[id]
			sort.Slice(query.Scores, func(i, j int) bool {
				if query.Scores[i].GlobalRank != query.Scores[j].GlobalRank {
					return query.Scores[i].GlobalRank < query.Scores[j].GlobalRank
				}
				return query.Scores[i].ParentID < query.Scores[j].ParentID
			})
			if len(query.Scores) < ProtectedPrimaryK {
				return packagingUniverse{}, fmt.Errorf("completion query lacks protected top five")
			}
			for i := 0; i < ProtectedPrimaryK; i++ {
				if query.Scores[i].ParentID != query.Primary[i] {
					return packagingUniverse{Inconclusive: "primary top-five identity mismatch against prepared topology"}, nil
				}
			}
		}
	}
	result := packagingUniverse{Parents: parents, Labels: labels, Hops: hops}
	for _, id := range order {
		result.Queries = append(result.Queries, *queries[id])
	}
	if len(result.Queries) != prepared.Contract.QueryCount {
		return packagingUniverse{}, fmt.Errorf("packaging query cardinality mismatch")
	}
	return result, nil
}

func evaluatePackaging(contract PackagingContract, universe packagingUniverse) (packagingEvaluation, error) {
	if err := validatePackagingContract(contract); err != nil {
		return packagingEvaluation{}, err
	}
	digest, err := packagingContractDigest(contract)
	if err != nil {
		return packagingEvaluation{}, err
	}
	contract.Digest = digest
	if universe.Inconclusive != "" {
		return packagingEvaluation{Contract: contract, Decision: PackagingDecision{SchemaVersion: 1, Kind: "cidx.relation_packaging.decision.v1", Decision: PackagingDecisionInconclusive, ContractDigest: digest, InconclusiveReason: universe.Inconclusive}}, nil
	}
	if len(universe.Queries) == 0 {
		return packagingEvaluation{}, fmt.Errorf("packaging universe is empty")
	}
	rows := []packagingQueryCell{}
	for _, query := range universe.Queries {
		primary, err := packagingPrimary(query)
		if err != nil {
			return packagingEvaluation{Contract: contract, Decision: PackagingDecision{SchemaVersion: 1, Kind: "cidx.relation_packaging.decision.v1", Decision: PackagingDecisionInconclusive, ContractDigest: digest, InconclusiveReason: err.Error()}}, nil
		}
		armA := scorePackagingQuery(contract, universe, query, primary, "A", PackagingBudget{Count: ProtectedPrimaryK}, nil, nil, 0)
		rows = append(rows, armA)
		for _, count := range contract.Sibling.CountGrid {
			for _, bytes := range contract.Sibling.ByteGrid {
				extras, omissions, extraBytes := packageSiblings(query, primary, universe.Parents, count, bytes)
				rows = append(rows, scorePackagingQuery(contract, universe, query, primary, "B", PackagingBudget{Count: count, Bytes: bytes}, extras, omissions, extraBytes))
			}
		}
		for _, files := range contract.OneHop.FileGrid {
			for _, bytes := range contract.OneHop.ByteGrid {
				clusters, extras, omissions, isolated := packageOneHop(query, primary, universe.Parents, files, bytes)
				row := scorePackagingQuery(contract, universe, query, primary, "C", PackagingBudget{Files: files, Bytes: bytes}, extras, omissions, 0)
				row.Clusters = clusters
				row.ClusterFileCount = packagingClusterFileCount(clusters)
				row.ClusterBytes = packagingClusterByteSum(clusters)
				row.IsolatedHopsOmitted = isolated
				rows = append(rows, row)
			}
		}
	}
	aggregates := aggregatePackagingCells(contract, rows)
	decision := decidePackaging(contract, rows, aggregates)
	return packagingEvaluation{Contract: contract, Rows: rows, Aggregates: aggregates, Decision: decision}, nil
}

func packagingPrimary(query packagingQuery) ([]packagingParentRef, error) {
	if len(query.Primary) != ProtectedPrimaryK || len(query.Scores) < ProtectedPrimaryK {
		return nil, fmt.Errorf("query %s lacks protected top five", query.QueryID)
	}
	result := make([]packagingParentRef, 0, ProtectedPrimaryK)
	for i := 0; i < ProtectedPrimaryK; i++ {
		score := query.Scores[i]
		if score.ParentID != query.Primary[i] || score.GlobalRank != i+1 {
			return nil, fmt.Errorf("query %s primary identity or order mismatch", query.QueryID)
		}
		result = append(result, packagingRefFromScore(score))
	}
	return result, nil
}

func packagingRefFromScore(score semanticParentScore) packagingParentRef {
	bytes := score.EndByte - score.StartByte
	if bytes < 0 {
		bytes = 0
	}
	return packagingParentRef{ParentID: score.ParentID, Path: score.Path, QualifiedSymbol: score.QualifiedSymbol, StartByte: score.StartByte, EndByte: score.EndByte, BodyBytes: bytes}
}

func packageSiblings(query packagingQuery, primary []packagingParentRef, parents map[string]Parent, count, byteCap int) ([]packagingParentRef, []packagingOmission, int) {
	primaryIDs, primaryPaths := map[string]bool{}, map[string]bool{}
	fileOrder := []string{}
	for _, item := range primary {
		primaryIDs[item.ParentID] = true
		if !primaryPaths[item.Path] {
			primaryPaths[item.Path] = true
			fileOrder = append(fileOrder, item.Path)
		}
	}
	byPath := map[string][]packagingParentRef{}
	for _, score := range query.Scores {
		if primaryIDs[score.ParentID] || !primaryPaths[score.Path] {
			continue
		}
		byPath[score.Path] = append(byPath[score.Path], packagingRefFromScore(score))
	}
	candidates := []packagingParentRef{}
	for _, path := range fileOrder {
		values := byPath[path]
		sort.Slice(values, func(i, j int) bool {
			if values[i].QualifiedSymbol != values[j].QualifiedSymbol {
				return values[i].QualifiedSymbol < values[j].QualifiedSymbol
			}
			return values[i].ParentID < values[j].ParentID
		})
		candidates = append(candidates, values...)
	}
	emitted, seen := []packagingParentRef{}, map[string]bool{}
	omissions := []packagingOmission{}
	total := 0
	for _, candidate := range candidates {
		if seen[candidate.ParentID] || primaryIDs[candidate.ParentID] {
			omissions = append(omissions, packagingOmission{ParentID: candidate.ParentID, Path: candidate.Path, Reason: PackagingOmitSiblingDedupe})
			continue
		}
		if candidate.BodyBytes <= 0 {
			omissions = append(omissions, packagingOmission{ParentID: candidate.ParentID, Path: candidate.Path, Reason: PackagingOmitSiblingBody})
			continue
		}
		if len(emitted) >= count {
			omissions = append(omissions, packagingOmission{ParentID: candidate.ParentID, Path: candidate.Path, Reason: PackagingOmitSiblingCount})
			continue
		}
		if total+candidate.BodyBytes > byteCap {
			omissions = append(omissions, packagingOmission{ParentID: candidate.ParentID, Path: candidate.Path, Reason: PackagingOmitSiblingBytes})
			continue
		}
		emitted = append(emitted, candidate)
		seen[candidate.ParentID] = true
		total += candidate.BodyBytes
		_ = parents
	}
	return emitted, omissions, total
}

func packageOneHop(query packagingQuery, primary []packagingParentRef, parents map[string]Parent, fileCap, byteCap int) ([]packagingCluster, []packagingParentRef, []packagingOmission, int) {
	primaryIDs, primaryPaths := map[string]bool{}, map[string]bool{}
	for _, item := range primary {
		primaryIDs[item.ParentID] = true
		primaryPaths[item.Path] = true
	}
	byID := map[string]semanticParentScore{}
	for _, score := range query.Scores {
		byID[score.ParentID] = score
	}
	type accum struct {
		cluster packagingCluster
	}
	groups := map[string]*accum{}
	isolated := 0
	omissions := []packagingOmission{}
	for _, hint := range query.Hints {
		target := matchPackagingHintTarget(hint, query.Scores)
		source := matchPackagingHintSource(hint, query.Scores)
		if target.ParentID == "" {
			continue
		}
		if primaryIDs[target.ParentID] {
			continue
		}
		keep := primaryPaths[target.Path] || primaryIDs[source.ParentID]
		if !keep {
			isolated++
			omissions = append(omissions, packagingOmission{ParentID: target.ParentID, Path: target.Path, Reason: PackagingOmitIsolatedHop})
			continue
		}
		key := target.Path + "\x00" + target.QualifiedSymbol
		value := groups[key]
		role := string(TypeOtherRole)
		if parent, ok := parents[target.ParentID]; ok && parent.Kind != "" {
			_ = parent
		}
		if value == nil {
			value = &accum{cluster: packagingCluster{Path: target.Path, QualifiedSymbol: target.QualifiedSymbol, ParentID: target.ParentID, RelationKind: string(hint.Kind), Direction: string(hint.Direction), StructuralTier: string(hint.StructuralTier), Role: role, OccurrenceCount: hint.OccurrenceCount}}
			groups[key] = value
		} else {
			value.cluster.OccurrenceCount += hint.OccurrenceCount
			if hintKind := string(hint.Kind) + "\x00" + string(hint.Direction) + "\x00" + string(hint.StructuralTier); hintKind < value.cluster.RelationKind+"\x00"+value.cluster.Direction+"\x00"+value.cluster.StructuralTier {
				value.cluster.RelationKind, value.cluster.Direction, value.cluster.StructuralTier = string(hint.Kind), string(hint.Direction), string(hint.StructuralTier)
			}
		}
	}
	clusters := make([]packagingCluster, 0, len(groups))
	for _, value := range groups {
		disclosure := packagingClusterDisclosure{Schema: PackagingClusterSchemaID, Path: value.cluster.Path, QualifiedSymbol: value.cluster.QualifiedSymbol, RelationKind: value.cluster.RelationKind, Direction: value.cluster.Direction, StructuralTier: value.cluster.StructuralTier, Role: value.cluster.Role, OccurrenceCount: value.cluster.OccurrenceCount}
		data, err := json.Marshal(disclosure)
		if err != nil {
			continue
		}
		value.cluster.SerializedBytes = len(data)
		clusters = append(clusters, value.cluster)
	}
	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Path != clusters[j].Path {
			return clusters[i].Path < clusters[j].Path
		}
		if clusters[i].QualifiedSymbol != clusters[j].QualifiedSymbol {
			return clusters[i].QualifiedSymbol < clusters[j].QualifiedSymbol
		}
		return clusters[i].ParentID < clusters[j].ParentID
	})
	accepted := []packagingCluster{}
	extras := []packagingParentRef{}
	files, total := map[string]bool{}, 0
	for _, cluster := range clusters {
		if !files[cluster.Path] && len(files) >= fileCap {
			omissions = append(omissions, packagingOmission{ParentID: cluster.ParentID, Path: cluster.Path, Reason: PackagingOmitClusterFiles})
			continue
		}
		if total+cluster.SerializedBytes > byteCap {
			omissions = append(omissions, packagingOmission{ParentID: cluster.ParentID, Path: cluster.Path, Reason: PackagingOmitClusterBytes})
			continue
		}
		accepted = append(accepted, cluster)
		files[cluster.Path] = true
		total += cluster.SerializedBytes
		if score, ok := byID[cluster.ParentID]; ok {
			extras = append(extras, packagingRefFromScore(score))
		} else {
			extras = append(extras, packagingParentRef{ParentID: cluster.ParentID, Path: cluster.Path, QualifiedSymbol: cluster.QualifiedSymbol})
		}
	}
	return accepted, extras, omissions, isolated
}

func matchPackagingHintTarget(hint relationHint, scores []semanticParentScore) packagingParentRef {
	for _, score := range scores {
		if score.Path == hint.TargetPath && score.IndexedSHA256 == hint.TargetSHA256 && score.StartByte == hint.TargetStartByte && score.EndByte == hint.TargetEndByte {
			return packagingRefFromScore(score)
		}
	}
	return packagingParentRef{}
}

func matchPackagingHintSource(hint relationHint, scores []semanticParentScore) packagingParentRef {
	for _, score := range scores {
		if score.Path == hint.SourcePath && score.IndexedSHA256 == hint.SourceSHA256 && score.StartByte == hint.SourceStartByte && score.EndByte == hint.SourceEndByte {
			return packagingRefFromScore(score)
		}
	}
	return packagingParentRef{}
}

func scorePackagingQuery(contract PackagingContract, universe packagingUniverse, query packagingQuery, primary []packagingParentRef, arm string, cell PackagingBudget, extras []packagingParentRef, omissions []packagingOmission, extraBytes int) packagingQueryCell {
	payload := map[string]bool{}
	primaryIDs := make([]string, 0, len(primary))
	primaryPaths, primarySymbols := map[string]bool{}, map[string]bool{}
	for _, item := range primary {
		payload[item.ParentID] = true
		primaryIDs = append(primaryIDs, item.ParentID)
		primaryPaths[item.Path] = true
		primarySymbols[item.QualifiedSymbol] = true
	}
	extraIDs := make([]string, 0, len(extras))
	for _, item := range extras {
		payload[item.ParentID] = true
		extraIDs = append(extraIDs, item.ParentID)
	}
	complete, missing, classes, ranks := packagingCompleteness(query, payload)
	goldIDs, goldPaths, goldSymbols := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, group := range query.RequiredGroups {
		for _, id := range group.SourceParentIDs {
			goldIDs[id] = true
			if score := packagingScore(query, id); score.ParentID != "" {
				goldPaths[score.Path] = true
				goldSymbols[score.QualifiedSymbol] = true
			}
		}
	}
	for key, label := range universe.Labels {
		queryID, parentID, ok := strings.Cut(key, "\x00")
		if !ok || queryID != query.QueryID || label.Grade != 2 {
			continue
		}
		goldIDs[parentID] = true
		if score := packagingScore(query, parentID); score.ParentID != "" {
			goldPaths[score.Path] = true
			goldSymbols[score.QualifiedSymbol] = true
		}
	}
	neighbors := map[string][]string{}
	for _, hop := range universe.Hops {
		if hop.QueryID != query.QueryID {
			continue
		}
		neighbors[hop.SourceParentID] = append(neighbors[hop.SourceParentID], hop.TargetParentID)
		neighbors[hop.TargetParentID] = append(neighbors[hop.TargetParentID], hop.SourceParentID)
	}
	isolatedParents, isolatedFiles, unlabeled := 0, map[string]bool{}, 0
	for _, extra := range extras {
		label, labeled := universe.Labels[query.QueryID+"\x00"+extra.ParentID]
		if !labeled {
			unlabeled++
			continue
		}
		if label.Grade != 0 {
			continue
		}
		if packagingAdjacent(extra, goldIDs, goldPaths, goldSymbols, primaryIDs, primaryPaths, primarySymbols, neighbors) {
			continue
		}
		isolatedParents++
		isolatedFiles[extra.Path] = true
	}
	siblingCount := 0
	if arm == "B" {
		siblingCount = len(extras)
	}
	return packagingQueryCell{
		SchemaVersion: 1, Kind: "cidx.relation_packaging.query_cell.v1", Arm: arm, Cell: cell, QueryID: query.QueryID, CorpusID: query.CorpusID, Language: query.Language, Cohorts: append([]string(nil), query.Cohorts...),
		PrimaryParentIDs: primaryIDs, PrimaryEqual: packagingSameIDs(primaryIDs, query.Primary), ExtraParentIDs: extraIDs, PrimaryCount: len(primaryIDs), ExtraSiblingCount: siblingCount, ExtraSiblingBytes: extraBytes,
		CompleteGroups: complete, BaselineGroups: len(query.RequiredGroups), CompleteQuery: len(missing) == 0, MissingGroupIDs: missing, MissingGroupClasses: classes, MissingRequiredRanks: ranks, Omissions: omissions,
		LabeledIsolatedExtras: isolatedParents, LabeledIsolatedFiles: len(isolatedFiles), UnlabeledExtras: unlabeled,
	}
}

func packagingScore(query packagingQuery, parentID string) semanticParentScore {
	for _, score := range query.Scores {
		if score.ParentID == parentID {
			return score
		}
	}
	return semanticParentScore{}
}

func packagingCompleteness(query packagingQuery, payload map[string]bool) (int, []string, map[string]string, map[string]int) {
	complete := 0
	missing := []string{}
	classes := map[string]string{}
	ranks := map[string]int{}
	primaryPaths := map[string]bool{}
	for i := 0; i < ProtectedPrimaryK && i < len(query.Scores); i++ {
		primaryPaths[query.Scores[i].Path] = true
	}
	for _, group := range query.RequiredGroups {
		hit := false
		bestRank, sibling := 0, false
		for _, id := range group.SourceParentIDs {
			if payload[id] {
				hit = true
			}
			score := packagingScore(query, id)
			if score.ParentID != "" {
				if bestRank == 0 || score.GlobalRank < bestRank {
					bestRank = score.GlobalRank
				}
				if primaryPaths[score.Path] {
					sibling = true
				}
			}
		}
		if hit {
			complete++
			continue
		}
		missing = append(missing, group.ID)
		if sibling {
			classes[group.ID] = PackagingMissSibling
		} else {
			classes[group.ID] = PackagingMissNeededFile
		}
		if bestRank > 0 {
			ranks[group.ID] = bestRank
		}
	}
	sort.Strings(missing)
	return complete, missing, classes, ranks
}

func packagingAdjacent(extra packagingParentRef, goldIDs, goldPaths, goldSymbols map[string]bool, primaryIDs []string, primaryPaths, primarySymbols map[string]bool, neighbors map[string][]string) bool {
	if goldIDs[extra.ParentID] || goldPaths[extra.Path] || goldSymbols[extra.QualifiedSymbol] || primaryPaths[extra.Path] || primarySymbols[extra.QualifiedSymbol] {
		return true
	}
	for _, id := range primaryIDs {
		if id == extra.ParentID {
			return true
		}
	}
	for _, neighbor := range neighbors[extra.ParentID] {
		if goldIDs[neighbor] {
			return true
		}
		for _, id := range primaryIDs {
			if neighbor == id {
				return true
			}
		}
	}
	return false
}

func packagingSameIDs(left, right []string) bool {
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

func packagingClusterFileCount(values []packagingCluster) int {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value.Path] = true
	}
	return len(seen)
}

func packagingClusterByteSum(values []packagingCluster) int {
	total := 0
	for _, value := range values {
		total += value.SerializedBytes
	}
	return total
}

func aggregatePackagingCells(contract PackagingContract, rows []packagingQueryCell) []packagingCellAggregate {
	index := map[string]*packagingCellAggregate{}
	baseline := map[string]bool{}
	for _, row := range rows {
		if row.Arm == "A" {
			baseline[row.QueryID] = row.CompleteQuery
		}
	}
	siblingSet, nearbySet := packagingIDSet(contract.Gates.SiblingMissQueryIDs), packagingIDSet(contract.Gates.NearbyCrossFileQueryIDs)
	order := []string{}
	for _, row := range rows {
		key := row.Arm + "\x00" + fmt.Sprintf("%d:%d:%d", row.Cell.Count, row.Cell.Files, row.Cell.Bytes)
		value := index[key]
		if value == nil {
			value = &packagingCellAggregate{SchemaVersion: 1, Kind: "cidx.relation_packaging.cell_aggregate.v1", Arm: row.Arm, Cell: row.Cell}
			index[key] = value
			order = append(order, key)
		}
		value.Queries++
		value.RequiredGroups += row.BaselineGroups
		value.CompleteGroups += row.CompleteGroups
		if row.CompleteQuery {
			value.CompleteQueries++
		}
		if row.PrimaryEqual {
			value.PrimaryEqualQueries++
		}
		value.IsolatedHopsOmitted += row.IsolatedHopsOmitted
		value.LabeledIsolatedExtras += row.LabeledIsolatedExtras
		value.LabeledIsolatedFiles += row.LabeledIsolatedFiles
		if siblingSet[row.QueryID] && row.CompleteQuery {
			value.SiblingMissRecovered++
		}
		if nearbySet[row.QueryID] && row.CompleteQuery {
			value.NearbyRecovered++
		}
		if baseline[row.QueryID] && !row.CompleteQuery && row.Arm != "A" {
			value.AWinToLoss++
		}
		if !baseline[row.QueryID] && row.CompleteQuery && row.Arm != "A" {
			value.ALossToWin++
		}
		for _, class := range row.MissingGroupClasses {
			if class == PackagingMissSibling {
				value.SiblingNotPackaged++
			}
			if class == PackagingMissNeededFile {
				value.NeededFileAbsent++
			}
		}
	}
	result := make([]packagingCellAggregate, 0, len(order))
	for _, key := range order {
		result = append(result, *index[key])
	}
	return result
}

func packagingIDSet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		result[value] = true
	}
	return result
}

func decidePackaging(contract PackagingContract, rows []packagingQueryCell, aggregates []packagingCellAggregate) PackagingDecision {
	digest := contract.Digest
	decision := PackagingDecision{SchemaVersion: 1, Kind: "cidx.relation_packaging.decision.v1", ContractDigest: digest, SiblingMisses: len(contract.Gates.SiblingMissQueryIDs), NearbyMisses: len(contract.Gates.NearbyCrossFileQueryIDs)}
	var armA, armB, armC *packagingCellAggregate
	for i := range aggregates {
		row := &aggregates[i]
		switch {
		case row.Arm == "A":
			armA = row
		case row.Arm == "B" && row.Cell.Count == contract.Sibling.DecisionCell.Count && row.Cell.Bytes == contract.Sibling.DecisionCell.Bytes:
			armB = row
		case row.Arm == "C" && row.Cell.Files == contract.OneHop.DecisionCell.Files && row.Cell.Bytes == contract.OneHop.DecisionCell.Bytes:
			armC = row
		}
	}
	if armA == nil || armB == nil || armC == nil {
		decision.Decision = PackagingDecisionInconclusive
		decision.InconclusiveReason = "decision cells missing from aggregates"
		return decision
	}
	decision.BaselineComplete = armA.CompleteQueries
	decision.ArmBCompleteQueries = armB.CompleteQueries
	decision.ArmCCompleteQueries = armC.CompleteQueries
	decision.SiblingRecovered = armB.SiblingMissRecovered
	decision.NearbyRecovered = armC.NearbyRecovered
	decision.PrimaryEqual = armA.PrimaryEqualQueries == armA.Queries && armB.PrimaryEqualQueries == armB.Queries && armC.PrimaryEqualQueries == armC.Queries
	if !decision.PrimaryEqual {
		decision.Decision = PackagingDecisionInconclusive
		decision.InconclusiveReason = "primary top-five identity changed"
		return decision
	}
	limitation := packagingIDSet(contract.Gates.LimitationQueryIDs)
	for _, row := range rows {
		if row.Arm == "A" && limitation[row.QueryID] && !row.CompleteQuery {
			decision.LimitationIncomplete = true
		}
	}
	siblingOK := armB.SiblingMissRecovered >= contract.Gates.MinSiblingRecovered && armB.AWinToLoss == 0 && armB.LabeledIsolatedExtras <= contract.Sibling.DecisionCell.Count && decision.PrimaryEqual
	oneHopOK := armC.NearbyRecovered >= contract.Gates.MinNearbyRecovered && armC.AWinToLoss == 0 && packagingDefaultIsolatedHops(rows, "C", contract.OneHop.DecisionCell) == 0 && decision.PrimaryEqual
	baselineComplete := map[string]bool{}
	for _, row := range rows {
		if row.Arm == "A" {
			baselineComplete[row.QueryID] = row.CompleteQuery
		}
	}
	for _, row := range rows {
		if row.Arm == "C" && row.Cell.Files == contract.OneHop.DecisionCell.Files && row.Cell.Bytes == contract.OneHop.DecisionCell.Bytes && limitation[row.QueryID] && row.CompleteQuery && !baselineComplete[row.QueryID] {
			oneHopOK = false
		}
	}
	decision.SiblingGate = siblingOK
	decision.OneHopGate = oneHopOK
	switch {
	case siblingOK && oneHopOK:
		decision.Decision = PackagingDecisionContinueBoth
	case siblingOK:
		decision.Decision = PackagingDecisionContinueSibling
	case oneHopOK:
		decision.Decision = PackagingDecisionContinueOneHop
	default:
		decision.Decision = PackagingDecisionStop
	}
	return decision
}

func packagingDefaultIsolatedHops(rows []packagingQueryCell, arm string, cell PackagingBudget) int {
	leaked := 0
	for _, row := range rows {
		if row.Arm != arm || row.Cell != cell {
			continue
		}
		omitted := map[string]bool{}
		for _, omission := range row.Omissions {
			if omission.Reason == PackagingOmitIsolatedHop {
				omitted[omission.ParentID] = true
			}
		}
		for _, id := range row.ExtraParentIDs {
			if omitted[id] {
				leaked++
			}
		}
	}
	return leaked
}

func writePackagingArtifacts(root string, evaluation packagingEvaluation) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("packaging output directory must be empty")
	}
	if err := os.MkdirAll(filepath.Join(root, "arm-inputs"), 0o700); err != nil {
		return err
	}
	if err := writePortableJSON(filepath.Join(root, "experiment-contract.json"), evaluation.Contract, ""); err != nil {
		return err
	}
	binding := map[string]any{"kind": "cidx.relation_packaging.input_binding.v1", "contract_digest": evaluation.Contract.Digest, "query_count": len(uniquePackagingQueries(evaluation.Rows)), "arm_d_authorized": false, "provider_operations": 0}
	if err := writePortableJSON(filepath.Join(root, "arm-inputs", "input-binding.json"), binding, ""); err != nil {
		return err
	}
	queryRows := make([]any, len(evaluation.Rows))
	for i := range evaluation.Rows {
		queryRows[i] = evaluation.Rows[i]
	}
	if err := writeJSONL(filepath.Join(root, "per-query-results.jsonl"), queryRows); err != nil {
		return err
	}
	aggregateRows := make([]any, len(evaluation.Aggregates))
	for i := range evaluation.Aggregates {
		aggregateRows[i] = evaluation.Aggregates[i]
	}
	if err := writeJSONL(filepath.Join(root, "cell-aggregates.jsonl"), aggregateRows); err != nil {
		return err
	}
	if err := writePortableJSON(filepath.Join(root, "decision.json"), evaluation.Decision, ""); err != nil {
		return err
	}
	limitation := map[string]any{"kind": "cidx.relation_packaging.limitation_report.v1", "decision": evaluation.Decision.Decision, "limitation_query_ids": evaluation.Contract.Gates.LimitationQueryIDs, "limitation_query_still_incomplete": evaluation.Decision.LimitationIncomplete, "gg_g09_not_a_packaging_failure": true}
	if err := writePortableJSON(filepath.Join(root, "limitation-report.json"), limitation, ""); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "report.md"), []byte(packagingReportMarkdown(evaluation)), 0o600); err != nil {
		return err
	}
	return writeChecksums(root)
}

func uniquePackagingQueries(rows []packagingQueryCell) []string {
	seen, ids := map[string]bool{}, []string{}
	for _, row := range rows {
		if seen[row.QueryID] {
			continue
		}
		seen[row.QueryID] = true
		ids = append(ids, row.QueryID)
	}
	return ids
}

func packagingReportMarkdown(evaluation packagingEvaluation) string {
	var b strings.Builder
	b.WriteString("# Relation packaging experiment\n\n")
	b.WriteString("- Protocol: `" + PackagingProtocolVersion + "`\n")
	b.WriteString("- Contract digest: `" + evaluation.Contract.Digest + "`\n")
	b.WriteString("- Decision: `" + evaluation.Decision.Decision + "`\n")
	b.WriteString("- Provider operations: 0\n")
	b.WriteString("- Arm D authorized: false\n\n")
	if evaluation.Decision.InconclusiveReason != "" {
		b.WriteString("Inconclusive reason: " + evaluation.Decision.InconclusiveReason + "\n\n")
	}
	b.WriteString("## Decision cells\n\n")
	b.WriteString("| Arm | Complete queries | Sibling recovered | Nearby recovered | A-loss-to-win | A-win-to-loss | Isolated hops omitted |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, row := range evaluation.Aggregates {
		decision := false
		if row.Arm == "A" {
			decision = true
		}
		if row.Arm == "B" && row.Cell.Count == evaluation.Contract.Sibling.DecisionCell.Count && row.Cell.Bytes == evaluation.Contract.Sibling.DecisionCell.Bytes {
			decision = true
		}
		if row.Arm == "C" && row.Cell.Files == evaluation.Contract.OneHop.DecisionCell.Files && row.Cell.Bytes == evaluation.Contract.OneHop.DecisionCell.Bytes {
			decision = true
		}
		if !decision {
			continue
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d | %d |\n", row.Arm, row.CompleteQueries, row.SiblingMissRecovered, row.NearbyRecovered, row.ALossToWin, row.AWinToLoss, row.IsolatedHopsOmitted)
	}
	b.WriteString("\n`gg-g09` remaining incomplete is not a packaging failure.\n")
	b.WriteString("This artifact is evaluation-only. It does not authorize production search, MCP, confirmation, or assistant A/B.\n")
	return b.String()
}
