package app

import (
	"cidx/internal/config"
	"cidx/internal/runtimecheck"
	"cidx/internal/store"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInitializeFromNestedDirectoryWritesDefaultConfigAndProductionMeta(t *testing.T) {
	ctx, repository := context.Background(), t.TempDir()
	runGit(t, repository, "init")
	nested := filepath.Join(repository, "nested", "directory")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(ctx, nested, 512); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repository, ".cidx", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(repository, ".cidx", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config permissions=%#o want 0600", info.Mode().Perm())
	}
	var got config.RawConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want, err := config.DefaultRaw(512)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("written defaults differ:\n got=%#v\nwant=%#v", got, want)
	}
	application, err := OpenLocal(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()
	stored, err := application.Store.StatusSnapshot(ctx, application.Resolved)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Applied.ActiveGeneration != 0 {
		t.Fatalf("initial generation=%d", stored.Applied.ActiveGeneration)
	}
	canonical, err := filepath.EvalSymlinks(repository)
	if err != nil {
		t.Fatal(err)
	}
	if application.Store.Root != canonical {
		t.Fatalf("meta root=%q want %q", application.Store.Root, canonical)
	}
	if _, err := os.Stat(filepath.Join(repository, ".cidx", "lab")); !os.IsNotExist(err) {
		t.Fatalf("init created lab state: %v", err)
	}
}

func TestInitializeRejectsExistingConfigBeforeDatabaseMutation(t *testing.T) {
	ctx, repository := context.Background(), t.TempDir()
	runGit(t, repository, "init")
	path := filepath.Join(repository, ".cidx", "config.json")
	original := []byte(`{"version":"preserve-me"}`)
	mustWriteFile(t, path, string(original))
	if err := Initialize(ctx, repository, 1024); err == nil {
		t.Fatal("existing config accepted")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("config changed:\n got=%q\nwant=%q", got, original)
	}
	if _, err := os.Stat(filepath.Join(repository, ".cidx", "db", "index.db")); !os.IsNotExist(err) {
		t.Fatalf("existing config mutated database state: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repository, ".cidx", initialConfigTemporaryName)); !os.IsNotExist(err) {
		t.Fatalf("existing config created staging state: %v", err)
	}
}

func TestInitializeRejectsRuntimeFailureBeforeCidxMutation(t *testing.T) {
	ctx, repository := context.Background(), t.TempDir()
	runGit(t, repository, "init")
	dependencies := defaultInitializationDependencies()
	dependencies.runtimeCheck = func(context.Context) (runtimecheck.Capabilities, error) {
		return runtimecheck.Capabilities{}, errors.New("injected runtime capability failure")
	}
	if err := initialize(ctx, repository, 1024, dependencies); err == nil {
		t.Fatal("runtime capability failure succeeded")
	}
	if _, err := os.Stat(filepath.Join(repository, ".cidx")); !os.IsNotExist(err) {
		t.Fatalf("runtime failure mutated .cidx: %v", err)
	}
}

func TestInitializeRejectsConfiglessProductionStateWithoutMutation(t *testing.T) {
	ctx, repository := context.Background(), t.TempDir()
	runGit(t, repository, "init")
	path := filepath.Join(repository, ".cidx", "db", "index.db")
	original := []byte("orphaned production state")
	mustWriteFile(t, path, string(original))
	if err := Initialize(ctx, repository, 1024); err == nil {
		t.Fatal("configless production state accepted")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("production state changed:\n got=%q\nwant=%q", got, original)
	}
	for _, path := range []string{"config.json", initialConfigTemporaryName} {
		if _, err := os.Stat(filepath.Join(repository, ".cidx", path)); !os.IsNotExist(err) {
			t.Fatalf("orphan preflight created %s: %v", path, err)
		}
	}
}

func TestInitializeCleansFailedProductionAttemptAndCanRetry(t *testing.T) {
	ctx, repository := context.Background(), t.TempDir()
	runGit(t, repository, "init")
	dependencies := defaultInitializationDependencies()
	dependencies.openProduction = func(_ context.Context, root string, _ config.ResolvedConfig) (*store.ProductionStore, error) {
		for _, path := range productionStatePaths(filepath.Join(root, ".cidx")) {
			if err := os.WriteFile(path, []byte("created by failed initializer"), 0o600); err != nil {
				return nil, err
			}
		}
		return nil, errors.New("injected production open failure")
	}
	if err := initialize(ctx, repository, 1024, dependencies); err == nil {
		t.Fatal("injected production open failure succeeded")
	}
	if _, err := os.Stat(filepath.Join(repository, ".cidx")); !os.IsNotExist(err) {
		t.Fatalf("failed init left state directory: %v", err)
	}
	if err := Initialize(ctx, repository, 1024); err != nil {
		t.Fatalf("retry after cleanup failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repository, ".cidx", "config.json")); err != nil {
		t.Fatal(err)
	}
}

func TestInitializeDoesNotReplaceConfigCreatedBeforePublication(t *testing.T) {
	ctx, repository := context.Background(), t.TempDir()
	runGit(t, repository, "init")
	dependencies := defaultInitializationDependencies()
	external := []byte(`{"version":"external"}`)
	dependencies.beforePublish = func(root string) {
		if err := os.WriteFile(filepath.Join(root, ".cidx", "config.json"), external, 0o600); err != nil {
			t.Fatalf("create concurrent config: %v", err)
		}
	}
	if err := initialize(ctx, repository, 1024, dependencies); err == nil {
		t.Fatal("concurrently created config was replaced")
	}
	got, err := os.ReadFile(filepath.Join(repository, ".cidx", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(external) {
		t.Fatalf("concurrent config changed:\n got=%q\nwant=%q", got, external)
	}
	paths := append([]string{filepath.Join(repository, ".cidx", initialConfigTemporaryName)}, productionStatePaths(filepath.Join(repository, ".cidx"))...)
	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("failed publication left %s: %v", path, err)
		}
	}
}

func TestInitializePreservesDatabaseReplacedAfterClaim(t *testing.T) {
	ctx, repository := context.Background(), t.TempDir()
	runGit(t, repository, "init")
	dependencies := defaultInitializationDependencies()
	externalDatabase := []byte("external replacement database")
	externalWAL := []byte("external replacement wal")
	dependencies.openProduction = func(_ context.Context, root string, _ config.ResolvedConfig) (*store.ProductionStore, error) {
		dir := filepath.Join(root, ".cidx", "db")
		if err := os.Remove(filepath.Join(dir, "index.db")); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, "index.db"), externalDatabase, 0o600); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, "index.db-wal"), externalWAL, 0o600); err != nil {
			return nil, err
		}
		return nil, errors.New("injected failure after replacement")
	}
	if err := initialize(ctx, repository, 1024, dependencies); err == nil {
		t.Fatal("replacement failure succeeded")
	}
	for path, want := range map[string][]byte{
		"index.db":     externalDatabase,
		"index.db-wal": externalWAL,
	} {
		got, err := os.ReadFile(filepath.Join(repository, ".cidx", "db", path))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Fatalf("%s changed: got=%q want=%q", path, got, want)
		}
	}
	for _, path := range []string{"config.json", initialConfigTemporaryName} {
		if _, err := os.Stat(filepath.Join(repository, ".cidx", path)); !os.IsNotExist(err) {
			t.Fatalf("replacement failure left %s: %v", path, err)
		}
	}
}

func TestInitializeSucceedsWhenPostLinkStagingCleanupFails(t *testing.T) {
	ctx, repository := context.Background(), t.TempDir()
	runGit(t, repository, "init")
	dependencies := defaultInitializationDependencies()
	dependencies.removeTemporary = func(string) error { return errors.New("injected staging removal failure") }
	if err := initialize(ctx, repository, 1024, dependencies); err != nil {
		t.Fatalf("committed init reported failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repository, ".cidx", "config.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repository, ".cidx", initialConfigTemporaryName)); err != nil {
		t.Fatalf("expected leftover staging link: %v", err)
	}
	application, err := OpenLocal(ctx, repository)
	if err != nil {
		t.Fatalf("committed init is not usable: %v", err)
	}
	if err := application.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenProductionAssemblyDoesNotCreateLabState(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	dim := 512
	raw := config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: 4096, TargetSegmentBytes: 2048}, Embedding: config.RawEmbedding{ServingDimensions: &dim, Request: config.RawRequest{MaxInputs: 1, MaxTotalInputBytes: 8192, TimeoutSeconds: 1}}, MCP: config.RawMCP{HardMaxInlineBytes: 1024}}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), string(data))
	application, err := Open(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()
	if _, err := os.Stat(filepath.Join(root, ".cidx", "lab")); !os.IsNotExist(err) {
		t.Fatalf("serve assembly created lab state: %v", err)
	}
}

func TestOpenChecksRuntimeBeforeProductionOpen(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	dim := 512
	raw := config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: 4096, TargetSegmentBytes: 2048}, Embedding: config.RawEmbedding{ServingDimensions: &dim, Request: config.RawRequest{MaxInputs: 1, MaxTotalInputBytes: 8192, TimeoutSeconds: 1}}, MCP: config.RawMCP{HardMaxInlineBytes: 1024}}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), string(data))
	called := false
	dependencies := defaultOpenDependencies()
	dependencies.runtimeCheck = func(context.Context) (runtimecheck.Capabilities, error) {
		return runtimecheck.Capabilities{}, errors.New("injected runtime capability failure")
	}
	dependencies.openProduction = func(context.Context, string, config.ResolvedConfig) (*store.ProductionStore, error) {
		called = true
		return nil, errors.New("must not open production")
	}
	if _, err := open(ctx, root, false, dependencies); err == nil {
		t.Fatal("runtime capability failure succeeded")
	}
	if called {
		t.Fatal("production opened after runtime failure")
	}
	if _, err := os.Stat(filepath.Join(root, ".cidx", "db", "index.db")); !os.IsNotExist(err) {
		t.Fatalf("runtime failure opened production database: %v", err)
	}
}
