package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"cidx/internal/chunk"
	"cidx/internal/chunk/golang"
	"cidx/internal/chunk/typescript"
	"cidx/internal/config"
	"cidx/internal/ignore"
	"cidx/internal/index/canonicaltext"
	"cidx/internal/root"
	"cidx/internal/store"
	"cidx/internal/symbol"
)

type Reason string

const (
	ReasonManual Reason = "manual"
	ReasonCommit Reason = "commit"
	ReasonMCP    Reason = "mcp"
)

type Request struct {
	Root   string
	DryRun bool
	Reason Reason
	Config config.ResolvedConfig
}
type Plan struct {
	ObservedGeneration              int64
	Added, Changed, Reused, Deleted []string
	FullRebuildRequired             bool
	ReconcileRequired               bool
	CanonicalReconcile              bool
	Candidates                      []string
}
type Result struct {
	DryRun                   bool         `json:"dry_run"`
	ActivatedGeneration      int64        `json:"activated_generation,omitempty"`
	ManifestSHA256           string       `json:"manifest_sha256"`
	Scanned                  int          `json:"files_scanned"`
	Updated                  int          `json:"files_updated"`
	Reused                   int          `json:"files_reused"`
	Deleted                  int          `json:"files_deleted"`
	Chunks                   int          `json:"chunks_updated"`
	Segments                 int          `json:"segments_updated"`
	PlannedEmbeddingsReused  int          `json:"planned_embeddings_reused"`
	PlannedEmbeddingsPending int          `json:"planned_embeddings_pending"`
	Diagnostics              []Diagnostic `json:"diagnostics"`
}
type Diagnostic struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Safe    bool   `json:"safe"`
}
type Service struct {
	Store    *store.ProductionStore
	chunkers map[chunk.Language]chunk.Chunker
}

func New(store *store.ProductionStore) *Service {
	return &Service{Store: store, chunkers: map[chunk.Language]chunk.Chunker{chunk.Go: golang.New(), chunk.TypeScript: typescript.New(chunk.TypeScript), chunk.TSX: typescript.New(chunk.TSX)}}
}

func (s *Service) Plan(ctx context.Context, r Request) (Plan, error) {
	if !validReason(r.Reason) {
		return Plan{}, fmt.Errorf("invalid index reason %q", r.Reason)
	}
	canonical, err := root.Repository(ctx, r.Root)
	if err != nil {
		return Plan{}, err
	}
	if s.Store == nil || s.Store.Root != canonical {
		return Plan{}, fmt.Errorf("production store root does not match request root")
	}
	snapshot, err := s.Store.IndexSnapshot(ctx)
	if err != nil {
		return Plan{}, err
	}
	candidates, err := ignore.Enumerate(ctx, canonical, int64(r.Config.Index.MaxSourceFileBytes))
	if err != nil {
		return Plan{}, err
	}
	p := Plan{ObservedGeneration: snapshot.Applied.ActiveGeneration}
	live := map[string]struct{}{}
	enabled := map[chunk.Language]bool{}
	for _, l := range r.Config.Index.Languages {
		enabled[l] = true
	}
	for _, candidate := range candidates {
		if candidate.Exclusion != "" {
			continue
		}
		l, ok := language(candidate.Path)
		if !ok || !enabled[l] {
			continue
		}
		live[candidate.Path] = struct{}{}
		p.Candidates = append(p.Candidates, candidate.Path)
		if _, ok := snapshot.Files[candidate.Path]; ok {
			p.Reused = append(p.Reused, candidate.Path)
		} else {
			p.Added = append(p.Added, candidate.Path)
		}
	}
	for path := range snapshot.Files {
		if _, ok := live[path]; !ok {
			p.Deleted = append(p.Deleted, path)
		}
	}
	impact := config.PlanImpact(r.Config.Profiles, snapshot.Applied, store.ProductionSchemaVersion)
	p.ReconcileRequired = impact.Class != config.ImpactNone && impact.Class != config.ImpactRestartOnly
	p.CanonicalReconcile = impact.RequiresCanonicalReconciliation
	p.FullRebuildRequired = impact.Class == config.ImpactLocalReindex && !p.CanonicalReconcile
	if p.FullRebuildRequired {
		p.Changed = append(p.Changed, p.Reused...)
		p.Reused = nil
	}
	sort.Strings(p.Candidates)
	sort.Strings(p.Added)
	sort.Strings(p.Changed)
	sort.Strings(p.Reused)
	sort.Strings(p.Deleted)
	return p, nil
}

func (s *Service) Execute(ctx context.Context, r Request) (Result, error) {
	if !validReason(r.Reason) {
		return Result{}, fmt.Errorf("invalid index reason %q", r.Reason)
	}
	canonical, err := root.Repository(ctx, r.Root)
	if err != nil {
		return Result{}, err
	}
	if s.Store == nil || s.Store.Root != canonical {
		return Result{}, fmt.Errorf("production store root does not match request root")
	}
	release, err := acquireLock(ctx, filepath.Join(canonical, ".cidx", "index.lock"))
	if err != nil {
		return Result{}, err
	}
	defer release()
	// The locked plan and snapshot are authoritative; Plan remains a preliminary
	// inspection API and never supplies a publication base.
	p, err := s.Plan(ctx, Request{Root: canonical, DryRun: r.DryRun, Reason: r.Reason, Config: r.Config})
	if err != nil {
		return Result{}, err
	}
	result := Result{DryRun: r.DryRun, Scanned: len(p.Candidates), Deleted: len(p.Deleted)}
	snapshot, err := s.Store.IndexSnapshot(ctx)
	if err != nil {
		return result, err
	}
	var prepared []store.PreparedIndexFile
	changed := map[string]bool{}
	for _, v := range p.Added {
		changed[v] = true
	}
	for _, v := range p.Changed {
		changed[v] = true
	}
	for _, v := range p.Deleted {
		changed[v] = true
	}
	for _, path := range p.Candidates {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		body, info, err := readOne(canonical, path, r.Config.Index.MaxSourceFileBytes)
		if err != nil {
			return result, fmt.Errorf("%s: %w", path, err)
		}
		if len(body) > r.Config.Index.MaxSourceFileBytes {
			return result, fmt.Errorf("%s: source file exceeds configured limit", path)
		}
		hash := sha(body)
		old, exists := snapshot.Files[path]
		if exists && !p.FullRebuildRequired && old.SHA256 == hash {
			result.Reused++
			continue
		}
		changed[path] = true
		file, err := s.prepare(ctx, r.Config, path, body, info, hash)
		if err != nil {
			return result, err
		}
		prepared = append(prepared, file)
		result.Updated++
		result.Chunks += len(file.Chunks)
		for _, c := range file.Chunks {
			result.Segments += len(c.Segments)
		}
	}
	noChanges := len(changed) == 0 && len(p.Deleted) == 0 && !p.ReconcileRequired
	manifest := manifestFor(p.Candidates, prepared, snapshot.Files)
	if noChanges {
		manifest = snapshot.Applied.ManifestSHA256
	}
	var updates []store.SegmentUpdate
	if p.ReconcileRequired && !p.FullRebuildRequired {
		segments, err := s.Store.ReconciliationSegments(ctx, snapshot.Applied.ActiveGeneration)
		if err != nil {
			return result, err
		}
		for _, segment := range segments {
			if changed[segment.Path] {
				continue
			}
			hash := segment.CanonicalInputSHA256
			if p.CanonicalReconcile {
				var err error
				hash, err = canonicalStored(segment)
				if err != nil {
					return result, err
				}
			}
			updates = append(updates, store.SegmentUpdate{ID: segment.ID, CanonicalInputSHA256: hash, CanonicalTextProfile: string(r.Config.Profiles.Fingerprints.CanonicalText), ServingProfile: string(r.Config.Profiles.Fingerprints.VectorStorage)})
		}
	}
	result.ManifestSHA256 = manifest
	if r.DryRun {
		segments, err := s.Store.ReconciliationSegments(ctx, snapshot.Applied.ActiveGeneration)
		if err != nil {
			return result, err
		}
		updateByID := map[int64]string{}
		for _, update := range updates {
			updateByID[update.ID] = update.CanonicalInputSHA256
		}
		keys := map[string]bool{}
		for _, segment := range segments {
			if !changed[segment.Path] {
				if hash, ok := updateByID[segment.ID]; ok {
					keys[hash] = true
				} else {
					keys[segment.CanonicalInputSHA256] = true
				}
			}
		}
		for _, file := range prepared {
			for _, chunk := range file.Chunks {
				for _, segment := range chunk.Segments {
					keys[segment.CanonicalInputSHA256] = true
				}
			}
		}
		hashes := make([]string, 0, len(keys))
		for hash := range keys {
			hashes = append(hashes, hash)
		}
		states, err := s.Store.DesiredEmbeddingStates(ctx, r.Config, hashes)
		if err != nil {
			return result, err
		}
		for _, state := range states {
			if state == store.EmbeddingReady {
				result.PlannedEmbeddingsReused++
			} else {
				result.PlannedEmbeddingsPending++
			}
		}
	}
	if r.DryRun {
		return result, nil
	}
	if noChanges {
		return result, nil
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	store.SortPreparedFiles(prepared)
	commit, dirty, err := gitObservation(ctx, canonical)
	if err != nil {
		return result, err
	}
	err = s.Store.PublishIndexGeneration(ctx, store.IndexPublishPlan{BaseGeneration: snapshot.Applied.ActiveGeneration, NextGeneration: snapshot.Applied.ActiveGeneration + 1, ManifestSHA256: manifest, Reason: string(r.Reason), GitCommit: commit, GitDirty: dirty, Desired: r.Config, Deleted: p.Deleted, Changed: prepared, SegmentUpdates: updates})
	if err != nil {
		return result, err
	}
	result.ActivatedGeneration = snapshot.Applied.ActiveGeneration + 1
	return result, nil
}
func canonicalStored(s store.IndexedSegment) (string, error) {
	parts := make([][]byte, 0, len(s.Projections))
	for _, p := range s.Projections {
		if p.StartByte < 0 || p.EndByte > len(s.SourceBody) || p.EndByte <= p.StartByte {
			return "", fmt.Errorf("invalid stored segment projection")
		}
		parts = append(parts, s.SourceBody[p.StartByte:p.EndByte])
	}
	b, err := canonicaltext.Format(canonicaltext.Input{Path: s.Path, Kind: s.Kind, QualifiedSymbol: s.QualifiedSymbol, Signature: s.Signature, BodyParts: parts})
	if err != nil {
		return "", err
	}
	return config.CanonicalInputSHA256(b), nil
}
func validReason(r Reason) bool { return r == ReasonManual || r == ReasonCommit || r == ReasonMCP }
func readOne(rootPath, path string, maxBytes int) ([]byte, os.FileInfo, error) {
	if filepath.IsAbs(path) || strings.HasPrefix(filepath.ToSlash(filepath.Clean(path)), "../") {
		return nil, nil, fmt.Errorf("path escapes root")
	}
	current := rootPath
	for _, part := range strings.Split(filepath.ToSlash(path), "/")[:len(strings.Split(filepath.ToSlash(path), "/"))-1] {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, nil, fmt.Errorf("unsafe parent")
		}
	}
	full := filepath.Join(rootPath, filepath.FromSlash(path))
	before, err := os.Lstat(full)
	if err != nil {
		return nil, nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("unsafe file type")
	}
	rootHandle, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, nil, err
	}
	defer rootHandle.Close()
	file, err := rootHandle.Open(filepath.ToSlash(path))
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !os.SameFile(before, after) {
		return nil, nil, fmt.Errorf("file changed while opening")
	}
	body, err := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err != nil {
		return nil, nil, err
	}
	if len(body) > maxBytes {
		return nil, nil, fmt.Errorf("source file exceeds configured limit")
	}
	return body, after, nil
}
func (s *Service) prepare(ctx context.Context, cfg config.ResolvedConfig, path string, body []byte, info os.FileInfo, hash string) (store.PreparedIndexFile, error) {
	if !utf8.Valid(body) {
		return store.PreparedIndexFile{}, fmt.Errorf("%s: invalid UTF-8", path)
	}
	language, ok := language(path)
	if !ok {
		return store.PreparedIndexFile{}, fmt.Errorf("unsupported source path %s", path)
	}
	worker := s.chunkers[language]
	if worker == nil {
		return store.PreparedIndexFile{}, fmt.Errorf("missing %s chunker", language)
	}
	out, err := worker.Chunk(ctx, chunk.ChunkRequest{Path: path, Source: body, SegmentationPolicy: chunk.SegmentationPolicy{Version: 1, BoundaryPolicyID: "ast-boundaries-v1", MaxSegmentBytes: cfg.Index.MaxSegmentInputBytes}})
	if err != nil {
		return store.PreparedIndexFile{}, err
	}
	for _, d := range out.Diagnostics {
		if !d.SafeToIndex {
			return store.PreparedIndexFile{}, fmt.Errorf("%s: unsafe parser diagnostic %s", path, d.Code)
		}
	}
	f := store.PreparedIndexFile{Path: path, Language: string(language), SHA256: hash, MtimeNS: info.ModTime().UnixNano(), Size: info.Size()}
	normalizer := symbol.IdentifierNormalizer{}
	for _, source := range out.Chunks {
		c := store.PreparedIndexChunk{Kind: string(source.Kind), Symbol: source.Symbol, QualifiedSymbol: source.QualifiedSymbol, Signature: source.Signature, StartByte: source.SourceRange.Start, EndByte: source.SourceRange.End, StartLine: source.LineRange.Start, EndLine: source.LineRange.End, SourceBody: append([]byte(nil), source.SourceBody...)}
		c.Symbols = []store.PreparedIndexSymbol{{Original: source.Symbol, Normalized: normalizer.Normalize(source.Symbol)}, {Original: source.QualifiedSymbol, Normalized: normalizer.Normalize(source.QualifiedSymbol)}}
		for _, p := range source.Projections {
			c.Projections = append(c.Projections, store.PreparedIndexProjection{Kind: string(p.Kind), StartByte: p.Start, EndByte: p.End})
		}
		var ftsParts []string
		for _, p := range source.Projections {
			ftsParts = append(ftsParts, string(source.SourceBody[p.Start:p.End]))
		}
		c.FTSSymbols = source.Symbol + " " + normalizer.Normalize(source.Symbol) + " " + source.QualifiedSymbol + " " + normalizer.Normalize(source.QualifiedSymbol)
		c.FTSBody = source.Signature + "\n" + strings.Join(ftsParts, "\n")
		for _, segment := range source.Segments {
			canonical, err := canonicalInput(path, source, segment)
			if err != nil {
				return store.PreparedIndexFile{}, err
			}
			s := store.PreparedIndexSegment{Number: segment.Number, CanonicalInputSHA256: config.CanonicalInputSHA256(canonical), CanonicalTextProfile: string(cfg.Profiles.Fingerprints.CanonicalText), ServingProfile: string(cfg.Profiles.Fingerprints.VectorStorage), DisplayStartByte: segment.DisplayRange.Start, DisplayEndByte: segment.DisplayRange.End}
			for _, p := range segment.Projections {
				s.Projections = append(s.Projections, store.PreparedIndexRange{StartByte: p.Start, EndByte: p.End})
			}
			c.Segments = append(c.Segments, s)
		}
		f.Chunks = append(f.Chunks, c)
	}
	return f, nil
}
func language(path string) (chunk.Language, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return chunk.Go, true
	case ".ts":
		return chunk.TypeScript, true
	case ".tsx":
		return chunk.TSX, true
	}
	return "", false
}
func sha(body []byte) string { v := sha256.Sum256(body); return hex.EncodeToString(v[:]) }
func canonicalInput(path string, c chunk.SourceChunk, s chunk.SegmentCandidate) ([]byte, error) {
	parts := make([][]byte, 0, len(s.Projections))
	for _, p := range s.Projections {
		parts = append(parts, c.SourceBody[p.Start:p.End])
	}
	return canonicaltext.Format(canonicaltext.Input{Path: path, Kind: string(c.Kind), QualifiedSymbol: c.QualifiedSymbol, Signature: c.Signature, BodyParts: parts})
}
func manifestFor(paths []string, changed []store.PreparedIndexFile, old map[string]store.IndexedFile) string {
	hashes := map[string]string{}
	for p, v := range old {
		hashes[p] = v.SHA256
	}
	for _, v := range changed {
		hashes[v.Path] = v.SHA256
	}
	var rows []string
	for _, p := range paths {
		rows = append(rows, p+"\x00"+hashes[p])
	}
	sort.Strings(rows)
	return sha([]byte(strings.Join(rows, "\n")))
}
func gitObservation(ctx context.Context, root string) (string, bool, error) {
	head, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--verify", "-q", "HEAD").Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
			return "", false, fmt.Errorf("observe Git HEAD: %w", err)
		}
	}
	dirty, err := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain", "-z", "--", ".", ":(exclude).cidx/index.db", ":(exclude).cidx/index.db-wal", ":(exclude).cidx/index.db-shm", ":(exclude).cidx/index.lock", ":(exclude).cidx/embed.lock").Output()
	if err != nil {
		return "", false, fmt.Errorf("observe Git worktree: %w", err)
	}
	return strings.TrimSpace(string(head)), len(dirty) > 0, nil
}
