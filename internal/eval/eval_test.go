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
	"cidx/internal/search"
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
	m.CleanTreeRequired = true
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

func TestCorpusBindingFormsAndOptionalCleanTree(t *testing.T) {
	flat, err := LoadCorpusBindings([]byte(`{"sample":"/tmp/sample"}`))
	if err != nil || flat.Bindings["sample"] != "/tmp/sample" {
		t.Fatalf("flat binding=%+v err=%v", flat, err)
	}
	wrapped, err := LoadCorpusBindings([]byte(`{"bindings":{"sample":"/tmp/sample"}}`))
	if err != nil || wrapped.Bindings["sample"] != "/tmp/sample" {
		t.Fatalf("wrapped binding=%+v err=%v", wrapped, err)
	}
	if _, err := LoadCorpusBindings([]byte(`{"sample":"/tmp/sample","unexpected":1}`)); err == nil {
		t.Fatal("invalid flat binding accepted")
	}
	root := t.TempDir()
	run(t, root, "git", "init")
	run(t, root, "git", "config", "user.email", "test@example.invalid")
	run(t, root, "git", "config", "user.name", "test")
	write(t, filepath.Join(root, "LICENSE"), "MIT\n")
	write(t, filepath.Join(root, "a.go"), "package a\n")
	run(t, root, "git", "add", ".")
	run(t, root, "git", "commit", "-m", "initial")
	run(t, root, "git", "remote", "add", "origin", "https://example.invalid/acme/repo.git")
	m := fixtureManifest(root)
	m.PinnedCommit = strings.TrimSpace(run(t, root, "git", "rev-parse", "HEAD"))
	m.ExpectedTreeHash = strings.TrimSpace(run(t, root, "git", "rev-parse", "HEAD^{tree}"))
	hash, err := selectedContentHash(context.Background(), root, m)
	if err != nil {
		t.Fatal(err)
	}
	m.ExpectedContentSHA256 = hash
	write(t, filepath.Join(root, "dirty.go"), "package a\n")
	verified, err := VerifyCheckout(context.Background(), m, root)
	if err != nil {
		t.Fatalf("optional clean tree unexpectedly rejected: %v", err)
	}
	if verified.Clean {
		t.Fatal("dirty optional-clean checkout was recorded as clean")
	}
	if err := VerifyIndexedFiles(context.Background(), m, root, map[string]string{"a.go": hex.EncodeToString(sha256Bytes([]byte("package a\n")))}); err != nil {
		t.Fatalf("indexed source verification failed: %v", err)
	}
	if err := VerifyIndexedFiles(context.Background(), m, root, map[string]string{"other.go": strings.Repeat("a", 64)}); err == nil {
		t.Fatal("out-of-policy indexed source accepted")
	}
}

func sha256Bytes(value []byte) []byte { sum := sha256.Sum256(value); return sum[:] }

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

func TestRetrievalFidelityFusionAndBodyDiagnostics(t *testing.T) {
	left, right := span("a.go", "A", 0, 5), span("b.go", "B", 0, 5)
	caseValue := fixtureCase("retrieval", evalcontract.Go, []evalcontract.RequiredGroup{
		{ID: "left", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{left}}}},
		{ID: "right", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{right}}}},
	}, []evalcontract.RelevanceJudgment{{Span: left, Grade: 2, Rationale: "direct"}, {Span: right, Grade: 2, Rationale: "direct"}})
	first, second := 1.0, .5
	f32 := CaseRanking{QueryID: caseValue.ID, Variant: VariantTargetF32, Hits: []RetrievalHit{retrievalHit(left, 1, &first), retrievalHit(right, 2, &second)}}
	codec := CaseRanking{QueryID: caseValue.ID, Variant: VariantServingActiveCodec, Hits: []RetrievalHit{retrievalHit(right, 1, &first), retrievalHit(left, 2, &second)}}
	fidelity, err := CompareCodecFidelity(caseValue, f32, codec, 2)
	if err != nil || fidelity.TopKRetention != 1 || !fidelity.Top1Mismatch || fidelity.PairwiseInversionRate == nil || *fidelity.PairwiseInversionRate != 1 || fidelity.GoldF32Retention == nil || *fidelity.GoldF32Retention != 1 {
		t.Fatalf("fidelity=%+v err=%v", fidelity, err)
	}
	third := span("c.go", "C", 0, 5)
	tied := .25
	f32WithStrictScores := CaseRanking{QueryID: caseValue.ID, Variant: VariantTargetF32, Hits: []RetrievalHit{retrievalHit(left, 1, &first), retrievalHit(right, 2, &second), retrievalHit(third, 3, &tied)}}
	codecWithTies := CaseRanking{QueryID: caseValue.ID, Variant: VariantServingActiveCodec, Hits: []RetrievalHit{retrievalHit(right, 1, &tied), retrievalHit(left, 2, &tied), retrievalHit(third, 3, &tied)}}
	tiedFidelity, err := CompareCodecFidelity(caseValue, f32WithStrictScores, codecWithTies, 2)
	if err != nil || tiedFidelity.PairwiseInversionRate != nil || tiedFidelity.CodecScoreTieRate == nil || *tiedFidelity.CodecScoreTieRate != 1 || !tiedFidelity.CodecBoundaryTie || tiedFidelity.F32ScoreTieRate == nil || *tiedFidelity.F32ScoreTieRate != 0 || tiedFidelity.F32BoundaryTie {
		t.Fatalf("tied codec fidelity=%+v err=%v", tiedFidelity, err)
	}
	fts := CaseRanking{QueryID: caseValue.ID, Variant: VariantFTS, Hits: []RetrievalHit{retrievalHit(left, 1, &first)}}
	fused := CaseRanking{QueryID: caseValue.ID, Variant: VariantHybridFTSActiveCodec, Hits: []RetrievalHit{retrievalHit(left, 1, &first), retrievalHit(right, 2, &second)}}
	fusion, err := DiagnoseFusion(caseValue, fts, codec, fused, 2)
	if err != nil || !fusion.FusionRescue || fusion.FusionHarm || fusion.FTSOnlyCandidates != 0 || fusion.DenseOnlyCandidates != 1 {
		t.Fatalf("fusion=%+v err=%v", fusion, err)
	}
	bodyDigest := strings.Repeat("f", 64)
	body, err := DiagnoseBodyPackaging(caseValue, fused, []BodyPackageHit{
		{Hit: retrievalHit(left, 1, &first), BodyRange: &left, BodyComplete: true, BodyBytes: left.EndByte - left.StartByte, BodySHA256: bodyDigest},
		{Hit: retrievalHit(right, 2, &second), OmissionReason: "INLINE_BUDGET_EXCEEDED"},
	}, 2)
	if err != nil || body.FusedRequirementCoverage != 1 || body.PackagedRequirementCoverage != .5 || body.OmissionCounts["INLINE_BUDGET_EXCEEDED"] != 1 {
		t.Fatalf("body=%+v err=%v", body, err)
	}
	if _, err := DiagnoseBodyPackaging(caseValue, fused, []BodyPackageHit{{Hit: retrievalHit(left, 1, &first), BodyRange: &left, BodyBytes: 1}}, 2); err == nil {
		t.Fatal("inconsistent body byte accounting accepted")
	}
	if _, err := DiagnoseBodyPackaging(caseValue, fused, []BodyPackageHit{{Hit: retrievalHit(left, 1, &first), BodyRange: &left, BodyBytes: left.EndByte - left.StartByte}}, 2); err == nil {
		t.Fatal("full parent marked partial accepted")
	}
	if _, err := DiagnoseBodyPackaging(caseValue, fused, []BodyPackageHit{{Hit: retrievalHit(left, 1, &first), BodyRange: &left, BodyComplete: true, BodyBytes: left.EndByte - left.StartByte}}, 2); err == nil {
		t.Fatal("missing fused body omission record accepted")
	}
	if _, err := DiagnoseBodyPackaging(caseValue, fused, []BodyPackageHit{{Hit: retrievalHit(left, 1, &first), OmissionReason: "UNKNOWN"}, {Hit: retrievalHit(right, 2, &second), OmissionReason: "INLINE_BUDGET_EXCEEDED"}}, 2); err == nil {
		t.Fatal("unknown omission reason accepted")
	}
	overlap := left
	overlap.StartByte = 1
	caseWithOverlap := caseValue
	caseWithOverlap.Judgments = append(caseWithOverlap.Judgments, evalcontract.RelevanceJudgment{Span: overlap, Grade: 2, Rationale: "overlapping direct"})
	density, err := DiagnoseBodyPackaging(caseWithOverlap, fused, []BodyPackageHit{{Hit: retrievalHit(left, 1, &first), BodyRange: &left, BodyComplete: true, BodyBytes: 5, BodySHA256: bodyDigest}}, 1)
	if err != nil || density.RelevantByteDensity == nil || *density.RelevantByteDensity != 1 {
		t.Fatalf("density=%+v err=%v", density, err)
	}
	duplicateBodies, err := DiagnoseBodyPackaging(caseValue, fused, []BodyPackageHit{{Hit: retrievalHit(left, 1, &first), BodyRange: &left, BodyComplete: true, BodyBytes: 5, BodySHA256: bodyDigest}, {Hit: retrievalHit(right, 2, &second), BodyRange: &right, BodyComplete: true, BodyBytes: 5, BodySHA256: bodyDigest}}, 2)
	if err != nil || duplicateBodies.DuplicateBodyRatio != .5 {
		t.Fatalf("duplicate bodies=%+v err=%v", duplicateBodies, err)
	}
}

func TestRetrievalPlanAndCorePromotionEvidenceValidation(t *testing.T) {
	plan := DefaultRetrievalPlan([]int{1, 5})
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	plan.Variants = plan.Variants[:len(plan.Variants)-1]
	if err := plan.Validate(); err == nil {
		t.Fatal("missing retrieval arm accepted")
	}
	artifact := fixtureArtifact()
	paths := append([]string(nil), coreEvidencePaths...)
	entries := make([]evalcontract.ArtifactEntry, 0, len(paths))
	for index, path := range paths {
		sum := sha256.Sum256([]byte(path + string(rune(index))))
		entries = append(entries, evalcontract.ArtifactEntry{Path: path, MediaType: "application/json", ByteSize: 1, SHA256: hex.EncodeToString(sum[:])})
	}
	checksum, err := evalcontract.ArtifactChecksum(entries)
	if err != nil {
		t.Fatal(err)
	}
	contract := evalcontract.PromotionContract{SchemaVersion: 1, Scope: evalcontract.CoreRetrieval, CalibrationEvidenceSHA256: []string{strings.Repeat("a", 64)}, FrozenGates: []string{"correctness"}, ConfirmationDatasetSHA256: artifact.Manifest.QueryManifestSHA256, PairedControls: artifact.Manifest.PairedControls}
	result := evalcontract.PromotionResult{SchemaVersion: 1, Scope: evalcontract.CoreRetrieval, Status: evalcontract.PromotionEvidenceReady, PrerequisiteSHA256: []string{checksum}, PassedGates: []string{"correctness"}, ApplicableGates: []string{"correctness"}}
	evidence := CorePromotionEvidence{Contract: contract, Result: result, ConfirmationManifest: artifact.Manifest, Artifacts: evalcontract.ArtifactManifest{SchemaVersion: 1, Entries: entries, Complete: true}, ArtifactChecksum: checksum, CompletionArtifact: evalcontract.ArtifactEntry{Path: "artifact-checksums.json", MediaType: "application/json", ByteSize: 1, SHA256: strings.Repeat("a", 64)}}
	if err := evidence.Validate(); err != nil {
		t.Fatal(err)
	}
	evidence.Result.PassedGates = nil
	if err := evidence.Validate(); err == nil {
		t.Fatal("ready result missing passed gate accepted")
	}
	for _, omitted := range []string{"provider-usage.json", "report.md", "artifact-checksums.json"} {
		filtered := make([]evalcontract.ArtifactEntry, 0, len(entries)-1)
		for _, entry := range entries {
			if entry.Path != omitted {
				filtered = append(filtered, entry)
			}
		}
		filteredChecksum, err := evalcontract.ArtifactChecksum(filtered)
		if err != nil {
			t.Fatal(err)
		}
		missing := CorePromotionEvidence{Contract: contract, Result: result, ConfirmationManifest: artifact.Manifest, Artifacts: evalcontract.ArtifactManifest{SchemaVersion: 1, Entries: filtered, Complete: true}, ArtifactChecksum: filteredChecksum}
		missing.Result.PrerequisiteSHA256 = []string{filteredChecksum}
		if err := missing.Validate(); err == nil {
			t.Fatalf("promotion evidence missing %q accepted", omitted)
		}
	}
}

func TestRunRetrievalEvaluationOrchestratesEveryArmOnce(t *testing.T) {
	left, right := span("a.go", "A", 0, 5), span("b.go", "B", 0, 5)
	caseValue := fixtureCase("orchestration", evalcontract.Go, []evalcontract.RequiredGroup{
		{ID: "left", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{left}}}},
		{ID: "right", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{right}}}},
	}, []evalcontract.RelevanceJudgment{{Span: left, Grade: 2, Rationale: "direct"}, {Span: right, Grade: 2, Rationale: "direct"}})
	dataset := EvaluationDataset{SchemaVersion: 1, Version: "v1", CorpusID: "sample", Cases: []evalcontract.EvaluationCase{caseValue}}
	first, second := 1.0, .5
	queryHash := strings.Repeat("d", 64)
	arms := map[RetrievalVariant]RetrievalArmResult{}
	for _, variant := range requiredRetrievalVariants {
		hits := []RetrievalHit{retrievalHit(left, 1, &first), retrievalHit(right, 2, &second)}
		if variant == VariantServingActiveCodec {
			hits = []RetrievalHit{retrievalHit(right, 1, &first), retrievalHit(left, 2, &second)}
		}
		ranking := CaseRanking{QueryID: caseValue.ID, Variant: variant, Hits: hits}
		if variantUsesQueryVector(variant) {
			ranking.QueryVectorSHA256 = queryHash
		}
		arm := RetrievalArmResult{Ranking: ranking}
		if variant == VariantHybridFTSActiveCodec {
			arm.Packaged = []BodyPackageHit{{Hit: hits[0], OmissionReason: "INLINE_BUDGET_EXCEEDED"}, {Hit: hits[1], OmissionReason: "INLINE_BUDGET_EXCEEDED"}}
		}
		arms[variant] = arm
	}
	run, err := RunRetrievalEvaluation(context.Background(), dataset, DefaultRetrievalPlan([]int{1, 2}), fakeRetrievalExecutor{arms: arms})
	if err != nil || len(run.Cases) != 1 || len(run.Cases[0].Metrics) != len(requiredRetrievalVariants) || run.Cases[0].Fidelity.TopKRetention != 1 {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	badArms := make(map[RetrievalVariant]RetrievalArmResult, len(arms))
	for variant, arm := range arms {
		badArms[variant] = arm
	}
	bad := fakeRetrievalExecutor{arms: badArms}
	bad.arms[VariantHybridWithoutFTS] = RetrievalArmResult{Ranking: CaseRanking{QueryID: caseValue.ID, Variant: VariantHybridWithoutFTS, QueryVectorSHA256: strings.Repeat("e", 64)}}
	if _, err := RunRetrievalEvaluation(context.Background(), dataset, DefaultRetrievalPlan([]int{1}), bad); err == nil {
		t.Fatal("mismatched ephemeral query vector accepted")
	}
	failed, err := RunRetrievalEvaluation(context.Background(), dataset, DefaultRetrievalPlan([]int{1, 2}), fakeRetrievalExecutor{arms: arms, errors: map[RetrievalVariant]error{VariantProviderUnion: RetrievalArmFailure{Stage: evalcontract.FailureStage(evalcontract.StageProviderUnion)}}})
	if err != nil || failed.Cases[0].Metrics[3].Metrics.FailureStage != evalcontract.FailureStage(evalcontract.StageProviderUnion) {
		t.Fatalf("failed arm was not retained: run=%+v err=%v", failed, err)
	}
	failed.Cases[0].Fidelity.TopKRetention = .4
	if err := failed.Validate(dataset); err == nil {
		t.Fatal("forged retrieval evidence accepted")
	}
	fidelityFailed, err := RunRetrievalEvaluation(context.Background(), dataset, DefaultRetrievalPlan([]int{1, 2}), fakeRetrievalExecutor{arms: arms, errors: map[RetrievalVariant]error{VariantTargetF32: RetrievalArmFailure{Stage: evalcontract.FailureStage(evalcontract.StageDenseSegment)}}})
	if err != nil || fidelityFailed.Cases[0].Fidelity.Observed || fidelityFailed.Cases[0].Fidelity.FailureStage != evalcontract.FailureStage(evalcontract.StageDenseSegment) {
		t.Fatalf("failed prerequisite produced fidelity observation: run=%+v err=%v", fidelityFailed, err)
	}
	if err := (RetrievalArmResult{Ranking: CaseRanking{QueryID: caseValue.ID, Variant: VariantFTS}, FailureStage: evalcontract.FailureStage(evalcontract.StageDenseSegment)}).Validate(); err == nil {
		t.Fatal("fts arm accepted dense failure stage")
	}
	if err := (RetrievalArmResult{Ranking: CaseRanking{QueryID: caseValue.ID, Variant: VariantHybridFTSActiveCodec}, FailureStage: evalcontract.FailureStage(evalcontract.StageDenseSegment)}).Validate(); err != nil {
		t.Fatalf("full hybrid rejected upstream dense failure: %v", err)
	}
	if err := (RetrievalArmResult{Ranking: CaseRanking{QueryID: caseValue.ID, Variant: VariantHybridWithoutDense}, FailureStage: evalcontract.FailureStage(evalcontract.StageBodyPackaging)}).Validate(); err != nil {
		t.Fatalf("without-dense rejected body failure: %v", err)
	}
	if err := (RetrievalArmResult{Ranking: CaseRanking{QueryID: caseValue.ID, Variant: VariantHybridWithoutFTS}, FailureStage: evalcontract.FailureStage(evalcontract.StageFTSCandidate)}).Validate(); err == nil {
		t.Fatal("dense-only arm accepted fts failure stage")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	if _, err := RunRetrievalEvaluation(cancelled, dataset, DefaultRetrievalPlan([]int{1}), fakeRetrievalExecutor{arms: arms, cancel: cancel}); !errors.Is(err, context.Canceled) {
		t.Fatalf("post-arm cancellation err=%v", err)
	}
}

func TestRetrievalTraceAttributesPreCollapseSegmentLoss(t *testing.T) {
	left, right := span("a.go", "A", 0, 5), span("b.go", "B", 0, 5)
	caseValue := fixtureCase("segment-collapse", evalcontract.Go, []evalcontract.RequiredGroup{{ID: "required", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{left}}}}}, []evalcontract.RelevanceJudgment{{Span: left, Grade: evalcontract.DirectRequirement, Rationale: "direct"}})
	dataset := EvaluationDataset{SchemaVersion: 1, Version: "v1", CorpusID: "sample", Cases: []evalcontract.EvaluationCase{caseValue}}
	score, digest := 1.0, strings.Repeat("a", 64)
	arms := map[RetrievalVariant]RetrievalArmResult{}
	for _, variant := range requiredRetrievalVariants {
		hit := retrievalHit(left, 1, &score)
		if variant == VariantFTS || variant == VariantServingActiveCodec || variant == VariantProviderUnion || variant == VariantHybridFTSActiveCodec {
			hit = retrievalHit(right, 1, &score) // Collapsed parent misses the required span.
		}
		ranking := CaseRanking{QueryID: caseValue.ID, Variant: variant, Hits: []RetrievalHit{hit}}
		if variantUsesQueryVector(variant) {
			ranking.QueryVectorSHA256 = digest
		}
		arm := RetrievalArmResult{Ranking: ranking}
		if variant == VariantServingActiveCodec {
			arm.Segments = []search.EvaluationSegmentHit{{CanonicalInputSHA256: digest, Path: left.Path, IndexedSHA256: left.ContentSHA256, QualifiedSymbol: left.QualifiedSymbol, ParentStartByte: left.StartByte, ParentEndByte: left.EndByte, StartByte: left.StartByte, EndByte: left.EndByte, Rank: 1, Score: score}}
		}
		if variant == VariantHybridFTSActiveCodec {
			arm.Packaged = []BodyPackageHit{{Hit: hit, OmissionReason: "INLINE_BUDGET_EXCEEDED"}}
		}
		arms[variant] = arm
	}
	run, err := RunRetrievalEvaluation(context.Background(), dataset, DefaultRetrievalPlan([]int{1}), fakeRetrievalExecutor{arms: arms})
	if err != nil {
		t.Fatal(err)
	}
	trace, err := BuildRetrievalTrace(caseValue, run.Cases[0])
	if err != nil {
		t.Fatal(err)
	}
	dense, collapse := trace.Observations[3], trace.Observations[5]
	if dense.CandidateCount != 1 || !dense.GroupObservations[0].Present {
		t.Fatalf("dense segment evidence=%+v", dense)
	}
	if collapse.GroupObservations[0].Present || collapse.GroupObservations[0].FirstLoss != evalcontract.SegmentParentCollapse {
		t.Fatalf("collapse evidence=%+v", collapse)
	}
}

func TestRetrievalTraceKeepsFTSOnlyGroupThroughParentCollapse(t *testing.T) {
	left, right := span("a.go", "A", 0, 5), span("b.go", "B", 0, 5)
	caseValue := fixtureCase("fts-collapse", evalcontract.Go, []evalcontract.RequiredGroup{{ID: "required", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{left}}}}}, []evalcontract.RelevanceJudgment{{Span: left, Grade: evalcontract.DirectRequirement, Rationale: "direct"}})
	dataset := EvaluationDataset{SchemaVersion: 1, Version: "v1", CorpusID: "sample", Cases: []evalcontract.EvaluationCase{caseValue}}
	score, digest := 1.0, strings.Repeat("b", 64)
	arms := map[RetrievalVariant]RetrievalArmResult{}
	for _, variant := range requiredRetrievalVariants {
		hit := retrievalHit(left, 1, &score)
		if variant == VariantServingActiveCodec {
			hit = retrievalHit(right, 1, &score)
		}
		ranking := CaseRanking{QueryID: caseValue.ID, Variant: variant, Hits: []RetrievalHit{hit}}
		if variantUsesQueryVector(variant) {
			ranking.QueryVectorSHA256 = digest
		}
		arm := RetrievalArmResult{Ranking: ranking}
		if variant == VariantHybridFTSActiveCodec {
			arm.Packaged = []BodyPackageHit{{Hit: hit, OmissionReason: "INLINE_BUDGET_EXCEEDED"}}
		}
		arms[variant] = arm
	}
	run, err := RunRetrievalEvaluation(context.Background(), dataset, DefaultRetrievalPlan([]int{1}), fakeRetrievalExecutor{arms: arms})
	if err != nil {
		t.Fatal(err)
	}
	trace, err := BuildRetrievalTrace(caseValue, run.Cases[0])
	if err != nil {
		t.Fatal(err)
	}
	providerUnion, collapse := trace.Observations[4], trace.Observations[5]
	if !providerUnion.GroupObservations[0].Present || !collapse.GroupObservations[0].Present {
		t.Fatalf("fts-only group was lost: union=%+v collapse=%+v", providerUnion, collapse)
	}
	failedRun, err := RunRetrievalEvaluation(context.Background(), dataset, DefaultRetrievalPlan([]int{1}), fakeRetrievalExecutor{arms: arms, errors: map[RetrievalVariant]error{VariantServingActiveCodec: RetrievalArmFailure{Stage: evalcontract.FailureStage(evalcontract.StageDenseSegment)}}})
	if err != nil {
		t.Fatal(err)
	}
	failedTrace, err := BuildRetrievalTrace(caseValue, failedRun.Cases[0])
	if err != nil {
		t.Fatal(err)
	}
	providerUnion = failedTrace.Observations[4]
	if providerUnion.CandidateCount != 1 || !providerUnion.GroupObservations[0].Present {
		t.Fatalf("fts-only union with dense failure=%+v", providerUnion)
	}
}

func retrievalHit(value evalcontract.SourceSpan, rank int, score *float64) RetrievalHit {
	return RetrievalHit{Path: value.Path, IndexedSHA256: value.ContentSHA256, QualifiedSymbol: value.QualifiedSymbol, StartByte: value.StartByte, EndByte: value.EndByte, Rank: rank, Score: score}
}

type fakeInventory struct{ snapshot TruthInventorySnapshot }

func (value fakeInventory) Snapshot(context.Context) (TruthInventorySnapshot, error) {
	return value.snapshot, nil
}

type fakeSearch struct {
	result lexical.Result
	err    error
}

type fakeRetrievalExecutor struct {
	arms   map[RetrievalVariant]RetrievalArmResult
	errors map[RetrievalVariant]error
	cancel func()
}

func (value fakeRetrievalExecutor) EvaluateArm(_ context.Context, _ evalcontract.EvaluationCase, variant RetrievalVariant) (RetrievalArmResult, error) {
	if value.cancel != nil {
		value.cancel()
	}
	return value.arms[variant], value.errors[variant]
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
