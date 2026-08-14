package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cidx/internal/evalcontract"
	"cidx/internal/search/lexical"
)

func TestManifestStrictPortableFingerprint(t *testing.T) {
	m := fixtureManifest(t.TempDir())
	if _, err := m.Fingerprint(); err != nil {
		t.Fatal(err)
	}
	m.RootSubdir = "/machine/path"
	if err := m.Validate(); err == nil {
		t.Fatal("absolute root accepted")
	}
	data, _ := json.Marshal(fixtureManifest(t.TempDir()))
	data = []byte(strings.Replace(string(data), `"corpus_id"`, `"corpus_id","corpus_id"`, 1))
	if _, err := LoadCorpusManifest(data); err == nil {
		t.Fatal("duplicate field accepted")
	}
}

func TestBindingVerificationRejectsDirtyAndSymlink(t *testing.T) {
	root := t.TempDir()
	run(t, root, "git", "init")
	run(t, root, "git", "config", "user.email", "test@example.invalid")
	run(t, root, "git", "config", "user.name", "test")
	write(t, filepath.Join(root, "LICENSE"), "MIT\n")
	write(t, filepath.Join(root, "a.go"), "package a\n")
	run(t, root, "git", "add", ".")
	run(t, root, "git", "commit", "-m", "initial")
	run(t, root, "git", "remote", "add", "origin", "https://example.invalid/acme/repo.git")
	commit := strings.TrimSpace(run(t, root, "git", "rev-parse", "HEAD"))
	m := fixtureManifest(root)
	m.PinnedCommit = commit
	m.ExpectedTreeHash = strings.TrimSpace(run(t, root, "git", "rev-parse", "HEAD^{tree}"))
	hash, err := selectedContentHash(context.Background(), root, m)
	if err != nil {
		t.Fatal(err)
	}
	m.ExpectedContentSHA256 = hash
	if _, err := VerifyCheckout(context.Background(), m, root); err != nil {
		t.Fatal(err)
	}
	wrongHash := m
	wrongHash.ExpectedContentSHA256 = strings.Repeat("0", 64)
	if _, err := VerifyCheckout(context.Background(), wrongHash, root); err == nil {
		t.Fatal("wrong content hash accepted")
	}
	write(t, filepath.Join(root, "dirty.go"), "package a\n")
	if _, err := VerifyCheckout(context.Background(), m, root); err == nil {
		t.Fatal("dirty checkout accepted")
	}
}

func TestORAndMetricsFailuresAndSlices(t *testing.T) {
	one, two := span("a.go", "A", 0, 5), span("b.go", "B", 0, 5)
	c := fixtureCase("q", evalcontract.Go,
		[]evalcontract.RequiredGroup{
			{ID: "one", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{one}}, {Spans: []evalcontract.SourceSpan{two}}}},
			{ID: "two", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{two}}}},
		},
		[]evalcontract.RelevanceJudgment{{Span: one, Grade: 2, Rationale: "direct"}, {Span: two, Grade: 2, Rationale: "direct"}},
	)
	got, err := EvaluateCase(c, []lexical.Hit{{Path: "a.go", IndexedSHA256: one.ContentSHA256, QualifiedSymbol: "A", StartByte: 0, EndByte: 5}, {Path: "b.go", IndexedSHA256: two.ContentSHA256, QualifiedSymbol: "B", StartByte: 0, EndByte: 5}}, []int{1, 5}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.RecallAt[1] != 0.5 || !got.CompleteRequirementHitAt[5] || got.MRRAt[1] != 1 {
		t.Fatalf("metrics=%+v", got)
	}
	failed, err := EvaluateCase(c, nil, []int{1, 5}, errors.New("boom"))
	if err != nil {
		t.Fatal(err)
	}
	summary := Summarize([]CaseResult{got, failed}, []int{1, 5})
	if summary.Denominators.AnswerableQueries != 2 || summary.Failures != 1 || summary.ByLanguage[evalcontract.Go].Denominators.AnswerableQueries != 2 {
		t.Fatalf("summary=%+v", summary)
	}
}

func TestArtifactIsAtomicAndImmutable(t *testing.T) {
	artifact := fixtureArtifact()
	root := t.TempDir()
	if _, err := WriteRunArtifact(root, artifact); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteRunArtifact(root, artifact); err == nil {
		t.Fatal("artifact overwrite accepted")
	}
	if _, err := os.Stat(filepath.Join(root, "run-1", "run.json")); err != nil {
		t.Fatal(err)
	}
}

func TestWholeSegmentGlobPolicy(t *testing.T) {
	for _, test := range []struct {
		pattern, file string
		want          bool
	}{{"**", "a.go", true}, {"**/*.go", "a.go", true}, {"**/*.go", "src/a.go", true}, {"src/**", "src/a.go", true}, {"src/**", "other/a.go", false}} {
		if got := globMatch(test.pattern, test.file); got != test.want {
			t.Fatalf("glob %q %q = %v", test.pattern, test.file, got)
		}
	}
	if !globMatch("a/**/b/**/c.go", "a/x/b/y/z/c.go") {
		t.Fatal("nested globstar did not match")
	}
	manifest := fixtureManifest("")
	manifest.Exclude = nil
	if err := manifest.Validate(); err != nil {
		t.Fatalf("empty excludes rejected: %v", err)
	}
	if matchesPolicy("src/vendor/a.go", []string{"src/**"}, []string{"src/vendor/**"}) {
		t.Fatal("exclude did not win")
	}
	if validPatterns([]string{"src/**name"}, false) || validPatterns([]string{"["}, false) || !validPatterns(nil, true) {
		t.Fatal("invalid glob syntax accepted")
	}
}

func TestRunnerPreflightTraceDriftAndCancellation(t *testing.T) {
	span := span("a.go", "A", 0, 5)
	caseValue := fixtureCase("q", evalcontract.Go, []evalcontract.RequiredGroup{{ID: "g", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{span}}}}}, []evalcontract.RelevanceJudgment{{Span: span, Grade: 2, Rationale: "direct"}})
	dataset := EvaluationDataset{SchemaVersion: 1, Version: "v1", CorpusID: "sample", Cases: []evalcontract.EvaluationCase{caseValue}}
	inventory := fakeInventory{snapshot: TruthInventorySnapshot{Generation: 7, ManifestSHA256: strings.Repeat("a", 64), Chunks: []IndexedTruth{{Path: span.Path, IndexedSHA256: span.ContentSHA256, QualifiedSymbol: span.QualifiedSymbol, Kind: "function", StartByte: 0, EndByte: 5}}}}
	runner := LexicalRunner{Inventory: inventory, Searcher: fakeSearch{result: lexical.Result{IndexGeneration: 7, ManifestSHA256: strings.Repeat("a", 64), CandidateCount: 1, Hits: []lexical.Hit{{Path: span.Path, IndexedSHA256: span.ContentSHA256, QualifiedSymbol: span.QualifiedSymbol, Kind: "function", StartByte: 0, EndByte: 5, BM25Rank: 1}}}}, CandidateK: 1, Ks: []int{1}}
	run, err := runner.Run(context.Background(), dataset)
	if err != nil || run.Results[0].Trace.TerminalState != evalcontract.TerminalComplete || run.Generation != 7 {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	bad := inventory
	bad.snapshot.Chunks = append([]IndexedTruth(nil), inventory.snapshot.Chunks...)
	bad.snapshot.Chunks[0].IndexedSHA256 = strings.Repeat("b", 64)
	if _, err := (LexicalRunner{Inventory: bad, Searcher: runner.Searcher, CandidateK: 1, Ks: []int{1}}).Run(context.Background(), dataset); err == nil {
		t.Fatal("stale truth accepted")
	}
	drift := runner
	drift.Searcher = fakeSearch{result: lexical.Result{IndexGeneration: 8, ManifestSHA256: strings.Repeat("a", 64)}}
	if _, err := drift.Run(context.Background(), dataset); err == nil || err.Error() != string(NonReproducibleRun) {
		t.Fatalf("generation drift err=%v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runner.Run(cancelled, dataset); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation err=%v", err)
	}
}

func TestOperationFailureTraceAndArtifactAuthority(t *testing.T) {
	artifact := fixtureArtifact()
	if err := artifact.Results[0].Trace.Validate(); err != nil || artifact.Results[0].FailureStage != evalcontract.FailureStage(evalcontract.StageFTSCandidate) {
		t.Fatalf("operation trace err=%v", err)
	}
	artifact.Summary.Cases++
	if _, err := WriteRunArtifact(t.TempDir(), artifact); err == nil {
		t.Fatal("forged summary accepted")
	}
	artifact = fixtureArtifact()
	artifact.Results[0].Hits = []RankedHit{{Path: "a.go", IndexedSHA256: strings.Repeat("a", 64), Kind: "function", QualifiedSymbol: "A", StartByte: 0, EndByte: 1, Rank: 2}}
	if _, err := WriteRunArtifact(t.TempDir(), artifact); err == nil {
		t.Fatal("non-contiguous ranks accepted")
	}
	artifact = fixtureArtifact()
	delete(artifact.Results[0].HitAt, 1)
	if _, err := WriteRunArtifact(t.TempDir(), artifact); err == nil {
		t.Fatal("missing per-k observation accepted")
	}
}

func TestMetricParentDedupAndMultiSpanRequirements(t *testing.T) {
	left, right := span("a.go", "A", 0, 5), span("a.go", "A", 5, 10)
	support := span("a.go", "A", 10, 15)
	caseValue := fixtureCase("multi", evalcontract.Go, []evalcontract.RequiredGroup{{ID: "both", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{left, right}}}}}, []evalcontract.RelevanceJudgment{{Span: left, Grade: 2, Rationale: "direct"}, {Span: right, Grade: 2, Rationale: "direct"}, {Span: support, Grade: 1, Rationale: "support"}})
	hits := []lexical.Hit{{Path: "a.go", IndexedSHA256: left.ContentSHA256, QualifiedSymbol: "A", StartByte: 0, EndByte: 5}, {Path: "a.go", IndexedSHA256: left.ContentSHA256, QualifiedSymbol: "A", StartByte: 0, EndByte: 5}, {Path: "a.go", IndexedSHA256: right.ContentSHA256, QualifiedSymbol: "A", StartByte: 5, EndByte: 10}}
	result, err := EvaluateCase(caseValue, hits, []int{1, 3}, nil)
	if err != nil {
		t.Fatal(err)
	}
	duplicateOnly, err := EvaluateCase(caseValue, hits[:2], []int{1, 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HitAt[1] || result.CompleteRequirementHitAt[1] || !result.CompleteRequirementHitAt[3] || duplicateOnly.NDCGAt[2] != duplicateOnly.NDCGAt[1] {
		t.Fatalf("multi-span/dedup metrics=%+v", result)
	}
	wrong := hits[0]
	wrong.IndexedSHA256 = strings.Repeat("0", 64)
	if match([]lexical.Hit{wrong}, left) {
		t.Fatal("wrong indexed hash matched")
	}
}

func TestHardNegativeDenominatorAndAbstainableTrace(t *testing.T) {
	negative := span("n.go", "N", 0, 1)
	answer := fixtureCase("answer", evalcontract.Go, []evalcontract.RequiredGroup{{ID: "g", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{span("a.go", "A", 0, 1)}}}}}, []evalcontract.RelevanceJudgment{{Span: span("a.go", "A", 0, 1), Grade: 2, Rationale: "direct"}, {Span: negative, Grade: 0, Rationale: "negative"}})
	answer.HardNegatives = []evalcontract.HardNegative{{Span: negative, Reason: "misleading"}}
	abstain := fixtureCase("none", evalcontract.TSX, nil, []evalcontract.RelevanceJudgment{{Span: negative, Grade: 0, Rationale: "negative"}})
	abstain.AnswerMode = evalcontract.Abstainable
	zero := 0
	abstain.ExpectedCardinality = &zero
	abstain.HardNegatives = []evalcontract.HardNegative{{Span: negative, Reason: "misleading"}}
	noNegative := answer
	noNegative.ID = "plain"
	noNegative.HardNegatives = nil
	noNegative.Judgments = noNegative.Judgments[:1]
	first, _ := EvaluateCase(answer, []lexical.Hit{{Path: negative.Path, IndexedSHA256: negative.ContentSHA256, QualifiedSymbol: negative.QualifiedSymbol, StartByte: 0, EndByte: 1}}, []int{1}, nil)
	second, _ := EvaluateCase(abstain, nil, []int{1}, nil)
	third, _ := EvaluateCase(noNegative, nil, []int{1}, nil)
	summary := Summarize([]CaseResult{first, second, third}, []int{1})
	if summary.Denominators.HardNegativeQueries != 2 || summary.KnownHardNegativeHitAt[1] != .5 || summary.ByLanguage[evalcontract.TSX].Denominators.RequiredQueries != 1 || summary.ByCohort["core"].Cases != 3 {
		t.Fatalf("summary=%+v", summary)
	}
	trace := lexicalTrace(abstain, lexical.Result{}, nil)
	if err := trace.Validate(); err != nil {
		t.Fatal(err)
	}
}

type fakeInventory struct{ snapshot TruthInventorySnapshot }

func (value fakeInventory) Snapshot(context.Context) (TruthInventorySnapshot, error) {
	return value.snapshot, nil
}

type fakeSearch struct {
	result lexical.Result
	err    error
}

func (value fakeSearch) Search(context.Context, lexical.Request) (lexical.Result, error) {
	return value.result, value.err
}

func fixtureManifest(_ string) CorpusManifest {
	return CorpusManifest{SchemaVersion: 1, CorpusID: "sample", UpstreamURL: "https://example.invalid/acme/repo.git", PinnedCommit: strings.Repeat("a", 40), LicenseSPDX: "MIT", LicenseEvidence: "LICENSE", LanguageSlices: []evalcontract.Language{evalcontract.Go}, Include: []string{"**"}, Exclude: []string{"vendor/**"}, ExpectedContentSHA256: strings.Repeat("b", 64), ExpectedTreeHash: strings.Repeat("a", 40)}
}
func span(path, symbol string, start, end int) evalcontract.SourceSpan {
	sum := sha256.Sum256([]byte(path))
	return evalcontract.SourceSpan{Path: path, ContentSHA256: hex.EncodeToString(sum[:]), QualifiedSymbol: symbol, StartByte: start, EndByte: end}
}
func fixtureCase(id string, lang evalcontract.Language, groups []evalcontract.RequiredGroup, judgments []evalcontract.RelevanceJudgment) evalcontract.EvaluationCase {
	return evalcontract.EvaluationCase{SchemaVersion: 1, ID: id, Text: "find", Language: lang, Cohorts: []string{"core"}, AnswerMode: evalcontract.BestN, Split: evalcontract.Calibration, RequiredConstraints: evalcontract.RequiredConstraints{Identifiers: []string{"find"}, Paths: []string{"a.go"}, Languages: []evalcontract.Language{lang}, Scopes: []string{"repository"}}, RequiredGroups: groups, Judgments: judgments, Review: evalcontract.ReviewRecord{State: evalcontract.ReviewDraft, Passes: []evalcontract.ReviewPass{{ID: "one", Reviewer: "reviewer"}}, Rationale: "fixture"}, Digest: strings.Repeat("c", 64)}
}
func fixtureArtifact() PortableRunArtifact {
	c := fixtureCase("q", evalcontract.Go, []evalcontract.RequiredGroup{{ID: "g", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{span("a.go", "A", 0, 5)}}}}}, []evalcontract.RelevanceJudgment{{Span: span("a.go", "A", 0, 5), Grade: 2, Rationale: "direct"}})
	result, _ := EvaluateCase(c, nil, []int{1}, errors.New("failed"))
	return PortableRunArtifact{SchemaVersion: 1, RunID: "run-1", CreatedAt: "2026-08-15T00:00:00Z", Manifest: evalcontract.EvaluationRunManifest{SchemaVersion: 1, CorpusManifestSHA256: strings.Repeat("a", 64), QueryManifestSHA256: strings.Repeat("b", 64), CodeCommit: "deadbeef", ProfileFingerprint: strings.Repeat("c", 64), Generation: 1, CandidatePolicy: "candidate-k=5", Platform: "local", PairedControls: evalcontract.PairedRunControls{CorpusStateSHA256: strings.Repeat("d", 64), LabelDigestSHA256: strings.Repeat("e", 64), ParserVersion: "v1", ChunkerVersion: "v1", FTSSchemaVersion: "v1", SourceModel: "not-used", SourceDimensions: 1, ReducerID: "not-used", TargetDimensions: 1, CandidatePolicy: "candidate-k=5", BodyBudget: "not-used", MCPVersion: "v1"}}, Corpus: VerifiedCorpus{CorpusID: "sample", PinnedCommit: strings.Repeat("a", 40), ContentSHA256: strings.Repeat("f", 64), Clean: true}, CorpusManifestFingerprint: strings.Repeat("a", 64), DatasetFingerprint: strings.Repeat("b", 64), ExpectedQueryIDs: []string{"q"}, Generation: 1, ManifestSHA256: strings.Repeat("f", 64), Ks: []int{1}, Results: []RankedCase{{CaseResult: result, Trace: lexicalTrace(c, lexical.Result{}, errors.New("failed"))}}, Summary: Summarize([]CaseResult{result}, []int{1})}
}
func run(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v: %s", name, args, err, out)
	}
	return string(out)
}
func write(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
