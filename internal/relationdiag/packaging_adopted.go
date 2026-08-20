package relationdiag

import "fmt"

const (
	PackagingAdoptedKind                 = "cidx.relation_packaging.adopted_evaluation_contract.v1"
	PackagingAdoptedSiblingStatus        = "ADOPTED_EVALUATION_ONLY"
	PackagingAdoptedOneHopStatus         = "REJECTED_DEFAULT_PUSH"
	PackagingAdoptedGraphProductPath     = "NO_PRODUCT_GRAPH_PATH"
	PackagingAdoptedConfirmation         = "REQUIRES_OWNER_SELECTED_UNEXPOSED_UNIT"
	PackagingAdoptedAssistant            = "DEFERRED_NOT_A_GATE"
	PackagingAdoptedProductionMCP        = false
	PackagingAdoptedSiblingCount         = 4
	PackagingAdoptedSiblingBytes         = 4096
	PackagingLiveDecisionSHA256          = "49f2d483fb5d75a18854c36de33cf583d2b7c62f648737112faff33b59b3b5ae"
	PackagingLiveArtifactChecksumsSHA256 = "3d01aec4fff3967aa9a0c0409a60ed206f30df5cc20235f20fa08b5e8fc14288"
)

// AdoptedPackagingContract is the immutable evaluation-only follow-up of the
// closed-unit packaging replay. It is not a production search or MCP change.
type AdoptedPackagingContract struct {
	SchemaVersion         int      `json:"schema_version"`
	Kind                  string   `json:"kind"`
	ProtocolVersion       string   `json:"protocol_version"`
	SourceExperimentID    string   `json:"source_experiment_id"`
	SourceContractDigest  string   `json:"source_contract_digest"`
	LiveDecision          string   `json:"live_decision"`
	LiveDecisionSHA256    string   `json:"live_decision_sha256"`
	LiveChecksumsSHA256   string   `json:"live_artifact_checksums_sha256"`
	FrozenDigest          string   `json:"frozen_digest"`
	SiblingStatus         string   `json:"sibling_status"`
	SiblingCount          int      `json:"sibling_count"`
	SiblingBytes          int      `json:"sibling_bytes"`
	SiblingEligibility    string   `json:"sibling_eligibility"`
	SiblingBytePolicy     string   `json:"sibling_byte_policy"`
	OneHopStatus          string   `json:"one_hop_status"`
	OneHopKeepProxy       string   `json:"one_hop_keep_proxy"`
	OneHopRejectionReason string   `json:"one_hop_rejection_reason"`
	ArmDAuthorized        bool     `json:"arm_d_authorized"`
	GraphProductPath      string   `json:"graph_product_path"`
	ProductionMCP         bool     `json:"production_mcp"`
	AssistantAB           string   `json:"assistant_ab"`
	Confirmation          string   `json:"confirmation"`
	ClosedCalibration     []string `json:"closed_calibration_units"`
	Digest                string   `json:"digest,omitempty"`
}

func canonicalAdoptedPackagingContract() AdoptedPackagingContract {
	return AdoptedPackagingContract{
		SchemaVersion:         1,
		Kind:                  PackagingAdoptedKind,
		ProtocolVersion:       PackagingProtocolVersion,
		SourceExperimentID:    "relation-packaging-v1",
		SourceContractDigest:  "cb726ace5f81d980260a8111520d5b2f00f9318f128682f3ddc6cc8ff7a54c28",
		LiveDecision:          PackagingDecisionContinueSibling,
		LiveDecisionSHA256:    PackagingLiveDecisionSHA256,
		LiveChecksumsSHA256:   PackagingLiveArtifactChecksumsSHA256,
		FrozenDigest:          packagingFrozenDigest,
		SiblingStatus:         PackagingAdoptedSiblingStatus,
		SiblingCount:          PackagingAdoptedSiblingCount,
		SiblingBytes:          PackagingAdoptedSiblingBytes,
		SiblingEligibility:    PackagingSiblingEligibilityID,
		SiblingBytePolicy:     PackagingSiblingBytePolicyID,
		OneHopStatus:          PackagingAdoptedOneHopStatus,
		OneHopKeepProxy:       PackagingKeepProxyID,
		OneHopRejectionReason: "label-free one-hop proxy completes gg-g09-rename-change on every predeclared grid cell",
		ArmDAuthorized:        false,
		GraphProductPath:      PackagingAdoptedGraphProductPath,
		ProductionMCP:         PackagingAdoptedProductionMCP,
		AssistantAB:           PackagingAdoptedAssistant,
		Confirmation:          PackagingAdoptedConfirmation,
		ClosedCalibration: []string{
			"chi-rhf-32-owner-adopted-dual-ai-v1",
			"relation-calibration-go-git-zustand-memos-40-stage-ef",
		},
	}
}

func adoptedPackagingContractDigest(value AdoptedPackagingContract) (string, error) {
	value.Digest = ""
	return canonicalHash(value)
}

func exactCanonicalAdoptedPackagingContract(value AdoptedPackagingContract) bool {
	expected := canonicalAdoptedPackagingContract()
	want, err := adoptedPackagingContractDigest(expected)
	if err != nil {
		return false
	}
	got, err := adoptedPackagingContractDigest(value)
	if err != nil {
		return false
	}
	return want == got && value.Kind == PackagingAdoptedKind && !value.ProductionMCP && !value.ArmDAuthorized && value.SiblingCount == PackagingAdoptedSiblingCount && value.SiblingBytes == PackagingAdoptedSiblingBytes && value.OneHopStatus == PackagingAdoptedOneHopStatus && value.LiveDecision == PackagingDecisionContinueSibling
}

func validateAdoptedPackagingContract(value AdoptedPackagingContract) error {
	if !exactCanonicalAdoptedPackagingContract(value) {
		return fmt.Errorf("packaging adopted contract is not the frozen evaluation-only sibling cell")
	}
	if value.ProductionMCP || value.ArmDAuthorized || value.GraphProductPath != PackagingAdoptedGraphProductPath {
		return fmt.Errorf("adopted packaging contract cannot authorize production graph or MCP")
	}
	return nil
}
