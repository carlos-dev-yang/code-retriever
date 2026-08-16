package devlab

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
	"sync"
	"testing"

	"cidx/internal/app"
	"cidx/internal/config"
	"cidx/internal/devapp"
	"cidx/internal/embedclient"
	"cidx/internal/eval"
	"cidx/internal/evalcontract"
	"cidx/internal/index"
	"cidx/internal/lab"
	"cidx/internal/search"
	"cidx/internal/store"
)

func TestRetrievalAdapterPlanApplyAndArtifactAreLocalSafe(t *testing.T) {
	ctx, prepared, root, raw := retrievalFixture(t)
	defer raw.Close()
	if plan := prepared.Plan(); plan.QueryCount != 1 || plan.LogicalQueryOperationsPlanned != 1 {
		t.Fatalf("plan=%+v", plan)
	}
	assertEvaluationRows(t, raw, 0)
	if _, err := os.Stat(filepath.Join(root, ".cidx", "evaluations")); !os.IsNotExist(err) {
		t.Fatalf("plan created artifact state: %v", err)
	}
	client := &adapterFakeClient{response: adapterQueryResponse(prepared.application.Resolved)}
	applied, err := prepared.Apply(ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	if client.Count() != 1 || len(applied.Run.Cases) != 1 {
		t.Fatalf("calls=%d run=%+v", client.Count(), applied.Run)
	}
	caseEvidence := applied.Run.Cases[0]
	if len(caseEvidence.Arms) != 8 || len(caseEvidence.Arms[5].Packaged) == 0 {
		t.Fatalf("arms=%+v", caseEvidence.Arms)
	}
	var digest string
	for _, arm := range caseEvidence.Arms {
		if arm.Ranking.Variant == eval.VariantFTS || arm.Ranking.Variant == eval.VariantHybridWithoutDense {
			if arm.Ranking.QueryVectorSHA256 != "" {
				t.Fatalf("FTS-only arm carried query digest: %+v", arm)
			}
			continue
		}
		if digest == "" {
			digest = arm.Ranking.QueryVectorSHA256
		}
		if digest == "" || arm.Ranking.QueryVectorSHA256 != digest {
			t.Fatalf("query digest mismatch: %+v", caseEvidence.Arms)
		}
	}
	if applied.Artifact.RunID == "" || applied.Artifact.Reference == "" || len(applied.Artifact.Checksum) != 64 {
		t.Fatalf("artifact=%+v", applied.Artifact)
	}
	artifactRoot := filepath.Join(root, ".cidx", filepath.FromSlash(applied.Artifact.Reference))
	for _, name := range requiredRetrievalArtifactFiles {
		if _, err := os.Stat(filepath.Join(artifactRoot, name)); err != nil {
			t.Fatalf("artifact missing %s: %v", name, err)
		}
	}
	manifestBytes, err := os.ReadFile(filepath.Join(artifactRoot, "run-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	providerUsage, ok := manifest["provider_usage"].(map[string]any)
	if !ok || providerUsage["logical_query_operations"] != float64(1) || providerUsage["query_operations"] != nil {
		t.Fatalf("run manifest leaked provider operation detail: %#v", manifest["provider_usage"])
	}
	all, err := os.ReadFile(filepath.Join(artifactRoot, "per-query-trace.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(all), "synthetic query text") || strings.Contains(string(all), "func FindThing") {
		t.Fatal("portable artifact copied query text or source body")
	}
	assertEvaluationRows(t, raw, 1)
}

func TestRetrievalExperimentControlsRequireCompleteFingerprintAndBudget(t *testing.T) {
	_, prepared, _, raw := retrievalFixture(t)
	defer raw.Close()
	policy, err := search.SafeTokenOREvaluationPolicy(prepared.application.Resolved)
	if err != nil {
		t.Fatal(err)
	}
	valid := RetrievalExperimentOptions{FTSPolicy: &policy, ExperimentSeriesID: "series-1", SeriesQueryOperationsPlanned: 32, AuthorizationReference: "user-thread-authorization", USDCap: 5, PricingTableIdentity: "voyage-code-4-price-2026-08-16", USDPerMillionTokens: 0.12}
	if err := validateRetrievalExperiment(prepared.application, valid, 12, 10_000); err != nil {
		t.Fatalf("valid experiment rejected: %v", err)
	}
	incomplete := valid
	incomplete.AuthorizationReference = ""
	if err := validateRetrievalExperiment(prepared.application, incomplete, 12, 10_000); err == nil {
		t.Fatal("incomplete experiment authority accepted")
	}
	forged := valid
	forged.FTSPolicy = &search.EvaluationFTSPolicy{}
	if err := validateRetrievalExperiment(prepared.application, forged, 12, 10_000); err == nil {
		t.Fatal("forged FTS policy accepted")
	}
	overBudget := valid
	overBudget.USDCap = 0.000001
	if err := validateRetrievalExperiment(prepared.application, overBudget, 12, 10_000); err == nil {
		t.Fatal("over-budget experiment accepted")
	}
	if err := validateRetrievalExperiment(prepared.application, RetrievalExperimentOptions{ExperimentSeriesID: "metadata-without-policy"}, 12, 10_000); err == nil {
		t.Fatal("experiment metadata without FTS policy accepted")
	}
}

func TestRetrievalArtifactCompensationRejectsMalformedRunID(t *testing.T) {
	ctx, prepared, root, raw := retrievalFixture(t)
	defer raw.Close()
	if err := removeRetrievalArtifact(ctx, prepared, RetrievalArtifactReference{RunID: "retrieval-../../outside", Reference: "evaluations/retrieval-../../outside"}); err == nil {
		t.Fatal("malformed compensation target accepted")
	}
	if _, err := os.Stat(filepath.Join(root, ".cidx", "evaluations", "outside")); !os.IsNotExist(err) {
		t.Fatalf("malformed compensation touched outside target: %v", err)
	}
}

func TestRetrievalAdapterRetainsProviderFailureButAbortsInvariantFailure(t *testing.T) {
	ctx, prepared, root, raw := retrievalFixture(t)
	defer raw.Close()
	failed, err := prepared.Apply(ctx, &adapterFakeClient{err: errors.New("provider unavailable")})
	if err != nil {
		t.Fatal(err)
	}
	if len(failed.Run.Cases) != 1 || failed.Run.Cases[0].Metrics[1].Metrics.FailureStage != evalcontract.FailureStage(evalcontract.StageOperational) {
		t.Fatalf("provider failure did not remain denominator evidence: %+v", failed.Run)
	}
	if _, err := os.Stat(filepath.Join(root, ".cidx", filepath.FromSlash(failed.Artifact.Reference), "promotion-result.json")); err != nil {
		t.Fatal(err)
	}
	usageBytes, err := os.ReadFile(filepath.Join(root, ".cidx", filepath.FromSlash(failed.Artifact.Reference), "provider-usage.json"))
	if err != nil {
		t.Fatal(err)
	}
	var usage retrievalProviderUsage
	if err := json.Unmarshal(usageBytes, &usage); err != nil {
		t.Fatal(err)
	}
	if len(usage.QueryOperations) != 1 || usage.QueryOperations[0].TerminalStatus != providerUsageTerminalFailed || usage.QueryOperations[0].ObservedTotalTokens != nil || usage.Aggregate.ObservedTotalTokens != nil || usage.Aggregate.TokenObservedAttempts != 0 {
		t.Fatalf("terminal failure usage=%+v", usage)
	}
	assertEvaluationRows(t, raw, 1)

	_, prepared, _, raw = retrievalFixture(t)
	defer raw.Close()
	_, err = prepared.Apply(ctx, &adapterFakeClient{response: embedclient.EmbeddingResponse{Model: "wrong", Data: []embedclient.EmbeddingDatum{{Index: 0, IndexPresent: true, Values: adapterSourceVector()}}, TotalTokens: 1}})
	if err == nil {
		t.Fatal("malformed provider response became a denominator failure")
	}
	assertEvaluationRows(t, raw, 0)
}

func TestRetrievalAdapterCancellationDoesNotPublish(t *testing.T) {
	ctx, prepared, root, raw := retrievalFixture(t)
	defer raw.Close()
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := prepared.Apply(cancelled, &adapterFakeClient{response: adapterQueryResponse(prepared.application.Resolved)}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation err=%v", err)
	}
	assertEvaluationRows(t, raw, 0)
	if _, err := os.Stat(filepath.Join(root, ".cidx", "evaluations")); !os.IsNotExist(err) {
		t.Fatalf("cancellation published artifact state: %v", err)
	}
}

func TestRetrievalAdapterPerAttemptDeadlineRemainsProviderFailure(t *testing.T) {
	ctx, prepared, root, raw := retrievalFixture(t)
	defer raw.Close()
	// Zero retries makes this focused test an exhausted first attempt without
	// duplicating the shared executor's configured backoff timing matrix.
	prepared.application.Resolved.Embedding.Retry.MaxRetries = 0
	prepared.application.Resolved.Embedding.Retry.WaitSeconds = nil
	deadline := embedclient.ProviderError{Class: "timeout", Retryable: false, Cause: context.DeadlineExceeded}
	applied, err := prepared.Apply(ctx, &adapterFakeClient{err: deadline})
	if err != nil {
		t.Fatal(err)
	}
	if len(applied.Run.Cases) != 1 || applied.Run.Cases[0].Metrics[1].Metrics.FailureStage != evalcontract.FailureStage(evalcontract.StageOperational) {
		t.Fatalf("attempt deadline did not remain operational evidence: %+v", applied.Run)
	}
	usageBytes, err := os.ReadFile(filepath.Join(root, ".cidx", filepath.FromSlash(applied.Artifact.Reference), "provider-usage.json"))
	if err != nil {
		t.Fatal(err)
	}
	var usage retrievalProviderUsage
	if err := json.Unmarshal(usageBytes, &usage); err != nil {
		t.Fatal(err)
	}
	if len(usage.QueryOperations) != 1 || usage.QueryOperations[0].TerminalStatus != providerUsageTerminalFailed || usage.QueryOperations[0].ProviderAttempts != 1 || usage.QueryOperations[0].ObservedTotalTokens != nil {
		t.Fatalf("attempt deadline usage=%+v", usage)
	}
	assertEvaluationRows(t, raw, 1)
}

func TestProviderUsageOperationAccountingAndWireValidation(t *testing.T) {
	ctx, prepared, _, raw := retrievalFixture(t)
	defer raw.Close()
	applied, err := prepared.Apply(ctx, &adapterFakeClient{response: adapterQueryResponse(prepared.application.Resolved)})
	if err != nil {
		t.Fatal(err)
	}
	dataset := prepared.dataset
	second := dataset.Cases[0]
	second.ID = "q2"
	dataset.Cases = append(dataset.Cases, second)
	run := applied.Run
	secondEvidence := run.Cases[0]
	secondEvidence.Case.QueryID = "q2"
	run.Cases = append(run.Cases, secondEvidence)

	zero := 0
	three := 3
	usage := newRetrievalProviderUsage(prepared, nil).usage
	usage.QueryOperations = []retrievalProviderQueryOperation{
		{LogicalOperationID: 1, QueryID: "q1", LogicalQueryOperations: 1, ProviderAttempts: 1, Retries: 0, ValidatedResponses: 1, FailedAttempts: 0, TerminalStatus: providerUsageTerminalValidated, ObservedTotalTokens: &zero, TokenObservedAttempts: 1, TokenAccountingComplete: true},
		{LogicalOperationID: 2, QueryID: "q2", LogicalQueryOperations: 1, ProviderAttempts: 2, Retries: 1, ValidatedResponses: 1, FailedAttempts: 1, TerminalStatus: providerUsageTerminalValidated, ObservedTotalTokens: &three, TokenObservedAttempts: 1, TokenAccountingComplete: false},
	}
	usage.Aggregate = deriveProviderUsageAggregate(usage.QueryOperations)
	if err := usage.Validate(prepared, dataset, run); err != nil {
		t.Fatalf("valid mixed usage rejected: %v", err)
	}
	if usage.Aggregate.ObservedTotalTokens == nil || *usage.Aggregate.ObservedTotalTokens != 3 || usage.Aggregate.TokenAccountingComplete || usage.Aggregate.GeneratedResponseTokens != generatedResponseTokensNotApplied {
		t.Fatalf("mixed aggregate=%+v", usage.Aggregate)
	}
	mutations := []func(*retrievalProviderUsage){
		func(value *retrievalProviderUsage) { value.AttemptTimeoutSeconds++ },
		func(value *retrievalProviderUsage) {
			value.QueryOperations[0], value.QueryOperations[1] = value.QueryOperations[1], value.QueryOperations[0]
		},
		func(value *retrievalProviderUsage) { value.QueryOperations[1].QueryID = "q1" },
		func(value *retrievalProviderUsage) { value.QueryOperations[1].LogicalOperationID = 3 },
		func(value *retrievalProviderUsage) { value.Aggregate.ProviderAttempts++ },
		func(value *retrievalProviderUsage) { value.QueryOperations[1].ObservedTotalTokens = nil },
		func(value *retrievalProviderUsage) { value.QueryOperations[1].TokenAccountingComplete = true },
	}
	for _, mutate := range mutations {
		invalid := usage
		invalid.QueryOperations = append([]retrievalProviderQueryOperation(nil), usage.QueryOperations...)
		mutate(&invalid)
		if err := invalid.Validate(prepared, dataset, run); err == nil {
			t.Fatal("forged provider usage wire accepted")
		}
	}
}

func TestProviderUsageOperationRecordsRetriesFailuresAndPostSuccessAttempts(t *testing.T) {
	request := embedclient.EmbeddingRequest{}
	valid := adapterQueryResponse(adapterConfig(t))
	zeroResponse := valid
	zeroResponse.TotalTokens = 0
	immediate := &operationRecordingClient{delegate: &sequenceEmbeddingClient{steps: []sequenceEmbeddingStep{{response: zeroResponse}}}, queryID: "q1"}
	if _, err := immediate.Embed(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	completed, err := immediate.finalize(context.Background(), nil)
	if err != nil || completed.ObservedTotalTokens == nil || *completed.ObservedTotalTokens != 0 || !completed.TokenAccountingComplete {
		t.Fatalf("immediate zero-token success=%+v err=%v", completed, err)
	}
	retry := &sequenceEmbeddingClient{steps: []sequenceEmbeddingStep{{err: errors.New("retry")}, {response: valid}}}
	operation := &operationRecordingClient{delegate: retry, queryID: "q1"}
	if _, err := operation.Embed(context.Background(), request); err == nil {
		t.Fatal("first transient failure unexpectedly succeeded")
	}
	if _, err := operation.Embed(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	completed, err = operation.finalize(context.Background(), nil)
	if err != nil || completed.ProviderAttempts != 2 || completed.Retries != 1 || completed.FailedAttempts != 1 || completed.ObservedTotalTokens == nil || *completed.ObservedTotalTokens != 3 || completed.TokenAccountingComplete {
		t.Fatalf("retry-success accounting=%+v err=%v", completed, err)
	}

	failed := &operationRecordingClient{delegate: &sequenceEmbeddingClient{steps: []sequenceEmbeddingStep{{err: errors.New("failed")}}}, queryID: "q1"}
	if _, err := failed.Embed(context.Background(), request); err == nil {
		t.Fatal("terminal failure unexpectedly succeeded")
	}
	completed, err = failed.finalize(context.Background(), search.QueryEmbeddingProviderError{Err: errors.New("failed")})
	if err != nil || completed.TerminalStatus != providerUsageTerminalFailed || completed.ObservedTotalTokens != nil || completed.TokenObservedAttempts != 0 || completed.TokenAccountingComplete {
		t.Fatalf("terminal failure accounting=%+v err=%v", completed, err)
	}

	success := &operationRecordingClient{delegate: &sequenceEmbeddingClient{steps: []sequenceEmbeddingStep{{response: valid}}}, queryID: "q1"}
	if _, err := success.Embed(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if _, err := success.Embed(context.Background(), request); err == nil {
		t.Fatal("attempt after successful response accepted")
	}
	if _, err := success.finalize(context.Background(), errors.New("post-success attempt")); err == nil {
		t.Fatal("post-success attempt did not abort publication")
	}
}

var requiredRetrievalArtifactFiles = []string{"run-manifest.json", "per-query-trace.jsonl", "fts-candidates.jsonl", "dense-segment-candidates.jsonl", "collapsed-parent-candidates.jsonl", "rrf-results.jsonl", "inline-body-packages.jsonl", "per-query-metrics.jsonl", "aggregate-metrics.json", "cohort-language-report.json", "first-loss-report.json", "provider-usage.json", "implementation-audit.json", "promotion-contract.json", "promotion-result.json", "report.md", "artifact-checksums.json"}

type adapterFakeClient struct {
	mu       sync.Mutex
	calls    int
	response embedclient.EmbeddingResponse
	err      error
}

type sequenceEmbeddingStep struct {
	response embedclient.EmbeddingResponse
	err      error
}

type sequenceEmbeddingClient struct {
	steps []sequenceEmbeddingStep
}

func (client *sequenceEmbeddingClient) Embed(context.Context, embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	if len(client.steps) == 0 {
		return embedclient.EmbeddingResponse{}, errors.New("unexpected provider call")
	}
	step := client.steps[0]
	client.steps = client.steps[1:]
	return step.response, step.err
}

func (client *adapterFakeClient) Embed(_ context.Context, _ embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.calls++
	return client.response, client.err
}
func (client *adapterFakeClient) Count() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.calls
}

func retrievalFixture(t *testing.T) (context.Context, retrievalPrepared, string, *lab.Store) {
	t.Helper()
	ctx, root := context.Background(), t.TempDir()
	runAdapter(t, root, "git", "init")
	runAdapter(t, root, "git", "config", "user.email", "test@example.invalid")
	runAdapter(t, root, "git", "config", "user.name", "test")
	const source = "package p\n\nfunc FindThing() {}\n"
	mustWriteAdapter(t, filepath.Join(root, "a.go"), source)
	mustWriteAdapter(t, filepath.Join(root, "LICENSE"), "MIT\n")
	runAdapter(t, root, "git", "add", "a.go", "LICENSE")
	runAdapter(t, root, "git", "commit", "-m", "fixture")
	runAdapter(t, root, "git", "remote", "add", "origin", "https://example.invalid/acme/repo.git")
	resolved := adapterConfig(t)
	data, err := json.Marshal(resolvedRaw(t))
	if err != nil {
		t.Fatal(err)
	}
	mustWriteAdapter(t, filepath.Join(root, ".cidx", "config.json"), string(data))
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
		production.Close()
		t.Fatal(err)
	}
	if err := production.Close(); err != nil {
		t.Fatal(err)
	}
	application, err := app.OpenLocal(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close() })
	raw, err := lab.OpenStore(ctx, lab.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	materializer := devapp.Materialize{Production: application.Store, Lab: raw, Resolved: application.Resolved}
	materialization, err := materializer.Plan(ctx)
	if err != nil {
		raw.Close()
		t.Fatal(err)
	}
	vector, err := lab.NewF32Vector(adapterSourceVector(), 1024)
	if err != nil {
		raw.Close()
		t.Fatal(err)
	}
	for _, key := range materialization.RequiredKeys() {
		if err := raw.PutDocumentSource(ctx, lab.DocumentRaw{SourceProfile: string(application.Resolved.Profiles.Fingerprints.Source), InputHash: key, RequestedModel: embedclient.Model, ResponseModel: embedclient.Model, Vector: vector}, 1024); err != nil {
			raw.Close()
			t.Fatal(err)
		}
	}
	materialization, err = materializer.Plan(ctx)
	if err != nil {
		raw.Close()
		t.Fatalf("materialization replan: %v", err)
	}
	if _, err := materializer.Activate(ctx, materialization); err != nil {
		raw.Close()
		t.Fatalf("materialization activate: %v", err)
	}
	active, err := application.Store.ActiveVectorPlanningSnapshot(ctx)
	if err != nil {
		raw.Close()
		t.Fatal(err)
	}
	covered, err := raw.ExistingKeys(ctx, string(application.Resolved.Profiles.Fingerprints.Source), active.CanonicalInputs)
	if err != nil || len(covered) != len(active.CanonicalInputs) {
		raw.Close()
		t.Fatalf("raw coverage keys=%v covered=%v err=%v", active.CanonicalInputs, covered, err)
	}
	inventory, err := (eval.ProductionTruthInventory{Store: application.Store}).Snapshot(ctx)
	if err != nil || len(inventory.Chunks) != 1 {
		raw.Close()
		t.Fatalf("inventory=%+v err=%v", inventory, err)
	}
	truth := inventory.Chunks[0]
	span := evalcontract.SourceSpan{Path: truth.Path, ContentSHA256: truth.IndexedSHA256, QualifiedSymbol: truth.QualifiedSymbol, StartByte: truth.StartByte, EndByte: truth.EndByte}
	caseValue := evalcontract.EvaluationCase{SchemaVersion: 1, ID: "q1", Text: "synthetic query text", Language: evalcontract.Go, Cohorts: []string{"identifier"}, AnswerMode: evalcontract.Single, Split: evalcontract.Calibration, RequiredConstraints: evalcontract.RequiredConstraints{Identifiers: []string{"FindThing"}, Paths: []string{"a.go"}, Languages: []evalcontract.Language{evalcontract.Go}, Scopes: []string{"repository"}}, RequiredGroups: []evalcontract.RequiredGroup{{ID: "g1", Alternatives: []evalcontract.ExpectedAlternative{{Spans: []evalcontract.SourceSpan{span}}}}}, Judgments: []evalcontract.RelevanceJudgment{{Span: span, Grade: evalcontract.DirectRequirement, Rationale: "fixture"}}, Review: evalcontract.ReviewRecord{State: evalcontract.ReviewDraft, Passes: []evalcontract.ReviewPass{{ID: "pass1", Reviewer: "fixture"}}, Rationale: "fixture"}, Digest: strings.Repeat("a", 64)}
	dataset := eval.EvaluationDataset{SchemaVersion: 1, Version: "v1", CorpusID: "fixture", Cases: []evalcontract.EvaluationCase{caseValue}}
	datasetPath := filepath.Join(root, "dataset.json")
	datasetData, _ := json.Marshal(dataset)
	mustWriteAdapter(t, datasetPath, string(datasetData))
	manifest := adapterManifest(t, root)
	manifestPath := filepath.Join(root, "manifest.json")
	manifestData, _ := json.Marshal(manifest)
	mustWriteAdapter(t, manifestPath, string(manifestData))
	active, err = application.Store.ActiveVectorPlanningSnapshot(ctx)
	if err != nil {
		raw.Close()
		t.Fatal(err)
	}
	covered, err = raw.ExistingKeys(ctx, string(application.Resolved.Profiles.Fingerprints.Source), active.CanonicalInputs)
	if err != nil || len(covered) != len(active.CanonicalInputs) {
		raw.Close()
		t.Fatalf("pre-prepare raw coverage keys=%v covered=%v err=%v", active.CanonicalInputs, covered, err)
	}
	prepared, err := PrepareRetrievalEvaluation(ctx, application, raw, manifestPath, datasetPath, root)
	if err != nil {
		raw.Close()
		t.Fatalf("prepare: %v", err)
	}
	return ctx, prepared, root, raw
}

func adapterConfig(t *testing.T) config.ResolvedConfig {
	value, err := config.Resolve(resolvedRaw(t))
	if err != nil {
		t.Fatal(err)
	}
	return value
}
func resolvedRaw(t *testing.T) config.RawConfig {
	t.Helper()
	dimensions, max, batch, returnK, candidateK, rrf := 256, 4096, 4, 2, 4, 60
	allow := true
	return config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: max, TargetSegmentBytes: max}, Embedding: config.RawEmbedding{ServingDimensions: &dimensions, Request: config.RawRequest{MaxInputs: batch, MaxTotalInputBytes: max, TimeoutSeconds: 1}}, Search: config.RawSearch{AllowPaidQueryEmbedding: &allow, ReturnK: &returnK, CandidateK: &candidateK, RRFK: &rrf}, MCP: config.RawMCP{HardMaxInlineBytes: max}}
}

func adapterManifest(t *testing.T, root string) eval.CorpusManifest {
	t.Helper()
	commit := strings.TrimSpace(runAdapter(t, root, "git", "rev-parse", "HEAD"))
	tree := strings.TrimSpace(runAdapter(t, root, "git", "rev-parse", "HEAD^{tree}"))
	type entry struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	entries := []entry{{Path: "LICENSE", SHA256: adapterSHA(t, filepath.Join(root, "LICENSE"))}, {Path: "a.go", SHA256: adapterSHA(t, filepath.Join(root, "a.go"))}}
	canonical, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	return eval.CorpusManifest{SchemaVersion: 1, CorpusID: "fixture", UpstreamURL: "https://example.invalid/acme/repo.git", PinnedCommit: commit, LicenseSPDX: "MIT", LicenseEvidence: "LICENSE", LanguageSlices: []evalcontract.Language{evalcontract.Go}, Include: []string{"**"}, Exclude: []string{"vendor/**"}, ExpectedTreeHash: tree, ExpectedContentSHA256: hex.EncodeToString(sum[:])}
}
func adapterSHA(t *testing.T, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func adapterSourceVector() []float32 {
	value := make([]float32, 1024)
	value[0], value[1] = 1, .5
	return value
}
func adapterQueryResponse(resolved config.ResolvedConfig) embedclient.EmbeddingResponse {
	return embedclient.EmbeddingResponse{Model: resolved.Embedding.Model.Model, Data: []embedclient.EmbeddingDatum{{Index: 0, IndexPresent: true, Values: adapterSourceVector()}}, TotalTokens: 3}
}
func assertEvaluationRows(t *testing.T, raw *lab.Store, expected int) {
	t.Helper()
	count, err := raw.EvaluationRunCount(context.Background())
	if err != nil || count != expected {
		t.Fatalf("evaluation rows=%d err=%v", count, err)
	}
}
func runAdapter(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = dir
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v: %s", name, args, err, out)
	}
	return string(out)
}
func mustWriteAdapter(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
