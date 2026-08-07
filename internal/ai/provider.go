package ai

import (
	"context"
	"errors"
)

const (
	LegacyPromptVersion             = "learnos-evaluator-v1"
	Phase5PromptVersion             = "learnos-evaluator-v2"
	PromptVersion                   = "learnos-evaluator-v3"
	TransferGeneratorPromptVersion  = "learnos-transfer-generator-v1"
	ChallengeEvaluatorPromptVersion = "learnos-challenge-evaluator-v1"
	DefaultBaseURL                  = "https://api.deepseek.com"
	DefaultModel                    = "deepseek-v4-flash"
)

var (
	ErrNotConfigured   = errors.New("ai not configured")
	ErrTimeout         = errors.New("ai timeout")
	ErrProvider        = errors.New("ai provider error")
	ErrInvalidResponse = errors.New("ai invalid response")
)

type EvaluationRequest struct {
	CourseName            string
	UnitTitle             string
	LessonTitle           string
	CoreQuestion          string
	ExpectedUnderstanding string
	AssessmentTargetLevel string
	UserAnswer            string
}

type EvaluationMisconception struct {
	OriginalUnderstanding string                       `json:"original_understanding"`
	CorrectUnderstanding  string                       `json:"correct_understanding"`
	BoundaryNotes         string                       `json:"boundary_notes"`
	ReasoningPatterns     []EvaluationReasoningPattern `json:"reasoning_patterns"`
}

type EvaluationReasoningPattern struct {
	PatternKey  string `json:"pattern_key"`
	Explanation string `json:"explanation"`
}

type EvaluationEvidence struct {
	EvidenceType   string `json:"evidence_type"`
	CognitiveLevel string `json:"cognitive_level"`
	Polarity       string `json:"polarity"`
	Description    string `json:"description"`
}

type EvaluationResult struct {
	Result                   string                    `json:"result"`
	Feedback                 string                    `json:"feedback"`
	Explanation              string                    `json:"explanation"`
	CorrectParts             []string                  `json:"correct_parts"`
	MissingParts             []string                  `json:"missing_parts"`
	Misconceptions           []EvaluationMisconception `json:"misconceptions"`
	BoundaryConditions       []string                  `json:"boundary_conditions"`
	MasteryEvidence          []string                  `json:"mastery_evidence"`
	MasteryScore             float64                   `json:"mastery_score"`
	NeedsReview              bool                      `json:"needs_review"`
	DemonstratedLevel        string                    `json:"demonstrated_level"`
	UserUnderstandingSummary string                    `json:"user_understanding_summary"`
	CognitiveEvidence        []EvaluationEvidence      `json:"cognitive_evidence"`
}

type ProviderMeta struct {
	Provider      string
	Model         string
	PromptVersion string
	LatencyMS     int64
	RawResponse   string
	InputTokens   int
	OutputTokens  int
	AttemptCount  int
}

type AIProvider interface {
	EvaluateLessonAnswer(context.Context, EvaluationRequest) (EvaluationResult, ProviderMeta, error)
}

type ProviderError struct {
	Code       error
	StatusCode int
	Err        error
}

func (e *ProviderError) Error() string {
	if e == nil || e.Code == nil {
		return ErrProvider.Error()
	}
	return e.Code.Error()
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Code
}

func newProviderError(code, err error) error {
	return &ProviderError{Code: code, Err: err}
}

func ErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrNotConfigured):
		return "AI_NOT_CONFIGURED"
	case errors.Is(err, ErrTimeout):
		return "AI_TIMEOUT"
	case errors.Is(err, ErrInvalidResponse):
		return "AI_INVALID_RESPONSE"
	case errors.Is(err, ErrProvider):
		return "AI_PROVIDER_ERROR"
	default:
		return "AI_PROVIDER_ERROR"
	}
}

// ErrorDetail exposes the safe validation/provider cause for server logs.
// It intentionally excludes request headers, credentials, and raw model text.
func ErrorDetail(err error) string {
	if err == nil {
		return ""
	}
	var providerErr *ProviderError
	if errors.As(err, &providerErr) && providerErr.Err != nil {
		return ErrorDetail(providerErr.Err)
	}
	return err.Error()
}
