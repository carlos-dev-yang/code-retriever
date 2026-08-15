package devlab

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"cidx/internal/app"
	"cidx/internal/embed"
	"cidx/internal/embedclient"
	"cidx/internal/eval"
	"cidx/internal/evalcontract"
	"cidx/internal/lab"
	"cidx/internal/search"
	"cidx/internal/vector"
)

// RetrievalEvaluationPlan is the vector-free result of strict local
// preflight. It intentionally contains no checkout path, source text, raw
// vector, query vector, credential, or provider usage.
type RetrievalEvaluationPlan struct {
	CorpusID                  string `json:"corpus_id"`
	CorpusManifestSHA256      string `json:"corpus_manifest_sha256"`
	DatasetSHA256             string `json:"dataset_sha256"`
	PinnedCommit              string `json:"pinned_commit"`
	ContentSHA256             string `json:"content_sha256"`
	IndexGeneration           int64  `json:"index_generation"`
	IndexManifestSHA256       string `json:"index_manifest_sha256"`
	RawDocumentInputs         int    `json:"raw_document_inputs"`
	QueryCount                int    `json:"query_count"`
	EstimatedQueryTokens      int    `json:"estimated_query_tokens"`
	ServingProfile            string `json:"serving_profile"`
	QueryProviderCallsPlanned int    `json:"query_provider_calls_planned"`
	CostEstimateAvailable     bool   `json:"cost_estimate_available"`
	CostEstimateReason        string `json:"cost_estimate_reason"`
}

type retrievalPrepared struct {
	plan        RetrievalEvaluationPlan
	dataset     eval.EvaluationDataset
	application *app.Application
	raw         *lab.Store
	required    []string
}

type retrievalInputs struct {
	manifest eval.CorpusManifest
	dataset  eval.EvaluationDataset
	verified eval.VerifiedCorpus
}

// preflightRetrievalInputs is intentionally independent of either SQLite
// store. It lets the CLI reject bad corpus provenance before opening a lab DB
// that could otherwise need creation or migration.
func preflightRetrievalInputs(ctx context.Context, repositoryRoot, manifestPath, datasetPath, explicitCorpusPath string) (retrievalInputs, error) {
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return retrievalInputs{}, err
	}
	manifest, err := eval.LoadCorpusManifest(manifestBytes)
	if err != nil {
		return retrievalInputs{}, err
	}
	datasetBytes, err := os.ReadFile(datasetPath)
	if err != nil {
		return retrievalInputs{}, err
	}
	dataset, err := eval.LoadDataset(datasetBytes)
	if err != nil {
		return retrievalInputs{}, err
	}
	if dataset.CorpusID != manifest.CorpusID {
		return retrievalInputs{}, fmt.Errorf("dataset corpus does not match manifest")
	}
	bindings := eval.CorpusBindings{}
	if explicitCorpusPath == "" {
		bindings, err = eval.LoadIgnoredCorpusBindings(ctx, repositoryRoot)
		if err != nil {
			return retrievalInputs{}, err
		}
	}
	checkout, err := eval.ResolveCheckout(manifest, bindings, explicitCorpusPath)
	if err != nil {
		return retrievalInputs{}, err
	}
	if filepath.Clean(checkout) != filepath.Clean(repositoryRoot) {
		return retrievalInputs{}, fmt.Errorf("evaluation checkout must be the configured repository root")
	}
	verified, err := eval.VerifyCheckout(ctx, manifest, checkout)
	if err != nil {
		return retrievalInputs{}, err
	}
	return retrievalInputs{manifest: manifest, dataset: dataset, verified: verified}, nil
}

// PrepareRetrievalEvaluation performs every pre-provider check required by
// the development command. It is read-only and makes zero embedding calls.
func PrepareRetrievalEvaluation(ctx context.Context, application *app.Application, raw *lab.Store, manifestPath, datasetPath, explicitCorpusPath string) (retrievalPrepared, error) {
	if application == nil || application.Store == nil || application.Search == nil || raw == nil {
		return retrievalPrepared{}, fmt.Errorf("production application, search service, and lab store are required")
	}
	inputs, err := preflightRetrievalInputs(ctx, application.Root, manifestPath, datasetPath, explicitCorpusPath)
	if err != nil {
		return retrievalPrepared{}, err
	}
	manifest, dataset, verified := inputs.manifest, inputs.dataset, inputs.verified
	checkout := application.Root
	inventory, err := (eval.ProductionTruthInventory{Store: application.Store}).Snapshot(ctx)
	if err != nil {
		return retrievalPrepared{}, err
	}
	if err := eval.ValidateTruthMapping(dataset, inventory); err != nil {
		return retrievalPrepared{}, err
	}
	active, err := application.Store.ActiveVectorPlanningSnapshot(ctx)
	if err != nil {
		return retrievalPrepared{}, err
	}
	if active.Applied.Fingerprints.Source != application.Resolved.Profiles.Fingerprints.Source || active.Applied.Fingerprints.VectorSpace != application.Resolved.Profiles.Fingerprints.VectorSpace || active.Applied.Fingerprints.VectorStorage != application.Resolved.Profiles.Fingerprints.VectorStorage || active.Applied.ActiveServingProfile != application.Resolved.Profiles.Fingerprints.VectorStorage {
		return retrievalPrepared{}, fmt.Errorf("PROFILE_RECONCILIATION_REQUIRED")
	}
	if active.Applied.ActiveGeneration != inventory.Generation || active.Applied.ManifestSHA256 != inventory.ManifestSHA256 {
		return retrievalPrepared{}, fmt.Errorf("NON_REPRODUCIBLE_RUN")
	}
	indexed, err := application.Store.IndexSnapshot(ctx)
	if err != nil {
		return retrievalPrepared{}, err
	}
	indexedFiles := make(map[string]string, len(indexed.Files))
	for path, file := range indexed.Files {
		indexedFiles[path] = file.SHA256
	}
	if err := eval.VerifyIndexedFiles(ctx, manifest, checkout, indexedFiles); err != nil {
		return retrievalPrepared{}, err
	}
	labRoot, err := raw.CanonicalRoot(ctx)
	if err != nil || labRoot != application.Root {
		return retrievalPrepared{}, fmt.Errorf("lab database belongs to different root")
	}
	required := append([]string(nil), active.CanonicalInputs...)
	hits, err := raw.ExistingKeys(ctx, string(application.Resolved.Profiles.Fingerprints.Source), required)
	if err != nil {
		return retrievalPrepared{}, err
	}
	if len(hits) != len(required) {
		return retrievalPrepared{}, fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
	}
	for _, key := range required {
		if !hits[key] {
			return retrievalPrepared{}, fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
		}
	}
	manifestFingerprint, err := manifest.Fingerprint()
	if err != nil {
		return retrievalPrepared{}, err
	}
	datasetFingerprint, err := dataset.Fingerprint()
	if err != nil {
		return retrievalPrepared{}, err
	}
	estimatedTokens := 0
	for _, item := range dataset.Cases {
		if err := application.Search.ValidateEvaluationQuery(item.Text); err != nil {
			return retrievalPrepared{}, err
		}
		estimate := embed.ConservativeInputTokenUpperBound([]byte(item.Text))
		if estimate > application.Resolved.Embedding.Batch.MaxInputTokens {
			return retrievalPrepared{}, fmt.Errorf("evaluation query exceeds local batch token budget")
		}
		estimatedTokens += estimate
	}
	// This is the exact Phase 11 no-provider snapshot preflight. It rejects a
	// profile mismatch, corrupt row, or incomplete current materialization
	// before --apply can request even the first query vector.
	if _, err := application.Search.StartEvaluationSession(ctx, dataset.Cases[0].Text); err != nil {
		return retrievalPrepared{}, err
	}
	return retrievalPrepared{plan: RetrievalEvaluationPlan{CorpusID: manifest.CorpusID, CorpusManifestSHA256: manifestFingerprint, DatasetSHA256: datasetFingerprint, PinnedCommit: verified.PinnedCommit, ContentSHA256: verified.ContentSHA256, IndexGeneration: inventory.Generation, IndexManifestSHA256: inventory.ManifestSHA256, RawDocumentInputs: len(required), QueryCount: len(dataset.Cases), EstimatedQueryTokens: estimatedTokens, ServingProfile: string(application.Resolved.Profiles.Fingerprints.VectorStorage), QueryProviderCallsPlanned: len(dataset.Cases), CostEstimateAvailable: false, CostEstimateReason: "no dated provider price has been frozen for this run"}, dataset: dataset, application: application, raw: raw, required: required}, nil
}

func (prepared retrievalPrepared) Plan() RetrievalEvaluationPlan { return prepared.plan }

// RetrievalEvaluationApplied is returned only after the complete in-memory
// run has been published atomically and its vector-free reference recorded in
// the lab database.
type RetrievalEvaluationApplied struct {
	Run      eval.RetrievalEvaluationRun `json:"run"`
	Artifact RetrievalArtifactReference  `json:"artifact"`
}

// Apply performs the separately authorized query embedding operation. It
// keeps every raw/query f32 only in run memory and delegates all arm metrics
// and diagnostics to eval.RunRetrievalEvaluation.
func (prepared retrievalPrepared) Apply(ctx context.Context, client embedclient.EmbeddingClient) (RetrievalEvaluationApplied, error) {
	if client == nil {
		return RetrievalEvaluationApplied{}, fmt.Errorf("query embedding client is required")
	}
	documents, err := prepared.targetDocuments(ctx)
	if err != nil {
		return RetrievalEvaluationApplied{}, err
	}
	usageClient := &recordingEmbeddingClient{delegate: client}
	executor := &retrievalExecutor{application: prepared.application, client: usageClient, documents: documents, sessions: map[string]search.EvaluationSession{}, arms: map[string]search.EvaluationVectorArms{}, queries: map[string]queryVector{}, failures: map[string]error{}}
	run, err := eval.RunRetrievalEvaluation(ctx, prepared.dataset, eval.DefaultRetrievalPlan([]int{prepared.application.Resolved.Search.ReturnK}), executor)
	if err != nil {
		return RetrievalEvaluationApplied{}, err
	}
	usage := usageClient.Usage()
	artifact, err := publishRetrievalArtifact(ctx, prepared, run, usage)
	if err != nil {
		return RetrievalEvaluationApplied{}, err
	}
	if _, err := prepared.raw.RecordEvaluationRun(ctx, lab.EvaluationRunRecord{RunID: artifact.RunID, RepositoryIdentity: prepared.plan.ContentSHA256, CorpusID: prepared.plan.CorpusID, CorpusManifestSHA256: prepared.plan.CorpusManifestSHA256, PinnedCommit: prepared.plan.PinnedCommit, ContentSHA256: prepared.plan.ContentSHA256, Generation: prepared.plan.IndexGeneration, IndexManifestSHA256: prepared.plan.IndexManifestSHA256, QueryManifestSHA256: prepared.plan.DatasetSHA256, QueryCount: prepared.plan.QueryCount, CandidateProfile: prepared.plan.ServingProfile, SourceProfile: string(prepared.application.Resolved.Profiles.Fingerprints.Source), VectorSpaceProfile: string(prepared.application.Resolved.Profiles.Fingerprints.VectorSpace), RawDocumentInputs: prepared.plan.RawDocumentInputs, QueryProviderCalls: usage.QueryProviderCalls, QueryTokens: usage.QueryTokens, ArtifactReference: artifact.Reference, ArtifactChecksum: artifact.Checksum}); err != nil {
		_ = removeRetrievalArtifact(context.Background(), prepared, artifact)
		return RetrievalEvaluationApplied{}, err
	}
	return RetrievalEvaluationApplied{Run: run, Artifact: artifact}, nil
}

type recordingEmbeddingClient struct {
	delegate embedclient.EmbeddingClient
	mu       sync.Mutex
	usage    retrievalProviderUsage
}

func (client *recordingEmbeddingClient) Embed(ctx context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	response, err := client.delegate.Embed(ctx, request)
	client.mu.Lock()
	defer client.mu.Unlock()
	client.usage.QueryProviderCalls++
	if err != nil {
		client.usage.FailedCalls++
		return embedclient.EmbeddingResponse{}, err
	}
	client.usage.SuccessfulCalls++
	client.usage.QueryTokens += response.TotalTokens
	return response, nil
}

func (client *recordingEmbeddingClient) Usage() retrievalProviderUsage {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.usage
}

func (prepared retrievalPrepared) targetDocuments(ctx context.Context) (map[string][]float32, error) {
	raws, err := prepared.raw.RawDocuments(ctx, string(prepared.application.Resolved.Profiles.Fingerprints.Source), prepared.required)
	if err != nil {
		return nil, err
	}
	if len(raws) != len(prepared.required) {
		return nil, fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
	}
	transformer := vector.Transformer{Spec: prepared.application.Resolved.Embedding.TransformSpec()}
	result := make(map[string][]float32, len(raws))
	for _, key := range prepared.required {
		raw, ok := raws[key]
		if !ok || raw.Dimensions != prepared.application.Resolved.Embedding.Model.SourceDimensions {
			return nil, fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
		}
		decoded, err := lab.DecodeF32(raw.VectorF32LE, raw.Dimensions, raw.Checksum)
		if err != nil {
			return nil, err
		}
		space, err := transformer.Transform(decoded.Values)
		if err != nil {
			return nil, err
		}
		result[key] = space
	}
	return result, nil
}

type queryVector struct {
	values []float32
	sha256 string
}

type retrievalExecutor struct {
	application *app.Application
	client      embedclient.EmbeddingClient
	documents   map[string][]float32
	sessions    map[string]search.EvaluationSession
	arms        map[string]search.EvaluationVectorArms
	queries     map[string]queryVector
	failures    map[string]error
}

func (executor *retrievalExecutor) EvaluateArm(ctx context.Context, item evalcontract.EvaluationCase, variant eval.RetrievalVariant) (eval.RetrievalArmResult, error) {
	session, err := executor.session(ctx, item)
	if err != nil {
		return eval.RetrievalArmResult{}, err
	}
	if variant == eval.VariantFTS {
		hits, err := session.FTS(executor.application.Resolved.Search.ReturnK)
		if err != nil {
			return eval.RetrievalArmResult{}, err
		}
		return eval.RetrievalArmResult{Ranking: ranking(item.ID, variant, "", hits)}, nil
	}
	if variant == eval.VariantHybridWithoutDense {
		hits, err := session.FTS(executor.application.Resolved.Search.ReturnK)
		if err != nil {
			return eval.RetrievalArmResult{}, err
		}
		delete(executor.sessions, item.ID)
		delete(executor.arms, item.ID)
		delete(executor.queries, item.ID)
		delete(executor.failures, item.ID)
		return eval.RetrievalArmResult{Ranking: ranking(item.ID, variant, "", hits)}, nil
	}
	armSet, query, err := executor.vectorArms(ctx, item, session)
	if err != nil {
		var providerFailure search.QueryEmbeddingProviderError
		if errors.As(err, &providerFailure) {
			return eval.RetrievalArmResult{}, eval.RetrievalArmFailure{Stage: evalcontract.FailureStage(evalcontract.StageOperational)}
		}
		return eval.RetrievalArmResult{}, err
	}
	var hits []search.EvaluationRankedHit
	result := eval.RetrievalArmResult{}
	switch variant {
	case eval.VariantTargetF32:
		hits = armSet.TargetF32
		result.Segments = armSet.TargetF32Segments
	case eval.VariantServingActiveCodec:
		hits = armSet.ServingActiveCodec
		result.Segments = armSet.ServingActiveSegments
	case eval.VariantProviderUnion:
		hits = armSet.ProviderUnion
	case eval.VariantHybridFTSTargetF32:
		hits = armSet.HybridFTSTargetF32
	case eval.VariantHybridFTSActiveCodec:
		hits = armSet.HybridFTSActiveCodec
		result.Packaged = packaged(armSet.ActiveCodecBodies)
	case eval.VariantHybridWithoutFTS:
		hits = armSet.HybridWithoutFTS
	default:
		return eval.RetrievalArmResult{}, fmt.Errorf("unknown retrieval variant")
	}
	result.Ranking = ranking(item.ID, variant, query.sha256, hits)
	return result, nil
}

func (executor *retrievalExecutor) session(ctx context.Context, item evalcontract.EvaluationCase) (search.EvaluationSession, error) {
	if value, ok := executor.sessions[item.ID]; ok {
		return value, nil
	}
	value, err := executor.application.Search.StartEvaluationSession(ctx, item.Text)
	if err != nil {
		return search.EvaluationSession{}, err
	}
	executor.sessions[item.ID] = value
	return value, nil
}

func (executor *retrievalExecutor) vectorArms(ctx context.Context, item evalcontract.EvaluationCase, session search.EvaluationSession) (search.EvaluationVectorArms, queryVector, error) {
	if value, ok := executor.arms[item.ID]; ok {
		return value, executor.queries[item.ID], nil
	}
	if err, ok := executor.failures[item.ID]; ok {
		return search.EvaluationVectorArms{}, queryVector{}, err
	}
	query, err := search.EmbedEvaluationQuery(ctx, executor.client, executor.application.Resolved, item.Text)
	if err != nil {
		executor.failures[item.ID] = err
		return search.EvaluationVectorArms{}, queryVector{}, err
	}
	vector := queryVector{values: query, sha256: querySHA256(query)}
	arms, err := session.EvaluateVectorArms(ctx, query, executor.documents, executor.application.Resolved.Search.ReturnK, executor.application.Resolved.MCP.HardMaxInlineBytes)
	if err != nil {
		executor.failures[item.ID] = err
		return search.EvaluationVectorArms{}, queryVector{}, err
	}
	executor.queries[item.ID], executor.arms[item.ID] = vector, arms
	return arms, vector, nil
}

func ranking(queryID string, variant eval.RetrievalVariant, digest string, hits []search.EvaluationRankedHit) eval.CaseRanking {
	result := eval.CaseRanking{QueryID: queryID, Variant: variant, QueryVectorSHA256: digest, Hits: make([]eval.RetrievalHit, 0, len(hits))}
	for _, hit := range hits {
		result.Hits = append(result.Hits, eval.RetrievalHit{Path: hit.Path, IndexedSHA256: hit.IndexedSHA256, QualifiedSymbol: hit.QualifiedSymbol, StartByte: hit.StartByte, EndByte: hit.EndByte, Rank: hit.Rank, Score: hit.Score})
	}
	return result
}

func packaged(hits []search.Hit) []eval.BodyPackageHit {
	result := make([]eval.BodyPackageHit, 0, len(hits))
	for _, hit := range hits {
		value := eval.BodyPackageHit{Hit: eval.RetrievalHit{Path: hit.Path, IndexedSHA256: hit.IndexedSHA256, QualifiedSymbol: hit.QualifiedSymbol, StartByte: hit.ParentRange.StartByte, EndByte: hit.ParentRange.EndByte, Rank: len(result) + 1}, BodyComplete: hit.BodyComplete, BodyBytes: hit.BodyBytes, OmissionReason: hit.BodyOmissionReason}
		if hit.BodyRange != nil {
			span := evalcontract.SourceSpan{Path: hit.Path, ContentSHA256: hit.IndexedSHA256, QualifiedSymbol: hit.QualifiedSymbol, StartByte: hit.BodyRange.StartByte, EndByte: hit.BodyRange.EndByte}
			value.BodyRange = &span
			sum := sha256.Sum256(hit.Body)
			value.BodySHA256 = hex.EncodeToString(sum[:])
		}
		result = append(result, value)
	}
	return result
}

func querySHA256(values []float32) string {
	bytes := make([]byte, len(values)*4)
	for index, value := range values {
		binary.LittleEndian.PutUint32(bytes[index*4:], math.Float32bits(value))
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}
