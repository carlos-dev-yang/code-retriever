// cidx-spike is a deliberately non-production runtime verification command.
// It has no network/client path and accepts only explicit local input/output.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"cidx/internal/chunk"
	"cidx/internal/store"
)

func main() {
	input := flag.String("input", "", "source file to parse")
	language := flag.String("language", "", "go, typescript, or tsx")
	database := flag.String("db", "", "explicit output SQLite spike database path")
	flag.Parse()
	if *input == "" || *language == "" || *database == "" {
		fmt.Fprintln(os.Stderr, "-input, -language, and -db are required")
		os.Exit(2)
	}
	if isProductionIndexPath(*database) {
		fmt.Fprintln(os.Stderr, "spike runner refuses the production database path")
		os.Exit(2)
	}
	source, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	parser := chunk.NewEmbeddedParser()
	parsed, err := parser.Parse(chunk.Language(*language), source)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	stores, capabilities, err := store.OpenSpikeStores(context.Background(), *database)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer stores.Close()
	fmt.Printf("parsed root=%s has_error=%t sqlite=%s@%s fts5=%t wal=%t platform=%s/%s\n", parsed.RootKind, parsed.HasError, capabilities.Driver, capabilities.DriverVersion, capabilities.FTS5, capabilities.WAL, capabilities.GOOS, capabilities.GOARCH)
}

// isProductionIndexPath refuses any direct or resolved path structurally
// shaped as .cidx/db/index.db. This spike command has no repository-root contract,
// so the conservative structural guard avoids modifying production state.
func isProductionIndexPath(path string) bool {
	if productionIndexShape(filepath.Clean(path)) {
		return true
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil && productionIndexShape(resolved) {
		return true
	}
	if resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(path)); err == nil && productionIndexShape(filepath.Join(resolvedParent, filepath.Base(path))) {
		return true
	}
	return false
}

func productionIndexShape(path string) bool {
	return filepath.Base(path) == "index.db" && filepath.Base(filepath.Dir(path)) == ".cidx"
}
