package relationdiag

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalPackagingContractFingerprint(t *testing.T) {
	contract := canonicalPackagingContract()
	if err := validatePackagingContract(contract); err != nil {
		t.Fatal(err)
	}
	digest, err := packagingContractDigest(contract)
	if err != nil || !validDigest(digest) {
		t.Fatalf("digest=%q err=%v", digest, err)
	}
	contract.Digest = digest
	if !exactCanonicalPackagingContract(contract) {
		t.Fatal("canonical contract does not match itself")
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "retrieval", "relation-packaging-experiment-contract-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tracked PackagingContract
	if err := json.Unmarshal(data, &tracked); err != nil {
		t.Fatal(err)
	}
	if !exactCanonicalPackagingContract(tracked) || tracked.Digest != digest {
		t.Fatal("tracked packaging contract diverged from canonical freeze")
	}
	contract.ArmDAuthorized = true
	if err := validatePackagingContract(contract); err == nil {
		t.Fatal("authorized arm D accepted")
	}
}

func TestCanonicalAdoptedPackagingContract(t *testing.T) {
	contract := canonicalAdoptedPackagingContract()
	digest, err := adoptedPackagingContractDigest(contract)
	if err != nil || !validDigest(digest) {
		t.Fatalf("digest=%q err=%v", digest, err)
	}
	contract.Digest = digest
	if err := validateAdoptedPackagingContract(contract); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("WRITE_ADOPTED_PACKAGING_CONTRACT") == "1" {
		data, err := json.MarshalIndent(contract, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join("..", "..", "testdata", "retrieval", "relation-sibling-packaging-adopted-v1.json")
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "retrieval", "relation-sibling-packaging-adopted-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tracked AdoptedPackagingContract
	if err := json.Unmarshal(data, &tracked); err != nil {
		t.Fatal(err)
	}
	if err := validateAdoptedPackagingContract(tracked); err != nil || tracked.Digest != digest {
		t.Fatalf("tracked adopted contract diverged: %v digest=%q", err, tracked.Digest)
	}
	tracked.ProductionMCP = true
	if err := validateAdoptedPackagingContract(tracked); err == nil {
		t.Fatal("production MCP adoption accepted")
	}
}

func TestPackagingDecisionCellContinuesBoth(t *testing.T) {
	evaluation, err := evaluatePackaging(canonicalPackagingContract(), packagingGateUniverse())
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Decision.Decision != PackagingDecisionContinueBoth {
		t.Fatalf("decision=%s reason=%s sibling=%v onehop=%v recovered=%d/%d nearby=%d/%d", evaluation.Decision.Decision, evaluation.Decision.InconclusiveReason, evaluation.Decision.SiblingGate, evaluation.Decision.OneHopGate, evaluation.Decision.SiblingRecovered, evaluation.Decision.SiblingMisses, evaluation.Decision.NearbyRecovered, evaluation.Decision.NearbyMisses)
	}
	if !evaluation.Decision.LimitationIncomplete || evaluation.Decision.ArmBCompleteQueries < evaluation.Decision.BaselineComplete {
		t.Fatalf("limitation or completeness regression: %+v", evaluation.Decision)
	}
	if evaluation.Decision.SiblingRecovered != 4 || evaluation.Decision.NearbyRecovered != 2 || evaluation.Decision.BaselineComplete != 31 || evaluation.Decision.ArmBCompleteQueries != 35 || evaluation.Decision.ArmCCompleteQueries != 33 {
		t.Fatalf("unexpected decision cells: %+v", evaluation.Decision)
	}
}

func TestPackagingSiblingCountCapLeavesDeepSibling(t *testing.T) {
	evaluation, err := evaluatePackaging(canonicalPackagingContract(), packagingGateUniverse())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range evaluation.Rows {
		if row.QueryID != "me-x06-navigation-item" || row.Arm != "B" || row.Cell.Count != 4 || row.Cell.Bytes != 4096 {
			continue
		}
		found = true
		if row.CompleteQuery || row.MissingGroupClasses["required"] != PackagingMissSibling {
			t.Fatalf("deep sibling unexpectedly packaged: %+v", row)
		}
	}
	if !found {
		t.Fatal("missing deep-sibling decision cell")
	}
	found = false
	for _, row := range evaluation.Rows {
		if row.QueryID != "me-x06-navigation-item" || row.Arm != "B" || row.Cell.Count != 8 || row.Cell.Bytes != 4096 {
			continue
		}
		found = true
		if !row.CompleteQuery {
			t.Fatalf("count 8 should recover deep sibling: %+v", row)
		}
	}
	if !found {
		t.Fatal("missing deep-sibling count-8 cell")
	}
}

func TestPackagingPrimaryMismatchIsInconclusive(t *testing.T) {
	universe := packagingGateUniverse()
	universe.Queries[0].Primary[0] = "altered-primary"
	evaluation, err := evaluatePackaging(canonicalPackagingContract(), universe)
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Decision.Decision != PackagingDecisionInconclusive || evaluation.Decision.InconclusiveReason == "" {
		t.Fatalf("decision=%+v", evaluation.Decision)
	}
}

func TestPackagingEmitterIgnoresLabels(t *testing.T) {
	universe := packagingGateUniverse()
	junk := packagingScore(universe.Queries[packagingQueryIndex(universe, "gg-g09-rename-change")], "gg-g09-rename-change-junk")
	if junk.ParentID == "" {
		t.Fatal("fixture junk parent missing")
	}
	universe.Labels["gg-g09-rename-change\x00"+junk.ParentID] = reviewLabel{AttachmentID: "label", Grade: 2, GroupIDs: []string{"required"}}
	evaluation, err := evaluatePackaging(canonicalPackagingContract(), universe)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range evaluation.Rows {
		if row.QueryID != "gg-g09-rename-change" || row.Arm != "C" {
			continue
		}
		for _, id := range row.ExtraParentIDs {
			if id == junk.ParentID {
				t.Fatal("label-grade-2 isolated hop leaked into one-hop payload")
			}
		}
		if row.CompleteQuery {
			t.Fatal("limitation query completed by reading labels")
		}
	}
}

func TestPackagingOmitsIsolatedHopFromPayload(t *testing.T) {
	evaluation, err := evaluatePackaging(canonicalPackagingContract(), packagingGateUniverse())
	if err != nil {
		t.Fatal(err)
	}
	seenOmission := false
	for _, row := range evaluation.Rows {
		if row.Arm != "C" {
			continue
		}
		for _, omission := range row.Omissions {
			if omission.Reason == PackagingOmitIsolatedHop {
				seenOmission = true
				for _, id := range row.ExtraParentIDs {
					if id == omission.ParentID {
						t.Fatalf("isolated hop %s remained in payload", id)
					}
				}
			}
		}
	}
	if !seenOmission {
		t.Fatal("expected isolated-hop omissions")
	}
}

func TestPackagingArtifactsArePortable(t *testing.T) {
	evaluation, err := evaluatePackaging(canonicalPackagingContract(), packagingGateUniverse())
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	output := filepath.Join(root, "relation-packaging-v1")
	if err := writePackagingArtifacts(output, evaluation); err != nil {
		t.Fatal(err)
	}
	if err := writePackagingArtifacts(output, evaluation); err == nil {
		t.Fatal("overwrite of packaging artifacts accepted")
	}
	report, err := os.ReadFile(filepath.Join(output, "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(report), "/Users/") || strings.Contains(string(report), "VOYAGE") {
		t.Fatal("portable report contains local path or credential marker")
	}
	if !strings.Contains(string(report), PackagingDecisionContinueBoth) {
		t.Fatal("report missing decision")
	}
}

func packagingQueryIndex(universe packagingUniverse, id string) int {
	for i, query := range universe.Queries {
		if query.QueryID == id {
			return i
		}
	}
	return -1
}

func packagingGateUniverse() packagingUniverse {
	contract := canonicalPackagingContract()
	queries := []packagingQuery{
		packagingSiblingQuery(contract.Gates.SiblingMissQueryIDs[0], "mod.Needed", 0, 80),
		packagingSiblingQuery(contract.Gates.SiblingMissQueryIDs[1], "mod.Needed", 0, 80),
		packagingSiblingQuery(contract.Gates.SiblingMissQueryIDs[2], "mod.Needed", 0, 80),
		packagingSiblingQuery(contract.Gates.SiblingMissQueryIDs[3], "mod.Needed", 0, 80),
		packagingSiblingQuery(contract.Gates.SiblingMissQueryIDs[4], "mod.E", 4, 80),
		packagingSiblingQuery(contract.Gates.SiblingMissQueryIDs[5], "mod.Aaa", 0, 5000),
		packagingNearbyQuery(contract.Gates.NearbyCrossFileQueryIDs[0]),
		packagingNearbyQuery(contract.Gates.NearbyCrossFileQueryIDs[1]),
		packagingFarQuery(contract.Gates.LimitationQueryIDs[0]),
	}
	for i := 0; i < 31; i++ {
		queries = append(queries, packagingCompleteQuery(fmtPackagingCompleteID(i)))
	}
	parents := map[string]Parent{}
	labels := map[string]reviewLabel{}
	hops := []packagingHop{}
	for _, query := range queries {
		for _, score := range query.Scores {
			parents[score.ParentID] = Parent{ID: score.ParentID, Path: score.Path, IndexedSHA256: score.IndexedSHA256, Language: "go", Kind: "function", QualifiedSymbol: score.QualifiedSymbol, StartByte: score.StartByte, EndByte: score.EndByte}
			labels[query.QueryID+"\x00"+score.ParentID] = reviewLabel{AttachmentID: score.ParentID, Grade: 0}
		}
		for _, id := range query.Primary {
			labels[query.QueryID+"\x00"+id] = reviewLabel{AttachmentID: id, Grade: 1}
		}
		for _, group := range query.RequiredGroups {
			for _, id := range group.SourceParentIDs {
				labels[query.QueryID+"\x00"+id] = reviewLabel{AttachmentID: id, Grade: 2, GroupIDs: []string{group.ID}}
			}
		}
		for _, hint := range query.Hints {
			src, dst := matchPackagingHintSource(hint, query.Scores), matchPackagingHintTarget(hint, query.Scores)
			if src.ParentID != "" && dst.ParentID != "" {
				hops = append(hops, packagingHop{QueryID: query.QueryID, SourceParentID: src.ParentID, TargetParentID: dst.ParentID})
			}
		}
	}
	return packagingUniverse{Queries: queries, Parents: parents, Labels: labels, Hops: hops}
}

func fmtPackagingCompleteID(i int) string {
	return "fx-complete-" + string(rune('0'+i/10)) + string(rune('0'+i%10))
}

func packagingCompleteQuery(id string) packagingQuery {
	primary := []semanticParentScore{
		packagingScoreRow(id, id+"-p1", id+"/a.go", "mod.A", 1, 0, 40),
		packagingScoreRow(id, id+"-p2", id+"/b.go", "mod.B", 2, 0, 40),
		packagingScoreRow(id, id+"-p3", id+"/c.go", "mod.C", 3, 0, 40),
		packagingScoreRow(id, id+"-p4", id+"/d.go", "mod.D", 4, 0, 40),
		packagingScoreRow(id, id+"-p5", id+"/e.go", "mod.E", 5, 0, 40),
	}
	return packagingQuery{QueryID: id, CorpusID: "fixture", Language: "go", Cohorts: []string{"complete"}, Primary: packagingPrimaryIDs(primary), Scores: primary, RequiredGroups: []ReviewRequiredGroup{{ID: "required", SourceParentIDs: []string{id + "-p1"}}}}
}

func packagingSiblingQuery(id, neededSymbol string, neededIndex, neededBytes int) packagingQuery {
	file := id + ".go"
	primary := []semanticParentScore{
		packagingScoreRow(id, id+"-p1", file, "mod.Primary", 1, 0, 50),
		packagingScoreRow(id, id+"-p2", id+"/other.go", "mod.Other", 2, 0, 40),
		packagingScoreRow(id, id+"-p3", id+"/c.go", "mod.C", 3, 0, 40),
		packagingScoreRow(id, id+"-p4", id+"/d.go", "mod.D", 4, 0, 40),
		packagingScoreRow(id, id+"-p5", id+"/e.go", "mod.E", 5, 0, 40),
	}
	extras := []semanticParentScore{}
	symbols := []string{"mod.A", "mod.B", "mod.C", "mod.D", "mod.E", "mod.F", "mod.G", "mod.H"}
	if neededIndex == 0 {
		extras = append(extras, packagingScoreRow(id, id+"-needed", file, neededSymbol, 6, 100, 100+neededBytes))
		extras = append(extras, packagingScoreRow(id, id+"-z", file, "mod.Zzz", 7, 7000, 7010))
	} else {
		rank := 6
		for i, symbol := range symbols {
			parentID := id + "-x" + symbol
			bytes := 80
			if i == neededIndex {
				parentID = id + "-needed"
				symbol = neededSymbol
				bytes = neededBytes
			}
			extras = append(extras, packagingScoreRow(id, parentID, file, symbol, rank, rank*100, rank*100+bytes))
			rank++
		}
	}
	needed := id + "-needed"
	scores := append(append([]semanticParentScore{}, primary...), extras...)
	junk := packagingScoreRow(id, id+"-junk", id+"/junk.go", "mod.Junk", 90, 0, 20)
	scores = append(scores, junk)
	hint := packagingHint(id, extras[len(extras)-1], junk)
	return packagingQuery{QueryID: id, CorpusID: "fixture", Language: "go", Cohorts: []string{"sibling"}, Primary: packagingPrimaryIDs(primary), Scores: scores, RequiredGroups: []ReviewRequiredGroup{{ID: "required", SourceParentIDs: []string{needed}}}, Hints: []relationHint{hint}}
}

func packagingNearbyQuery(id string) packagingQuery {
	primary := []semanticParentScore{
		packagingScoreRow(id, id+"-p1", id+"/a.go", "mod.A", 1, 0, 40),
		packagingScoreRow(id, id+"-p2", id+"/b.go", "mod.B", 2, 0, 40),
		packagingScoreRow(id, id+"-p3", id+"/c.go", "mod.C", 3, 0, 40),
		packagingScoreRow(id, id+"-p4", id+"/d.go", "mod.D", 4, 0, 40),
		packagingScoreRow(id, id+"-p5", id+"/e.go", "mod.E", 5, 0, 40),
	}
	needed := packagingScoreRow(id, id+"-needed", id+"/needed.go", "mod.Needed", 14, 0, 80)
	junk := packagingScoreRow(id, id+"-junk", id+"/junk.go", "mod.Junk", 80, 0, 20)
	scores := append(append([]semanticParentScore{}, primary...), needed, junk)
	return packagingQuery{QueryID: id, CorpusID: "fixture", Language: "go", Cohorts: []string{"nearby"}, Primary: packagingPrimaryIDs(primary), Scores: scores, RequiredGroups: []ReviewRequiredGroup{{ID: "required", SourceParentIDs: []string{needed.ParentID}}}, Hints: []relationHint{packagingHint(id, primary[0], needed), packagingHint(id, needed, junk)}}
}

func packagingFarQuery(id string) packagingQuery {
	primary := []semanticParentScore{
		packagingScoreRow(id, id+"-p1", id+"/a.go", "mod.A", 1, 0, 40),
		packagingScoreRow(id, id+"-p2", id+"/b.go", "mod.B", 2, 0, 40),
		packagingScoreRow(id, id+"-p3", id+"/c.go", "mod.C", 3, 0, 40),
		packagingScoreRow(id, id+"-p4", id+"/d.go", "mod.D", 4, 0, 40),
		packagingScoreRow(id, id+"-p5", id+"/e.go", "mod.E", 5, 0, 40),
	}
	anchor := packagingScoreRow(id, id+"-anchor", id+"/anchor.go", "mod.Anchor", 20, 0, 40)
	needed := packagingScoreRow(id, id+"-needed", id+"/far.go", "mod.Change", 134, 0, 80)
	junk := packagingScoreRow(id, id+"-junk", id+"/junk.go", "mod.Junk", 200, 0, 20)
	scores := append(append([]semanticParentScore{}, primary...), anchor, needed, junk)
	return packagingQuery{QueryID: id, CorpusID: "fixture", Language: "go", Cohorts: []string{"far"}, Primary: packagingPrimaryIDs(primary), Scores: scores, RequiredGroups: []ReviewRequiredGroup{{ID: "required", SourceParentIDs: []string{needed.ParentID}}}, Hints: []relationHint{packagingHint(id, anchor, needed), packagingHint(id, anchor, junk)}}
}

func packagingScoreRow(query, parent, path, symbol string, rank, start, end int) semanticParentScore {
	return semanticParentScore{QueryID: query, ParentID: parent, Path: path, IndexedSHA256: completionTestDigest, QualifiedSymbol: symbol, StartByte: start, EndByte: end, GlobalRank: rank, NativeScore: float64(1000 - rank)}
}

func packagingPrimaryIDs(values []semanticParentScore) []string {
	ids := make([]string, ProtectedPrimaryK)
	for i := 0; i < ProtectedPrimaryK; i++ {
		ids[i] = values[i].ParentID
	}
	return ids
}

func packagingHint(query string, source, target semanticParentScore) relationHint {
	return relationHint{QueryID: query, Kind: Calls, Direction: Forward, StructuralTier: ExecutableDependencyTier, SourcePath: source.Path, SourceSHA256: source.IndexedSHA256, SourceStartByte: source.StartByte, SourceEndByte: source.EndByte, TargetPath: target.Path, TargetSHA256: target.IndexedSHA256, TargetQualified: target.QualifiedSymbol, TargetStartByte: target.StartByte, TargetEndByte: target.EndByte, OccurrenceCount: 1}
}
