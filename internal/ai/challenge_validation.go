package ai

import (
	"fmt"
	"strings"
)

func NormalizeAndValidateChallengeGeneration(result ChallengeGenerationResult, challengeType string) (ChallengeGenerationResult, error) {
	result.Prompt = strings.TrimSpace(result.Prompt)
	result.ScenarioContext = strings.TrimSpace(result.ScenarioContext)
	result.TargetLevel = strings.TrimSpace(result.TargetLevel)
	result.WhyThisIsTransfer = strings.TrimSpace(result.WhyThisIsTransfer)
	result.EvaluationCriteria = normalizeStrings(result.EvaluationCriteria)
	result.SourceConcepts = normalizeStrings(result.SourceConcepts)
	if challengeType != "transfer" && challengeType != "misconception_recheck" {
		return ChallengeGenerationResult{}, fmt.Errorf("unsupported challenge type %q", challengeType)
	}
	if result.Prompt == "" || result.ScenarioContext == "" || result.WhyThisIsTransfer == "" {
		return ChallengeGenerationResult{}, fmt.Errorf("challenge prompt, scenario_context and why_this_is_transfer are required")
	}
	expectedTarget := "transfer"
	if challengeType == "misconception_recheck" {
		expectedTarget = "understand"
	}
	if result.TargetLevel != expectedTarget {
		return ChallengeGenerationResult{}, fmt.Errorf("challenge target_level must be %s", expectedTarget)
	}
	if len(result.EvaluationCriteria) < 1 || len(result.EvaluationCriteria) > 6 {
		return ChallengeGenerationResult{}, fmt.Errorf("evaluation_criteria must contain 1 to 6 items")
	}
	if len(result.SourceConcepts) < 1 || len(result.SourceConcepts) > 8 {
		return ChallengeGenerationResult{}, fmt.Errorf("source_concepts must contain 1 to 8 items")
	}
	for _, value := range append(result.EvaluationCriteria, result.SourceConcepts...) {
		if value == "" || len(value) > maxArrayItemLength {
			return ChallengeGenerationResult{}, fmt.Errorf("challenge text item is invalid")
		}
	}
	return result, nil
}

func NormalizeAndValidateChallengeEvaluation(result ChallengeEvaluationResult, challengeType string, targetLevel string, targetMisconceptionID uint) (ChallengeEvaluationResult, error) {
	result.Result = strings.TrimSpace(result.Result)
	result.Feedback = strings.TrimSpace(result.Feedback)
	result.Explanation = strings.TrimSpace(result.Explanation)
	result.DemonstratedLevel = strings.TrimSpace(result.DemonstratedLevel)
	if result.CognitiveEvidence == nil {
		result.CognitiveEvidence = []EvaluationEvidence{}
	}
	if result.Misconceptions == nil {
		result.Misconceptions = []EvaluationMisconception{}
	}
	if len(result.Misconceptions) > maxMisconceptions {
		return ChallengeEvaluationResult{}, fmt.Errorf("too many challenge misconceptions")
	}
	normalizedMisconceptions, misconceptionErr := normalizeEvaluationMisconceptions(result.Misconceptions)
	if misconceptionErr != nil {
		return ChallengeEvaluationResult{}, misconceptionErr
	}
	result.Misconceptions = normalizedMisconceptions
	if _, ok := allowedResults[result.Result]; !ok || result.Feedback == "" || result.Explanation == "" {
		return ChallengeEvaluationResult{}, fmt.Errorf("invalid challenge result fields")
	}
	if _, ok := allowedCognitiveLevels[result.DemonstratedLevel]; !ok || result.DemonstratedLevel == "unseen" {
		return ChallengeEvaluationResult{}, fmt.Errorf("invalid challenge demonstrated_level")
	}
	if cognitiveRank(result.DemonstratedLevel) > cognitiveRank(targetLevel) {
		return ChallengeEvaluationResult{}, fmt.Errorf("challenge demonstrated_level exceeds target")
	}
	if err := validateChallengeEvidence(result.CognitiveEvidence); err != nil {
		return ChallengeEvaluationResult{}, err
	}
	if challengeType == "transfer" && result.MisconceptionValidation != nil {
		return ChallengeEvaluationResult{}, fmt.Errorf("transfer challenge cannot include misconception_validation")
	}
	if challengeType == "misconception_recheck" {
		if result.MisconceptionValidation == nil {
			return ChallengeEvaluationResult{}, fmt.Errorf("misconception_validation is required")
		}
		result.MisconceptionValidation.Status = strings.TrimSpace(result.MisconceptionValidation.Status)
		result.MisconceptionValidation.Evidence = strings.TrimSpace(result.MisconceptionValidation.Evidence)
		if result.MisconceptionValidation.TargetMisconceptionID != targetMisconceptionID || result.MisconceptionValidation.Evidence == "" {
			return ChallengeEvaluationResult{}, fmt.Errorf("misconception_validation target or evidence is invalid")
		}
		if result.MisconceptionValidation.Status != "corrected" && result.MisconceptionValidation.Status != "persists" && result.MisconceptionValidation.Status != "unclear" {
			return ChallengeEvaluationResult{}, fmt.Errorf("unsupported misconception validation status")
		}
	}
	return result, nil
}

func validateChallengeEvidence(evidence []EvaluationEvidence) error {
	if len(evidence) > maxCognitiveEvidence {
		return fmt.Errorf("too many challenge cognitive evidence items")
	}
	for _, item := range evidence {
		if _, ok := allowedEvidenceTypes[item.EvidenceType]; !ok {
			return fmt.Errorf("unsupported challenge evidence type %q", item.EvidenceType)
		}
		if _, ok := allowedCognitiveLevels[item.CognitiveLevel]; !ok || item.CognitiveLevel == "unseen" {
			return fmt.Errorf("unsupported challenge evidence level %q", item.CognitiveLevel)
		}
		if item.Polarity != "support" && item.Polarity != "contradict" || strings.TrimSpace(item.Description) == "" {
			return fmt.Errorf("invalid challenge evidence")
		}
		if cognitiveRank(item.CognitiveLevel) > evidenceMaxRank(item.EvidenceType) {
			return fmt.Errorf("challenge evidence type %q cannot support level %q", item.EvidenceType, item.CognitiveLevel)
		}
	}
	return nil
}

func CalculateChallengePassed(result ChallengeEvaluationResult, challengeType string, targetMisconceptionID uint) bool {
	if challengeType == "transfer" {
		if result.Result != "correct" && result.Result != "mostly_correct" || result.DemonstratedLevel != "transfer" {
			return false
		}
		hasTransfer := false
		for _, evidence := range result.CognitiveEvidence {
			if evidence.EvidenceType == "transfer" && evidence.CognitiveLevel == "transfer" && evidence.Polarity == "support" {
				hasTransfer = true
			}
			if evidence.Polarity == "contradict" {
				return false
			}
		}
		return hasTransfer
	}
	validation := result.MisconceptionValidation
	if validation == nil || validation.TargetMisconceptionID != targetMisconceptionID || validation.Status != "corrected" {
		return false
	}
	if result.Result != "correct" && result.Result != "mostly_correct" {
		return false
	}
	for _, evidence := range result.CognitiveEvidence {
		if evidence.Polarity == "contradict" {
			return false
		}
	}
	return true
}
