package evalcontract

import "testing"

func TestSmokeTraceAndArtifactFraming(t *testing.T) {
	trace := allStageSmokeTrace()
	if err := trace.Validate(); err != nil {
		t.Fatal(err)
	}
	digestA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestB := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	first, err := ArtifactChecksum([]ArtifactEntry{{Path: "trace.json", MediaType: "application/json", ByteSize: 1, SHA256: digestA}, {Path: "report.md", MediaType: "text/markdown", ByteSize: 2, SHA256: digestB}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ArtifactChecksum([]ArtifactEntry{{Path: "report.md", MediaType: "text/markdown", ByteSize: 2, SHA256: digestB}, {Path: "trace.json", MediaType: "application/json", ByteSize: 1, SHA256: digestA}})
	if err != nil || first != second {
		t.Fatalf("artifact checksum not deterministic: %s %s %v", first, second, err)
	}
}

func TestArtifactPathsAndDigestsArePortableAndStrict(t *testing.T) {
	digest := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := ArtifactChecksum([]ArtifactEntry{{Path: "safe..name.json", MediaType: "application/json", ByteSize: 1, SHA256: digest}}); err != nil {
		t.Fatalf("harmless dot-dot substring rejected: %v", err)
	}
	for _, path := range []string{"../escape", "a/../escape", "/absolute", "a\\escape"} {
		if _, err := ArtifactChecksum([]ArtifactEntry{{Path: path, MediaType: "text/plain", ByteSize: 0, SHA256: digest}}); err == nil {
			t.Fatalf("unsafe path accepted: %q", path)
		}
	}
	if _, err := ArtifactChecksum([]ArtifactEntry{{Path: "valid", MediaType: "text/plain", ByteSize: 0, SHA256: "ABC"}}); err == nil {
		t.Fatal("noncanonical digest accepted")
	}
	if _, err := ArtifactChecksum([]ArtifactEntry{{Path: "same.json", MediaType: "application/json", ByteSize: 0, SHA256: digest}, {Path: "same.json", MediaType: "application/json", ByteSize: 0, SHA256: digest}}); err == nil {
		t.Fatal("duplicate artifact path accepted")
	}
	if err := (ArtifactManifest{SchemaVersion: SchemaVersion, Entries: []ArtifactEntry{{Path: "same.json", MediaType: "application/json", ByteSize: 0, SHA256: digest}, {Path: "same.json", MediaType: "application/json", ByteSize: 0, SHA256: digest}}}).Validate(); err == nil {
		t.Fatal("artifact manifest accepted duplicate path")
	}
}

func allStageSmokeTrace() StageTrace {
	observations := make([]StageObservation, 0, len(PlannedStages))
	for index, stage := range PlannedStages {
		if stage == StageOperational {
			observations = append(observations, StageObservation{Stage: stage, Required: true, Status: Observed, Denominators: []DenominatorRecord{{Name: "operation_attempts", TruthUnit: "operation", Count: 1}}, CandidateCount: 0})
		} else if index < 8 {
			observations = append(observations, StageObservation{Stage: stage, Required: true, Status: Observed, Denominators: []DenominatorRecord{{Name: "required_groups", TruthUnit: "required group", Count: 1}}, GroupObservations: []GroupObservation{{GroupID: "g1", Present: true, FirstLoss: NoLoss}}, CandidateCount: 0})
		} else {
			observations = append(observations, StageObservation{Stage: stage, Required: false, Status: ObservationNotObserved, CandidateCount: 0})
		}
	}
	return StageTrace{SchemaVersion: SchemaVersion, QueryID: "q1", RequiredGroupIDs: []string{"g1"}, TerminalState: TerminalComplete, Observations: observations}
}

func TestEvaluationCaseAbstainableReviewAndGroupTraceContracts(t *testing.T) {
	zero := 0
	base := EvaluationCase{SchemaVersion: SchemaVersion, ID: "q", Text: "none", Language: Go, Cohorts: []string{"hard-negative"}, AnswerMode: Abstainable, ExpectedCardinality: &zero, Split: Confirmation, RequiredConstraints: RequiredConstraints{Identifiers: []string{"none"}, Paths: []string{"pkg/file.go"}, Languages: []Language{Go}, Scopes: []string{"repository"}}, HardNegatives: []HardNegative{{Span: testSpan(), Reason: "reviewed no answer"}}, Judgments: []RelevanceJudgment{{Span: testSpan(), Grade: Irrelevant, Rationale: "not relevant"}}, Review: ReviewRecord{State: ReviewFrozen, Passes: []ReviewPass{{ID: "one", Reviewer: "reviewer"}, {ID: "two", Reviewer: "reviewer"}}, Rationale: "two solo passes", SoloReviewLimitation: "one reviewer"}, Digest: testDigest()}
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	base.Cohorts = []string{"hard-negative", "hard-negative"}
	if err := base.Validate(); err == nil {
		t.Fatal("duplicate cohort accepted")
	}
	base.Cohorts = []string{"hard-negative"}
	base.Digest = "not-a-digest"
	if err := base.Validate(); err == nil {
		t.Fatal("noncanonical case digest accepted")
	}
	base.Digest = testDigest()
	base.ExpectedCardinality = nil
	if err := base.Validate(); err != nil {
		t.Fatalf("abstainable case with omitted cardinality rejected: %v", err)
	}
	base.ExpectedCardinality = &zero
	base.Review.SoloReviewLimitation = ""
	if err := base.Validate(); err == nil {
		t.Fatal("frozen same-reviewer passes accepted without limitation")
	}
	trace := allStageSmokeTrace()
	trace.Observations[4].GroupObservations[0] = GroupObservation{GroupID: "g1", Present: false, FirstLoss: ProviderUnionMiss}
	trace.Observations[5].GroupObservations[0] = GroupObservation{GroupID: "g1", Present: true, FirstLoss: NoLoss}
	if err := trace.Validate(); err == nil {
		t.Fatal("group reappeared after provider union loss")
	}
	trace = allStageSmokeTrace()
	trace.Observations[0].GroupObservations[0] = GroupObservation{GroupID: "g1", Present: false, FirstLoss: SourceDiscovery}
	if err := trace.Validate(); err == nil {
		t.Fatal("group reappeared after source discovery loss")
	}
	failure := allStageSmokeTrace()
	failure.Observations[4].FailureStage = FailureStage(StageProviderUnion)
	failure.Observations[4].GroupObservations[0] = GroupObservation{GroupID: "g1", Present: false, FirstLoss: "OPERATION_FAILURE:provider_union"}
	for index := 5; index < 8; index++ {
		failure.Observations[index].FailureStage = FailureStage(StageProviderUnion)
		failure.Observations[index].GroupObservations[0] = GroupObservation{GroupID: "g1", Present: false, FirstLoss: "OPERATION_FAILURE:provider_union"}
	}
	if err := failure.Validate(); err != nil {
		t.Fatalf("operation failure trace rejected: %v", err)
	}
	failure.Observations[4].FailureStage = FailureStage(StageRRFFusion)
	if err := failure.Validate(); err == nil {
		t.Fatal("operation failure loss accepted with wrong failure stage")
	}
	abstainFailure := allStageSmokeTrace()
	abstainFailure.RequiredGroupIDs = nil
	for index := range abstainFailure.Observations {
		if abstainFailure.Observations[index].Required {
			abstainFailure.Observations[index].GroupObservations = nil
		}
	}
	abstainFailure.Observations[0].FailureStage = FailureStage(StageSourceDiscovery)
	if err := abstainFailure.Validate(); err != nil {
		t.Fatalf("abstainable operation failure trace rejected: %v", err)
	}
	denominator := allStageSmokeTrace()
	denominator.Observations[0].Denominators = append(denominator.Observations[0].Denominators, denominator.Observations[0].Denominators[0])
	if err := denominator.Validate(); err == nil {
		t.Fatal("duplicate denominator accepted")
	}
	optional := allStageSmokeTrace()
	optional.Observations[8].CandidateCount = 1
	if err := optional.Validate(); err == nil {
		t.Fatal("not-observed candidate count accepted")
	}
	span := testSpan()
	span.EndByte = span.StartByte
	if err := span.Validate(); err == nil {
		t.Fatal("empty source span accepted")
	}
}

func TestEvaluationCaseJudgmentGradeRelationships(t *testing.T) {
	cardinality := 1
	required, negative, support := testSpan(), testSpan(), testSpan()
	negative.Path, negative.QualifiedSymbol = "pkg/other.go", "pkg.Other"
	support.Path, support.QualifiedSymbol = "pkg/support.go", "pkg.Support"
	caseValue := EvaluationCase{SchemaVersion: SchemaVersion, ID: "answerable", Text: "find F", Language: Go, Cohorts: []string{"identifier"}, AnswerMode: Single, ExpectedCardinality: &cardinality, Split: Calibration, RequiredConstraints: RequiredConstraints{Identifiers: []string{"F"}, Paths: []string{"pkg/file.go"}, Languages: []Language{Go}, Scopes: []string{"declaration"}}, RequiredGroups: []RequiredGroup{{ID: "g1", Alternatives: []ExpectedAlternative{{Spans: []SourceSpan{required}}}}}, HardNegatives: []HardNegative{{Span: negative, Reason: "wrong symbol"}}, Judgments: []RelevanceJudgment{{Span: required, Grade: DirectRequirement, Rationale: "required"}, {Span: negative, Grade: Irrelevant, Rationale: "negative"}, {Span: support, Grade: UsefulSupport, Rationale: "support"}}, Review: ReviewRecord{State: ReviewFrozen, Passes: []ReviewPass{{ID: "one", Reviewer: "a"}, {ID: "two", Reviewer: "b"}}, Rationale: "reviewed"}, Digest: testDigest()}
	if err := caseValue.Validate(); err != nil {
		t.Fatal(err)
	}
	caseValue.RequiredConstraints.Identifiers = []string{"F", "F"}
	if err := caseValue.Validate(); err == nil {
		t.Fatal("duplicate required constraint accepted")
	}
	caseValue.RequiredConstraints.Identifiers = []string{"F"}
	caseValue.Judgments[0].Grade = UsefulSupport
	if err := caseValue.Validate(); err == nil {
		t.Fatal("required span accepted without grade 2")
	}
}

func testDigest() string { return "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" }

func testSpan() SourceSpan {
	return SourceSpan{Path: "pkg/file.go", ContentSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", QualifiedSymbol: "pkg.F", StartByte: 0, EndByte: 1}
}
