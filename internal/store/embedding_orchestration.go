package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cidx/internal/config"
	"cidx/internal/vector"
)

var ErrEmbeddingStateChanged = fmt.Errorf("EMBEDDING_STATE_CHANGED")

// EmbeddingPlanningSegment is copied from one pinned active snapshot. Its
// source fields are reconstructed by the application into canonical input;
// production storage never persists canonical input bodies separately.
type EmbeddingPlanningSegment struct {
	ID                                                           int64
	Path, Kind, QualifiedSymbol, Signature, CanonicalInputSHA256 string
	SourceBody                                                   []byte
	Number                                                       int
	Projections                                                  []PreparedIndexRange
	State                                                        EmbeddingState
}

type EmbeddingPlanningSnapshot struct {
	Applied  config.AppliedProfiles
	Segments []EmbeddingPlanningSegment
}

// EmbeddingPlanningSnapshot derives generation, manifest, profiles, active
// keys, vector validity, and the latest failure class in one read snapshot.
func (store *ProductionStore) EmbeddingPlanningSnapshot(ctx context.Context, resolved config.ResolvedConfig) (EmbeddingPlanningSnapshot, error) {
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return EmbeddingPlanningSnapshot{}, err
	}
	defer tx.Rollback()
	if err := requireResolvedActiveProfile(ctx, tx, resolved); err != nil {
		return EmbeddingPlanningSnapshot{}, err
	}
	var out EmbeddingPlanningSnapshot
	var index, canonical, source, space, storage, serving string
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,active_generation,manifest_sha256,index_profile,canonical_text_profile,source_profile,vector_space_profile,vector_storage_profile,active_serving_profile FROM meta WHERE id=1`).Scan(&out.Applied.SchemaVersion, &out.Applied.ActiveGeneration, &out.Applied.ManifestSHA256, &index, &canonical, &source, &space, &storage, &serving); err != nil {
		return out, err
	}
	out.Applied.ActiveServingProfile = configFingerprint(serving)
	out.Applied.Fingerprints.Index, out.Applied.Fingerprints.CanonicalText, out.Applied.Fingerprints.Source, out.Applied.Fingerprints.VectorSpace, out.Applied.Fingerprints.VectorStorage = configFingerprint(index), configFingerprint(canonical), configFingerprint(source), configFingerprint(space), configFingerprint(storage)
	rows, err := tx.QueryContext(ctx, `SELECT s.id,f.path,c.kind,c.qualified_symbol,c.signature,s.canonical_input_sha256,c.source_body,s.segment_number,
		v.dimensions,v.codec_id,v.codec_version,v.blob,v.scale,v.norm,v.source_profile,v.vector_space_profile,v.raw_vector_sha256,v.materialization_fingerprint,v.materialized_at,
		COALESCE((SELECT classification FROM embedding_failures ef WHERE ef.source_profile=m.source_profile AND ef.canonical_input_sha256=s.canonical_input_sha256 ORDER BY ef.id DESC LIMIT 1),'')
		FROM embedding_segments s JOIN chunks c ON c.id=s.chunk_id JOIN files f ON f.id=c.file_id JOIN meta m ON m.id=1
		LEFT JOIN vector_cache v ON v.serving_profile=m.active_serving_profile AND v.canonical_input_sha256=s.canonical_input_sha256
		WHERE s.serving_profile=m.active_serving_profile ORDER BY s.id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var segment EmbeddingPlanningSegment
		var dimensions, version sql.NullInt64
		var codec, sourceRow, spaceRow, rawSHA, materialization, at sql.NullString
		var blob []byte
		var scale, norm sql.NullFloat64
		var failureClass string
		if err := rows.Scan(&segment.ID, &segment.Path, &segment.Kind, &segment.QualifiedSymbol, &segment.Signature, &segment.CanonicalInputSHA256, &segment.SourceBody, &segment.Number, &dimensions, &codec, &version, &blob, &scale, &norm, &sourceRow, &spaceRow, &rawSHA, &materialization, &at, &failureClass); err != nil {
			rows.Close()
			return out, err
		}
		valid := false
		if dimensions.Valid && codec.Valid && version.Valid && sourceRow.Valid && spaceRow.Valid && rawSHA.Valid && materialization.Valid && at.Valid {
			stored := vector.StoredVector{Dimensions: int(dimensions.Int64), CodecID: codec.String, CodecVersion: uint16(version.Int64), Blob: blob}
			if scale.Valid {
				stored.Scale = float32(scale.Float64)
			}
			if norm.Valid {
				stored.Norm = float32(norm.Float64)
			}
			_, timestampErr := time.Parse(time.RFC3339Nano, at.String)
			valid = sourceRow.String == string(resolved.Profiles.Fingerprints.Source) && spaceRow.String == string(resolved.Profiles.Fingerprints.VectorSpace) && materialization.String == string(resolved.Profiles.Fingerprints.VectorStorage) && validSHA256(rawSHA.String) && timestampErr == nil && ValidateServingVector(resolved, stored) == nil
		}
		segment.State = DeriveEmbeddingState(valid, failureClass == "terminal")
		out.Segments = append(out.Segments, segment)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	if err := rows.Close(); err != nil {
		return out, err
	}
	projections, err := tx.QueryContext(ctx, `SELECT p.segment_id,p.start_byte,p.end_byte FROM segment_projections p JOIN embedding_segments s ON s.id=p.segment_id JOIN meta m ON m.id=1 WHERE s.serving_profile=m.active_serving_profile ORDER BY p.segment_id,p.ordinal`)
	if err != nil {
		return out, err
	}
	byID := make(map[int64]int, len(out.Segments))
	for i := range out.Segments {
		byID[out.Segments[i].ID] = i
	}
	for projections.Next() {
		var id int64
		var r PreparedIndexRange
		if err := projections.Scan(&id, &r.StartByte, &r.EndByte); err != nil {
			projections.Close()
			return out, err
		}
		if i, ok := byID[id]; ok {
			out.Segments[i].Projections = append(out.Segments[i].Projections, r)
		}
	}
	if err := projections.Err(); err != nil {
		projections.Close()
		return out, err
	}
	if err := projections.Close(); err != nil {
		return out, err
	}
	if err := tx.Commit(); err != nil {
		return out, err
	}
	return out, nil
}

type EmbeddingRun struct {
	Generation                                                                                      int64
	ManifestSHA256, SourceProfile, VectorSpaceProfile, StorageProfile                               string
	Planned, Ready, Skipped, Requested, Succeeded, Failed, Discarded, EstimatedTokens, ActualTokens int
}

func (store *ProductionStore) StartEmbeddingRun(ctx context.Context, resolved config.ResolvedConfig, run EmbeddingRun) (int64, error) {
	if err := resolved.ValidateIntegrity(); err != nil {
		return 0, err
	}
	if run.Generation < 0 || !validSHA256(run.ManifestSHA256) || !validSHA256(run.SourceProfile) || !validSHA256(run.VectorSpaceProfile) || !validSHA256(run.StorageProfile) || run.Planned < 0 || run.Ready < 0 || run.Skipped < 0 || run.Requested != 0 || run.Succeeded != 0 || run.Failed != 0 || run.Discarded != 0 || run.EstimatedTokens < 0 || run.ActualTokens != 0 || run.Ready+run.Skipped > run.Planned {
		return 0, fmt.Errorf("invalid embedding run")
	}
	tx, err := store.Write.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var generation int64
	var manifest, source, space, storage string
	if err := tx.QueryRowContext(ctx, `SELECT active_generation,manifest_sha256,source_profile,vector_space_profile,vector_storage_profile FROM meta WHERE id=1`).Scan(&generation, &manifest, &source, &space, &storage); err != nil {
		return 0, err
	}
	if generation != run.Generation || manifest != run.ManifestSHA256 || source != run.SourceProfile || space != run.VectorSpaceProfile || storage != run.StorageProfile {
		return 0, ErrEmbeddingStateChanged
	}
	r, err := tx.ExecContext(ctx, `INSERT INTO embedding_runs(generation,manifest_sha256,source_profile,vector_space_profile,storage_profile,planned_count,ready_count,skipped_count,requested_count,succeeded_count,failed_count,discarded_count,estimated_tokens,actual_tokens,status,started_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?, 'running',?)`, run.Generation, run.ManifestSHA256, run.SourceProfile, run.VectorSpaceProfile, run.StorageProfile, run.Planned, run.Ready, run.Skipped, run.Requested, run.Succeeded, run.Failed, run.Discarded, run.EstimatedTokens, run.ActualTokens, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (store *ProductionStore) FinishEmbeddingRun(ctx context.Context, id int64, run EmbeddingRun, status string) error {
	if id <= 0 || (status != "partially_succeeded" && status != "succeeded" && status != "failed" && status != "cancelled") {
		return fmt.Errorf("invalid embedding run completion")
	}
	payable := run.Planned - run.Ready - run.Skipped
	if run.Planned < 0 || run.Ready < 0 || run.Skipped < 0 || run.Requested < 0 || run.Succeeded < 0 || run.Failed < 0 || run.Discarded < 0 || run.EstimatedTokens < 0 || run.ActualTokens < 0 || run.Ready+run.Skipped > run.Planned || run.Succeeded+run.Failed+run.Discarded > payable || run.Requested < run.Succeeded+run.Failed+run.Discarded {
		return fmt.Errorf("invalid embedding run counts")
	}
	if status == "succeeded" && run.Succeeded != run.Planned-run.Ready-run.Skipped {
		return fmt.Errorf("invalid succeeded embedding run")
	}
	if status == "partially_succeeded" && run.Succeeded == 0 {
		return fmt.Errorf("invalid partial embedding run")
	}
	if status == "failed" && run.Succeeded != 0 {
		return fmt.Errorf("invalid failed embedding run")
	}
	r, err := store.Write.db.ExecContext(ctx, `UPDATE embedding_runs SET requested_count=?,succeeded_count=?,failed_count=?,discarded_count=?,actual_tokens=?,status=?,finished_at=? WHERE id=? AND status='running'`, run.Requested, run.Succeeded, run.Failed, run.Discarded, run.ActualTokens, status, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("embedding run not running")
	}
	return nil
}

type EmbeddingWriteExpectation struct {
	Generation     int64
	ManifestSHA256 string
}

func (store *ProductionStore) PublishEmbeddedVector(ctx context.Context, resolved config.ResolvedConfig, expected EmbeddingWriteExpectation, inputHash, rawSHA string, stored vector.StoredVector) (bool, error) {
	if !validSHA256(inputHash) || !validSHA256(rawSHA) || !validSHA256(expected.ManifestSHA256) {
		return false, fmt.Errorf("invalid embedding publication provenance")
	}
	if err := ValidateServingVector(resolved, stored); err != nil {
		return false, err
	}
	tx, err := store.Write.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if err := requireResolvedActiveProfile(ctx, tx, resolved); err != nil {
		return false, err
	}
	var generation int64
	var manifest, source, space, storage string
	if err := tx.QueryRowContext(ctx, `SELECT active_generation,manifest_sha256,source_profile,vector_space_profile,vector_storage_profile FROM meta WHERE id=1`).Scan(&generation, &manifest, &source, &space, &storage); err != nil {
		return false, err
	}
	if generation != expected.Generation || manifest != expected.ManifestSHA256 || source != string(resolved.Profiles.Fingerprints.Source) || space != string(resolved.Profiles.Fingerprints.VectorSpace) || storage != string(resolved.Profiles.Fingerprints.VectorStorage) {
		return false, ErrEmbeddingStateChanged
	}
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM embedding_segments WHERE serving_profile=? AND canonical_input_sha256=?`, storage, inputHash).Scan(&active); err != nil {
		return false, err
	}
	if active == 0 {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `INSERT INTO vector_cache(serving_profile,canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,materialization_fingerprint,source_profile,vector_space_profile,raw_vector_sha256,materialized_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(serving_profile,canonical_input_sha256) DO UPDATE SET dimensions=excluded.dimensions,codec_id=excluded.codec_id,codec_version=excluded.codec_version,blob=excluded.blob,scale=excluded.scale,norm=excluded.norm,materialization_fingerprint=excluded.materialization_fingerprint,source_profile=excluded.source_profile,vector_space_profile=excluded.vector_space_profile,raw_vector_sha256=excluded.raw_vector_sha256,materialized_at=excluded.materialized_at`, storage, inputHash, stored.Dimensions, stored.CodecID, stored.CodecVersion, stored.Blob, nullableFloat(stored.Scale), nullableFloat(stored.Norm), storage, source, space, rawSHA, now); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM embedding_failures WHERE source_profile=? AND canonical_input_sha256=?`, source, inputHash); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (store *ProductionStore) RecordCurrentEmbeddingFailure(ctx context.Context, resolved config.ResolvedConfig, expected EmbeddingWriteExpectation, inputHash, classification, class, message string, attempts int) (bool, error) {
	if !validSHA256(inputHash) || !validSHA256(expected.ManifestSHA256) || (classification != "terminal" && classification != "retryable") || class == "" || message == "" || attempts <= 0 {
		return false, fmt.Errorf("invalid embedding failure")
	}
	tx, err := store.Write.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if err := requireResolvedActiveProfile(ctx, tx, resolved); err != nil {
		return false, err
	}
	var generation int64
	var manifest, source, space, storage string
	if err := tx.QueryRowContext(ctx, `SELECT active_generation,manifest_sha256,source_profile,vector_space_profile,vector_storage_profile FROM meta WHERE id=1`).Scan(&generation, &manifest, &source, &space, &storage); err != nil {
		return false, err
	}
	if generation != expected.Generation || manifest != expected.ManifestSHA256 || source != string(resolved.Profiles.Fingerprints.Source) || space != string(resolved.Profiles.Fingerprints.VectorSpace) || storage != string(resolved.Profiles.Fingerprints.VectorStorage) {
		return false, ErrEmbeddingStateChanged
	}
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM embedding_segments WHERE serving_profile=? AND canonical_input_sha256=?`, storage, inputHash).Scan(&active); err != nil {
		return false, err
	}
	if active == 0 {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	var dimensions, version sql.NullInt64
	var codec, rowSource, rowSpace, rawSHA, materialization, materializedAt sql.NullString
	var blob []byte
	var scale, norm sql.NullFloat64
	err = tx.QueryRowContext(ctx, `SELECT dimensions,codec_id,codec_version,blob,scale,norm,source_profile,vector_space_profile,raw_vector_sha256,materialization_fingerprint,materialized_at FROM vector_cache WHERE serving_profile=? AND canonical_input_sha256=?`, storage, inputHash).Scan(&dimensions, &codec, &version, &blob, &scale, &norm, &rowSource, &rowSpace, &rawSHA, &materialization, &materializedAt)
	ready := false
	if err == nil && dimensions.Valid && codec.Valid && version.Valid && rowSource.Valid && rowSpace.Valid && rawSHA.Valid && materialization.Valid && materializedAt.Valid {
		stored := vector.StoredVector{Dimensions: int(dimensions.Int64), CodecID: codec.String, CodecVersion: uint16(version.Int64), Blob: blob}
		if scale.Valid {
			stored.Scale = float32(scale.Float64)
		}
		if norm.Valid {
			stored.Norm = float32(norm.Float64)
		}
		_, timestampErr := time.Parse(time.RFC3339Nano, materializedAt.String)
		ready = rowSource.String == source && rowSpace.String == space && materialization.String == storage && validSHA256(rawSHA.String) && timestampErr == nil && ValidateServingVector(resolved, stored) == nil
	} else if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	if ready {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO embedding_failures(source_profile,canonical_input_sha256,classification,attempts,error_class,last_error,last_attempted_at) VALUES(?,?,?,?,?,?,?)`, source, inputHash, classification, attempts, class, message, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
