package model

import "time"

const (
	CognitiveLevelUnseen     = "unseen"
	CognitiveLevelExposed    = "exposed"
	CognitiveLevelRecognize  = "recognize"
	CognitiveLevelUnderstand = "understand"
	CognitiveLevelApply      = "apply"
	CognitiveLevelTransfer   = "transfer"
)

const (
	CognitiveStatusUnknown     = "unknown"
	CognitiveStatusDeveloping  = "developing"
	CognitiveStatusStable      = "stable"
	CognitiveStatusNeedsReview = "needs_review"
)

const (
	CognitiveEvidenceRecognition        = "recognition"
	CognitiveEvidenceConceptExplanation = "concept_explanation"
	CognitiveEvidenceBoundaryAwareness  = "boundary_awareness"
	CognitiveEvidenceApplication        = "application"
	CognitiveEvidenceTransfer           = "transfer"
	CognitiveEvidenceContradiction      = "contradiction"
)

const (
	CognitiveEvidenceSupport    = "support"
	CognitiveEvidenceContradict = "contradict"
)

type CognitiveState struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	CourseID             uint       `gorm:"not null;index" json:"course_id"`
	LessonID             uint       `gorm:"not null;uniqueIndex" json:"lesson_id"`
	CurrentLevel         string     `gorm:"size:32;not null;default:'unseen'" json:"current_level"`
	Status               string     `gorm:"size:32;not null;default:'unknown'" json:"status"`
	UnderstandingSummary string     `gorm:"type:text" json:"understanding_summary"`
	LastLearningTurnID   *uint      `gorm:"index" json:"last_learning_turn_id"`
	LastEvaluatedAt      *time.Time `json:"last_evaluated_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type CognitiveEvidence struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CourseID       uint      `gorm:"not null;index" json:"course_id"`
	LessonID       uint      `gorm:"not null;index" json:"lesson_id"`
	LearningTurnID uint      `gorm:"not null;index;uniqueIndex:idx_cognitive_evidence_turn_index" json:"learning_turn_id"`
	EvidenceIndex  int       `gorm:"not null;uniqueIndex:idx_cognitive_evidence_turn_index" json:"evidence_index"`
	EvidenceType   string    `gorm:"size:32;not null" json:"evidence_type"`
	CognitiveLevel string    `gorm:"size:32;not null" json:"cognitive_level"`
	Polarity       string    `gorm:"size:16;not null" json:"polarity"`
	Description    string    `gorm:"type:text;not null" json:"description"`
	Source         string    `gorm:"size:32;not null" json:"source"`
	CreatedAt      time.Time `json:"created_at"`
}

type CognitiveStateEvent struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	CourseID             uint      `gorm:"not null;index" json:"course_id"`
	LessonID             uint      `gorm:"not null;index" json:"lesson_id"`
	LearningTurnID       uint      `gorm:"not null;index" json:"learning_turn_id"`
	FromLevel            string    `gorm:"size:32" json:"from_level"`
	ToLevel              string    `gorm:"size:32" json:"to_level"`
	FromStatus           string    `gorm:"size:32" json:"from_status"`
	ToStatus             string    `gorm:"size:32" json:"to_status"`
	UnderstandingSummary string    `gorm:"type:text" json:"understanding_summary"`
	Reason               string    `gorm:"type:text" json:"reason"`
	CreatedAt            time.Time `json:"created_at"`
}
