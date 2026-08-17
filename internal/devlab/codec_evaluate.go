package devlab

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	"cidx/internal/app"
	"cidx/internal/buildinfo"
	"cidx/internal/config"
	"cidx/internal/embedclient"
	"cidx/internal/eval"
	"cidx/internal/evalcontract"
	"cidx/internal/lab"
	"cidx/internal/search"
	"cidx/internal/vector"
)

var codecComparisonKs = []int{1, 5, 10, 20}

type CodecComparisonOptions struct {
	ExperimentSeriesID           string
	SeriesQueryOperationsPlanned int
	AuthorizationReference       string
	USDCap                       float64
	PricingTableIdentity         string
	USDPerMillionTokens          float64
}

// CodecComparisonPlan is the source/vector-free pre-provider contract for an
// isolated f32/binary/int8 comparison.
type CodecComparisonPlan struct {
	CorpusID                      string   `json:"corpus_id"`
	CorpusManifestSHA256          string   `json:"corpus_manifest_sha256"`
	DatasetSHA256                 string   `json:"dataset_sha256"`
	PinnedCommit                  string   `json:"pinned_commit"`
	ContentSHA256                 string   `json:"content_sha256"`
	IndexGeneration               int64    `json:"index_generation"`
	IndexManifestSHA256           string   `json:"index_manifest_sha256"`
	RawDocumentInputs             int      `json:"raw_document_inputs"`
	QueryCount                    int      `json:"query_count"`
	LogicalQueryOperationsPlanned int      `json:"logical_query_operations_planned"`
	DocumentProviderOperations    int      `json:"document_provider_operations_planned"`
	QueryVectorsPersisted         bool     `json:"query_vectors_persisted"`
	QueryVectorReuseArms          int      `json:"query_vector_reuse_arms"`
	SourceDimensions              int      `json:"source_dimensions"`
	ServingDimensions             int      `json:"serving_dimensions"`
	ProductionStorageCodec        string   `json:"production_storage_codec"`
	ComparedRepresentations       []string `json:"compared_representations"`
	UsesFTS                       bool     `json:"uses_fts"`
	UsesRRF                       bool     `json:"uses_rrf"`
	Depth                         int      `json:"depth"`
	Ks                            []int    `json:"ks"`
	EstimatedQueryTokens          int      `json:"estimated_query_tokens"`
	ExperimentSeriesID            string   `json:"experiment_series_id"`
	SeriesQueryOperationsPlanned  int      `json:"series_query_operations_planned"`
	AuthorizationReference        string   `json:"authorization_reference"`
	USDCap                        float64  `json:"usd_cap"`
	PricingTableIdentity          string   `json:"pricing_table_identity"`
	USDPerMillionTokens           float64  `json:"usd_per_million_tokens"`
	PlannedMaximumCostUSD         float64  `json:"planned_maximum_cost_usd"`
	CodeCommit                    string   `json:"code_commit"`
	SourceModified                string   `json:"source_modified"`
	EvaluationExecutableSHA256    string   `json:"evaluation_executable_sha256"`
}

type codecComparisonPrepared struct {
	base    retrievalPrepared
	plan    CodecComparisonPlan
	options CodecComparisonOptions
}

func PrepareCodecComparisonExperimentAt(ctx context.Context, application *app.Application, raw *lab.Store, bindingRoot, manifestPath, datasetPath, explicitCorpusPath string, options CodecComparisonOptions) (codecComparisonPrepared, error) {
	base, err := PrepareRetrievalEvaluationExperimentAt(ctx, application, raw, bindingRoot, manifestPath, datasetPath, explicitCorpusPath, RetrievalExperimentOptions{VectorOnly: true})
	if err != nil {
		return codecComparisonPrepared{}, err
	}
	resolved := application.Resolved
	if resolved.Embedding.Model.SourceDimensions != 1024 || resolved.Embedding.ServingDimensions != 1024 || resolved.Embedding.StorageCodec != config.StorageCodecBinary || resolved.Search.CandidateK < 20 {
		return codecComparisonPrepared{}, fmt.Errorf("codec comparison requires 1024-dimensional active binary with candidate_k at least 20")
	}
	if options.ExperimentSeriesID == "" || options.SeriesQueryOperationsPlanned != 32 || options.AuthorizationReference == "" || !isFinitePositive(options.USDCap) || options.PricingTableIdentity == "" || !isFinitePositive(options.USDPerMillionTokens) {
		return codecComparisonPrepared{}, fmt.Errorf("incomplete codec comparison authority or pricing controls")
	}
	attempts := resolved.Embedding.Retry.MaxRetries + 1
	plannedMaximumCost := float64(base.plan.EstimatedQueryTokens*attempts) * options.USDPerMillionTokens / 1_000_000
	if plannedMaximumCost > options.USDCap {
		return codecComparisonPrepared{}, fmt.Errorf("codec comparison exceeds USD cap")
	}
	info := buildinfo.Current()
	if err := validateLexicalCodeProvenance(info); err != nil {
		return codecComparisonPrepared{}, err
	}
	executableSHA256, err := currentExecutableSHA256()
	if err != nil {
		return codecComparisonPrepared{}, err
	}
	plan := CodecComparisonPlan{
		CorpusID: base.plan.CorpusID, CorpusManifestSHA256: base.plan.CorpusManifestSHA256,
		DatasetSHA256: base.plan.DatasetSHA256, PinnedCommit: base.plan.PinnedCommit,
		ContentSHA256: base.plan.ContentSHA256, IndexGeneration: base.plan.IndexGeneration,
		IndexManifestSHA256: base.plan.IndexManifestSHA256, RawDocumentInputs: base.plan.RawDocumentInputs,
		QueryCount: len(base.dataset.Cases), LogicalQueryOperationsPlanned: len(base.dataset.Cases),
		DocumentProviderOperations: 0, QueryVectorsPersisted: false, QueryVectorReuseArms: 3,
		SourceDimensions: resolved.Embedding.Model.SourceDimensions, ServingDimensions: resolved.Embedding.ServingDimensions,
		ProductionStorageCodec:  resolved.Embedding.StorageCodec,
		ComparedRepresentations: []string{"target_f32", "active_binary", "candidate_int8"},
		UsesFTS:                 false, UsesRRF: false, Depth: 20, Ks: append([]int(nil), codecComparisonKs...),
		EstimatedQueryTokens: base.plan.EstimatedQueryTokens,
		ExperimentSeriesID:   options.ExperimentSeriesID, SeriesQueryOperationsPlanned: options.SeriesQueryOperationsPlanned,
		AuthorizationReference: options.AuthorizationReference, USDCap: options.USDCap,
		PricingTableIdentity: options.PricingTableIdentity, USDPerMillionTokens: options.USDPerMillionTokens,
		PlannedMaximumCostUSD: plannedMaximumCost, CodeCommit: info.Commit,
		SourceModified: info.SourceModified, EvaluationExecutableSHA256: executableSHA256,
	}
	return codecComparisonPrepared{base: base, plan: plan, options: options}, nil
}

func (prepared codecComparisonPrepared) Plan() CodecComparisonPlan { return prepared.plan }

type CodecComparisonMetric struct {
	Representation string          `json:"representation"`
	Metrics        eval.CaseResult `json:"metrics"`
}

type CodecSegmentRanking struct {
	Representation string                        `json:"representation"`
	Segments       []search.EvaluationSegmentHit `json:"segments"`
}

type CodecComparisonCase struct {
	QueryID           string                  `json:"query_id"`
	QueryVectorSHA256 string                  `json:"query_vector_sha256"`
	Rankings          []eval.CaseRanking      `json:"rankings"`
	Segments          []CodecSegmentRanking   `json:"segments"`
	Metrics           []CodecComparisonMetric `json:"metrics"`
	BinaryFidelity    eval.CodecFidelity      `json:"binary_fidelity"`
	Int8Fidelity      eval.CodecFidelity      `json:"int8_fidelity"`
}

type CodecFidelitySummary struct {
	Cases                     int     `json:"cases"`
	TopKRetentionMean         float64 `json:"top_k_retention_mean"`
	Top1MismatchRate          float64 `json:"top_1_mismatch_rate"`
	MeanRankDisplacementMean  float64 `json:"mean_rank_displacement_mean"`
	F32CandidatesMissingTotal int     `json:"f32_candidates_missing_total"`
	GoldF32RetentionObserved  int     `json:"gold_f32_retention_observed"`
	GoldF32RetentionMean      float64 `json:"gold_f32_retention_mean"`
	BoundaryTieRate           float64 `json:"boundary_tie_rate"`
}

type CodecComparisonRun struct {
	Depth             int                             `json:"depth"`
	Ks                []int                           `json:"ks"`
	Cases             []CodecComparisonCase           `json:"cases"`
	AggregateMetrics  map[string]eval.Summary         `json:"aggregate_metrics"`
	FidelitySummaries map[string]CodecFidelitySummary `json:"fidelity_summaries"`
}

type CodecComparisonApplied struct {
	Run      CodecComparisonRun         `json:"run"`
	Artifact RetrievalArtifactReference `json:"artifact"`
}

func (prepared codecComparisonPrepared) Apply(ctx context.Context, client embedclient.EmbeddingClient) (CodecComparisonApplied, error) {
	if client == nil {
		return CodecComparisonApplied{}, fmt.Errorf("query embedding client is required")
	}
	documents, documentBankSHA256, err := prepared.base.targetDocuments(ctx)
	if err != nil {
		return CodecComparisonApplied{}, err
	}
	int8Documents := make(map[string]vector.StoredVector, len(documents))
	keys := make([]string, 0, len(documents))
	for key := range documents {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		encoded, err := vector.EncodeInt8(documents[key])
		if err != nil {
			return CodecComparisonApplied{}, err
		}
		int8Documents[key] = encoded
	}
	prepared.base.documentBankFingerprint = documentBankSHA256
	if err := prepared.base.validateCurrentRetrievalState(ctx); err != nil {
		return CodecComparisonApplied{}, err
	}
	usageRecorder := newRetrievalProviderUsage(prepared.base, client)
	run := CodecComparisonRun{Depth: prepared.plan.Depth, Ks: append([]int(nil), prepared.plan.Ks...), Cases: make([]CodecComparisonCase, 0, len(prepared.base.dataset.Cases))}
	metricSets := map[string][]eval.CaseResult{"target_f32": {}, "active_binary": {}, "candidate_int8": {}}
	session, err := prepared.base.application.Search.StartCodecEvaluationSession(ctx)
	if err != nil {
		return CodecComparisonApplied{}, err
	}
	if err := session.ValidateSnapshot(prepared.plan.IndexGeneration, prepared.plan.IndexManifestSHA256); err != nil {
		return CodecComparisonApplied{}, err
	}
	for _, item := range prepared.base.dataset.Cases {
		if err := ctx.Err(); err != nil {
			return CodecComparisonApplied{}, err
		}
		recorder, err := usageRecorder.Start(item.ID)
		if err != nil {
			return CodecComparisonApplied{}, err
		}
		query, embedErr := search.EmbedEvaluationQuery(ctx, recorder, prepared.base.application.Resolved, item.Text)
		if err := usageRecorder.Finish(ctx, recorder, embedErr); err != nil {
			return CodecComparisonApplied{}, err
		}
		if embedErr != nil {
			var providerFailure search.QueryEmbeddingProviderError
			if !errors.As(embedErr, &providerFailure) {
				return CodecComparisonApplied{}, embedErr
			}
			rankings := []eval.CaseRanking{
				{QueryID: item.ID, Variant: eval.VariantTargetF32},
				{QueryID: item.ID, Variant: eval.VariantServingActiveCodec},
				{QueryID: item.ID, Variant: eval.VariantCandidateInt8},
			}
			metrics := make([]CodecComparisonMetric, 0, len(rankings))
			for index, value := range rankings {
				representation := []string{"target_f32", "active_binary", "candidate_int8"}[index]
				measured, err := eval.EvaluateRetrievalRanking(item, value, prepared.plan.Ks, eval.RetrievalArmFailure{Stage: evalcontract.FailureStage(evalcontract.StageOperational)})
				if err != nil {
					return CodecComparisonApplied{}, err
				}
				metrics = append(metrics, CodecComparisonMetric{Representation: representation, Metrics: measured})
				metricSets[representation] = append(metricSets[representation], measured)
			}
			run.Cases = append(run.Cases, CodecComparisonCase{
				QueryID: item.ID, Rankings: rankings,
				Segments:       []CodecSegmentRanking{{Representation: "target_f32"}, {Representation: "active_binary"}, {Representation: "candidate_int8"}},
				Metrics:        metrics,
				BinaryFidelity: eval.CodecFidelity{FailureStage: evalcontract.FailureStage(evalcontract.StageOperational)},
				Int8Fidelity:   eval.CodecFidelity{FailureStage: evalcontract.FailureStage(evalcontract.StageOperational)},
			})
			continue
		}
		queryDigest := querySHA256(query)
		arms, err := session.EvaluateCodecArms(ctx, query, documents, int8Documents, prepared.plan.Depth)
		if err != nil {
			return CodecComparisonApplied{}, err
		}
		rankings := []eval.CaseRanking{
			ranking(item.ID, eval.VariantTargetF32, queryDigest, arms.TargetF32),
			ranking(item.ID, eval.VariantServingActiveCodec, queryDigest, arms.ServingActiveCodec),
			ranking(item.ID, eval.VariantCandidateInt8, queryDigest, arms.CandidateInt8),
		}
		representations := []string{"target_f32", "active_binary", "candidate_int8"}
		metrics := make([]CodecComparisonMetric, 0, len(rankings))
		for index, value := range rankings {
			measured, err := eval.EvaluateRetrievalRanking(item, value, prepared.plan.Ks, nil)
			if err != nil {
				return CodecComparisonApplied{}, err
			}
			metrics = append(metrics, CodecComparisonMetric{Representation: representations[index], Metrics: measured})
			metricSets[representations[index]] = append(metricSets[representations[index]], measured)
		}
		binaryFidelity, err := eval.CompareCodecFidelity(item, rankings[0], rankings[1], prepared.plan.Depth)
		if err != nil {
			return CodecComparisonApplied{}, err
		}
		int8Fidelity, err := eval.CompareCodecFidelity(item, rankings[0], rankings[2], prepared.plan.Depth)
		if err != nil {
			return CodecComparisonApplied{}, err
		}
		run.Cases = append(run.Cases, CodecComparisonCase{
			QueryID: item.ID, QueryVectorSHA256: queryDigest, Rankings: rankings,
			Segments: []CodecSegmentRanking{
				{Representation: "target_f32", Segments: limitedSegments(arms.TargetF32Segments, prepared.plan.Depth)},
				{Representation: "active_binary", Segments: limitedSegments(arms.ServingActiveSegments, prepared.plan.Depth)},
				{Representation: "candidate_int8", Segments: limitedSegments(arms.CandidateInt8Segments, prepared.plan.Depth)},
			},
			Metrics: metrics, BinaryFidelity: binaryFidelity, Int8Fidelity: int8Fidelity,
		})
	}
	run.AggregateMetrics = map[string]eval.Summary{}
	for _, representation := range []string{"target_f32", "active_binary", "candidate_int8"} {
		run.AggregateMetrics[representation] = eval.Summarize(metricSets[representation], prepared.plan.Ks)
	}
	run.FidelitySummaries = map[string]CodecFidelitySummary{
		"active_binary":  summarizeCodecFidelity(run.Cases, false),
		"candidate_int8": summarizeCodecFidelity(run.Cases, true),
	}
	if err := validateCodecComparisonRun(prepared, run); err != nil {
		return CodecComparisonApplied{}, err
	}
	usage, err := usageRecorder.FinalizeOperations(prepared.base, prepared.base.dataset)
	if err != nil {
		return CodecComparisonApplied{}, err
	}
	if usage.Aggregate.LogicalQueryOperations != prepared.plan.QueryCount {
		return CodecComparisonApplied{}, fmt.Errorf("codec comparison did not use exactly one logical query operation per case")
	}
	if err := prepared.base.validateCurrentRetrievalState(ctx); err != nil {
		return CodecComparisonApplied{}, err
	}
	artifact, err := publishCodecComparisonArtifact(ctx, prepared, run, usage)
	if err != nil {
		return CodecComparisonApplied{}, err
	}
	if _, err := prepared.base.raw.RecordEvaluationRun(ctx, lab.EvaluationRunRecord{
		RunID: artifact.RunID, RepositoryIdentity: prepared.plan.ContentSHA256, CorpusID: prepared.plan.CorpusID,
		CorpusManifestSHA256: prepared.plan.CorpusManifestSHA256, PinnedCommit: prepared.plan.PinnedCommit,
		ContentSHA256: prepared.plan.ContentSHA256, Generation: prepared.plan.IndexGeneration,
		IndexManifestSHA256: prepared.plan.IndexManifestSHA256, QueryManifestSHA256: prepared.plan.DatasetSHA256,
		QueryCount: prepared.plan.QueryCount, CandidateProfile: prepared.plan.ProductionStorageCodec + "+candidate-int8",
		SourceProfile:      string(prepared.base.application.Resolved.Profiles.Fingerprints.Source),
		VectorSpaceProfile: string(prepared.base.application.Resolved.Profiles.Fingerprints.VectorSpace),
		RawDocumentInputs:  prepared.plan.RawDocumentInputs, LogicalQueryOperations: usage.Aggregate.LogicalQueryOperations,
		ProviderAttempts: usage.Aggregate.ProviderAttempts, ValidatedResponses: usage.Aggregate.ValidatedResponses,
		FailedAttempts: usage.Aggregate.FailedAttempts, Retries: usage.Aggregate.Retries,
		ObservedTotalTokens: usage.Aggregate.ObservedTotalTokens, TokenObservedAttempts: usage.Aggregate.TokenObservedAttempts,
		TokenAccountingComplete: usage.Aggregate.TokenAccountingComplete,
		ArtifactReference:       artifact.Reference, ArtifactChecksum: artifact.Checksum,
	}); err != nil {
		_ = removeCodecComparisonArtifact(context.Background(), prepared, artifact)
		return CodecComparisonApplied{}, err
	}
	return CodecComparisonApplied{Run: run, Artifact: artifact}, nil
}

func limitedSegments(values []search.EvaluationSegmentHit, depth int) []search.EvaluationSegmentHit {
	if len(values) > depth {
		values = values[:depth]
	}
	return append([]search.EvaluationSegmentHit(nil), values...)
}

func summarizeCodecFidelity(cases []CodecComparisonCase, int8Candidate bool) CodecFidelitySummary {
	result := CodecFidelitySummary{}
	for _, item := range cases {
		value := item.BinaryFidelity
		if int8Candidate {
			value = item.Int8Fidelity
		}
		if !value.Observed {
			continue
		}
		result.Cases++
		result.TopKRetentionMean += value.TopKRetention
		result.MeanRankDisplacementMean += value.MeanRankDisplacement
		result.F32CandidatesMissingTotal += value.F32CandidatesMissing
		if value.Top1Mismatch {
			result.Top1MismatchRate++
		}
		if value.CodecBoundaryTie {
			result.BoundaryTieRate++
		}
		if value.GoldF32Retention != nil {
			result.GoldF32RetentionObserved++
			result.GoldF32RetentionMean += *value.GoldF32Retention
		}
	}
	if result.Cases == 0 {
		return result
	}
	denominator := float64(result.Cases)
	result.TopKRetentionMean /= denominator
	result.Top1MismatchRate /= denominator
	result.MeanRankDisplacementMean /= denominator
	result.BoundaryTieRate /= denominator
	if result.GoldF32RetentionObserved > 0 {
		result.GoldF32RetentionMean /= float64(result.GoldF32RetentionObserved)
	}
	return result
}

func validateCodecComparisonRun(prepared codecComparisonPrepared, run CodecComparisonRun) error {
	if run.Depth != 20 || len(run.Ks) != len(codecComparisonKs) || len(run.Cases) != len(prepared.base.dataset.Cases) {
		return fmt.Errorf("invalid codec comparison run")
	}
	for index, expected := range codecComparisonKs {
		if run.Ks[index] != expected {
			return fmt.Errorf("invalid codec comparison cutoffs")
		}
	}
	for index, item := range run.Cases {
		if item.QueryID != prepared.base.dataset.Cases[index].ID || len(item.Rankings) != 3 || len(item.Segments) != 3 || len(item.Metrics) != 3 {
			return fmt.Errorf("invalid codec comparison case")
		}
		failed := item.BinaryFidelity.FailureStage != "" || item.Int8Fidelity.FailureStage != ""
		if failed != (item.QueryVectorSHA256 == "") {
			return fmt.Errorf("invalid codec comparison query outcome")
		}
		variants := []eval.RetrievalVariant{eval.VariantTargetF32, eval.VariantServingActiveCodec, eval.VariantCandidateInt8}
		for rankingIndex, value := range item.Rankings {
			if value.QueryID != item.QueryID || value.QueryVectorSHA256 != item.QueryVectorSHA256 || value.Variant != variants[rankingIndex] || len(value.Hits) > run.Depth || value.Validate() != nil {
				return fmt.Errorf("invalid codec comparison ranking")
			}
			if failed && len(value.Hits) != 0 {
				return fmt.Errorf("failed codec comparison returned hits")
			}
		}
		if failed {
			if item.BinaryFidelity.Observed || item.Int8Fidelity.Observed || item.BinaryFidelity.FailureStage != evalcontract.FailureStage(evalcontract.StageOperational) || item.Int8Fidelity.FailureStage != evalcontract.FailureStage(evalcontract.StageOperational) {
				return fmt.Errorf("invalid failed codec fidelity evidence")
			}
		} else if !item.BinaryFidelity.Observed || !item.Int8Fidelity.Observed || !finiteCodecFidelity(item.BinaryFidelity) || !finiteCodecFidelity(item.Int8Fidelity) {
			return fmt.Errorf("invalid codec fidelity evidence")
		}
	}
	if len(run.AggregateMetrics) != 3 || len(run.FidelitySummaries) != 2 {
		return fmt.Errorf("incomplete codec comparison aggregate")
	}
	return nil
}

func finiteCodecFidelity(value eval.CodecFidelity) bool {
	values := []float64{value.TopKRetention, value.MeanRankDisplacement, value.MedianDisplacement, value.P95Displacement}
	for _, number := range values {
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return false
		}
	}
	return true
}
