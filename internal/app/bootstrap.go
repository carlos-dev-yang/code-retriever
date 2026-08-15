// Package app assembles production application services. It deliberately has
// no lab dependency; development-only assembly lives under internal/devlab.
package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cidx/internal/config"
	"cidx/internal/embedclient"
	"cidx/internal/index"
	"cidx/internal/root"
	"cidx/internal/search"
	"cidx/internal/store"
)

type Application struct {
	Root     string
	Resolved config.ResolvedConfig
	Store    *store.ProductionStore
	Index    *index.Service
	Search   *search.Service
	ReadSpan ReadSpanService
	Status   StatusService
}

func Open(ctx context.Context, requestedRoot string) (*Application, error) {
	return open(ctx, requestedRoot, true)
}

// OpenLocal assembles only local production services. Development planning
// uses it to guarantee that corpus/profile/raw preflight neither reads
// VOYAGE_API_KEY nor constructs a provider client.
func OpenLocal(ctx context.Context, requestedRoot string) (*Application, error) {
	return open(ctx, requestedRoot, false)
}

func open(ctx context.Context, requestedRoot string, allowProvider bool) (*Application, error) {
	canonical, err := root.Repository(ctx, requestedRoot)
	if err != nil {
		return nil, err
	}
	resolved, err := config.Load(filepath.Join(canonical, ".cidx", "config.json"))
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	production, err := store.OpenProduction(ctx, canonical, resolved)
	if err != nil {
		return nil, err
	}
	var client embedclient.EmbeddingClient
	if allowProvider {
		if key := os.Getenv("VOYAGE_API_KEY"); key != "" {
			client = embedclient.VoyageClient{APIKey: key, HTTPClient: &http.Client{Timeout: time.Duration(resolved.Embedding.Request.TimeoutSeconds) * time.Second}}
		}
	}
	searchService, err := search.New(production, resolved, client)
	if err != nil {
		_ = production.Close()
		return nil, err
	}
	return &Application{Root: canonical, Resolved: resolved, Store: production, Index: index.New(production), Search: searchService, ReadSpan: ReadSpanService{Root: canonical, Resolved: resolved}, Status: StatusService{Root: canonical, Resolved: resolved, Store: production}}, nil
}

func (application *Application) Close() error {
	if application == nil || application.Store == nil {
		return nil
	}
	return application.Store.Close()
}
