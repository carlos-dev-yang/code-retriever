package lab

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"

	"cidx/internal/vector"
)

type MaterializationRun struct {
	BuildID, ManifestSHA256, SourceProfile, VectorSpaceProfile, StorageProfile string
	Generation                                                                 int64
	Planned, Staged, Missing, Rejected                                         int
	Status                                                                     string
	Error                                                                      string
}

type MaterializedVariant struct {
	InputHash, RawVectorSHA256 string
	Stored                     vector.StoredVector
}

func (s *Store) StartMaterialization(ctx context.Context, run MaterializationRun) (int64, error) {
	if run.BuildID == "" || !validDigest(run.ManifestSHA256) || !validDigest(run.SourceProfile) || !validDigest(run.VectorSpaceProfile) || !validDigest(run.StorageProfile) || run.Generation < 0 || run.Planned < 0 || run.Missing < 0 || run.Rejected < 0 {
		return 0, fmt.Errorf("invalid materialization run")
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO materialization_runs(build_id,generation,manifest_sha256,source_profile,vector_space_profile,storage_profile,planned_count,staged_count,missing_count,rejected_count,status,error) VALUES(?,?,?,?,?,?,?,?,?,?,?,'')`, run.BuildID, run.Generation, run.ManifestSHA256, run.SourceProfile, run.VectorSpaceProfile, run.StorageProfile, run.Planned, 0, run.Missing, run.Rejected, "building")
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// PutMaterializedVariants stages fully encoded candidates in the lab only.
// It has no production-store import and cannot make a row search-visible.
func (s *Store) PutMaterializedVariants(ctx context.Context, runID int64, variants []MaterializedVariant) error {
	if runID <= 0 || len(variants) == 0 {
		return fmt.Errorf("materialized variants are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM materialization_runs WHERE id=?`, runID).Scan(&status); err != nil {
		return err
	}
	if status != "building" {
		return fmt.Errorf("materialization run is not building")
	}
	for _, item := range variants {
		if !validDigest(item.InputHash) || !validDigest(item.RawVectorSHA256) || item.Stored.Validate() != nil {
			return fmt.Errorf("invalid materialized variant")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO materialized_variants(materialization_id,canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,raw_vector_sha256) VALUES(?,?,?,?,?,?,?,?,?)`, runID, item.InputHash, item.Stored.Dimensions, item.Stored.CodecID, item.Stored.CodecVersion, item.Stored.Blob, nullableFloat(item.Stored.Scale), nullableFloat(item.Stored.Norm), item.RawVectorSHA256); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) FinishMaterialization(ctx context.Context, runID int64, status string, staged, missing, rejected int, sanitizedError string) error {
	if runID <= 0 || (status != "ready" && status != "published" && status != "aborted" && status != "failed") || staged < 0 || missing < 0 || rejected < 0 {
		return fmt.Errorf("invalid materialization completion")
	}
	if status == "ready" {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		coverage, checksum, err := materializationEvidence(ctx, tx, runID, staged, missing, rejected)
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE materialization_runs SET staged_count=?,missing_count=?,rejected_count=?,raw_coverage=?,output_checksum=?,status='ready',error=?,ended_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=? AND status='building'`, staged, missing, rejected, coverage, checksum, sanitizedError, runID)
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return fmt.Errorf("materialization run is not building")
		}
		return tx.Commit()
	}
	if status == "published" {
		result, err := s.db.ExecContext(ctx, `UPDATE materialization_runs SET status='published',error=?,ended_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=? AND status='ready'`, sanitizedError, runID)
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return fmt.Errorf("materialization run is not ready")
		}
		return nil
	}
	where := `status='building'`
	if status == "failed" {
		where = `status IN ('building','ready')`
	}
	result, err := s.db.ExecContext(ctx, `UPDATE materialization_runs SET staged_count=?,missing_count=?,rejected_count=?,status=?,error=?,ended_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=? AND `+where, staged, missing, rejected, status, sanitizedError, runID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("materialization run is not building")
	}
	return nil
}

func materializationEvidence(ctx context.Context, tx *sql.Tx, runID int64, staged, missing, rejected int) (float64, string, error) {
	var planned int
	if err := tx.QueryRowContext(ctx, `SELECT planned_count FROM materialization_runs WHERE id=? AND status='building'`, runID).Scan(&planned); err != nil {
		return 0, "", err
	}
	if planned != staged+missing+rejected {
		return 0, "", fmt.Errorf("materialization counts do not cover plan")
	}
	rows, err := tx.QueryContext(ctx, `SELECT canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,raw_vector_sha256 FROM materialized_variants WHERE materialization_id=? ORDER BY canonical_input_sha256`, runID)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	hash := sha256.New()
	count := 0
	for rows.Next() {
		var input, codec, raw string
		var dimensions, version int
		var blob []byte
		var scale, norm sql.NullFloat64
		if err := rows.Scan(&input, &dimensions, &codec, &version, &blob, &scale, &norm, &raw); err != nil {
			return 0, "", err
		}
		if !validDigest(input) || !validDigest(raw) {
			return 0, "", fmt.Errorf("invalid staged evidence provenance")
		}
		stored := vector.StoredVector{Dimensions: dimensions, CodecID: codec, CodecVersion: uint16(version), Blob: blob}
		if scale.Valid {
			stored.Scale = float32(scale.Float64)
		}
		if norm.Valid {
			stored.Norm = float32(norm.Float64)
		}
		if err := stored.Validate(); err != nil {
			return 0, "", err
		}
		writeEvidenceField(hash, []byte(input))
		writeEvidenceField(hash, []byte(codec))
		writeEvidenceInt(hash, int64(dimensions))
		writeEvidenceInt(hash, int64(version))
		writeEvidenceField(hash, blob)
		writeEvidenceFloat(hash, scale)
		writeEvidenceFloat(hash, norm)
		writeEvidenceField(hash, []byte(raw))
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, "", err
	}
	if count != staged {
		return 0, "", fmt.Errorf("staged variant count mismatch")
	}
	coverage := 1.0
	if planned > 0 {
		coverage = float64(staged) / float64(planned)
	}
	return coverage, hex.EncodeToString(hash.Sum(nil)), nil
}

func writeEvidenceField(hash interface{ Write([]byte) (int, error) }, value []byte) {
	writeEvidenceInt(hash, int64(len(value)))
	_, _ = hash.Write(value)
}
func writeEvidenceInt(hash interface{ Write([]byte) (int, error) }, value int64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(value))
	_, _ = hash.Write(b[:])
}
func writeEvidenceFloat(hash interface{ Write([]byte) (int, error) }, value sql.NullFloat64) {
	if !value.Valid {
		writeEvidenceInt(hash, 0)
		return
	}
	writeEvidenceInt(hash, 1)
	writeEvidenceInt(hash, int64(math.Float64bits(value.Float64)))
}

func (s *Store) MaterializedVariants(ctx context.Context, runID int64, storageProfile string) ([]MaterializedVariant, error) {
	if runID <= 0 || !validDigest(storageProfile) {
		return nil, fmt.Errorf("materialization storage profile is required")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT v.canonical_input_sha256,v.dimensions,v.codec_id,v.codec_version,v.blob,v.scale,v.norm,v.raw_vector_sha256 FROM materialized_variants v JOIN materialization_runs r ON r.id=v.materialization_id WHERE v.materialization_id=? AND r.storage_profile=? AND r.status='ready' ORDER BY v.canonical_input_sha256`, runID, storageProfile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var output []MaterializedVariant
	for rows.Next() {
		var item MaterializedVariant
		var scale, norm sql.NullFloat64
		var version int
		if err := rows.Scan(&item.InputHash, &item.Stored.Dimensions, &item.Stored.CodecID, &version, &item.Stored.Blob, &scale, &norm, &item.RawVectorSHA256); err != nil {
			return nil, err
		}
		item.Stored.CodecVersion = uint16(version)
		if scale.Valid {
			item.Stored.Scale = float32(scale.Float64)
		}
		if norm.Valid {
			item.Stored.Norm = float32(norm.Float64)
		}
		if !validDigest(item.InputHash) || !validDigest(item.RawVectorSHA256) {
			return nil, fmt.Errorf("invalid staged materialization provenance")
		}
		if err := item.Stored.Validate(); err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, rows.Err()
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func nullableFloat(value float32) any {
	if value == 0 {
		return nil
	}
	return value
}
