package relationdiag

import (
	"fmt"
	"testing"
)

func TestSelectBundleIsLabelBlindDeterministicAndCapsAddedParents(t *testing.T) {
	facts := []Fact{
		{RelationID: "z", Direction: Forward, AnchorID: "anchor-rank-one", EndpointID: "endpoint", Kind: Calls, OccurrenceByte: 8},
		{RelationID: "a", Direction: Reverse, AnchorID: "anchor-rank-eight", EndpointID: "required-target", Kind: TypeRef, OccurrenceByte: 40},
	}
	ranks := map[string]int{"anchor-rank-one": 1, "anchor-rank-eight": 8}
	first := selectBundle("query", append([]Fact(nil), facts...), ranks)
	second := selectBundle("query", append([]Fact(nil), facts...), ranks)
	if first.Selected == nil || first.Selected.RelationID != "a" || first.Selected.Kind != TypeRef {
		t.Fatalf("selected=%+v", first.Selected)
	}
	if len(first.AddedParentIDs) != 2 || first.AddedParentIDs[0] != "anchor-rank-eight" || first.AddedParentIDs[1] != "required-target" {
		t.Fatalf("added=%v", first.AddedParentIDs)
	}
	if first.SelectionPolicy != SelectionPolicyID || first.QueryID != second.QueryID || first.Selected.RelationID != second.Selected.RelationID {
		t.Fatalf("selection was not deterministic")
	}
}

func TestPrimaryTop5SeparatesProtectedParentsFromTopTwentySeeds(t *testing.T) {
	byHit := map[string]string{}
	hits := make([]rankHit, 0, MaxDenseDepth)
	for index := 0; index < MaxDenseDepth; index++ {
		hit := rankHit{Path: "a.go", IndexedSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", QualifiedSymbol: "p.F", StartByte: index * 2, EndByte: index*2 + 1, Rank: index + 1}
		hits = append(hits, hit)
		byHit[hitKey(hit.Path, hit.IndexedSHA256, hit.QualifiedSymbol, hit.StartByte, hit.EndByte)] = string(rune('a' + index))
	}
	primary, primaryIDs, seeds, err := primaryTop5(hits, byHit)
	if err != nil || len(primary) != 5 || len(primaryIDs) != 5 || len(seeds) != 20 || primaryIDs[4] == seeds[7] {
		t.Fatalf("primary=%d primaryIDs=%d seeds=%d err=%v", len(primary), len(primaryIDs), len(seeds), err)
	}
}

func TestDiagnosticGateReportsUniqueHardNegativeAndWalkXFF(t *testing.T) {
	parents := map[string]Parent{
		"negative": {ID: "negative", QualifiedSymbol: "pkg.Negative"},
		"walk":     {ID: "walk", QualifiedSymbol: "middleware.walkXFF"},
	}
	traces := []queryTrace{
		{QueryID: "q1", Attachments: []ParentAttachment{{ParentID: "negative", Classification: "HARD_NEGATIVE"}, {ParentID: "walk", Classification: "UNREVIEWED"}}},
		{QueryID: "q2", Attachments: []ParentAttachment{{ParentID: "negative", Classification: "HARD_NEGATIVE"}}},
	}
	gate := diagnosticGate(traces, parents)
	if gate.Eligible || len(gate.Reasons) != 2 {
		t.Fatalf("gate=%+v", gate)
	}
	for _, reason := range gate.Reasons {
		if reason.ParentID == "negative" && len(reason.QueryIDs) != 2 {
			t.Fatalf("hard-negative queries=%v", reason.QueryIDs)
		}
	}
}

func TestFrontierCapAndBridgeArmsShareFrontierAndRespectBounds(t *testing.T) {
	group, facts, stats := frontierFixture(true)
	feature, ranks := queryFeatures{Direction: Forward}, map[string]int{}
	first, err := buildFrontierTrace(group, feature, facts, stats, ranks)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildFrontierTrace(group, feature, facts, stats, ranks)
	if err != nil || first.FinalDigest != second.FinalDigest {
		t.Fatalf("frontier digest is not deterministic: %q %q %v", first.FinalDigest, second.FinalDigest, err)
	}
	if len(first.FinalFrontier) > FrontierGlobalLimit || first.Counts.CanonicalUnionEdges != FrontierGlobalLimit || first.Counts.CrossBucketDuplicates != 0 || first.Counts.RawDirectionalFacts != 35 || first.Counts.SelfFactsRemoved != 1 || first.Counts.NonSelfDirectionalFacts != 34 || first.Counts.BucketDistinctCanonicalViews != 33 || first.Counts.RepeatedOccurrenceCollapse != 1 || first.Counts.GlobalCanonicalUniverseEdges != 33 || first.Counts.UniverseCrossBucketDuplicateViews != 0 || first.Counts.BucketTruncations != 1 || first.Counts.ProvisionalRows != FrontierGlobalLimit || first.Counts.FinalRetained != FrontierGlobalLimit {
		t.Fatalf("frontier bounds/counts=%+v", first.Counts)
	}
	bridgeBucket := frontierBucketKey(frontierBucket{AnchorOrdinal: 1, Direction: Forward, Tier: ExecutableDependencyTier})
	if counts := first.BucketCounts[bridgeBucket]; counts.PreCapDistinct != 3 || counts.Retained != 2 || counts.Truncated != 1 {
		t.Fatalf("bridge bucket counts=%+v", counts)
	}
	bucketCounts := map[string]int{}
	for _, edge := range first.Provisional {
		bucketCounts[frontierBucketKey(edge.Bucket)]++
	}
	for bucket, count := range bucketCounts {
		if count > FrontierBucketLimit {
			t.Fatalf("bucket %q retained %d", bucket, count)
		}
	}
	for bucket, counts := range first.BucketCounts {
		if counts.Retained > FrontierBucketLimit || counts.Truncated != counts.PreCapDistinct-counts.Retained {
			t.Fatalf("bucket %q accounting=%+v", bucket, counts)
		}
	}
	expectedDisplaced := storedEdgeKey("a", "endpoint-a-0-0-1", Calls, ExecutableDependencyTier)
	if !containsDirectFrontier(first.CanonicalUnion) || !containsDirectFrontier(first.FinalFrontier) || len(first.BridgeReservations) != 1 || first.BridgeReservations[0].Outcome != "RESERVED" || first.BridgeReservations[0].DisplacedCanonicalEdgeID != expectedDisplaced || len(first.BridgeDisplacements) != 1 || first.BridgeDisplacements[0].DisplacedCanonicalEdgeID != expectedDisplaced {
		t.Fatalf("bridge reservation=%+v union=%v final=%v", first.BridgeReservations, frontierIDs(first.CanonicalUnion), frontierIDs(first.FinalFrontier))
	}
	if containsFrontier(first.CanonicalUnion, expectedDisplaced) {
		t.Fatalf("canonical union backfilled displaced edge %q", expectedDisplaced)
	}
	cap, capTrace, err := selectFrontierBundle("q", feature, group, facts, stats, ranks, nil, nil, AnchorFrontierCapOnlyPolicyID)
	if err != nil || cap.Selected == nil {
		t.Fatalf("cap selection=%+v trace=%+v err=%v", cap, capTrace, err)
	}
	bridge, bridgeTrace, err := selectFrontierBundle("q", feature, group, facts, stats, ranks, nil, nil, AnchorFrontierBridgePolicyID)
	if err != nil || bridge.Selected == nil || bridge.Selected.AnchorID == bridge.Selected.EndpointID || bridgeTrace.Selected == nil || !bridgeTrace.Selected.DirectBridge || capTrace.FinalDigest != bridgeTrace.FinalDigest {
		t.Fatalf("bridge selection=%+v trace=%+v capDigest=%q bridgeDigest=%q err=%v", bridge, bridgeTrace, capTrace.FinalDigest, bridgeTrace.FinalDigest, err)
	}
}

func TestFrontierBridgeAbstainsWithoutDirectBridgeAndOverflowAbstains(t *testing.T) {
	group, facts, stats := frontierFixture(false)
	feature, ranks := queryFeatures{Direction: Forward}, map[string]int{}
	cap, _, err := selectFrontierBundle("q", feature, group, facts, stats, ranks, nil, nil, AnchorFrontierCapOnlyPolicyID)
	if err != nil || cap.Selected == nil {
		t.Fatalf("cap-only should retain an ordinary edge: %+v %v", cap, err)
	}
	bridge, trace, err := selectFrontierBundle("q", feature, group, facts, stats, ranks, nil, nil, AnchorFrontierBridgePolicyID)
	if err != nil || bridge.Selected != nil || trace.AbstentionReason != "NO_DIRECT_ANCHOR_BRIDGE" {
		t.Fatalf("bridge abstention bundle=%+v trace=%+v err=%v", bridge, trace, err)
	}
	bucket := frontierBucket{AnchorOrdinal: 1, Direction: Forward, Tier: ExecutableDependencyTier}
	bucketKey := frontierBucketKey(bucket)
	surviving := frontierEdge{CanonicalEdgeID: "surviving", Bucket: bucket, DirectBridge: true, Candidate: frontierTestCandidate("surviving", "a", "b")}
	reservations, _, overflow := assignFrontierBridges(map[string][]frontierEdge{bucketKey: {surviving}}, map[string][]frontierEdge{bucketKey: {surviving}}, map[string]int{})
	if overflow || len(reservations) != 1 || reservations[0].Outcome != "SURVIVED" {
		t.Fatalf("surviving bridge=%+v overflow=%v", reservations, overflow)
	}
	provisional := map[string][]frontierEdge{bucketKey: {frontierEdge{CanonicalEdgeID: "ordinary-a", Bucket: bucket, Candidate: frontierTestCandidate("ordinary-a", "a", "x")}, frontierEdge{CanonicalEdgeID: "ordinary-b", Bucket: bucket, Candidate: frontierTestCandidate("ordinary-b", "a", "y")}}}
	eligible := map[string][]frontierEdge{bucketKey: provisional[bucketKey]}
	for index := 0; index < 3; index++ {
		eligible[bucketKey] = append(eligible[bucketKey], frontierEdge{CanonicalEdgeID: fmt.Sprintf("bridge-%02d", index), Bucket: bucket, DirectBridge: true, Candidate: frontierTestCandidate(fmt.Sprintf("bridge-%02d", index), "a", "b")})
	}
	_, _, overflow = assignFrontierBridges(provisional, eligible, map[string]int{})
	if !overflow {
		t.Fatal("bridge assignment overflow was not reported")
	}
}

func TestFrontierCanonicalDuplicateUnionDoesNotBackfill(t *testing.T) {
	group := anchorGroup{Anchors: []anchorSelection{{Ordinal: 1, ParentID: "a"}, {Ordinal: 2, ParentID: "b"}}}
	facts := []Fact{
		frontierTestFact("aa-direct-forward", "a", "b", Forward, Calls, BodyZone, 1),
		frontierTestFact("ab-ordinary", "a", "x", Forward, Calls, BodyZone, 2),
		frontierTestFact("ac-backfill-forbidden", "a", "y", Forward, Calls, BodyZone, 3),
		frontierTestFact("ad-direct-reverse", "a", "b", Reverse, Calls, BodyZone, 4),
	}
	stats := frontierTestStats(facts)
	directKey := storedEdgeKey("a", "b", Calls, ExecutableDependencyTier)
	directStat := stats[directKey]
	directStat.Best = facts[0]
	stats[directKey] = directStat
	trace, err := buildFrontierTrace(group, queryFeatures{Direction: Forward}, facts, stats, map[string]int{})
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range trace.Provisional {
		count := 0
		for _, candidate := range trace.Provisional {
			if frontierBucketKey(candidate.Bucket) == frontierBucketKey(edge.Bucket) {
				count++
			}
		}
		if count > FrontierBucketLimit {
			t.Fatalf("bucket %q has %d", frontierBucketKey(edge.Bucket), count)
		}
	}
	forbidden := storedEdgeKey("a", "y", Calls, ExecutableDependencyTier)
	if trace.Counts.ProvisionalBucketEdges != 3 || trace.Counts.CanonicalUnionEdges != 2 || trace.Counts.CrossBucketDuplicates != 1 || containsFrontier(trace.CanonicalUnion, forbidden) {
		t.Fatalf("union=%+v ids=%v", trace.Counts, frontierIDs(trace.CanonicalUnion))
	}
}

func TestFrontierGraphOnlyParetoUsesOnlyFinalFrontier(t *testing.T) {
	t.Run("direct bridge takes precedence in final frontier order", func(t *testing.T) {
		final := []frontierEdge{
			graphOnlyTestEdge("ordinary", ExecutableDependencyTier, Forward, "outside", false, 1, 1, 1, 1),
			graphOnlyTestEdge("bridge-first", ExecutableDependencyTier, Reverse, "anchor-a", true, 1, 1, 1, 1),
			graphOnlyTestEdge("bridge-second", ExecutableDependencyTier, Forward, "anchor-b", true, 1, 1, 1, 1),
			graphOnlyTestEdge("incoming", BodyReferenceTier, Reverse, "outside-incoming", false, 1, 1, 1, 1),
			graphOnlyTestEdge("dense", BodyReferenceTier, Forward, "dense-parent", false, 1, 1, 1, 1),
		}
		decision, selected, err := frontierGraphOnlyDecision(final, map[string]int{"dense-parent": 1})
		if err != nil || selected == nil || selected.CanonicalEdgeID != "bridge-first" || decision.Outcome != "DIRECT_BRIDGE" || decision.DirectBridgeCandidates != 2 || decision.IncomingExcluded != 1 || decision.DenseEndpointExcluded != 1 || decision.GraphOnlyCandidates != 1 || decision.UnionCount != 1 || len(decision.Tiers[ExecutableDependencyTier].Candidates) != 1 {
			t.Fatalf("decision=%+v selected=%+v err=%v", decision, selected, err)
		}
		if err := validateFrontierGraphOnlyDecision(decision, final, map[string]int{"dense-parent": 1}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("incoming and dense endpoints are excluded before pareto", func(t *testing.T) {
		final := []frontierEdge{
			graphOnlyTestEdge("incoming", ExecutableDependencyTier, Reverse, "outside-incoming", false, 1, 1, 1, 1),
			graphOnlyTestEdge("dense", ExecutableDependencyTier, Forward, "dense-parent", false, 1, 1, 1, 1),
			graphOnlyTestEdge("winner", ExecutableDependencyTier, Forward, "outside", false, 2, 3, 1, 1),
		}
		decision, selected, err := frontierGraphOnlyDecision(final, map[string]int{"dense-parent": 7})
		if err != nil || selected == nil || selected.CanonicalEdgeID != "winner" || decision.Outcome != "ONE_WINNER" || decision.IncomingExcluded != 1 || decision.DenseEndpointExcluded != 1 || decision.GraphOnlyCandidates != 1 || decision.UnionCount != 1 {
			t.Fatalf("decision=%+v selected=%+v err=%v", decision, selected, err)
		}
	})

	t.Run("exact rational equality does not dominate", func(t *testing.T) {
		final := []frontierEdge{
			graphOnlyTestEdge("one-half", ExecutableDependencyTier, Forward, "outside-a", false, 1, 2, 2, 2),
			graphOnlyTestEdge("two-fourths", ExecutableDependencyTier, Forward, "outside-b", false, 2, 4, 2, 2),
		}
		decision, selected, err := frontierGraphOnlyDecision(final, nil)
		if err != nil || selected != nil || decision.Outcome != "MULTIPLE_WINNERS" || decision.UnionCount != 2 {
			t.Fatalf("decision=%+v selected=%+v err=%v", decision, selected, err)
		}
		for _, candidate := range decision.Tiers[ExecutableDependencyTier].Candidates {
			if !candidate.Nondominated || len(candidate.DominatedBy) != 0 {
				t.Fatalf("equal candidate=%+v", candidate)
			}
		}
	})

	t.Run("dominance is exact and never crosses tiers", func(t *testing.T) {
		final := []frontierEdge{
			graphOnlyTestEdge("dominant", ExecutableDependencyTier, Forward, "outside-a", false, 2, 3, 1, 1),
			graphOnlyTestEdge("dominated", ExecutableDependencyTier, Forward, "outside-b", false, 1, 2, 2, 2),
			graphOnlyTestEdge("other-tier", BodyReferenceTier, Forward, "outside-c", false, 1, 9, 9, 9),
		}
		decision, selected, err := frontierGraphOnlyDecision(final, nil)
		if err != nil || selected != nil || decision.Outcome != "MULTIPLE_WINNERS" || decision.UnionCount != 2 {
			t.Fatalf("decision=%+v selected=%+v err=%v", decision, selected, err)
		}
		candidates := decision.Tiers[ExecutableDependencyTier].Candidates
		if len(candidates) != 2 || !candidates[0].Nondominated || candidates[1].Nondominated || len(candidates[1].DominatedBy) != 1 || candidates[1].DominatedBy[0] != "dominant" {
			t.Fatalf("tier candidates=%+v", candidates)
		}
	})

	t.Run("empty admission reports no candidate", func(t *testing.T) {
		final := []frontierEdge{
			graphOnlyTestEdge("incoming", ExecutableDependencyTier, Reverse, "outside", false, 1, 1, 1, 1),
			graphOnlyTestEdge("dense", ExecutableDependencyTier, Forward, "dense-parent", false, 1, 1, 1, 1),
		}
		decision, selected, err := frontierGraphOnlyDecision(final, map[string]int{"dense-parent": 1})
		if err != nil || selected != nil || decision.Outcome != "NO_CANDIDATE" || decision.UnionCount != 0 {
			t.Fatalf("decision=%+v selected=%+v err=%v", decision, selected, err)
		}
	})

	t.Run("selection preserves the shared frontier digest and caps", func(t *testing.T) {
		group, facts, stats := frontierFixture(false)
		feature, ranks := queryFeatures{Direction: Forward}, map[string]int{}
		base, err := buildFrontierTrace(group, feature, facts, stats, ranks)
		if err != nil {
			t.Fatal(err)
		}
		bundle, trace, err := selectFrontierBundle("q", feature, group, facts, stats, ranks, nil, nil, AnchorFrontierGraphOnlyParetoPolicyID)
		if err != nil || trace.FinalDigest != base.FinalDigest || len(trace.FinalFrontier) != len(base.FinalFrontier) || trace.Counts != base.Counts || len(trace.FinalFrontier) > FrontierGlobalLimit || bundle.Selected != nil || trace.GraphOnly == nil || trace.GraphOnly.Outcome != "MULTIPLE_WINNERS" {
			t.Fatalf("bundle=%+v trace=%+v base=%+v err=%v", bundle, trace, base, err)
		}
	})
}

func graphOnlyTestEdge(id string, tier StructuralTier, direction Direction, endpoint string, bridge bool, edgeOccurrences, sourceOccurrences, sourceTargets, targetSources int) frontierEdge {
	fact := Fact{RelationID: id, Direction: direction, AnchorID: "anchor", EndpointID: endpoint}
	stats := edgeStats{SourceID: "source-" + id, TargetID: endpoint, Kind: Calls, Tier: tier, EdgeOccurrences: edgeOccurrences, SourceStratumOccurrences: sourceOccurrences, SourceStratumDistinctTargets: sourceTargets, TargetIncomingStratumDistinctSources: targetSources}
	return frontierEdge{CanonicalEdgeID: id, Bucket: frontierBucket{AnchorOrdinal: 1, Direction: direction, Tier: tier}, Candidate: anchorEdgeCandidate{Fact: fact, AnchorOrdinal: 1, Stats: stats}, DirectBridge: bridge}
}

func frontierFixture(includeBridge bool) (anchorGroup, []Fact, map[string]edgeStats) {
	group := anchorGroup{Anchors: []anchorSelection{{Ordinal: 1, ParentID: "a"}, {Ordinal: 2, ParentID: "b"}}}
	tiers := []struct {
		kind RelationKind
		zone OccurrenceZone
	}{{Calls, BodyZone}, {TypeRef, BodyZone}, {TypeRef, SignatureZone}, {MemberOf, SignatureZone}}
	facts := make([]Fact, 0, 33)
	ordinal := 1
	for anchorIndex, anchor := range []string{"a", "b"} {
		for _, direction := range []Direction{Forward, Reverse} {
			for tierIndex, tier := range tiers {
				for copyIndex := 0; copyIndex < FrontierBucketLimit; copyIndex++ {
					id := fmt.Sprintf("%02d-%d-%d-%d-%d", anchorIndex, directionIndex(direction), tierIndex, copyIndex, ordinal)
					source, target := anchor, fmt.Sprintf("endpoint-%s-%d-%d-%d", anchor, directionIndex(direction), tierIndex, copyIndex)
					if direction == Reverse {
						source, target = target, anchor
					}
					facts = append(facts, frontierTestFact(id, source, target, direction, tier.kind, tier.zone, ordinal))
					ordinal++
				}
			}
		}
	}
	if includeBridge {
		facts = append(facts, frontierTestFact("zz-bridge", "a", "b", Forward, Calls, BodyZone, ordinal))
		ordinal++
		facts = append(facts, frontierTestFact("zz-bridge-repeat", "a", "b", Forward, Calls, BodyZone, ordinal))
		ordinal++
		facts = append(facts, frontierTestFact("self-removed", "a", "a", Forward, Calls, BodyZone, ordinal))
	}
	return group, facts, frontierTestStats(facts)
}

func directionIndex(direction Direction) int {
	if direction == Reverse {
		return 1
	}
	return 0
}

func frontierTestFact(id, source, target string, direction Direction, kind RelationKind, zone OccurrenceZone, ordinal int) Fact {
	metadata := DefaultOccurrenceMetadata("fixture.go", ordinal)
	metadata.Zone = zone
	fact := Fact{RelationID: id, Direction: direction, Kind: kind, OccurrencePath: "fixture.go", OccurrenceByte: ordinal, OccurrenceEndByte: ordinal + 1, Metadata: metadata}
	if direction == Forward {
		fact.AnchorID, fact.EndpointID = source, target
	} else {
		fact.AnchorID, fact.EndpointID = target, source
	}
	return fact
}

func frontierTestStats(facts []Fact) map[string]edgeStats {
	stats := map[string]edgeStats{}
	for _, fact := range facts {
		source, target := fact.AnchorID, fact.EndpointID
		if fact.Direction == Reverse {
			source, target = fact.EndpointID, fact.AnchorID
		}
		tier, err := structuralTier(fact.Kind, fact.Metadata)
		if err != nil {
			panic(err)
		}
		best := fact
		best.Direction, best.AnchorID, best.EndpointID = Forward, source, target
		stats[storedEdgeKey(source, target, fact.Kind, tier)] = edgeStats{SourceID: source, TargetID: target, Kind: fact.Kind, Tier: tier, EdgeOccurrences: 1, SourceStratumOccurrences: 1, SourceStratumDistinctTargets: 1, TargetIncomingStratumOccurrences: 1, TargetIncomingStratumDistinctSources: 1, Best: best}
	}
	return stats
}

func frontierTestCandidate(id, source, target string) anchorEdgeCandidate {
	fact := frontierTestFact(id, source, target, Forward, Calls, BodyZone, 1)
	stats := edgeStats{SourceID: source, TargetID: target, Kind: Calls, Tier: ExecutableDependencyTier, EdgeOccurrences: 1, SourceStratumOccurrences: 1, SourceStratumDistinctTargets: 1, TargetIncomingStratumOccurrences: 1, TargetIncomingStratumDistinctSources: 1, Best: fact}
	return anchorEdgeCandidate{Fact: fact, AnchorOrdinal: 1, Stats: stats, RankingTuple: anchorRankingTuple(anchorEdgeCandidate{Fact: fact, AnchorOrdinal: 1, Stats: stats}, nil, AnchorEdgeBidirectionalPolicyID)}
}

func containsDirectFrontier(values []frontierEdge) bool {
	for _, value := range values {
		if value.DirectBridge {
			return true
		}
	}
	return false
}
func containsFrontier(values []frontierEdge, wanted string) bool {
	for _, value := range values {
		if value.CanonicalEdgeID == wanted {
			return true
		}
	}
	return false
}
func frontierIDs(values []frontierEdge) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.CanonicalEdgeID)
	}
	return result
}
