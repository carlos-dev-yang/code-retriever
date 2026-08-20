package evalcontract

import (
	"fmt"
	"strings"
)

const SchemaVersion = 1

func (value EvaluationCase) Validate() error {
	if value.SchemaVersion != SchemaVersion || value.ID == "" || value.Text == "" || !validLanguage(value.Language) || !validAnswerMode(value.AnswerMode) || (value.Split != Calibration && value.Split != Confirmation) || !uniqueNonEmpty(value.Cohorts) || !validSHA256(value.Digest) || !value.RequiredConstraints.valid() || !value.Review.valid() || len(value.Judgments) == 0 {
		return fmt.Errorf("invalid evaluation case")
	}
	if value.AnswerMode == Abstainable {
		if len(value.RequiredGroups) != 0 || (value.ExpectedCardinality != nil && *value.ExpectedCardinality != 0) || len(value.HardNegatives) == 0 {
			return fmt.Errorf("invalid abstainable case")
		}
	} else if len(value.RequiredGroups) == 0 || (value.ExpectedCardinality != nil && *value.ExpectedCardinality <= 0) {
		return fmt.Errorf("invalid answerable case")
	}
	groupIDs := map[string]struct{}{}
	requiredSpans := map[string]struct{}{}
	for _, group := range value.RequiredGroups {
		if group.ID == "" || len(group.Alternatives) == 0 {
			return fmt.Errorf("invalid required group")
		}
		if _, exists := groupIDs[group.ID]; exists {
			return fmt.Errorf("duplicate required group")
		}
		groupIDs[group.ID] = struct{}{}
		for _, alternative := range group.Alternatives {
			if len(alternative.Spans) == 0 {
				return fmt.Errorf("empty expected alternative")
			}
			for _, span := range alternative.Spans {
				if err := span.Validate(); err != nil {
					return err
				}
				requiredSpans[spanIdentity(span)] = struct{}{}
			}
		}
	}
	hardNegativeSpans := map[string]struct{}{}
	for _, negative := range value.HardNegatives {
		if negative.Reason == "" {
			return fmt.Errorf("hard negative lacks reason")
		}
		if err := negative.Span.Validate(); err != nil {
			return err
		}
		hardNegativeSpans[spanIdentity(negative.Span)] = struct{}{}
	}
	judgments := map[string]Relevance{}
	for _, judgment := range value.Judgments {
		if judgment.Grade < Irrelevant || judgment.Grade > DirectRequirement || judgment.Rationale == "" {
			return fmt.Errorf("invalid relevance judgment")
		}
		if err := judgment.Span.Validate(); err != nil {
			return err
		}
		identity := spanIdentity(judgment.Span)
		if _, exists := judgments[identity]; exists {
			return fmt.Errorf("duplicate or conflicting relevance judgment")
		}
		judgments[identity] = judgment.Grade
	}
	for identity := range requiredSpans {
		if judgments[identity] != DirectRequirement {
			return fmt.Errorf("required alternative lacks grade-2 judgment")
		}
	}
	for identity := range hardNegativeSpans {
		if judgments[identity] != Irrelevant {
			return fmt.Errorf("hard negative lacks grade-0 judgment")
		}
	}
	if value.AssistantTask != nil && (len(value.AssistantTask.Requirements) == 0 || len(value.AssistantTask.ExpectedTestOutcomes) == 0) {
		return fmt.Errorf("invalid assistant task requirements")
	}
	return nil
}
func (value StageTrace) Validate() error {
	if value.SchemaVersion != SchemaVersion || value.QueryID == "" || len(value.Observations) != len(PlannedStages) || (value.TerminalState != TerminalComplete && value.TerminalState != TerminalFailed) {
		return fmt.Errorf("invalid stage trace")
	}
	if !uniqueNonEmpty(value.RequiredGroupIDs) {
		return fmt.Errorf("invalid required group ids")
	}
	lost := map[string]FirstLoss{}
	for index, observation := range value.Observations {
		if observation.Stage != PlannedStages[index] || observation.CandidateCount < 0 || (observation.FailureStage != "" && !validStage(Stage(observation.FailureStage))) {
			return fmt.Errorf("invalid stage observation")
		}
		if observation.Required && (observation.Status != Observed || !validDenominators(observation.Denominators)) {
			return fmt.Errorf("required stage lacks observed denominators")
		}
		if !observation.Required && (observation.Status != ObservationNotObserved || len(observation.Denominators) != 0 || len(observation.GroupObservations) != 0 || observation.FailureStage != "" || observation.CandidateCount != 0 || observation.QueryVectorSHA256 != "") {
			return fmt.Errorf("optional stage must be explicitly not observed")
		}
		if observation.Stage == StageOperational {
			if len(observation.GroupObservations) != 0 {
				return fmt.Errorf("operational stage cannot report evidence-group survival")
			}
			continue
		}
		if observation.Required {
			if !validGroupObservations(observation.GroupObservations, value.RequiredGroupIDs, observation.FailureStage) {
				return fmt.Errorf("required evidence stage lacks complete group observations")
			}
			if index <= stageIndex(StageParserChunker) || index >= stageIndex(StageProviderUnion) {
				for _, group := range observation.GroupObservations {
					if original, exists := lost[group.GroupID]; exists {
						if group.Present || group.FirstLoss != original {
							return fmt.Errorf("required group reappeared or changed first loss")
						}
					} else if !group.Present {
						lost[group.GroupID] = group.FirstLoss
					}
				}
			}
		}
	}
	return nil
}
func (value PromotionResult) Validate() error {
	if value.SchemaVersion != SchemaVersion || !validScope(value.Scope) || (value.Status != PromotionEvidenceReady && value.Status != NotPromotionReady) || len(value.ApplicableGates) == 0 || !validFrozenReviewAuthority(value.ReviewProtocolVersion, value.RelevanceAuthority, value.ReviewValidation) {
		return fmt.Errorf("invalid promotion result")
	}
	if value.Status == PromotionEvidenceReady && (len(value.FailedGates) != 0 || value.IncompleteReason != "") {
		return fmt.Errorf("ready promotion has failed or incomplete gates")
	}
	if value.Status == NotPromotionReady && len(value.FailedGates) == 0 && value.IncompleteReason == "" {
		return fmt.Errorf("not-ready promotion lacks reason")
	}
	return nil
}
func (value EvaluationRunManifest) Validate() error {
	if value.SchemaVersion != SchemaVersion || !validSHA256(value.CorpusManifestSHA256) || !validSHA256(value.QueryManifestSHA256) || !validSHA256(value.ProfileFingerprint) || value.CodeCommit == "" || value.Generation < 0 || value.CandidatePolicy == "" || value.Platform == "" || !value.PairedControls.valid() {
		return fmt.Errorf("invalid evaluation run manifest")
	}
	if (value.ReviewProtocolVersion != "" || value.RelevanceAuthority != "" || value.ReviewValidation != "") && !validFrozenReviewAuthority(value.ReviewProtocolVersion, value.RelevanceAuthority, value.ReviewValidation) {
		return fmt.Errorf("invalid evaluation review authority")
	}
	if value.QuestionSet != nil && (!value.QuestionSet.valid() || value.QuestionSet.SHA256 != value.QueryManifestSHA256) {
		return fmt.Errorf("invalid question set identity")
	}
	return nil
}

func (value QuestionSetIdentity) valid() bool {
	return value.ID != "" && value.Version != "" && validSHA256(value.SHA256) && value.TaxonomyVersion != "" && validSHA256(value.TaxonomySHA256)
}
func (value PromotionContract) Validate() error {
	if value.SchemaVersion != SchemaVersion || !validScope(value.Scope) || len(value.CalibrationEvidenceSHA256) == 0 || len(value.FrozenGates) == 0 || !validSHA256(value.ConfirmationDatasetSHA256) || !value.PairedControls.valid() || !validFrozenReviewAuthority(value.ReviewProtocolVersion, value.RelevanceAuthority, value.ReviewValidation) {
		return fmt.Errorf("invalid promotion contract")
	}
	for _, digest := range value.CalibrationEvidenceSHA256 {
		if !validSHA256(digest) {
			return fmt.Errorf("invalid calibration digest")
		}
	}
	return nil
}
func (value ArtifactManifest) Validate() error {
	paths := map[string]struct{}{}
	for _, entry := range value.Entries {
		if !validRelativePath(entry.Path) || entry.MediaType == "" || entry.ByteSize < 0 || !validSHA256(entry.SHA256) {
			return fmt.Errorf("invalid artifact manifest")
		}
		if _, exists := paths[entry.Path]; exists {
			return fmt.Errorf("duplicate artifact path")
		}
		paths[entry.Path] = struct{}{}
	}
	return value.requireVersion()
}
func (value ArtifactManifest) requireVersion() error {
	if value.SchemaVersion != SchemaVersion {
		return fmt.Errorf("invalid artifact manifest version")
	}
	return nil
}
func (value PairedRunControls) valid() bool {
	return validSHA256(value.CorpusStateSHA256) && validSHA256(value.LabelDigestSHA256) && value.ParserVersion != "" && value.ChunkerVersion != "" && value.FTSSchemaVersion != "" && value.SourceModel != "" && value.SourceDimensions == 1024 && value.ReducerID != "" && (value.ServingDimensions == 1024 || value.ServingDimensions == 512) && value.CandidatePolicy != "" && value.BodyBudget != "" && value.MCPVersion != ""
}
func (value SourceSpan) Validate() error {
	if !validRelativePath(value.Path) || !validSHA256(value.ContentSHA256) || value.QualifiedSymbol == "" || value.StartByte < 0 || value.EndByte <= value.StartByte {
		return fmt.Errorf("invalid source span")
	}
	return nil
}
func validGroupFirstLoss(loss FirstLoss, failure FailureStage) bool {
	if failedStage, operationFailure := operationFailureStage(loss); operationFailure {
		return failure == FailureStage(failedStage)
	}
	for _, known := range []FirstLoss{SourceDiscovery, ParseOrChunk, FTSCandidateMiss, DenseSegmentMiss, ProviderUnionMiss, SegmentParentCollapse, RRFFusion, BodyPackaging, AssistantUse, AssistantResolution, NoLoss} {
		if loss == known {
			return true
		}
	}
	return false
}
func validDenominators(values []DenominatorRecord) bool {
	if len(values) == 0 {
		return false
	}
	names := map[string]struct{}{}
	for _, value := range values {
		if value.Name == "" || value.TruthUnit == "" || value.Count <= 0 {
			return false
		}
		if _, exists := names[value.Name]; exists {
			return false
		}
		names[value.Name] = struct{}{}
	}
	return true
}
func (value RequiredConstraints) valid() bool {
	if len(value.Identifiers) == 0 || len(value.Paths) == 0 || len(value.Languages) == 0 || len(value.Scopes) == 0 {
		return false
	}
	if !uniqueNonEmpty(value.Identifiers) || !uniqueNonEmpty(value.Paths) || !uniqueNonEmpty(value.Scopes) {
		return false
	}
	for _, path := range value.Paths {
		if !validRelativePath(path) {
			return false
		}
	}
	languages := map[Language]struct{}{}
	for _, language := range value.Languages {
		if !validLanguage(language) {
			return false
		}
		if _, exists := languages[language]; exists {
			return false
		}
		languages[language] = struct{}{}
	}
	return true
}
func (value ReviewRecord) valid() bool {
	if (value.State != ReviewDraft && value.State != ReviewFrozen) || len(value.Passes) == 0 || value.Rationale == "" {
		return false
	}
	ids, reviewers := map[string]struct{}{}, map[string]struct{}{}
	for _, pass := range value.Passes {
		if pass.ID == "" || pass.Reviewer == "" {
			return false
		}
		if pass.ArtifactSHA256 != "" && !validSHA256(pass.ArtifactSHA256) {
			return false
		}
		if _, exists := ids[pass.ID]; exists {
			return false
		}
		ids[pass.ID] = struct{}{}
		reviewers[pass.Reviewer] = struct{}{}
	}
	if value.State == ReviewDraft {
		return value.ProtocolVersion == "" && value.RelevanceAuthority == "" && value.ReviewValidation == "" && value.OwnerAdoptionSHA256 == ""
	}
	if len(value.Passes) < 2 || len(reviewers) < 2 || value.ProtocolVersion != ReviewProtocolOwnerAdoptedDualAI || value.RelevanceAuthority != RelevanceAuthorityOwnerAdoptedDualAIReview || value.ReviewValidation != ReviewValidationNoIndependentHumanReview || !validSHA256(value.OwnerAdoptionSHA256) {
		return false
	}
	for _, pass := range value.Passes {
		if !validSHA256(pass.ArtifactSHA256) {
			return false
		}
	}
	return true
}
func validFrozenReviewAuthority(protocol string, authority RelevanceAuthority, validation ReviewValidation) bool {
	return protocol == ReviewProtocolOwnerAdoptedDualAI && authority == RelevanceAuthorityOwnerAdoptedDualAIReview && validation == ReviewValidationNoIndependentHumanReview
}
func uniqueNonEmpty(values []string) bool {
	if len(values) == 0 {
		return true
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}
func validGroupObservations(values []GroupObservation, wanted []string, failure FailureStage) bool {
	if len(values) != len(wanted) {
		return false
	}
	if len(wanted) == 0 {
		return true
	}
	failureAccountedFor := false
	for index, value := range values {
		if value.GroupID != wanted[index] || (value.Present && value.FirstLoss != NoLoss) || (!value.Present && (value.FirstLoss == NoLoss || !validGroupFirstLoss(value.FirstLoss, failure))) {
			return false
		}
		if !value.Present {
			if _, operationFailure := operationFailureStage(value.FirstLoss); operationFailure {
				failureAccountedFor = true
			}
		}
	}
	return failure == "" || failureAccountedFor
}
func operationFailureStage(loss FirstLoss) (Stage, bool) {
	const prefix = "OPERATION_FAILURE:"
	if !strings.HasPrefix(string(loss), prefix) {
		return "", false
	}
	stage := Stage(strings.TrimPrefix(string(loss), prefix))
	return stage, validStage(stage)
}
func spanIdentity(value SourceSpan) string {
	return value.Path + "\x00" + value.ContentSHA256 + "\x00" + value.QualifiedSymbol + fmt.Sprintf("\x00%d\x00%d", value.StartByte, value.EndByte)
}
func stageIndex(stage Stage) int {
	for index, candidate := range PlannedStages {
		if candidate == stage {
			return index
		}
	}
	return len(PlannedStages)
}
func validStage(stage Stage) bool {
	for _, value := range PlannedStages {
		if stage == value {
			return true
		}
	}
	return false
}
func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func validLanguage(value Language) bool {
	return value == Go || value == TypeScript || value == TSX || value == Mixed
}
func validAnswerMode(value AnswerMode) bool {
	return value == Single || value == BestN || value == Exhaustive || value == Abstainable
}
func validScope(value PromotionScope) bool {
	return value == CoreRetrieval || value == ReleaseCandidate
}
