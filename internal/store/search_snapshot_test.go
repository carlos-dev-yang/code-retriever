package store

import (
	"context"
	"math"
	"testing"
)

func TestSearchFTSOrdersAllStableKeysBeforeCandidateLimit(t *testing.T) {
	ctx := context.Background()
	resolved := testResolvedConfig(t)
	production, err := OpenProduction(ctx, t.TempDir(), resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()

	exact := indexFile("z.go", "Exact", "package p\nfunc Exact() {}\n", resolved)
	exactChunk := &exact.Chunks[0]
	exactChunk.QualifiedSymbol = "pkg.Exact"
	exactChunk.Symbols = []PreparedIndexSymbol{{Original: "Exact", Normalized: "exact"}, {Original: "pkg.Exact", Normalized: "pkg exact"}}
	exactChunk.FTSSymbols, exactChunk.FTSBody = "pkg exact", "same lexical body"
	other := indexFile("a.go", "Other", "package p\nfunc Other() {}\n", resolved)
	otherChunk := &other.Chunks[0]
	otherChunk.QualifiedSymbol = "pkg.Other"
	otherChunk.Symbols = []PreparedIndexSymbol{{Original: "Other", Normalized: "other"}, {Original: "pkg.Other", Normalized: "pkg other"}}
	otherChunk.FTSSymbols, otherChunk.FTSBody = "pkg exact", "same lexical body"
	if err := production.PublishIndexGeneration(ctx, IndexPublishPlan{BaseGeneration: 0, NextGeneration: 1, ManifestSHA256: fixtureHash("search"), Reason: "manual", Desired: resolved, Changed: []PreparedIndexFile{exact, other}}); err != nil {
		t.Fatal(err)
	}

	result, err := production.SearchFTS(ctx, FTSSearchRequest{MatchExpression: `"pkg" AND "exact"`, CandidateK: 1, SymbolWeight: resolved.Search.FTSSymbolWeight, BodyWeight: resolved.Search.FTSBodyWeight, ExactNormalizedSymbol: "pkg exact"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].Path != "z.go" || !result.Candidates[0].ExactQualifiedSymbol {
		t.Fatalf("candidate boundary order=%#v", result.Candidates)
	}
	if _, err := production.SearchFTS(ctx, FTSSearchRequest{MatchExpression: `"pkg"`, CandidateK: 1, SymbolWeight: math.NaN(), BodyWeight: 1}); err == nil {
		t.Fatal("NaN BM25 weight accepted")
	}
}

func TestTruthSnapshotPinsAuthoritativeChunks(t *testing.T) {
	ctx := context.Background()
	resolved := testResolvedConfig(t)
	production, err := OpenProduction(ctx, t.TempDir(), resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	file := indexFile("a.go", "One", "package p\nfunc One() {}\n", resolved)
	if err := production.PublishIndexGeneration(ctx, IndexPublishPlan{BaseGeneration: 0, NextGeneration: 1, ManifestSHA256: fixtureHash("truth"), Reason: "manual", Desired: resolved, Changed: []PreparedIndexFile{file}}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := production.TruthSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Generation != 1 || snapshot.ManifestSHA256 != fixtureHash("truth") || len(snapshot.Chunks) != 1 || snapshot.Chunks[0].IndexedSHA256 != file.SHA256 || snapshot.Chunks[0].Path != "a.go" {
		t.Fatalf("truth snapshot=%#v", snapshot)
	}
	semantic, err := production.SemanticParentsSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if semantic.Generation != snapshot.Generation || semantic.ManifestSHA256 != snapshot.ManifestSHA256 || len(semantic.Parents) != 1 || semantic.Parents[0].Path != "a.go" || semantic.Parents[0].SourceBody != string(file.Chunks[0].SourceBody) || semantic.Parents[0].Symbol == "" || semantic.Parents[0].Signature == "" {
		t.Fatalf("semantic snapshot=%#v", semantic)
	}
}
