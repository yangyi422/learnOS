package model

import "time"

// Life facts deliberately keep source IDs and a title snapshot without cascading
// source foreign keys: deleting a course must not delete a personal memory.
type LifeEvent struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"-" gorm:"not null;index;uniqueIndex:idx_life_event_key,priority:1;uniqueIndex:idx_life_event_source,priority:1"`
	CreationKey     *string   `json:"-" gorm:"uniqueIndex:idx_life_event_key,priority:2"`
	Title           string    `json:"title"`
	OccurredOn      string    `json:"occurred_on" gorm:"type:text;not null;index"`
	Description     string    `json:"description" gorm:"type:text"`
	PrimaryDomain   string    `json:"primary_domain"`
	SecondaryDomain string    `json:"secondary_domain"`
	Milestone       bool      `json:"milestone"`
	GoalID          *uint     `json:"goal_id" gorm:"index"`
	ExternalURL     string    `json:"external_url"`
	LinkName        string    `json:"link_name"`
	SourceType      string    `json:"source_type"`
	SourceID        *uint     `json:"source_id"`
	SourceKey       *string   `json:"-" gorm:"uniqueIndex:idx_life_event_source,priority:2"`
	SourceTitle     string    `json:"source_title"`
	SourceAvailable bool      `json:"source_available" gorm:"-"`
	SourceStatus    string    `json:"source_status" gorm:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type LifeGoal struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	UserID           uint      `json:"-" gorm:"not null;index;uniqueIndex:idx_life_goal_key,priority:1"`
	CreationKey      *string   `json:"-" gorm:"uniqueIndex:idx_life_goal_key,priority:2"`
	Title            string    `json:"title"`
	Why              string    `json:"why" gorm:"type:text"`
	CurrentNote      string    `json:"current_note" gorm:"type:text"`
	Status           string    `json:"status"`
	Domain           string    `json:"domain"`
	ProjectID        *uint     `json:"project_id" gorm:"index"`
	ExternalURL      string    `json:"external_url"`
	LinkName         string    `json:"link_name"`
	ProjectAvailable bool      `json:"project_available" gorm:"-"`
	ProjectTitle     string    `json:"project_title" gorm:"-"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type LifeGoalEntry struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	GoalID      uint      `json:"goal_id" gorm:"not null;index;uniqueIndex:idx_life_entry_key,priority:1"`
	CreationKey *string   `json:"-" gorm:"uniqueIndex:idx_life_entry_key,priority:2"`
	OccurredOn  string    `json:"occurred_on" gorm:"type:text;not null"`
	Content     string    `json:"content" gorm:"type:text"`
	Reason      string    `json:"reason" gorm:"type:text"`
	FromStatus  string    `json:"from_status"`
	ToStatus    string    `json:"to_status"`
	CreatedAt   time.Time `json:"created_at"`
}
