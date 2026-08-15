package config

import "cidx/internal/profile"

type ImpactClass string

const (
	ImpactNone                    ImpactClass = "none"
	ImpactRestartOnly             ImpactClass = "restart_only"
	ImpactLocalReindex            ImpactClass = "local_reindex"
	ImpactLocalRematerializeIfRaw ImpactClass = "local_rematerialize_if_raw"
	ImpactPaidEmbeddingRequired   ImpactClass = "paid_embedding_required"
	ImpactSchemaMigration         ImpactClass = "schema_migration"
)

type AppliedProfiles struct {
	SchemaVersion        int                         `json:"schema_version"`
	ActiveGeneration     int64                       `json:"active_generation"`
	ManifestSHA256       string                      `json:"manifest_sha256"`
	ActiveServingProfile profile.Fingerprint         `json:"active_serving_profile"`
	Fingerprints         profile.ProfileFingerprints `json:"fingerprints"`
}
type ConfigImpactPlan struct {
	Class                           ImpactClass
	Reasons                         []string
	HybridFTSOnlyFallback           bool
	RequiresCanonicalReconciliation bool
}

// PlanImpact receives the production-schema authority from the store caller.
// Config-file versioning is deliberately not a database compatibility rule.
func PlanImpact(desired DesiredProfiles, applied AppliedProfiles, expectedProductionSchemaVersion int) ConfigImpactPlan {
	if expectedProductionSchemaVersion <= 0 || (applied.SchemaVersion != 0 && applied.SchemaVersion != expectedProductionSchemaVersion) {
		return ConfigImpactPlan{Class: ImpactSchemaMigration, Reasons: []string{"production schema version differs"}, HybridFTSOnlyFallback: true}
	}
	if applied.Fingerprints.Index != "" && applied.Fingerprints.Index != desired.Fingerprints.Index {
		return ConfigImpactPlan{Class: ImpactLocalReindex, Reasons: []string{"index profile differs"}, HybridFTSOnlyFallback: true}
	}
	if applied.Fingerprints.CanonicalText != "" && applied.Fingerprints.CanonicalText != desired.Fingerprints.CanonicalText {
		return ConfigImpactPlan{Class: ImpactLocalReindex, Reasons: []string{"canonical-text profile differs; recompute canonical text and hashes before selecting paid misses"}, HybridFTSOnlyFallback: true, RequiresCanonicalReconciliation: true}
	}
	if applied.Fingerprints.Source != "" && applied.Fingerprints.Source != desired.Fingerprints.Source {
		return ConfigImpactPlan{Class: ImpactPaidEmbeddingRequired, Reasons: []string{"embedding source profile differs"}, HybridFTSOnlyFallback: true}
	}
	if applied.Fingerprints.VectorSpace != "" && applied.Fingerprints.VectorSpace != desired.Fingerprints.VectorSpace {
		return ConfigImpactPlan{Class: ImpactLocalRematerializeIfRaw, Reasons: []string{"vector space differs"}, HybridFTSOnlyFallback: true}
	}
	if applied.Fingerprints.VectorStorage != "" && applied.Fingerprints.VectorStorage != desired.Fingerprints.VectorStorage {
		return ConfigImpactPlan{Class: ImpactLocalRematerializeIfRaw, Reasons: []string{"storage codec differs"}, HybridFTSOnlyFallback: true}
	}
	if applied.Fingerprints.Policy != "" && applied.Fingerprints.Policy != desired.Fingerprints.Policy {
		return ConfigImpactPlan{Class: ImpactRestartOnly, Reasons: []string{"serving policy differs"}}
	}
	return ConfigImpactPlan{Class: ImpactNone}
}
