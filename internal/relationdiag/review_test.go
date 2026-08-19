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
	prepared := reviewPrepared{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_prepared.v1", Policy: ReviewPolicyID, Contract: contract, SemanticStatus: ReviewSemanticStatus, ClosureCells: reviewClosureCells(), HintCells: reviewHintCells(), Universe: []ReviewAttachment{{AttachmentID: "a", QueryID: "q", Path: "a.go", IndexedSHA256: completionTestDigest, StartByte: 0, EndByte: 4, Body: "body"}}, Candidates: []reviewCandidate{{AttachmentID: "a", QueryID: "q", RequiredGroupIDs: []string{"g-1"}}}}
	for i := 0; i < 40; i++ {
		prepared.Queries = append(prepared.Queries, reviewQueryRecord{Packet: ReviewPacketQuery{QueryID: fmt.Sprintf("q%02d", i), Text: "q", Language: "go", AnswerMode: "SINGLE"}})
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
	pass := ReviewPass{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_pass.v1", PreparedDigest: digest, PacketDigest: packetDigest, PassID: "pass-1", ReviewerID: "reviewer-a", ModelFamily: "model-a", SourceVerified: true, BlindnessAttestation: reviewBlindnessAttestation, Grades: []ReviewGrade{{AttachmentID: "a", Grade: 2, RequiredGroupIDs: []string{"g-1"}, Rationale: "supported"}}}
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
	second := ReviewPass{SchemaVersion: 1, Kind: "cidx.relation_calibration.review_pass.v1", PreparedDigest: digest, PacketDigest: secondDigest, PassID: "pass-2", ReviewerID: "reviewer-b", ModelFamily: "model-b", SourceVerified: true, BlindnessAttestation: reviewBlindnessAttestation, Grades: []ReviewGrade{{AttachmentID: "a", Grade: 0, HardNegative: true, HardNegativeGroupIDs: []string{"g-1"}, HardNegativeReason: "misleading", Rationale: "not useful"}}}
	frozenDir := filepath.Join(root, "frozen")
	if _, err := PrepareReviewAdoption(root, frozenDir, pass, second); err == nil {
		t.Fatal("previously blessed grade-2 conflict was accepted")
	}
	second.Grades[0] = ReviewGrade{AttachmentID: "a", Grade: 2, RequiredGroupIDs: []string{"g-1"}, Rationale: "supported"}
	if err := writeReviewJSON(filepath.Join(root, "pass-2-packet.json"), packetTwo); err != nil {
		t.Fatal(err)
	}
	expected, err := PrepareReviewAdoption(root, frozenDir, pass, second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := FreezeReview(root, frozenDir, pass, second, ReviewAdoption{SchemaVersion: 1, Kind: "cidx.relation_calibration.owner_adoption.v1", FrozenDigest: expected, Adopted: true, ProtocolVersion: "owner-adopted-dual-ai-v1", RelevanceAuthority: "OWNER_ADOPTED_DUAL_AI_REVIEW", ReviewValidation: "NO_INDEPENDENT_HUMAN_REVIEW", Overrides: []string{}}); err != nil {
		t.Fatal(err)
	}
	if err := SelectReview(root, frozenDir, filepath.Join(root, "selected")); err != nil {
		t.Fatal(err)
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
