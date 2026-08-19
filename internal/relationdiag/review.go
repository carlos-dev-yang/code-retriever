package relationdiag

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const reviewBlindnessAttestation = "RANK_SCORE_ARM_DELIVERY_COHORT_CELL_OUTCOME_LABEL_BLIND"

// ReviewSeriesContract is tracked and source-free. Source bodies are supplied
// only through the local prepare input and are written solely to review packets.
type ReviewSeriesContract struct {
	SchemaVersion  int                  `json:"schema_version"`
	Kind           string               `json:"kind"`
	SeriesID       string               `json:"series_id"`
	AcceptedRepeat string               `json:"accepted_completion_repeat"`
	QueryCount     int                  `json:"query_count"`
	Endpoints      int                  `json:"raw_endpoint_features"`
	Hints          int                  `json:"raw_relation_hints"`
	Closures       int                  `json:"raw_contract_closure_candidates"`
	Members        []ReviewSeriesMember `json:"members"`
}

// ReviewSeriesMember is source-free provenance for one immutable completion
// input. Its checksum is the completion artifact checksum, never source text.
type ReviewSeriesMember struct {
	CorpusID                   string `json:"corpus_id"`
	DatasetSHA256              string `json:"dataset_sha256"`
	CompletionArtifactChecksum string `json:"completion_artifact_checksum"`
	QueryCount                 int    `json:"query_count"`
}

type ReviewSourceBody struct {
	ParentID        string `json:"parent_id,omitempty"`
	Path            string `json:"path"`
	IndexedSHA256   string `json:"indexed_sha256"`
	Language        string `json:"language,omitempty"`
	StartByte       int    `json:"start_byte"`
	EndByte         int    `json:"end_byte"`
	Kind            string `json:"kind,omitempty"`
	Symbol          string `json:"symbol,omitempty"`
	QualifiedSymbol string `json:"qualified_symbol,omitempty"`
	Signature       string `json:"signature,omitempty"`
	Body            string `json:"body"`
}

type ReviewCompletionInput struct {
	Directory  string             `json:"directory"`
	SourceRoot string             `json:"source_root,omitempty"`
	Bodies     []ReviewSourceBody `json:"bodies"`
	Queries    []ReviewQuery      `json:"queries"`
}

// ReviewQuery is local, source-backed topology for blinded review. Required
// groups remain unadopted until both review passes and owner adoption finish.
type ReviewQuery struct {
	QueryID        string                `json:"query_id"`
	Text           string                `json:"text"`
	Language       string                `json:"language"`
	AnswerMode     string                `json:"answer_mode"`
	Cohorts        []string              `json:"cohorts"`
	RequiredGroups []ReviewRequiredGroup `json:"required_groups"`
	ProtectedTop5  []string              `json:"protected_top5_parent_ids"`
}
type ReviewRequiredGroup struct {
	ID              string   `json:"id"`
	SourceParentIDs []string `json:"source_parent_ids"`
}

type ReviewPrepareRequest struct {
	Contract          ReviewSeriesContract    `json:"contract"`
	EmissionDirectory string                  `json:"emission_directory"`
	Completions       []ReviewCompletionInput `json:"completions"`
	OutputDir         string                  `json:"output_dir"`
}

// ReviewEmissionPrepareRequest deliberately has no source bodies, draft
// groups, relevance judgments, or hard-negative fields.  It freezes exactly
// the 25 predeclared closure/hint emission controls before topology is opened.
type ReviewEmissionQuery struct {
	QueryID       string   `json:"query_id"`
	Text          string   `json:"text"`
	Language      string   `json:"language"`
	AnswerMode    string   `json:"answer_mode"`
	ProtectedTop5 []string `json:"protected_top5_parent_ids"`
}
type ReviewEmissionCompletionInput struct {
	Directory string                `json:"directory"`
	Queries   []ReviewEmissionQuery `json:"queries"`
}
type ReviewEmissionPrepareRequest struct {
	Contract    ReviewSeriesContract            `json:"contract"`
	Completions []ReviewEmissionCompletionInput `json:"completions"`
	OutputDir   string                          `json:"output_dir"`
}

type ReviewAttachment struct {
	AttachmentID    string `json:"attachment_id"`
	QueryID         string `json:"query_id"`
	Path            string `json:"path"`
	IndexedSHA256   string `json:"indexed_sha256"`
	Language        string `json:"language,omitempty"`
	StartByte       int    `json:"start_byte"`
	EndByte         int    `json:"end_byte"`
	Kind            string `json:"kind,omitempty"`
	Symbol          string `json:"symbol,omitempty"`
	QualifiedSymbol string `json:"qualified_symbol,omitempty"`
	Signature       string `json:"signature,omitempty"`
	Body            string `json:"body"`
}

type ReviewPacket struct {
	SchemaVersion           int                    `json:"schema_version"`
	Kind                    string                 `json:"kind"`
	Policy                  string                 `json:"policy"`
	PreparedDigest          string                 `json:"prepared_digest"`
	PassID                  string                 `json:"pass_id"`
	CanonicalUniverseDigest string                 `json:"canonical_universe_digest"`
	Attachments             []ReviewAttachment     `json:"attachments"`
	Queries                 []ReviewPacketQuery    `json:"queries"`
	Relations               []ReviewPacketRelation `json:"relations"`
}

// ReviewPacketRelation hides the candidate delivery family while retaining
// the exact opaque source-target relation evidence for independent review.
type ReviewPacketRelation struct {
	AttachmentID       string   `json:"attachment_id"`
	QueryID            string   `json:"query_id"`
	SourceAttachmentID string   `json:"source_attachment_id"`
	TargetAttachmentID string   `json:"target_attachment_id"`
	RelationIDs        []string `json:"relation_ids"`
	RelationKind       string   `json:"relation_kind"`
	Direction          string   `json:"direction"`
	StructuralTier     string   `json:"structural_tier"`
	Role               string   `json:"role"`
	OccurrenceCount    int      `json:"occurrence_count"`
}
type ReviewRelationEvidence struct {
	AttachmentID       string   `json:"attachment_id"`
	QueryID            string   `json:"query_id"`
	DeliveryFamily     string   `json:"delivery_family"`
	SourceAttachmentID string   `json:"source_attachment_id"`
	TargetAttachmentID string   `json:"target_attachment_id"`
	RelationIDs        []string `json:"relation_ids"`
	RelationKind       string   `json:"relation_kind"`
	Direction          string   `json:"direction"`
	StructuralTier     string   `json:"structural_tier"`
	Role               string   `json:"role"`
	OccurrenceCount    int      `json:"occurrence_count"`
}
type ReviewPacketQuery struct {
	QueryID                   string   `json:"query_id"`
	Text                      string   `json:"text"`
	Language                  string   `json:"language"`
	AnswerMode                string   `json:"answer_mode"`
	UnadoptedRequiredGroupIDs []string `json:"unadopted_required_group_ids"`
}

type ReviewGrade struct {
	AttachmentID         string   `json:"attachment_id"`
	Grade                int      `json:"grade"`
	RequiredGroupIDs     []string `json:"required_group_ids"`
	HardNegative         bool     `json:"hard_negative"`
	HardNegativeGroupIDs []string `json:"hard_negative_group_ids"`
	HardNegativeReason   string   `json:"hard_negative_reason"`
	Rationale            string   `json:"rationale"`
}

type ReviewPass struct {
	SchemaVersion        int           `json:"schema_version"`
	Kind                 string        `json:"kind"`
	PreparedDigest       string        `json:"prepared_digest"`
	PacketDigest         string        `json:"packet_digest"`
	PassID               string        `json:"pass_id"`
	ReviewerID           string        `json:"reviewer_id"`
	ModelFamily          string        `json:"model_family"`
	SourceVerified       bool          `json:"source_verified"`
	BlindnessAttestation string        `json:"blindness_attestation"`
	Grades               []ReviewGrade `json:"grades"`
	RelationGrades       []ReviewGrade `json:"relation_grades"`
}

type ReviewAdjudication struct {
	Subject              string   `json:"subject"`
	AttachmentID         string   `json:"attachment_id"`
	Grade                int      `json:"grade"`
	RequiredGroupIDs     []string `json:"required_group_ids"`
	HardNegative         bool     `json:"hard_negative"`
	HardNegativeGroupIDs []string `json:"hard_negative_group_ids"`
	HardNegativeReason   string   `json:"hard_negative_reason"`
	Rationale            string   `json:"rationale"`
}
type ReviewAdjudications struct {
	SchemaVersion  int                  `json:"schema_version"`
	Kind           string               `json:"kind"`
	PreparedDigest string               `json:"prepared_digest"`
	PassOneDigest  string               `json:"pass_one_digest"`
	PassTwoDigest  string               `json:"pass_two_digest"`
	Entries        []ReviewAdjudication `json:"entries"`
}

type ReviewAdoption struct {
	SchemaVersion      int      `json:"schema_version"`
	Kind               string   `json:"kind"`
	FrozenDigest       string   `json:"frozen_digest"`
	Adopted            bool     `json:"adopted"`
	ProtocolVersion    string   `json:"protocol_version"`
	RelevanceAuthority string   `json:"relevance_authority"`
	ReviewValidation   string   `json:"review_validation"`
	Overrides          []string `json:"overrides"`
}

type reviewCandidate struct {
	AttachmentID     string   `json:"attachment_id"`
	QueryID          string   `json:"query_id"`
	Families         []string `json:"families"`
	RequiredGroupIDs []string `json:"unadopted_required_group_ids"`
}

type reviewPrepared struct {
	SchemaVersion  int                      `json:"schema_version"`
	Kind           string                   `json:"kind"`
	Policy         string                   `json:"policy"`
	Contract       ReviewSeriesContract     `json:"contract"`
	SemanticStatus string                   `json:"semantic_status"`
	ClosureCells   []ReviewBudgetCell       `json:"closure_cells"`
	HintCells      []ReviewBudgetCell       `json:"hint_cells"`
	Universe       []ReviewAttachment       `json:"canonical_universe"`
	UniverseDigest string                   `json:"canonical_universe_digest"`
	Emissions      []reviewEmission         `json:"emissions"`
	Candidates     []reviewCandidate        `json:"candidates"`
	Queries        []reviewQueryRecord      `json:"query_topology"`
	Relations      []ReviewRelationEvidence `json:"relation_attachments"`
	Digest         string                   `json:"digest"`
}
type reviewUniverseBinding struct {
	Attachments []ReviewAttachment       `json:"attachments"`
	Queries     []ReviewPacketQuery      `json:"queries"`
	Relations   []ReviewRelationEvidence `json:"relations"`
}

type reviewQueryRecord struct {
	CorpusID       string                `json:"corpus_id"`
	Packet         ReviewPacketQuery     `json:"packet"`
	Cohorts        []string              `json:"cohorts"`
	RequiredGroups []ReviewRequiredGroup `json:"required_groups"`
	ProtectedTop5  []string              `json:"protected_top5_parent_ids"`
}

type reviewEmission struct {
	QueryID               string           `json:"query_id"`
	Cell                  ReviewBudgetCell `json:"cell"`
	AttachmentIDs         []string         `json:"attachment_ids"`
	RelationAttachmentIDs []string         `json:"relation_attachment_ids"`
}
type reviewEmissionControl struct {
	QueryID        string           `json:"query_id"`
	Cell           ReviewBudgetCell `json:"cell"`
	CandidateCount int              `json:"candidate_count"`
	EmittedCount   int              `json:"emitted_count"`
	ActualBytes    int              `json:"actual_bytes"`
	OmissionCounts map[string]int   `json:"omission_counts"`
}
type reviewEmissionFreeze struct {
	SchemaVersion int                     `json:"schema_version"`
	Kind          string                  `json:"kind"`
	Contract      ReviewSeriesContract    `json:"contract"`
	Queries       []ReviewEmissionQuery   `json:"queries"`
	Controls      []reviewEmissionControl `json:"controls"`
	Digest        string                  `json:"digest"`
}

type reviewFrozen struct {
	SchemaVersion       int           `json:"schema_version"`
	Kind                string        `json:"kind"`
	PreparedDigest      string        `json:"prepared_digest"`
	PassOneDigest       string        `json:"pass_one_digest"`
	PassTwoDigest       string        `json:"pass_two_digest"`
	ReconciledDigest    string        `json:"reconciled_digest"`
	OwnerAdoptionSHA256 string        `json:"owner_adoption_sha256"`
	ParentLabels        []reviewLabel `json:"parent_labels"`
	RelationLabels      []reviewLabel `json:"relation_labels"`
	Digest              string        `json:"digest"`
}

type reviewLabel struct {
	AttachmentID         string   `json:"attachment_id"`
	Grade                int      `json:"grade"`
	HardNegative         bool     `json:"hard_negative"`
	GroupIDs             []string `json:"required_group_ids"`
	HardNegativeGroupIDs []string `json:"hard_negative_group_ids"`
}

func PrepareReview(request ReviewPrepareRequest) (string, error) {
	if err := validateReviewContract(request.Contract); err != nil || request.OutputDir == "" || request.EmissionDirectory == "" || len(request.Completions) != 3 {
		return "", fmt.Errorf("invalid relation review prepare request")
	}
	emissionFreeze, err := readReviewEmissionFreeze(request.EmissionDirectory, request.Contract)
	if err != nil {
		return "", err
	}
	endpoints, hints, closures, candidates, bodies, queryIDs := 0, 0, 0, map[string]*reviewCandidate{}, map[string]ReviewSourceBody{}, map[string]bool{}
	allFeatures, allHints, allClosures := []semanticEndpointFeature{}, []relationHint{}, []closureCandidate{}
	packetQueries := []reviewQueryRecord{}
	seenMembers := map[string]bool{}
	for _, input := range request.Completions {
		if input.SourceRoot == "" {
			return "", fmt.Errorf("source-root reproof is required for review packet preparation")
		}
		features, hintRows, closureRows, ids, err := loadReviewCompletion(input)
		if err != nil {
			return "", err
		}
		if err := validateReviewMember(input.Directory, request.Contract.Members, len(ids), seenMembers); err != nil {
			return "", err
		}
		endpoints += len(features)
		hints += len(hintRows)
		closures += len(closureRows)
		allFeatures, allHints, allClosures = append(allFeatures, features...), append(allHints, hintRows...), append(allClosures, closureRows...)
		for _, id := range ids {
			if queryIDs[id] {
				return "", fmt.Errorf("duplicate review query id")
			}
			queryIDs[id] = true
		}
		if err := validateReviewQueries(input.Queries, ids); err != nil {
			return "", err
		}
		var completionManifest struct {
			CorpusID     string `json:"corpus_id"`
			Kind         string `json:"kind"`
			LabelLoading string `json:"label_loading"`
		}
		if err := readReviewJSON(filepath.Join(input.Directory, "run-manifest.json"), &completionManifest); err != nil {
			return "", err
		}
		for _, query := range input.Queries {
			groups := []string{}
			for _, group := range query.RequiredGroups {
				groups = append(groups, group.ID)
			}
			sort.Strings(groups)
			packetQueries = append(packetQueries, reviewQueryRecord{CorpusID: completionManifest.CorpusID, Packet: ReviewPacketQuery{QueryID: query.QueryID, Text: query.Text, Language: query.Language, AnswerMode: query.AnswerMode, UnadoptedRequiredGroupIDs: groups}, Cohorts: uniqueSortedReviewIDs(query.Cohorts), RequiredGroups: append([]ReviewRequiredGroup(nil), query.RequiredGroups...), ProtectedTop5: append([]string(nil), query.ProtectedTop5...)})
		}
		for _, body := range input.Bodies {
			if err := validateReviewBody(input.SourceRoot, body); err != nil {
				return "", fmt.Errorf("invalid local review source body")
			}
			key := reviewBodyKey(body.Path, body.IndexedSHA256, body.StartByte, body.EndByte)
			if prior, ok := bodies[key]; ok && !sameReviewSourceBody(prior, body) {
				return "", fmt.Errorf("conflicting duplicate review source body")
			}
			bodies[key] = body
			if body.ParentID != "" {
				if prior, ok := bodies[body.ParentID]; ok && !sameReviewSourceBody(prior, body) {
					return "", fmt.Errorf("conflicting review parent body")
				}
				bodies[body.ParentID] = body
			}
		}
		for _, feature := range features {
			addReviewCandidate(candidates, feature.QueryID, feature.EndpointParentID, "semantic")
			addReviewCandidate(candidates, feature.QueryID, feature.AnchorParentID, "semantic_anchor")
		}
		for _, closure := range closureRows {
			addReviewCandidate(candidates, closure.QueryID, closure.TargetParentID, "closure")
			addReviewCandidate(candidates, closure.QueryID, closure.PrimaryParentID, "closure_source")
		}
		for _, hint := range hintRows {
			addReviewCandidate(candidates, hint.QueryID, reviewBodyKey(hint.TargetPath, hint.TargetSHA256, hint.TargetStartByte, hint.TargetEndByte), "hint")
			if hint.SourcePath != "" && validDigest(hint.SourceSHA256) && hint.SourceEndByte > hint.SourceStartByte {
				addReviewCandidate(candidates, hint.QueryID, reviewBodyKey(hint.SourcePath, hint.SourceSHA256, hint.SourceStartByte, hint.SourceEndByte), "hint_source")
			}
		}
		for _, query := range input.Queries {
			for _, parentID := range query.ProtectedTop5 {
				addReviewCandidate(candidates, query.QueryID, parentID, "protected_primary")
			}
			for _, group := range query.RequiredGroups {
				for _, parentID := range group.SourceParentIDs {
					addReviewCandidateGroup(candidates, query.QueryID, parentID, "truth_source", group.ID)
				}
			}
		}
	}
	if len(queryIDs) != request.Contract.QueryCount || endpoints != request.Contract.Endpoints || hints != request.Contract.Hints || closures != request.Contract.Closures {
		return "", fmt.Errorf("relation review completion totals do not match series contract")
	}
	if len(request.Contract.Members) != 0 && len(seenMembers) != len(request.Contract.Members) {
		return "", fmt.Errorf("incomplete relation review series members")
	}
	if !samePrelabelQueries(emissionFreeze.Queries, packetQueries) {
		return "", fmt.Errorf("packet topology differs from immutable pre-label query payload")
	}
	// A parent can arrive from an endpoint/closure by parent ID and from a
	// hint by source identity. Resolve both to the same query/body attachment
	// before either blinded packet is shuffled.
	resolved := map[string]*reviewCandidate{}
	resolvedBodies := map[string]ReviewSourceBody{}
	attachmentByReference := map[string]string{}
	values := make([]*reviewCandidate, 0, len(candidates))
	for _, v := range candidates {
		body, ok := bodies[v.AttachmentID]
		if !ok {
			return "", fmt.Errorf("source-complete review attachment is missing")
		}
		key := v.QueryID + "\x00" + reviewBodyKey(body.Path, body.IndexedSHA256, body.StartByte, body.EndByte)
		value := resolved[key]
		if value == nil {
			value = &reviewCandidate{AttachmentID: reviewAttachmentID(v.QueryID, body), QueryID: v.QueryID}
			resolved[key] = value
			resolvedBodies[value.AttachmentID] = body
			values = append(values, value)
		}
		attachmentByReference[v.QueryID+"\x00"+v.AttachmentID] = value.AttachmentID
		for _, family := range v.Families {
			addReviewFamily(value, family)
		}
		for _, groupID := range v.RequiredGroupIDs {
			addReviewGroup(value, groupID)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].QueryID != values[j].QueryID {
			return values[i].QueryID < values[j].QueryID
		}
		return values[i].AttachmentID < values[j].AttachmentID
	})
	attachments := make([]ReviewAttachment, 0, len(values))
	for _, candidate := range values {
		body, ok := resolvedBodies[candidate.AttachmentID]
		if !ok {
			return "", fmt.Errorf("source-complete review attachment resolution failed")
		}
		attachments = append(attachments, ReviewAttachment{AttachmentID: candidate.AttachmentID, QueryID: candidate.QueryID, Path: body.Path, IndexedSHA256: body.IndexedSHA256, Language: body.Language, StartByte: body.StartByte, EndByte: body.EndByte, Kind: body.Kind, Symbol: body.Symbol, QualifiedSymbol: body.QualifiedSymbol, Signature: body.Signature, Body: body.Body})
	}
	for _, candidate := range values {
		sort.Strings(candidate.Families)
		sort.Strings(candidate.RequiredGroupIDs)
	}
	relations := make([]ReviewRelationEvidence, 0)
	addRelation := func(query, family, source, target string, ids []string, kind, direction, tier, role string, count int) {
		if source == "" || target == "" || len(ids) == 0 || kind == "" || direction == "" || tier == "" || role == "" {
			return
		}
		ids = uniqueSortedReviewIDs(ids)
		// Preserve the full canonical relation identity. Delivery family remains
		// non-blind provenance; direction/role prevent a reverse or role variant
		// from silently coalescing with the same endpoints.
		identity := query + "\x00" + family + "\x00" + source + "\x00" + strings.Join(ids, "\x00") + "\x00" + kind + "\x00" + direction + "\x00" + tier + "\x00" + role + "\x00" + target
		sum := sha256.Sum256([]byte(identity))
		relations = append(relations, ReviewRelationEvidence{AttachmentID: hex.EncodeToString(sum[:]), QueryID: query, DeliveryFamily: family, SourceAttachmentID: source, TargetAttachmentID: target, RelationIDs: append([]string(nil), ids...), RelationKind: kind, Direction: direction, StructuralTier: tier, Role: role, OccurrenceCount: count})
	}
	for _, row := range allFeatures {
		addRelation(row.QueryID, "semantic", attachmentByReference[row.QueryID+"\x00"+row.AnchorParentID], attachmentByReference[row.QueryID+"\x00"+row.EndpointParentID], row.SupportingRelationIDs, string(row.RelationKind), string(row.Direction), string(row.StructuralTier), "ENDPOINT", len(row.SupportingViews))
	}
	for _, row := range allHints {
		sourceKey := reviewBodyKey(row.SourcePath, row.SourceSHA256, row.SourceStartByte, row.SourceEndByte)
		targetKey := reviewBodyKey(row.TargetPath, row.TargetSHA256, row.TargetStartByte, row.TargetEndByte)
		addRelation(row.QueryID, "hint", attachmentByReference[row.QueryID+"\x00"+sourceKey], attachmentByReference[row.QueryID+"\x00"+targetKey], row.SupportingRelationIDs, string(row.Kind), string(row.Direction), string(row.StructuralTier), "HINT", row.OccurrenceCount)
	}
	for _, row := range allClosures {
		addRelation(row.QueryID, "closure", attachmentByReference[row.QueryID+"\x00"+row.PrimaryParentID], attachmentByReference[row.QueryID+"\x00"+row.TargetParentID], []string{row.RelationID}, "TYPE_REF", "NOT_RECORDED_IN_STAGE_A", string(row.StructuralTier), row.RoleClass, 1)
	}
	sort.Slice(relations, func(i, j int) bool { return relations[i].AttachmentID < relations[j].AttachmentID })
	sort.Slice(packetQueries, func(i, j int) bool { return packetQueries[i].Packet.QueryID < packetQueries[j].Packet.QueryID })
	prepared := reviewPrepared{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_prepared.v1", Policy: ReviewPolicyID, Contract: request.Contract, SemanticStatus: ReviewSemanticStatus, ClosureCells: reviewClosureCells(), HintCells: reviewHintCells(), Universe: attachments, Candidates: dereferenceCandidates(values), Queries: packetQueries, Relations: relations}
	var universeErr error
	prepared.UniverseDigest, universeErr = canonicalReviewUniverseHash(prepared)
	if universeErr != nil {
		return "", universeErr
	}
	for id := range queryIDs {
		for _, cell := range prepared.ClosureCells {
			prepared.Emissions = append(prepared.Emissions, reviewEmission{QueryID: id, Cell: cell, AttachmentIDs: closureEmissionIDs(id, cell, allClosures, attachmentByReference)})
		}
		for _, cell := range prepared.HintCells {
			prepared.Emissions = append(prepared.Emissions, reviewEmission{QueryID: id, Cell: cell, AttachmentIDs: hintEmissionIDs(id, cell, allHints, attachmentByReference)})
		}
	}
	sort.Slice(prepared.Emissions, func(i, j int) bool {
		if prepared.Emissions[i].QueryID != prepared.Emissions[j].QueryID {
			return prepared.Emissions[i].QueryID < prepared.Emissions[j].QueryID
		}
		if prepared.Emissions[i].Cell.Family != prepared.Emissions[j].Cell.Family {
			return prepared.Emissions[i].Cell.Family < prepared.Emissions[j].Cell.Family
		}
		if prepared.Emissions[i].Cell.Count != prepared.Emissions[j].Cell.Count {
			return prepared.Emissions[i].Cell.Count < prepared.Emissions[j].Cell.Count
		}
		return prepared.Emissions[i].Cell.Bytes < prepared.Emissions[j].Cell.Bytes
	})
	for i := range prepared.Emissions {
		family := prepared.Emissions[i].Cell.Family
		for _, relation := range relations {
			if relation.QueryID != prepared.Emissions[i].QueryID || relation.DeliveryFamily != family {
				continue
			}
			for _, parentID := range prepared.Emissions[i].AttachmentIDs {
				if relation.TargetAttachmentID == parentID {
					prepared.Emissions[i].RelationAttachmentIDs = append(prepared.Emissions[i].RelationAttachmentIDs, relation.AttachmentID)
				}
			}
		}
		prepared.Emissions[i].RelationAttachmentIDs = uniqueSortedReviewIDs(prepared.Emissions[i].RelationAttachmentIDs)
	}
	if !sameReviewEmissionControls(emissionFreeze.Controls, reviewEmissionControls(allClosures, allHints, queryIDs)) {
		return "", fmt.Errorf("prepared topology does not match immutable pre-label emission controls")
	}
	digest, err := canonicalReviewPreparedHash(prepared)
	if err != nil {
		return "", err
	}
	prepared.Digest = digest
	if err := writeReviewPrepared(request.OutputDir, prepared, attachments); err != nil {
		return "", err
	}
	return digest, nil
}

func samePrelabelQueries(frozen []ReviewEmissionQuery, topology []reviewQueryRecord) bool {
	if len(frozen) != len(topology) {
		return false
	}
	values := map[string]ReviewEmissionQuery{}
	for _, query := range frozen {
		if _, exists := values[query.QueryID]; exists {
			return false
		}
		values[query.QueryID] = query
	}
	for _, query := range topology {
		frozenQuery, ok := values[query.Packet.QueryID]
		if !ok || frozenQuery.Text != query.Packet.Text || frozenQuery.Language != query.Packet.Language || frozenQuery.AnswerMode != query.Packet.AnswerMode || !sameReviewStringList(frozenQuery.ProtectedTop5, query.ProtectedTop5) {
			return false
		}
	}
	return true
}
func sameReviewStringList(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// PrepareReviewEmissions is Stage E's first operation. It reads only the
// immutable Stage-A outputs and source-free query identity/protected primary
// metadata, writes the frozen 25-cell control universe, and never accepts
// labels or draft required groups.
func PrepareReviewEmissions(request ReviewEmissionPrepareRequest) (string, error) {
	if validateReviewContract(request.Contract) != nil || request.OutputDir == "" || len(request.Completions) != 3 {
		return "", fmt.Errorf("invalid pre-label emission request")
	}
	seenMembers, queryIDs := map[string]bool{}, map[string]bool{}
	featuresTotal, hintsTotal, closuresTotal := 0, 0, 0
	allClosures, allHints := []closureCandidate{}, []relationHint{}
	queries := []ReviewEmissionQuery{}
	for _, input := range request.Completions {
		features, hints, closures, traces, err := loadReviewCompletion(ReviewCompletionInput{Directory: input.Directory})
		if err != nil {
			return "", err
		}
		if err := validateReviewMember(input.Directory, request.Contract.Members, len(traces), seenMembers); err != nil {
			return "", err
		}
		if len(input.Queries) != len(traces) {
			return "", fmt.Errorf("pre-label query cardinality mismatch")
		}
		traceSet := map[string]bool{}
		for _, id := range traces {
			traceSet[id] = true
		}
		for _, q := range input.Queries {
			if q.QueryID == "" || q.Text == "" || q.Language == "" || q.AnswerMode == "" || len(q.ProtectedTop5) != ProtectedPrimaryK || !traceSet[q.QueryID] || queryIDs[q.QueryID] {
				return "", fmt.Errorf("invalid pre-label query")
			}
			queryIDs[q.QueryID] = true
			queries = append(queries, q)
		}
		featuresTotal += len(features)
		hintsTotal += len(hints)
		closuresTotal += len(closures)
		allHints = append(allHints, hints...)
		allClosures = append(allClosures, closures...)
	}
	if len(seenMembers) != 3 || len(queryIDs) != request.Contract.QueryCount || featuresTotal != request.Contract.Endpoints || hintsTotal != request.Contract.Hints || closuresTotal != request.Contract.Closures {
		return "", fmt.Errorf("pre-label immutable series totals mismatch")
	}
	sort.Slice(queries, func(i, j int) bool { return queries[i].QueryID < queries[j].QueryID })
	freeze := reviewEmissionFreeze{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_emissions_prelabels.v1", Contract: request.Contract, Queries: queries, Controls: reviewEmissionControls(allClosures, allHints, queryIDs)}
	digest, err := canonicalReviewEmissionFreezeHash(freeze)
	if err != nil {
		return "", err
	}
	freeze.Digest = digest
	if err = writeReviewJSON(filepath.Join(request.OutputDir, "emissions-prelabels.json"), freeze); err != nil {
		return "", err
	}
	return digest, nil
}
func reviewEmissionControls(closures []closureCandidate, hints []relationHint, queryIDs map[string]bool) []reviewEmissionControl {
	queries := make([]string, 0, len(queryIDs))
	for q := range queryIDs {
		queries = append(queries, q)
	}
	sort.Strings(queries)
	out := []reviewEmissionControl{}
	for _, q := range queries {
		for _, cell := range reviewClosureCells() {
			c := reviewEmissionControl{QueryID: q, Cell: cell, OmissionCounts: map[string]int{}}
			for _, r := range closures {
				if r.QueryID != q {
					continue
				}
				if r.OmissionReason != "" {
					c.OmissionCounts[r.OmissionReason]++
					continue
				}
				c.CandidateCount++
				if !intMember(cell.Count, r.RequestCountBudgetEligible) {
					c.OmissionCounts["COUNT_CAP"]++
					continue
				}
				if !intMember(cell.Bytes, r.ByteBudgetEligible) {
					c.OmissionCounts["BYTE_CAP"]++
					continue
				}
				c.EmittedCount++
				c.ActualBytes += r.BodyBytes
			}
			out = append(out, c)
		}
		for _, cell := range reviewHintCells() {
			c := reviewEmissionControl{QueryID: q, Cell: cell, OmissionCounts: map[string]int{}}
			for _, r := range hints {
				if r.QueryID != q {
					continue
				}
				if r.OmissionStatus != "" && r.OmissionStatus != "CANDIDATE" {
					c.OmissionCounts[r.OmissionStatus]++
					continue
				}
				c.CandidateCount++
				if !intMember(cell.Count, r.CountBudgetEligible) {
					c.OmissionCounts["COUNT_CAP"]++
					continue
				}
				if !intMember(cell.Bytes, r.ByteBudgetEligible) {
					c.OmissionCounts["BYTE_CAP"]++
					continue
				}
				c.EmittedCount++
				c.ActualBytes += r.SerializedBytes
			}
			out = append(out, c)
		}
	}
	return out
}
func canonicalReviewEmissionFreezeHash(f reviewEmissionFreeze) (string, error) {
	d := f.Digest
	f.Digest = ""
	v, err := canonicalHash(f)
	f.Digest = d
	return v, err
}
func readReviewEmissionFreeze(dir string, contract ReviewSeriesContract) (reviewEmissionFreeze, error) {
	var f reviewEmissionFreeze
	if err := readReviewJSON(filepath.Join(dir, "emissions-prelabels.json"), &f); err != nil {
		return f, err
	}
	d, err := canonicalReviewEmissionFreezeHash(f)
	if err != nil || !validDigest(f.Digest) || d != f.Digest || f.SchemaVersion != 1 || f.Kind != "cidx.relation_calibration.review_emissions_prelabels.v1" {
		return f, fmt.Errorf("invalid pre-label emission digest")
	}
	want, err := canonicalHash(contract)
	got, hashErr := canonicalHash(f.Contract)
	if err != nil || hashErr != nil || want != got || len(f.Queries) != contract.QueryCount || len(f.Controls) != contract.QueryCount*25 {
		return f, fmt.Errorf("pre-label emission contract mismatch")
	}
	return f, nil
}
func sameReviewEmissionControls(a, b []reviewEmissionControl) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].QueryID != b[i].QueryID || a[i].Cell != b[i].Cell || a[i].CandidateCount != b[i].CandidateCount || a[i].EmittedCount != b[i].EmittedCount || a[i].ActualBytes != b[i].ActualBytes || len(a[i].OmissionCounts) != len(b[i].OmissionCounts) {
			return false
		}
		for k, v := range a[i].OmissionCounts {
			if b[i].OmissionCounts[k] != v {
				return false
			}
		}
	}
	return true
}

func addReviewCandidate(values map[string]*reviewCandidate, query, parent, family string) {
	if query == "" || parent == "" {
		return
	}
	key := query + "\x00" + parent
	value := values[key]
	if value == nil {
		value = &reviewCandidate{AttachmentID: parent, QueryID: query}
		values[key] = value
	}
	addReviewFamily(value, family)
}
func addReviewCandidateGroup(values map[string]*reviewCandidate, query, parent, family, groupID string) {
	addReviewCandidate(values, query, parent, family)
	if value := values[query+"\x00"+parent]; value != nil {
		addReviewGroup(value, groupID)
	}
}
func addReviewFamily(value *reviewCandidate, family string) {
	for _, existing := range value.Families {
		if existing == family {
			return
		}
	}
	value.Families = append(value.Families, family)
}
func addReviewGroup(value *reviewCandidate, group string) {
	if group == "" {
		return
	}
	for _, existing := range value.RequiredGroupIDs {
		if existing == group {
			return
		}
	}
	value.RequiredGroupIDs = append(value.RequiredGroupIDs, group)
}
func dereferenceCandidates(values []*reviewCandidate) []reviewCandidate {
	result := make([]reviewCandidate, len(values))
	for i, v := range values {
		result[i] = *v
	}
	return result
}
func reviewBodyKey(path, sha string, start, end int) string {
	return fmt.Sprintf("%s\x00%s\x00%d\x00%d", path, sha, start, end)
}
func validateReviewBody(sourceRoot string, body ReviewSourceBody) error {
	if body.Path == "" || filepath.IsAbs(body.Path) || filepath.Clean(body.Path) != body.Path || strings.HasPrefix(filepath.ToSlash(body.Path), "../") || !validDigest(body.IndexedSHA256) || body.StartByte < 0 || body.EndByte <= body.StartByte || len([]byte(body.Body)) != body.EndByte-body.StartByte {
		return fmt.Errorf("invalid review source body")
	}
	if body.ParentID != "" {
		if body.Language == "" || body.Kind == "" || body.QualifiedSymbol == "" {
			return fmt.Errorf("parent-bound review source body lacks identity fields")
		}
		expected := ParentID(Parent{Path: body.Path, IndexedSHA256: body.IndexedSHA256, Language: body.Language, Kind: body.Kind, QualifiedSymbol: body.QualifiedSymbol, StartByte: body.StartByte, EndByte: body.EndByte})
		if body.ParentID != expected {
			return fmt.Errorf("parent-bound review source body identity mismatch")
		}
	}
	if sourceRoot == "" {
		return nil
	}
	root, err := filepath.EvalSymlinks(sourceRoot)
	if err != nil {
		return err
	}
	file, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(body.Path)))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, file)
	if err != nil || rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
		return fmt.Errorf("review source body escapes root")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != body.IndexedSHA256 || body.EndByte > len(data) || string(data[body.StartByte:body.EndByte]) != body.Body {
		return fmt.Errorf("review source body reproof failed")
	}
	return nil
}
func sameReviewSourceBody(a, b ReviewSourceBody) bool {
	return a.Path == b.Path && a.IndexedSHA256 == b.IndexedSHA256 && a.Language == b.Language && a.StartByte == b.StartByte && a.EndByte == b.EndByte && a.Kind == b.Kind && a.Symbol == b.Symbol && a.QualifiedSymbol == b.QualifiedSymbol && a.Signature == b.Signature && a.Body == b.Body
}
func reviewAttachmentID(query string, body ReviewSourceBody) string {
	sum := sha256.Sum256([]byte(query + "\x00" + reviewBodyKey(body.Path, body.IndexedSHA256, body.StartByte, body.EndByte)))
	return hex.EncodeToString(sum[:])
}

func canonicalReviewPreparedHash(prepared reviewPrepared) (string, error) {
	digest := prepared.Digest
	prepared.Digest = ""
	result, err := canonicalHash(prepared)
	prepared.Digest = digest
	return result, err
}
func canonicalReviewUniverseHash(prepared reviewPrepared) (string, error) {
	queries := make([]ReviewPacketQuery, len(prepared.Queries))
	for i := range prepared.Queries {
		queries[i] = prepared.Queries[i].Packet
	}
	sort.Slice(queries, func(i, j int) bool { return queries[i].QueryID < queries[j].QueryID })
	attachments := append([]ReviewAttachment(nil), prepared.Universe...)
	sort.Slice(attachments, func(i, j int) bool { return attachments[i].AttachmentID < attachments[j].AttachmentID })
	relations := append([]ReviewRelationEvidence(nil), prepared.Relations...)
	sort.Slice(relations, func(i, j int) bool { return relations[i].AttachmentID < relations[j].AttachmentID })
	return canonicalHash(reviewUniverseBinding{Attachments: attachments, Queries: queries, Relations: relations})
}

func closureEmissionIDs(query string, cell ReviewBudgetCell, rows []closureCandidate, attachments map[string]string) []string {
	result := []string{}
	for _, row := range rows {
		if row.QueryID != query || row.OmissionReason != "" || !intMember(cell.Count, row.RequestCountBudgetEligible) || !intMember(cell.Bytes, row.ByteBudgetEligible) {
			continue
		}
		if id := attachments[query+"\x00"+row.TargetParentID]; id != "" {
			result = append(result, id)
		}
	}
	return uniqueSortedReviewIDs(result)
}

func hintEmissionIDs(query string, cell ReviewBudgetCell, rows []relationHint, attachments map[string]string) []string {
	result := []string{}
	for _, row := range rows {
		if row.QueryID != query || (row.OmissionStatus != "" && row.OmissionStatus != "CANDIDATE") || !intMember(cell.Count, row.CountBudgetEligible) || !intMember(cell.Bytes, row.ByteBudgetEligible) {
			continue
		}
		key := reviewBodyKey(row.TargetPath, row.TargetSHA256, row.TargetStartByte, row.TargetEndByte)
		if id := attachments[query+"\x00"+key]; id != "" {
			result = append(result, id)
		}
	}
	return uniqueSortedReviewIDs(result)
}

func intMember(value int, values []int) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
func uniqueSortedReviewIDs(values []string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func ValidateReviewPass(preparedDir string, pass ReviewPass) error {
	prepared, packets, err := readReviewPrepared(preparedDir)
	if err != nil {
		return err
	}
	if pass.SchemaVersion != 1 || pass.Kind != "cidx.relation_calibration.review_pass.v1" || pass.PreparedDigest != prepared.Digest || pass.PassID == "" || pass.ReviewerID == "" || pass.ModelFamily == "" || !pass.SourceVerified || pass.BlindnessAttestation != reviewBlindnessAttestation || !validDigest(pass.PacketDigest) {
		return fmt.Errorf("invalid relation review pass")
	}
	packet, ok := packets[pass.PassID]
	if !ok {
		return fmt.Errorf("unknown review packet")
	}
	digest, err := canonicalHash(packet)
	if err != nil || digest != pass.PacketDigest {
		return fmt.Errorf("review packet digest mismatch")
	}
	if len(pass.Grades) != len(packet.Attachments) {
		return fmt.Errorf("review pass grade cardinality mismatch")
	}
	if len(pass.RelationGrades) != len(packet.Relations) {
		return fmt.Errorf("review relation grade cardinality mismatch")
	}
	seen := map[string]bool{}
	for _, grade := range pass.Grades {
		if seen[grade.AttachmentID] || !packetAttachment(packet, grade.AttachmentID) || !validReviewGradeFor(prepared, grade.AttachmentID, grade) {
			return fmt.Errorf("invalid relation review grade")
		}
		seen[grade.AttachmentID] = true
	}
	seen = map[string]bool{}
	for _, grade := range pass.RelationGrades {
		if seen[grade.AttachmentID] || !packetRelation(packet, grade.AttachmentID) || !validReviewGradeFor(prepared, grade.AttachmentID, grade) {
			return fmt.Errorf("invalid relation review grade")
		}
		seen[grade.AttachmentID] = true
	}
	return nil
}
func packetRelation(packet ReviewPacket, id string) bool {
	for _, value := range packet.Relations {
		if value.AttachmentID == id {
			return true
		}
	}
	return false
}
func validReviewGradeFor(prepared reviewPrepared, id string, grade ReviewGrade) bool {
	if strings.TrimSpace(grade.Rationale) == "" || (grade.Grade != 0 && grade.Grade != 1 && grade.Grade != 2) || !reviewGroupIDs(grade.RequiredGroupIDs) || !reviewGroupIDs(grade.HardNegativeGroupIDs) {
		return false
	}
	allowed := reviewAllowedGroupIDs(prepared, id)
	if grade.Grade == 2 {
		if len(grade.RequiredGroupIDs) == 0 || !reviewGroupSubset(grade.RequiredGroupIDs, allowed) {
			return false
		}
	} else if len(grade.RequiredGroupIDs) != 0 {
		return false
	}
	if grade.HardNegative {
		return grade.Grade == 0 && grade.HardNegativeReason != "" && len(grade.HardNegativeGroupIDs) > 0 && reviewGroupSubset(grade.HardNegativeGroupIDs, allowed)
	}
	return grade.HardNegativeReason == "" && len(grade.HardNegativeGroupIDs) == 0
}
func reviewAllowedGroupIDs(prepared reviewPrepared, id string) []string {
	for _, candidate := range prepared.Candidates {
		if candidate.AttachmentID == id {
			return candidate.RequiredGroupIDs
		}
	}
	for _, relation := range prepared.Relations {
		if relation.AttachmentID == id {
			return reviewAllowedGroupIDs(prepared, relation.TargetAttachmentID)
		}
	}
	return nil
}
func reviewGroupSubset(values, allowed []string) bool {
	set := map[string]bool{}
	for _, value := range allowed {
		set[value] = true
	}
	for _, value := range values {
		if !set[value] {
			return false
		}
	}
	return true
}

func FreezeReview(preparedDir, outputDir string, passOne, passTwo ReviewPass, adoption ReviewAdoption) (string, error) {
	frozen, err := reconcileReview(preparedDir, passOne, passTwo, ReviewAdjudications{})
	if err != nil {
		return "", err
	}
	if !validReviewAdoption(adoption, frozen.ReconciledDigest) {
		return "", fmt.Errorf("whole-digest review adoption required")
	}
	adoptionDigest, err := canonicalHash(adoption)
	if err != nil {
		return "", err
	}
	frozen.OwnerAdoptionSHA256 = adoptionDigest
	frozen.Digest = ""
	digest, err := canonicalHash(frozen)
	if err != nil {
		return "", err
	}
	frozen.Digest = digest
	if err := writeReviewJSON(filepath.Join(outputDir, "frozen.json"), frozen); err != nil {
		return "", err
	}
	if err := writeReviewJSON(filepath.Join(outputDir, "owner-adoption.json"), adoption); err != nil {
		return "", err
	}
	return digest, nil
}

// PrepareReviewAdoption writes the exact conservative reconciliation digest
// that the owner must adopt as one whole object. It cannot freeze labels.
func PrepareReviewAdoption(preparedDir, outputDir string, passOne, passTwo ReviewPass) (string, error) {
	frozen, err := reconcileReview(preparedDir, passOne, passTwo, ReviewAdjudications{})
	if err != nil {
		return "", err
	}
	input := ReviewAdoption{SchemaVersion: 1, Kind: "cidx.relation_calibration.owner_adoption.v1", FrozenDigest: frozen.ReconciledDigest, Adopted: false, ProtocolVersion: "owner-adopted-dual-ai-v1", RelevanceAuthority: "OWNER_ADOPTED_DUAL_AI_REVIEW", ReviewValidation: "NO_INDEPENDENT_HUMAN_REVIEW", Overrides: []string{}}
	if err := writeReviewJSON(filepath.Join(outputDir, "owner-adoption-input.json"), input); err != nil {
		return "", err
	}
	return frozen.ReconciledDigest, nil
}

func FreezeReviewWithAdjudications(preparedDir, outputDir string, passOne, passTwo ReviewPass, adjudications ReviewAdjudications, adoption ReviewAdoption) (string, error) {
	frozen, err := reconcileReview(preparedDir, passOne, passTwo, adjudications)
	if err != nil {
		return "", err
	}
	if !validReviewAdoption(adoption, frozen.ReconciledDigest) {
		return "", fmt.Errorf("whole-digest review adoption required")
	}
	adoptionDigest, err := canonicalHash(adoption)
	if err != nil {
		return "", err
	}
	frozen.OwnerAdoptionSHA256 = adoptionDigest
	frozen.Digest = ""
	digest, err := canonicalHash(frozen)
	if err != nil {
		return "", err
	}
	frozen.Digest = digest
	if err := writeReviewJSON(filepath.Join(outputDir, "frozen.json"), frozen); err != nil {
		return "", err
	}
	if err := writeReviewJSON(filepath.Join(outputDir, "owner-adoption.json"), adoption); err != nil {
		return "", err
	}
	if adjudications.SchemaVersion != 0 {
		if err := writeReviewJSON(filepath.Join(outputDir, "adjudications.json"), adjudications); err != nil {
			return "", err
		}
	}
	return digest, nil
}

func reconcileReview(preparedDir string, passOne, passTwo ReviewPass, adjudications ReviewAdjudications) (reviewFrozen, error) {
	prepared, _, err := readReviewPrepared(preparedDir)
	if err != nil {
		return reviewFrozen{}, err
	}
	if err = ValidateReviewPass(preparedDir, passOne); err != nil {
		return reviewFrozen{}, err
	}
	if err = ValidateReviewPass(preparedDir, passTwo); err != nil {
		return reviewFrozen{}, err
	}
	if passOne.PassID == passTwo.PassID {
		return reviewFrozen{}, fmt.Errorf("review passes must be independent")
	}
	if passOne.ModelFamily == passTwo.ModelFamily || passOne.ReviewerID == passTwo.ReviewerID {
		return reviewFrozen{}, fmt.Errorf("review passes must use distinct model families")
	}
	one, two := reviewGradeMap(passOne.Grades), reviewGradeMap(passTwo.Grades)
	oneRelations, twoRelations := reviewGradeMap(passOne.RelationGrades), reviewGradeMap(passTwo.RelationGrades)
	p1, err := canonicalHash(passOne)
	if err != nil {
		return reviewFrozen{}, err
	}
	p2, err := canonicalHash(passTwo)
	if err != nil {
		return reviewFrozen{}, err
	}
	adjudicated := map[string]ReviewAdjudication{}
	if adjudications.SchemaVersion != 0 {
		if adjudications.SchemaVersion != 1 || adjudications.Kind != "cidx.relation_calibration.review_adjudications.v1" || adjudications.PreparedDigest != prepared.Digest || adjudications.PassOneDigest != p1 || adjudications.PassTwoDigest != p2 {
			return reviewFrozen{}, fmt.Errorf("invalid review adjudication binding")
		}
		for _, entry := range adjudications.Entries {
			key := entry.Subject + "\x00" + entry.AttachmentID
			if (entry.Subject != "parent" && entry.Subject != "relation") || entry.AttachmentID == "" || adjudicated[key].AttachmentID != "" || !validReviewAdjudication(prepared, entry) {
				return reviewFrozen{}, fmt.Errorf("invalid review adjudication")
			}
			adjudicated[key] = entry
		}
	}
	reconcile := func(ids []string, leftMap, rightMap map[string]ReviewGrade, subject string) ([]reviewLabel, error) {
		labels := make([]reviewLabel, 0, len(ids))
		for _, id := range ids {
			left, right := leftMap[id], rightMap[id]
			conflict := (left.Grade == 2 && (right.Grade != 2 || !sameReviewGroups(left.RequiredGroupIDs, right.RequiredGroupIDs))) || (right.Grade == 2 && (left.Grade != 2 || !sameReviewGroups(left.RequiredGroupIDs, right.RequiredGroupIDs)))
			if conflict {
				entry, ok := adjudicated[subject+"\x00"+id]
				if !ok {
					return nil, fmt.Errorf("grade-2/group conflict requires explicit adjudication")
				}
				labels = append(labels, reviewLabel{AttachmentID: id, Grade: entry.Grade, HardNegative: entry.HardNegative, GroupIDs: uniqueSortedReviewIDs(entry.RequiredGroupIDs), HardNegativeGroupIDs: uniqueSortedReviewIDs(entry.HardNegativeGroupIDs)})
				continue
			}
			grade, groups := 0, []string{}
			if left.Grade == 2 && right.Grade == 2 {
				grade, groups = 2, append([]string(nil), left.RequiredGroupIDs...)
			} else if left.Grade == 1 && right.Grade == 1 {
				grade = 1
			}
			hn := left.HardNegative && right.HardNegative && left.HardNegativeReason == right.HardNegativeReason && sameReviewGroups(left.HardNegativeGroupIDs, right.HardNegativeGroupIDs)
			hnGroups := []string{}
			if hn {
				hnGroups = append([]string(nil), left.HardNegativeGroupIDs...)
			}
			labels = append(labels, reviewLabel{AttachmentID: id, Grade: grade, HardNegative: hn, GroupIDs: groups, HardNegativeGroupIDs: hnGroups})
		}
		sort.Slice(labels, func(i, j int) bool { return labels[i].AttachmentID < labels[j].AttachmentID })
		return labels, nil
	}
	parentIDs := make([]string, 0, len(prepared.Candidates))
	for _, candidate := range prepared.Candidates {
		parentIDs = append(parentIDs, candidate.AttachmentID)
	}
	relationIDs := make([]string, 0, len(prepared.Relations))
	for _, relation := range prepared.Relations {
		relationIDs = append(relationIDs, relation.AttachmentID)
	}
	parents, err := reconcile(parentIDs, one, two, "parent")
	if err != nil {
		return reviewFrozen{}, err
	}
	relations, err := reconcile(relationIDs, oneRelations, twoRelations, "relation")
	if err != nil {
		return reviewFrozen{}, err
	}
	frozen := reviewFrozen{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_frozen.v1", PreparedDigest: prepared.Digest, PassOneDigest: p1, PassTwoDigest: p2, ParentLabels: parents, RelationLabels: relations}
	base := frozen
	base.ReconciledDigest = ""
	base.OwnerAdoptionSHA256 = ""
	base.Digest = ""
	digest, err := canonicalHash(base)
	if err != nil {
		return reviewFrozen{}, err
	}
	frozen.ReconciledDigest = digest
	return frozen, nil
}

func SelectReview(preparedDir, frozenDir, outputDir string) error {
	prepared, _, err := readReviewPrepared(preparedDir)
	if err != nil {
		return err
	}
	var frozen reviewFrozen
	if err = readReviewJSON(filepath.Join(frozenDir, "frozen.json"), &frozen); err != nil {
		return err
	}
	var adoption ReviewAdoption
	if err = readReviewJSON(filepath.Join(frozenDir, "owner-adoption.json"), &adoption); err != nil {
		return err
	}
	digest := frozen.Digest
	frozen.Digest = ""
	actual, hashErr := canonicalHash(frozen)
	frozen.Digest = digest
	adoptionDigest, adoptionErr := canonicalHash(adoption)
	if hashErr != nil || adoptionErr != nil || !validDigest(digest) || actual != digest || frozen.Kind != "cidx.relation_calibration.review_frozen.v1" || frozen.PreparedDigest != prepared.Digest || len(frozen.ParentLabels) != len(prepared.Candidates) || len(frozen.RelationLabels) != len(prepared.Relations) || !validReviewAdoption(adoption, frozen.ReconciledDigest) || frozen.OwnerAdoptionSHA256 != adoptionDigest {
		return fmt.Errorf("invalid frozen review binding")
	}
	labels := map[string]reviewLabel{}
	for _, label := range frozen.ParentLabels {
		labels[label.AttachmentID] = label
	}
	denominators := map[string]map[string]int{}
	for _, candidate := range prepared.Candidates {
		label, ok := labels[candidate.AttachmentID]
		if !ok {
			return fmt.Errorf("missing frozen review label")
		}
		for _, family := range candidate.Families {
			if denominators[family] == nil {
				denominators[family] = map[string]int{"total": 0, "grade_2": 0, "grade_1": 0, "grade_0": 0, "hard_negative": 0}
			}
			d := denominators[family]
			d["total"]++
			d[fmt.Sprintf("grade_%d", label.Grade)]++
			if label.HardNegative {
				d["hard_negative"]++
			}
		}
	}
	cellDenominators := map[string]map[string]int{}
	for _, emission := range prepared.Emissions {
		key := reviewCellKey(emission.Cell)
		if cellDenominators[key] == nil {
			cellDenominators[key] = map[string]int{"total": 0, "grade_2": 0, "grade_1": 0, "grade_0": 0, "hard_negative": 0}
		}
		for _, attachmentID := range emission.AttachmentIDs {
			label, ok := labels[attachmentID]
			if !ok {
				return fmt.Errorf("emission references an unlabelled attachment")
			}
			d := cellDenominators[key]
			d["total"]++
			d[fmt.Sprintf("grade_%d", label.Grade)]++
			if label.HardNegative {
				d["hard_negative"]++
			}
		}
	}
	relationLabels := map[string]reviewLabel{}
	for _, label := range frozen.RelationLabels {
		relationLabels[label.AttachmentID] = label
	}
	relationDenominators := map[string]int{"total": 0, "grade_2": 0, "grade_1": 0, "grade_0": 0, "hard_negative": 0}
	for _, label := range relationLabels {
		relationDenominators["total"]++
		relationDenominators[fmt.Sprintf("grade_%d", label.Grade)]++
		if label.HardNegative {
			relationDenominators["hard_negative"]++
		}
	}
	result := map[string]any{"schema_version": 1, "kind": "cidx.relation_calibration.review_selection.v1", "policy": ReviewPolicyID, "prepared_digest": prepared.Digest, "frozen_digest": frozen.Digest, "owner_adoption_sha256": frozen.OwnerAdoptionSHA256, "semantic_status": ReviewSemanticStatus, "candidate_denominators": denominators, "relation_denominators": relationDenominators, "cell_emission_denominators": cellDenominators, "selection": "NO_POLICY_SELECTED_NO_PREDECLARED_DECISION_RULE"}
	return writeReviewJSON(filepath.Join(outputDir, "selection.json"), result)
}

func loadReviewCompletion(input ReviewCompletionInput) ([]semanticEndpointFeature, []relationHint, []closureCandidate, []string, error) {
	if input.Directory == "" {
		return nil, nil, nil, nil, fmt.Errorf("missing completion directory")
	}
	expected := []string{"run-manifest.json", "input-artifact-binding.json", "semantic-parent-scores.jsonl", "relation-endpoint-features.jsonl", "contract-closure-candidates.jsonl", "relation-hints.jsonl", "semantic-admission-results.jsonl", "closure-package-results.jsonl", "per-query-relation-trace.jsonl", "aggregate-relation-metrics.json", "cohort-language-report.json", "first-loss-report.json", "report.md"}
	if err := verifyChecksums(input.Directory, expected); err != nil {
		return nil, nil, nil, nil, err
	}
	var manifest struct {
		Kind         string `json:"kind"`
		LabelLoading string `json:"label_loading"`
	}
	if err := readReviewJSON(filepath.Join(input.Directory, "run-manifest.json"), &manifest); err != nil {
		return nil, nil, nil, nil, err
	}
	if manifest.Kind != ReviewAcceptedCompletionKind || manifest.LabelLoading != "LABEL_FIELDS_NOT_DECODED_STAGE_A" {
		return nil, nil, nil, nil, fmt.Errorf("completion directory is not accepted review v2")
	}
	features := []semanticEndpointFeature{}
	hints := []relationHint{}
	closures := []closureCandidate{}
	traces := []struct {
		QueryID string `json:"query_id"`
	}{}
	if err := readReviewJSONL(filepath.Join(input.Directory, "relation-endpoint-features.jsonl"), &features); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := readReviewJSONL(filepath.Join(input.Directory, "relation-hints.jsonl"), &hints); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := readReviewJSONL(filepath.Join(input.Directory, "contract-closure-candidates.jsonl"), &closures); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := readReviewJSONL(filepath.Join(input.Directory, "per-query-relation-trace.jsonl"), &traces); err != nil {
		return nil, nil, nil, nil, err
	}
	ids := map[string]bool{}
	for _, trace := range traces {
		if trace.QueryID == "" || ids[trace.QueryID] {
			return nil, nil, nil, nil, fmt.Errorf("invalid completion query trace")
		}
		ids[trace.QueryID] = true
	}
	for _, v := range features {
		if !ids[v.QueryID] {
			return nil, nil, nil, nil, fmt.Errorf("endpoint feature lacks completion query trace")
		}
	}
	for _, v := range hints {
		if !ids[v.QueryID] {
			return nil, nil, nil, nil, fmt.Errorf("relation hint lacks completion query trace")
		}
	}
	for _, v := range closures {
		if !ids[v.QueryID] {
			return nil, nil, nil, nil, fmt.Errorf("closure candidate lacks completion query trace")
		}
	}
	result := make([]string, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	return features, hints, closures, result, nil
}
func validateReviewContract(c ReviewSeriesContract) error {
	if !exactReviewSeriesContract(c) {
		return fmt.Errorf("relation review contract is not the canonical tracked three-member binding")
	}
	if c.SchemaVersion != 1 || c.Kind != "cidx.relation_calibration.review_series.v1" || c.SeriesID != "relation-calibration-review-series-v1" || c.AcceptedRepeat != ReviewAcceptedCompletionRepeat || c.QueryCount != 40 || c.Endpoints != 250 || c.Hints != 289 || c.Closures != 576 || len(c.Members) != 3 {
		return fmt.Errorf("invalid relation review series contract")
	}
	total := 0
	seen := map[string]bool{}
	for _, member := range c.Members {
		if member.CorpusID == "" || !validDigest(member.DatasetSHA256) || !validDigest(member.CompletionArtifactChecksum) || member.QueryCount < 1 || seen[member.CorpusID] {
			return fmt.Errorf("invalid relation review member")
		}
		seen[member.CorpusID] = true
		total += member.QueryCount
	}
	if total != c.QueryCount {
		return fmt.Errorf("invalid relation review member query totals")
	}
	return nil
}

func validateReviewMember(root string, members []ReviewSeriesMember, queryCount int, seen map[string]bool) error {
	if len(members) == 0 {
		return nil
	}
	var manifest struct {
		CorpusID      string `json:"corpus_id"`
		DatasetSHA256 string `json:"dataset_sha256"`
	}
	if err := readReviewJSON(filepath.Join(root, "run-manifest.json"), &manifest); err != nil {
		return err
	}
	digest, err := fileSHA256(filepath.Join(root, "artifact-checksums.json"))
	if err != nil {
		return err
	}
	for _, member := range members {
		if member.CorpusID == manifest.CorpusID && member.DatasetSHA256 == manifest.DatasetSHA256 && member.CompletionArtifactChecksum == digest && member.QueryCount == queryCount {
			if seen[member.CorpusID] {
				return fmt.Errorf("duplicate review completion member")
			}
			seen[member.CorpusID] = true
			return nil
		}
	}
	return fmt.Errorf("completion directory does not match review series member")
}

func validateReviewQueries(queries []ReviewQuery, traceIDs []string) error {
	if len(queries) != len(traceIDs) {
		return fmt.Errorf("review query topology cardinality mismatch")
	}
	wanted, seen := map[string]bool{}, map[string]bool{}
	for _, id := range traceIDs {
		wanted[id] = true
	}
	for _, query := range queries {
		if query.QueryID == "" || query.Text == "" || query.Language == "" || query.AnswerMode == "" || !wanted[query.QueryID] || seen[query.QueryID] || len(query.ProtectedTop5) != ProtectedPrimaryK || !reviewGroupTopology(query.RequiredGroups) {
			return fmt.Errorf("invalid review query topology")
		}
		seen[query.QueryID] = true
	}
	return nil
}
func reviewGroupTopology(groups []ReviewRequiredGroup) bool {
	seen := map[string]bool{}
	for _, group := range groups {
		if group.ID == "" || seen[group.ID] || len(group.SourceParentIDs) == 0 {
			return false
		}
		seen[group.ID] = true
		parents := map[string]bool{}
		for _, parent := range group.SourceParentIDs {
			if parent == "" || parents[parent] {
				return false
			}
			parents[parent] = true
		}
	}
	return true
}
func writeReviewPrepared(root string, prepared reviewPrepared, attachments []ReviewAttachment) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	if err := writeReviewJSON(filepath.Join(root, "prepared.json"), prepared); err != nil {
		return err
	}
	first := shuffleReviewAttachments(attachments, prepared.Digest, "pass-1")
	second := shuffleReviewAttachments(attachments, prepared.Digest, "pass-2")
	if len(second) > 1 && sameReviewAttachmentOrder(first, second) {
		second = append(second[1:], second[0])
	}
	for _, value := range []struct {
		pass        string
		attachments []ReviewAttachment
	}{{"pass-1", first}, {"pass-2", second}} {
		relations := shuffleReviewRelations(prepared.Relations, prepared.Digest, value.pass)
		queries := make([]ReviewPacketQuery, len(prepared.Queries))
		for i := range prepared.Queries {
			queries[i] = prepared.Queries[i].Packet
		}
		packet := ReviewPacket{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_packet.v1", Policy: ReviewPolicyID, PreparedDigest: prepared.Digest, CanonicalUniverseDigest: prepared.UniverseDigest, PassID: value.pass, Attachments: value.attachments, Queries: queries, Relations: relations}
		if err := writeReviewJSON(filepath.Join(root, value.pass+"-packet.json"), packet); err != nil {
			return err
		}
	}
	return nil
}
func readReviewPrepared(root string) (reviewPrepared, map[string]ReviewPacket, error) {
	var prepared reviewPrepared
	if err := readReviewJSON(filepath.Join(root, "prepared.json"), &prepared); err != nil {
		return reviewPrepared{}, nil, err
	}
	digest := prepared.Digest
	prepared.Digest = ""
	actual, err := canonicalReviewPreparedHash(prepared)
	prepared.Digest = digest
	if err != nil || !validDigest(digest) || actual != digest || prepared.Kind != "cidx.relation_calibration.review_prepared.v1" || validateReviewContract(prepared.Contract) != nil || len(prepared.ClosureCells) != 9 || len(prepared.HintCells) != 16 || !validReviewEmissionSet(prepared) || prepared.SemanticStatus != ReviewSemanticStatus || !validReviewUniverse(prepared) {
		return reviewPrepared{}, nil, fmt.Errorf("invalid prepared review")
	}
	packets := map[string]ReviewPacket{}
	for _, pass := range []string{"pass-1", "pass-2"} {
		var packet ReviewPacket
		if err := readReviewJSON(filepath.Join(root, pass+"-packet.json"), &packet); err != nil {
			return reviewPrepared{}, nil, err
		}
		if packet.SchemaVersion != 1 || packet.Kind != "cidx.relation_calibration.review_packet.v1" || packet.Policy != ReviewPolicyID || packet.PreparedDigest != prepared.Digest || packet.CanonicalUniverseDigest != prepared.UniverseDigest || packet.PassID != pass || len(packet.Attachments) != len(prepared.Universe) || !sameReviewAttachmentSet(packet.Attachments, prepared.Universe) || !sameReviewPacketQueries(packet.Queries, prepared.Queries) {
			return reviewPrepared{}, nil, fmt.Errorf("invalid review packet")
		}
		if len(packet.Relations) != len(prepared.Relations) || !sameReviewPacketRelationSet(packet.Relations, prepared.Relations) {
			return reviewPrepared{}, nil, fmt.Errorf("invalid relation review packet")
		}
		for _, relation := range packet.Relations {
			if !validReviewPacketRelation(relation) {
				return reviewPrepared{}, nil, fmt.Errorf("invalid relation review evidence")
			}
		}
		packets[pass] = packet
	}
	if !sameReviewAttachmentSet(packets["pass-1"].Attachments, packets["pass-2"].Attachments) || (len(prepared.Candidates) > 1 && sameReviewAttachmentOrder(packets["pass-1"].Attachments, packets["pass-2"].Attachments)) || (len(prepared.Relations) > 1 && sameReviewPacketRelationOrder(packets["pass-1"].Relations, packets["pass-2"].Relations)) {
		return reviewPrepared{}, nil, fmt.Errorf("review packet shuffle binding failed")
	}
	return prepared, packets, nil
}
func sameReviewPacketQueries(packet []ReviewPacketQuery, prepared []reviewQueryRecord) bool {
	if len(packet) != len(prepared) {
		return false
	}
	expected := map[string]ReviewPacketQuery{}
	for _, query := range prepared {
		expected[query.Packet.QueryID] = query.Packet
	}
	seen := map[string]bool{}
	for _, query := range packet {
		want, ok := expected[query.QueryID]
		if !ok || seen[query.QueryID] || want.Text != query.Text || want.Language != query.Language || want.AnswerMode != query.AnswerMode || !sameReviewGroups(want.UnadoptedRequiredGroupIDs, query.UnadoptedRequiredGroupIDs) {
			return false
		}
		seen[query.QueryID] = true
	}
	return true
}
func validReviewPacketRelation(v ReviewPacketRelation) bool {
	return v.AttachmentID != "" && v.QueryID != "" && v.SourceAttachmentID != "" && v.TargetAttachmentID != "" && len(v.RelationIDs) > 0 && v.RelationKind != "" && v.Direction != "" && v.StructuralTier != "" && v.Role != "" && v.OccurrenceCount > 0
}
func validReviewUniverse(prepared reviewPrepared) bool {
	digest, err := canonicalReviewUniverseHash(prepared)
	if err != nil || digest != prepared.UniverseDigest || len(prepared.Universe) != len(prepared.Candidates) {
		return false
	}
	seen := map[string]bool{}
	for _, attachment := range prepared.Universe {
		if attachment.AttachmentID == "" || seen[attachment.AttachmentID] || len([]byte(attachment.Body)) != attachment.EndByte-attachment.StartByte {
			return false
		}
		seen[attachment.AttachmentID] = true
	}
	for _, candidate := range prepared.Candidates {
		if candidate.QueryID == "" || !seen[candidate.AttachmentID] {
			return false
		}
	}
	for _, relation := range prepared.Relations {
		if !validReviewRelation(relation) || !seen[relation.SourceAttachmentID] || !seen[relation.TargetAttachmentID] {
			return false
		}
	}
	return true
}
func validReviewRelation(v ReviewRelationEvidence) bool {
	return v.AttachmentID != "" && v.QueryID != "" && v.DeliveryFamily != "" && v.SourceAttachmentID != "" && v.TargetAttachmentID != "" && len(v.RelationIDs) > 0 && v.RelationKind != "" && v.RelationKind != "UNSPECIFIED" && v.Direction != "" && v.Direction != "UNSPECIFIED" && v.StructuralTier != "" && v.StructuralTier != "UNSPECIFIED" && v.Role != "" && v.Role != "UNSPECIFIED" && v.OccurrenceCount > 0
}
func sameReviewAttachmentSet(a, b []ReviewAttachment) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]ReviewAttachment{}
	for _, v := range a {
		if _, exists := seen[v.AttachmentID]; exists {
			return false
		}
		seen[v.AttachmentID] = v
	}
	for _, v := range b {
		value, exists := seen[v.AttachmentID]
		if !exists || value != v {
			return false
		}
	}
	return true
}
func shuffleReviewRelations(values []ReviewRelationEvidence, seed, pass string) []ReviewPacketRelation {
	result := make([]ReviewPacketRelation, len(values))
	for i, value := range values {
		result[i] = ReviewPacketRelation{AttachmentID: value.AttachmentID, QueryID: value.QueryID, SourceAttachmentID: value.SourceAttachmentID, TargetAttachmentID: value.TargetAttachmentID, RelationIDs: append([]string(nil), value.RelationIDs...), RelationKind: value.RelationKind, Direction: value.Direction, StructuralTier: value.StructuralTier, Role: value.Role, OccurrenceCount: value.OccurrenceCount}
	}
	sort.Slice(result, func(i, j int) bool {
		return reviewShuffleKey(seed, pass, result[i].AttachmentID) < reviewShuffleKey(seed, pass, result[j].AttachmentID)
	})
	return result
}
func sameReviewPacketRelationOrder(a, b []ReviewPacketRelation) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].AttachmentID != b[i].AttachmentID {
			return false
		}
	}
	return true
}
func sameReviewPacketRelationSet(packet []ReviewPacketRelation, prepared []ReviewRelationEvidence) bool {
	if len(packet) != len(prepared) {
		return false
	}
	values := map[string]ReviewPacketRelation{}
	for _, value := range packet {
		if _, ok := values[value.AttachmentID]; ok {
			return false
		}
		values[value.AttachmentID] = value
	}
	for _, value := range prepared {
		candidate, ok := values[value.AttachmentID]
		if !ok || candidate.QueryID != value.QueryID || candidate.SourceAttachmentID != value.SourceAttachmentID || candidate.TargetAttachmentID != value.TargetAttachmentID || !sameReviewGroups(candidate.RelationIDs, value.RelationIDs) || candidate.RelationKind != value.RelationKind || candidate.Direction != value.Direction || candidate.StructuralTier != value.StructuralTier || candidate.Role != value.Role || candidate.OccurrenceCount != value.OccurrenceCount {
			return false
		}
	}
	return true
}
func sameReviewAttachmentOrder(a, b []ReviewAttachment) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].AttachmentID != b[i].AttachmentID {
			return false
		}
	}
	return true
}

func validReviewEmissionSet(prepared reviewPrepared) bool {
	cells := append(append([]ReviewBudgetCell{}, reviewClosureCells()...), reviewHintCells()...)
	wanted, queries, seen := map[string]bool{}, map[string]int{}, map[string]bool{}
	for _, cell := range cells {
		wanted[reviewCellKey(cell)] = true
	}
	if len(prepared.Emissions) != prepared.Contract.QueryCount*len(cells) {
		return false
	}
	for _, emission := range prepared.Emissions {
		if emission.QueryID == "" || !wanted[reviewCellKey(emission.Cell)] {
			return false
		}
		key := emission.QueryID + "\x00" + reviewCellKey(emission.Cell)
		if seen[key] {
			return false
		}
		seen[key] = true
		queries[emission.QueryID]++
	}
	if len(queries) != prepared.Contract.QueryCount {
		return false
	}
	for _, count := range queries {
		if count != len(cells) {
			return false
		}
	}
	return true
}
func reviewCellKey(cell ReviewBudgetCell) string {
	return fmt.Sprintf("%s\x00%d\x00%d", cell.Family, cell.Count, cell.Bytes)
}
func shuffleReviewAttachments(values []ReviewAttachment, seed, pass string) []ReviewAttachment {
	result := append([]ReviewAttachment(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		return reviewShuffleKey(seed, pass, result[i].AttachmentID) < reviewShuffleKey(seed, pass, result[j].AttachmentID)
	})
	return result
}
func reviewShuffleKey(seed, pass, id string) string {
	sum := sha256.Sum256([]byte(seed + "\x00" + pass + "\x00" + id))
	return hex.EncodeToString(sum[:])
}
func packetAttachment(packet ReviewPacket, id string) bool {
	for _, a := range packet.Attachments {
		if a.AttachmentID == id {
			return true
		}
	}
	return false
}
func reviewGroupIDs(values []string) bool {
	seen := map[string]bool{}
	for _, v := range values {
		if v == "" || strings.TrimSpace(v) != v || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}
func reviewGradeMap(grades []ReviewGrade) map[string]ReviewGrade {
	result := map[string]ReviewGrade{}
	for _, v := range grades {
		result[v.AttachmentID] = v
	}
	return result
}
func validReviewAdoption(adoption ReviewAdoption, reconciled string) bool {
	return adoption.SchemaVersion == 1 && adoption.Kind == "cidx.relation_calibration.owner_adoption.v1" && adoption.Adopted && adoption.FrozenDigest == reconciled && adoption.ProtocolVersion == "owner-adopted-dual-ai-v1" && adoption.RelevanceAuthority == "OWNER_ADOPTED_DUAL_AI_REVIEW" && adoption.ReviewValidation == "NO_INDEPENDENT_HUMAN_REVIEW" && len(adoption.Overrides) == 0
}
func minReviewGrade(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func unionReviewGroups(a, b []string) []string {
	seen := map[string]bool{}
	for _, v := range append(append([]string{}, a...), b...) {
		seen[v] = true
	}
	result := make([]string, 0, len(seen))
	for v := range seen {
		result = append(result, v)
	}
	sort.Strings(result)
	return result
}
func sameReviewGroups(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	left, right := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(left)
	sort.Strings(right)
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
func validReviewAdjudication(prepared reviewPrepared, entry ReviewAdjudication) bool {
	return validReviewGradeFor(prepared, entry.AttachmentID, ReviewGrade{AttachmentID: entry.AttachmentID, Grade: entry.Grade, RequiredGroupIDs: entry.RequiredGroupIDs, HardNegative: entry.HardNegative, HardNegativeGroupIDs: entry.HardNegativeGroupIDs, HardNegativeReason: entry.HardNegativeReason, Rationale: entry.Rationale})
}
func writeReviewJSON(file string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	return os.WriteFile(file, append(data, '\n'), 0o600)
}
func readReviewJSON(file string, value any) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return strictJSON(data, value)
}
func readReviewJSONL[T any](file string, target *[]T) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var value T
		if err := strictJSON([]byte(line), &value); err != nil {
			return err
		}
		*target = append(*target, value)
	}
	return nil
}
