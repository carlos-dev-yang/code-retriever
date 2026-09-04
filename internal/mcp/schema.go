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
		{Name: "status", Description: "Diagnose local index freshness or generation state when that information matters; ordinary source lookup does not require calling this first. Returns index metadata and stale, unindexed, and deleted file counts, with no source code.", InputSchema: empty},
		{Name: "search", Description: "Locate ranked functions, methods, and types from a natural-language, identifier, or path query. Returns locators without source text; inspect selected locators with read_span. Omit k to use the recommended default of 5; valid values are 1 through 20. Top-k results do not prove repository-wide absence.", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"query", "max_inline_bytes"}, "properties": map[string]any{"query": map[string]any{"type": "string", "minLength": 1, "description": "Natural-language behavior, exact identifier, or path-oriented repository query."}, "k": map[string]any{"type": "integer", "minimum": 1, "maximum": config.AbsoluteMaxReturnK, "description": "Maximum ranked locators to return; omit to use the configured default."}, "mode": map[string]any{"type": "string", "enum": []string{"fts", "hybrid"}, "description": "Search mode; omit to use the configured default. FTS is provider-free."}, "max_inline_bytes": map[string]any{"type": "integer", "minimum": 0, "description": "Retained v1 compatibility input; search always returns zero source bytes."}}}},
		{Name: "read_span", Description: "Return the complete current source for one selected locator's exact path, range, and indexed hash.", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"path", "start_line", "end_line", "expected_sha256"}, "properties": map[string]any{"path": map[string]any{"type": "string", "minLength": 1, "description": "Repository-relative path from a search locator."}, "start_line": map[string]any{"type": "integer", "minimum": 1, "description": "Inclusive start line from the selected locator."}, "end_line": map[string]any{"type": "integer", "minimum": 1, "description": "Inclusive end line from the selected locator."}, "expected_sha256": map[string]any{"type": "string", "pattern": "^[0-9a-f]{64}$", "description": "The selected locator's indexed_sha256; prevents reading stale coordinates."}}}},
		{Name: "reindex", Description: "Refresh local AST and FTS index state after repository files change or freshness diagnostics report drift; it is unnecessary for ordinary lookup and never calls an embedding provider. With dry_run=true, returns planned changes without activating a generation; otherwise returns applied counts and active generation metadata.", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"dry_run": map[string]any{"type": "boolean", "description": "Preview local index changes without committing a new generation."}}}},
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
