package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

var allowedResults = map[string]struct{}{
	"correct":           {},
	"mostly_correct":    {},
	"partially_correct": {},
	"incorrect":         {},
	"insufficient":      {},
}

var allowedCognitiveLevels = map[string]struct{}{
	"exposed": {}, "recognize": {}, "understand": {}, "apply": {}, "transfer": {},
}

var allowedEvidenceTypes = map[string]struct{}{
	"recognition": {}, "concept_explanation": {}, "boundary_awareness": {},
	"application": {}, "transfer": {}, "contradiction": {},
}

var allowedEvidencePolarities = map[string]struct{}{
	"support": {}, "contradict": {},
}

var allowedReasoningPatterns = map[string]struct{}{
	"binary_thinking": {}, "single_factor_reasoning": {}, "overgeneralization": {},
	"boundary_neglect": {}, "dose_neglect": {}, "correlation_causation": {},
	"category_confusion": {}, "unsupported_assumption": {},
}

const (
	maxCorrectParts       = 8
	maxMissingParts       = 8
	maxMisconceptions     = 5
	maxBoundaryConditions = 6
	maxMasteryEvidence    = 8
	maxEvidenceUsed       = 8
	maxCognitiveEvidence  = 8
	maxReasoningPatterns  = 2
	maxStringLength       = 4000
	maxArrayItemLength    = 2000
)

func ParseAndValidate(content string) (EvaluationResult, error) {
	return ParseAndValidateForTarget(content, "")
}

func ParseAndValidateForTarget(content, target string) (EvaluationResult, error) {
	if strings.TrimSpace(content) == "" {
		return EvaluationResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("empty content"))
	}
	decoder := json.NewDecoder(bytes.NewBufferString(content))
	decoder.DisallowUnknownFields()
	var result EvaluationResult
	if err := decoder.Decode(&result); err != nil {
		return EvaluationResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("decode evaluation JSON: %w", err))
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return EvaluationResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("evaluation JSON has trailing content"))
	}
	result, err := NormalizeAndValidateForTarget(result, target)
	if err != nil {
		return EvaluationResult{}, newProviderError(ErrInvalidResponse, err)
	}
	return result, nil
}

func NormalizeAndValidate(result EvaluationResult) (EvaluationResult, error) {
	return NormalizeAndValidateForTarget(result, "")
}

func NormalizeAndValidateForTarget(result EvaluationResult, target string) (EvaluationResult, error) {
	result.Result = strings.TrimSpace(result.Result)
	result.Feedback = strings.TrimSpace(result.Feedback)
	result.Explanation = strings.TrimSpace(result.Explanation)
	result.CorrectParts = normalizeStrings(result.CorrectParts)
	result.MissingParts = normalizeStrings(result.MissingParts)
	if result.Misconceptions == nil {
		result.Misconceptions = []EvaluationMisconception{}
	}
	result.BoundaryConditions = normalizeStrings(result.BoundaryConditions)
	result.MasteryEvidence = normalizeStrings(result.MasteryEvidence)
	result.EvidenceUsed = normalizeStrings(result.EvidenceUsed)
	result.Uncertainty = strings.TrimSpace(result.Uncertainty)
	result.RecommendedNextAction = strings.TrimSpace(result.RecommendedNextAction)
	result.DemonstratedLevel = strings.TrimSpace(result.DemonstratedLevel)
	result.UserUnderstandingSummary = strings.TrimSpace(result.UserUnderstandingSummary)
	if result.CognitiveEvidence == nil {
		result.CognitiveEvidence = []EvaluationEvidence{}
	}
	normalizedMisconceptions, err := normalizeEvaluationMisconceptions(result.Misconceptions)
	if err != nil {
		return EvaluationResult{}, err
	}
	result.Misconceptions = normalizedMisconceptions

	if _, ok := allowedResults[result.Result]; !ok {
		return EvaluationResult{}, fmt.Errorf("unsupported result %q", result.Result)
	}
	if result.Feedback == "" || result.Explanation == "" {
		return EvaluationResult{}, fmt.Errorf("feedback and explanation are required")
	}
	if _, ok := allowedCognitiveLevels[result.DemonstratedLevel]; !ok {
		return EvaluationResult{}, fmt.Errorf("unsupported demonstrated_level %q", result.DemonstratedLevel)
	}
	if result.UserUnderstandingSummary == "" {
		return EvaluationResult{}, fmt.Errorf("user_understanding_summary is required")
	}
	if len(result.UserUnderstandingSummary) > maxStringLength {
		return EvaluationResult{}, fmt.Errorf("user_understanding_summary is too long")
	}
	if target != "" {
		if _, ok := map[string]struct{}{"recognize": {}, "understand": {}, "apply": {}, "transfer": {}}[target]; !ok {
			return EvaluationResult{}, fmt.Errorf("unsupported assessment target %q", target)
		}
		if cognitiveRank(result.DemonstratedLevel) > cognitiveRank(target) {
			return EvaluationResult{}, fmt.Errorf("demonstrated_level %q exceeds assessment target %q", result.DemonstratedLevel, target)
		}
	}
	if result.MasteryScore < 0 || result.MasteryScore > 1 {
		return EvaluationResult{}, fmt.Errorf("mastery_score must be between 0 and 1")
	}
	if result.Confidence <= 0 || result.Confidence > 1 {
		return EvaluationResult{}, fmt.Errorf("confidence must be greater than 0 and at most 1")
	}
	if result.Uncertainty == "" || result.RecommendedNextAction == "" {
		return EvaluationResult{}, fmt.Errorf("uncertainty and recommended_next_action are required")
	}
	if len(result.Uncertainty) > maxArrayItemLength || len(result.RecommendedNextAction) > maxArrayItemLength {
		return EvaluationResult{}, fmt.Errorf("uncertainty or recommended_next_action is too long")
	}
	if len(result.Feedback) > maxStringLength || len(result.Explanation) > maxStringLength {
		return EvaluationResult{}, fmt.Errorf("feedback or explanation is too long")
	}
	if err := validateStrings("correct_parts", result.CorrectParts, maxCorrectParts); err != nil {
		return EvaluationResult{}, err
	}
	if err := validateStrings("missing_parts", result.MissingParts, maxMissingParts); err != nil {
		return EvaluationResult{}, err
	}
	if err := validateStrings("boundary_conditions", result.BoundaryConditions, maxBoundaryConditions); err != nil {
		return EvaluationResult{}, err
	}
	if err := validateStrings("mastery_evidence", result.MasteryEvidence, maxMasteryEvidence); err != nil {
		return EvaluationResult{}, err
	}
	if err := validateStrings("evidence_used", result.EvidenceUsed, maxEvidenceUsed); err != nil {
		return EvaluationResult{}, err
	}
	if len(result.EvidenceUsed) == 0 {
		return EvaluationResult{}, fmt.Errorf("evidence_used is required")
	}
	eligible := (result.Result == "correct" || result.Result == "mostly_correct") && cognitiveRank(result.DemonstratedLevel) >= cognitiveRank("understand")
	result.TransferChallengeEligible = eligible
	if len(result.Misconceptions) > maxMisconceptions {
		return EvaluationResult{}, fmt.Errorf("too many misconceptions")
	}
	if len(result.CognitiveEvidence) > maxCognitiveEvidence {
		return EvaluationResult{}, fmt.Errorf("too many cognitive_evidence items")
	}
	for index := range result.CognitiveEvidence {
		evidence := &result.CognitiveEvidence[index]
		evidence.EvidenceType = strings.TrimSpace(evidence.EvidenceType)
		evidence.CognitiveLevel = strings.TrimSpace(evidence.CognitiveLevel)
		evidence.Polarity = strings.TrimSpace(evidence.Polarity)
		evidence.Description = strings.TrimSpace(evidence.Description)
		if _, ok := allowedEvidenceTypes[evidence.EvidenceType]; !ok {
			return EvaluationResult{}, fmt.Errorf("unsupported evidence_type %q", evidence.EvidenceType)
		}
		if _, ok := allowedCognitiveLevels[evidence.CognitiveLevel]; !ok {
			return EvaluationResult{}, fmt.Errorf("unsupported evidence cognitive_level %q", evidence.CognitiveLevel)
		}
		if _, ok := allowedEvidencePolarities[evidence.Polarity]; !ok {
			return EvaluationResult{}, fmt.Errorf("unsupported evidence polarity %q", evidence.Polarity)
		}
		if evidence.Description == "" || len(evidence.Description) > maxArrayItemLength {
			return EvaluationResult{}, fmt.Errorf("invalid cognitive evidence description")
		}
		if evidence.CognitiveLevel != "exposed" && cognitiveRank(evidence.CognitiveLevel) > evidenceMaxRank(evidence.EvidenceType) {
			return EvaluationResult{}, fmt.Errorf("evidence type %q cannot support level %q", evidence.EvidenceType, evidence.CognitiveLevel)
		}
	}
	for _, misconception := range result.Misconceptions {
		if misconception.OriginalUnderstanding == "" || misconception.CorrectUnderstanding == "" || misconception.BoundaryNotes == "" {
			return EvaluationResult{}, fmt.Errorf("misconception fields are required")
		}
		if len(misconception.OriginalUnderstanding) > maxArrayItemLength || len(misconception.CorrectUnderstanding) > maxArrayItemLength || len(misconception.BoundaryNotes) > maxArrayItemLength {
			return EvaluationResult{}, fmt.Errorf("misconception field is too long")
		}
	}
	return result, nil
}

func normalizeEvaluationMisconceptions(misconceptions []EvaluationMisconception) ([]EvaluationMisconception, error) {
	if misconceptions == nil {
		return []EvaluationMisconception{}, nil
	}
	for i := range misconceptions {
		misconceptions[i].OriginalUnderstanding = strings.TrimSpace(misconceptions[i].OriginalUnderstanding)
		misconceptions[i].CorrectUnderstanding = strings.TrimSpace(misconceptions[i].CorrectUnderstanding)
		misconceptions[i].BoundaryNotes = strings.TrimSpace(misconceptions[i].BoundaryNotes)
		if misconceptions[i].ReasoningPatterns == nil {
			misconceptions[i].ReasoningPatterns = []EvaluationReasoningPattern{}
		}
		if len(misconceptions[i].ReasoningPatterns) > maxReasoningPatterns {
			return nil, fmt.Errorf("too many reasoning_patterns")
		}
		seen := map[string]struct{}{}
		for index := range misconceptions[i].ReasoningPatterns {
			pattern := &misconceptions[i].ReasoningPatterns[index]
			pattern.PatternKey = strings.TrimSpace(pattern.PatternKey)
			pattern.Explanation = strings.TrimSpace(pattern.Explanation)
			if _, ok := allowedReasoningPatterns[pattern.PatternKey]; !ok {
				return nil, fmt.Errorf("unsupported reasoning pattern %q", pattern.PatternKey)
			}
			if _, duplicate := seen[pattern.PatternKey]; duplicate {
				return nil, fmt.Errorf("duplicate reasoning pattern %q", pattern.PatternKey)
			}
			seen[pattern.PatternKey] = struct{}{}
			if pattern.Explanation == "" || len(pattern.Explanation) > maxArrayItemLength {
				return nil, fmt.Errorf("invalid reasoning pattern explanation")
			}
		}
	}
	return misconceptions, nil
}

func cognitiveRank(level string) int {
	switch level {
	case "unseen":
		return 0
	case "exposed":
		return 1
	case "recognize":
		return 2
	case "understand":
		return 3
	case "apply":
		return 4
	case "transfer":
		return 5
	default:
		return -1
	}
}

func evidenceMaxRank(evidenceType string) int {
	switch evidenceType {
	case "recognition":
		return 2
	case "concept_explanation", "boundary_awareness", "contradiction":
		return 3
	case "application":
		return 4
	case "transfer":
		return 5
	default:
		return -1
	}
}

func normalizeStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	return values
}

func validateStrings(name string, values []string, max int) error {
	if len(values) > max {
		return fmt.Errorf("too many %s", name)
	}
	for _, value := range values {
		if value == "" {
			return fmt.Errorf("%s contains an empty item", name)
		}
		if len(value) > maxArrayItemLength {
			return fmt.Errorf("%s item is too long", name)
		}
	}
	return nil
}
