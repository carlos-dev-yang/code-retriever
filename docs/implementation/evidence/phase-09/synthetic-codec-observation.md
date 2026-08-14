# Synthetic Codec Rank Observation

- Date: 2026-08-15
- Purpose: deterministic implementation smoke evidence only; it is not corpus relevance evidence or a release threshold.
- Source: four synthetic 1024-dimensional document vectors and one synthetic query, using the fixed local prefix-plus-L2 transform.

| Target dimension | Codec | Target-f32 rank | Codec rank | Mean absolute score error | Pair inversions |
| --- | --- | --- | --- | ---: | ---: |
| 256 | binary | `[0, 2, 3, 1]` | `[0, 1, 3, 2]` | 0.083009677 | 3 / 6 |
| 256 | int8 | `[0, 2, 3, 1]` | `[0, 2, 3, 1]` | 0.000257373 | 0 / 6 |
| 512 | binary | `[0, 2, 3, 1]` | `[0, 1, 3, 2]` | 0.081778490 | 2 / 6 |
| 512 | int8 | `[0, 2, 3, 1]` | `[0, 2, 3, 1]` | 0.000696707 | 0 / 6 |

Command: `go test -count=1 -run TestSyntheticCodecRankObservation -v ./internal/vector`.

This observation intentionally demonstrates that the binary scorer is an approximation rather than exact cosine. Phase 12 must provide exhaustive target-f32 and human-relevance evidence on user-selected corpora before any quality decision.
