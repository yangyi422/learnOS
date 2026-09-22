package model

import "time"

const (
	ChallengeTypeTransfer             = "transfer"
	ChallengeTypeMisconceptionRecheck = "misconception_recheck"
)

type ReasoningPattern struct {
	PatternKey  string
	Explanation string
}

type MisconceptionObservation struct {
	OriginalUnderstanding string
	CorrectUnderstanding  string
	BoundaryNotes         string
	Patterns              []ReasoningPattern
}

const (
	ChallengeStatusPending   = "pending"
	ChallengeStatusAnswered  = "answered"
	ChallengeStatusCancelled = "cancelled"
)

const (
	TurnKindLessonAnswer         = "lesson_answer"
	TurnKindTransferChallenge    = "transfer_challenge"
	TurnKindMisconceptionRecheck = "misconception_recheck"
)

const (
	MisconceptionStatusActive   = "active"
	MisconceptionStatusResolved = "resolved"
	MisconceptionEventObserved  = "observed"
	MisconceptionEventResolved  = "resolved"
	MisconceptionEventReopened  = "reopened"
	MisconceptionEventConfirmed = "user_confirmed"
	MisconceptionEventCorrected = "user_corrected"
	MisconceptionEventIgnored   = "ignored"
)

const (
	MisconceptionReviewAIInferred    = "ai_inferred"
	MisconceptionReviewUserConfirmed = "user_confirmed"
	MisconceptionReviewUserCorrected = "user_corrected"
	MisconceptionReviewIgnored       = "ignored"
)

type AssessmentChallenge struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	IdempotencyKey         *string   `gorm:"size:128;uniqueIndex" json:"-"`
	CourseID               uint      `gorm:"not null;index" json:"course_id"`
	LessonID               uint      `gorm:"not null;index" json:"lesson_id"`
	ChallengeType          string    `gorm:"size:32;not null;index" json:"challenge_type"`
	TargetMisconceptionID  *uint     `gorm:"index" json:"target_misconception_id"`
	Prompt                 string    `gorm:"type:text;not null" json:"prompt"`
	ScenarioContext        string    `gorm:"type:text" json:"scenario_context"`
	EvaluationCriteriaJSON string    `gorm:"type:text;not null" json:"evaluation_criteria"`
	WhyThisIsTransfer      string    `gorm:"type:text" json:"why_this_is_transfer"`
	SourceConceptsJSON     string    `gorm:"type:text" json:"source_concepts"`
	TargetLevel            string    `gorm:"size:32;not null" json:"target_level"`
	Status                 string    `gorm:"size:32;not null;default:'pending'" json:"status"`
	Provider               string    `gorm:"size:64" json:"provider"`
	Model                  string    `gorm:"size:128" json:"model"`
	PromptVersion          string    `gorm:"size:128" json:"prompt_version"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type ChallengeAttempt struct {
	ID                          uint      `gorm:"primaryKey" json:"id"`
	IdempotencyKey              *string   `gorm:"size:128;uniqueIndex" json:"-"`
	CourseID                    uint      `gorm:"not null;index" json:"course_id"`
	LessonID                    uint      `gorm:"not null;index" json:"lesson_id"`
	ChallengeID                 uint      `gorm:"not null;uniqueIndex" json:"challenge_id"`
	LearningTurnID              uint      `gorm:"not null;index" json:"learning_turn_id"`
	UserAnswer                  string    `gorm:"type:text;not null" json:"user_answer"`
	Result                      string    `gorm:"size:32;not null" json:"result"`
	DemonstratedLevel           string    `gorm:"size:32;not null" json:"demonstrated_level"`
	Passed                      bool      `gorm:"not null;default:false" json:"passed"`
	Feedback                    string    `gorm:"type:text" json:"feedback"`
	Explanation                 string    `gorm:"type:text" json:"explanation"`
	EvidenceJSON                string    `gorm:"type:text" json:"evidence"`
	MisconceptionValidationJSON string    `gorm:"type:text" json:"misconception_validation"`
	CreatedAt                   time.Time `json:"created_at"`
}

type MisconceptionEvent struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	CourseID        uint      `gorm:"not null;index" json:"course_id"`
	LessonID        uint      `gorm:"not null;index" json:"lesson_id"`
	MisconceptionID uint      `gorm:"not null;index" json:"misconception_id"`
	LearningTurnID  *uint     `gorm:"index" json:"learning_turn_id"`
	ChallengeID     *uint     `gorm:"index" json:"challenge_id"`
	EventType       string    `gorm:"size:32;not null" json:"event_type"`
	Notes           string    `gorm:"type:text" json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
}

type MisconceptionPatternLink struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	MisconceptionID uint      `gorm:"not null;index;uniqueIndex:idx_misconception_pattern" json:"misconception_id"`
	PatternKey      string    `gorm:"size:64;not null;index;uniqueIndex:idx_misconception_pattern" json:"pattern_key"`
	Explanation     string    `gorm:"type:text" json:"explanation"`
	Source          string    `gorm:"size:32;not null" json:"source"`
	PromptVersion   string    `gorm:"size:128" json:"prompt_version"`
	CreatedAt       time.Time `json:"created_at"`
}
