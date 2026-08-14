package embedclient

import (
	"testing"
)

func TestSourceContractFixes1024AndDisablesDirectTargets(t *testing.T) {
	spec := EmbeddingSourceSpec{Provider: ProviderID, Model: Model, SourceDimensions: 1024, OutputDType: OutputDType, DocumentInputType: "document", QueryInputType: "query", AdapterVersion: AdapterVersion}
	if err := spec.Validate(); err != nil {
		t.Fatal(err)
	}
	if spec.AllowDirectTargetCompare {
		t.Fatal("direct targets must stay disabled without explicit approval")
	}
	values := make([]float32, spec.SourceDimensions)
	values[0] = 1
	if err := ValidateSourceVector(spec, values); err != nil {
		t.Fatal(err)
	}
	spec.AllowDirectTargetCompare = true
	if err := spec.Validate(); err == nil {
		t.Fatal("expected direct-target guard")
	}
}
