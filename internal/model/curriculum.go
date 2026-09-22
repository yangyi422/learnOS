package model

import "time"

const (
	CurriculumBlueprintStatusDraft    = "draft"
	CurriculumBlueprintStatusActive   = "active"
	CurriculumBlueprintStatusArchived = "archived"
)

const (
	CurriculumBlueprintCreatedByManual = "manual"
	CurriculumBlueprintCreatedBySeed   = "seed"
	CurriculumBlueprintCreatedByAI     = "ai_assisted"
)

const (
	CurriculumGroundingUngrounded  = "ungrounded"
	CurriculumGroundingProvisional = "provisional"
	CurriculumGroundingPartially   = "partially_grounded"
	CurriculumGroundingGrounded    = "grounded"
	CurriculumGroundingConflicted  = "conflicted"
)

const (
	CurriculumImportanceCore        = "core"
	CurriculumImportanceRecommended = "recommended"
	CurriculumImportanceOptional    = "optional"
)

const (
	CurriculumDraftStatusDraft      = "draft"
	CurriculumDraftStatusGenerating = "generating"
	CurriculumDraftStatusApplied    = "applied"
	CurriculumDraftStatusRejected   = "rejected"
)

const (
	CurriculumGeneratedByRule = "rule"
	CurriculumGeneratedByAI   = "ai"
)

const (
	CurriculumUnitUnexpanded = "unexpanded"
	CurriculumUnitExpanding  = "expanding"
	CurriculumUnitExpanded   = "expanded"
)

type CurriculumBlueprint struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	CourseID        *uint     `gorm:"index;uniqueIndex:idx_active_curriculum_blueprint,where:status = 'active'" json:"course_id"`
	Name            string    `gorm:"size:255;not null" json:"name"`
	Domain          string    `gorm:"size:128;not null" json:"domain"`
	Description     string    `gorm:"type:text" json:"description"`
	LearningGoal    string    `gorm:"type:text" json:"learning_goal"`
	Audience        string    `gorm:"type:text" json:"audience"`
	TargetDepth     string    `gorm:"size:64" json:"target_depth"`
	Version         string    `gorm:"size:64;not null" json:"version"`
	Status          string    `gorm:"size:32;not null;index" json:"status"`
	CreatedBy       string    `gorm:"size:32;not null" json:"created_by"`
	GroundingStatus string    `gorm:"size:32;not null" json:"grounding_status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CurriculumBlueprintUnit struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	BlueprintID     uint      `gorm:"not null;index;uniqueIndex:idx_blueprint_unit_key" json:"blueprint_id"`
	Key             string    `gorm:"size:160;not null;uniqueIndex:idx_blueprint_unit_key" json:"key"`
	Title           string    `gorm:"size:255;not null" json:"title"`
	Description     string    `gorm:"type:text" json:"description"`
	SortOrder       int       `gorm:"not null;default:0" json:"sort_order"`
	Importance      string    `gorm:"size:32;not null" json:"importance"`
	ExpansionStatus string    `gorm:"size:32;not null;default:'expanded';index" json:"expansion_status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CurriculumBlueprintLesson struct {
	ID                    uint        `gorm:"primaryKey" json:"id"`
	BlueprintID           uint        `gorm:"not null;index;uniqueIndex:idx_blueprint_lesson_key" json:"blueprint_id"`
	BlueprintUnitID       uint        `gorm:"not null;index" json:"blueprint_unit_id"`
	Key                   string      `gorm:"size:200;not null;uniqueIndex:idx_blueprint_lesson_key" json:"key"`
	Title                 string      `gorm:"size:255;not null" json:"title"`
	Summary               string      `gorm:"type:text" json:"summary"`
	Importance            string      `gorm:"size:32;not null" json:"importance"`
	ContentRole           ContentRole `gorm:"size:32;not null" json:"content_role"`
	DepthLevel            int         `gorm:"not null;default:1" json:"depth_level"`
	AssessmentTargetLevel string      `gorm:"size:32;not null" json:"assessment_target_level"`
	SortOrder             int         `gorm:"not null;default:0" json:"sort_order"`
	AppliedLessonID       *uint       `gorm:"index" json:"applied_lesson_id"`
	GroundingStatus       string      `gorm:"size:32;not null;default:'ungrounded';index" json:"grounding_status"`
	CreatedAt             time.Time   `json:"created_at"`
	UpdatedAt             time.Time   `json:"updated_at"`
}

type CurriculumBlueprintRelation struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	BlueprintID   uint      `gorm:"not null;index;uniqueIndex:idx_blueprint_relation_key" json:"blueprint_id"`
	FromLessonKey string    `gorm:"size:200;not null;uniqueIndex:idx_blueprint_relation_key" json:"from_lesson_key"`
	ToLessonKey   string    `gorm:"size:200;not null;uniqueIndex:idx_blueprint_relation_key" json:"to_lesson_key"`
	RelationType  string    `gorm:"size:32;not null;uniqueIndex:idx_blueprint_relation_key" json:"relation_type"`
	CreatedAt     time.Time `json:"created_at"`
}

type CurriculumDraft struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	CourseID        uint       `gorm:"not null;index;uniqueIndex:idx_pending_curriculum_draft_key,priority:1,where:(status = 'generating' OR status = 'draft') AND pending_key <> ''" json:"course_id"`
	BlueprintID     uint       `gorm:"not null;index" json:"blueprint_id"`
	BlueprintUnitID *uint      `gorm:"index;uniqueIndex:idx_pending_curriculum_draft_key,priority:2,where:(status = 'generating' OR status = 'draft') AND pending_key <> ''" json:"blueprint_unit_id"`
	PendingKey      string     `gorm:"size:255;uniqueIndex:idx_pending_curriculum_draft_key,priority:3,where:(status = 'generating' OR status = 'draft') AND pending_key <> ''" json:"-"`
	Title           string     `gorm:"size:255;not null" json:"title"`
	Summary         string     `gorm:"type:text" json:"summary"`
	Status          string     `gorm:"size:32;not null;index" json:"status"`
	GeneratedBy     string     `gorm:"size:16;not null" json:"generated_by"`
	Provider        string     `gorm:"size:64" json:"provider"`
	Model           string     `gorm:"size:128" json:"model"`
	PromptVersion   string     `gorm:"size:128" json:"prompt_version"`
	ChangeSetJSON   string     `gorm:"type:text;not null" json:"change_set"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	AppliedAt       *time.Time `json:"applied_at"`
}

type CurriculumChangeSet struct {
	NewUnits          []CurriculumDraftUnit        `json:"new_units"`
	NewLessons        []CurriculumDraftLesson      `json:"new_lessons"`
	NewRelations      []CurriculumDraftRelation    `json:"new_relations"`
	BlueprintMappings []CurriculumBlueprintMapping `json:"blueprint_mappings"`
}

type CurriculumDraftUnit struct {
	TempKey          string `json:"temp_key"`
	BlueprintUnitKey string `json:"blueprint_unit_key"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	SortOrder        int    `json:"sort_order"`
}

type CurriculumDraftLesson struct {
	TempKey               string `json:"temp_key"`
	UnitTempKey           string `json:"unit_temp_key"`
	BlueprintLessonKey    string `json:"blueprint_lesson_key"`
	Title                 string `json:"title"`
	Summary               string `json:"summary"`
	CoreQuestion          string `json:"core_question"`
	ExpectedUnderstanding string `json:"expected_understanding"`
	ContentRole           string `json:"content_role"`
	DepthLevel            int    `json:"depth_level"`
	AssessmentTargetLevel string `json:"assessment_target_level"`
	SortOrder             int    `json:"sort_order"`
	IsCore                bool   `json:"is_core"`
}

type CurriculumDraftRelation struct {
	FromKey      string `json:"from_key"`
	ToKey        string `json:"to_key"`
	RelationType string `json:"relation_type"`
}

type CurriculumBlueprintMapping struct {
	BlueprintLessonKey string `json:"blueprint_lesson_key"`
	LessonTempKey      string `json:"lesson_temp_key"`
	AppliedLessonID    *uint  `json:"applied_lesson_id"`
}
