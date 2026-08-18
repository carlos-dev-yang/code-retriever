package relationdiag

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"cidx/internal/eval"
	"cidx/internal/search"
	"cidx/internal/store"
)

const completionTestDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestCompletionCollapsedTopKAllowsOnlyPortableTieOrderVariation(t *testing.T) {
	values := []semanticParentScore{
		{ParentID: "a", NativeScore: 1, GlobalRank: 1, TieStartRank: 1, TieEndRank: 2},
		{ParentID: "b", NativeScore: 1, GlobalRank: 2, TieStartRank: 1, TieEndRank: 2},
	}
	value := 1.0
	row := completionCollapsedRow{}
	row.Ranking.Hits = []eval.RetrievalHit{{Path: "b.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.B", StartByte: 2, EndByte: 3, Rank: 1, Score: &value}, {Path: "a.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.A", StartByte: 0, EndByte: 1, Rank: 2, Score: &value}}
	byHit := map[string]string{hitKey("a.go", completionTestDigest, "p.A", 0, 1): "a", hitKey("b.go", completionTestDigest, "p.B", 2, 3): "b"}
	if err := validateCollapsedTopK(values, row, byHit); err != nil {
		t.Fatal(err)
	}
	value = 0.5
	row.Ranking.Hits[0].Score = &value
	if err := validateCollapsedTopK(values, row, byHit); err == nil {
		t.Fatal("expected score binding failure")
	}
}

func TestCompletionFinalFrontierEndpointsAndHintsAreDeduplicated(t *testing.T) {
	metadata := DefaultOccurrenceMetadata("a.go", 1)
	first := frontierEdge{CanonicalEdgeID: "edge-a", Candidate: anchorEdgeCandidate{Fact: Fact{RelationID: "r-a", Direction: Forward, AnchorID: "anchor", EndpointID: "endpoint", Kind: Calls, Metadata: metadata}, Stats: edgeStats{Tier: ExecutableDependencyTier, EdgeOccurrences: 2}}}
	second := frontierEdge{CanonicalEdgeID: "edge-b", Candidate: anchorEdgeCandidate{Fact: Fact{RelationID: "r-b", Direction: Forward, AnchorID: "anchor", EndpointID: "endpoint", Kind: Calls, Metadata: metadata}, Stats: edgeStats{Tier: ExecutableDependencyTier, EdgeOccurrences: 3}}}
	scores := map[string]semanticParentScore{"anchor": {ParentID: "anchor", NativeScore: 3, GlobalRank: 1, RankPercentileNumerator: 1, RankPercentileDenominator: 3}, "endpoint": {ParentID: "endpoint", NativeScore: 2, GlobalRank: 2, RankPercentileNumerator: 2, RankPercentileDenominator: 3}}
	endpointRows, err := semanticEndpointRows("q", []frontierEdge{first, second}, []string{"anchor"}, map[string]bool{"anchor": true, "endpoint": true}, scores, semanticDistribution{Span: 2})
	if err != nil || len(endpointRows) != 1 || len(endpointRows[0].SupportingRelationIDs) != 2 {
		t.Fatalf("rows=%+v err=%v", endpointRows, err)
	}
	parents := map[string]Parent{"anchor": {ID: "anchor", Path: "a.go", IndexedSHA256: completionTestDigest}, "endpoint": {ID: "endpoint", Path: "b.go", IndexedSHA256: completionTestDigest, Symbol: "E", QualifiedSymbol: "p.E"}}
	hints, err := hintRowsForQuery("q", []frontierEdge{first, second}, []string{"anchor"}, map[string]bool{"anchor": true, "endpoint": true}, scores, parents, map[string]completionParentLines{})
	if err != nil || len(hints) != 1 || hints[0].OccurrenceCount != 5 || hints[0].SupportingViewCount != 2 || hints[0].DisclosureSchema != hintDisclosureSchemaID || hints[0].SerializedBytes == 0 || !validDigest(hints[0].DisclosureSHA256) {
		t.Fatalf("hints=%+v err=%v", hints, err)
	}
}

func TestCompletionClosureRetainsSignatureOmissionReasons(t *testing.T) {
	metadata := DefaultOccurrenceMetadata("a.go", 1)
	metadata.Zone, metadata.Role = SignatureZone, TypeReturnRole
	facts := []Fact{
		{RelationID: "self", Direction: Forward, AnchorID: "primary", EndpointID: "primary", Kind: TypeRef, Metadata: metadata},
		{RelationID: "test", Direction: Forward, AnchorID: "primary", EndpointID: "test-target", Kind: TypeRef, Metadata: metadata},
	}
	parents := map[string]Parent{"test-target": {ID: "test-target", FileRole: TestFileRole, SourceBody: "body"}}
	rows, err := closureRowsForQuery("q", facts, []string{"primary"}, parents)
	if err != nil || len(rows) != 2 || rows[0].OmissionReason != "SELF_CYCLE" || rows[1].OmissionReason != "TARGET_NOT_PRODUCTION" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}

func TestCompletionTieBoundaryIsExplicitRatherThanDerivedFromPortableOrder(t *testing.T) {
	value := semanticParentScore{ParentID: "endpoint", TieStartRank: 5, TieEndRank: 6}
	if got := tieBoundaryStatus(value, ProtectedPrimaryK, true, true); got != "TIE_SPANS_BOUNDARY" {
		t.Fatalf("primary tie status=%q", got)
	}
	value.TieStartRank, value.TieEndRank = 20, 21
	if got := tieBoundaryStatus(value, MaxDenseDepth, false, true); got != "TIE_SPANS_BOUNDARY" {
		t.Fatalf("dense tie status=%q", got)
	}
}

func TestCompletionUsesAuthoritativeCollapsedOrderForTopTwenty(t *testing.T) {
	value := 1.0
	row := completionCollapsedRow{}
	for index := 0; index < MaxDenseDepth; index++ {
		row.Ranking.Hits = append(row.Ranking.Hits, eval.RetrievalHit{Path: fmt.Sprintf("p%02d.go", MaxDenseDepth-index), IndexedSHA256: completionTestDigest, QualifiedSymbol: fmt.Sprintf("p.F%02d", MaxDenseDepth-index), StartByte: index, EndByte: index + 1, Rank: index + 1, Score: &value})
	}
	hits, err := completionRankHits(row)
	if err != nil || hits[0].Path != "p20.go" || hits[ProtectedPrimaryK-1].Path != "p16.go" {
		t.Fatalf("hits=%+v err=%v", hits[:ProtectedPrimaryK], err)
	}
}

func TestCompletionRequestRequiresFinalReproofSeam(t *testing.T) {
	request := CompletionRequest{RunID: "relation-completion-q", EvaluationRoot: "out", GraphDirectory: "graph", RetrievalDirectory: "retrieval", DatasetPath: "dataset", Parents: store.SemanticParentSnapshot{Generation: 1, ManifestSHA256: completionTestDigest}}
	if err := validateCompletionRequest(request); err == nil {
		t.Fatal("expected missing reproof rejection")
	}
	request.Reproof = func(context.Context) (store.SemanticParentSnapshot, error) { return request.Parents, nil }
	if err := validateCompletionRequest(request); err != nil {
		t.Fatal(err)
	}
}

func TestCompletionExperimentAllowsSeriesTotalBeyondArtifactQueryCount(t *testing.T) {
	valid := map[string]any{"experiment_series_id": "relation-series", "evidence_class": completionEvidenceClass, "promotion_eligible": false, "label_state": "DRAFT_TWO_PASS_PENDING", "query_execution_mode": "LIVE_ALL_QUERIES", "series_query_operations_planned": 32, "corpus_clean_verified": true, "fts_mode": "PRODUCTION", "build_info": map[string]string{"commit": completionTestDigest, "source_modified": "false"}, "evaluation_executable_sha256": completionTestDigest, "source_dimensions": 1024, "serving_dimensions": 1024, "storage_codec": "int8", "document_provider_operations": 0, "reused_query_vectors": 0, "reused_dense_rankings": 0, "query_vector_persisted": false, "query_vector_sha256_recorded": true, "queries": []map[string]any{{"query_id": "q1", "text_sha256": completionTestDigest, "serving_active_segment_observations": 1}, {"query_id": "q2", "text_sha256": completionTestDigest, "serving_active_segment_observations": 1}}}
	value, err := json.Marshal(valid)
	if err != nil || validateCompletionExperiment(value, []string{"q1", "q2"}) != nil {
		t.Fatalf("valid series header err=%v", err)
	}
	valid["series_query_operations_planned"] = 1
	value, _ = json.Marshal(valid)
	if err := validateCompletionExperiment(value, []string{"q1", "q2"}); err == nil {
		t.Fatal("expected insufficient series total rejection")
	}
}

func TestCompletionQueryTextBindingRejectsMismatch(t *testing.T) {
	header := completionExperimentHeader{}
	header.Queries = append(header.Queries, struct {
		QueryID                          string `json:"query_id"`
		TextSHA256                       string `json:"text_sha256"`
		ServingActiveSegmentObservations int    `json:"serving_active_segment_observations"`
	}{QueryID: "q1", TextSHA256: completionTestDigest, ServingActiveSegmentObservations: 1})
	if err := validateCompletionQueryTextBinding(header, map[string]string{"q1": completionTestDigest}); err != nil {
		t.Fatal(err)
	}
	if err := validateCompletionQueryTextBinding(header, map[string]string{"q1": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}); err == nil {
		t.Fatal("expected query text hash mismatch rejection")
	}
}

func TestCompletionCanonicalInputUniverseAllowsRepeatedOccurrenceButRejectsMissingKey(t *testing.T) {
	first := search.EvaluationSegmentHit{CanonicalInputSHA256: completionTestDigest}
	second := first
	second.StartByte, second.EndByte = 10, 20
	if err := validateCompletionCanonicalInputUniverse([]search.EvaluationSegmentHit{first, second}, 1); err != nil {
		t.Fatal(err)
	}
	if err := validateCompletionCanonicalInputUniverse([]search.EvaluationSegmentHit{first, second}, 2); err == nil {
		t.Fatal("expected missing distinct canonical input rejection")
	}
}

func TestCompletionSegmentObservationIdentityKeepsDistinctSpans(t *testing.T) {
	first := search.EvaluationSegmentHit{CanonicalInputSHA256: completionTestDigest, Path: "p.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.F", ParentStartByte: 0, ParentEndByte: 20, StartByte: 0, EndByte: 10}
	second := first
	second.StartByte, second.EndByte = 10, 20
	if completionSegmentObservationKey(first) == completionSegmentObservationKey(second) {
		t.Fatal("distinct spans collapsed despite matching canonical input")
	}
	if completionSegmentObservationKey(first) != completionSegmentObservationKey(first) {
		t.Fatal("observation identity is not deterministic")
	}
}
