package evalcontract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"

	"cidx/internal/config"
)

func ArtifactChecksum(entries []ArtifactEntry) (string, error) {
	ordered := append([]ArtifactEntry(nil), entries...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	paths := map[string]struct{}{}
	for _, entry := range ordered {
		if !validRelativePath(entry.Path) || entry.MediaType == "" || !validSHA256(entry.SHA256) || entry.ByteSize < 0 {
			return "", fmt.Errorf("invalid portable artifact entry")
		}
		if _, exists := paths[entry.Path]; exists {
			return "", fmt.Errorf("duplicate portable artifact path")
		}
		paths[entry.Path] = struct{}{}
	}
	canonical, err := config.CanonicalJSON(ordered)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(canonical)
	return hex.EncodeToString(hash[:]), nil
}

func validRelativePath(value string) bool {
	if value == "" || strings.Contains(value, "\\") || path.IsAbs(value) {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return path.Clean(value) == value
}
