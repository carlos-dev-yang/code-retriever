package evalcontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestPortableJSONSchemasRejectNestedContractViolations(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	base := filepath.Join("..", "..", "schemas", "evaluation")
	for _, name := range []string{"evaluation-case.schema.json", "stage-trace.schema.json", "artifact-manifest.schema.json", "run-manifest.schema.json", "promotion-contract.schema.json", "promotion-result.schema.json"} {
		contents, err := os.Open(filepath.Join(base, name))
		if err != nil {
			t.Fatal(err)
		}
		doc, err := jsonschema.UnmarshalJSON(contents)
		_ = contents.Close()
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if err := compiler.AddResource(name, doc); err != nil {
			t.Fatalf("add %s: %v", name, err)
		}
		if _, err := compiler.Compile(name); err != nil {
			t.Fatalf("compile %s: %v", name, err)
		}
	}

	validCase := `{"schema_version":1,"id":"q1","text":"where","language":"go","cohorts":["identifier"],"answer_mode":"SINGLE","expected_cardinality":1,"split":"calibration","required_constraints":{"identifiers":["F"],"paths":["pkg/file.go"],"languages":["go"],"scopes":["declaration"]},"required_groups":[{"id":"g1","alternatives":[{"spans":[{"path":"pkg/file.go","content_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","qualified_symbol":"pkg.F","start_byte":0,"end_byte":1}]}]}],"hard_negatives":[{"span":{"path":"pkg/other.go","content_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","qualified_symbol":"pkg.Other","start_byte":0,"end_byte":1},"reason":"misleading name"}],"judgments":[{"span":{"path":"pkg/file.go","content_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","qualified_symbol":"pkg.F","start_byte":0,"end_byte":1},"grade":2,"rationale":"direct"},{"span":{"path":"pkg/other.go","content_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","qualified_symbol":"pkg.Other","start_byte":0,"end_byte":1},"grade":0,"rationale":"negative"}],"review":{"state":"frozen","passes":[{"id":"pass-1","reviewer":"a"},{"id":"pass-2","reviewer":"b"}],"rationale":"reviewed"},"assistant_task_requirements":{"requirements":["find F"],"expected_test_outcomes":["test passes"]},"digest":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}`
	assertSchema(t, compiler, "evaluation-case.schema.json", validCase, true)
	assertSchema(t, compiler, "evaluation-case.schema.json", strings.Replace(validCase, `"end_byte":1`, `"end_byte":1,"chunk_row_id":7`, 1), false)
	assertSchema(t, compiler, "evaluation-case.schema.json", strings.Replace(validCase, `"split":"calibration"`, `"split":"tuning"`, 1), false)
	assertSchema(t, compiler, "evaluation-case.schema.json", strings.Replace(validCase, `{"id":"pass-1","reviewer":"a"},{"id":"pass-2","reviewer":"b"}`, `{"id":"pass-1","reviewer":"a"}`, 1), false)
	assertSchema(t, compiler, "evaluation-case.schema.json", strings.Replace(validCase, `"pkg/file.go"`, `"a/../file.go"`, 1), false)
	assertSchema(t, compiler, "evaluation-case.schema.json", strings.Replace(validCase, `"end_byte":1`, `"end_byte":0`, 1), false)
	assertSchema(t, compiler, "evaluation-case.schema.json", strings.Replace(validCase, `"cohorts":["identifier"]`, `"cohorts":[]`, 1), false)
	assertSchema(t, compiler, "evaluation-case.schema.json", strings.Replace(validCase, `"identifiers":["F"]`, `"identifiers":["F","F"]`, 1), false)
	abstainable := strings.Replace(strings.Replace(validCase, `"answer_mode":"SINGLE","expected_cardinality":1`, `"answer_mode":"ABSTAINABLE","expected_cardinality":0`, 1), `"required_groups":[{"id":"g1","alternatives":[{"spans":[{"path":"pkg/file.go","content_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","qualified_symbol":"pkg.F","start_byte":0,"end_byte":1}]}]}]`, `"required_groups":[]`, 1)
	assertSchema(t, compiler, "evaluation-case.schema.json", abstainable, true)

	validTrace := `{"schema_version":1,"query_id":"q1","required_group_ids":["g1"],"observations":[` + stageJSON() + `],"terminal_state":"complete"}`
	assertSchema(t, compiler, "stage-trace.schema.json", validTrace, true)
	assertSchema(t, compiler, "stage-trace.schema.json", strings.Replace(validTrace, `"denominators":[{"name":"required_groups","truth_unit":"required group","count":1}]`, `"denominators":[]`, 1), false)
	assertSchema(t, compiler, "stage-trace.schema.json", strings.Replace(validTrace, `"stage":"source_discovery"`, `"stage":"parser_chunker"`, 1), false)
	assertSchema(t, compiler, "stage-trace.schema.json", strings.Replace(validTrace, `"group_observations":[]`, `"group_observations":[{"group_id":"g1","present":true,"first_loss":"NO_LOSS"}]`, 1), false)
	assertSchema(t, compiler, "stage-trace.schema.json", strings.Replace(validTrace, `"stage":"assistant_use","required":false,"status":"NOT_OBSERVED","denominators":[],"group_observations":[],"candidate_count":0`, `"stage":"assistant_use","required":false,"status":"NOT_OBSERVED","denominators":[],"group_observations":[],"candidate_count":1`, 1), false)
	provider := `"stage":"provider_union","required":true,"status":"OBSERVED","denominators":[{"name":"required_groups","truth_unit":"required group","count":1}],"group_observations":[{"group_id":"g1","present":true,"first_loss":"NO_LOSS"}],"candidate_count":0`
	operationFailure := strings.Replace(validTrace, provider, `"stage":"provider_union","failure_stage":"provider_union","required":true,"status":"OBSERVED","denominators":[{"name":"required_groups","truth_unit":"required group","count":1}],"group_observations":[{"group_id":"g1","present":false,"first_loss":"OPERATION_FAILURE:provider_union"}],"candidate_count":0`, 1)
	assertSchema(t, compiler, "stage-trace.schema.json", operationFailure, true)

	validRun := `{"schema_version":1,"corpus_manifest_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","query_manifest_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","code_commit":"deadbeef","profile_fingerprint":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","generation":1,"candidate_policy":"fixed","platform":"local","paired_controls":{"corpus_state_sha256":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","label_digest_sha256":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","parser_version":"v1","chunker_version":"v1","fts_schema_version":"v1","source_model":"voyage-code-4","source_dimensions":1024,"reducer_id":"prefix-l2-v1","serving_dimensions":256,"candidate_policy":"fixed","body_budget":"64KiB","mcp_version":"v1"}}`
	assertSchema(t, compiler, "run-manifest.schema.json", validRun, true)
	assertSchema(t, compiler, "run-manifest.schema.json", strings.Replace(validRun, `"serving_dimensions":256`, `"target_dimensions":256`, 1), false)
	validPromotionContract := `{"schema_version":1,"scope":"core_retrieval","calibration_evidence_sha256":["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"],"frozen_gates":["all-observations"],"confirmation_dataset_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","paired_controls":{"corpus_state_sha256":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","label_digest_sha256":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","parser_version":"v1","chunker_version":"v1","fts_schema_version":"v1","source_model":"voyage-code-4","source_dimensions":1024,"reducer_id":"prefix-l2-v1","serving_dimensions":256,"candidate_policy":"fixed","body_budget":"64KiB","mcp_version":"v1"}}`
	assertSchema(t, compiler, "promotion-contract.schema.json", validPromotionContract, true)
	assertSchema(t, compiler, "promotion-contract.schema.json", strings.Replace(validPromotionContract, `"serving_dimensions":256`, `"target_dimensions":256`, 1), false)

	validResult := `{"schema_version":1,"scope":"core_retrieval","status":"NOT_PROMOTION_READY","prerequisite_sha256":[],"passed_gates":[],"failed_gates":["missing"],"incomplete_reason":"missing evidence","applicable_gates":["all-observations"]}`
	assertSchema(t, compiler, "promotion-result.schema.json", validResult, true)
	assertSchema(t, compiler, "promotion-result.schema.json", strings.Replace(validResult, `"applicable_gates"`, `"weighted_total":1,"applicable_gates"`, 1), false)
	ready := `{"schema_version":1,"scope":"core_retrieval","status":"PROMOTION_EVIDENCE_READY","prerequisite_sha256":[],"passed_gates":["all"],"failed_gates":[],"applicable_gates":["all"]}`
	assertSchema(t, compiler, "promotion-result.schema.json", ready, true)
	assertSchema(t, compiler, "promotion-result.schema.json", strings.Replace(ready, `"failed_gates":[]`, `"failed_gates":["bad"]`, 1), false)
}

func assertSchema(t *testing.T, compiler *jsonschema.Compiler, name, instance string, wantValid bool) {
	t.Helper()
	schema, err := compiler.Compile(name)
	if err != nil {
		t.Fatalf("compile %s: %v", name, err)
	}
	value, err := jsonschema.UnmarshalJSON(strings.NewReader(instance))
	if err != nil {
		t.Fatalf("instance: %v", err)
	}
	err = schema.Validate(value)
	if (err == nil) != wantValid {
		t.Fatalf("%s valid=%v err=%v", name, err == nil, err)
	}
}

func stageJSON() string {
	stages := []string{"source_discovery", "parser_chunker", "fts_candidate", "dense_segment", "provider_union", "segment_parent_collapse", "rrf_fusion", "body_packaging", "assistant_use", "assistant_resolution", "operational"}
	values := make([]string, 0, len(stages))
	for index, stage := range stages {
		if stage == "operational" {
			values = append(values, `{"stage":"`+stage+`","required":true,"status":"OBSERVED","denominators":[{"name":"operation_attempts","truth_unit":"operation","count":1}],"group_observations":[],"candidate_count":0}`)
		} else if index < 8 {
			values = append(values, `{"stage":"`+stage+`","required":true,"status":"OBSERVED","denominators":[{"name":"required_groups","truth_unit":"required group","count":1}],"group_observations":[{"group_id":"g1","present":true,"first_loss":"NO_LOSS"}],"candidate_count":0}`)
		} else {
			values = append(values, `{"stage":"`+stage+`","required":false,"status":"NOT_OBSERVED","denominators":[],"group_observations":[],"candidate_count":0}`)
		}
	}
	return strings.Join(values, ",")
}
