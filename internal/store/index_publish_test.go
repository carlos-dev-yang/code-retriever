package store

import (
	"cidx/internal/config"
	"context"
	"database/sql"
	"testing"
)

func TestIndexPublisherKeepsProductionReaderSnapshotAndRollsBack(t *testing.T) {
	ctx := context.Background()
	resolved := testResolvedConfig(t)
	p, err := OpenProduction(ctx, t.TempDir(), resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	first := indexFile("a.go", "One", "package p\nfunc One() {}\n", resolved)
	secondSegment := first.Chunks[0].Segments[0]
	secondSegment.Number = 1
	secondSegment.CanonicalInputSHA256 = fixtureHash("one-second")
	first.Chunks[0].Segments = append(first.Chunks[0].Segments, secondSegment)
	if err := p.PublishIndexGeneration(ctx, IndexPublishPlan{BaseGeneration: 0, NextGeneration: 1, ManifestSHA256: fixtureHash("one"), Reason: "manual", Desired: resolved, Changed: []PreparedIndexFile{first}}); err != nil {
		t.Fatal(err)
	}
	old, err := p.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer old.Rollback()
	var generation int64
	if err := old.QueryRowContext(ctx, `SELECT active_generation FROM meta WHERE id=1`).Scan(&generation); err != nil || generation != 1 {
		t.Fatalf("old generation=%d err=%v", generation, err)
	}
	second := indexFile("a.go", "Two", "package p\nfunc Two() {}\n", resolved)
	if err := p.PublishIndexGeneration(ctx, IndexPublishPlan{BaseGeneration: 1, NextGeneration: 2, ManifestSHA256: fixtureHash("two"), Reason: "manual", Desired: resolved, Changed: []PreparedIndexFile{second}}); err != nil {
		t.Fatal(err)
	}
	if err := p.PublishIndexGeneration(ctx, IndexPublishPlan{BaseGeneration: 2, NextGeneration: 3, ManifestSHA256: fixtureHash("delete"), Reason: "manual", Desired: resolved, Deleted: []string{"a.go"}}); err != nil {
		t.Fatalf("multi-segment deletion failed: %v", err)
	}
	var symbol string
	if err := old.QueryRowContext(ctx, `SELECT symbol FROM chunks`).Scan(&symbol); err != nil || symbol != "One" {
		t.Fatalf("old reader mixed snapshot: %q %v", symbol, err)
	}
	if err := old.Commit(); err != nil {
		t.Fatal(err)
	}
	bad := indexFile("a.go", "Bad", "package p\nfunc Bad() {}\n", resolved)
	bad.Chunks[0].EndByte++
	if err := p.PublishIndexGeneration(ctx, IndexPublishPlan{BaseGeneration: 3, NextGeneration: 4, ManifestSHA256: fixtureHash("bad"), Reason: "manual", Desired: resolved, Changed: []PreparedIndexFile{bad}}); err == nil {
		t.Fatal("invalid publish accepted")
	}
	applied, err := p.AppliedProfiles(ctx)
	if err != nil || applied.ActiveGeneration != 3 {
		t.Fatalf("rollback changed generation: %#v %v", applied, err)
	}
}
func indexFile(path, symbol, body string, resolved config.ResolvedConfig) PreparedIndexFile {
	chunk := PreparedIndexChunk{Kind: "function", Symbol: symbol, QualifiedSymbol: symbol, Signature: "func " + symbol + "()", StartByte: 10, EndByte: 10 + len(body), StartLine: 1, EndLine: 2, SourceBody: []byte(body), Projections: []PreparedIndexProjection{{Kind: "body", StartByte: 0, EndByte: len(body)}}, Symbols: []PreparedIndexSymbol{{Original: symbol, Normalized: symbol}}, FTSSymbols: symbol + " " + symbol, FTSBody: "func " + symbol + "()\n" + body, Segments: []PreparedIndexSegment{{Number: 0, CanonicalInputSHA256: fixtureHash(symbol + "hash"), CanonicalTextProfile: string(resolved.Profiles.Fingerprints.CanonicalText), ServingProfile: string(resolved.Profiles.Fingerprints.VectorStorage), DisplayStartByte: 0, DisplayEndByte: len(body), Projections: []PreparedIndexRange{{StartByte: 0, EndByte: len(body)}}}}}
	return PreparedIndexFile{Path: path, Language: "go", SHA256: fixtureHash(symbol + "sha"), Size: int64(len(body)), Chunks: []PreparedIndexChunk{chunk}}
}
