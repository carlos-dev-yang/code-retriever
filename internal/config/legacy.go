package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// LegacyConfigError identifies a pre-Revision-4 shape before any caller can
// open a database. The mappings are migration guidance, never accepted aliases.
type LegacyConfigError struct {
	Mappings []LegacyFieldMapping
}

type LegacyFieldMapping struct {
	RemovedField string
	FinalPolicy  string
}

func (err *LegacyConfigError) Error() string {
	if len(err.Mappings) == 0 {
		return "legacy pre-R4 config shape"
	}
	parts := make([]string, len(err.Mappings))
	for index, mapping := range err.Mappings {
		parts[index] = mapping.RemovedField + " -> " + mapping.FinalPolicy
	}
	return fmt.Sprintf("legacy pre-R4 config fields require explicit migration: %s", strings.Join(parts, "; "))
}

func detectLegacyFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	root, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	mappings := make([]LegacyFieldMapping, 0, 6)
	if objectHas(root, "index", "max_chunk_bytes") {
		mappings = append(mappings, LegacyFieldMapping{"index.max_chunk_bytes", "removed; semantic parents have no configurable byte cap"})
	}
	if objectHas(root, "index", "max_segment_input_bytes") {
		mappings = append(mappings, LegacyFieldMapping{"index.max_segment_input_bytes", "index.target_segment_bytes"})
	}
	if objectHas(root, "embedding", "target_dimensions") {
		mappings = append(mappings, LegacyFieldMapping{"embedding.target_dimensions", "embedding.serving_dimensions"})
	}
	if embedding, ok := root["embedding"].(map[string]any); ok {
		if batchValue, exists := embedding["batch"]; exists {
			batch, object := batchValue.(map[string]any)
			if object {
				mapped := false
				if _, ok := batch["max_inputs"]; ok {
					mappings = append(mappings, LegacyFieldMapping{"embedding.batch.max_inputs", "embedding.request.max_inputs"})
					mapped = true
				}
				if _, ok := batch["max_input_tokens"]; ok {
					mappings = append(mappings, LegacyFieldMapping{"embedding.batch.max_input_tokens", "embedding.request.max_total_input_bytes; explicitly choose bytes, never convert token estimates"})
					mapped = true
				}
				if _, ok := batch["max_retries"]; ok {
					mappings = append(mappings, LegacyFieldMapping{"embedding.batch.max_retries", "embedding.retry.max_retries"})
					mapped = true
				}
				if _, ok := batch["request_timeout_ms"]; ok {
					mappings = append(mappings, LegacyFieldMapping{"embedding.batch.request_timeout_ms", "embedding.request.timeout_seconds; explicitly choose seconds"})
					mapped = true
				}
				if !mapped {
					mappings = append(mappings, LegacyFieldMapping{"embedding.batch", "embedding.request and embedding.retry"})
				}
			} else {
				mappings = append(mappings, LegacyFieldMapping{"embedding.batch", "embedding.request and embedding.retry"})
			}
		}
	}
	if objectHas(root, "mcp", "max_read_span_lines") {
		mappings = append(mappings, LegacyFieldMapping{"mcp.max_read_span_lines", "removed; read_span is byte-bounded without a line cap"})
	}
	if len(mappings) == 0 {
		return nil
	}
	sort.Slice(mappings, func(i, j int) bool { return mappings[i].RemovedField < mappings[j].RemovedField })
	return &LegacyConfigError{Mappings: mappings}
}

func objectHas(root map[string]any, object, field string) bool {
	value, ok := root[object]
	if !ok {
		return false
	}
	fields, ok := value.(map[string]any)
	if !ok {
		return false
	}
	_, ok = fields[field]
	return ok
}
