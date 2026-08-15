// Package buildinfo reports immutable facts about the executable. It does not
// read project configuration or open repository state.
package buildinfo

import (
	"runtime"
	"runtime/debug"

	golangchunk "cidx/internal/chunk/golang"
	typescriptchunk "cidx/internal/chunk/typescript"
	"cidx/internal/config"
	"cidx/internal/store"
)

// Version, Commit, and CGOEnabled are set by the local packaging command. An
// ordinary developer build intentionally remains useful through the build-info
// fallbacks below.
var (
	Version    = ""
	Commit     = ""
	CGOEnabled = ""
)

const (
	ManifestSchemaVersion  = 1
	SQLiteImplementationID = "modernc.org/sqlite"
	GrammarBindingID       = "github.com/tree-sitter/go-tree-sitter"
	GoGrammarID            = "github.com/tree-sitter/tree-sitter-go"
	TypeScriptGrammarID    = "github.com/tree-sitter/tree-sitter-typescript"
	LinkPolicy             = "cgo-required-for-grammars; static-linkage-not-claimed"
)

// Info is deliberately made from scalar fields and ordered slices so that its
// JSON representation is stable. It contains no build time: reproducible
// local artifacts must not acquire a clock-dependent identity.
type Info struct {
	ManifestSchemaVersion    int      `json:"manifest_schema_version"`
	Version                  string   `json:"version"`
	Commit                   string   `json:"commit"`
	SourceModified           string   `json:"source_modified"`
	GoVersion                string   `json:"go_version"`
	TargetOS                 string   `json:"target_os"`
	TargetArch               string   `json:"target_arch"`
	CGOEnabled               string   `json:"cgo_enabled"`
	SQLiteImplementationID   string   `json:"sqlite_implementation_id"`
	SQLiteVersion            string   `json:"sqlite_version"`
	GrammarImplementationIDs []string `json:"grammar_implementation_ids"`
	ChunkerImplementationIDs []string `json:"chunker_implementation_ids"`
	IndexChunkerVersion      int      `json:"index_chunker_version"`
	FTSSchemaVersion         int      `json:"fts_schema_version"`
	ProductionSchemaVersion  int      `json:"production_schema_version"`
	LinkPolicy               string   `json:"link_policy"`
}

// Current returns facts for the running executable. Unknown provenance is
// represented explicitly rather than guessed from the checkout.
func Current() Info {
	build, ok := debug.ReadBuildInfo()
	info := Info{
		ManifestSchemaVersion:  ManifestSchemaVersion,
		Version:                "devel",
		Commit:                 "unknown",
		SourceModified:         "unknown",
		GoVersion:              runtime.Version(),
		TargetOS:               runtime.GOOS,
		TargetArch:             runtime.GOARCH,
		CGOEnabled:             "unknown",
		SQLiteImplementationID: SQLiteImplementationID,
		SQLiteVersion:          dependencyVersion(build, SQLiteImplementationID),
		GrammarImplementationIDs: []string{
			GrammarBindingID + "@" + dependencyVersion(build, GrammarBindingID),
			GoGrammarID + "@" + dependencyVersion(build, GoGrammarID),
			TypeScriptGrammarID + "@" + dependencyVersion(build, TypeScriptGrammarID),
		},
		ChunkerImplementationIDs: []string{
			golangchunk.ChunkerVersion,
			typescriptchunk.ChunkerVersion,
		},
		IndexChunkerVersion:     config.IndexChunkerVersion,
		FTSSchemaVersion:        config.FTSSchemaVersion,
		ProductionSchemaVersion: store.ProductionSchemaVersion,
		LinkPolicy:              LinkPolicy,
	}
	if ok && build.GoVersion != "" {
		info.GoVersion = build.GoVersion
	}
	if Version != "" {
		info.Version = Version
	} else if ok && build.Main.Version != "" && build.Main.Version != "(devel)" {
		info.Version = build.Main.Version
	}
	if Commit != "" {
		info.Commit = Commit
	}
	if CGOEnabled != "" {
		info.CGOEnabled = CGOEnabled
	}
	if ok {
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				if Commit == "" && setting.Value != "" {
					info.Commit = setting.Value
				}
			case "vcs.modified":
				if setting.Value == "true" || setting.Value == "false" {
					info.SourceModified = setting.Value
				}
			case "CGO_ENABLED":
				if CGOEnabled == "" && setting.Value != "" {
					info.CGOEnabled = setting.Value
				}
			}
		}
	}
	return info
}

func dependencyVersion(build *debug.BuildInfo, path string) string {
	if build == nil {
		return "unknown"
	}
	for _, dependency := range build.Deps {
		if dependency.Path == path {
			return dependency.Version
		}
	}
	return "unknown"
}
