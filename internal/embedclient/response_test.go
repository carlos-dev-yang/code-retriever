package embedclient

import (
	"testing"

	"cidx/internal/vector"
)

func TestSourceContractFixes1024AndDisablesDirectTargets(t *testing.T) {
	spec := DefaultEmbeddingSourceSpec()
	if err := spec.Validate(); err != nil {
		t.Fatal(err)
	}
	if spec.AllowDirectTargetCompare {
		t.Fatal("direct targets must stay disabled without explicit approval")
	}
	values := make([]float32, vector.SourceDimensions)
	values[0] = 1
	if err := ValidateSourceVector(spec, values); err != nil {
		t.Fatal(err)
	}
	spec.AllowDirectTargetCompare = true
	if err := spec.Validate(); err == nil {
		t.Fatal("expected direct-target guard")
	}
}
