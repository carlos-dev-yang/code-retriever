package config

import (
	"fmt"

	"cidx/internal/embedclient"
)

type ModelSpec struct {
	Provider                string
	Model                   string
	SourceDimensions        int
	AllowedTargetDimensions []int
	OutputDType             string
	DocumentInputType       string
	QueryInputType          string
	Truncation              bool
	AdapterVersion          int
}

func VoyageCode4() ModelSpec {
	return ModelSpec{
		Provider: embedclient.ProviderID, Model: embedclient.Model,
		// The model registry is the sole runtime authority for source and
		// allowed serving dimensions. Vector algorithms receive a concrete spec.
		SourceDimensions: 1024, AllowedTargetDimensions: []int{256, 512, 1024},
		OutputDType: embedclient.OutputDType, DocumentInputType: "document", QueryInputType: "query",
		Truncation: false, AdapterVersion: embedclient.AdapterVersion,
	}
}

func ResolveModel(model string) (ModelSpec, error) {
	spec := VoyageCode4()
	if model == "" || model == spec.Model {
		return spec, nil
	}
	return ModelSpec{}, fmt.Errorf("unsupported embedding model %q", model)
}

func (spec ModelSpec) SupportsTarget(dimensions int) bool {
	for _, allowed := range spec.AllowedTargetDimensions {
		if dimensions == allowed {
			return true
		}
	}
	return false
}
