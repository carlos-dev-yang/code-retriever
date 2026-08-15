// Package app assembles production application services. It deliberately has
// no lab dependency; development-only assembly lives under internal/devlab.
package app

import (
	"context"
	"encoding/json"
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

const initialConfigTemporaryName = ".config.json.init"

type initializationDependencies struct {
	openProduction  func(context.Context, string, config.ResolvedConfig) (*store.ProductionStore, error)
	beforePublish   func(string)
	removeTemporary func(string) error
}

func defaultInitializationDependencies() initializationDependencies {
	return initializationDependencies{
		openProduction:  store.OpenProduction,
		beforePublish:   func(string) {},
		removeTemporary: os.Remove,
	}
}

// Initialize creates the complete local production state at the containing
// Git worktree root. It validates the default configuration before any write,
// stages it privately until production SQLite is ready, and never constructs
// an embedding provider.
func Initialize(ctx context.Context, requestedRoot string, servingDimensions int, codec string) error {
	return initialize(ctx, requestedRoot, servingDimensions, codec, defaultInitializationDependencies())
}

func initialize(ctx context.Context, requestedRoot string, servingDimensions int, codec string, dependencies initializationDependencies) error {
	canonical, err := root.GitRoot(ctx, requestedRoot)
	if err != nil {
		return err
	}
	raw, err := config.DefaultRaw(servingDimensions, codec)
	if err != nil {
		return err
	}
	resolved, err := config.Resolve(raw)
	if err != nil {
		return fmt.Errorf("resolve default config: %w", err)
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("encode default config: %w", err)
	}
	data = append(data, '\n')
	state, err := prepareInitialState(canonical)
	if err != nil {
		return err
	}
	completed := false
	defer func() {
		if !completed {
			state.cleanup()
		}
	}()
	if err := state.stageConfig(data); err != nil {
		return err
	}
	if err := state.claimProductionDatabase(); err != nil {
		return err
	}
	production, err := dependencies.openProduction(ctx, canonical, resolved)
	if err != nil {
		return err
	}
	if err := production.Close(); err != nil {
		return err
	}
	dependencies.beforePublish(canonical)
	if err := state.publishConfig(dependencies.removeTemporary); err != nil {
		return err
	}
	completed = true
	return nil
}

type initialState struct {
	cidxDir, temporary, final string
	createdDir                bool
	temporaryCreated          bool
	configPublished           bool
	claimedDatabase           os.FileInfo
}

func prepareInitialState(canonical string) (initialState, error) {
	state := initialState{
		cidxDir:   filepath.Join(canonical, ".cidx"),
		temporary: filepath.Join(canonical, ".cidx", initialConfigTemporaryName),
		final:     filepath.Join(canonical, ".cidx", "config.json"),
	}
	if err := rejectExistingInitialState(state); err != nil {
		return initialState{}, err
	}
	info, err := os.Lstat(state.cidxDir)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return initialState{}, fmt.Errorf("repository state path .cidx must be a real directory")
		}
	} else if os.IsNotExist(err) {
		if err := os.Mkdir(state.cidxDir, 0o700); err != nil {
			return initialState{}, err
		}
		state.createdDir = true
	} else {
		return initialState{}, err
	}
	if state.createdDir {
		if err := os.Chmod(state.cidxDir, 0o700); err != nil {
			_ = os.Remove(state.cidxDir)
			return initialState{}, err
		}
	}
	return state, nil
}

func rejectExistingInitialState(state initialState) error {
	for _, path := range append([]string{state.final, state.temporary}, productionStatePaths(state.cidxDir)...) {
		if _, err := os.Lstat(path); err == nil {
			if path == state.final {
				return fmt.Errorf("repository config already exists")
			}
			return fmt.Errorf("repository production state already exists")
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func productionStatePaths(cidxDir string) []string {
	return []string{
		filepath.Join(cidxDir, "index.db"),
		filepath.Join(cidxDir, "index.db-wal"),
		filepath.Join(cidxDir, "index.db-shm"),
		filepath.Join(cidxDir, "index.db-journal"),
	}
}

func (state *initialState) claimProductionDatabase() error {
	path := filepath.Join(state.cidxDir, "index.db")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("repository production state already exists")
		}
		return err
	}
	info, err := file.Stat()
	if err == nil {
		state.claimedDatabase = info
	}
	if err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func (state *initialState) stageConfig(data []byte) error {
	file, err := os.OpenFile(state.temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("repository initialization is already in progress")
		}
		return err
	}
	state.temporaryCreated = true
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	for len(data) > 0 {
		written, err := file.Write(data)
		if err != nil {
			_ = file.Close()
			return err
		}
		if written == 0 {
			_ = file.Close()
			return fmt.Errorf("write initial config: no progress")
		}
		data = data[written:]
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return nil
}

func (state *initialState) publishConfig(removeTemporary func(string) error) error {
	if err := os.Link(state.temporary, state.final); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("repository config already exists")
		}
		return err
	}
	state.configPublished = true
	state.temporaryCreated = false
	// The hard link is the no-replace commit point. Cleanup cannot turn a
	// usable final config and database into a failed initialization.
	_ = removeTemporary(state.temporary)
	return nil
}

func (state initialState) cleanup() {
	if state.temporaryCreated {
		_ = os.Remove(state.temporary)
	}
	if !state.configPublished && state.ownsClaimedDatabase() {
		paths := productionStatePaths(state.cidxDir)
		for _, path := range paths[1:] {
			_ = os.Remove(path)
		}
		_ = os.Remove(paths[0])
	}
	if state.createdDir {
		_ = os.Remove(state.cidxDir)
	}
}

func (state initialState) ownsClaimedDatabase() bool {
	if state.claimedDatabase == nil {
		return false
	}
	current, err := os.Stat(filepath.Join(state.cidxDir, "index.db"))
	return err == nil && os.SameFile(state.claimedDatabase, current)
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
