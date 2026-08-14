package ignore

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

type Candidate struct {
	Path      string
	Exclusion string
}

func Enumerate(ctx context.Context, root string, maxBytes int64) ([]Candidate, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("enumerate Git worktree: %w", err)
	}
	return Filter(root, out, maxBytes)
}
func Filter(root string, nul []byte, maxBytes int64) ([]Candidate, error) {
	patterns, err := readPatterns(filepath.Join(root, ".cidxignore"))
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var result []Candidate
	for _, raw := range bytes.Split(nul, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		path, err := normalize(string(raw))
		if err != nil {
			return nil, err
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		exclusion, err := excluded(root, path, patterns, maxBytes)
		if err != nil {
			return nil, err
		}
		if exclusion == "" {
			result = append(result, Candidate{Path: path})
		} else {
			result = append(result, Candidate{Path: path, Exclusion: exclusion})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}
func normalize(p string) (string, error) {
	if !utf8.ValidString(p) || strings.ContainsAny(p, "\r\n\x00") {
		return "", fmt.Errorf("invalid Git path")
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("invalid Git path")
		}
	}
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("absolute Git path rejected")
	}
	p = filepath.ToSlash(filepath.Clean(p))
	if p == "." || p == "" || p == ".." || strings.HasPrefix(p, "../") {
		return "", fmt.Errorf("Git path escapes root")
	}
	return p, nil
}
func readPatterns(path string) ([]string, error) {
	data, err := osReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var result []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			line = strings.TrimPrefix(line, "/")
			if _, err := filepath.Match(line, ""); err != nil {
				return nil, fmt.Errorf("invalid .cidxignore pattern %q: %w", line, err)
			}
			result = append(result, line)
		}
	}
	return result, nil
}

var osReadFile = func(path string) ([]byte, error) { return os.ReadFile(path) }

func excluded(root, path string, patterns []string, max int64) (string, error) {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == ".git" || part == ".cidx" || part == "node_modules" || part == "vendor" || part == "dist" || part == "build" {
			return "built-in", nil
		}
	}
	base := filepath.Base(path)
	if base == "go.sum" || strings.HasSuffix(base, ".lock") || strings.HasSuffix(base, ".min.js") || strings.HasSuffix(base, ".min.ts") || strings.Contains(base, ".generated.") {
		return "built-in", nil
	}
	for _, p := range patterns {
		ok, _ := filepath.Match(p, path)
		if ok || strings.HasPrefix(path, strings.TrimSuffix(p, "/")+"/") {
			return ".cidxignore", nil
		}
	}
	info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path)))
	if os.IsNotExist(err) {
		return "missing", nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "symlink", nil
	}
	if !info.Mode().IsRegular() {
		return "non-regular", nil
	}
	if info.Size() > max {
		return "oversize", nil
	}
	return "", nil
}
