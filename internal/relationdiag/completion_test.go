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

func TestCompletionCollapsedRowConsumesNestedVariantAndBindsIt(t *testing.T) {
	const rowJSON = `{"query_id":"q1","variant":"serving_active_codec","ranking":{"query_id":"q1","variant":"serving_active_codec","query_vector_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","hits":[]},"failure_stage":""}`
	var row completionCollapsedRow
	if err := strictJSON([]byte(rowJSON), &row); err != nil {
		t.Fatalf("strict artifact decode: %v", err)
	}
	if err := validateCollapsedRowWire(row); err != nil {
		t.Fatalf("variant binding: %v", err)
	}
	row.Ranking.Variant = "target_f32"
	if err := validateCollapsedRowWire(row); err == nil {
		t.Fatal("expected nested/top-level variant mismatch rejection")
	}
}

func TestCompletionBindsActiveSegmentAndCollapsedQueryDigest(t *testing.T) {
	segments := completionSegmentRow{QueryID: "q1", Variant: "serving_active_codec", QueryVectorSHA256: completionTestDigest, Segments: []search.EvaluationSegmentHit{{Rank: 1}}}
	collapsed := completionCollapsedRow{QueryID: "q1"}
	collapsed.Ranking.QueryVectorSHA256 = completionTestDigest
	if err := validateActiveInt8QueryVectorBinding(segments, collapsed); err != nil {
		t.Fatal(err)
	}
	segments.QueryVectorSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := validateActiveInt8QueryVectorBinding(segments, collapsed); err == nil {
		t.Fatal("expected segment/collapsed query digest mismatch rejection")
	}
}

func TestCompletionCollapseUsesProducerObservedSegmentRank(t *testing.T) {
	parentA := Parent{ID: "a", Path: "a.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.A", StartByte: 0, EndByte: 40}
	parentB := Parent{ID: "b", Path: "b.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.B", StartByte: 0, EndByte: 40}
	segments := []search.EvaluationSegmentHit{
		{CanonicalInputSHA256: completionTestDigest, Path: parentA.Path, IndexedSHA256: parentA.IndexedSHA256, QualifiedSymbol: parentA.QualifiedSymbol, ParentStartByte: parentA.StartByte, ParentEndByte: parentA.EndByte, StartByte: 10, EndByte: 20, Rank: 4, Score: 1},
		{CanonicalInputSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Path: parentA.Path, IndexedSHA256: parentA.IndexedSHA256, QualifiedSymbol: parentA.QualifiedSymbol, ParentStartByte: parentA.StartByte, ParentEndByte: parentA.EndByte, StartByte: 20, EndByte: 30, Rank: 2, Score: 1},
		{CanonicalInputSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Path: parentB.Path, IndexedSHA256: parentB.IndexedSHA256, QualifiedSymbol: parentB.QualifiedSymbol, ParentStartByte: parentB.StartByte, ParentEndByte: parentB.EndByte, StartByte: 0, EndByte: 10, Rank: 1, Score: 1},
	}
	byHit := map[string]string{
		hitKey(parentA.Path, parentA.IndexedSHA256, parentA.QualifiedSymbol, parentA.StartByte, parentA.EndByte): parentA.ID,
		hitKey(parentB.Path, parentB.IndexedSHA256, parentB.QualifiedSymbol, parentB.StartByte, parentB.EndByte): parentB.ID,
	}
	values, err := collapseActiveScores("q1", segments, byHit, map[string]Parent{parentA.ID: parentA, parentB.ID: parentB})
	if err != nil || len(values) != 2 || values[0].ParentID != parentB.ID || values[1].ParentID != parentA.ID || values[1].WinningSegment.Rank != 2 {
		t.Fatalf("values=%+v err=%v", values, err)
	}
}

func TestCompletionCollapsedTopKRequiresExactProducerOrder(t *testing.T) {
	values := []semanticParentScore{
		{ParentID: "a", Path: "a.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.A", StartByte: 0, EndByte: 1, NativeScore: 1, GlobalRank: 1, TieStartRank: 1, TieEndRank: 2, WinningSegment: search.EvaluationSegmentHit{Rank: 1}},
		{ParentID: "b", Path: "b.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.B", StartByte: 2, EndByte: 3, NativeScore: 1, GlobalRank: 2, TieStartRank: 1, TieEndRank: 2, WinningSegment: search.EvaluationSegmentHit{Rank: 2}},
		{ParentID: "c", Path: "c.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.C", StartByte: 4, EndByte: 5, NativeScore: 0.8, GlobalRank: 3, TieStartRank: 3, TieEndRank: 3, WinningSegment: search.EvaluationSegmentHit{Rank: 3}},
		{ParentID: "d", Path: "d.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.D", StartByte: 6, EndByte: 7, NativeScore: 0.7, GlobalRank: 4, TieStartRank: 4, TieEndRank: 4, WinningSegment: search.EvaluationSegmentHit{Rank: 4}},
		{ParentID: "e", Path: "e.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.E", StartByte: 8, EndByte: 9, NativeScore: 0.6, GlobalRank: 5, TieStartRank: 5, TieEndRank: 5, WinningSegment: search.EvaluationSegmentHit{Rank: 5}},
	}
	value, c, d, e := 1.0, 0.8, 0.7, 0.6
	row := completionCollapsedRow{}
	row.Ranking.Hits = []eval.RetrievalHit{
		{Path: "a.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.A", StartByte: 0, EndByte: 1, Rank: 1, Score: &value},
		{Path: "b.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.B", StartByte: 2, EndByte: 3, Rank: 2, Score: &value},
		{Path: "c.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.C", StartByte: 4, EndByte: 5, Rank: 3, Score: &c},
		{Path: "d.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.D", StartByte: 6, EndByte: 7, Rank: 4, Score: &d},
		{Path: "e.go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p.E", StartByte: 8, EndByte: 9, Rank: 5, Score: &e},
	}
	byHit := map[string]string{
		hitKey("a.go", completionTestDigest, "p.A", 0, 1): "a",
		hitKey("b.go", completionTestDigest, "p.B", 2, 3): "b",
		hitKey("c.go", completionTestDigest, "p.C", 4, 5): "c",
		hitKey("d.go", completionTestDigest, "p.D", 6, 7): "d",
		hitKey("e.go", completionTestDigest, "p.E", 8, 9): "e",
	}
	if err := validateCollapsedTopK(values, row, byHit); err != nil {
		t.Fatal(err)
	}
	row.Ranking.Hits[0], row.Ranking.Hits[1] = row.Ranking.Hits[1], row.Ranking.Hits[0]
	row.Ranking.Hits[0].Rank, row.Ranking.Hits[1].Rank = 1, 2
	if err := validateCollapsedTopK(values, row, byHit); err == nil {
		t.Fatal("expected tied top-five order rejection")
	}
	row.Ranking.Hits[0], row.Ranking.Hits[1] = row.Ranking.Hits[1], row.Ranking.Hits[0]
	row.Ranking.Hits[0].Rank, row.Ranking.Hits[1].Rank = 1, 2
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

func TestCompletionDerivesTopTwentyAfterAuthoritativeProductTopFive(t *testing.T) {
	row := completionCollapsedRow{}
	values, byHit := make([]semanticParentScore, 0, MaxDenseDepth+2), map[string]string{}
	for index := 0; index < MaxDenseDepth+2; index++ {
		id := fmt.Sprintf("p%02d", index)
		score := float64(MaxDenseDepth - index)
		if index < ProtectedPrimaryK {
			score = float64(MaxDenseDepth)
		}
		value := semanticParentScore{ParentID: id, Path: id + ".go", IndexedSHA256: completionTestDigest, QualifiedSymbol: "p." + id, StartByte: index * 2, EndByte: index*2 + 1, NativeScore: score, GlobalRank: index + 1, TieStartRank: index + 1, TieEndRank: index + 1, WinningSegment: search.EvaluationSegmentHit{Rank: index + 1}}
		if index < ProtectedPrimaryK {
			value.TieStartRank, value.TieEndRank = 1, ProtectedPrimaryK
		}
		values = append(values, value)
		byHit[hitKey(value.Path, value.IndexedSHA256, value.QualifiedSymbol, value.StartByte, value.EndByte)] = id
	}
	for rank := 1; rank <= ProtectedPrimaryK; rank++ {
		value := values[rank-1]
		score := value.NativeScore
		row.Ranking.Hits = append(row.Ranking.Hits, eval.RetrievalHit{Path: value.Path, IndexedSHA256: value.IndexedSHA256, QualifiedSymbol: value.QualifiedSymbol, StartByte: value.StartByte, EndByte: value.EndByte, Rank: rank, Score: &score})
	}
	hits, err := completionRankHits(row, values, byHit)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != MaxDenseDepth || hits[0].Path != "p00.go" || hits[ProtectedPrimaryK-1].Path != "p04.go" || hits[ProtectedPrimaryK].Path != "p05.go" || hits[MaxDenseDepth-1].Path != "p19.go" {
		t.Fatalf("hits=%+v", hits)
	}
	seen := map[string]bool{}
	for index, hit := range hits {
		if hit.Rank != index+1 || seen[hit.Path] {
			t.Fatalf("reconstructed hit[%d]=%+v", index, hit)
		}
		seen[hit.Path] = true
	}
	replacement := values[ProtectedPrimaryK]
	replacementScore := replacement.NativeScore
	row.Ranking.Hits[ProtectedPrimaryK-1] = eval.RetrievalHit{Path: replacement.Path, IndexedSHA256: replacement.IndexedSHA256, QualifiedSymbol: replacement.QualifiedSymbol, StartByte: replacement.StartByte, EndByte: replacement.EndByte, Rank: ProtectedPrimaryK, Score: &replacementScore}
	if _, err := completionRankHits(row, values, byHit); err == nil {
		t.Fatal("expected fifth/sixth parent substitution rejection")
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
	second = first
	second.Rank = first.Rank + 1
	if completionSegmentObservationKey(first) != completionSegmentObservationKey(second) {
		t.Fatal("producer rank changed occurrence identity")
	}
}
