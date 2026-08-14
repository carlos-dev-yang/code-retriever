// Package embedclient owns provider response validation. It intentionally has
// no persistence dependency and does not perform calls in the runtime spike.
package embedclient

import (
	"fmt"

	"cidx/internal/vector"
)

const (
	ProviderID             = "voyage-official"
	Model                  = "voyage-code-4"
	Endpoint               = "https://api.voyageai.com/v1/embeddings"
	APIKeyEnvironment      = "VOYAGE_API_KEY"
	OutputDType            = "float"
	AdapterVersion         = 1
	DirectTargetComparison = false
)

type EmbeddingSourceSpec struct {
	Provider                 string
	Model                    string
	SourceDimensions         int
	OutputDType              string
	DocumentInputType        string
	QueryInputType           string
	Truncation               bool
	AdapterVersion           int
	AllowDirectTargetCompare bool
}

func DefaultEmbeddingSourceSpec() EmbeddingSourceSpec {
	return EmbeddingSourceSpec{
		Provider: ProviderID, Model: Model, SourceDimensions: vector.SourceDimensions,
		OutputDType: OutputDType, DocumentInputType: "document", QueryInputType: "query",
		Truncation: false, AdapterVersion: AdapterVersion, AllowDirectTargetCompare: DirectTargetComparison,
	}
}

func (s EmbeddingSourceSpec) Validate() error {
	if s.Provider != ProviderID || s.Model != Model || s.SourceDimensions != vector.SourceDimensions || s.OutputDType != OutputDType || s.DocumentInputType != "document" || s.QueryInputType != "query" || s.Truncation || s.AdapterVersion != AdapterVersion {
		return fmt.Errorf("unsupported voyage source specification")
	}
	if s.AllowDirectTargetCompare {
		return fmt.Errorf("direct target comparison is disabled pending explicit approval")
	}
	return nil
}

// ValidateSourceVector enforces the source-1024 provider contract. Query data
// stays caller-owned and is never accepted by a storage API.
func ValidateSourceVector(spec EmbeddingSourceSpec, values []float32) error {
	if err := spec.Validate(); err != nil {
		return err
	}
	if err := vector.ValidateF32(values, spec.SourceDimensions); err != nil {
		return err
	}
	_, err := vector.Cosine(values, values)
	return err
}
