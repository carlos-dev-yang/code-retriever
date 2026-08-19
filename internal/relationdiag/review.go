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
	PrelabelDigest     string   `json:"prelabel_digest"`
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
	PrelabelDigest string                   `json:"prelabel_digest"`
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
	PrelabelDigest      string        `json:"prelabel_digest"`
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
		features, hintRows, closureRows, ids, completionManifest, err := loadReviewCompletion(input)
		if err != nil {
			return "", err
		}
		if err := validateReviewMember(completionManifest, request.Contract.Members, len(ids), seenMembers); err != nil {
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
	prepared := reviewPrepared{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_prepared.v1", Policy: ReviewPolicyID, Contract: request.Contract, SemanticStatus: ReviewSemanticStatus, ClosureCells: reviewClosureCells(), HintCells: reviewHintCells(), Universe: attachments, Candidates: dereferenceCandidates(values), Queries: packetQueries, Relations: relations, PrelabelDigest: emissionFreeze.Digest}
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
	if err := writeReviewPrepared(request.OutputDir, prepared, attachments, emissionFreeze); err != nil {
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
		features, hints, closures, traces, completionManifest, err := loadReviewCompletion(ReviewCompletionInput{Directory: input.Directory})
		if err != nil {
			return "", err
		}
		if err := validateReviewMember(completionManifest, request.Contract.Members, len(traces), seenMembers); err != nil {
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
	parentGrades := map[string]ReviewGrade{}
	for _, grade := range pass.Grades {
		if seen[grade.AttachmentID] || !packetAttachment(packet, grade.AttachmentID) || !validReviewGradeFor(prepared, grade.AttachmentID, grade) {
			return fmt.Errorf("invalid relation review grade")
		}
		seen[grade.AttachmentID] = true
		parentGrades[grade.AttachmentID] = grade
	}
	seen = map[string]bool{}
	for _, grade := range pass.RelationGrades {
		if seen[grade.AttachmentID] || !packetRelation(packet, grade.AttachmentID) || !validReviewGradeFor(prepared, grade.AttachmentID, grade) || !validReviewRelationGrade(prepared, grade, parentGrades) {
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

// validReviewRelationGrade keeps relation evidence subordinate to the exact
// target parent that it annotates. HN remains an independent grade-0 finding,
// but it cannot elevate a relation above its target parent's grade.
func validReviewRelationGrade(prepared reviewPrepared, grade ReviewGrade, parentGrades map[string]ReviewGrade) bool {
	targetID, ok := reviewRelationTargetAttachmentID(prepared, grade.AttachmentID)
	if !ok {
		return false
	}
	target, ok := parentGrades[targetID]
	if !ok || grade.Grade > target.Grade {
		return false
	}
	return grade.Grade != 2 || (target.Grade == 2 && reviewGroupSubset(grade.RequiredGroupIDs, target.RequiredGroupIDs))
}

func reviewRelationTargetAttachmentID(prepared reviewPrepared, id string) (string, bool) {
	result := ""
	for _, relation := range prepared.Relations {
		if relation.AttachmentID != id {
			continue
		}
		if result != "" || relation.TargetAttachmentID == "" {
			return "", false
		}
		result = relation.TargetAttachmentID
	}
	return result, result != ""
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
		return grade.Grade == 0 && strings.TrimSpace(grade.HardNegativeReason) != "" && len(grade.HardNegativeGroupIDs) > 0 && reviewGroupSubset(grade.HardNegativeGroupIDs, reviewQueryTopologyGroupIDs(prepared, id))
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

// reviewQueryTopologyGroupIDs deliberately differs from reviewAllowedGroupIDs.
// Grade-2 evidence must remain tied to a direct candidate/target truth group,
// while a grade-0 hard negative may be a non-truth attachment that is still
// misleading for one of the query's known required groups.
func reviewQueryTopologyGroupIDs(prepared reviewPrepared, id string) []string {
	queryID, ok := reviewAttachmentQueryID(prepared, id)
	if !ok {
		return nil
	}
	matchedQuery := false
	var allowed []string
	for _, query := range prepared.Queries {
		if query.Packet.QueryID != queryID {
			continue
		}
		if matchedQuery {
			return nil
		}
		matchedQuery = true
		groups := make([]string, 0, len(query.RequiredGroups))
		for _, group := range query.RequiredGroups {
			groups = append(groups, group.ID)
		}
		groups = uniqueSortedReviewIDs(groups)
		if !reviewGroupIDs(groups) || !sameReviewGroups(groups, query.Packet.UnadoptedRequiredGroupIDs) {
			return nil
		}
		allowed = groups
	}
	return allowed
}

// reviewAttachmentQueryID proves the candidate/relation is attached to the
// exact prepared query topology before an HN group can be accepted.
func reviewAttachmentQueryID(prepared reviewPrepared, id string) (string, bool) {
	attachments := map[string]ReviewAttachment{}
	for _, attachment := range prepared.Universe {
		if attachment.AttachmentID == "" || attachments[attachment.AttachmentID].AttachmentID != "" {
			return "", false
		}
		attachments[attachment.AttachmentID] = attachment
	}
	candidateQuery, matchedCandidate := "", false
	for _, candidate := range prepared.Candidates {
		if candidate.AttachmentID != id {
			continue
		}
		if matchedCandidate {
			return "", false
		}
		matchedCandidate = true
		attachment, ok := attachments[id]
		if !ok || attachment.QueryID != candidate.QueryID {
			return "", false
		}
		candidateQuery = candidate.QueryID
	}
	relationQuery, matchedRelation := "", false
	for _, relation := range prepared.Relations {
		if relation.AttachmentID != id {
			continue
		}
		if matchedRelation {
			return "", false
		}
		matchedRelation = true
		source, sourceOK := attachments[relation.SourceAttachmentID]
		target, targetOK := attachments[relation.TargetAttachmentID]
		if !sourceOK || !targetOK || source.QueryID != relation.QueryID || target.QueryID != relation.QueryID {
			return "", false
		}
		relationQuery = relation.QueryID
	}
	if matchedCandidate == matchedRelation {
		return "", false
	}
	if matchedCandidate {
		return candidateQuery, true
	}
	return relationQuery, true
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
	if !validReviewAdoption(adoption, frozen.ReconciledDigest, frozen.PrelabelDigest) {
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
	input := ReviewAdoption{SchemaVersion: 1, Kind: "cidx.relation_calibration.owner_adoption.v1", FrozenDigest: frozen.ReconciledDigest, PrelabelDigest: frozen.PrelabelDigest, Adopted: false, ProtocolVersion: "owner-adopted-dual-ai-v1", RelevanceAuthority: "OWNER_ADOPTED_DUAL_AI_REVIEW", ReviewValidation: "NO_INDEPENDENT_HUMAN_REVIEW", Overrides: []string{}}
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
	if !validReviewAdoption(adoption, frozen.ReconciledDigest, frozen.PrelabelDigest) {
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
	if !validFrozenReviewRelationGrades(prepared, labelsByReviewAttachmentID(parents), labelsByReviewAttachmentID(relations)) {
		return reviewFrozen{}, fmt.Errorf("reconciled relation grade exceeds target parent")
	}
	frozen := reviewFrozen{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_frozen.v1", PreparedDigest: prepared.Digest, PassOneDigest: p1, PassTwoDigest: p2, PrelabelDigest: prepared.PrelabelDigest, ParentLabels: parents, RelationLabels: relations}
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

// reviewPolicyQueryCell is one of the fixed 40 x 25 Stage F observations.
// All source-sensitive candidate construction has already happened in Stage E;
// this row only joins the adopted labels to an immutable pre-label control.
type reviewPolicyQueryCell struct {
	SchemaVersion       int              `json:"schema_version"`
	Kind                string           `json:"kind"`
	PreparedDigest      string           `json:"prepared_digest"`
	FrozenDigest        string           `json:"frozen_digest"`
	OwnerAdoptionSHA256 string           `json:"owner_adoption_sha256"`
	PrelabelDigest      string           `json:"prelabel_digest"`
	QueryID             string           `json:"query_id"`
	CorpusID            string           `json:"corpus_id"`
	Language            string           `json:"language"`
	Cohorts             []string         `json:"cohorts"`
	Cell                ReviewBudgetCell `json:"cell"`
	CandidateCount      int              `json:"candidate_count"`
	EmittedCount        int              `json:"emitted_count"`
	ActualBytes         int              `json:"actual_bytes"`
	OmissionCounts      map[string]int   `json:"omission_counts"`
	NoCandidateAbstain  bool             `json:"no_candidate_abstain"`
	GateAbstain         bool             `json:"gate_abstain"`
	BaselineGroups      int              `json:"baseline_groups"`
	BaselineComplete    int              `json:"baseline_complete_groups"`
	ParentNewCoverage   int              `json:"parent_direct_useful_new_coverage"`
	ParentComplete      int              `json:"parent_complete_groups"`
	DeliveryNewCoverage int              `json:"delivery_useful_new_coverage"`
	DeliveryComplete    int              `json:"delivery_complete_groups"`
	ParentAttachments   int              `json:"emitted_parent_attachments"`
	ParentGrade2        int              `json:"emitted_parent_grade_2"`
	ParentSupport       int              `json:"emitted_parent_support"`
	ParentNoise         int              `json:"emitted_parent_noise"`
	ParentHardNeg       int              `json:"emitted_parent_hard_negative"`
	DeliveryCount       int              `json:"emitted_delivery_attachments"`
	DeliveryUseful      int              `json:"delivery_useful"`
	DeliverySupport     int              `json:"delivery_support"`
	DeliveryNoise       int              `json:"delivery_noise"`
	DeliveryHardNeg     int              `json:"delivery_hard_negative"`
}

type reviewPolicyDelivery struct {
	PreparedDigest      string
	FrozenDigest        string
	OwnerAdoptionSHA256 string
	PrelabelDigest      string
	QueryID             string
	CorpusID            string
	Language            string
	Cohorts             []string
	Cell                ReviewBudgetCell
	RelationKind        string
	Direction           string
	Role                string
	Outcome             string
}

type reviewPolicyCellAggregate struct {
	SchemaVersion         int              `json:"schema_version"`
	Kind                  string           `json:"kind"`
	PreparedDigest        string           `json:"prepared_digest"`
	FrozenDigest          string           `json:"frozen_digest"`
	OwnerAdoptionSHA256   string           `json:"owner_adoption_sha256"`
	PrelabelDigest        string           `json:"prelabel_digest"`
	ScopeType             string           `json:"scope_type"`
	Scope                 string           `json:"scope"`
	Cell                  ReviewBudgetCell `json:"cell"`
	Queries               int              `json:"queries"`
	QueriesWithCandidates int              `json:"queries_with_candidates"`
	NoCandidateAbstain    int              `json:"no_candidate_abstain"`
	GateAbstain           int              `json:"gate_abstain"`
	EmittingQueries       int              `json:"emitting_queries"`
	CandidateRows         int              `json:"candidate_rows"`
	PrelabelEmittedRows   int              `json:"prelabel_emitted_rows"`
	ActualBytes           int              `json:"actual_bytes"`
	OmissionCounts        map[string]int   `json:"omission_counts"`
	BaselineGroups        int              `json:"baseline_groups"`
	BaselineComplete      int              `json:"baseline_complete_groups"`
	ParentNewCoverage     int              `json:"parent_direct_useful_new_coverage"`
	ParentComplete        int              `json:"parent_complete_groups"`
	DeliveryNewCoverage   int              `json:"delivery_useful_new_coverage"`
	DeliveryComplete      int              `json:"delivery_complete_groups"`
	ParentAttachments     int              `json:"emitted_parent_attachments"`
	ParentGrade2          int              `json:"emitted_parent_grade_2"`
	ParentSupport         int              `json:"emitted_parent_support"`
	ParentNoise           int              `json:"emitted_parent_noise"`
	ParentHardNeg         int              `json:"emitted_parent_hard_negative"`
	DeliveryCount         int              `json:"emitted_delivery_attachments"`
	DeliveryUseful        int              `json:"delivery_useful"`
	DeliverySupport       int              `json:"delivery_support"`
	DeliveryNoise         int              `json:"delivery_noise"`
	DeliveryHardNeg       int              `json:"delivery_hard_negative"`
}

type reviewPolicyDeliveryAggregate struct {
	SchemaVersion       int              `json:"schema_version"`
	Kind                string           `json:"kind"`
	PreparedDigest      string           `json:"prepared_digest"`
	FrozenDigest        string           `json:"frozen_digest"`
	OwnerAdoptionSHA256 string           `json:"owner_adoption_sha256"`
	PrelabelDigest      string           `json:"prelabel_digest"`
	ScopeType           string           `json:"scope_type"`
	Scope               string           `json:"scope"`
	Cell                ReviewBudgetCell `json:"cell"`
	Dimension           string           `json:"dimension"`
	Value               string           `json:"value"`
	Deliveries          int              `json:"deliveries"`
	Useful              int              `json:"useful"`
	Support             int              `json:"support"`
	Noise               int              `json:"noise"`
	HardNegative        int              `json:"hard_negative"`
}

type reviewPolicyScope struct {
	Type  string
	Value string
}

// SelectReview evaluates every frozen closure and hint cell. It does not
// choose a policy: semantic cells were deliberately unavailable before labels
// opened, and this bounded run is evidence accounting only.
func SelectReview(preparedDir, frozenDir, outputDir string) error {
	prepared, _, err := readReviewPrepared(preparedDir)
	if err != nil {
		return err
	}
	// The Stage E prepared packet carries the immutable pre-label source and
	// its digest. Freeze and owner adoption bind that digest before labels are
	// used; a self-hashed replacement can therefore never be substituted here.
	prelabel, err := readReviewEmissionFreeze(preparedDir, prepared.Contract)
	if err != nil || prelabel.Digest != prepared.PrelabelDigest || !samePrelabelQueries(prelabel.Queries, prepared.Queries) || !validReviewPrelabelControls(prelabel, prepared) {
		return fmt.Errorf("invalid immutable pre-label control binding")
	}
	var frozen reviewFrozen
	if err = readReviewJSON(filepath.Join(frozenDir, "frozen.json"), &frozen); err != nil {
		return err
	}
	var adoption ReviewAdoption
	if err = readReviewJSON(filepath.Join(frozenDir, "owner-adoption.json"), &adoption); err != nil {
		return err
	}
	parentLabels, relationLabels, err := validatedReviewFrozenLabels(prepared, frozen, adoption)
	if err != nil {
		return err
	}
	if err = ensureEmptyReviewSelectionDir(outputDir); err != nil {
		return err
	}
	rows, deliveries, err := evaluateReviewPolicyCells(prepared, prelabel, frozen, parentLabels, relationLabels)
	if err != nil {
		return err
	}
	if len(rows) != prepared.Contract.QueryCount*(len(reviewClosureCells())+len(reviewHintCells())) {
		return fmt.Errorf("incomplete policy evaluation cell set")
	}
	cellAggregates := aggregateReviewPolicyCells(rows)
	if err = validateReviewPolicyDenominators(rows, cellAggregates); err != nil {
		return err
	}
	deliveryAggregates := aggregateReviewPolicyDeliveries(deliveries)
	selection := struct {
		SchemaVersion            int    `json:"schema_version"`
		Kind                     string `json:"kind"`
		EvaluationPolicy         string `json:"evaluation_policy"`
		SelectionState           string `json:"selection_state"`
		SemanticStatus           string `json:"semantic_status"`
		PreparedDigest           string `json:"prepared_digest"`
		FrozenDigest             string `json:"frozen_digest"`
		OwnerAdoptionSHA256      string `json:"owner_adoption_sha256"`
		PrelabelDigest           string `json:"prelabel_digest"`
		QueryCellRecords         int    `json:"query_cell_records"`
		CellAggregateRecords     int    `json:"cell_aggregate_records"`
		DeliveryAggregateRecords int    `json:"delivery_aggregate_records"`
	}{1, ReviewPolicyEvaluationKind, ReviewPolicyID, ReviewPolicySelectionState, ReviewSemanticStatus, prepared.Digest, frozen.Digest, frozen.OwnerAdoptionSHA256, prelabel.Digest, len(rows), len(cellAggregates), len(deliveryAggregates)}
	if err = writeReviewJSON(filepath.Join(outputDir, "selection.json"), selection); err != nil {
		return err
	}
	queryRows := make([]any, len(rows))
	for i := range rows {
		queryRows[i] = rows[i]
	}
	if err = writeJSONL(filepath.Join(outputDir, "per-query-cell.jsonl"), queryRows); err != nil {
		return err
	}
	aggregateRows := make([]any, len(cellAggregates))
	for i := range cellAggregates {
		aggregateRows[i] = cellAggregates[i]
	}
	if err = writeJSONL(filepath.Join(outputDir, "cell-aggregates.jsonl"), aggregateRows); err != nil {
		return err
	}
	deliveryRows := make([]any, len(deliveryAggregates))
	for i := range deliveryAggregates {
		deliveryRows[i] = deliveryAggregates[i]
	}
	if err = writeJSONL(filepath.Join(outputDir, "delivery-aggregates.jsonl"), deliveryRows); err != nil {
		return err
	}
	return writeChecksums(outputDir)
}

func ensureEmptyReviewSelectionDir(root string) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("policy evaluation output directory must be empty")
	}
	return nil
}

func validReviewPrelabelControls(prelabel reviewEmissionFreeze, prepared reviewPrepared) bool {
	controls := map[string]reviewEmissionControl{}
	for _, control := range prelabel.Controls {
		key := control.QueryID + "\x00" + reviewCellKey(control.Cell)
		if key == "\x00" || controls[key].QueryID != "" || !validReviewPolicyControl(control) {
			return false
		}
		controls[key] = control
	}
	if len(controls) != prepared.Contract.QueryCount*(len(reviewClosureCells())+len(reviewHintCells())) {
		return false
	}
	attachmentQuery := map[string]string{}
	for _, candidate := range prepared.Candidates {
		attachmentQuery[candidate.AttachmentID] = candidate.QueryID
	}
	relations := map[string]ReviewRelationEvidence{}
	for _, relation := range prepared.Relations {
		relations[relation.AttachmentID] = relation
	}
	for _, emission := range prepared.Emissions {
		control, ok := controls[emission.QueryID+"\x00"+reviewCellKey(emission.Cell)]
		if !ok || len(emission.AttachmentIDs) > control.EmittedCount || !sameReviewStringList(emission.AttachmentIDs, uniqueSortedReviewIDs(emission.AttachmentIDs)) || !sameReviewStringList(emission.RelationAttachmentIDs, uniqueSortedReviewIDs(emission.RelationAttachmentIDs)) {
			return false
		}
		targets := map[string]bool{}
		for _, id := range emission.AttachmentIDs {
			if attachmentQuery[id] != emission.QueryID {
				return false
			}
			targets[id] = true
		}
		for _, id := range emission.RelationAttachmentIDs {
			relation, ok := relations[id]
			if !ok || relation.QueryID != emission.QueryID || relation.DeliveryFamily != emission.Cell.Family || !targets[relation.TargetAttachmentID] {
				return false
			}
		}
	}
	return true
}

func validReviewPolicyControl(control reviewEmissionControl) bool {
	if control.QueryID == "" || (control.Cell.Family != "closure" && control.Cell.Family != "hint") || control.CandidateCount < 0 || control.EmittedCount < 0 || control.ActualBytes < 0 || control.EmittedCount > control.CandidateCount || control.EmittedCount > control.Cell.Count || control.ActualBytes > control.Cell.Bytes {
		return false
	}
	if control.Cell.Family == "closure" && (!intMember(control.Cell.Count, closureCountBudgetGrid) || !intMember(control.Cell.Bytes, closureByteBudgetGrid)) {
		return false
	}
	if control.Cell.Family == "hint" && (!intMember(control.Cell.Count, hintCountBudgetGrid) || !intMember(control.Cell.Bytes, hintByteBudgetGrid)) {
		return false
	}
	for key, count := range control.OmissionCounts {
		if key == "" || count <= 0 {
			return false
		}
	}
	return true
}

func validatedReviewFrozenLabels(prepared reviewPrepared, frozen reviewFrozen, adoption ReviewAdoption) (map[string]reviewLabel, map[string]reviewLabel, error) {
	digest := frozen.Digest
	frozen.Digest = ""
	actual, hashErr := canonicalHash(frozen)
	frozen.Digest = digest
	base := frozen
	base.ReconciledDigest = ""
	base.OwnerAdoptionSHA256 = ""
	base.Digest = ""
	reconciled, reconcileErr := canonicalHash(base)
	adoptionDigest, adoptionErr := canonicalHash(adoption)
	if hashErr != nil || reconcileErr != nil || adoptionErr != nil || !validDigest(digest) || actual != digest || reconciled != frozen.ReconciledDigest || frozen.Kind != "cidx.relation_calibration.review_frozen.v1" || frozen.PreparedDigest != prepared.Digest || frozen.PrelabelDigest != prepared.PrelabelDigest || !validReviewAdoption(adoption, frozen.ReconciledDigest, frozen.PrelabelDigest) || frozen.OwnerAdoptionSHA256 != adoptionDigest {
		return nil, nil, fmt.Errorf("invalid frozen review binding")
	}
	parentIDs, relationIDs := map[string]bool{}, map[string]bool{}
	for _, candidate := range prepared.Candidates {
		parentIDs[candidate.AttachmentID] = true
	}
	for _, relation := range prepared.Relations {
		relationIDs[relation.AttachmentID] = true
	}
	parents, err := reviewFrozenLabelMap(prepared, frozen.ParentLabels, parentIDs)
	if err != nil {
		return nil, nil, err
	}
	relations, err := reviewFrozenLabelMap(prepared, frozen.RelationLabels, relationIDs)
	if err != nil {
		return nil, nil, err
	}
	if !validFrozenReviewRelationGrades(prepared, parents, relations) {
		return nil, nil, fmt.Errorf("invalid frozen relation grade")
	}
	return parents, relations, nil
}

func labelsByReviewAttachmentID(labels []reviewLabel) map[string]reviewLabel {
	result := make(map[string]reviewLabel, len(labels))
	for _, label := range labels {
		result[label.AttachmentID] = label
	}
	return result
}

func validFrozenReviewRelationGrades(prepared reviewPrepared, parents, relations map[string]reviewLabel) bool {
	for id, relation := range relations {
		targetID, ok := reviewRelationTargetAttachmentID(prepared, id)
		target, targetOK := parents[targetID]
		if !ok || !targetOK || relation.Grade > target.Grade {
			return false
		}
		if relation.Grade == 2 && (target.Grade != 2 || !reviewGroupSubset(relation.GroupIDs, target.GroupIDs)) {
			return false
		}
	}
	return true
}

func reviewFrozenLabelMap(prepared reviewPrepared, labels []reviewLabel, expected map[string]bool) (map[string]reviewLabel, error) {
	if len(labels) != len(expected) {
		return nil, fmt.Errorf("frozen review label cardinality mismatch")
	}
	result := map[string]reviewLabel{}
	for _, label := range labels {
		if !expected[label.AttachmentID] || result[label.AttachmentID].AttachmentID != "" || !validFrozenReviewLabel(prepared, label) {
			return nil, fmt.Errorf("invalid frozen review label")
		}
		result[label.AttachmentID] = label
	}
	if len(result) != len(expected) {
		return nil, fmt.Errorf("missing frozen review label")
	}
	return result, nil
}

func validFrozenReviewLabel(prepared reviewPrepared, label reviewLabel) bool {
	if (label.Grade != 0 && label.Grade != 1 && label.Grade != 2) || !reviewGroupIDs(label.GroupIDs) || !reviewGroupIDs(label.HardNegativeGroupIDs) {
		return false
	}
	allowed := reviewAllowedGroupIDs(prepared, label.AttachmentID)
	if label.Grade == 2 {
		if len(label.GroupIDs) == 0 || !reviewGroupSubset(label.GroupIDs, allowed) {
			return false
		}
	} else if len(label.GroupIDs) != 0 {
		return false
	}
	if label.HardNegative {
		return label.Grade == 0 && len(label.HardNegativeGroupIDs) > 0 && reviewGroupSubset(label.HardNegativeGroupIDs, reviewQueryTopologyGroupIDs(prepared, label.AttachmentID))
	}
	return len(label.HardNegativeGroupIDs) == 0
}

func evaluateReviewPolicyCells(prepared reviewPrepared, prelabel reviewEmissionFreeze, frozen reviewFrozen, parents, relationLabels map[string]reviewLabel) ([]reviewPolicyQueryCell, []reviewPolicyDelivery, error) {
	controls := map[string]reviewEmissionControl{}
	for _, control := range prelabel.Controls {
		controls[control.QueryID+"\x00"+reviewCellKey(control.Cell)] = control
	}
	queries := map[string]reviewQueryRecord{}
	for _, query := range prepared.Queries {
		queries[query.Packet.QueryID] = query
	}
	candidates := map[string]reviewCandidate{}
	protected := map[string]map[string]bool{}
	for _, candidate := range prepared.Candidates {
		candidates[candidate.AttachmentID] = candidate
		for _, family := range candidate.Families {
			if family == "protected_primary" {
				if protected[candidate.QueryID] == nil {
					protected[candidate.QueryID] = map[string]bool{}
				}
				protected[candidate.QueryID][candidate.AttachmentID] = true
			}
		}
	}
	relations := map[string]ReviewRelationEvidence{}
	for _, relation := range prepared.Relations {
		relations[relation.AttachmentID] = relation
	}
	baseline := map[string]map[string]bool{}
	for queryID, query := range queries {
		if len(protected[queryID]) != ProtectedPrimaryK {
			return nil, nil, fmt.Errorf("protected primary identity mismatch")
		}
		baseline[queryID] = map[string]bool{}
		for id := range protected[queryID] {
			label, ok := parents[id]
			if !ok {
				return nil, nil, fmt.Errorf("protected primary label missing")
			}
			if label.Grade == 2 {
				for _, group := range label.GroupIDs {
					baseline[queryID][group] = true
				}
			}
		}
		if !reviewGroupSetSubset(baseline[queryID], reviewQueryGroupSet(query)) {
			return nil, nil, fmt.Errorf("baseline group outside query topology")
		}
	}
	rows := make([]reviewPolicyQueryCell, 0, len(prepared.Emissions))
	deliveries := []reviewPolicyDelivery{}
	for _, emission := range prepared.Emissions {
		query, ok := queries[emission.QueryID]
		if !ok {
			return nil, nil, fmt.Errorf("emission query is absent from topology")
		}
		control := controls[emission.QueryID+"\x00"+reviewCellKey(emission.Cell)]
		groups := reviewQueryGroupSet(query)
		base := baseline[emission.QueryID]
		parentAdded := map[string]bool{}
		parentGrade2, parentSupport, parentNoise, parentHardNeg := 0, 0, 0, 0
		for _, attachmentID := range emission.AttachmentIDs {
			label := parents[attachmentID]
			switch reviewParentOutcome(label) {
			case "HARD_NEGATIVE":
				parentHardNeg++
			case "GRADE_2":
				parentGrade2++
				for _, group := range label.GroupIDs {
					parentAdded[group] = true
				}
			case "SUPPORT":
				parentSupport++
			default:
				parentNoise++
			}
		}
		parentComplete := unionReviewGroupSets(base, parentAdded)
		deliveryAdded := map[string]bool{}
		deliveryUseful, deliverySupport, deliveryNoise, deliveryHardNeg := 0, 0, 0, 0
		for _, relationID := range emission.RelationAttachmentIDs {
			relation := relations[relationID]
			parentLabel := parents[relation.TargetAttachmentID]
			relationLabel := relationLabels[relationID]
			outcome, shared := reviewDeliveryOutcome(parentLabel, relationLabel)
			switch outcome {
			case "HARD_NEGATIVE":
				deliveryHardNeg++
			case "USEFUL":
				deliveryUseful++
				for _, group := range shared {
					deliveryAdded[group] = true
				}
			case "SUPPORT":
				deliverySupport++
			default:
				deliveryNoise++
			}
			deliveries = append(deliveries, reviewPolicyDelivery{PreparedDigest: prepared.Digest, FrozenDigest: frozen.Digest, OwnerAdoptionSHA256: frozen.OwnerAdoptionSHA256, PrelabelDigest: prelabel.Digest, QueryID: emission.QueryID, CorpusID: query.CorpusID, Language: query.Packet.Language, Cohorts: append([]string(nil), query.Cohorts...), Cell: emission.Cell, RelationKind: relation.RelationKind, Direction: relation.Direction, Role: relation.Role, Outcome: outcome})
		}
		deliveryComplete := unionReviewGroupSets(base, deliveryAdded)
		row := reviewPolicyQueryCell{SchemaVersion: 1, Kind: "policy_evaluation.query_cell.v1", PreparedDigest: prepared.Digest, FrozenDigest: frozen.Digest, OwnerAdoptionSHA256: frozen.OwnerAdoptionSHA256, PrelabelDigest: prelabel.Digest, QueryID: emission.QueryID, CorpusID: query.CorpusID, Language: query.Packet.Language, Cohorts: append([]string(nil), query.Cohorts...), Cell: emission.Cell, CandidateCount: control.CandidateCount, EmittedCount: control.EmittedCount, ActualBytes: control.ActualBytes, OmissionCounts: copyReviewCounts(control.OmissionCounts), NoCandidateAbstain: control.CandidateCount == 0, GateAbstain: false, BaselineGroups: len(groups), BaselineComplete: reviewGroupSetCount(base, groups), ParentNewCoverage: reviewNewGroupCount(parentAdded, base, groups), ParentComplete: reviewGroupSetCount(parentComplete, groups), DeliveryNewCoverage: reviewNewGroupCount(deliveryAdded, base, groups), DeliveryComplete: reviewGroupSetCount(deliveryComplete, groups), ParentAttachments: len(emission.AttachmentIDs), ParentGrade2: parentGrade2, ParentSupport: parentSupport, ParentNoise: parentNoise, ParentHardNeg: parentHardNeg, DeliveryCount: len(emission.RelationAttachmentIDs), DeliveryUseful: deliveryUseful, DeliverySupport: deliverySupport, DeliveryNoise: deliveryNoise, DeliveryHardNeg: deliveryHardNeg}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].QueryID != rows[j].QueryID {
			return rows[i].QueryID < rows[j].QueryID
		}
		if rows[i].Cell.Family != rows[j].Cell.Family {
			return rows[i].Cell.Family < rows[j].Cell.Family
		}
		if rows[i].Cell.Count != rows[j].Cell.Count {
			return rows[i].Cell.Count < rows[j].Cell.Count
		}
		return rows[i].Cell.Bytes < rows[j].Cell.Bytes
	})
	sort.Slice(deliveries, func(i, j int) bool {
		if deliveries[i].QueryID != deliveries[j].QueryID {
			return deliveries[i].QueryID < deliveries[j].QueryID
		}
		if deliveries[i].Cell.Family != deliveries[j].Cell.Family {
			return deliveries[i].Cell.Family < deliveries[j].Cell.Family
		}
		if deliveries[i].Cell.Count != deliveries[j].Cell.Count {
			return deliveries[i].Cell.Count < deliveries[j].Cell.Count
		}
		if deliveries[i].Cell.Bytes != deliveries[j].Cell.Bytes {
			return deliveries[i].Cell.Bytes < deliveries[j].Cell.Bytes
		}
		if deliveries[i].RelationKind != deliveries[j].RelationKind {
			return deliveries[i].RelationKind < deliveries[j].RelationKind
		}
		if deliveries[i].Direction != deliveries[j].Direction {
			return deliveries[i].Direction < deliveries[j].Direction
		}
		return deliveries[i].Role < deliveries[j].Role
	})
	return rows, deliveries, nil
}

func reviewQueryGroupSet(query reviewQueryRecord) map[string]bool {
	result := map[string]bool{}
	for _, group := range query.RequiredGroups {
		result[group.ID] = true
	}
	return result
}
func reviewGroupSetSubset(values, allowed map[string]bool) bool {
	for value := range values {
		if !allowed[value] {
			return false
		}
	}
	return true
}
func unionReviewGroupSets(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range a {
		out[k] = true
	}
	for k := range b {
		out[k] = true
	}
	return out
}
func reviewGroupSetCount(values, allowed map[string]bool) int {
	n := 0
	for value := range values {
		if allowed[value] {
			n++
		}
	}
	return n
}
func reviewNewGroupCount(values, baseline, allowed map[string]bool) int {
	n := 0
	for value := range values {
		if allowed[value] && !baseline[value] {
			n++
		}
	}
	return n
}
func copyReviewCounts(values map[string]int) map[string]int {
	out := map[string]int{}
	for k, v := range values {
		out[k] = v
	}
	return out
}

func reviewParentOutcome(label reviewLabel) string {
	if label.HardNegative {
		return "HARD_NEGATIVE"
	}
	switch label.Grade {
	case 2:
		return "GRADE_2"
	case 1:
		return "SUPPORT"
	default:
		return "NOISE"
	}
}

func validateReviewPolicyDenominators(rows []reviewPolicyQueryCell, aggregates []reviewPolicyCellAggregate) error {
	expected := map[string]reviewPolicyCellAggregate{}
	for _, row := range rows {
		if row.NoCandidateAbstain != (row.CandidateCount == 0) || row.GateAbstain || row.CandidateCount < 0 || row.EmittedCount < 0 || row.EmittedCount > row.CandidateCount || row.EmittedCount > row.Cell.Count || row.ActualBytes < 0 || row.ActualBytes > row.Cell.Bytes || row.ParentAttachments != row.ParentGrade2+row.ParentSupport+row.ParentNoise+row.ParentHardNeg || row.DeliveryCount != row.DeliveryUseful+row.DeliverySupport+row.DeliveryNoise+row.DeliveryHardNeg || row.BaselineComplete > row.BaselineGroups || row.ParentComplete != row.BaselineComplete+row.ParentNewCoverage || row.DeliveryComplete != row.BaselineComplete+row.DeliveryNewCoverage || row.ParentComplete > row.BaselineGroups || row.DeliveryComplete > row.BaselineGroups {
			return fmt.Errorf("invalid policy evaluation denominator equation")
		}
		key := reviewCellKey(row.Cell)
		value := expected[key]
		value.Cell = row.Cell
		value.Queries++
		if row.CandidateCount > 0 {
			value.QueriesWithCandidates++
		}
		if row.NoCandidateAbstain {
			value.NoCandidateAbstain++
		}
		if row.EmittedCount > 0 {
			value.EmittingQueries++
		}
		value.CandidateRows += row.CandidateCount
		value.PrelabelEmittedRows += row.EmittedCount
		value.ActualBytes += row.ActualBytes
		value.BaselineGroups += row.BaselineGroups
		value.BaselineComplete += row.BaselineComplete
		value.ParentNewCoverage += row.ParentNewCoverage
		value.ParentComplete += row.ParentComplete
		value.DeliveryNewCoverage += row.DeliveryNewCoverage
		value.DeliveryComplete += row.DeliveryComplete
		value.ParentAttachments += row.ParentAttachments
		value.ParentGrade2 += row.ParentGrade2
		value.ParentSupport += row.ParentSupport
		value.ParentNoise += row.ParentNoise
		value.ParentHardNeg += row.ParentHardNeg
		value.DeliveryCount += row.DeliveryCount
		value.DeliveryUseful += row.DeliveryUseful
		value.DeliverySupport += row.DeliverySupport
		value.DeliveryNoise += row.DeliveryNoise
		value.DeliveryHardNeg += row.DeliveryHardNeg
		if value.OmissionCounts == nil {
			value.OmissionCounts = map[string]int{}
		}
		for omission, count := range row.OmissionCounts {
			value.OmissionCounts[omission] += count
		}
		expected[key] = value
	}
	seen := map[string]bool{}
	for _, aggregate := range aggregates {
		if aggregate.ScopeType != "global" || aggregate.Scope != "all" {
			continue
		}
		key := reviewCellKey(aggregate.Cell)
		if seen[key] || !sameReviewPolicyAggregate(expected[key], aggregate) {
			return fmt.Errorf("global policy evaluation aggregate mismatch")
		}
		seen[key] = true
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("missing global policy evaluation aggregate")
	}
	return nil
}

func sameReviewPolicyAggregate(expected, actual reviewPolicyCellAggregate) bool {
	return expected.Queries == actual.Queries && expected.QueriesWithCandidates == actual.QueriesWithCandidates && expected.NoCandidateAbstain == actual.NoCandidateAbstain && expected.GateAbstain == actual.GateAbstain && expected.EmittingQueries == actual.EmittingQueries && expected.CandidateRows == actual.CandidateRows && expected.PrelabelEmittedRows == actual.PrelabelEmittedRows && expected.ActualBytes == actual.ActualBytes && expected.BaselineGroups == actual.BaselineGroups && expected.BaselineComplete == actual.BaselineComplete && expected.ParentNewCoverage == actual.ParentNewCoverage && expected.ParentComplete == actual.ParentComplete && expected.DeliveryNewCoverage == actual.DeliveryNewCoverage && expected.DeliveryComplete == actual.DeliveryComplete && expected.ParentAttachments == actual.ParentAttachments && expected.ParentGrade2 == actual.ParentGrade2 && expected.ParentSupport == actual.ParentSupport && expected.ParentNoise == actual.ParentNoise && expected.ParentHardNeg == actual.ParentHardNeg && expected.DeliveryCount == actual.DeliveryCount && expected.DeliveryUseful == actual.DeliveryUseful && expected.DeliverySupport == actual.DeliverySupport && expected.DeliveryNoise == actual.DeliveryNoise && expected.DeliveryHardNeg == actual.DeliveryHardNeg && sameReviewCountMap(expected.OmissionCounts, actual.OmissionCounts)
}

func sameReviewCountMap(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func reviewDeliveryOutcome(parent, relation reviewLabel) (string, []string) {
	if parent.HardNegative || relation.HardNegative {
		return "HARD_NEGATIVE", nil
	}
	if parent.Grade == 2 && relation.Grade == 2 {
		shared := []string{}
		for _, group := range parent.GroupIDs {
			if intMemberString(group, relation.GroupIDs) {
				shared = append(shared, group)
			}
		}
		if len(shared) > 0 {
			return "USEFUL", uniqueSortedReviewIDs(shared)
		}
	}
	if parent.Grade == 1 || relation.Grade == 1 {
		return "SUPPORT", nil
	}
	return "NOISE", nil
}
func intMemberString(value string, values []string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func reviewPolicyScopes(corpus, language string, cohorts []string) []reviewPolicyScope {
	result := []reviewPolicyScope{{Type: "global", Value: "all"}, {Type: "corpus", Value: corpus}, {Type: "language", Value: language}}
	for _, cohort := range uniqueSortedReviewIDs(cohorts) {
		result = append(result, reviewPolicyScope{Type: "cohort", Value: cohort})
	}
	return result
}

func aggregateReviewPolicyCells(rows []reviewPolicyQueryCell) []reviewPolicyCellAggregate {
	values := map[string]*reviewPolicyCellAggregate{}
	for _, row := range rows {
		for _, scope := range reviewPolicyScopes(row.CorpusID, row.Language, row.Cohorts) {
			key := scope.Type + "\x00" + scope.Value + "\x00" + reviewCellKey(row.Cell)
			value := values[key]
			if value == nil {
				value = &reviewPolicyCellAggregate{SchemaVersion: 1, Kind: "policy_evaluation.cell_aggregate.v1", PreparedDigest: row.PreparedDigest, FrozenDigest: row.FrozenDigest, OwnerAdoptionSHA256: row.OwnerAdoptionSHA256, PrelabelDigest: row.PrelabelDigest, ScopeType: scope.Type, Scope: scope.Value, Cell: row.Cell, OmissionCounts: map[string]int{}}
				values[key] = value
			}
			value.Queries++
			if row.CandidateCount > 0 {
				value.QueriesWithCandidates++
			}
			if row.NoCandidateAbstain {
				value.NoCandidateAbstain++
			}
			if row.GateAbstain {
				value.GateAbstain++
			}
			if row.EmittedCount > 0 {
				value.EmittingQueries++
			}
			value.CandidateRows += row.CandidateCount
			value.PrelabelEmittedRows += row.EmittedCount
			value.ActualBytes += row.ActualBytes
			value.BaselineGroups += row.BaselineGroups
			value.BaselineComplete += row.BaselineComplete
			value.ParentNewCoverage += row.ParentNewCoverage
			value.ParentComplete += row.ParentComplete
			value.DeliveryNewCoverage += row.DeliveryNewCoverage
			value.DeliveryComplete += row.DeliveryComplete
			value.ParentAttachments += row.ParentAttachments
			value.ParentGrade2 += row.ParentGrade2
			value.ParentSupport += row.ParentSupport
			value.ParentNoise += row.ParentNoise
			value.ParentHardNeg += row.ParentHardNeg
			value.DeliveryCount += row.DeliveryCount
			value.DeliveryUseful += row.DeliveryUseful
			value.DeliverySupport += row.DeliverySupport
			value.DeliveryNoise += row.DeliveryNoise
			value.DeliveryHardNeg += row.DeliveryHardNeg
			for omission, count := range row.OmissionCounts {
				value.OmissionCounts[omission] += count
			}
		}
	}
	result := make([]reviewPolicyCellAggregate, 0, len(values))
	for _, value := range values {
		result = append(result, *value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ScopeType != result[j].ScopeType {
			return result[i].ScopeType < result[j].ScopeType
		}
		if result[i].Scope != result[j].Scope {
			return result[i].Scope < result[j].Scope
		}
		if result[i].Cell.Family != result[j].Cell.Family {
			return result[i].Cell.Family < result[j].Cell.Family
		}
		if result[i].Cell.Count != result[j].Cell.Count {
			return result[i].Cell.Count < result[j].Cell.Count
		}
		return result[i].Cell.Bytes < result[j].Cell.Bytes
	})
	return result
}

func aggregateReviewPolicyDeliveries(deliveries []reviewPolicyDelivery) []reviewPolicyDeliveryAggregate {
	values := map[string]*reviewPolicyDeliveryAggregate{}
	for _, delivery := range deliveries {
		for _, scope := range reviewPolicyScopes(delivery.CorpusID, delivery.Language, delivery.Cohorts) {
			for _, dimension := range []struct{ Name, Value string }{{"relation_kind", delivery.RelationKind}, {"direction", delivery.Direction}, {"role", delivery.Role}} {
				key := scope.Type + "\x00" + scope.Value + "\x00" + reviewCellKey(delivery.Cell) + "\x00" + dimension.Name + "\x00" + dimension.Value
				value := values[key]
				if value == nil {
					value = &reviewPolicyDeliveryAggregate{SchemaVersion: 1, Kind: "policy_evaluation.delivery_aggregate.v1", PreparedDigest: delivery.PreparedDigest, FrozenDigest: delivery.FrozenDigest, OwnerAdoptionSHA256: delivery.OwnerAdoptionSHA256, PrelabelDigest: delivery.PrelabelDigest, ScopeType: scope.Type, Scope: scope.Value, Cell: delivery.Cell, Dimension: dimension.Name, Value: dimension.Value}
					values[key] = value
				}
				value.Deliveries++
				switch delivery.Outcome {
				case "USEFUL":
					value.Useful++
				case "SUPPORT":
					value.Support++
				case "HARD_NEGATIVE":
					value.HardNegative++
				default:
					value.Noise++
				}
			}
		}
	}
	result := make([]reviewPolicyDeliveryAggregate, 0, len(values))
	for _, value := range values {
		result = append(result, *value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ScopeType != result[j].ScopeType {
			return result[i].ScopeType < result[j].ScopeType
		}
		if result[i].Scope != result[j].Scope {
			return result[i].Scope < result[j].Scope
		}
		if result[i].Cell.Family != result[j].Cell.Family {
			return result[i].Cell.Family < result[j].Cell.Family
		}
		if result[i].Cell.Count != result[j].Cell.Count {
			return result[i].Cell.Count < result[j].Cell.Count
		}
		if result[i].Cell.Bytes != result[j].Cell.Bytes {
			return result[i].Cell.Bytes < result[j].Cell.Bytes
		}
		if result[i].Dimension != result[j].Dimension {
			return result[i].Dimension < result[j].Dimension
		}
		return result[i].Value < result[j].Value
	})
	return result
}

// reviewCompletionManifestProjection intentionally projects only the fields
// consumed by Stage E. It is decoded only after the complete immutable
// completion artifact has passed checksum verification; ordinary review input
// and packet JSON continue to use strict decoding.
type reviewCompletionManifestProjection struct {
	Kind                    string `json:"kind"`
	LabelLoading            string `json:"label_loading"`
	CorpusID                string `json:"corpus_id"`
	DatasetSHA256           string `json:"dataset_sha256"`
	ArtifactChecksumsSHA256 string `json:"-"`
}

func loadReviewCompletion(input ReviewCompletionInput) ([]semanticEndpointFeature, []relationHint, []closureCandidate, []string, reviewCompletionManifestProjection, error) {
	if input.Directory == "" {
		return nil, nil, nil, nil, reviewCompletionManifestProjection{}, fmt.Errorf("missing completion directory")
	}
	expected := []string{"run-manifest.json", "input-artifact-binding.json", "semantic-parent-scores.jsonl", "relation-endpoint-features.jsonl", "contract-closure-candidates.jsonl", "relation-hints.jsonl", "semantic-admission-results.jsonl", "closure-package-results.jsonl", "per-query-relation-trace.jsonl", "aggregate-relation-metrics.json", "cohort-language-report.json", "first-loss-report.json", "report.md"}
	if err := verifyChecksums(input.Directory, expected); err != nil {
		return nil, nil, nil, nil, reviewCompletionManifestProjection{}, err
	}
	manifest, err := readChecksumVerifiedCompletionManifestProjection(input.Directory)
	if err != nil {
		return nil, nil, nil, nil, reviewCompletionManifestProjection{}, err
	}
	if manifest.Kind != ReviewAcceptedCompletionKind || manifest.LabelLoading != "LABEL_FIELDS_NOT_DECODED_STAGE_A" || manifest.CorpusID == "" || !validDigest(manifest.DatasetSHA256) {
		return nil, nil, nil, nil, reviewCompletionManifestProjection{}, fmt.Errorf("completion directory is not accepted review v2")
	}
	artifactChecksum, err := fileSHA256(filepath.Join(input.Directory, "artifact-checksums.json"))
	if err != nil {
		return nil, nil, nil, nil, reviewCompletionManifestProjection{}, err
	}
	manifest.ArtifactChecksumsSHA256 = artifactChecksum
	features := []semanticEndpointFeature{}
	hints := []relationHint{}
	closures := []closureCandidate{}
	if err := readReviewJSONL(filepath.Join(input.Directory, "relation-endpoint-features.jsonl"), &features); err != nil {
		return nil, nil, nil, nil, reviewCompletionManifestProjection{}, err
	}
	if err := readReviewJSONL(filepath.Join(input.Directory, "relation-hints.jsonl"), &hints); err != nil {
		return nil, nil, nil, nil, reviewCompletionManifestProjection{}, err
	}
	if err := readReviewJSONL(filepath.Join(input.Directory, "contract-closure-candidates.jsonl"), &closures); err != nil {
		return nil, nil, nil, nil, reviewCompletionManifestProjection{}, err
	}
	traceIDs, err := readChecksumVerifiedCompletionTraceIDs(input.Directory)
	if err != nil {
		return nil, nil, nil, nil, reviewCompletionManifestProjection{}, err
	}
	ids := make(map[string]bool, len(traceIDs))
	for _, id := range traceIDs {
		ids[id] = true
	}
	for _, v := range features {
		if !ids[v.QueryID] {
			return nil, nil, nil, nil, reviewCompletionManifestProjection{}, fmt.Errorf("endpoint feature lacks completion query trace")
		}
	}
	for _, v := range hints {
		if !ids[v.QueryID] {
			return nil, nil, nil, nil, reviewCompletionManifestProjection{}, fmt.Errorf("relation hint lacks completion query trace")
		}
	}
	for _, v := range closures {
		if !ids[v.QueryID] {
			return nil, nil, nil, nil, reviewCompletionManifestProjection{}, fmt.Errorf("closure candidate lacks completion query trace")
		}
	}
	return features, hints, closures, traceIDs, manifest, nil
}
func readChecksumVerifiedCompletionManifestProjection(fileRoot string) (reviewCompletionManifestProjection, error) {
	data, err := os.ReadFile(filepath.Join(fileRoot, "run-manifest.json"))
	if err != nil {
		return reviewCompletionManifestProjection{}, err
	}
	var manifest reviewCompletionManifestProjection
	// The enclosing loader has verified artifact-checksums.json against the
	// full fixed Stage-A file set. Allow producer-owned additive fields such as
	// build_info without weakening strict decoding for mutable inputs.
	if err := json.Unmarshal(data, &manifest); err != nil {
		return reviewCompletionManifestProjection{}, err
	}
	return manifest, nil
}

// readChecksumVerifiedCompletionTraceIDs projects only the stable trace
// identity. As with the manifest projection, it is called only after the
// enclosing loader has verified the complete immutable Stage-A artifact.
// Per-query trace rows contain producer-owned diagnostic detail (including
// anchor_group) that Stage E neither consumes nor reinterprets.
func readChecksumVerifiedCompletionTraceIDs(fileRoot string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(fileRoot, "per-query-relation-trace.jsonl"))
	if err != nil {
		return nil, err
	}
	type traceProjection struct {
		QueryID string `json:"query_id"`
	}
	seen := map[string]bool{}
	ids := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var trace traceProjection
		if err := json.Unmarshal([]byte(line), &trace); err != nil {
			return nil, err
		}
		if trace.QueryID == "" || seen[trace.QueryID] {
			return nil, fmt.Errorf("invalid completion query trace")
		}
		seen[trace.QueryID] = true
		ids = append(ids, trace.QueryID)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("completion query trace is empty")
	}
	sort.Strings(ids)
	return ids, nil
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

func validateReviewMember(manifest reviewCompletionManifestProjection, members []ReviewSeriesMember, queryCount int, seen map[string]bool) error {
	if len(members) == 0 {
		return nil
	}
	for _, member := range members {
		if member.CorpusID == manifest.CorpusID && member.DatasetSHA256 == manifest.DatasetSHA256 && member.CompletionArtifactChecksum == manifest.ArtifactChecksumsSHA256 && member.QueryCount == queryCount {
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
func writeReviewPrepared(root string, prepared reviewPrepared, attachments []ReviewAttachment, prelabel reviewEmissionFreeze) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	if err := writeReviewJSON(filepath.Join(root, "prepared.json"), prepared); err != nil {
		return err
	}
	if prelabel.Digest != prepared.PrelabelDigest {
		return fmt.Errorf("prepared pre-label digest mismatch")
	}
	if err := writeReviewJSON(filepath.Join(root, "emissions-prelabels.json"), prelabel); err != nil {
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
	if err != nil || !validDigest(digest) || actual != digest || prepared.Kind != "cidx.relation_calibration.review_prepared.v1" || validateReviewContract(prepared.Contract) != nil || len(prepared.ClosureCells) != 9 || len(prepared.HintCells) != 16 || !validReviewEmissionSet(prepared) || prepared.SemanticStatus != ReviewSemanticStatus || !validReviewUniverse(prepared) || !validDigest(prepared.PrelabelDigest) {
		return reviewPrepared{}, nil, fmt.Errorf("invalid prepared review")
	}
	prelabel, prelabelErr := readReviewEmissionFreeze(root, prepared.Contract)
	if prelabelErr != nil || prelabel.Digest != prepared.PrelabelDigest || !samePrelabelQueries(prelabel.Queries, prepared.Queries) || !validReviewPrelabelControls(prelabel, prepared) {
		return reviewPrepared{}, nil, fmt.Errorf("invalid prepared pre-label binding")
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
func validReviewAdoption(adoption ReviewAdoption, reconciled, prelabel string) bool {
	return adoption.SchemaVersion == 1 && adoption.Kind == "cidx.relation_calibration.owner_adoption.v1" && adoption.Adopted && adoption.FrozenDigest == reconciled && adoption.PrelabelDigest == prelabel && adoption.ProtocolVersion == "owner-adopted-dual-ai-v1" && adoption.RelevanceAuthority == "OWNER_ADOPTED_DUAL_AI_REVIEW" && adoption.ReviewValidation == "NO_INDEPENDENT_HUMAN_REVIEW" && len(adoption.Overrides) == 0
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
