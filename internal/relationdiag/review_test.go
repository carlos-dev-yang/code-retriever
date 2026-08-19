package relationdiag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reviewTestContract() ReviewSeriesContract {
	return canonicalReviewSeriesContract()
}

func TestChecksumVerifiedCompletionManifestProjectionAllowsProducerBuildInfo(t *testing.T) {
	root := t.TempDir()
	manifest := []byte(`{
  "kind": "cidx.relation_evidence_completion.stage_a.v1",
  "label_loading": "LABEL_FIELDS_NOT_DECODED_STAGE_A",
  "corpus_id": "producer-corpus",
  "dataset_sha256": "b5ddc298a3a7c8b5a816ce439208667fa566c69e175777d9205a6bf983ef80ba",
  "build_info": {"producer": "stage-a", "version": 2}
}`)
	if err := os.WriteFile(filepath.Join(root, "run-manifest.json"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	var strict struct {
		Kind string `json:"kind"`
	}
	if err := readReviewJSON(filepath.Join(root, "run-manifest.json"), &strict); err == nil {
		t.Fatal("ordinary strict JSON reader accepted producer-only build_info")
	}
	got, err := readChecksumVerifiedCompletionManifestProjection(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != ReviewAcceptedCompletionKind || got.CorpusID != "producer-corpus" || got.DatasetSHA256 != "b5ddc298a3a7c8b5a816ce439208667fa566c69e175777d9205a6bf983ef80ba" {
		t.Fatalf("unexpected manifest projection: %+v", got)
	}
}

func TestChecksumVerifiedCompletionTraceProjectionAllowsProducerAnchorGroup(t *testing.T) {
	root := t.TempDir()
	trace := []byte(`{"query_id":"q-2","anchor_group":{"query_id":"q-2","anchors":[]},"frontier_final_edges":3}` + "\n" +
		`{"query_id":"q-1","anchor_group":{"query_id":"q-1","anchors":[]},"frontier_final_edges":1}` + "\n")
	path := filepath.Join(root, "per-query-relation-trace.jsonl")
	if err := os.WriteFile(path, trace, 0o600); err != nil {
		t.Fatal(err)
	}
	var strict []struct {
		QueryID string `json:"query_id"`
	}
	if err := readReviewJSONL(path, &strict); err == nil {
		t.Fatal("ordinary strict JSONL reader accepted producer-only anchor_group")
	}
	ids, err := readChecksumVerifiedCompletionTraceIDs(root)
	if err != nil {
		t.Fatal(err)
	}
	if !sameReviewStringList(ids, []string{"q-1", "q-2"}) {
		t.Fatalf("unexpected trace IDs: %v", ids)
	}
	if err := os.WriteFile(path, append(trace, []byte(`{"query_id":"q-1"}`+"\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readChecksumVerifiedCompletionTraceIDs(root); err == nil {
		t.Fatal("accepted duplicate completion trace query ID")
	}
}

func TestReviewPolicyFreezesCellsAndEmissionsBeforeLabels(t *testing.T) {
	if got := len(reviewClosureCells()); got != 9 {
		t.Fatalf("closure cells=%d", got)
	}
	if got := len(reviewHintCells()); got != 16 {
		t.Fatalf("hint cells=%d", got)
	}
	if ReviewSemanticStatus != "NOT_OPENED_NO_FINITE_CELL_MANIFEST" {
		t.Fatalf("semantic status=%q", ReviewSemanticStatus)
	}
	if reviewShuffleKey(completionTestDigest, "pass-1", "a") == reviewShuffleKey(completionTestDigest, "pass-2", "a") {
		t.Fatal("review passes use the same shuffle seed")
	}
}

func TestReviewPassRejectsUngradedAndInvalidHardNegative(t *testing.T) {
	root := t.TempDir()
	contract := reviewTestContract()
	prepared := reviewPrepared{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_prepared.v1", Policy: ReviewPolicyID, Contract: contract, SemanticStatus: ReviewSemanticStatus, ClosureCells: reviewClosureCells(), HintCells: reviewHintCells()}
	for i := 0; i < 40; i++ {
		queryID := fmt.Sprintf("q%02d", i)
		query := reviewQueryRecord{Packet: ReviewPacketQuery{QueryID: queryID, Text: "q", Language: "go", AnswerMode: "SINGLE"}, CorpusID: "test-corpus"}
		if i == 0 {
			query.RequiredGroups = []ReviewRequiredGroup{{ID: "g-1", SourceParentIDs: []string{"p0"}}}
		}
		prepared.Queries = append(prepared.Queries, query)
		for primary := 0; primary < ProtectedPrimaryK; primary++ {
			id := fmt.Sprintf("%s-a%d", queryID, primary)
			prepared.Universe = append(prepared.Universe, ReviewAttachment{AttachmentID: id, QueryID: queryID, Path: id + ".go", IndexedSHA256: completionTestDigest, StartByte: 0, EndByte: 4, Body: "body"})
			candidate := reviewCandidate{AttachmentID: id, QueryID: queryID, Families: []string{"protected_primary"}}
			if queryID == "q00" && primary == 0 {
				candidate.RequiredGroupIDs = []string{"g-1"}
			}
			prepared.Candidates = append(prepared.Candidates, candidate)
		}
	}
	var universeErr error
	prepared.UniverseDigest, universeErr = canonicalReviewUniverseHash(prepared)
	if universeErr != nil {
		t.Fatal(universeErr)
	}
	for query := 0; query < 40; query++ {
		for _, cell := range append(reviewClosureCells(), reviewHintCells()...) {
			prepared.Emissions = append(prepared.Emissions, reviewEmission{QueryID: fmt.Sprintf("q%02d", query), Cell: cell})
		}
	}
	prelabelQueries := make([]ReviewEmissionQuery, 0, len(prepared.Queries))
	controls := make([]reviewEmissionControl, 0, len(prepared.Queries)*25)
	for _, query := range prepared.Queries {
		prelabelQueries = append(prelabelQueries, ReviewEmissionQuery{QueryID: query.Packet.QueryID, Text: query.Packet.Text, Language: query.Packet.Language, AnswerMode: query.Packet.AnswerMode})
		for _, cell := range append(reviewClosureCells(), reviewHintCells()...) {
			controls = append(controls, reviewEmissionControl{QueryID: query.Packet.QueryID, Cell: cell, OmissionCounts: map[string]int{}})
		}
	}
	prelabel := reviewEmissionFreeze{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_emissions_prelabels.v1", Contract: contract, Queries: prelabelQueries, Controls: controls}
	var err error
	prelabel.Digest, err = canonicalReviewEmissionFreezeHash(prelabel)
	if err != nil {
		t.Fatal(err)
	}
	prepared.PrelabelDigest = prelabel.Digest
	digest, err := canonicalReviewPreparedHash(prepared)
	if err != nil {
		t.Fatal(err)
	}
	prepared.Digest = digest
	packetQueries := make([]ReviewPacketQuery, len(prepared.Queries))
	for i := range prepared.Queries {
		packetQueries[i] = prepared.Queries[i].Packet
	}
	packet := ReviewPacket{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_packet.v1", Policy: ReviewPolicyID, PreparedDigest: digest, CanonicalUniverseDigest: prepared.UniverseDigest, PassID: "pass-1", Attachments: prepared.Universe, Queries: packetQueries}
	if err := writeReviewJSON(filepath.Join(root, "prepared.json"), prepared); err != nil {
		t.Fatal(err)
	}
	if err := writeReviewJSON(filepath.Join(root, "emissions-prelabels.json"), prelabel); err != nil {
		t.Fatal(err)
	}
	if err := writeReviewJSON(filepath.Join(root, "pass-1-packet.json"), packet); err != nil {
		t.Fatal(err)
	}
	tamperedQuery := packet
	tamperedQuery.Queries = append([]ReviewPacketQuery(nil), packet.Queries...)
	tamperedQuery.Queries[0].Text = "different question"
	if err := writeReviewJSON(filepath.Join(root, "pass-1-packet.json"), tamperedQuery); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readReviewPrepared(root); err == nil {
		t.Fatal("accepted packet query tampering")
	}
	if err := writeReviewJSON(filepath.Join(root, "pass-1-packet.json"), packet); err != nil {
		t.Fatal(err)
	}
	packetTwo := packet
	packetTwo.PassID = "pass-2"
	packetTwo.Attachments = shuffleReviewAttachments(prepared.Universe, digest, "pass-2")
	if sameReviewAttachmentOrder(packet.Attachments, packetTwo.Attachments) {
		packetTwo.Attachments = append(packetTwo.Attachments[1:], packetTwo.Attachments[0])
	}
	if err := writeReviewJSON(filepath.Join(root, "pass-2-packet.json"), packetTwo); err != nil {
		t.Fatal(err)
	}
	tampered := packet
	tampered.Attachments = append([]ReviewAttachment(nil), packet.Attachments...)
	tampered.Attachments[0].Body = "evil"
	if err := writeReviewJSON(filepath.Join(root, "pass-1-packet.json"), tampered); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readReviewPrepared(root); err == nil {
		t.Fatal("accepted source-body packet tampering")
	}
	if err := writeReviewJSON(filepath.Join(root, "pass-1-packet.json"), packet); err != nil {
		t.Fatal(err)
	}
	packetDigest, err := canonicalHash(packet)
	if err != nil {
		t.Fatal(err)
	}
	grades := make([]ReviewGrade, 0, len(prepared.Universe))
	for _, attachment := range prepared.Universe {
		grade := ReviewGrade{AttachmentID: attachment.AttachmentID, Grade: 0, Rationale: "not needed"}
		if attachment.AttachmentID == "q00-a0" {
			grade = ReviewGrade{AttachmentID: attachment.AttachmentID, Grade: 2, RequiredGroupIDs: []string{"g-1"}, Rationale: "supported"}
		}
		grades = append(grades, grade)
	}
	pass := ReviewPass{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_pass.v1", PreparedDigest: digest, PacketDigest: packetDigest, PassID: "pass-1", ReviewerID: "reviewer-a", ModelFamily: "model-a", SourceVerified: true, BlindnessAttestation: reviewBlindnessAttestation, Grades: grades}
	if err := ValidateReviewPass(root, pass); err != nil {
		t.Fatal(err)
	}
	pass.Grades[0].RequiredGroupIDs = []string{"invented"}
	if err := ValidateReviewPass(root, pass); err == nil {
		t.Fatal("accepted invented group")
	}
	pass.Grades[0].RequiredGroupIDs = []string{"g-1"}
	pass.Grades[0].HardNegative = true
	if err := ValidateReviewPass(root, pass); err == nil {
		t.Fatal("accepted hard negative without group IDs")
	}
	pass.Grades[0].HardNegative = false
	secondDigest, err := canonicalHash(packetTwo)
	if err != nil {
		t.Fatal(err)
	}
	secondGrades := append([]ReviewGrade(nil), grades...)
	secondGrades[0] = ReviewGrade{AttachmentID: "q00-a0", Grade: 0, HardNegative: true, HardNegativeGroupIDs: []string{"g-1"}, HardNegativeReason: "misleading", Rationale: "not useful"}
	second := ReviewPass{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_pass.v1", PreparedDigest: digest, PacketDigest: secondDigest, PassID: "pass-2", ReviewerID: "reviewer-b", ModelFamily: "model-b", SourceVerified: true, BlindnessAttestation: reviewBlindnessAttestation, Grades: secondGrades}
	frozenDir := filepath.Join(root, "frozen")
	if _, err := PrepareReviewAdoption(root, frozenDir, pass, second); err == nil {
		t.Fatal("previously blessed grade-2 conflict was accepted")
	}
	second.Grades[0] = ReviewGrade{AttachmentID: "q00-a0", Grade: 2, RequiredGroupIDs: []string{"g-1"}, Rationale: "supported"}
	if err := writeReviewJSON(filepath.Join(root, "pass-2-packet.json"), packetTwo); err != nil {
		t.Fatal(err)
	}
	expected, err := PrepareReviewAdoption(root, frozenDir, pass, second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := FreezeReview(root, frozenDir, pass, second, ReviewAdoption{SchemaVersion: 1, Kind: "cidx.relation_calibration.owner_adoption.v1", FrozenDigest: expected, PrelabelDigest: prelabel.Digest, Adopted: true, ProtocolVersion: "owner-adopted-dual-ai-v1", RelevanceAuthority: "OWNER_ADOPTED_DUAL_AI_REVIEW", ReviewValidation: "NO_INDEPENDENT_HUMAN_REVIEW", Overrides: []string{}}); err != nil {
		t.Fatal(err)
	}
	selected := filepath.Join(root, "selected")
	if err := SelectReview(root, frozenDir, selected); err != nil {
		t.Fatal(err)
	}
	var selection map[string]any
	if err := readReviewJSON(filepath.Join(selected, "selection.json"), &selection); err != nil {
		t.Fatal(err)
	}
	if selection["kind"] != ReviewPolicyEvaluationKind || selection["selection_state"] != ReviewPolicySelectionState || selection["query_cell_records"] != float64(1000) || !validDigest(selection["prepared_digest"].(string)) || !validDigest(selection["frozen_digest"].(string)) || !validDigest(selection["owner_adoption_sha256"].(string)) || !validDigest(selection["prelabel_digest"].(string)) {
		t.Fatalf("unexpected policy selection: %+v", selection)
	}
	rows := []reviewPolicyQueryCell{}
	if err := readReviewJSONL(filepath.Join(selected, "per-query-cell.jsonl"), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1000 {
		t.Fatalf("per-query cell rows=%d", len(rows))
	}
	familyRows := map[string]int{}
	for _, row := range rows {
		familyRows[row.Cell.Family]++
	}
	if familyRows["closure"] != 360 || familyRows["hint"] != 640 {
		t.Fatalf("unexpected cell family rows=%v", familyRows)
	}
	if err := verifyChecksums(selected, []string{"selection.json", "per-query-cell.jsonl", "cell-aggregates.jsonl", "delivery-aggregates.jsonl"}); err != nil {
		t.Fatal(err)
	}
	repeat := filepath.Join(root, "selected-repeat")
	if err := SelectReview(root, frozenDir, repeat); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"selection.json", "per-query-cell.jsonl", "cell-aggregates.jsonl", "delivery-aggregates.jsonl", "artifact-checksums.json"} {
		first, err := os.ReadFile(filepath.Join(selected, name))
		if err != nil {
			t.Fatal(err)
		}
		second, err := os.ReadFile(filepath.Join(repeat, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(first) != string(second) {
			t.Fatalf("selection repeat differs for %s", name)
		}
	}
	// A replacement that recomputes its own digest is still rejected: the
	// prepared, frozen, and adopted digests bind the original pre-label object.
	tamperedPrelabel := prelabel
	tamperedPrelabel.Controls = append([]reviewEmissionControl(nil), prelabel.Controls...)
	tamperedPrelabel.Controls[0].CandidateCount = 1
	tamperedPrelabel.Digest, err = canonicalReviewEmissionFreezeHash(tamperedPrelabel)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeReviewJSON(filepath.Join(root, "emissions-prelabels.json"), tamperedPrelabel); err != nil {
		t.Fatal(err)
	}
	if err := SelectReview(root, frozenDir, filepath.Join(root, "selected-tampered-prelabel")); err == nil {
		t.Fatal("accepted self-hashed pre-label replacement")
	}
}

func TestPrepareReviewRejectsIncompleteImmutableSeries(t *testing.T) {
	completion := t.TempDir()
	queries := make([]string, 40)
	traces := make([]any, 0, len(queries))
	queryTopology := make([]ReviewQuery, 0, len(queries))
	for i := range queries {
		queries[i] = fmt.Sprintf("q%02d", i)
		traces = append(traces, map[string]string{"query_id": queries[i]})
		queryTopology = append(queryTopology, ReviewQuery{QueryID: queries[i], Text: "question", Language: "go", AnswerMode: "answerable", Cohorts: []string{"natural"}, ProtectedTop5: []string{"p000", "p001", "p002", "p003", "p004"}, RequiredGroups: []ReviewRequiredGroup{{ID: "g-" + queries[i], SourceParentIDs: []string{fmt.Sprintf("p%03d", i)}}}})
	}
	features := make([]any, 0, 250)
	bodies := make([]ReviewSourceBody, 0, 251)
	for i := 0; i < 250; i++ {
		query, parent := queries[i%len(queries)], fmt.Sprintf("p%03d", i)
		features = append(features, semanticEndpointFeature{QueryID: query, EndpointParentID: parent})
		bodies = append(bodies, ReviewSourceBody{ParentID: parent, Path: parent + ".go", IndexedSHA256: completionTestDigest, StartByte: 0, EndByte: 6, Body: "source"})
	}
	bodies = append(bodies, ReviewSourceBody{Path: "hint.go", IndexedSHA256: completionTestDigest, StartByte: 0, EndByte: 11, Body: "hint source"})
	hints, closures := make([]any, 0, 289), make([]any, 0, 576)
	for i := 0; i < 289; i++ {
		hints = append(hints, relationHint{QueryID: queries[i%len(queries)], TargetPath: "hint.go", TargetSHA256: completionTestDigest, TargetStartByte: 0, TargetEndByte: 1})
	}
	for i := 0; i < 576; i++ {
		closures = append(closures, closureCandidate{QueryID: queries[i%len(queries)], TargetParentID: fmt.Sprintf("p%03d", i%250)})
	}
	if err := writeReviewJSON(filepath.Join(completion, "run-manifest.json"), map[string]string{"kind": ReviewAcceptedCompletionKind, "label_loading": "LABEL_FIELDS_NOT_DECODED_STAGE_A"}); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"input-artifact-binding.json", "aggregate-relation-metrics.json", "cohort-language-report.json", "first-loss-report.json"} {
		if err := writeReviewJSON(filepath.Join(completion, file), map[string]string{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(completion, "report.md"), []byte("review"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, output := range []struct {
		name string
		rows []any
	}{{"semantic-parent-scores.jsonl", nil}, {"relation-endpoint-features.jsonl", features}, {"contract-closure-candidates.jsonl", closures}, {"relation-hints.jsonl", hints}, {"semantic-admission-results.jsonl", nil}, {"closure-package-results.jsonl", nil}, {"per-query-relation-trace.jsonl", traces}} {
		if err := writeJSONL(filepath.Join(completion, output.name), output.rows); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeChecksums(completion); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "review")
	contract := ReviewSeriesContract{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_series.v1", SeriesID: "relation-calibration-review-series-v1", QueryCount: 40, Endpoints: 250, Hints: 289, Closures: 576}
	if _, err := PrepareReview(ReviewPrepareRequest{Contract: contract, Completions: []ReviewCompletionInput{{Directory: completion, Bodies: bodies, Queries: queryTopology}}, OutputDir: output}); err == nil {
		t.Fatal("accepted an incomplete one-member immutable review series")
	}
}

func TestReviewPacketBlindnessAndHintControl(t *testing.T) {
	packet := ReviewPacket{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_packet.v1", Policy: ReviewPolicyID, PreparedDigest: completionTestDigest, CanonicalUniverseDigest: completionTestDigest, PassID: "pass-1", Queries: []ReviewPacketQuery{{QueryID: "q", Text: "question", Language: "go", AnswerMode: "SINGLE", UnadoptedRequiredGroupIDs: []string{"g"}}}, Relations: []ReviewPacketRelation{{AttachmentID: "r", QueryID: "q", SourceAttachmentID: "a", TargetAttachmentID: "b", RelationIDs: []string{"opaque"}, RelationKind: "CALL", Direction: "OUTGOING", StructuralTier: "LOCAL", Role: "USE", OccurrenceCount: 1}}}
	encoded, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"delivery_family", "cohorts", "rank", "score", "cell", "outcome", "label"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("blind packet leaked %q: %s", forbidden, encoded)
		}
	}
	ids := map[string]bool{"q": true}
	controls := reviewEmissionControls(nil, []relationHint{{QueryID: "q", OmissionStatus: "CANDIDATE", CountBudgetEligible: []int{4, 8, 16, 32}, ByteBudgetEligible: []int{512, 1024, 2048, 4096}, SerializedBytes: 37}}, ids)
	found := false
	for _, control := range controls {
		if control.Cell.Family == "hint" && control.Cell.Count == 4 && control.Cell.Bytes == 512 {
			found = control.CandidateCount == 1 && control.EmittedCount == 1 && control.ActualBytes == 37
		}
	}
	if !found {
		t.Fatal("non-empty hint control was not frozen")
	}
}

func TestReviewDeliveryOutcomeRequiresSharedDirectGroup(t *testing.T) {
	useful, groups := reviewDeliveryOutcome(
		reviewLabel{AttachmentID: "parent", Grade: 2, GroupIDs: []string{"g-1"}},
		reviewLabel{AttachmentID: "relation", Grade: 2, GroupIDs: []string{"g-1"}},
	)
	if useful != "USEFUL" || !sameReviewGroups(groups, []string{"g-1"}) {
		t.Fatalf("shared grade-2 delivery=%q groups=%v", useful, groups)
	}
	for _, value := range []struct {
		parent   reviewLabel
		relation reviewLabel
		want     string
	}{
		{reviewLabel{Grade: 2, GroupIDs: []string{"g-1"}}, reviewLabel{Grade: 2, GroupIDs: []string{"g-2"}}, "NOISE"},
		{reviewLabel{Grade: 1}, reviewLabel{Grade: 2, GroupIDs: []string{"g-1"}}, "SUPPORT"},
		{reviewLabel{Grade: 0, HardNegative: true, HardNegativeGroupIDs: []string{"g-1"}}, reviewLabel{Grade: 2, GroupIDs: []string{"g-1"}}, "HARD_NEGATIVE"},
	} {
		got, _ := reviewDeliveryOutcome(value.parent, value.relation)
		if got != value.want {
			t.Fatalf("delivery=%q want=%q", got, value.want)
		}
	}
}

func TestReviewParentHardNegativeIsNotNoise(t *testing.T) {
	if got := reviewParentOutcome(reviewLabel{Grade: 0, HardNegative: true, HardNegativeGroupIDs: []string{"g"}}); got != "HARD_NEGATIVE" {
		t.Fatalf("hard-negative parent outcome=%q", got)
	}
	if got := reviewParentOutcome(reviewLabel{Grade: 0}); got != "NOISE" {
		t.Fatalf("ordinary grade-zero parent outcome=%q", got)
	}
	row := reviewPolicyQueryCell{QueryID: "q", CorpusID: "corpus", Language: "go", Cohorts: []string{"relation"}, Cell: ReviewBudgetCell{Family: "closure", Count: 1, Bytes: 1024}, CandidateCount: 1, EmittedCount: 1, ParentAttachments: 1, ParentHardNeg: 1, OmissionCounts: map[string]int{}}
	aggregates := aggregateReviewPolicyCells([]reviewPolicyQueryCell{row})
	if err := validateReviewPolicyDenominators([]reviewPolicyQueryCell{row}, aggregates); err != nil {
		t.Fatal(err)
	}
	for _, aggregate := range aggregates {
		if aggregate.ScopeType == "global" && (aggregate.ParentHardNeg != 1 || aggregate.ParentNoise != 0) {
			t.Fatalf("parent hard-negative aggregate=%+v", aggregate)
		}
	}
}
