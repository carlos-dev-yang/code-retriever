package lexical

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"cidx/internal/config"
	"cidx/internal/index"
	"cidx/internal/store"
)

func TestSearcherFindsLiveIndexedLanguagesAndRanksDeterministically(t *testing.T) {
	ctx, _, production, resolved := lexicalFixture(t)
	defer production.Close()
	searcher, err := New(production, resolved)
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		query, symbol string
	}{
		{"GetUserByID", "GetUserByID"},
		{"render user panel", "UserPanel"},
		{"validate user input", "validateUserInput"},
		{"render user panel tokenabsentfromcorpus", "UserPanel"},
		{"special_location.go", "LocateByFile"},
	} {
		result, err := searcher.Search(ctx, Request{Query: check.query})
		if err != nil {
			t.Fatalf("%q: %v", check.query, err)
		}
		if result.IndexGeneration != 1 || len(result.Hits) == 0 || result.Hits[0].Symbol != check.symbol {
			t.Fatalf("%q result=%#v", check.query, result)
		}
		for ordinal, hit := range result.Hits {
			if hit.LexicalRank != ordinal+1 || len(hit.IndexedSHA256) != 64 {
				t.Fatalf("%q ranks=%#v", check.query, result.Hits)
			}
		}
	}
	anchored, err := searcher.Search(ctx, Request{Query: "GetUserByID"})
	if err != nil || anchored.Diagnostics.SymbolCandidateCount == 0 || anchored.Hits[0].SymbolRank == 0 || anchored.Hits[0].MatchedTerms != anchored.Hits[0].SelectedTerms {
		t.Fatalf("symbol lane=%#v err=%v", anchored, err)
	}
	pathLed, err := searcher.Search(ctx, Request{Query: "special_location.go"})
	if err != nil || pathLed.Diagnostics.PathCandidateCount == 0 || pathLed.Hits[0].PathRank == 0 || pathLed.Hits[0].Path != "special_location.go" {
		t.Fatalf("path lane=%#v err=%v", pathLed, err)
	}
	first, err := searcher.Search(ctx, Request{Query: "GetUserByID"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := searcher.Search(ctx, Request{Query: "GetUserByID"})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Hits) != len(second.Hits) {
		t.Fatalf("result count changed: %d/%d", len(first.Hits), len(second.Hits))
	}
	for i := range first.Hits {
		if first.Hits[i].ChunkID != second.Hits[i].ChunkID || first.Hits[i].LexicalRank != second.Hits[i].LexicalRank {
			t.Fatalf("non-deterministic result: %#v / %#v", first.Hits, second.Hits)
		}
	}
	exact, err := searcher.Search(ctx, Request{Query: "sample.GetUserByID"})
	if err != nil || len(exact.Hits) == 0 || !exact.Hits[0].ExactSymbolMatched {
		t.Fatalf("qualified exact tie-break=%#v err=%v", exact, err)
	}
}

func TestSearcherRejectsFTSGrammarAndFailsClosedForOrphan(t *testing.T) {
	ctx, root, production, resolved := lexicalFixture(t)
	defer production.Close()
	searcher, err := New(production, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := searcher.Search(ctx, Request{Query: `GetUserByID" OR "*`}); err != nil {
		t.Fatalf("safe FTS-like input rejected: %v", err)
	}
	database, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "db", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.ExecContext(ctx, `INSERT INTO chunk_fts(rowid,symbols,body) VALUES(999999,'orphaned symbol','orphaned body')`); err != nil {
		t.Fatal(err)
	}
	_, err = searcher.Search(ctx, Request{Query: "orphaned"})
	if !errors.Is(err, store.ErrIndexCorrupt) {
		t.Fatalf("orphan result error=%v", err)
	}
	if _, err := searcher.Search(ctx, Request{Query: "GetUserByID", CandidateK: -1}); err == nil {
		t.Fatal("negative candidate limit accepted")
	}
}

func lexicalFixture(t *testing.T) (context.Context, string, *store.ProductionStore, config.ResolvedConfig) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	runGit(t, root, "init")
	mustWrite(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWrite(t, filepath.Join(root, "user.go"), "package sample\n\nfunc GetUserByID() error {\n\treturn nil\n}\n")
	mustWrite(t, filepath.Join(root, "input.ts"), "export function validateUserInput(value: string): boolean { return value.length > 0 }\n")
	mustWrite(t, filepath.Join(root, "panel.tsx"), "export function UserPanel() { return <section>render user panel</section> }\n")
	mustWrite(t, filepath.Join(root, "special_location.go"), "package sample\n\nfunc LocateByFile() error { return nil }\n")
	runGit(t, root, "add", "user.go")
	resolved := testConfig(t)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
		production.Close()
		t.Fatal(err)
	}
	return ctx, root, production, resolved
}

func testConfig(t *testing.T) config.ResolvedConfig {
	t.Helper()
	target := 1024
	resolved, err := config.Resolve(config.RawConfig{
		Version:   1,
		Index:     config.RawIndex{Languages: []string{"go", "typescript", "tsx"}, MaxSourceFileBytes: 1 << 20, TargetSegmentBytes: 1 << 20},
		Embedding: config.RawEmbedding{ServingDimensions: &target, Request: config.RawRequest{MaxInputs: 1, MaxTotalInputBytes: 1, TimeoutSeconds: 1}},
		MCP:       config.RawMCP{HardMaxInlineBytes: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func mustWrite(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
