package eval

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"

	"cidx/internal/evalcontract"
	"cidx/internal/search"
	"cidx/internal/search/lexical"
)

// RetrievalVariant names an independently observable retrieval arm. It is a
// development-evaluation contract, not a production search mode.
type RetrievalVariant string

const (
	VariantFTS                  RetrievalVariant = "fts"
	VariantTargetF32            RetrievalVariant = "target_f32"
	VariantServingActiveCodec   RetrievalVariant = "serving_active_codec"
	VariantProviderUnion        RetrievalVariant = "provider_union"
	VariantHybridFTSTargetF32   RetrievalVariant = "hybrid_fts_target_f32"
	VariantHybridFTSActiveCodec RetrievalVariant = "hybrid_fts_active_codec"
	VariantHybridWithoutFTS     RetrievalVariant = "hybrid_without_fts"
	VariantHybridWithoutDense   RetrievalVariant = "hybrid_without_dense"
)

var requiredRetrievalVariants = []RetrievalVariant{
	VariantFTS,
	VariantTargetF32,
	VariantServingActiveCodec,
	VariantProviderUnion,
	VariantHybridFTSTargetF32,
	VariantHybridFTSActiveCodec,
	VariantHybridWithoutFTS,
	VariantHybridWithoutDense,
}

// RetrievalPlan freezes the arms and cutoffs for one run. Callers inject the
// Phase 11 production search implementations; this package supplies no
// alternate FTS, vector scan, collapse, or RRF implementation.
type RetrievalPlan struct {
	Variants []RetrievalVariant `json:"variants"`
	Ks       []int              `json:"ks"`
}

func DefaultRetrievalPlan(ks []int) RetrievalPlan {
	return RetrievalPlan{Variants: append([]RetrievalVariant(nil), requiredRetrievalVariants...), Ks: norm(ks)}
}

func (value RetrievalPlan) Validate() error {
	if len(norm(value.Ks)) == 0 || len(value.Ks) != len(norm(value.Ks)) {
		return fmt.Errorf("invalid retrieval cutoffs")
	}
	if len(value.Variants) != len(requiredRetrievalVariants) {
		return fmt.Errorf("incomplete retrieval variants")
	}
	for index, variant := range requiredRetrievalVariants {
		if value.Variants[index] != variant {
			return fmt.Errorf("invalid retrieval variant order")
		}
	}
	return nil
}

// RetrievalHit is portable parent identity and native per-arm score only. It
// intentionally contains neither source body bytes nor query-vector bytes.
type RetrievalHit struct {
	Path            string   `json:"path"`
	IndexedSHA256   string   `json:"indexed_sha256"`
	QualifiedSymbol string   `json:"qualified_symbol"`
	StartByte       int      `json:"start_byte"`
	EndByte         int      `json:"end_byte"`
	Rank            int      `json:"rank"`
	Score           *float64 `json:"score,omitempty"`
}

func (value RetrievalHit) Validate() error {
	if !validRelative(value.Path, false) || !validSHA256(value.IndexedSHA256) || value.QualifiedSymbol == "" || value.StartByte < 0 || value.EndByte <= value.StartByte || value.Rank <= 0 {
		return fmt.Errorf("invalid retrieval hit")
	}
	if value.Score != nil && (math.IsNaN(*value.Score) || math.IsInf(*value.Score, 0)) {
		return fmt.Errorf("invalid retrieval score")
	}
	return nil
}

// CaseRanking is one ranked arm for one frozen query. QueryVectorSHA256 is
// optional for FTS/failed arms and, when present, is only a digest.
type CaseRanking struct {
	QueryID           string           `json:"query_id"`
	Variant           RetrievalVariant `json:"variant"`
	QueryVectorSHA256 string           `json:"query_vector_sha256,omitempty"`
	Hits              []RetrievalHit   `json:"hits"`
}

func (value CaseRanking) Validate() error {
	if !validID(value.QueryID) || !knownVariant(value.Variant) || (value.QueryVectorSHA256 != "" && !validSHA256(value.QueryVectorSHA256)) {
		return fmt.Errorf("invalid case ranking")
	}
	seen := map[string]struct{}{}
	for index, hit := range value.Hits {
		if err := hit.Validate(); err != nil || hit.Rank != index+1 {
			return fmt.Errorf("invalid ranked retrieval hit")
		}
		key := retrievalParentKey(hit)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate ranked parent")
		}
		seen[key] = struct{}{}
	}
	return nil
}

// RetrievalCase is the complete set of Phase 12 arms for a query. Its
// validation prevents a hybrid result from concealing an absent standalone or
// lane-ablation observation.
type RetrievalCase struct {
	QueryID  string        `json:"query_id"`
	Rankings []CaseRanking `json:"rankings"`
}

// RetrievalArmExecutor is the Phase 13 handoff seam. Its implementation must
// adapt the Phase 11 production lanes and return their actual ranked parents;
// it must not substitute evaluation-only retrieval logic. Query-vector hashes
// are digests of request-local vectors, never vector contents.
type RetrievalArmExecutor interface {
	EvaluateArm(context.Context, evalcontract.EvaluationCase, RetrievalVariant) (RetrievalArmResult, error)
}

type RetrievalArmResult struct {
	Ranking      CaseRanking                   `json:"ranking"`
	Packaged     []BodyPackageHit              `json:"packaged,omitempty"`
	Segments     []search.EvaluationSegmentHit `json:"segments,omitempty"`
	FailureStage evalcontract.FailureStage     `json:"failure_stage,omitempty"`
}

// RetrievalArmFailure is the typed error an adapter returns for a required
// failed/timed-out arm. The orchestration core retains it as a zero-valued
// denominator observation; untyped errors remain structural failures.
type RetrievalArmFailure struct{ Stage evalcontract.FailureStage }

func (value RetrievalArmFailure) Error() string {
	return "retrieval arm failure: " + string(value.Stage)
}

func (value RetrievalArmResult) Validate() error {
	if err := value.Ranking.Validate(); err != nil {
		return err
	}
	if value.FailureStage != "" {
		if !validRetrievalFailureStage(value.Ranking.Variant, value.FailureStage) || len(value.Ranking.Hits) != 0 || value.Ranking.QueryVectorSHA256 != "" || len(value.Packaged) != 0 || len(value.Segments) != 0 {
			return fmt.Errorf("invalid failed retrieval arm")
		}
	}
	return nil
}

type RetrievalVariantMetrics struct {
	Variant RetrievalVariant `json:"variant"`
	Metrics CaseResult       `json:"metrics"`
}

// RetrievalCaseEvidence is the complete metric, fidelity, fusion, and body
// evidence for one query. It has no source/query vector bodies and is safe to
// feed into the immutable artifact writer once a corpus run is authorized.
type RetrievalCaseEvidence struct {
	Case     RetrievalCase             `json:"case"`
	Arms     []RetrievalArmResult      `json:"arms"`
	Metrics  []RetrievalVariantMetrics `json:"metrics"`
	Fidelity CodecFidelity             `json:"fidelity"`
	Fusion   LaneContribution          `json:"fusion"`
	Body     BodyDiagnostic            `json:"body"`
}

type RetrievalEvaluationRun struct {
	Plan  RetrievalPlan           `json:"plan"`
	Cases []RetrievalCaseEvidence `json:"cases"`
}

// RunRetrievalEvaluation is an executable, corpus-independent orchestration
// core. It obtains every frozen arm from an injected adapter, checks that all
// vector-bearing arms reused one ephemeral query vector, then calculates each
// diagnostic once from the resulting parent rankings.
func RunRetrievalEvaluation(ctx context.Context, dataset EvaluationDataset, plan RetrievalPlan, executor RetrievalArmExecutor) (RetrievalEvaluationRun, error) {
	if err := dataset.Validate(); err != nil {
		return RetrievalEvaluationRun{}, err
	}
	if err := plan.Validate(); err != nil || executor == nil {
		return RetrievalEvaluationRun{}, fmt.Errorf("invalid retrieval evaluation core")
	}
	result := RetrievalEvaluationRun{Plan: plan, Cases: make([]RetrievalCaseEvidence, 0, len(dataset.Cases))}
	for _, evaluationCase := range dataset.Cases {
		if err := ctx.Err(); err != nil {
			return RetrievalEvaluationRun{}, err
		}
		arms := make([]RetrievalArmResult, 0, len(plan.Variants))
		for _, variant := range plan.Variants {
			arm, err := executor.EvaluateArm(ctx, evaluationCase, variant)
			if ctx.Err() != nil {
				return RetrievalEvaluationRun{}, ctx.Err()
			}
			if err != nil {
				var failure RetrievalArmFailure
				if !errors.As(err, &failure) {
					return RetrievalEvaluationRun{}, fmt.Errorf("evaluate %s/%s: %w", evaluationCase.ID, variant, err)
				}
				arm = RetrievalArmResult{Ranking: CaseRanking{QueryID: evaluationCase.ID, Variant: variant}, FailureStage: failure.Stage}
			}
			if arm.Ranking.QueryID != evaluationCase.ID || arm.Ranking.Variant != variant || (variant != VariantHybridFTSActiveCodec && len(arm.Packaged) != 0) || arm.Validate() != nil {
				return RetrievalEvaluationRun{}, fmt.Errorf("invalid injected retrieval arm")
			}
			arms = append(arms, arm)
		}
		evidence, err := buildRetrievalCaseEvidence(evaluationCase, plan, arms)
		if err != nil {
			return RetrievalEvaluationRun{}, err
		}
		result.Cases = append(result.Cases, evidence)
	}
	if err := result.Validate(dataset); err != nil {
		return RetrievalEvaluationRun{}, err
	}
	return result, nil
}

func (value RetrievalEvaluationRun) Validate(dataset EvaluationDataset) error {
	if err := dataset.Validate(); err != nil || value.Plan.Validate() != nil || len(value.Cases) != len(dataset.Cases) {
		return fmt.Errorf("invalid retrieval evaluation run")
	}
	seen := map[string]bool{}
	for _, evidence := range value.Cases {
		if seen[evidence.Case.QueryID] || evidence.Case.Validate(value.Plan) != nil || len(evidence.Arms) != len(value.Plan.Variants) || len(evidence.Metrics) != len(value.Plan.Variants) {
			return fmt.Errorf("invalid retrieval case evidence")
		}
		seen[evidence.Case.QueryID] = true
		var evaluationCase *evalcontract.EvaluationCase
		for index := range dataset.Cases {
			if dataset.Cases[index].ID == evidence.Case.QueryID {
				evaluationCase = &dataset.Cases[index]
				break
			}
		}
		if evaluationCase == nil {
			return fmt.Errorf("unknown retrieval case evidence")
		}
		expected, err := buildRetrievalCaseEvidence(*evaluationCase, value.Plan, evidence.Arms)
		if err != nil || !reflect.DeepEqual(expected, evidence) {
			return fmt.Errorf("forged or inconsistent retrieval evidence")
		}
	}
	for _, evaluationCase := range dataset.Cases {
		if !seen[evaluationCase.ID] {
			return fmt.Errorf("missing retrieval case evidence")
		}
	}
	return nil
}

func buildRetrievalCaseEvidence(evaluationCase evalcontract.EvaluationCase, plan RetrievalPlan, arms []RetrievalArmResult) (RetrievalCaseEvidence, error) {
	if len(arms) != len(plan.Variants) {
		return RetrievalCaseEvidence{}, fmt.Errorf("incomplete retrieval arms")
	}
	rankings := make([]CaseRanking, 0, len(arms))
	for index, arm := range arms {
		if arm.Ranking.QueryID != evaluationCase.ID || arm.Ranking.Variant != plan.Variants[index] || (plan.Variants[index] != VariantHybridFTSActiveCodec && len(arm.Packaged) != 0) || arm.Validate() != nil {
			return RetrievalCaseEvidence{}, fmt.Errorf("invalid retrieval arm evidence")
		}
		rankings = append(rankings, arm.Ranking)
	}
	caseValue := RetrievalCase{QueryID: evaluationCase.ID, Rankings: rankings}
	if err := caseValue.Validate(plan); err != nil {
		return RetrievalCaseEvidence{}, err
	}
	if err := compatibleEphemeralQuery(arms); err != nil {
		return RetrievalCaseEvidence{}, err
	}
	evidence := RetrievalCaseEvidence{Case: caseValue, Arms: append([]RetrievalArmResult(nil), arms...), Metrics: make([]RetrievalVariantMetrics, 0, len(rankings))}
	for index, ranking := range rankings {
		metrics, err := EvaluateRetrievalRanking(evaluationCase, ranking, plan.Ks, failureForArm(arms[index]))
		if err != nil {
			return RetrievalCaseEvidence{}, err
		}
		evidence.Metrics = append(evidence.Metrics, RetrievalVariantMetrics{Variant: ranking.Variant, Metrics: metrics})
	}
	k := plan.Ks[len(plan.Ks)-1]
	var err error
	if failure := firstArmFailure(arms[1], arms[2]); failure != "" {
		evidence.Fidelity = CodecFidelity{FailureStage: failure}
	} else if evidence.Fidelity, err = CompareCodecFidelity(evaluationCase, rankings[1], rankings[2], k); err != nil {
		return RetrievalCaseEvidence{}, err
	}
	if failure := firstArmFailure(arms[0], arms[2], arms[5]); failure != "" {
		evidence.Fusion = LaneContribution{FailureStage: failure}
	} else if evidence.Fusion, err = DiagnoseFusion(evaluationCase, rankings[0], rankings[2], rankings[5], k); err != nil {
		return RetrievalCaseEvidence{}, err
	}
	if arms[5].FailureStage != "" {
		evidence.Body = BodyDiagnostic{FailureStage: arms[5].FailureStage}
	} else if evidence.Body, err = DiagnoseBodyPackaging(evaluationCase, rankings[5], arms[5].Packaged, k); err != nil {
		return RetrievalCaseEvidence{}, err
	}
	return evidence, nil
}

func failureForArm(arm RetrievalArmResult) error {
	if arm.FailureStage == "" {
		return nil
	}
	return RetrievalArmFailure{Stage: arm.FailureStage}
}

func firstArmFailure(arms ...RetrievalArmResult) evalcontract.FailureStage {
	for _, arm := range arms {
		if arm.FailureStage != "" {
			return arm.FailureStage
		}
	}
	return ""
}

func (value RetrievalCase) Validate(plan RetrievalPlan) error {
	if err := plan.Validate(); err != nil || !validID(value.QueryID) || len(value.Rankings) != len(plan.Variants) {
		return fmt.Errorf("invalid retrieval case")
	}
	for index, ranking := range value.Rankings {
		if ranking.QueryID != value.QueryID || ranking.Variant != plan.Variants[index] {
			return fmt.Errorf("incomplete or unordered retrieval case")
		}
		if err := ranking.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// EvaluateRetrievalRanking deliberately delegates human-gold metrics to the
// Phase 07 shared metric implementation. The adapter is identity-only.
func EvaluateRetrievalRanking(evaluationCase evalcontract.EvaluationCase, ranking CaseRanking, ks []int, failure error) (CaseResult, error) {
	if ranking.QueryID != evaluationCase.ID {
		return CaseResult{}, fmt.Errorf("ranking query does not match evaluation case")
	}
	if err := ranking.Validate(); err != nil {
		return CaseResult{}, err
	}
	hits := make([]lexical.Hit, 0, len(ranking.Hits))
	for _, hit := range ranking.Hits {
		hits = append(hits, lexical.Hit{Path: hit.Path, IndexedSHA256: hit.IndexedSHA256, QualifiedSymbol: hit.QualifiedSymbol, StartByte: hit.StartByte, EndByte: hit.EndByte})
	}
	if failure == nil {
		return EvaluateCase(evaluationCase, hits, ks, nil)
	}
	var armFailure RetrievalArmFailure
	if !errors.As(failure, &armFailure) || !validRetrievalFailureStage(ranking.Variant, armFailure.Stage) {
		return CaseResult{}, fmt.Errorf("invalid retrieval arm failure")
	}
	result, err := EvaluateCase(evaluationCase, nil, ks, errors.New("required retrieval arm failed"))
	if err != nil {
		return CaseResult{}, err
	}
	result.FailureStage = armFailure.Stage
	result.FirstLoss = evalcontract.FirstLoss("OPERATION_FAILURE:" + string(armFailure.Stage))
	return result, nil
}

// CodecFidelity keeps representation fidelity separate from human relevance.
// GoldF32Retention is nil when no gold parent appears in the f32 cutoff.
type CodecFidelity struct {
	Observed              bool                      `json:"observed"`
	FailureStage          evalcontract.FailureStage `json:"failure_stage,omitempty"`
	TopKRetention         float64                   `json:"top_k_retention"`
	F32CandidatesMissing  int                       `json:"f32_candidates_missing"`
	Top1Mismatch          bool                      `json:"top_1_mismatch"`
	GoldF32Retention      *float64                  `json:"gold_f32_retention,omitempty"`
	MeanRankDisplacement  float64                   `json:"mean_rank_displacement"`
	MedianDisplacement    float64                   `json:"median_displacement"`
	P95Displacement       float64                   `json:"p95_displacement"`
	MaxDisplacement       int                       `json:"max_displacement"`
	PairwiseInversionRate *float64                  `json:"pairwise_inversion_rate,omitempty"`
	F32ScoreTieRate       *float64                  `json:"f32_score_tie_rate,omitempty"`
	CodecScoreTieRate     *float64                  `json:"codec_score_tie_rate,omitempty"`
	F32BoundaryTie        bool                      `json:"f32_boundary_tie"`
	CodecBoundaryTie      bool                      `json:"codec_boundary_tie"`
}

func CompareCodecFidelity(evaluationCase evalcontract.EvaluationCase, targetF32, codec CaseRanking, k int) (CodecFidelity, error) {
	if err := evaluationCase.Validate(); err != nil {
		return CodecFidelity{}, err
	}
	if targetF32.QueryID != evaluationCase.ID || codec.QueryID != evaluationCase.ID || targetF32.Variant != VariantTargetF32 || codec.Variant != VariantServingActiveCodec || k <= 0 {
		return CodecFidelity{}, fmt.Errorf("invalid codec fidelity comparison")
	}
	if err := targetF32.Validate(); err != nil {
		return CodecFidelity{}, err
	}
	if err := codec.Validate(); err != nil {
		return CodecFidelity{}, err
	}
	f32 := limitedHits(targetF32.Hits, k)
	encoded := limitedHits(codec.Hits, k)
	f32Ranks, codecRanks := hitRanks(f32), hitRanks(encoded)
	shared := make([]string, 0, len(f32Ranks))
	missing := 0
	for key := range f32Ranks {
		if _, ok := codecRanks[key]; ok {
			shared = append(shared, key)
		} else {
			missing++
		}
	}
	sort.Strings(shared)
	result := CodecFidelity{Observed: true, F32CandidatesMissing: missing}
	if len(f32) > 0 {
		result.TopKRetention = float64(len(shared)) / float64(len(f32))
	}
	if len(f32) > 0 && len(encoded) > 0 {
		result.Top1Mismatch = retrievalParentKey(f32[0]) != retrievalParentKey(encoded[0])
	} else {
		result.Top1Mismatch = len(f32) != len(encoded)
	}
	gold := directParentKeys(evaluationCase)
	goldInF32, goldInBoth := 0, 0
	for key := range gold {
		if _, ok := f32Ranks[key]; ok {
			goldInF32++
			if _, present := codecRanks[key]; present {
				goldInBoth++
			}
		}
	}
	if goldInF32 > 0 {
		value := float64(goldInBoth) / float64(goldInF32)
		result.GoldF32Retention = &value
	}
	displacements := make([]int, 0, len(shared))
	for _, key := range shared {
		d := abs(f32Ranks[key] - codecRanks[key])
		displacements = append(displacements, d)
		result.MeanRankDisplacement += float64(d)
		if d > result.MaxDisplacement {
			result.MaxDisplacement = d
		}
	}
	if len(displacements) > 0 {
		result.MeanRankDisplacement /= float64(len(displacements))
		sort.Ints(displacements)
		result.MedianDisplacement = percentile(displacements, .50)
		result.P95Displacement = percentile(displacements, .95)
		if len(shared) > 1 {
			inversions, pairs := 0, 0
			for left := 0; left < len(shared); left++ {
				for right := left + 1; right < len(shared); right++ {
					f32Left, f32Right := f32[f32Ranks[shared[left]]-1], f32[f32Ranks[shared[right]]-1]
					codecLeft, codecRight := encoded[codecRanks[shared[left]]-1], encoded[codecRanks[shared[right]]-1]
					if f32Left.Score == nil || f32Right.Score == nil || codecLeft.Score == nil || codecRight.Score == nil || *f32Left.Score == *f32Right.Score || *codecLeft.Score == *codecRight.Score {
						continue
					}
					pairs++
					if (f32Ranks[shared[left]] < f32Ranks[shared[right]]) != (codecRanks[shared[left]] < codecRanks[shared[right]]) {
						inversions++
					}
				}
			}
			if pairs > 0 {
				value := float64(inversions) / float64(pairs)
				result.PairwiseInversionRate = &value
			}
		}
	}
	result.F32ScoreTieRate, result.F32BoundaryTie = tieDiagnostics(f32, targetF32.Hits, k)
	result.CodecScoreTieRate, result.CodecBoundaryTie = tieDiagnostics(encoded, codec.Hits, k)
	return result, nil
}

// LaneContribution captures RRF diagnostics using ranks only. Native scores
// remain deliberately incomparable across FTS, f32, codecs, and RRF.
type LaneContribution struct {
	Observed            bool                      `json:"observed"`
	FailureStage        evalcontract.FailureStage `json:"failure_stage,omitempty"`
	CandidateJaccard    float64                   `json:"candidate_jaccard"`
	FTSOnlyCandidates   int                       `json:"fts_only_candidates"`
	DenseOnlyCandidates int                       `json:"dense_only_candidates"`
	BothLaneCandidates  int                       `json:"both_lane_candidates"`
	FusedFTSOnly        int                       `json:"fused_fts_only"`
	FusedDenseOnly      int                       `json:"fused_dense_only"`
	FusedBothLanes      int                       `json:"fused_both_lanes"`
	FusionRescue        bool                      `json:"fusion_rescue"`
	FusionHarm          bool                      `json:"fusion_harm"`
	MeanRankMovement    float64                   `json:"mean_rank_movement"`
	P95RankMovement     float64                   `json:"p95_rank_movement"`
	MaxRankMovement     int                       `json:"max_rank_movement"`
}

func DiagnoseFusion(evaluationCase evalcontract.EvaluationCase, fts, dense, fused CaseRanking, k int) (LaneContribution, error) {
	if err := evaluationCase.Validate(); err != nil {
		return LaneContribution{}, err
	}
	if fts.QueryID != evaluationCase.ID || dense.QueryID != evaluationCase.ID || fused.QueryID != evaluationCase.ID || fts.Variant != VariantFTS || dense.Variant != VariantServingActiveCodec || fused.Variant != VariantHybridFTSActiveCodec || k <= 0 {
		return LaneContribution{}, fmt.Errorf("invalid fusion diagnostic")
	}
	for _, ranking := range []CaseRanking{fts, dense, fused} {
		if err := ranking.Validate(); err != nil {
			return LaneContribution{}, err
		}
	}
	ftsHits, denseHits, fusedHits := limitedHits(fts.Hits, k), limitedHits(dense.Hits, k), limitedHits(fused.Hits, k)
	ftsRanks, denseRanks := hitRanks(ftsHits), hitRanks(denseHits)
	result := LaneContribution{Observed: true}
	union := map[string]struct{}{}
	for key := range ftsRanks {
		union[key] = struct{}{}
		if _, ok := denseRanks[key]; ok {
			result.BothLaneCandidates++
		} else {
			result.FTSOnlyCandidates++
		}
	}
	for key := range denseRanks {
		union[key] = struct{}{}
		if _, ok := ftsRanks[key]; !ok {
			result.DenseOnlyCandidates++
		}
	}
	if len(union) > 0 {
		result.CandidateJaccard = float64(result.BothLaneCandidates) / float64(len(union))
	}
	movements := []int{}
	for _, hit := range fusedHits {
		key := retrievalParentKey(hit)
		_, inFTS := ftsRanks[key]
		_, inDense := denseRanks[key]
		switch {
		case inFTS && inDense:
			result.FusedBothLanes++
		case inFTS:
			result.FusedFTSOnly++
		case inDense:
			result.FusedDenseOnly++
		}
		best := 0
		if rank, ok := ftsRanks[key]; ok {
			best = rank
		}
		if rank, ok := denseRanks[key]; ok && (best == 0 || rank < best) {
			best = rank
		}
		if best > 0 {
			d := abs(hit.Rank - best)
			movements = append(movements, d)
			result.MeanRankMovement += float64(d)
			if d > result.MaxRankMovement {
				result.MaxRankMovement = d
			}
		}
	}
	if len(movements) > 0 {
		result.MeanRankMovement /= float64(len(movements))
		sort.Ints(movements)
		result.P95RankMovement = percentile(movements, .95)
	}
	ftsMetric, err := EvaluateRetrievalRanking(evaluationCase, fts, []int{k}, nil)
	if err != nil {
		return LaneContribution{}, err
	}
	denseMetric, err := EvaluateRetrievalRanking(evaluationCase, dense, []int{k}, nil)
	if err != nil {
		return LaneContribution{}, err
	}
	fusedMetric, err := EvaluateRetrievalRanking(evaluationCase, fused, []int{k}, nil)
	if err != nil {
		return LaneContribution{}, err
	}
	result.FusionRescue = fusedMetric.CompleteRequirementHitAt[k] && (!ftsMetric.CompleteRequirementHitAt[k] || !denseMetric.CompleteRequirementHitAt[k])
	result.FusionHarm = !fusedMetric.CompleteRequirementHitAt[k] && (ftsMetric.CompleteRequirementHitAt[k] || denseMetric.CompleteRequirementHitAt[k])
	return result, nil
}

// BodyPackageHit holds body coordinates and accounting only. It never carries
// source text, and it lets the evaluator distinguish body-package loss from a
// fused retrieval miss.
type BodyPackageHit struct {
	Hit            RetrievalHit             `json:"hit"`
	BodyRange      *evalcontract.SourceSpan `json:"body_range,omitempty"`
	BodyComplete   bool                     `json:"body_complete"`
	BodyBytes      int                      `json:"body_bytes"`
	BodySHA256     string                   `json:"body_sha256,omitempty"`
	OmissionReason search.OmissionReason    `json:"omission_reason,omitempty"`
}

type BodyDiagnostic struct {
	Observed                    bool                      `json:"observed"`
	FailureStage                evalcontract.FailureStage `json:"failure_stage,omitempty"`
	FusedRequirementCoverage    float64                   `json:"fused_requirement_coverage"`
	PackagedRequirementCoverage float64                   `json:"packaged_requirement_coverage"`
	FusedGroups                 int                       `json:"fused_groups"`
	PackagedGroups              int                       `json:"packaged_groups"`
	SerializedBytes             int                       `json:"serialized_bytes"`
	RelevantByteDensity         *float64                  `json:"relevant_byte_density,omitempty"`
	DuplicateBodyRatio          float64                   `json:"duplicate_body_ratio"`
	OmissionCounts              map[string]int            `json:"omission_counts"`
}

func DiagnoseBodyPackaging(evaluationCase evalcontract.EvaluationCase, fused CaseRanking, packaged []BodyPackageHit, k int) (BodyDiagnostic, error) {
	if err := evaluationCase.Validate(); err != nil {
		return BodyDiagnostic{}, err
	}
	if fused.QueryID != evaluationCase.ID || !isHybrid(fused.Variant) || k <= 0 {
		return BodyDiagnostic{}, fmt.Errorf("invalid body diagnostic")
	}
	if err := fused.Validate(); err != nil {
		return BodyDiagnostic{}, err
	}
	fusedHits := limitedHits(fused.Hits, k)
	if len(packaged) != len(fusedHits) {
		return BodyDiagnostic{}, fmt.Errorf("invalid packaged hit count")
	}
	for index, body := range packaged {
		if err := body.Hit.Validate(); err != nil || body.Hit.Rank != index+1 || index >= len(fusedHits) || retrievalParentKey(body.Hit) != retrievalParentKey(fusedHits[index]) || body.BodyBytes < 0 {
			return BodyDiagnostic{}, fmt.Errorf("invalid packaged hit")
		}
		if body.BodyRange == nil {
			if body.BodyComplete || body.BodyBytes != 0 || body.BodySHA256 != "" || (body.OmissionReason != search.BodyOmittedBudget && body.OmissionReason != search.BodyOmittedNoMatchedSegment) {
				return BodyDiagnostic{}, fmt.Errorf("invalid omitted body")
			}
		} else {
			fullParent := body.BodyRange.StartByte == body.Hit.StartByte && body.BodyRange.EndByte == body.Hit.EndByte
			if err := body.BodyRange.Validate(); err != nil || body.BodyRange.Path != body.Hit.Path || body.BodyRange.ContentSHA256 != body.Hit.IndexedSHA256 || body.BodyRange.QualifiedSymbol != body.Hit.QualifiedSymbol || body.BodyRange.StartByte < body.Hit.StartByte || body.BodyRange.EndByte > body.Hit.EndByte || body.BodyComplete != fullParent || body.BodyBytes != body.BodyRange.EndByte-body.BodyRange.StartByte || !validSHA256(body.BodySHA256) || body.OmissionReason != "" {
				return BodyDiagnostic{}, fmt.Errorf("invalid packaged body range")
			}
		}
	}
	fusedGroups := groups(evaluationCase.RequiredGroups, retrievalLexicalHits(fusedHits))
	packagedGroups := map[string]bool{}
	seenBodies := map[string]bool{}
	duplicates, bodies, relevantBytes := 0, 0, 0
	result := BodyDiagnostic{Observed: true, FusedGroups: len(fusedGroups), OmissionCounts: map[string]int{}}
	for _, body := range packaged {
		if body.BodyRange == nil {
			result.OmissionCounts[string(body.OmissionReason)]++
			continue
		}
		result.SerializedBytes += body.BodyBytes
		key := body.BodySHA256
		if seenBodies[key] {
			duplicates++
		}
		seenBodies[key] = true
		bodies++
		for _, group := range evaluationCase.RequiredGroups {
			for _, alternative := range group.Alternatives {
				complete := true
				for _, span := range alternative.Spans {
					if !bodyContains(body, span) {
						complete = false
						break
					}
				}
				if complete {
					packagedGroups[group.ID] = true
					break
				}
			}
		}
		relevantBytes += directRelevantBytes(body, evaluationCase.Judgments)
	}
	result.PackagedGroups = len(packagedGroups)
	if len(evaluationCase.RequiredGroups) > 0 {
		result.FusedRequirementCoverage = float64(result.FusedGroups) / float64(len(evaluationCase.RequiredGroups))
		result.PackagedRequirementCoverage = float64(result.PackagedGroups) / float64(len(evaluationCase.RequiredGroups))
	}
	if bodies > 0 {
		result.DuplicateBodyRatio = float64(duplicates) / float64(bodies)
	}
	if result.SerializedBytes > 0 {
		value := float64(relevantBytes) / float64(result.SerializedBytes)
		result.RelevantByteDensity = &value
	}
	return result, nil
}

// CorePromotionEvidence validates Phase 12 evidence composition. The shared
// wire schemas remain in evalcontract; this layer adds Phase 12's scoped
// relation between frozen contract, confirmation controls, artifacts, and
// result without creating a second promotion schema.
type CorePromotionEvidence struct {
	Contract             evalcontract.PromotionContract     `json:"contract"`
	Result               evalcontract.PromotionResult       `json:"result"`
	ConfirmationManifest evalcontract.EvaluationRunManifest `json:"confirmation_manifest"`
	Artifacts            evalcontract.ArtifactManifest      `json:"artifacts"`
	ArtifactChecksum     string                             `json:"artifact_checksum"`
	CompletionArtifact   evalcontract.ArtifactEntry         `json:"completion_artifact"`
}

func (value CorePromotionEvidence) Validate() error {
	if err := value.Contract.Validate(); err != nil {
		return err
	}
	if err := value.Result.Validate(); err != nil {
		return err
	}
	if err := value.ConfirmationManifest.Validate(); err != nil {
		return err
	}
	if err := value.Artifacts.Validate(); err != nil || !value.Artifacts.Complete {
		return fmt.Errorf("incomplete core evidence artifacts")
	}
	if value.Contract.Scope != evalcontract.CoreRetrieval || value.Result.Scope != evalcontract.CoreRetrieval || value.Contract.ConfirmationDatasetSHA256 != value.ConfirmationManifest.QueryManifestSHA256 || value.Contract.PairedControls != value.ConfirmationManifest.PairedControls || value.Contract.ReviewProtocolVersion != value.ConfirmationManifest.ReviewProtocolVersion || value.Contract.RelevanceAuthority != value.ConfirmationManifest.RelevanceAuthority || value.Contract.ReviewValidation != value.ConfirmationManifest.ReviewValidation || value.Result.ReviewProtocolVersion != value.Contract.ReviewProtocolVersion || value.Result.RelevanceAuthority != value.Contract.RelevanceAuthority || value.Result.ReviewValidation != value.Contract.ReviewValidation || !validSHA256(value.ArtifactChecksum) {
		return fmt.Errorf("incompatible core promotion evidence")
	}
	checksum, err := evalcontract.ArtifactChecksum(value.Artifacts.Entries)
	if err != nil || checksum != value.ArtifactChecksum {
		return fmt.Errorf("artifact checksum mismatch")
	}
	if !sameStrings(value.Contract.FrozenGates, value.Result.ApplicableGates) || !uniqueStrings(value.Result.PassedGates) || !uniqueStrings(value.Result.FailedGates) || !subsetsDisjoint(value.Result.PassedGates, value.Result.FailedGates, value.Result.ApplicableGates) || !containsString(value.Result.PrerequisiteSHA256, value.ArtifactChecksum) {
		return fmt.Errorf("invalid promotion gate result")
	}
	if value.Result.Status == evalcontract.PromotionEvidenceReady && !sameStrings(value.Result.PassedGates, value.Result.ApplicableGates) {
		return fmt.Errorf("ready promotion did not pass every applicable gate")
	}
	if value.CompletionArtifact.Path != "artifact-checksums.json" || value.CompletionArtifact.MediaType != "application/json" || value.CompletionArtifact.ByteSize < 0 || !validSHA256(value.CompletionArtifact.SHA256) {
		return fmt.Errorf("invalid artifact completion marker")
	}
	for _, path := range coreEvidencePaths {
		if !hasArtifact(value.Artifacts.Entries, path) {
			return fmt.Errorf("missing core evidence artifact %q", path)
		}
	}
	return nil
}

var coreEvidencePaths = []string{"run-manifest.json", "per-query-trace.jsonl", "fts-candidates.jsonl", "dense-segment-candidates.jsonl", "collapsed-parent-candidates.jsonl", "rrf-results.jsonl", "inline-body-packages.jsonl", "per-query-metrics.jsonl", "aggregate-metrics.json", "cohort-language-report.json", "first-loss-report.json", "provider-usage.json", "implementation-audit.json", "promotion-contract.json", "promotion-result.json", "report.md"}

func knownVariant(value RetrievalVariant) bool {
	for _, variant := range requiredRetrievalVariants {
		if value == variant {
			return true
		}
	}
	return false
}
func isHybrid(value RetrievalVariant) bool {
	return value == VariantHybridFTSTargetF32 || value == VariantHybridFTSActiveCodec || value == VariantHybridWithoutFTS || value == VariantHybridWithoutDense
}
func retrievalParentKey(value RetrievalHit) string {
	return value.Path + "\x00" + value.IndexedSHA256 + "\x00" + value.QualifiedSymbol
}
func limitedHits(values []RetrievalHit, k int) []RetrievalHit {
	if len(values) > k {
		return values[:k]
	}
	return values
}
func hitRanks(values []RetrievalHit) map[string]int {
	result := make(map[string]int, len(values))
	for _, value := range values {
		result[retrievalParentKey(value)] = value.Rank
	}
	return result
}
func retrievalLexicalHits(values []RetrievalHit) []lexical.Hit {
	result := make([]lexical.Hit, 0, len(values))
	for _, value := range values {
		result = append(result, lexical.Hit{Path: value.Path, IndexedSHA256: value.IndexedSHA256, QualifiedSymbol: value.QualifiedSymbol, StartByte: value.StartByte, EndByte: value.EndByte})
	}
	return result
}
func retrievalSegmentLexicalHits(values []search.EvaluationSegmentHit) []lexical.Hit {
	result := make([]lexical.Hit, 0, len(values))
	for _, value := range values {
		result = append(result, lexical.Hit{Path: value.Path, IndexedSHA256: value.IndexedSHA256, QualifiedSymbol: value.QualifiedSymbol, StartByte: value.ParentStartByte, EndByte: value.ParentEndByte})
	}
	return result
}
func directParentKeys(value evalcontract.EvaluationCase) map[string]bool {
	result := map[string]bool{}
	for _, judgment := range value.Judgments {
		if judgment.Grade == evalcontract.DirectRequirement {
			result[pid(judgment.Span)] = true
		}
	}
	return result
}
func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
func percentile(values []int, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	index := int(math.Ceil(p*float64(len(values)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return float64(values[index])
}
func tieDiagnostics(limited, all []RetrievalHit, k int) (*float64, bool) {
	comparable, ties := 0, 0
	for left := 0; left < len(limited); left++ {
		if limited[left].Score == nil {
			continue
		}
		for right := left + 1; right < len(limited); right++ {
			if limited[right].Score == nil {
				continue
			}
			comparable++
			if *limited[left].Score == *limited[right].Score {
				ties++
			}
		}
	}
	var rate *float64
	if comparable > 0 {
		value := float64(ties) / float64(comparable)
		rate = &value
	}
	boundary := len(all) > k && len(limited) == k && limited[k-1].Score != nil && all[k].Score != nil && *limited[k-1].Score == *all[k].Score
	return rate, boundary
}
func bodyContains(body BodyPackageHit, span evalcontract.SourceSpan) bool {
	return body.BodyRange != nil && body.BodyRange.Path == span.Path && body.BodyRange.ContentSHA256 == span.ContentSHA256 && body.BodyRange.QualifiedSymbol == span.QualifiedSymbol && body.BodyRange.StartByte <= span.StartByte && body.BodyRange.EndByte >= span.EndByte
}
func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
func uniqueStrings(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}
func subsetsDisjoint(passed, failed, applicable []string) bool {
	allowed := map[string]bool{}
	for _, value := range applicable {
		if value == "" || allowed[value] {
			return false
		}
		allowed[value] = true
	}
	seen := map[string]bool{}
	for _, values := range [][]string{passed, failed} {
		for _, value := range values {
			if !allowed[value] || seen[value] {
				return false
			}
			seen[value] = true
		}
	}
	return true
}
func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func hasArtifact(entries []evalcontract.ArtifactEntry, wanted string) bool {
	for _, entry := range entries {
		if entry.Path == wanted {
			return true
		}
	}
	return false
}

func compatibleEphemeralQuery(arms []RetrievalArmResult) error {
	var expected string
	for _, arm := range arms {
		ranking := arm.Ranking
		if arm.FailureStage != "" {
			continue
		}
		if !variantUsesQueryVector(ranking.Variant) {
			if ranking.QueryVectorSHA256 != "" {
				return fmt.Errorf("non-vector arm cannot carry a query vector hash")
			}
			continue
		}
		if !validSHA256(ranking.QueryVectorSHA256) {
			return fmt.Errorf("vector-bearing arm lacks query vector hash")
		}
		if expected == "" {
			expected = ranking.QueryVectorSHA256
		} else if ranking.QueryVectorSHA256 != expected {
			return fmt.Errorf("retrieval arms used different ephemeral query vectors")
		}
	}
	return nil
}

// BuildRetrievalTrace derives the normative Phase 12 stage trace from the
// frozen eight-arm evidence. It is intentionally kept with the metric core so
// artifact publication cannot invent a second first-loss implementation.
func BuildRetrievalTrace(evaluationCase evalcontract.EvaluationCase, evidence RetrievalCaseEvidence) (evalcontract.StageTrace, error) {
	if evaluationCase.ID != evidence.Case.QueryID || evidence.Case.Validate(RetrievalPlan{Variants: requiredRetrievalVariants, Ks: []int{1}}) != nil {
		return evalcontract.StageTrace{}, fmt.Errorf("invalid retrieval trace evidence")
	}
	byVariant := map[RetrievalVariant]RetrievalArmResult{}
	for _, arm := range evidence.Arms {
		byVariant[arm.Ranking.Variant] = arm
	}
	groups := make([]string, 0, len(evaluationCase.RequiredGroups))
	for _, group := range evaluationCase.RequiredGroups {
		groups = append(groups, group.ID)
	}
	present := func(arm RetrievalArmResult) map[string]bool {
		if arm.FailureStage != "" {
			return map[string]bool{}
		}
		return groupsForHits(evaluationCase.RequiredGroups, retrievalLexicalHits(arm.Ranking.Hits))
	}
	presentSegments := func(arm RetrievalArmResult) map[string]bool {
		if arm.FailureStage != "" {
			return map[string]bool{}
		}
		return groupsForHits(evaluationCase.RequiredGroups, retrievalSegmentLexicalHits(arm.Segments))
	}
	withQuery := func(stage evalcontract.Stage, arm RetrievalArmResult) evalcontract.StageObservation {
		observation := retrievalObservation(stage, groups, present(arm), len(arm.Ranking.Hits), arm.FailureStage)
		observation.QueryVectorSHA256 = arm.Ranking.QueryVectorSHA256
		return observation
	}
	withDenseSegments := func(arm RetrievalArmResult) evalcontract.StageObservation {
		observation := retrievalObservation(evalcontract.StageDenseSegment, groups, presentSegments(arm), len(arm.Segments), arm.FailureStage)
		observation.QueryVectorSHA256 = arm.Ranking.QueryVectorSHA256
		return observation
	}
	withParentCollapse := func(arm RetrievalArmResult) evalcontract.StageObservation {
		observation := retrievalObservation(evalcontract.StageParentCollapse, groups, present(arm), len(arm.Ranking.Hits), arm.FailureStage)
		observation.QueryVectorSHA256 = arm.Ranking.QueryVectorSHA256
		return observation
	}
	fts := byVariant[VariantFTS]
	dense := byVariant[VariantServingActiveCodec]
	union := byVariant[VariantProviderUnion]
	fused := byVariant[VariantHybridFTSActiveCodec]
	providerUnion := retrievalObservation(evalcontract.StageProviderUnion, groups, mergeGroupPresence(present(fts), presentSegments(dense)), providerUnionCandidates(fts, dense), union.FailureStage)
	providerUnion.QueryVectorSHA256 = union.Ranking.QueryVectorSHA256
	bodyPresent := map[string]bool{}
	if fused.FailureStage == "" {
		for _, group := range evaluationCase.RequiredGroups {
			for _, alternative := range group.Alternatives {
				complete := true
				for _, span := range alternative.Spans {
					found := false
					for _, body := range fused.Packaged {
						if bodyContains(body, span) {
							found = true
							break
						}
					}
					if !found {
						complete = false
						break
					}
				}
				if complete {
					bodyPresent[group.ID] = true
					break
				}
			}
		}
	}
	observations := []evalcontract.StageObservation{
		requiredObservation(evalcontract.StageSourceDiscovery, groups, allPresent(groups), 0, ""),
		requiredObservation(evalcontract.StageParserChunker, groups, allPresent(groups), 0, ""),
		withQuery(evalcontract.StageFTSCandidate, fts),
		withDenseSegments(dense),
		providerUnion,
		withParentCollapse(union),
		withQuery(evalcontract.StageRRFFusion, fused),
		retrievalObservation(evalcontract.StageBodyPackaging, groups, bodyPresent, len(fused.Packaged), fused.FailureStage),
		{Stage: evalcontract.StageAssistantUse, Required: false, Status: evalcontract.ObservationNotObserved},
		{Stage: evalcontract.StageAssistantResolution, Required: false, Status: evalcontract.ObservationNotObserved},
		{Stage: evalcontract.StageOperational, Required: true, Status: evalcontract.Observed, Denominators: []evalcontract.DenominatorRecord{{Name: "operation_attempts", TruthUnit: "operation", Count: 1}}},
	}
	observations[7].QueryVectorSHA256 = fused.Ranking.QueryVectorSHA256
	if err := preserveRetrievalFirstLoss(observations); err != nil {
		return evalcontract.StageTrace{}, err
	}
	for _, arm := range evidence.Arms {
		if arm.FailureStage != "" {
			observations[10].FailureStage = arm.FailureStage
			break
		}
	}
	terminal := evalcontract.TerminalComplete
	if observations[10].FailureStage != "" {
		terminal = evalcontract.TerminalFailed
	}
	trace := evalcontract.StageTrace{SchemaVersion: evalcontract.SchemaVersion, QueryID: evaluationCase.ID, RequiredGroupIDs: groups, Observations: observations, TerminalState: terminal}
	if err := trace.Validate(); err != nil {
		return evalcontract.StageTrace{}, err
	}
	return trace, nil
}

func mergeGroupPresence(left, right map[string]bool) map[string]bool {
	result := make(map[string]bool, len(left)+len(right))
	for group, present := range left {
		result[group] = present
	}
	for group, present := range right {
		result[group] = result[group] || present
	}
	return result
}

// providerUnionCandidates is the exact union of FTS parents and dense segment
// parents before collapse/ranking. It intentionally does not count segments
// twice when several segments belong to one parent.
func providerUnionCandidates(fts, dense RetrievalArmResult) int {
	parents := map[string]struct{}{}
	if fts.FailureStage == "" {
		for _, hit := range fts.Ranking.Hits {
			parents[retrievalParentKey(hit)] = struct{}{}
		}
	}
	if dense.FailureStage == "" {
		for _, segment := range dense.Segments {
			key := segment.Path + "\x00" + segment.IndexedSHA256 + "\x00" + segment.QualifiedSymbol + fmt.Sprintf("\x00%d\x00%d", segment.ParentStartByte, segment.ParentEndByte)
			parents[key] = struct{}{}
		}
	}
	return len(parents)
}

func preserveRetrievalFirstLoss(observations []evalcontract.StageObservation) error {
	lost := map[string]evalcontract.FirstLoss{}
	providerUnionReached := false
	for index := range observations {
		observation := &observations[index]
		if observation.Stage == evalcontract.StageProviderUnion {
			providerUnionReached = true
		}
		if !providerUnionReached || !observation.Required || observation.Stage == evalcontract.StageOperational {
			continue
		}
		for groupIndex := range observation.GroupObservations {
			group := &observation.GroupObservations[groupIndex]
			if original, exists := lost[group.GroupID]; exists {
				if group.Present {
					// A downstream lane may have independently found the group,
					// but it did not survive the staged evaluation funnel.
					group.Present = false
				}
				group.FirstLoss = original
				continue
			}
			if !group.Present {
				lost[group.GroupID] = group.FirstLoss
			}
		}
	}
	return nil
}

func retrievalObservation(stage evalcontract.Stage, groupIDs []string, present map[string]bool, candidates int, failure evalcontract.FailureStage) evalcontract.StageObservation {
	observation := requiredObservation(stage, groupIDs, present, candidates, failure)
	if failure != "" {
		return observation
	}
	for index := range observation.GroupObservations {
		if !observation.GroupObservations[index].Present {
			observation.GroupObservations[index].FirstLoss = retrievalStageLoss(stage)
		}
	}
	return observation
}

func retrievalStageLoss(stage evalcontract.Stage) evalcontract.FirstLoss {
	switch stage {
	case evalcontract.StageDenseSegment:
		return evalcontract.DenseSegmentMiss
	case evalcontract.StageProviderUnion:
		return evalcontract.ProviderUnionMiss
	case evalcontract.StageParentCollapse:
		return evalcontract.SegmentParentCollapse
	case evalcontract.StageRRFFusion:
		return evalcontract.RRFFusion
	case evalcontract.StageBodyPackaging:
		return evalcontract.BodyPackaging
	default:
		return evalcontract.FTSCandidateMiss
	}
}

func variantUsesQueryVector(value RetrievalVariant) bool {
	switch value {
	case VariantTargetF32, VariantServingActiveCodec, VariantProviderUnion, VariantHybridFTSTargetF32, VariantHybridFTSActiveCodec, VariantHybridWithoutFTS:
		return true
	default:
		return false
	}
}

func validRetrievalFailureStage(variant RetrievalVariant, value evalcontract.FailureStage) bool {
	stage := evalcontract.Stage(value)
	switch variant {
	case VariantFTS:
		return stage == evalcontract.StageFTSCandidate || stage == evalcontract.StageOperational
	case VariantTargetF32, VariantServingActiveCodec, VariantHybridWithoutFTS:
		return stage == evalcontract.StageDenseSegment || stage == evalcontract.StageParentCollapse || stage == evalcontract.StageOperational
	case VariantProviderUnion:
		return stage == evalcontract.StageFTSCandidate || stage == evalcontract.StageDenseSegment || stage == evalcontract.StageProviderUnion || stage == evalcontract.StageOperational
	case VariantHybridFTSTargetF32, VariantHybridFTSActiveCodec:
		return stage == evalcontract.StageFTSCandidate || stage == evalcontract.StageDenseSegment || stage == evalcontract.StageProviderUnion || stage == evalcontract.StageParentCollapse || stage == evalcontract.StageRRFFusion || stage == evalcontract.StageBodyPackaging || stage == evalcontract.StageOperational
	case VariantHybridWithoutDense:
		return stage == evalcontract.StageFTSCandidate || stage == evalcontract.StageRRFFusion || stage == evalcontract.StageBodyPackaging || stage == evalcontract.StageOperational
	default:
		return false
	}
}

func directRelevantBytes(body BodyPackageHit, judgments []evalcontract.RelevanceJudgment) int {
	type interval struct{ start, end int }
	intervals := []interval{}
	for _, judgment := range judgments {
		if judgment.Grade == evalcontract.DirectRequirement && bodyContains(body, judgment.Span) {
			intervals = append(intervals, interval{start: judgment.Span.StartByte, end: judgment.Span.EndByte})
		}
	}
	sort.Slice(intervals, func(i, j int) bool {
		if intervals[i].start != intervals[j].start {
			return intervals[i].start < intervals[j].start
		}
		return intervals[i].end < intervals[j].end
	})
	total, start, end := 0, 0, 0
	for index, item := range intervals {
		if index == 0 {
			start, end = item.start, item.end
			continue
		}
		if item.start <= end {
			if item.end > end {
				end = item.end
			}
			continue
		}
		total += end - start
		start, end = item.start, item.end
	}
	if len(intervals) > 0 {
		total += end - start
	}
	return total
}
