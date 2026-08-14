package eval

import (
	"cidx/internal/evalcontract"
	"cidx/internal/search/lexical"
	"fmt"
	"math"
	"sort"
)

type CaseResult struct {
	QueryID                  string                    `json:"query_id"`
	Language                 evalcontract.Language     `json:"language"`
	Cohorts                  []string                  `json:"cohorts"`
	Answerable               bool                      `json:"answerable"`
	HasReviewedHardNegatives bool                      `json:"has_reviewed_hard_negatives"`
	ReturnedCount            int                       `json:"returned_count"`
	FailureStage             evalcontract.FailureStage `json:"failure_stage,omitempty"`
	FirstLoss                evalcontract.FirstLoss    `json:"first_loss"`
	HitAt                    map[int]bool              `json:"hit_at"`
	RecallAt                 map[int]float64           `json:"recall_at"`
	MRRAt                    map[int]float64           `json:"mrr_at"`
	NDCGAt                   map[int]float64           `json:"ndcg_at"`
	RequirementCoverageAt    map[int]float64           `json:"requirement_coverage_at"`
	CompleteRequirementHitAt map[int]bool              `json:"complete_requirement_hit_at"`
	KnownHardNegativeHitAt   map[int]bool              `json:"known_hard_negative_hit_at"`
}
type Denominators struct {
	RequiredQueries     int `json:"required_queries"`
	AnswerableQueries   int `json:"answerable_queries"`
	HardNegativeQueries int `json:"hard_negative_queries"`
}
type Summary struct {
	Denominators             Denominators                      `json:"denominators"`
	Cases                    int                               `json:"cases"`
	Failures                 int                               `json:"failures"`
	HitAt                    map[int]float64                   `json:"hit_at"`
	RecallAt                 map[int]float64                   `json:"recall_at"`
	MRRAt                    map[int]float64                   `json:"mrr_at"`
	NDCGAt                   map[int]float64                   `json:"ndcg_at"`
	RequirementCoverageAt    map[int]float64                   `json:"requirement_coverage_at"`
	CompleteRequirementHitAt map[int]float64                   `json:"complete_requirement_hit_at"`
	KnownHardNegativeHitAt   map[int]float64                   `json:"known_hard_negative_hit_at"`
	ReturnedCountMean        float64                           `json:"returned_count_mean"`
	FirstLossCounts          map[evalcontract.FirstLoss]int    `json:"first_loss_counts"`
	ByLanguage               map[evalcontract.Language]Summary `json:"by_language"`
	ByCohort                 map[string]Summary                `json:"by_cohort"`
}

func EvaluateCase(c evalcontract.EvaluationCase, h []lexical.Hit, ks []int, f error) (CaseResult, error) {
	if e := c.Validate(); e != nil {
		return CaseResult{}, e
	}
	ks = norm(ks)
	if len(ks) == 0 {
		return CaseResult{}, fmt.Errorf("k required")
	}
	o := CaseResult{QueryID: c.ID, Language: c.Language, Cohorts: append([]string{}, c.Cohorts...), Answerable: c.AnswerMode != evalcontract.Abstainable, HasReviewedHardNegatives: len(c.HardNegatives) > 0, ReturnedCount: len(h), HitAt: map[int]bool{}, RecallAt: map[int]float64{}, MRRAt: map[int]float64{}, NDCGAt: map[int]float64{}, RequirementCoverageAt: map[int]float64{}, CompleteRequirementHitAt: map[int]bool{}, KnownHardNegativeHitAt: map[int]bool{}}
	if f != nil {
		for _, k := range ks {
			o.HitAt[k], o.RecallAt[k], o.MRRAt[k], o.NDCGAt[k], o.RequirementCoverageAt[k], o.CompleteRequirementHitAt[k], o.KnownHardNegativeHitAt[k] = false, 0, 0, 0, 0, false, false
		}
		o.ReturnedCount = 0
		o.FailureStage = evalcontract.FailureStage(evalcontract.StageFTSCandidate)
		o.FirstLoss = "OPERATION_FAILURE:fts_candidate"
		return o, nil
	}
	for _, k := range ks {
		p := h
		if len(p) > k {
			p = p[:k]
		}
		d := direct(c, p)
		tot := directTotal(c)
		o.HitAt[k] = len(d) > 0
		if tot > 0 {
			o.RecallAt[k] = float64(len(d)) / float64(tot)
		}
		o.MRRAt[k] = mrr(c, p)
		o.NDCGAt[k] = ndcg(c, p, k)
		g := groups(c.RequiredGroups, p)
		if len(c.RequiredGroups) > 0 {
			o.RequirementCoverageAt[k] = float64(len(g)) / float64(len(c.RequiredGroups))
			o.CompleteRequirementHitAt[k] = len(g) == len(c.RequiredGroups)
		}
		o.KnownHardNegativeHitAt[k] = hard(c, p, k)
	}
	if c.AnswerMode == evalcontract.Abstainable || o.CompleteRequirementHitAt[ks[len(ks)-1]] {
		o.FirstLoss = evalcontract.NoLoss
	} else {
		o.FirstLoss = evalcontract.FTSCandidateMiss
	}
	return o, nil
}
func Summarize(v []CaseResult, ks []int) Summary {
	ks = norm(ks)
	o := sum(v, ks)
	o.ByLanguage = map[evalcontract.Language]Summary{}
	o.ByCohort = map[string]Summary{}
	for _, l := range []evalcontract.Language{evalcontract.Go, evalcontract.TypeScript, evalcontract.TSX, evalcontract.Mixed} {
		var x []CaseResult
		for _, r := range v {
			if r.Language == l {
				x = append(x, r)
			}
		}
		if len(x) > 0 {
			o.ByLanguage[l] = sum(x, ks)
		}
	}
	m := map[string][]CaseResult{}
	for _, r := range v {
		for _, c := range r.Cohorts {
			m[c] = append(m[c], r)
		}
	}
	for c, x := range m {
		o.ByCohort[c] = sum(x, ks)
	}
	return o
}
func sum(v []CaseResult, ks []int) Summary {
	o := Summary{Cases: len(v), HitAt: map[int]float64{}, RecallAt: map[int]float64{}, MRRAt: map[int]float64{}, NDCGAt: map[int]float64{}, RequirementCoverageAt: map[int]float64{}, CompleteRequirementHitAt: map[int]float64{}, KnownHardNegativeHitAt: map[int]float64{}, FirstLossCounts: map[evalcontract.FirstLoss]int{}}
	for _, r := range v {
		o.Denominators.RequiredQueries++
		o.ReturnedCountMean += float64(r.ReturnedCount)
		o.FirstLossCounts[r.FirstLoss]++
		if r.FailureStage != "" {
			o.Failures++
		}
		if r.Answerable {
			o.Denominators.AnswerableQueries++
			for _, k := range ks {
				if r.HitAt[k] {
					o.HitAt[k]++
				}
				o.RecallAt[k] += r.RecallAt[k]
				o.MRRAt[k] += r.MRRAt[k]
				o.NDCGAt[k] += r.NDCGAt[k]
				o.RequirementCoverageAt[k] += r.RequirementCoverageAt[k]
				if r.CompleteRequirementHitAt[k] {
					o.CompleteRequirementHitAt[k]++
				}
			}
		}
		if r.HasReviewedHardNegatives {
			o.Denominators.HardNegativeQueries++
			for _, k := range ks {
				if r.KnownHardNegativeHitAt[k] {
					o.KnownHardNegativeHitAt[k]++
				}
			}
		}
	}
	if o.Cases > 0 {
		o.ReturnedCountMean /= float64(o.Cases)
	}
	for _, k := range ks {
		if n := o.Denominators.AnswerableQueries; n > 0 {
			o.HitAt[k] /= float64(n)
			o.RecallAt[k] /= float64(n)
			o.MRRAt[k] /= float64(n)
			o.NDCGAt[k] /= float64(n)
			o.RequirementCoverageAt[k] /= float64(n)
			o.CompleteRequirementHitAt[k] /= float64(n)
		}
		if n := o.Denominators.HardNegativeQueries; n > 0 {
			o.KnownHardNegativeHitAt[k] /= float64(n)
		}
	}
	return o
}
func match(h []lexical.Hit, s evalcontract.SourceSpan) bool {
	for _, x := range h {
		if x.Path == s.Path && x.IndexedSHA256 == s.ContentSHA256 && x.QualifiedSymbol == s.QualifiedSymbol && x.StartByte <= s.StartByte && x.EndByte >= s.EndByte {
			return true
		}
	}
	return false
}
func groups(gs []evalcontract.RequiredGroup, h []lexical.Hit) map[string]bool {
	o := map[string]bool{}
	for _, g := range gs {
		for _, a := range g.Alternatives {
			ok := true
			for _, s := range a.Spans {
				if !match(h, s) {
					ok = false
				}
			}
			if ok {
				o[g.ID] = true
				break
			}
		}
	}
	return o
}
func pid(s evalcontract.SourceSpan) string {
	return s.Path + "\x00" + s.ContentSHA256 + "\x00" + s.QualifiedSymbol
}
func direct(c evalcontract.EvaluationCase, h []lexical.Hit) map[string]bool {
	o := map[string]bool{}
	for _, j := range c.Judgments {
		if j.Grade == 2 && match(h, j.Span) {
			o[pid(j.Span)] = true
		}
	}
	return o
}
func directTotal(c evalcontract.EvaluationCase) int { return len(direct(c, allJudgmentHits(c))) }
func allJudgmentHits(c evalcontract.EvaluationCase) []lexical.Hit {
	var h []lexical.Hit
	for _, j := range c.Judgments {
		h = append(h, lexical.Hit{Path: j.Span.Path, IndexedSHA256: j.Span.ContentSHA256, QualifiedSymbol: j.Span.QualifiedSymbol, StartByte: j.Span.StartByte, EndByte: j.Span.EndByte})
	}
	return h
}
func mrr(c evalcontract.EvaluationCase, h []lexical.Hit) float64 {
	for i, x := range h {
		if len(direct(c, []lexical.Hit{x})) > 0 {
			return 1 / float64(i+1)
		}
	}
	return 0
}
func hard(c evalcontract.EvaluationCase, h []lexical.Hit, k int) bool {
	if len(h) > k {
		h = h[:k]
	}
	for _, n := range c.HardNegatives {
		if match(h, n.Span) {
			return true
		}
	}
	return false
}
func ndcg(c evalcontract.EvaluationCase, h []lexical.Hit, k int) float64 {
	if len(h) > k {
		h = h[:k]
	}
	m := map[string]evalcontract.Relevance{}
	for _, j := range c.Judgments {
		if j.Grade > m[pid(j.Span)] {
			m[pid(j.Span)] = j.Grade
		}
	}
	used := map[string]bool{}
	d := 0.
	for i, x := range h {
		best := evalcontract.Relevance(0)
		p := ""
		for _, j := range c.Judgments {
			if match([]lexical.Hit{x}, j.Span) && j.Grade > best {
				best = j.Grade
				p = pid(j.Span)
			}
		}
		if p != "" && !used[p] {
			used[p] = true
			d += (math.Pow(2, float64(best)) - 1) / math.Log2(float64(i)+2)
		}
	}
	var ideal []int
	for _, g := range m {
		ideal = append(ideal, int(g))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ideal)))
	z := 0.
	for i, g := range ideal {
		if i >= k {
			break
		}
		z += (math.Pow(2, float64(g)) - 1) / math.Log2(float64(i)+2)
	}
	if z == 0 {
		return 0
	}
	return d / z
}
func norm(v []int) []int {
	m := map[int]bool{}
	var o []int
	for _, k := range v {
		if k > 0 && !m[k] {
			m[k] = true
			o = append(o, k)
		}
	}
	sort.Ints(o)
	return o
}
