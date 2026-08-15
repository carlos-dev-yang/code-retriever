package config

import "fmt"

// DefaultRaw returns the complete final v1 file shape. Writing it is owned by
// the later CLI/init phase; this helper has no filesystem side effect.
func DefaultRaw(servingDimensions int, codec string) (RawConfig, error) {
	spec := VoyageCode4()
	if !spec.SupportsServingDimensions(servingDimensions) {
		return RawConfig{}, fmt.Errorf("unsupported serving dimension")
	}
	if codec != StorageCodecBinary && codec != StorageCodecInt8 {
		return RawConfig{}, fmt.Errorf("unsupported storage codec")
	}
	waits := defaultRetryWaitSchedule()
	return RawConfig{
		Version: SchemaVersion,
		Index:   RawIndex{Languages: []string{"go", "typescript", "tsx"}, MaxSourceFileBytes: DefaultMaxSourceFileBytes, TargetSegmentBytes: DefaultTargetSegmentBytes},
		Embedding: RawEmbedding{
			Model:             spec.Model,
			ServingDimensions: &servingDimensions,
			Reducer:           "prefix-l2-v1",
			Normalizer:        "l2-v1",
			Metric:            "cosine",
			StorageCodec:      &codec,
			Request:           RawRequest{MaxInputs: DefaultRequestMaxInputs, MaxTotalInputBytes: DefaultRequestMaxTotalInputBytes, MaxConcurrency: DefaultRequestMaxConcurrency, TimeoutSeconds: DefaultRequestTimeoutSeconds},
			Retry:             RawRetry{MaxRetries: intPointer(DefaultRetryMaxRetries), WaitSeconds: &waits},
		},
		Search: RawSearch{
			DefaultMode:             stringPointer(DefaultSearchMode),
			AllowPaidQueryEmbedding: boolPointer(false),
			ReturnK:                 intPointer(DefaultReturnK),
			CandidateK:              intPointer(DefaultCandidateK),
			RRFK:                    intPointer(DefaultRRFK),
			MaxQueryBytes:           intPointer(DefaultMaxQueryBytes),
			MaxQueryTokens:          intPointer(DefaultMaxQueryTokens),
			MaxQueryTokenRunes:      intPointer(DefaultMaxQueryTokenRunes),
			FTSWeights:              RawFTSWeights{Symbols: floatPointer(DefaultFTSSymbolWeight), Body: floatPointer(DefaultFTSBodyWeight)},
		},
		MCP: RawMCP{HardMaxInlineBytes: DefaultHardMaxInlineBytes},
	}, nil
}

func intPointer(value int) *int           { return &value }
func boolPointer(value bool) *bool        { return &value }
func stringPointer(value string) *string  { return &value }
func floatPointer(value float64) *float64 { return &value }
