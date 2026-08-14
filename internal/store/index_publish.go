package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"cidx/internal/chunk"
	"cidx/internal/config"
)

// PreparedIndexFile is persistence-shaped staging data. It is complete before
// the writer transaction starts and has no parser, Git, or provider behavior.
type PreparedIndexFile struct {
	Path, Language, SHA256 string
	MtimeNS, Size          int64
	Chunks                 []PreparedIndexChunk
}
type PreparedIndexChunk struct {
	Kind, Symbol, QualifiedSymbol, Signature string
	StartByte, EndByte, StartLine, EndLine   int
	SourceBody                               []byte
	Projections                              []PreparedIndexProjection
	Segments                                 []PreparedIndexSegment
	Symbols                                  []PreparedIndexSymbol
	FTSSymbols, FTSBody                      string
}
type PreparedIndexSymbol struct{ Original, Normalized string }
type PreparedIndexProjection struct {
	Kind               string
	StartByte, EndByte int
}
type PreparedIndexSegment struct {
	Number                                                     int
	CanonicalInputSHA256, CanonicalTextProfile, ServingProfile string
	DisplayStartByte, DisplayEndByte                           int
	Projections                                                []PreparedIndexRange
}
type PreparedIndexRange struct{ StartByte, EndByte int }
type SegmentUpdate struct {
	ID                                                         int64
	CanonicalInputSHA256, CanonicalTextProfile, ServingProfile string
}
type IndexPublishPlan struct {
	BaseGeneration, NextGeneration    int64
	ManifestSHA256, Reason, GitCommit string
	GitDirty                          bool
	Desired                           config.ResolvedConfig
	Deleted                           []string
	Changed                           []PreparedIndexFile
	SegmentUpdates                    []SegmentUpdate
}

func (store *ProductionStore) PublishIndexGeneration(ctx context.Context, p IndexPublishPlan) error {
	if err := p.validate(); err != nil {
		return err
	}
	tx, err := store.Write.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var active int64
	if err := tx.QueryRowContext(ctx, `SELECT active_generation FROM meta WHERE id=1`).Scan(&active); err != nil {
		return err
	}
	if active != p.BaseGeneration {
		return fmt.Errorf("BASE_GENERATION_CHANGED")
	}
	for _, path := range p.Deleted {
		if err := deleteIndexFile(ctx, tx, path); err != nil {
			return err
		}
	}
	for _, file := range p.Changed {
		if err := deleteIndexFile(ctx, tx, file.Path); err != nil {
			return err
		}
		if err := insertIndexFile(ctx, tx, file); err != nil {
			return err
		}
	}
	for _, update := range p.SegmentUpdates {
		result, err := tx.ExecContext(ctx, `UPDATE embedding_segments SET canonical_input_sha256=?,canonical_text_profile=?,serving_profile=? WHERE id=?`, update.CanonicalInputSHA256, update.CanonicalTextProfile, update.ServingProfile, update.ID)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n != 1 {
			return fmt.Errorf("segment update missing")
		}
	}
	profiles := p.Desired.Profiles
	indexJSON, err := config.CanonicalJSON(p.Desired.Profiles.Index)
	if err != nil {
		return err
	}
	canonicalJSON, err := config.CanonicalJSON(p.Desired.Profiles.CanonicalText)
	if err != nil {
		return err
	}
	sourceJSON, err := config.CanonicalJSON(p.Desired.Profiles.Source)
	if err != nil {
		return err
	}
	spaceJSON, err := config.CanonicalJSON(p.Desired.Profiles.VectorSpace)
	if err != nil {
		return err
	}
	storageJSON, err := config.CanonicalJSON(p.Desired.Profiles.VectorStorage)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `UPDATE meta SET active_generation=?,manifest_sha256=?,index_profile=?,index_profile_json=?,canonical_text_profile=?,canonical_text_profile_json=?,source_profile=?,source_profile_json=?,vector_space_profile=?,vector_space_profile_json=?,vector_storage_profile=?,vector_storage_profile_json=?,active_serving_profile=?,index_attempted_at=?,index_succeeded_at=?,observed_git_commit=?,observed_git_dirty=? WHERE id=1`, p.NextGeneration, p.ManifestSHA256, profiles.Fingerprints.Index, indexJSON, profiles.Fingerprints.CanonicalText, canonicalJSON, profiles.Fingerprints.Source, sourceJSON, profiles.Fingerprints.VectorSpace, spaceJSON, profiles.Fingerprints.VectorStorage, storageJSON, profiles.Fingerprints.VectorStorage, now, now, p.GitCommit, boolInt(p.GitDirty)); err != nil {
		return err
	}
	run, err := tx.ExecContext(ctx, `INSERT INTO index_runs(phase,state,reason,started_at,ended_at) VALUES('05-worktree-index-pipeline','succeeded',?,?,?)`, p.Reason, now, now)
	if err != nil {
		return err
	}
	runID, err := run.LastInsertId()
	if err != nil {
		return err
	}
	for _, path := range p.Deleted {
		if _, err := tx.ExecContext(ctx, `INSERT INTO index_run_files(run_id,path,planned_action,outcome) VALUES(?,?,?,?)`, runID, path, "deleted", "succeeded"); err != nil {
			return err
		}
	}
	for _, file := range p.Changed {
		if _, err := tx.ExecContext(ctx, `INSERT INTO index_run_files(run_id,path,planned_action,outcome) VALUES(?,?,?,?)`, runID, file.Path, "updated", "succeeded"); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func deleteIndexFile(ctx context.Context, tx *sql.Tx, path string) error {
	rows, err := tx.QueryContext(ctx, `SELECT c.id,s.id FROM files f LEFT JOIN chunks c ON c.file_id=f.id LEFT JOIN embedding_segments s ON s.chunk_id=c.id WHERE f.path=?`, path)
	if err != nil {
		return err
	}
	chunkSet, segmentSet := map[int64]struct{}{}, map[int64]struct{}{}
	for rows.Next() {
		var c, s sql.NullInt64
		if err := rows.Scan(&c, &s); err != nil {
			rows.Close()
			return err
		}
		if c.Valid {
			chunkSet[c.Int64] = struct{}{}
		}
		if s.Valid {
			segmentSet[s.Int64] = struct{}{}
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	chunks, segments := sortedIDs(chunkSet), sortedIDs(segmentSet)
	for _, s := range segments {
		if _, err := tx.ExecContext(ctx, `DELETE FROM segment_projections WHERE segment_id=?`, s); err != nil {
			return err
		}
	}
	for _, c := range chunks {
		if err := deleteOldFTS(ctx, tx, c); err != nil {
			return err
		}
		for _, q := range []string{`DELETE FROM embedding_segments WHERE chunk_id=?`, `DELETE FROM chunk_projections WHERE chunk_id=?`, `DELETE FROM symbols WHERE chunk_id=?`, `DELETE FROM chunks WHERE id=?`} {
			if _, err := tx.ExecContext(ctx, q, c); err != nil {
				return err
			}
		}
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM files WHERE path=?`, path)
	return err
}
func sortedIDs(set map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
func insertIndexFile(ctx context.Context, tx *sql.Tx, f PreparedIndexFile) error {
	if f.Path == "" || f.Language == "" || f.SHA256 == "" {
		return fmt.Errorf("invalid prepared file")
	}
	r, err := tx.ExecContext(ctx, `INSERT INTO files(path,language,indexed_sha256,observed_mtime_ns,observed_size) VALUES(?,?,?,?,?)`, f.Path, f.Language, f.SHA256, f.MtimeNS, f.Size)
	if err != nil {
		return err
	}
	fid, err := r.LastInsertId()
	if err != nil {
		return err
	}
	for _, c := range f.Chunks {
		if c.Symbol == "" || c.QualifiedSymbol == "" || c.EndByte < c.StartByte || len(c.SourceBody) != c.EndByte-c.StartByte {
			return fmt.Errorf("invalid prepared chunk")
		}
		r, err := tx.ExecContext(ctx, `INSERT INTO chunks(file_id,kind,symbol,qualified_symbol,signature,start_byte,end_byte,start_line,end_line,source_body) VALUES(?,?,?,?,?,?,?,?,?,?)`, fid, c.Kind, c.Symbol, c.QualifiedSymbol, c.Signature, c.StartByte, c.EndByte, c.StartLine, c.EndLine, c.SourceBody)
		if err != nil {
			return err
		}
		cid, _ := r.LastInsertId()
		for i, p := range c.Projections {
			if _, err := tx.ExecContext(ctx, `INSERT INTO chunk_projections(chunk_id,projection_kind,ordinal,start_byte,end_byte) VALUES(?,?,?,?,?)`, cid, p.Kind, i, p.StartByte, p.EndByte); err != nil {
				return err
			}
		}
		for _, name := range c.Symbols {
			if name.Original == "" || name.Normalized == "" {
				return fmt.Errorf("invalid prepared symbol")
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO symbols(chunk_id,original_name,normalized_name) VALUES(?,?,?)`, cid, name.Original, name.Normalized); err != nil {
				return err
			}
		}
		for _, s := range c.Segments {
			r, err := tx.ExecContext(ctx, `INSERT INTO embedding_segments(chunk_id,segment_number,canonical_input_sha256,canonical_text_profile,serving_profile,display_start_byte,display_end_byte) VALUES(?,?,?,?,?,?,?)`, cid, s.Number, s.CanonicalInputSHA256, s.CanonicalTextProfile, s.ServingProfile, s.DisplayStartByte, s.DisplayEndByte)
			if err != nil {
				return err
			}
			sid, _ := r.LastInsertId()
			for i, p := range s.Projections {
				if _, err := tx.ExecContext(ctx, `INSERT INTO segment_projections(segment_id,ordinal,start_byte,end_byte) VALUES(?,?,?,?)`, sid, i, p.StartByte, p.EndByte); err != nil {
					return err
				}
			}
		}
		if c.FTSSymbols == "" || c.FTSBody == "" {
			return fmt.Errorf("prepared FTS values are required")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO chunk_fts(rowid,symbols,body) VALUES(?,?,?)`, cid, c.FTSSymbols, c.FTSBody); err != nil {
			return err
		}
	}
	return nil
}
func deleteOldFTS(ctx context.Context, tx *sql.Tx, cid int64) error {
	var signature string
	var body []byte
	if err := tx.QueryRowContext(ctx, `SELECT signature,source_body FROM chunks WHERE id=?`, cid).Scan(&signature, &body); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT original_name,normalized_name FROM symbols WHERE chunk_id=? ORDER BY rowid`, cid)
	if err != nil {
		return err
	}
	var symbols []string
	for rows.Next() {
		var a, b string
		if err := rows.Scan(&a, &b); err != nil {
			rows.Close()
			return err
		}
		symbols = append(symbols, a, b)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	p, err := tx.QueryContext(ctx, `SELECT start_byte,end_byte FROM chunk_projections WHERE chunk_id=? ORDER BY ordinal`, cid)
	if err != nil {
		return err
	}
	var parts []string
	for p.Next() {
		var a, b int
		if err := p.Scan(&a, &b); err != nil {
			p.Close()
			return err
		}
		if a < 0 || b <= a || b > len(body) {
			p.Close()
			return fmt.Errorf("invalid stored projection")
		}
		parts = append(parts, string(body[a:b]))
	}
	if err := p.Close(); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO chunk_fts(chunk_fts,rowid,symbols,body) VALUES('delete',?,?,?)`, cid, strings.Join(symbols, " "), signature+"\n"+strings.Join(parts, "\n"))
	return err
}
func (p IndexPublishPlan) validate() error {
	resolved := p.Desired
	if err := config.Validate(&resolved); err != nil {
		return fmt.Errorf("invalid desired config: %w", err)
	}
	profiles, err := config.FingerprintProfiles(resolved)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(profiles, p.Desired.Profiles) {
		return fmt.Errorf("forged desired profiles")
	}
	if p.BaseGeneration < 0 || p.NextGeneration <= 0 || p.NextGeneration != p.BaseGeneration+1 || !hexHash(p.ManifestSHA256) || p.Reason == "" {
		return fmt.Errorf("invalid index publish plan")
	}
	seen := map[string]struct{}{}
	for _, path := range p.Deleted {
		if !safePath(path) {
			return fmt.Errorf("invalid deleted path")
		}
		if _, ok := seen[path]; ok {
			return fmt.Errorf("duplicate publish path")
		}
		seen[path] = struct{}{}
	}
	for _, f := range p.Changed {
		if !safePath(f.Path) || !chunk.Language(f.Language).Valid() || !hexHash(f.SHA256) || f.Size < 0 {
			return fmt.Errorf("invalid prepared file")
		}
		if _, ok := seen[f.Path]; ok {
			return fmt.Errorf("duplicate publish path")
		}
		seen[f.Path] = struct{}{}
		for _, c := range f.Chunks {
			if !chunk.ChunkKind(c.Kind).Valid() || c.Symbol == "" || c.QualifiedSymbol == "" || c.Signature == "" || len(c.Projections) == 0 || len(c.Segments) == 0 || len(c.Symbols) == 0 || c.StartByte < 0 || c.EndByte-c.StartByte != len(c.SourceBody) || c.StartLine < 1 || c.EndLine < c.StartLine || c.FTSSymbols == "" || c.FTSBody == "" {
				return fmt.Errorf("invalid prepared chunk")
			}
			last := 0
			for i, r := range c.Projections {
				if (r.Kind != string(chunk.ProjectionSignature) && r.Kind != string(chunk.ProjectionBody)) || r.StartByte < 0 || r.EndByte <= r.StartByte || r.EndByte > len(c.SourceBody) || (i > 0 && r.StartByte < last) {
					return fmt.Errorf("invalid chunk projection")
				}
				last = r.EndByte
			}
			symbols := map[string]struct{}{}
			for _, symbol := range c.Symbols {
				if symbol.Original == "" || symbol.Normalized == "" {
					return fmt.Errorf("invalid prepared symbol")
				}
				if _, ok := symbols[symbol.Original]; ok {
					return fmt.Errorf("duplicate prepared symbol")
				}
				symbols[symbol.Original] = struct{}{}
			}
			for i, s := range c.Segments {
				if len(s.Projections) == 0 || s.Number != i || !hexHash(s.CanonicalInputSHA256) || s.CanonicalTextProfile != string(p.Desired.Profiles.Fingerprints.CanonicalText) || s.ServingProfile != string(p.Desired.Profiles.Fingerprints.VectorStorage) || s.DisplayStartByte < 0 || s.DisplayEndByte < s.DisplayStartByte || s.DisplayEndByte > len(c.SourceBody) {
					return fmt.Errorf("invalid prepared segment")
				}
				last = 0
				for j, r := range s.Projections {
					if r.StartByte < 0 || r.EndByte <= r.StartByte || r.EndByte > len(c.SourceBody) || (j > 0 && r.StartByte < last) || r.StartByte < s.DisplayStartByte || r.EndByte > s.DisplayEndByte {
						return fmt.Errorf("invalid segment projection")
					}
					last = r.EndByte
				}
			}
		}
	}
	updates := map[int64]struct{}{}
	for _, u := range p.SegmentUpdates {
		if u.ID <= 0 || !hexHash(u.CanonicalInputSHA256) || u.CanonicalTextProfile != string(p.Desired.Profiles.Fingerprints.CanonicalText) || u.ServingProfile != string(p.Desired.Profiles.Fingerprints.VectorStorage) {
			return fmt.Errorf("invalid segment update")
		}
		if _, ok := updates[u.ID]; ok {
			return fmt.Errorf("duplicate segment update")
		}
		updates[u.ID] = struct{}{}
	}
	return nil
}
func hexHash(v string) bool {
	if len(v) != 64 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil
}
func safePath(v string) bool {
	return v != "" && !strings.ContainsAny(v, "\r\n\x00") && !strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "../") && !strings.Contains(v, "/../")
}
func fixtureHash(v string) string { sum := sha256.Sum256([]byte(v)); return hex.EncodeToString(sum[:]) }
func SortPreparedFiles(files []PreparedIndexFile) {
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
}
