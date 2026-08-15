package app

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"

	"cidx/internal/index"
	"cidx/internal/search"
)

type SearchRequest struct {
	Query, Mode string
	K           int
	MaxInline   int
}

func (application *Application) SearchCommand(ctx context.Context, request SearchRequest) (search.Response, error) {
	if application == nil || application.Search == nil {
		return search.Response{}, fmt.Errorf("search service is unavailable")
	}
	effective := request.MaxInline
	if effective > application.Resolved.MCP.HardMaxInlineBytes {
		effective = application.Resolved.MCP.HardMaxInlineBytes
	}
	response, err := application.Search.Search(ctx, search.Request{Query: request.Query, Mode: search.SearchMode(request.Mode), K: request.K, EffectiveMaxInlineBytes: effective})
	if err != nil {
		return response, err
	}
	response.RequestedMaxInlineBytes, response.EffectiveMaxInlineBytes, response.MaxInlineBytesClamped = request.MaxInline, effective, request.MaxInline != effective
	seen := map[string]string{}
	for index := range response.Hits {
		path := response.Hits[index].Path
		state, ok := seen[path]
		if !ok {
			body, readErr := readRegularNoSymlink(application.Root, path, application.Resolved.Index.MaxSourceFileBytes)
			switch {
			case os.IsNotExist(readErr):
				state = "deleted"
			case readErr != nil:
				state = "stale"
			case fmt.Sprintf("%x", sha256.Sum256(body)) == response.Hits[index].IndexedSHA256:
				state = "current"
			default:
				state = "stale"
			}
			seen[path] = state
		}
		response.Hits[index].SourceState = state
	}
	return response, nil
}

func (application *Application) Reindex(ctx context.Context, dryRun bool, reason index.Reason) (index.Result, error) {
	if application == nil || application.Index == nil {
		return index.Result{}, fmt.Errorf("index service is unavailable")
	}
	return application.Index.Execute(ctx, index.Request{Root: application.Root, DryRun: dryRun, Reason: reason, Config: application.Resolved})
}
