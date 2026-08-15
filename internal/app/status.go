package app

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"cidx/internal/chunk"
	"cidx/internal/config"
	"cidx/internal/ignore"
	"cidx/internal/store"
)

type StatusResponse struct {
	Desired                       config.AppliedProfiles `json:"desired"`
	Applied                       config.AppliedProfiles `json:"applied"`
	ObservedGeneration            int64                  `json:"observed_generation"`
	ManifestSHA256                string                 `json:"manifest_sha256"`
	Files                         int                    `json:"file_count"`
	Chunks                        int                    `json:"chunk_count"`
	Segments                      int                    `json:"segment_count"`
	Dirty                         bool                   `json:"dirty"`
	Stale                         int                    `json:"stale_count"`
	Unindexed                     int                    `json:"unindexed_count"`
	Deleted                       int                    `json:"deleted_count"`
	IndexError                    int                    `json:"index_error_count"`
	Ready                         int                    `json:"ready_count"`
	Pending                       int                    `json:"pending_count"`
	Failed                        int                    `json:"failed_count"`
	CoverageReady                 int                    `json:"vector_coverage_numerator"`
	CoverageTotal                 int                    `json:"vector_coverage_denominator"`
	IndexAttemptedAt              string                 `json:"index_attempted_at"`
	IndexSucceededAt              string                 `json:"index_succeeded_at"`
	EmbedAttemptedAt              string                 `json:"embed_attempted_at"`
	EmbedSucceededAt              string                 `json:"embed_succeeded_at"`
	GenerationChangedDuringStatus bool                   `json:"generation_changed_during_status"`
}
type StatusService struct {
	Root     string
	Resolved config.ResolvedConfig
	Store    *store.ProductionStore
}

func (service StatusService) Get(ctx context.Context) (StatusResponse, error) {
	if service.Store == nil {
		return StatusResponse{}, fmt.Errorf("production store is required")
	}
	snapshot, err := service.Store.StatusSnapshot(ctx, service.Resolved)
	if err != nil {
		return StatusResponse{}, err
	}
	response := StatusResponse{Desired: config.AppliedProfiles{Fingerprints: service.Resolved.Profiles.Fingerprints, ActiveServingProfile: service.Resolved.Profiles.Fingerprints.VectorStorage}, Applied: snapshot.Applied, ObservedGeneration: snapshot.Applied.ActiveGeneration, ManifestSHA256: snapshot.Applied.ManifestSHA256, Files: snapshot.Files, Chunks: snapshot.Chunks, Segments: snapshot.Segments, Ready: snapshot.Ready, Pending: snapshot.Pending, Failed: snapshot.Failed, CoverageReady: snapshot.CoverageReady, CoverageTotal: snapshot.CoverageTotal, IndexAttemptedAt: snapshot.IndexAttemptedAt, IndexSucceededAt: snapshot.IndexSucceededAt, EmbedAttemptedAt: snapshot.EmbedAttemptedAt, EmbedSucceededAt: snapshot.EmbedSucceededAt}
	candidates, err := ignore.Enumerate(ctx, service.Root, int64(service.Resolved.Index.MaxSourceFileBytes))
	if err != nil {
		return response, err
	}
	live := map[string]bool{}
	for _, candidate := range candidates {
		if candidate.Exclusion != "" {
			continue
		}
		if !enabledStatusLanguage(candidate.Path, service.Resolved.Index.Languages) {
			continue
		}
		live[candidate.Path] = true
		stored, exists := snapshot.FilesByPath[candidate.Path]
		if !exists {
			response.Unindexed++
			continue
		}
		body, err := readRegularNoSymlink(service.Root, candidate.Path, service.Resolved.Index.MaxSourceFileBytes)
		if err != nil {
			response.IndexError++
			continue
		}
		if fmt.Sprintf("%x", sha256.Sum256(body)) != stored.SHA256 {
			response.Stale++
		}
	}
	for path := range snapshot.FilesByPath {
		if !live[path] {
			response.Deleted++
		}
	}
	dirty, err := worktreeDirty(ctx, service.Root)
	if err != nil {
		return response, err
	}
	response.Dirty = dirty
	current, err := service.Store.AppliedProfiles(ctx)
	if err != nil {
		return response, err
	}
	response.GenerationChangedDuringStatus = current.ActiveGeneration != response.ObservedGeneration
	return response, nil
}

func worktreeDirty(ctx context.Context, root string) (bool, error) {
	output, err := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain", "-z", "--", ".", ":(exclude).cidx/index.db", ":(exclude).cidx/index.db-wal", ":(exclude).cidx/index.db-shm", ":(exclude).cidx/index.lock", ":(exclude).cidx/embed.lock").Output()
	if err != nil {
		return false, fmt.Errorf("observe Git worktree: %w", err)
	}
	return len(output) > 0, nil
}

func enabledStatusLanguage(path string, languages []chunk.Language) bool {
	extension := strings.ToLower(filepath.Ext(path))
	for _, language := range languages {
		switch language {
		case chunk.Go:
			if extension == ".go" {
				return true
			}
		case chunk.TypeScript:
			if extension == ".ts" {
				return true
			}
		case chunk.TSX:
			if extension == ".tsx" {
				return true
			}
		}
	}
	return false
}
