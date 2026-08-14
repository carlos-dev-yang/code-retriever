package eval

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// CorpusBindings is intentionally local-only. Callers must keep its file
// below ignored state (normally .cidx/lab/corpora.local.json).
type CorpusBindings struct {
	Bindings map[string]string `json:"bindings"`
}

// LoadIgnoredCorpusBindings accepts only the conventional ignored local file.
// This prevents accidentally treating a tracked checkout path as portable
// corpus provenance.
func LoadIgnoredCorpusBindings(ctx context.Context, repositoryRoot string) (CorpusBindings, error) {
	root, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return CorpusBindings{}, err
	}
	path := filepath.Join(root, ".cidx", "lab", "corpora.local.json")
	relative := ".cidx/lab/corpora.local.json"
	command := exec.CommandContext(ctx, "git", "-C", root, "check-ignore", "--quiet", "--", relative)
	if err := command.Run(); err != nil {
		return CorpusBindings{}, fmt.Errorf("local corpus binding file must be ignored")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return CorpusBindings{}, err
	}
	return LoadCorpusBindings(data)
}

func LoadCorpusBindings(data []byte) (CorpusBindings, error) {
	var value CorpusBindings
	if err := decodeStrict(data, &value); err != nil {
		return CorpusBindings{}, fmt.Errorf("decode corpus bindings: %w", err)
	}
	if len(value.Bindings) == 0 {
		return CorpusBindings{}, fmt.Errorf("empty corpus bindings")
	}
	for id, checkout := range value.Bindings {
		if !validID(id) || checkout == "" {
			return CorpusBindings{}, fmt.Errorf("invalid corpus binding")
		}
	}
	return value, nil
}

// ResolveCheckout gives an explicit local input precedence over an ignored
// binding. It does not record that path in any portable value.
func ResolveCheckout(manifest CorpusManifest, bindings CorpusBindings, explicit string) (string, error) {
	if err := manifest.Validate(); err != nil {
		return "", err
	}
	requested := explicit
	if requested == "" {
		requested = bindings.Bindings[manifest.CorpusID]
	}
	if requested == "" {
		return "", fmt.Errorf("missing local binding for corpus %q", manifest.CorpusID)
	}
	abs, err := filepath.Abs(requested)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("canonicalize checkout: %w", err)
	}
	info, err := os.Stat(canonical)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("checkout is not a directory")
	}
	return canonical, nil
}

type VerifiedCorpus struct {
	CorpusID      string `json:"corpus_id"`
	PinnedCommit  string `json:"pinned_commit"`
	ContentSHA256 string `json:"content_sha256"`
	Clean         bool   `json:"clean"`
}

// VerifyCheckout is a read-only local Git and file-system verification. It
// never fetches, changes refs, alters files, or contacts a remote.
func VerifyCheckout(ctx context.Context, manifest CorpusManifest, checkout string) (VerifiedCorpus, error) {
	if err := manifest.Validate(); err != nil {
		return VerifiedCorpus{}, err
	}
	root, err := ResolveCheckout(manifest, CorpusBindings{}, checkout)
	if err != nil {
		return VerifiedCorpus{}, err
	}
	gitRoot, err := git(ctx, root, "rev-parse", "--show-toplevel")
	if err != nil {
		return VerifiedCorpus{}, fmt.Errorf("verify Git checkout: %w", err)
	}
	canonicalGitRoot, err := filepath.EvalSymlinks(strings.TrimSpace(gitRoot))
	if err != nil || canonicalGitRoot != root {
		return VerifiedCorpus{}, fmt.Errorf("checkout is not its Git worktree root")
	}
	origin, err := git(ctx, root, "remote", "get-url", "origin")
	if err != nil {
		return VerifiedCorpus{}, fmt.Errorf("read Git origin: %w", err)
	}
	if normalizeRemote(strings.TrimSpace(origin)) != normalizeRemote(manifest.UpstreamURL) {
		return VerifiedCorpus{}, fmt.Errorf("checkout origin does not match manifest")
	}
	head, err := git(ctx, root, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(head) != manifest.PinnedCommit {
		return VerifiedCorpus{}, fmt.Errorf("checkout commit does not match manifest")
	}
	status, err := git(ctx, root, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil || strings.TrimSpace(status) != "" {
		return VerifiedCorpus{}, fmt.Errorf("checkout is dirty")
	}
	if err := verifyLicense(root, manifest.LicenseEvidence); err != nil {
		return VerifiedCorpus{}, err
	}
	tree, err := git(ctx, root, "rev-parse", "HEAD^{tree}")
	if err != nil || strings.TrimSpace(tree) != manifest.ExpectedTreeHash {
		return VerifiedCorpus{}, fmt.Errorf("checkout tree does not match manifest")
	}
	content, err := selectedContentHash(ctx, root, manifest)
	if err != nil {
		return VerifiedCorpus{}, err
	}
	if content != manifest.ExpectedContentSHA256 {
		return VerifiedCorpus{}, fmt.Errorf("selected content hash does not match manifest")
	}
	return VerifiedCorpus{CorpusID: manifest.CorpusID, PinnedCommit: manifest.PinnedCommit, ContentSHA256: content, Clean: true}, nil
}

func git(ctx context.Context, root string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...).Output()
	return string(out), err
}
func normalizeRemote(value string) string {
	value = strings.TrimSuffix(strings.TrimSpace(value), "/")
	return strings.TrimSuffix(value, ".git")
}

func verifyLicense(root, evidence string) error {
	if err := ensureNoSymlinkPath(root, evidence); err != nil {
		return err
	}
	info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(evidence)))
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("checkout lacks manifest license evidence")
	}
	return nil
}

type contentEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func selectedContentHash(ctx context.Context, root string, manifest CorpusManifest) (string, error) {
	rootSlice := filepath.Join(root, filepath.FromSlash(manifest.RootSubdir))
	if err := ensureNoSymlinkPath(root, manifest.RootSubdir); err != nil {
		return "", err
	}
	info, err := os.Stat(rootSlice)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("root subdirectory does not exist")
	}
	output, err := git(ctx, root, "ls-files", "-z", "--cached")
	if err != nil {
		return "", err
	}
	var entries []contentEntry
	found := map[string]bool{}
	for _, raw := range bytes.Split([]byte(output), []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		file := string(raw)
		if !validRelative(file, false) {
			return "", fmt.Errorf("unsafe Git file path")
		}
		if manifest.RootSubdir != "" && file != manifest.RootSubdir && !strings.HasPrefix(file, manifest.RootSubdir+"/") {
			continue
		}
		selected := strings.TrimPrefix(file, manifest.RootSubdir)
		selected = strings.TrimPrefix(selected, "/")
		if !matchesPolicy(selected, manifest.Include, manifest.Exclude) {
			continue
		}
		if err := ensureNoSymlinkPath(root, file); err != nil {
			return "", err
		}
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil || !info.Mode().IsRegular() {
			return "", fmt.Errorf("selected file is not regular")
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		entries = append(entries, contentEntry{Path: file, SHA256: hex.EncodeToString(sum[:])})
		ext := strings.ToLower(filepath.Ext(file))
		if ext == ".go" {
			found["go"] = true
		} else if ext == ".ts" {
			found["typescript"] = true
		} else if ext == ".tsx" {
			found["tsx"] = true
		}
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("manifest selected no files")
	}
	for _, language := range manifest.LanguageSlices {
		if language == "mixed" {
			continue
		}
		if !found[string(language)] {
			return "", fmt.Errorf("selected files lack declared %s slice", language)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	canonical, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func ensureNoSymlinkPath(root, relative string) error {
	current := root
	for _, part := range strings.Split(relative, "/") {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinked checkout path rejected")
		}
	}
	return nil
}

func matchesPolicy(file string, include, exclude []string) bool {
	for _, p := range exclude {
		if globMatch(p, file) {
			return false
		}
	}
	for _, p := range include {
		if globMatch(p, file) {
			return true
		}
	}
	return false
}
func globMatch(pattern, value string) bool {
	patternSegments, err := compileGlob(pattern)
	if err != nil || !validRelative(value, false) {
		return false
	}
	valueSegments := strings.Split(value, "/")
	memo := map[[2]int]bool{}
	seen := map[[2]int]bool{}
	var matches func(int, int) bool
	matches = func(patternIndex, valueIndex int) bool {
		key := [2]int{patternIndex, valueIndex}
		if seen[key] {
			return memo[key]
		}
		seen[key] = true
		result := false
		if patternIndex == len(patternSegments) {
			result = valueIndex == len(valueSegments)
			memo[key] = result
			return result
		}
		if patternSegments[patternIndex] == "**" {
			for next := valueIndex; next <= len(valueSegments); next++ {
				if matches(patternIndex+1, next) {
					memo[key] = true
					return true
				}
			}
			memo[key] = false
			return false
		}
		if valueIndex == len(valueSegments) {
			memo[key] = false
			return false
		}
		matched, err := path.Match(patternSegments[patternIndex], valueSegments[valueIndex])
		result = err == nil && matched && matches(patternIndex+1, valueIndex+1)
		memo[key] = result
		return result
	}
	return matches(0, 0)
}
