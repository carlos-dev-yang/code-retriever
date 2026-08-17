package relationdiag

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ChecksumEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type Checksums struct {
	SchemaVersion int             `json:"schema_version"`
	Kind          string          `json:"kind"`
	Complete      bool            `json:"complete"`
	Entries       []ChecksumEntry `json:"entries"`
	Checksum      string          `json:"checksum"`
}

func writeChecksums(root string) error {
	entries, err := checksumEntries(root)
	if err != nil {
		return err
	}
	checksum, err := canonicalHash(entries)
	if err != nil {
		return err
	}
	value := Checksums{SchemaVersion: SchemaVersion, Kind: "cidx.relation_diagnostic.artifact_checksums.v1", Complete: true, Entries: entries, Checksum: checksum}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "artifact-checksums.json"), append(data, '\n'), 0o600)
}

// verifyChecksums proves an already-published immutable artifact before it is
// consumed. expectedPaths intentionally excludes artifact-checksums.json: the
// checksum manifest authenticates every other file and cannot include itself.
func verifyChecksums(root string, expectedPaths []string) error {
	data, err := os.ReadFile(filepath.Join(root, "artifact-checksums.json"))
	if err != nil {
		return err
	}
	var recorded Checksums
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&recorded); err != nil {
		return fmt.Errorf("invalid relation artifact checksums: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("invalid relation artifact checksums trailing data")
	}
	if recorded.SchemaVersion != SchemaVersion || recorded.Kind != "cidx.relation_diagnostic.artifact_checksums.v1" || !recorded.Complete {
		return fmt.Errorf("invalid relation artifact checksum manifest")
	}
	actual, err := checksumEntries(root)
	if err != nil {
		return err
	}
	if len(actual) != len(recorded.Entries) || len(actual) != len(expectedPaths) {
		return fmt.Errorf("relation artifact checksum entry-set mismatch")
	}
	wanted := map[string]bool{}
	for _, path := range expectedPaths {
		if !validRelative(path) || wanted[path] {
			return fmt.Errorf("invalid expected relation artifact entry")
		}
		wanted[path] = true
	}
	for index, entry := range recorded.Entries {
		if !validRelative(entry.Path) || !validDigest(entry.SHA256) || entry.Bytes < 0 || !wanted[entry.Path] || entry != actual[index] {
			return fmt.Errorf("relation artifact checksum entry mismatch")
		}
	}
	checksum, err := canonicalHash(recorded.Entries)
	if err != nil || checksum != recorded.Checksum || !validDigest(recorded.Checksum) {
		return fmt.Errorf("relation artifact checksum mismatch")
	}
	return nil
}

func checksumEntries(root string) ([]ChecksumEntry, error) {
	var entries []ChecksumEntry
	err := filepath.WalkDir(root, func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "artifact-checksums.json" {
			return nil
		}
		if !validRelative(rel) {
			return fmt.Errorf("unsafe relation artifact entry")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular relation artifact entry")
		}
		digest, err := fileSHA256(file)
		if err != nil {
			return err
		}
		entries = append(entries, ChecksumEntry{Path: rel, SHA256: digest, Bytes: info.Size()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func portableNoAbsolute(data []byte) bool {
	// JSON source bodies are forbidden by the caller. This is a conservative
	// final guard for platform-looking absolute locations.
	return !strings.Contains(string(data), `"/Users/`) && !strings.Contains(string(data), `"C:\\`)
}
