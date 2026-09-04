package mcp

import (
	"bytes"
	"cidx/internal/app"
	"cidx/internal/config"
	"cidx/internal/index"
	"cidx/internal/search"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"strings"
)

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type callToolResult struct {
	Content           []toolContent `json:"content"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError,omitempty"`
}

type ResultRepresentation string

const (
	ResultRepresentationDefault    ResultRepresentation = ResultRepresentationStructured
	ResultRepresentationDual       ResultRepresentation = "dual"
	ResultRepresentationText       ResultRepresentation = "text"
	ResultRepresentationStructured ResultRepresentation = "structured"
)

type Services interface {
	Status(context.Context) (app.StatusResponse, error)
	Search(context.Context, app.SearchRequest) (any, error)
	ReadSpan(context.Context, app.ReadSpanRequest) (app.ReadSpanResponse, error)
	ReadSpanBatch(context.Context, app.ReadSpanBatchRequest) (app.ReadSpanBatchResponse, error)
	Reindex(context.Context, bool) (any, error)
}
type ApplicationServices struct{ Application *app.Application }
type reindexOutcome struct {
	Result index.Result
	Status app.StatusResponse
}

func (value ApplicationServices) Status(ctx context.Context) (app.StatusResponse, error) {
	return value.Application.Status.Get(ctx)
}
func (value ApplicationServices) Search(ctx context.Context, request app.SearchRequest) (any, error) {
	return value.Application.SearchCommand(ctx, request)
}
func (value ApplicationServices) ReadSpan(ctx context.Context, request app.ReadSpanRequest) (app.ReadSpanResponse, error) {
	return value.Application.ReadSpan.Read(ctx, request)
}
func (value ApplicationServices) ReadSpanBatch(ctx context.Context, request app.ReadSpanBatchRequest) (app.ReadSpanBatchResponse, error) {
	return value.Application.ReadSpan.ReadBatch(ctx, request)
}
func (value ApplicationServices) Reindex(ctx context.Context, dry bool) (any, error) {
	result, err := value.Application.Reindex(ctx, dry, index.ReasonMCP)
	if err != nil {
		return result, err
	}
	status, err := value.Application.Status.Get(ctx)
	if err != nil {
		return result, err
	}
	return reindexOutcome{Result: result, Status: status}, nil
}

func callTool(ctx context.Context, services Services, raw json.RawMessage) (any, *Error) {
	return callToolWithContract(ctx, services, raw, ResultRepresentationDefault, ReadSpanContractScalarV1)
}

func callToolWithRepresentation(ctx context.Context, services Services, raw json.RawMessage, representation ResultRepresentation) (any, *Error) {
	return callToolWithContract(ctx, services, raw, representation, ReadSpanContractScalarV1)
}

func callToolWithContract(ctx context.Context, services Services, raw json.RawMessage, representation ResultRepresentation, contract ReadSpanContract) (any, *Error) {
	call, failure := decodeObject(raw, "name", "arguments", "_meta")
	if failure != nil {
		return nil, failure
	}
	var name string
	if input, exists := call["name"]; !exists || json.Unmarshal(input, &name) != nil || name == "" {
		return nil, &Error{Code: invalidParams, Message: "INVALID_TOOL_NAME"}
	}
	args := call["arguments"]
	switch name {
	case "status":
		if _, err := decodeObject(args); err != nil {
			return nil, err
		}
		result, err := services.Status(ctx)
		return toolOutcome(wireStatus(result), applicationError(err), representation), nil
	case "search":
		value, err := decodeObject(args, "query", "k", "mode", "max_inline_bytes")
		if err != nil {
			return nil, err
		}
		var request app.SearchRequest
		if !requiredString(value, "query", &request.Query) || !requiredInteger(value, "max_inline_bytes", &request.MaxInline) || request.MaxInline < 0 {
			return nil, &Error{Code: invalidParams, Message: "INVALID_SEARCH_REQUEST"}
		}
		if input, ok := value["k"]; ok {
			if json.Unmarshal(input, &request.K) != nil || request.K < 1 || request.K > config.AbsoluteMaxReturnK {
				return nil, &Error{Code: invalidParams, Message: "INVALID_RESULT_LIMIT"}
			}
		}
		if input, ok := value["mode"]; ok {
			if json.Unmarshal(input, &request.Mode) != nil || (request.Mode != "fts" && request.Mode != "hybrid") {
				return nil, &Error{Code: invalidParams, Message: "INVALID_SEARCH_MODE"}
			}
		}
		// The retained v1 max_inline_bytes field is validated above for wire
		// compatibility. MCP search is locator-only and must never ask the shared
		// search service to package source bodies.
		request.MaxInline = 0
		result, callErr := services.Search(ctx, request)
		return toolOutcome(wireSearch(result), applicationError(callErr), representation), nil
	case "read_span":
		return callReadSpan(ctx, services, args, representation, contract)
	case "reindex":
		value, err := decodeObject(args, "dry_run")
		if err != nil {
			return nil, err
		}
		var dry bool
		if !optionalBoolean(value, "dry_run", &dry) {
			return nil, &Error{Code: invalidParams, Message: "INVALID_DRY_RUN"}
		}
		result, callErr := services.Reindex(ctx, dry)
		return toolOutcome(wireReindex(result), applicationError(callErr), representation), nil
	default:
		return nil, &Error{Code: invalidParams, Message: "UNKNOWN_TOOL"}
	}
}

func callReadSpan(ctx context.Context, services Services, raw json.RawMessage, representation ResultRepresentation, contract ReadSpanContract) (any, *Error) {
	allowed := []string{"path", "start_line", "end_line", "expected_sha256"}
	if contract == ReadSpanContractBatchV2 {
		allowed = append(allowed, "input_version", "locators")
	}
	value, err := decodeObject(raw, allowed...)
	if err != nil {
		return nil, err
	}
	_, hasVersion := value["input_version"]
	_, hasLocators := value["locators"]
	if hasVersion || hasLocators {
		if contract != ReadSpanContractBatchV2 {
			return nil, &Error{Code: invalidParams, Message: "INVALID_READ_SPAN_REQUEST"}
		}
		for _, field := range []string{"path", "start_line", "end_line", "expected_sha256"} {
			if _, exists := value[field]; exists {
				return nil, &Error{Code: invalidParams, Message: "INVALID_READ_SPAN_REQUEST"}
			}
		}
		var inputVersion int
		if !hasVersion || !hasLocators || !requiredInteger(value, "input_version", &inputVersion) || inputVersion != 2 {
			return nil, &Error{Code: invalidParams, Message: "INVALID_READ_SPAN_REQUEST"}
		}
		locators, failure := decodeReadSpanLocators(value["locators"])
		if failure != nil {
			return nil, failure
		}
		result, callErr := services.ReadSpanBatch(ctx, app.ReadSpanBatchRequest{Locators: locators})
		return toolOutcome(wireReadSpanBatch(result), applicationError(callErr), representation), nil
	}
	request, failure := decodeReadSpanRequest(value)
	if failure != nil {
		return nil, failure
	}
	result, callErr := services.ReadSpan(ctx, request)
	return toolOutcome(wireReadSpan(result), applicationError(callErr), representation), nil
}

func decodeReadSpanLocators(raw json.RawMessage) ([]app.ReadSpanRequest, *Error) {
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return nil, &Error{Code: invalidParams, Message: "INVALID_READ_SPAN_REQUEST"}
	}
	locators := make([]app.ReadSpanRequest, 0, len(values))
	for locatorIndex, rawLocator := range values {
		value, err := decodeObject(rawLocator, "path", "start_line", "end_line", "expected_sha256")
		if err != nil {
			return nil, readSpanLocatorError(err, locatorIndex)
		}
		locator, failure := decodeReadSpanRequest(value)
		if failure != nil {
			return nil, readSpanLocatorError(failure, locatorIndex)
		}
		locators = append(locators, locator)
	}
	return locators, nil
}

func readSpanLocatorError(failure *Error, locatorIndex int) *Error {
	data := map[string]any{"locator_index": locatorIndex}
	if existing, ok := failure.Data.(map[string]string); ok {
		for key, value := range existing {
			data[key] = value
		}
	}
	return &Error{Code: failure.Code, Message: failure.Message, Data: data}
}

func decodeReadSpanRequest(value map[string]json.RawMessage) (app.ReadSpanRequest, *Error) {
	var request app.ReadSpanRequest
	if !requiredString(value, "path", &request.Path) || !requiredLowerSHA256(value, "expected_sha256", &request.ExpectedSHA256) || !requiredInteger(value, "start_line", &request.StartLine) || !requiredInteger(value, "end_line", &request.EndLine) || request.StartLine < 1 || request.EndLine < request.StartLine {
		return app.ReadSpanRequest{}, &Error{Code: invalidParams, Message: "INVALID_READ_SPAN_REQUEST"}
	}
	return request, nil
}

func toolOutcome(value any, failure *Error, representation ResultRepresentation) callToolResult {
	if failure != nil {
		payload := map[string]any{"code": failure.Message}
		if data, ok := failure.Data.(map[string]any); ok {
			for key, value := range data {
				payload[key] = value
			}
		} else if failure.Data != nil {
			payload["data"] = failure.Data
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return representedToolResult("APPLICATION_ERROR", nil, true, representation)
		}
		return representedToolResult(string(encoded), payload, true, representation)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return representedToolResult("APPLICATION_ERROR", nil, true, representation)
	}
	return representedToolResult(string(encoded), value, false, representation)
}

func representedToolResult(text string, structured any, isError bool, representation ResultRepresentation) callToolResult {
	switch representation {
	case ResultRepresentationText:
		return callToolResult{Content: []toolContent{{Type: "text", Text: text}}, IsError: isError}
	case ResultRepresentationDual:
		return callToolResult{Content: []toolContent{{Type: "text", Text: text}}, StructuredContent: structured, IsError: isError}
	default:
		return callToolResult{Content: []toolContent{}, StructuredContent: structured, IsError: isError}
	}
}

func wireStatus(value app.StatusResponse) map[string]any {
	return map[string]any{"observed_generation": value.ObservedGeneration, "manifest_sha256": value.ManifestSHA256, "desired_index_profile": value.Desired.Fingerprints.Index, "desired_canonical_text_profile": value.Desired.Fingerprints.CanonicalText, "desired_source_profile": value.Desired.Fingerprints.Source, "desired_vector_space_profile": value.Desired.Fingerprints.VectorSpace, "desired_vector_storage_profile": value.Desired.Fingerprints.VectorStorage, "desired_active_serving_profile": value.Desired.ActiveServingProfile, "applied_index_profile": value.Applied.Fingerprints.Index, "applied_canonical_text_profile": value.Applied.Fingerprints.CanonicalText, "applied_source_profile": value.Applied.Fingerprints.Source, "applied_vector_space_profile": value.Applied.Fingerprints.VectorSpace, "applied_vector_storage_profile": value.Applied.Fingerprints.VectorStorage, "applied_active_serving_profile": value.Applied.ActiveServingProfile, "file_count": value.Files, "chunk_count": value.Chunks, "segment_count": value.Segments, "dirty": value.Dirty, "stale_count": value.Stale, "unindexed_count": value.Unindexed, "deleted_count": value.Deleted, "index_error_count": value.IndexError, "vector_coverage_numerator": value.CoverageReady, "vector_coverage_denominator": value.CoverageTotal, "ready_count": value.Ready, "pending_count": value.Pending, "failed_count": value.Failed, "index_attempted_at": value.IndexAttemptedAt, "index_succeeded_at": value.IndexSucceededAt, "embed_attempted_at": value.EmbedAttemptedAt, "embed_succeeded_at": value.EmbedSucceededAt, "generation_changed_during_status": value.GenerationChangedDuringStatus}
}
func wireReindex(value any) any {
	if outcome, ok := value.(reindexOutcome); ok {
		result := outcome.Result
		if result.DryRun {
			return map[string]any{"dry_run": true, "planned_files_updated": result.Updated, "planned_files_deleted": result.Deleted, "planned_chunks": result.Chunks, "planned_embeddings_reused": result.PlannedEmbeddingsReused, "planned_embeddings_pending": result.PlannedEmbeddingsPending}
		}
		value := map[string]any{"dry_run": false, "files_scanned": result.Scanned, "files_updated": result.Updated, "files_reused": result.Reused, "files_deleted": result.Deleted, "chunks_updated": result.Chunks, "embeddings_reused": outcome.Status.Ready, "embeddings_pending": outcome.Status.Pending, "manifest_sha256": outcome.Status.ManifestSHA256}
		if result.ActivatedGeneration != 0 {
			value["activated_generation"] = result.ActivatedGeneration
		}
		return value
	}
	result, ok := value.(index.Result)
	if !ok {
		return value
	}
	if result.DryRun {
		return map[string]any{"dry_run": true, "planned_files_updated": result.Updated, "planned_files_deleted": result.Deleted, "planned_chunks": result.Chunks}
	}
	wired := map[string]any{"dry_run": false, "files_scanned": result.Scanned, "files_updated": result.Updated, "files_reused": result.Reused, "files_deleted": result.Deleted, "chunks_updated": result.Chunks, "manifest_sha256": result.ManifestSHA256}
	if result.ActivatedGeneration != 0 {
		wired["activated_generation"] = result.ActivatedGeneration
	}
	return wired
}

func wireReadSpan(value app.ReadSpanResponse) map[string]any {
	return map[string]any{"path": value.Path, "start_line": value.StartLine, "end_line": value.EndLine, "body": string(value.Body), "indexed_sha256": value.IndexedSHA256}
}
func wireReadSpanBatch(value app.ReadSpanBatchResponse) map[string]any {
	evidence := make([]map[string]any, 0, len(value.Evidence))
	for _, item := range value.Evidence {
		evidence = append(evidence, wireReadSpan(item))
	}
	return map[string]any{"input_version": 2, "evidence": evidence}
}
func wireSearch(value any) any {
	response, ok := value.(search.Response)
	if !ok {
		return value
	}
	type hit struct {
		ChunkID         int64    `json:"chunk_id"`
		Path            string   `json:"path"`
		Language        string   `json:"language"`
		Kind            string   `json:"kind"`
		QualifiedSymbol string   `json:"qualified_symbol"`
		StartLine       int      `json:"start_line"`
		EndLine         int      `json:"end_line"`
		IndexedSHA256   string   `json:"indexed_sha256"`
		MatchSources    []string `json:"match_sources"`
	}
	type locatorKey struct {
		IndexedSHA256, Path, QualifiedSymbol string
		StartLine, EndLine                   int
	}
	hits := make([]hit, 0, len(response.Hits))
	positions := make(map[locatorKey]int, len(response.Hits))
	for _, source := range response.Hits {
		matchSources := compactMatchSources(source)
		key := locatorKey{IndexedSHA256: source.IndexedSHA256, Path: source.Path, QualifiedSymbol: source.QualifiedSymbol, StartLine: source.ParentRange.StartLine, EndLine: source.ParentRange.EndLine}
		if position, exists := positions[key]; exists {
			hits[position].MatchSources = mergeStrings(hits[position].MatchSources, matchSources)
			continue
		}
		positions[key] = len(hits)
		hits = append(hits, hit{ChunkID: source.ChunkID, Path: source.Path, Language: source.Language, Kind: source.Kind, QualifiedSymbol: source.QualifiedSymbol, StartLine: source.ParentRange.StartLine, EndLine: source.ParentRange.EndLine, IndexedSHA256: source.IndexedSHA256, MatchSources: matchSources})
	}
	return map[string]any{"results": hits}
}

func compactMatchSources(source search.Hit) []string {
	values := append([]string(nil), source.LexicalSources...)
	if source.ScoreSource == search.ScoreSourceVector || source.ScoreSource == search.ScoreSourceBoth {
		values = append(values, "vector")
	}
	if len(values) == 0 && source.ScoreSource == search.ScoreSourceFTS {
		values = append(values, "fts")
	}
	return mergeStrings(nil, values)
}

func mergeStrings(existing, additions []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(additions))
	result := make([]string, 0, len(existing)+len(additions))
	for _, group := range [][]string{existing, additions} {
		for _, value := range group {
			if value == "" {
				continue
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}
func requiredString(value map[string]json.RawMessage, key string, target *string) bool {
	input, ok := value[key]
	return ok && json.Unmarshal(input, target) == nil && *target != ""
}

func requiredLowerSHA256(value map[string]json.RawMessage, key string, target *string) bool {
	if !requiredString(value, key, target) || len(*target) != 64 {
		return false
	}
	for _, character := range *target {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func optionalBoolean(value map[string]json.RawMessage, key string, target *bool) bool {
	input, exists := value[key]
	if !exists {
		return true
	}
	var decoded *bool
	if err := json.Unmarshal(input, &decoded); err != nil || decoded == nil {
		return false
	}
	*target = *decoded
	return true
}

func requiredInteger(value map[string]json.RawMessage, key string, target *int) bool {
	input, ok := value[key]
	if !ok {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return false
	}
	number, ok := raw.(json.Number)
	if !ok || strings.ContainsAny(number.String(), ".eE") {
		return false
	}
	parsed, ok := new(big.Int).SetString(number.String(), 10)
	if !ok || !parsed.IsInt64() {
		return false
	}
	integer := parsed.Int64()
	if int64(int(integer)) != integer {
		return false
	}
	*target = int(integer)
	return true
}
