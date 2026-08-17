package vector

import (
	"fmt"
	"sort"
	"testing"
)

// This is a deterministic synthetic observation, not a quality gate. Phase 12
// replaces it with corpus-backed f32 fidelity and human-relevance evidence.
func TestSyntheticCodecRankObservation(t *testing.T) {
	query := syntheticVector(1)
	documents := [][]float32{syntheticVector(1), syntheticVector(2), syntheticVector(3), syntheticVector(4)}
	for _, dimensions := range []int{512, 1024} {
		spec := TransformSpec{SourceDimensions: 1024, TargetDimensions: dimensions, ReducerID: ReducerID, NormalizerID: NormalizerID, MetricID: MetricID}
		q, err := ReduceAndNormalize(spec, query)
		if err != nil {
			t.Fatal(err)
		}
		f32 := make([]rankObservation, 0, len(documents))
		for id, source := range documents {
			space, err := ReduceAndNormalize(spec, source)
			if err != nil {
				t.Fatal(err)
			}
			score, err := Cosine(q, space)
			if err != nil {
				t.Fatal(err)
			}
			f32 = append(f32, rankObservation{ID: id, Score: score, Space: space})
		}
		sortRanks(f32)
		for _, codecID := range []string{Int8CodecID} {
			quantized := make([]rankObservation, 0, len(f32))
			for _, item := range f32 {
				stored, err := EncodeInt8(item.Space)
				if err != nil {
					t.Fatal(err)
				}
				var score float64
				score, err = ScoreInt8(q, stored)
				if err != nil {
					t.Fatal(err)
				}
				quantized = append(quantized, rankObservation{ID: item.ID, Score: score})
			}
			sortRanks(quantized)
			repeat := append([]rankObservation(nil), quantized...)
			sortRanks(repeat)
			if fmt.Sprint(quantized) != fmt.Sprint(repeat) {
				t.Fatalf("non-deterministic %s rank", codecID)
			}
			f32ByID := map[int]float64{}
			codecByID := map[int]float64{}
			for _, item := range f32 {
				f32ByID[item.ID] = item.Score
			}
			for _, item := range quantized {
				codecByID[item.ID] = item.Score
			}
			var absoluteError float64
			inversions := 0
			for _, item := range quantized {
				absoluteError += abs(f32ByID[item.ID] - item.Score)
			}
			for left := 0; left < len(documents); left++ {
				for right := left + 1; right < len(documents); right++ {
					if (f32ByID[left]-f32ByID[right])*(codecByID[left]-codecByID[right]) < 0 {
						inversions++
					}
				}
			}
			t.Logf("target=%d codec=%s f32=%v codec_rank=%v mean_abs_score_error=%.9f inversions=%d/%d", dimensions, codecID, rankIDs(f32), rankIDs(quantized), absoluteError/float64(len(documents)), inversions, len(documents)*(len(documents)-1)/2)
		}
	}
}

type rankObservation struct {
	ID    int
	Score float64
	Space []float32
}

func sortRanks(values []rankObservation) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Score == values[j].Score {
			return values[i].ID < values[j].ID
		}
		return values[i].Score > values[j].Score
	})
}
func rankIDs(values []rankObservation) []int {
	out := make([]int, len(values))
	for i := range values {
		out[i] = values[i].ID
	}
	return out
}
func syntheticVector(seed int) []float32 {
	out := make([]float32, 1024)
	for i := range out {
		out[i] = float32(((i+1)*(seed+3))%19 - 9)
		if i%31 == 0 {
			out[i] += float32(seed)
		}
	}
	return out
}
func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
