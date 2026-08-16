package eval

import (
	"context"
	"strings"
	"testing"

	"cidx/internal/config"
	"cidx/internal/evalcontract"
	"cidx/internal/search/lexical"
	"cidx/internal/store"
	"cidx/internal/symbol"
)

func TestSimpleQueryDeduplicatesAndValidatesResolvedLimits(t *testing.T) {
	limits := config.QueryLimits{MaxBytes: 64, MaxTokens: 2, MaxTokenRunes: 16}
	tokens, exact, err := BuildSimpleQuery("FindThing find thing", symbol.IdentifierNormalizer{}, limits)
	if err != nil || exact != "find thing find thing" || strings.Join(tokens, ",") != "find,thing" {
		t.Fatalf("simple query tokens=%v exact=%q err=%v", tokens, exact, err)
	}
	if _, _, err := BuildSimpleQuery("one two three", symbol.IdentifierNormalizer{}, limits); err == nil {
		t.Fatal("token-limit overflow accepted")
	}
	if _, _, err := BuildSimpleQuery("", symbol.IdentifierNormalizer{}, limits); err == nil {
		t.Fatal("empty query accepted")
	}
}

func TestSimpleSearchAdmissionOrderingAndCap(t *testing.T) {
	resolved := simpleResolved(t, 2)
	searcher, err := NewSimpleSearcher(simpleSnapshot(
		simpleParent("z.go", "z", "pkg.Query", "Query", "body", 4, "d"),
		simpleParent("y.go", "y", "other", "pkg.Query", "body", 3, "c"),
		simpleParent("query/path.go", "path", "other", "Other", "query", 2, "b"),
		simpleParent("a.go", "match", "other", "Other", "query alpha", 1, "a"),
		simpleParent("b.go", "match", "other", "Other", "query alpha", 1, "e"),
	), resolved)
	if err != nil {
		t.Fatal(err)
	}
	result, err := searcher.Search(context.Background(), lexical.Request{Query: "pkg.Query", CandidateK: 2})
	if err != nil || result.CandidateCount != 5 || len(result.Hits) != 2 || result.Hits[0].QualifiedSymbol != "pkg.Query" || result.Hits[1].Symbol != "pkg.Query" {
		t.Fatalf("exact ordering result=%+v err=%v", result, err)
	}
	result, err = searcher.Search(context.Background(), lexical.Request{Query: "query alpha", CandidateK: 2})
	if err != nil || result.CandidateCount != 5 || len(result.Hits) != 2 || result.Hits[0].Path != "query/path.go" || result.Hits[1].Path != "a.go" {
		t.Fatalf("admission/path/match/cap result=%+v err=%v", result, err)
	}
	tieSearcher, err := NewSimpleSearcher(simpleSnapshot(
		simpleParent("same.go", "match", "other", "Other", "match", 1, "e"),
		simpleParent("same.go", "match", "other", "Other", "match", 1, "a"),
	), resolved)
	if err != nil {
		t.Fatal(err)
	}
	result, err = tieSearcher.Search(context.Background(), lexical.Request{Query: "match", CandidateK: 2})
	if err != nil || result.CandidateCount != 2 || result.Hits[0].IndexedSHA256 >= result.Hits[1].IndexedSHA256 {
		t.Fatalf("stable tie ordering result=%+v err=%v", result, err)
	}
	rawQualifiedTie, err := NewSimpleSearcher(simpleSnapshot(
		simpleParent("same.go", "match", "pkg.foo_bar", "Other", "match", 1, "a"),
		simpleParent("same.go", "match", "pkg.FooBar", "Other", "match", 1, "e"),
	), resolved)
	if err != nil {
		t.Fatal(err)
	}
	result, err = rawQualifiedTie.Search(context.Background(), lexical.Request{Query: "match", CandidateK: 2})
	if err != nil || result.Hits[0].QualifiedSymbol != "pkg.FooBar" {
		t.Fatalf("raw qualified-symbol tie result=%+v err=%v", result, err)
	}
	rawPathTie, err := NewSimpleSearcher(simpleSnapshot(
		simpleParent("foo_bar.go", "match", "other", "Other", "match", 1, "a"),
		simpleParent("foo-bar.go", "match", "other", "Other", "match", 1, "a"),
	), resolved)
	if err != nil {
		t.Fatal(err)
	}
	result, err = rawPathTie.Search(context.Background(), lexical.Request{Query: "match", CandidateK: 2})
	if err != nil || result.Hits[0].Path != "foo-bar.go" {
		t.Fatalf("raw bytewise path tie result=%+v err=%v", result, err)
	}
	signatureOnly, err := NewSimpleSearcher(simpleSnapshot(
		simpleParent("other.go", "SignatureOnly", "other.Symbol", "Other", "unrelated body", 1, "a"),
	), resolved)
	if err != nil {
		t.Fatal(err)
	}
	result, err = signatureOnly.Search(context.Background(), lexical.Request{Query: "SignatureOnly", CandidateK: 2})
	if err != nil || result.CandidateCount != 1 || len(result.Hits) != 1 || result.Hits[0].Path != "other.go" {
		t.Fatalf("signature-only admission result=%+v err=%v", result, err)
	}
}

func TestSimpleRunnerPinsGenerationManifestAndAvoidsFTSTrace(t *testing.T) {
	parent := simpleParent("a.go", "FindThing", "pkg.FindThing", "FindThing", "find thing", 0, "a")
	snapshot := simpleSnapshot(parent)
	span := evalcontract.SourceSpan{Path: parent.Path, ContentSHA256: parent.IndexedSHA256, QualifiedSymbol: parent.QualifiedSymbol, StartByte: parent.StartByte, EndByte: parent.EndByte}
	caseValue := fixtureCase("simple", evalcontract.Go, []evalcontract.RequiredGroup{{ID: "g", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{span}}}}}, []evalcontract.RelevanceJudgment{{Span: span, Grade: 2, Rationale: "direct"}})
	caseValue.Text = "FindThing"
	dataset := EvaluationDataset{SchemaVersion: 1, Version: "simple-v1", CorpusID: "sample", Cases: []evalcontract.EvaluationCase{caseValue}}
	run, err := (SimpleRunner{SnapshotSource: FixedSemanticParentSnapshot{Value: snapshot}, Resolved: simpleResolved(t, 3), Ks: []int{1}}).Run(context.Background(), dataset)
	if err != nil || run.Generation != snapshot.Generation || run.ManifestSHA256 != snapshot.ManifestSHA256 || run.AlgorithmFingerprint != SimpleControlFingerprint() || run.Results[0].CandidateCount != 1 || run.Results[0].Metrics.QueryID != "simple" {
		t.Fatalf("simple run=%+v err=%v", run, err)
	}
	if run.Results[0].Metrics.ReturnedCount != 1 || run.Summary.Cases != 1 {
		t.Fatalf("simple metrics=%+v", run.Results[0])
	}
}

func TestSimpleControlFingerprintIsStableAndSealed(t *testing.T) {
	const want = "51bd998e42c4e477c7ef6f57b42b7011a31248d02e270a52e115ead68f2d2769"
	if got := SimpleControlFingerprint(); got != want {
		t.Fatalf("fingerprint=%q", got)
	}
}

func TestSimpleArtifactIsSeparateAndImmutable(t *testing.T) {
	parent := simpleParent("a.go", "FindThing", "pkg.FindThing", "FindThing", "find thing", 0, "a")
	snapshot := simpleSnapshot(parent)
	span := evalcontract.SourceSpan{Path: parent.Path, ContentSHA256: parent.IndexedSHA256, QualifiedSymbol: parent.QualifiedSymbol, StartByte: parent.StartByte, EndByte: parent.EndByte}
	caseValue := fixtureCase("simple-artifact", evalcontract.Go, []evalcontract.RequiredGroup{{ID: "g", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{span}}}}}, []evalcontract.RelevanceJudgment{{Span: span, Grade: 2, Rationale: "direct"}})
	caseValue.Text = "FindThing"
	run, err := (SimpleRunner{SnapshotSource: FixedSemanticParentSnapshot{Value: snapshot}, Resolved: simpleResolved(t, 3), Ks: []int{1, 2}}).Run(context.Background(), EvaluationDataset{SchemaVersion: 1, Version: "simple-v1", CorpusID: "sample", Cases: []evalcontract.EvaluationCase{caseValue}})
	if err != nil {
		t.Fatal(err)
	}
	base := fixtureArtifact()
	base.Manifest.Generation = run.Generation
	candidatePolicy := SimpleCandidatePolicy(run.AlgorithmFingerprint, 3, 2)
	base.Manifest.CandidatePolicy = candidatePolicy
	base.Manifest.PairedControls.CandidatePolicy = candidatePolicy
	base.Manifest.PairedControls.CorpusStateSHA256 = base.Corpus.ContentSHA256
	base.Manifest.PairedControls.LabelDigestSHA256 = base.DatasetFingerprint
	artifact := SimplePortableRunArtifact{SchemaVersion: evalcontract.SchemaVersion, RunID: "simple-run", CreatedAt: "2026-08-16T00:00:00Z", ControlAlgorithm: SimpleControlAlgorithm, AlgorithmFingerprint: run.AlgorithmFingerprint, Manifest: base.Manifest, Corpus: base.Corpus, CorpusManifestFingerprint: base.CorpusManifestFingerprint, DatasetFingerprint: base.DatasetFingerprint, ExpectedQueryIDs: []string{caseValue.ID}, Generation: run.Generation, ManifestSHA256: run.ManifestSHA256, CandidateK: 3, ReturnK: 2, Ks: run.Ks, Results: run.Results, Summary: run.Summary}
	contradictoryCorpus := artifact
	contradictoryCorpus.Manifest.PairedControls.CorpusStateSHA256 = strings.Repeat("0", 64)
	if _, err := WriteSimpleRunArtifact(t.TempDir(), contradictoryCorpus); err == nil {
		t.Fatal("contradictory corpus paired control accepted")
	}
	contradictoryPolicy := artifact
	contradictoryPolicy.Manifest.CandidatePolicy = "candidate-k=5"
	contradictoryPolicy.Manifest.PairedControls.CandidatePolicy = "candidate-k=5"
	if _, err := WriteSimpleRunArtifact(t.TempDir(), contradictoryPolicy); err == nil {
		t.Fatal("non-simple candidate policy accepted")
	}
	root := t.TempDir()
	if _, err := WriteSimpleRunArtifact(root, artifact); err != nil {
		t.Fatal(err)
	}
	incompletePool := artifact
	incompletePool.Results = append([]SimpleRankedCase(nil), artifact.Results...)
	incompletePool.Results[0].CandidateCount = 2
	if _, err := WriteSimpleRunArtifact(t.TempDir(), incompletePool); err == nil {
		t.Fatal("incomplete capped candidate pool accepted")
	}
	tooDeep := artifact
	tooDeep.Ks = []int{1, 3}
	if _, err := WriteSimpleRunArtifact(t.TempDir(), tooDeep); err == nil {
		t.Fatal("metric depth beyond return limit accepted")
	}
	forged := artifact
	forged.Results = append([]SimpleRankedCase(nil), artifact.Results...)
	forged.Results[0].Metrics.RecallAt = make(map[int]float64, len(artifact.Results[0].Metrics.RecallAt))
	for k, value := range artifact.Results[0].Metrics.RecallAt {
		forged.Results[0].Metrics.RecallAt[k] = value
	}
	delete(forged.Results[0].Metrics.RecallAt, 1)
	forged.Summary = summarizeSimple(forged.Results, forged.Ks)
	if _, err := WriteSimpleRunArtifact(t.TempDir(), forged); err == nil {
		t.Fatal("forged incomplete metrics accepted")
	}
	if _, err := WriteSimpleRunArtifact(root, artifact); err == nil {
		t.Fatal("simple artifact overwrite accepted")
	}
}

func simpleResolved(t *testing.T, candidateK int) config.ResolvedConfig {
	t.Helper()
	dimensions, max, returnK, rrf := 256, 4096, 2, 60
	allow := false
	raw := config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: max, TargetSegmentBytes: max}, Embedding: config.RawEmbedding{ServingDimensions: &dimensions, Request: config.RawRequest{MaxInputs: 1, MaxTotalInputBytes: max, TimeoutSeconds: 1}}, Search: config.RawSearch{AllowPaidQueryEmbedding: &allow, ReturnK: &returnK, CandidateK: &candidateK, RRFK: &rrf}, MCP: config.RawMCP{HardMaxInlineBytes: max}}
	resolved, err := config.Resolve(raw)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func simpleSnapshot(parents ...store.SemanticParent) store.SemanticParentSnapshot {
	return store.SemanticParentSnapshot{Generation: 7, ManifestSHA256: strings.Repeat("f", 64), Parents: parents}
}

func simpleParent(path, symbolValue, qualified, symbolField, body string, start int, hashSeed string) store.SemanticParent {
	return store.SemanticParent{Path: path, IndexedSHA256: strings.Repeat(hashSeed, 64), Language: "go", Kind: "function", Symbol: symbolField, QualifiedSymbol: qualified, Signature: "func " + symbolValue + "()", SourceBody: body, StartByte: start, EndByte: start + 1, StartLine: 1, EndLine: 1}
}
