package mcp

import (
	"cidx/internal/config"
	"encoding/json"
)

type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

func toolRegistry() []ToolDefinition {
	empty := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{}}
	return []ToolDefinition{
		{Name: "status", Description: "Inspect the local index without returning source bodies.", InputSchema: empty},
		{Name: "search", Description: "Search indexed code. Hybrid mode may send the query to Voyage AI only when configured.", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"query", "max_inline_bytes"}, "properties": map[string]any{"query": map[string]any{"type": "string", "minLength": 1}, "k": map[string]any{"type": "integer", "minimum": 1, "maximum": config.AbsoluteMaxReturnK}, "mode": map[string]any{"type": "string", "enum": []string{"fts", "hybrid"}}, "max_inline_bytes": map[string]any{"type": "integer", "minimum": 0}}}},
		{Name: "read_span", Description: "Read a current, hash-guarded live source range.", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"path", "start_line", "end_line", "expected_sha256"}, "properties": map[string]any{"path": map[string]any{"type": "string", "minLength": 1}, "start_line": map[string]any{"type": "integer", "minimum": 1}, "end_line": map[string]any{"type": "integer", "minimum": 1}, "expected_sha256": map[string]any{"type": "string", "pattern": "^[0-9a-f]{64}$"}}}},
		{Name: "reindex", Description: "Refresh local AST and FTS state; never calls an embedding provider.", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"dry_run": map[string]any{"type": "boolean"}}}},
	}
}

func decodeObject(raw json.RawMessage, allowed ...string) (map[string]json.RawMessage, *Error) {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]json.RawMessage{}, nil
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, &Error{Code: invalidParams, Message: "INVALID_PARAMS"}
	}
	known := map[string]bool{}
	for _, key := range allowed {
		known[key] = true
	}
	for key := range value {
		if !known[key] {
			return nil, &Error{Code: invalidParams, Message: "UNKNOWN_FIELD", Data: map[string]string{"field": key}}
		}
	}
	if meta, exists := value["_meta"]; exists {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(meta, &object); err != nil || object == nil {
			return nil, &Error{Code: invalidParams, Message: "INVALID_META"}
		}
	}
	return value, nil
}
