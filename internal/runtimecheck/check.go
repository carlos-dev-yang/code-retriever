// Package runtimecheck verifies the executable's bundled free-runtime
// dependencies before repository state is touched. It never repairs a failed
// check and has no network or provider dependency.
package runtimecheck

import (
	"context"
	"fmt"
	"os"

	"cidx/internal/chunk"
	"cidx/internal/store"
)

type Capabilities struct {
	FTS5Available           bool             `json:"fts5_available"`
	WALAvailable            bool             `json:"wal_available"`
	SQLiteDriver            string           `json:"sqlite_driver"`
	SQLiteVersion           string           `json:"sqlite_version"`
	RegisteredLanguages     []chunk.Language `json:"registered_languages"`
	ProductionSchemaMinimum int              `json:"production_schema_minimum"`
	ProductionSchemaMaximum int              `json:"production_schema_maximum"`
}

// Check opens a disposable SQLite database and parses one valid program for
// each bundled grammar. It intentionally probes all v1 languages so a binary
// cannot defer an absent grammar until a project happens to contain it.
func Check(ctx context.Context) (Capabilities, error) {
	file, err := os.CreateTemp("", "cidx-runtimecheck-*.db")
	if err != nil {
		return Capabilities{}, fmt.Errorf("create runtime check database: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return Capabilities{}, fmt.Errorf("close runtime check database: %w", err)
	}
	defer func() {
		for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
			_ = os.Remove(path + suffix)
		}
	}()
	stores, sqlite, err := store.OpenSQLiteStores(ctx, path)
	if err != nil {
		return Capabilities{}, fmt.Errorf("verify SQLite FTS5/WAL capability: %w", err)
	}
	defer stores.Close()
	parser := chunk.NewEmbeddedParser()
	probes := []struct {
		language chunk.Language
		source   []byte
	}{
		{chunk.Go, []byte("package probe\nfunc main() {}\n")},
		{chunk.TypeScript, []byte("export function probe(): number { return 1 }\n")},
		{chunk.TSX, []byte("export const Probe = () => <div>ok</div>\n")},
	}
	capabilities := Capabilities{
		FTS5Available:           sqlite.FTS5,
		WALAvailable:            sqlite.WAL,
		SQLiteDriver:            sqlite.Driver,
		SQLiteVersion:           sqlite.DriverVersion,
		ProductionSchemaMinimum: 1,
		ProductionSchemaMaximum: store.ProductionSchemaVersion,
	}
	for _, probe := range probes {
		result, err := parser.Parse(probe.language, probe.source)
		if err != nil {
			return Capabilities{}, fmt.Errorf("verify bundled %s grammar: %w", probe.language, err)
		}
		if result.HasError || result.RootKind == "" {
			return Capabilities{}, fmt.Errorf("verify bundled %s grammar: invalid parse result", probe.language)
		}
		capabilities.RegisteredLanguages = append(capabilities.RegisteredLanguages, probe.language)
	}
	return capabilities, nil
}
