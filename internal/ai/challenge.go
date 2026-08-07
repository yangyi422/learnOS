package ai

import "context"

type ChallengeProvider interface {
	GenerateChallenge(context.Context, ChallengeGenerationRequest) (ChallengeGenerationResult, ProviderMeta, error)
	EvaluateChallenge(context.Context, ChallengeEvaluationRequest) (ChallengeEvaluationResult, ProviderMeta, error)
}

type ChallengeGenerationRequest struct {
	CourseName               string
	UnitTitle                string
	LessonTitle              string
	CoreQuestion             string
	ExpectedUnderstanding    string
	PrerequisiteLessonTitles []string
	RelationContext          []string
	CurrentCognitiveLevel    string
	UnderstandingSummary     string
	ActiveMisconceptions     []EvaluationMisconception
	TargetMisconception      *EvaluationMisconception
	ChallengeType            string
}

type ChallengeGenerationResult struct {
	Prompt             string   `json:"prompt"`
	ScenarioContext    string   `json:"scenario_context"`
	TargetLevel        string   `json:"target_level"`
	EvaluationCriteria []string `json:"evaluation_criteria"`
	WhyThisIsTransfer  string   `json:"why_this_is_transfer"`
	SourceConcepts     []string `json:"source_concepts"`
}

type MisconceptionValidation struct {
	TargetMisconceptionID uint   `json:"target_misconception_id"`
	Status                string `json:"status"`
	Evidence              string `json:"evidence"`
}

type ChallengeEvaluationRequest struct {
	ChallengeType         string
	CourseName            string
	UnitTitle             string
	LessonTitle           string
	CoreQuestion          string
	ExpectedUnderstanding string
	ChallengePrompt       string
	ScenarioContext       string
	EvaluationCriteria    []string
	TargetLevel           string
	TargetMisconception   *EvaluationMisconception
	TargetMisconceptionID uint
	UserAnswer            string
}

type ChallengeEvaluationResult struct {
	Result                  string                    `json:"result"`
	Feedback                string                    `json:"feedback"`
	Explanation             string                    `json:"explanation"`
	DemonstratedLevel       string                    `json:"demonstrated_level"`
	CognitiveEvidence       []EvaluationEvidence      `json:"cognitive_evidence"`
	Misconceptions          []EvaluationMisconception `json:"misconceptions"`
	MisconceptionValidation *MisconceptionValidation  `json:"misconception_validation"`
}
