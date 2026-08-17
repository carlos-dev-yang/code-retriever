package relationdiag

import "testing"

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
