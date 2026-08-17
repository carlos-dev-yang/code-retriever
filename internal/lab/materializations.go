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

// MaterializedVariant is request-local staging data. It is held in memory only
// until one production publication attempt; evaluation.db never stores blobs.
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

func (s *Store) PutMaterializedVariants(ctx context.Context, runID int64, variants []MaterializedVariant) error {
	if runID <= 0 || len(variants) == 0 {
		return fmt.Errorf("materialized variants are required")
	}
	var status string
	if err := s.db.QueryRowContext(ctx, `SELECT status FROM materialization_runs WHERE id=?`, runID).Scan(&status); err != nil {
		return err
	}
	if status != "building" {
		return fmt.Errorf("materialization run is not building")
	}
	copyValues := make([]MaterializedVariant, len(variants))
	for index, item := range variants {
		if !validDigest(item.InputHash) || !validDigest(item.RawVectorSHA256) || item.Stored.Validate() != nil {
			return fmt.Errorf("invalid materialized variant")
		}
		copyValues[index] = cloneVariant(item)
	}
	s.stagedMu.Lock()
	defer s.stagedMu.Unlock()
	if s.staged == nil {
		s.staged = map[int64][]MaterializedVariant{}
	}
	if _, exists := s.staged[runID]; exists {
		return fmt.Errorf("materialization variants already staged")
	}
	s.staged[runID] = copyValues
	return nil
}

func (s *Store) FinishMaterialization(ctx context.Context, runID int64, status string, staged, missing, rejected int, sanitizedError string) error {
	if runID <= 0 || (status != "ready" && status != "published" && status != "aborted" && status != "failed") || staged < 0 || missing < 0 || rejected < 0 {
		return fmt.Errorf("invalid materialization completion")
	}
	if status == "ready" {
		variants, err := s.stagedVariants(runID)
		if err != nil {
			return err
		}
		var planned int
		if err := s.db.QueryRowContext(ctx, `SELECT planned_count FROM materialization_runs WHERE id=? AND status='building'`, runID).Scan(&planned); err != nil {
			return err
		}
		if planned != staged+missing+rejected || len(variants) != staged {
			return fmt.Errorf("materialization counts do not cover plan")
		}
		checksum, err := materializationEvidence(variants)
		if err != nil {
			return err
		}
		coverage := 1.0
		if planned > 0 {
			coverage = float64(staged) / float64(planned)
		}
		result, err := s.db.ExecContext(ctx, `UPDATE materialization_runs SET staged_count=?,missing_count=?,rejected_count=?,raw_coverage=?,output_checksum=?,status='ready',error=?,ended_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=? AND status='building'`, staged, missing, rejected, coverage, checksum, sanitizedError, runID)
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil || changed != 1 {
			return fmt.Errorf("materialization run is not building")
		}
		return nil
	}
	if status == "published" {
		result, err := s.db.ExecContext(ctx, `UPDATE materialization_runs SET status='published',error=?,ended_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=? AND status='ready'`, sanitizedError, runID)
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil || changed != 1 {
			return fmt.Errorf("materialization run is not ready")
		}
		s.clearStaged(runID)
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
	if err != nil || changed != 1 {
		return fmt.Errorf("materialization run is not building")
	}
	s.clearStaged(runID)
	return nil
}

func (s *Store) MaterializedVariants(ctx context.Context, runID int64, storageProfile string) ([]MaterializedVariant, error) {
	if runID <= 0 || !validDigest(storageProfile) {
		return nil, fmt.Errorf("materialization storage profile is required")
	}
	var status, storedProfile string
	if err := s.db.QueryRowContext(ctx, `SELECT status,storage_profile FROM materialization_runs WHERE id=?`, runID).Scan(&status, &storedProfile); err != nil {
		return nil, err
	}
	if status != "ready" || storedProfile != storageProfile {
		return nil, fmt.Errorf("materialization is not ready for requested profile")
	}
	return s.stagedVariants(runID)
}

func (s *Store) stagedVariants(runID int64) ([]MaterializedVariant, error) {
	s.stagedMu.Lock()
	defer s.stagedMu.Unlock()
	values, ok := s.staged[runID]
	if !ok {
		return nil, fmt.Errorf("materialization staging is unavailable")
	}
	result := make([]MaterializedVariant, len(values))
	for index, value := range values {
		result[index] = cloneVariant(value)
	}
	return result, nil
}

func (s *Store) clearStaged(runID int64) {
	s.stagedMu.Lock()
	delete(s.staged, runID)
	s.stagedMu.Unlock()
}

func cloneVariant(value MaterializedVariant) MaterializedVariant {
	value.Stored.Blob = append([]byte(nil), value.Stored.Blob...)
	return value
}

func materializationEvidence(values []MaterializedVariant) (string, error) {
	hash := sha256.New()
	for _, item := range values {
		if !validDigest(item.InputHash) || !validDigest(item.RawVectorSHA256) || item.Stored.Validate() != nil {
			return "", fmt.Errorf("invalid staged evidence provenance")
		}
		writeEvidenceField(hash, []byte(item.InputHash))
		writeEvidenceField(hash, []byte(item.Stored.CodecID))
		writeEvidenceInt(hash, int64(item.Stored.Dimensions))
		writeEvidenceInt(hash, int64(item.Stored.CodecVersion))
		writeEvidenceField(hash, item.Stored.Blob)
		writeEvidenceFloat(hash, evidenceNullable(item.Stored.Scale))
		writeEvidenceFloat(hash, evidenceNullable(item.Stored.Norm))
		writeEvidenceField(hash, []byte(item.RawVectorSHA256))
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeEvidenceField(hash interface{ Write([]byte) (int, error) }, value []byte) {
	writeEvidenceInt(hash, int64(len(value)))
	_, _ = hash.Write(value)
}

func writeEvidenceInt(hash interface{ Write([]byte) (int, error) }, value int64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], uint64(value))
	_, _ = hash.Write(encoded[:])
}

func writeEvidenceFloat(hash interface{ Write([]byte) (int, error) }, value sql.NullFloat64) {
	if !value.Valid {
		writeEvidenceInt(hash, 0)
		return
	}
	writeEvidenceInt(hash, 1)
	writeEvidenceInt(hash, int64(math.Float64bits(value.Float64)))
}

func evidenceNullable(value float32) sql.NullFloat64 {
	if value == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: float64(value), Valid: true}
}

func validDigest(value string) bool { return labSHA256(value) }
