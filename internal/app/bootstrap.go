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
	"cidx/internal/runtimecheck"
	"cidx/internal/search"
	"cidx/internal/store"
	"cidx/internal/workspace"
)

type Application struct {
	Root      string
	StateRoot string
	Resolved  config.ResolvedConfig
	Store     *store.ProductionStore
	Index     *index.Service
	Search    *search.Service
	ReadSpan  ReadSpanService
	Status    StatusService
}

const initialConfigTemporaryName = ".config.json.init"

type initializationDependencies struct {
	openProduction  func(context.Context, string, config.ResolvedConfig) (*store.ProductionStore, error)
	runtimeCheck    func(context.Context) (runtimecheck.Capabilities, error)
	beforePublish   func(string)
	removeTemporary func(string) error
}

func defaultInitializationDependencies() initializationDependencies {
	return initializationDependencies{
		openProduction:  store.OpenProduction,
		runtimeCheck:    runtimecheck.Check,
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

// InitializeWorkspace creates an isolated development state namespace while
// indexing the separately supplied source checkout through the same runtime
// services used by ordinary projects.
func InitializeWorkspace(ctx context.Context, layout workspace.Layout, servingDimensions int, codec string) error {
	source, err := root.SourceRepository(ctx, layout.SourceRoot)
	if err != nil {
		return err
	}
	if layout.StateRoot == "" {
		return fmt.Errorf("state root is required")
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
	if _, err := runtimecheck.Check(ctx); err != nil {
		return fmt.Errorf("runtime capability check: %w", err)
	}
	state, err := prepareStateRoot(layout.StateRoot)
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
	production, err := store.OpenProductionAt(ctx, source, layout.StateRoot, resolved)
	if err != nil {
		return err
	}
	if err := production.Close(); err != nil {
		return err
	}
	if err := state.publishConfig(os.Remove); err != nil {
		return err
	}
	completed = true
	return nil
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
	if _, err := dependencies.runtimeCheck(ctx); err != nil {
		return fmt.Errorf("runtime capability check: %w", err)
	}
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
	cidxDir, dbDir, temporary, final string
	createdDir                       bool
	createdDBDir                     bool
	temporaryCreated                 bool
	configPublished                  bool
	claimedDatabase                  os.FileInfo
}

func prepareInitialState(canonical string) (initialState, error) {
	return prepareStateRoot(filepath.Join(canonical, ".cidx"))
}

func prepareStateRoot(stateRoot string) (initialState, error) {
	state := initialState{
		cidxDir:   stateRoot,
		dbDir:     filepath.Join(stateRoot, "db"),
		temporary: filepath.Join(stateRoot, initialConfigTemporaryName),
		final:     filepath.Join(stateRoot, "config.json"),
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
		if err := os.MkdirAll(state.cidxDir, 0o700); err != nil {
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
	paths := append([]string{state.final, state.temporary}, productionStatePaths(state.cidxDir)...)
	paths = append(paths, legacyProductionStatePaths(state.cidxDir)...)
	for _, path := range paths {
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

func legacyProductionStatePaths(cidxDir string) []string {
	return []string{
		filepath.Join(cidxDir, "index.db"),
		filepath.Join(cidxDir, "index.db-wal"),
		filepath.Join(cidxDir, "index.db-shm"),
		filepath.Join(cidxDir, "index.db-journal"),
	}
}

func productionStatePaths(cidxDir string) []string {
	return []string{
		filepath.Join(cidxDir, "db", "index.db"),
		filepath.Join(cidxDir, "db", "index.db-wal"),
		filepath.Join(cidxDir, "db", "index.db-shm"),
		filepath.Join(cidxDir, "db", "index.db-journal"),
	}
}

func (state *initialState) claimProductionDatabase() error {
	if err := os.Mkdir(state.dbDir, 0o700); err != nil {
		if !os.IsExist(err) {
			return err
		}
		info, statErr := os.Lstat(state.dbDir)
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("repository state path .cidx/db must be a real directory")
		}
	} else {
		state.createdDBDir = true
	}
	path := filepath.Join(state.dbDir, "index.db")
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
		if state.createdDBDir {
			_ = os.Remove(state.dbDir)
		}
		_ = os.Remove(state.cidxDir)
	} else if state.createdDBDir {
		_ = os.Remove(state.dbDir)
	}
}

func (state initialState) ownsClaimedDatabase() bool {
	if state.claimedDatabase == nil {
		return false
	}
	current, err := os.Stat(filepath.Join(state.dbDir, "index.db"))
	return err == nil && os.SameFile(state.claimedDatabase, current)
}

func Open(ctx context.Context, requestedRoot string) (*Application, error) {
	return open(ctx, requestedRoot, true, defaultOpenDependencies())
}

// OpenLocal assembles only local production services. Development planning
// uses it to guarantee that corpus/profile/raw preflight neither reads
// VOYAGE_API_KEY nor constructs a provider client.
func OpenLocal(ctx context.Context, requestedRoot string) (*Application, error) {
	return open(ctx, requestedRoot, false, defaultOpenDependencies())
}

func OpenWorkspace(ctx context.Context, layout workspace.Layout) (*Application, error) {
	return openWorkspace(ctx, layout, true)
}

func OpenWorkspaceLocal(ctx context.Context, layout workspace.Layout) (*Application, error) {
	return openWorkspace(ctx, layout, false)
}

func openWorkspace(ctx context.Context, layout workspace.Layout, allowProvider bool) (*Application, error) {
	source, err := root.SourceRepository(ctx, layout.SourceRoot)
	if err != nil {
		return nil, err
	}
	state, err := filepath.Abs(layout.StateRoot)
	if err != nil || state == "" {
		return nil, fmt.Errorf("state root is required")
	}
	resolved, err := config.Load(filepath.Join(state, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if _, err := runtimecheck.Check(ctx); err != nil {
		return nil, fmt.Errorf("runtime capability check: %w", err)
	}
	production, err := store.OpenProductionAt(ctx, source, state, resolved)
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
	return &Application{Root: source, StateRoot: production.StateRoot, Resolved: resolved, Store: production, Index: index.New(production), Search: searchService, ReadSpan: ReadSpanService{Root: source, Resolved: resolved}, Status: StatusService{Root: source, Resolved: resolved, Store: production}}, nil
}

type openDependencies struct {
	runtimeCheck   func(context.Context) (runtimecheck.Capabilities, error)
	openProduction func(context.Context, string, config.ResolvedConfig) (*store.ProductionStore, error)
}

func defaultOpenDependencies() openDependencies {
	return openDependencies{runtimeCheck: runtimecheck.Check, openProduction: store.OpenProduction}
}

func open(ctx context.Context, requestedRoot string, allowProvider bool, dependencies openDependencies) (*Application, error) {
	canonical, err := root.Repository(ctx, requestedRoot)
	if err != nil {
		return nil, err
	}
	resolved, err := config.Load(filepath.Join(canonical, ".cidx", "config.json"))
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if _, err := dependencies.runtimeCheck(ctx); err != nil {
		return nil, fmt.Errorf("runtime capability check: %w", err)
	}
	production, err := dependencies.openProduction(ctx, canonical, resolved)
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
	return &Application{Root: canonical, StateRoot: production.StateRoot, Resolved: resolved, Store: production, Index: index.New(production), Search: searchService, ReadSpan: ReadSpanService{Root: canonical, Resolved: resolved}, Status: StatusService{Root: canonical, Resolved: resolved, Store: production}}, nil
}

func (application *Application) Close() error {
	if application == nil || application.Store == nil {
		return nil
	}
	return application.Store.Close()
}
