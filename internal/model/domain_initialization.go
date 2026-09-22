package model

import "time"

const (
	DomainInitializationStatusSkeletonDraft   = "skeleton_draft"
	DomainInitializationStatusSkeletonReady   = "skeleton_confirmed"
	DomainInitializationStatusStarterExpanded = "starter_expanded"
	DomainInitializationStatusWorldReady      = "world_ready"
	DomainInitializationStatusApplied         = "applied"
	DomainInitializationStatusRejected        = "rejected"
)

type DomainInitializationDraft struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	DomainName            string     `gorm:"size:255;not null;index" json:"domain_name"`
	LearningGoal          string     `gorm:"type:text;not null" json:"learning_goal"`
	TargetDepth           string     `gorm:"size:32;not null" json:"target_depth"`
	Status                string     `gorm:"size:32;not null;index" json:"status"`
	GeneratedBy           string     `gorm:"size:32;not null" json:"generated_by"`
	Provider              string     `gorm:"size:64" json:"provider"`
	Model                 string     `gorm:"size:128" json:"model"`
	SkeletonPromptVersion string     `gorm:"size:128" json:"skeleton_prompt_version"`
	StarterPromptVersion  string     `gorm:"size:128" json:"starter_prompt_version"`
	WorldPromptVersion    string     `gorm:"size:128" json:"world_prompt_version"`
	SkeletonJSON          string     `gorm:"type:text" json:"skeleton_json"`
	StarterBlueprintJSON  string     `gorm:"type:text" json:"starter_blueprint_json"`
	InitialWorldJSON      string     `gorm:"type:text" json:"initial_world_json"`
	AppliedCourseID       *uint      `gorm:"index" json:"applied_course_id"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	AppliedAt             *time.Time `json:"applied_at"`
}
