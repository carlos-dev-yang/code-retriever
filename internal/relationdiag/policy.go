package relationdiag

const (
	ReviewPolicyID       = "relation-calibration-review-stage-ef-v1"
	ReviewSemanticStatus = "NOT_OPENED_NO_FINITE_CELL_MANIFEST"
	// The accepted v2 name means the second byte-identical completion replay;
	// the producer manifest itself remains the v1 Stage-A schema.
	ReviewAcceptedCompletionKind   = "cidx.relation_evidence_completion.stage_a.v1"
	ReviewAcceptedCompletionRepeat = "v2_byte_identical_completion_replay"
)

type ReviewBudgetCell struct {
	Family string `json:"family"`
	Count  int    `json:"count"`
	Bytes  int    `json:"bytes"`
}

// canonicalReviewSeriesContract is compiled evidence, not caller-configurable
// input. The tracked JSON mirrors this value for reviewability; every request
// is checked against this exact immutable three-member binding.
func canonicalReviewSeriesContract() ReviewSeriesContract {
	return ReviewSeriesContract{
		SchemaVersion:  1,
		Kind:           "cidx.relation_calibration.review_series.v1",
		SeriesID:       "relation-calibration-review-series-v1",
		AcceptedRepeat: ReviewAcceptedCompletionRepeat,
		QueryCount:     40,
		Endpoints:      250,
		Hints:          289,
		Closures:       576,
		Members: []ReviewSeriesMember{
			{CorpusID: "go-git-go-git-v5.19.1", DatasetSHA256: "b5ddc298a3a7c8b5a816ce439208667fa566c69e175777d9205a6bf983ef80ba", CompletionArtifactChecksum: "0831205fe994f160aa313510faa20b59560a32881039e96a99b68b0acda82932", QueryCount: 12},
			{CorpusID: "pmndrs-zustand-v5.0.14", DatasetSHA256: "9ddff18f0a0b5d7665e60b322e25f7b11ad41f0201ac4d3c650886974aba4db7", CompletionArtifactChecksum: "33ad57507b998fd5b009d60762e853a58915fa35c85188d25ab9037c107b2659", QueryCount: 12},
			{CorpusID: "usememos-memos-v0.30.0", DatasetSHA256: "4d3a546c9c7fd891c80f10f5be3c2e9a105f3458275d345a93b38529fa47fbd7", CompletionArtifactChecksum: "3126284514d60ac651a38b5bbf2896634fa1b6f734e512a4415b2c82ba5368fa", QueryCount: 16},
		},
	}
}

func exactReviewSeriesContract(value ReviewSeriesContract) bool {
	expected := canonicalReviewSeriesContract()
	if value.SchemaVersion != expected.SchemaVersion || value.Kind != expected.Kind || value.SeriesID != expected.SeriesID || value.AcceptedRepeat != expected.AcceptedRepeat || value.QueryCount != expected.QueryCount || value.Endpoints != expected.Endpoints || value.Hints != expected.Hints || value.Closures != expected.Closures || len(value.Members) != len(expected.Members) {
		return false
	}
	for i := range expected.Members {
		if value.Members[i] != expected.Members[i] {
			return false
		}
	}
	return true
}

func reviewClosureCells() []ReviewBudgetCell {
	result := make([]ReviewBudgetCell, 0, len(closureCountBudgetGrid)*len(closureByteBudgetGrid))
	for _, count := range closureCountBudgetGrid {
		for _, bytes := range closureByteBudgetGrid {
			result = append(result, ReviewBudgetCell{Family: "closure", Count: count, Bytes: bytes})
		}
	}
	return result
}

func reviewHintCells() []ReviewBudgetCell {
	result := make([]ReviewBudgetCell, 0, len(hintCountBudgetGrid)*len(hintByteBudgetGrid))
	for _, count := range hintCountBudgetGrid {
		for _, bytes := range hintByteBudgetGrid {
			result = append(result, ReviewBudgetCell{Family: "hint", Count: count, Bytes: bytes})
		}
	}
	return result
}
