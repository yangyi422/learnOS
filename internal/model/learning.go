package model

import "time"

type CourseUnitStatus string

const (
	CourseUnitStatusPending   CourseUnitStatus = "pending"
	CourseUnitStatusLearning  CourseUnitStatus = "learning"
	CourseUnitStatusCompleted CourseUnitStatus = "completed"
)

type LessonStatus string

const (
	LessonStatusPending   LessonStatus = "pending"
	LessonStatusLearning  LessonStatus = "learning"
	LessonStatusCompleted LessonStatus = "completed"
)

const (
	AssessmentTargetRecognize  = "recognize"
	AssessmentTargetUnderstand = "understand"
	AssessmentTargetApply      = "apply"
	AssessmentTargetTransfer   = "transfer"
)

type ContentRole string

const (
	ContentRoleFoundation  ContentRole = "foundation"
	ContentRoleCore        ContentRole = "core"
	ContentRoleApplication ContentRole = "application"
	ContentRoleExtension   ContentRole = "extension"
)

type LessonRelationType string

const (
	LessonRelationPrerequisite LessonRelationType = "prerequisite"
	LessonRelationExtends      LessonRelationType = "extends"
	LessonRelationApplication  LessonRelationType = "application"
	LessonRelationRelated      LessonRelationType = "related"
)

type CourseUnit struct {
	ID        uint             `gorm:"primaryKey" json:"id"`
	CourseID  uint             `gorm:"not null;index" json:"course_id"`
	Title     string           `gorm:"size:255;not null" json:"title"`
	Objective string           `gorm:"type:text" json:"objective"`
	SortOrder int              `gorm:"not null;default:0" json:"sort_order"`
	Status    CourseUnitStatus `gorm:"size:32;not null;default:'pending'" json:"status"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type Lesson struct {
	ID                    uint         `gorm:"primaryKey" json:"id"`
	CourseID              uint         `gorm:"not null;index" json:"course_id"`
	UnitID                uint         `gorm:"not null;index" json:"unit_id"`
	Title                 string       `gorm:"size:255;not null" json:"title"`
	CoreQuestion          string       `gorm:"type:text;not null" json:"core_question"`
	ExpectedUnderstanding string       `gorm:"type:text" json:"expected_understanding"`
	SortOrder             int          `gorm:"not null;default:0" json:"sort_order"`
	Status                LessonStatus `gorm:"size:32;not null;default:'pending'" json:"status"`
	IsCore                bool         `gorm:"not null;default:true" json:"is_core"`
	ContentRole           ContentRole  `gorm:"size:32;not null;default:'core'" json:"content_role"`
	DepthLevel            int          `gorm:"not null;default:1" json:"depth_level"`
	AssessmentTargetLevel string       `gorm:"size:32;not null;default:'understand'" json:"assessment_target_level"`
	CreatedAt             time.Time    `json:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at"`
}

type LessonRelation struct {
	ID           uint               `gorm:"primaryKey" json:"id"`
	CourseID     uint               `gorm:"not null;index;uniqueIndex:idx_lesson_relation_identity" json:"course_id"`
	FromLessonID uint               `gorm:"not null;index;uniqueIndex:idx_lesson_relation_identity" json:"from_lesson_id"`
	ToLessonID   uint               `gorm:"not null;index;uniqueIndex:idx_lesson_relation_identity" json:"to_lesson_id"`
	RelationType LessonRelationType `gorm:"size:32;not null;index;uniqueIndex:idx_lesson_relation_identity" json:"relation_type"`
	CreatedAt    time.Time          `json:"created_at"`
}

type LearningTurn struct {
	ID                       uint      `gorm:"primaryKey" json:"id"`
	CourseID                 uint      `gorm:"not null;index" json:"course_id"`
	UnitID                   uint      `gorm:"not null;index" json:"unit_id"`
	LessonID                 uint      `gorm:"not null;index" json:"lesson_id"`
	TurnKind                 string    `gorm:"size:32;not null;default:'lesson_answer';index" json:"turn_kind"`
	ChallengeID              *uint     `gorm:"index" json:"challenge_id"`
	Question                 string    `gorm:"type:text;not null" json:"question"`
	UserAnswer               string    `gorm:"type:text;not null" json:"user_answer"`
	Result                   string    `gorm:"size:32;not null" json:"result"`
	Feedback                 string    `gorm:"type:text" json:"feedback"`
	Explanation              string    `gorm:"type:text" json:"explanation"`
	CorrectParts             string    `gorm:"type:text" json:"correct_parts"`
	MissingParts             string    `gorm:"type:text" json:"missing_parts"`
	MisconceptionsJSON       string    `gorm:"type:text" json:"misconceptions"`
	BoundaryConditions       string    `gorm:"type:text" json:"boundary_conditions"`
	MasteryEvidence          string    `gorm:"type:text" json:"mastery_evidence"`
	EvaluationSource         string    `gorm:"size:32;not null;default:'mock'" json:"evaluation_source"`
	Provider                 string    `gorm:"size:64" json:"provider"`
	Model                    string    `gorm:"size:128" json:"model"`
	PromptVersion            string    `gorm:"size:128" json:"prompt_version"`
	DemonstratedLevel        string    `gorm:"size:32" json:"demonstrated_level"`
	UserUnderstandingSummary string    `gorm:"type:text" json:"user_understanding_summary"`
	CognitiveEvidenceJSON    string    `gorm:"type:text" json:"cognitive_evidence"`
	MasteryScore             float64   `gorm:"not null;default:0" json:"mastery_score"`
	NeedsReview              bool      `gorm:"not null;default:false" json:"needs_review"`
	CreatedAt                time.Time `json:"created_at"`
}

type MasteryRecord struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	CourseID       uint       `gorm:"not null;index" json:"course_id"`
	LessonID       uint       `gorm:"not null;uniqueIndex" json:"lesson_id"`
	MasteryScore   float64    `gorm:"not null;default:0" json:"mastery_score"`
	AnswerCount    int        `gorm:"not null;default:0" json:"answer_count"`
	IncorrectCount int        `gorm:"not null;default:0" json:"incorrect_count"`
	NeedsReview    bool       `gorm:"not null;default:false" json:"needs_review"`
	NextReviewAt   *time.Time `json:"next_review_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Misconception struct {
	ID                       uint       `gorm:"primaryKey" json:"id"`
	CourseID                 uint       `gorm:"not null;index" json:"course_id"`
	LessonID                 uint       `gorm:"not null;index" json:"lesson_id"`
	OriginalUnderstanding    string     `gorm:"type:text" json:"original_understanding"`
	CorrectUnderstanding     string     `gorm:"type:text" json:"correct_understanding"`
	BoundaryNotes            string     `gorm:"type:text" json:"boundary_notes"`
	Status                   string     `gorm:"size:32;not null;default:'active'" json:"status"`
	ResolvedAt               *time.Time `json:"resolved_at"`
	ResolvedByLearningTurnID *uint      `gorm:"index" json:"resolved_by_learning_turn_id"`
	ResolvedByChallengeID    *uint      `gorm:"index" json:"resolved_by_challenge_id"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

type AIEvaluationRunStatus string

const (
	AIEvaluationRunStatusSuccess AIEvaluationRunStatus = "success"
	AIEvaluationRunStatusFailed  AIEvaluationRunStatus = "failed"
)

type AIEvaluationRun struct {
	ID             uint                  `gorm:"primaryKey" json:"id"`
	CourseID       uint                  `gorm:"not null;index" json:"course_id"`
	LessonID       uint                  `gorm:"not null;index" json:"lesson_id"`
	LearningTurnID *uint                 `gorm:"index" json:"learning_turn_id"`
	Provider       string                `gorm:"size:64;not null" json:"provider"`
	Model          string                `gorm:"size:128;not null" json:"model"`
	PromptVersion  string                `gorm:"size:128;not null" json:"prompt_version"`
	RunType        string                `gorm:"size:32;not null;default:'lesson_evaluation'" json:"run_type"`
	Status         AIEvaluationRunStatus `gorm:"size:32;not null;index" json:"status"`
	AttemptCount   int                   `gorm:"not null;default:0" json:"attempt_count"`
	LatencyMS      int64                 `gorm:"not null;default:0" json:"latency_ms"`
	InputTokens    int                   `gorm:"not null;default:0" json:"input_tokens"`
	OutputTokens   int                   `gorm:"not null;default:0" json:"output_tokens"`
	RawResponse    string                `gorm:"type:text" json:"raw_response"`
	ErrorMessage   string                `gorm:"type:text" json:"error_message"`
	CreatedAt      time.Time             `json:"created_at"`
}
