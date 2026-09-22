package model

import "time"

const (
	ExplorationDirectionAdjacent    = "adjacent"
	ExplorationDirectionCrossDomain = "cross_domain"
	ExplorationDirectionUnknown     = "unknown"
)

const (
	ExplorationDirectionActive    = "active"
	ExplorationDirectionSaved     = "saved"
	ExplorationDirectionDismissed = "dismissed"
	ExplorationDirectionOpened    = "opened"
)

const (
	ExplorationGeneratedByRule = "rule"
	ExplorationGeneratedByAI   = "ai"
)

const (
	ExplorationQuestionDeepen        = "deepen"
	ExplorationQuestionConnect       = "connect"
	ExplorationQuestionChallenge     = "challenge"
	ExplorationQuestionUnfamiliar    = "unfamiliar"
	ExplorationQuestionOpen          = "open"
	ExplorationQuestionExploring     = "exploring"
	ExplorationQuestionLater         = "later"
	ExplorationQuestionResolved      = "resolved"
	ExplorationQuestionArchived      = "archived"
	ExplorationQuestionFromDirection = "exploration_direction"
)

const (
	ExplorationPriorityLow    = "low"
	ExplorationPriorityNormal = "normal"
	ExplorationPriorityHigh   = "high"
)

type ExplorationDirection struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	CourseID          uint      `gorm:"not null;index" json:"course_id"`
	SourceLessonID    *uint     `gorm:"index" json:"source_lesson_id"`
	TargetCourseID    uint      `gorm:"not null;index" json:"target_course_id"`
	TargetLessonID    uint      `gorm:"not null;index" json:"target_lesson_id"`
	DirectionType     string    `gorm:"size:32;not null;index" json:"direction_type"`
	Title             string    `gorm:"size:255;not null" json:"title"`
	Summary           string    `gorm:"type:text" json:"summary"`
	WhyWorthExploring string    `gorm:"type:text;not null" json:"why_worth_exploring"`
	Score             float64   `gorm:"not null;default:0;index" json:"score"`
	ReasonCode        string    `gorm:"size:64;not null;index" json:"reason_code"`
	ReasonDataJSON    string    `gorm:"type:text" json:"reason_data"`
	Status            string    `gorm:"size:32;not null;default:'active';index" json:"status"`
	GeneratedBy       string    `gorm:"size:16;not null;default:'rule'" json:"generated_by"`
	Provider          string    `gorm:"size:64" json:"provider"`
	Model             string    `gorm:"size:128" json:"model"`
	PromptVersion     string    `gorm:"size:128" json:"prompt_version"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ExplorationQuestion struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	CourseID          uint      `gorm:"not null;index" json:"course_id"`
	SourceDirectionID *uint     `gorm:"index" json:"source_direction_id"`
	SourceLessonID    *uint     `gorm:"index" json:"source_lesson_id"`
	TargetCourseID    uint      `gorm:"not null;index" json:"target_course_id"`
	TargetLessonID    uint      `gorm:"not null;index" json:"target_lesson_id"`
	Question          string    `gorm:"type:text;not null" json:"question"`
	Context           string    `gorm:"type:text" json:"context"`
	WhyThisQuestion   string    `gorm:"type:text;not null" json:"why_this_question"`
	QuestionType      string    `gorm:"size:32;not null;index" json:"question_type"`
	Status            string    `gorm:"size:32;not null;default:'open';index" json:"status"`
	Priority          string    `gorm:"size:16;not null;default:'normal';index" json:"priority"`
	Origin            string    `gorm:"size:64;not null;default:'exploration_direction'" json:"origin"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
