package config

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"cidx/internal/profile"
)

func TestRFC8785ProfileFixturesAndStrictResolvedDefaults(t *testing.T) {
	canonical := profile.CanonicalTextProfile{FormatterID: "cidx-canonical-text", FormatterVersion: 1, ProjectionOrder: []string{"path", "kind", "qualified_symbol", "signature", "body"}}
	if digest, err := Fingerprint(canonical, CanonicalTextDomain); err != nil || digest != "eabf6198a0be430d4d5c15f1f036d99e8fbde12381a060be0debb34c211c7016" {
		t.Fatalf("canonical fixture = %s, %v", digest, err)
	}
	source := profile.EmbeddingSourceProfile{Provider: "voyage-official", Model: "voyage-code-4", SourceDimensions: 1024, OutputDType: "float", InputTypeMapping: profile.InputTypeMapping{Document: "document", Query: "query"}, Truncation: false, AdapterVersion: 1}
	if digest, err := Fingerprint(source, SourceProfileDomain); err != nil || digest != "923a0b84bf40d3880b5a081861ed6e17208380a5fa152847d66d2b98f222c0b3" {
		t.Fatalf("source fixture = %s, %v", digest, err)
	}
	implicit, err := LoadBytes([]byte(validConfigJSON("")))
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := LoadBytes([]byte(validConfigJSON(`,"model":"voyage-code-4","reducer":"prefix-l2-v1","normalizer":"l2-v1","metric":"cosine","storage_codec":"binary"`)))
	if err != nil {
		t.Fatal(err)
	}
	if implicit.Profiles.Fingerprints != explicit.Profiles.Fingerprints {
		t.Fatalf("defaults changed semantic fingerprints: %#v %#v", implicit.Profiles.Fingerprints, explicit.Profiles.Fingerprints)
	}
	if _, err := LoadBytes([]byte(`{"version":1,"unknown":true}`)); err == nil {
		t.Fatal("unknown field accepted")
	}
	if _, err := LoadBytes([]byte(`{"version":1,"version":1}`)); err == nil {
		t.Fatal("duplicate key accepted")
	}
	if plan := PlanImpact(implicit.Profiles, AppliedProfiles{Fingerprints: profile.ProfileFingerprints{Source: "different"}}, 7); plan.Class != ImpactPaidEmbeddingRequired || !plan.HybridFTSOnlyFallback {
		t.Fatalf("unexpected source impact: %#v", plan)
	}
	if plan := PlanImpact(implicit.Profiles, AppliedProfiles{Fingerprints: profile.ProfileFingerprints{CanonicalText: "different"}}, 7); plan.Class != ImpactLocalReindex || !plan.RequiresCanonicalReconciliation {
		t.Fatalf("canonical-text mismatch must reconcile locally before paid misses: %#v", plan)
	}
	if plan := PlanImpact(implicit.Profiles, AppliedProfiles{Fingerprints: profile.ProfileFingerprints{Policy: "different"}}, 7); plan.Class != ImpactRestartOnly || plan.HybridFTSOnlyFallback {
		t.Fatalf("policy mismatch must be restart-only: %#v", plan)
	}
}

func TestServingPolicyFingerprintIncludesQueryTextFormatVersion(t *testing.T) {
	base := profile.ServingPolicyProfile{DefaultMode: "fts", QueryTextFormatVersion: QueryTextFormatVersion}
	changed := base
	changed.QueryTextFormatVersion++
	left, err := Fingerprint(base, ServingPolicyDomain)
	if err != nil {
		t.Fatal(err)
	}
	right, err := Fingerprint(changed, ServingPolicyDomain)
	if err != nil || left == right {
		t.Fatalf("query format version is absent from policy fingerprint: %q %q %v", left, right, err)
	}
}

func TestCanonicalJSONRFC8785UsedValueConformance(t *testing.T) {
	// Phase 00's profile fixtures above prove the externally fixed digests.
	canonical, err := CanonicalJSON(map[string]any{"html": "<script>&\u2028", "\U0001f600": 1, "\ufffd": 2})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(canonical, []byte(`\u003c`)) || bytes.Contains(canonical, []byte(`\u0026`)) {
		t.Fatalf("JCS escaped HTML-sensitive text: %s", canonical)
	}
	if !bytes.Contains(canonical, []byte(`"😀":1,"�":2`)) {
		t.Fatalf("keys are not UTF-16 sorted: %s", canonical)
	}
	if _, err := DecodeRaw([]byte(`{"version":1,"index":{"languages":["go"],"max_source_file_bytes":1,"max_chunk_bytes":1,"max_segment_input_bytes":1},"embedding":{"target_dimensions":256,"batch":{"max_inputs":1,"max_input_tokens":1,"max_retries":0,"request_timeout_ms":1}},"search":{"default_mode":"\ud800"},"mcp":{"hard_max_inline_bytes":1,"max_read_span_lines":1}}`)); err == nil {
		t.Fatal("lone surrogate accepted")
	}
	if _, err := DecodeRaw([]byte(`{"version":1,"index":{"languages":["go"],"max_source_file_bytes":1,"max_chunk_bytes":1,"max_segment_input_bytes":1},"embedding":{"target_dimensions":256,"batch":{"max_inputs":1,"max_input_tokens":1,"max_retries":0,"request_timeout_ms":1}},"search":{"default_mode":"\ud83d\ude00"},"mcp":{"hard_max_inline_bytes":1,"max_read_span_lines":1}}`)); err != nil {
		t.Fatalf("valid surrogate pair rejected: %v", err)
	}
	zero, err := CanonicalJSON(struct {
		Value int `json:"value"`
	}{})
	if err != nil {
		t.Fatal(err)
	}
	if string(zero) != `{"value":0}` {
		t.Fatalf("explicit zero omitted: %s", zero)
	}
	if strings.Contains(string(zero), "null") {
		t.Fatal("zero unexpectedly encoded as null")
	}
	numbers, err := CanonicalJSON(map[string]any{"small": 1e-6, "fraction": 0.75, "large": 1e21, "negative_zero": math.Copysign(0, -1)})
	if err != nil || string(numbers) != `{"fraction":0.75,"large":1e+21,"negative_zero":0,"small":0.000001}` {
		t.Fatalf("JCS finite numbers=%s err=%v", numbers, err)
	}
	if _, err := CanonicalJSON(profile.CanonicalTextProfile{FormatterID: string([]byte{0xff}), FormatterVersion: 1}); err == nil {
		t.Fatal("invalid UTF-8 nested in profile struct accepted")
	}
}

func TestResolvedSearchDefaultsAndExplicitZeroRejection(t *testing.T) {
	implicit, err := LoadBytes([]byte(validConfigJSON("")))
	if err != nil {
		t.Fatalf("omitted defaults rejected: %v", err)
	}
	if implicit.Search.QueryLimits != (QueryLimits{MaxBytes: DefaultMaxQueryBytes, MaxTokens: DefaultMaxQueryTokens, MaxTokenRunes: DefaultMaxQueryTokenRunes}) {
		t.Fatalf("unexpected query defaults: %#v", implicit.Search.QueryLimits)
	}
	for _, field := range []string{`"return_k":0`, `"candidate_k":0`, `"rrf_k":0`, `"max_query_bytes":0`, `"max_query_tokens":0`, `"max_query_token_runes":0`, `"fts_weights":{"symbols":0}`, `"fts_weights":{"body":0}`} {
		config := strings.Replace(validConfigJSON(""), `"search":{}`, `"search":{`+field+`}`, 1)
		if _, err := LoadBytes([]byte(config)); err == nil {
			t.Fatalf("explicit zero accepted for %s", field)
		}
	}
	hybrid := strings.Replace(validConfigJSON(""), `"search":{}`, `"search":{"default_mode":"hybrid"}`, 1)
	if _, err := LoadBytes([]byte(hybrid)); err == nil {
		t.Fatal("hybrid without paid permission accepted")
	}
	withFloats := strings.Replace(validConfigJSON(""), `"search":{}`, `"search":{"fts_weights":{"symbols":2.5,"body":0.75}}`, 1)
	first, err := LoadBytes([]byte(withFloats))
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadBytes([]byte(withFloats))
	if err != nil {
		t.Fatal(err)
	}
	if first.Profiles.Fingerprints.Policy != second.Profiles.Fingerprints.Policy {
		t.Fatal("finite FTS-weight policy fingerprint is not deterministic")
	}
	configured := strings.Replace(validConfigJSON(""), `"search":{}`, `"search":{"max_query_bytes":4096,"max_query_tokens":32,"max_query_token_runes":96}`, 1)
	resolved, err := LoadBytes([]byte(configured))
	if err != nil || resolved.Search.QueryLimits != (QueryLimits{MaxBytes: 4096, MaxTokens: 32, MaxTokenRunes: 96}) {
		t.Fatalf("query policy=%#v err=%v", resolved.Search.QueryLimits, err)
	}
	if implicit.Profiles.Fingerprints.Policy == resolved.Profiles.Fingerprints.Policy {
		t.Fatal("query limits did not change serving-policy fingerprint")
	}
	tooLarge := strings.Replace(validConfigJSON(""), `"search":{}`, `"search":{"max_query_bytes":1048577}`, 1)
	if _, err := LoadBytes([]byte(tooLarge)); err == nil {
		t.Fatal("query byte ceiling accepted")
	}
}

func TestPlanImpactClassesAndPrecedence(t *testing.T) {
	resolved, err := LoadBytes([]byte(validConfigJSON("")))
	if err != nil {
		t.Fatal(err)
	}
	desired := resolved.Profiles
	cases := []struct {
		name    string
		applied AppliedProfiles
		want    ImpactClass
	}{
		{"none", AppliedProfiles{}, ImpactNone},
		{"restart", AppliedProfiles{Fingerprints: profile.ProfileFingerprints{Policy: "changed"}}, ImpactRestartOnly},
		{"storage", AppliedProfiles{Fingerprints: profile.ProfileFingerprints{VectorStorage: "changed"}}, ImpactLocalRematerializeIfRaw},
		{"space", AppliedProfiles{Fingerprints: profile.ProfileFingerprints{VectorSpace: "changed"}}, ImpactLocalRematerializeIfRaw},
		{"source", AppliedProfiles{Fingerprints: profile.ProfileFingerprints{Source: "changed"}}, ImpactPaidEmbeddingRequired},
		{"canonical", AppliedProfiles{Fingerprints: profile.ProfileFingerprints{CanonicalText: "changed"}}, ImpactLocalReindex},
		{"index", AppliedProfiles{Fingerprints: profile.ProfileFingerprints{Index: "changed"}}, ImpactLocalReindex},
		{"schema", AppliedProfiles{SchemaVersion: 99, Fingerprints: profile.ProfileFingerprints{Index: "changed"}}, ImpactSchemaMigration},
		{"index before source", AppliedProfiles{Fingerprints: profile.ProfileFingerprints{Index: "changed", Source: "changed"}}, ImpactLocalReindex},
	}
	for _, test := range cases {
		if got := PlanImpact(desired, test.applied, 7); got.Class != test.want {
			t.Fatalf("%s impact=%#v", test.name, got)
		}
	}
}

func TestResolvedConfigIntegrityRejectsPostResolutionMutation(t *testing.T) {
	resolved, err := LoadBytes([]byte(validConfigJSON("")))
	if err != nil {
		t.Fatal(err)
	}
	if err := resolved.ValidateIntegrity(); err != nil {
		t.Fatal(err)
	}
	resolved.Embedding.TargetDimensions = 512
	if err := resolved.ValidateIntegrity(); err == nil {
		t.Fatal("mutated resolved config retained old fingerprints")
	}
}

func validConfigJSON(extra string) string {
	return `{"version":1,"index":{"languages":["go","typescript"],"max_source_file_bytes":10000,"max_chunk_bytes":5000,"max_segment_input_bytes":2000},"embedding":{"target_dimensions":256` + extra + `,"batch":{"max_inputs":1,"max_input_tokens":1,"max_retries":0,"request_timeout_ms":1}},"search":{},"mcp":{"hard_max_inline_bytes":1,"max_read_span_lines":1}}`
}
