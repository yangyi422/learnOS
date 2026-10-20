package model

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	Context           string        `json:"context" gorm:"size:20;not null;default:unclassified;index:idx_projects_user_context,priority:2"`
	CompletedAt       *time.Time    `json:"completed_at"`
	ID                uint          `json:"id" gorm:"primaryKey"`
	UserID            uint          `json:"user_id" gorm:"not null;index;index:idx_projects_user_context,priority:1"`
	Title             string        `json:"title" gorm:"size:160;not null"`
	Description       string        `json:"description" gorm:"type:text"`
	Status            string        `json:"status" gorm:"size:20;not null;index"`
	Icon              string        `json:"icon" gorm:"size:32"`
	Accent            string        `json:"accent" gorm:"size:32"`
	ArchivedAt        *time.Time    `json:"archived_at"`
	Tasks             []ProjectTask `json:"tasks" gorm:"foreignKey:ProjectID"`
	OpenTaskCount     int64         `json:"open_task_count" gorm:"-"`
	DoingTaskCount    int64         `json:"doing_task_count" gorm:"-"`
	NextTaskCount     int64         `json:"next_task_count" gorm:"-"`
	ArchivedTaskCount int64         `json:"archived_task_count" gorm:"-"`
	DoneTaskCount     int64         `json:"done_task_count" gorm:"-"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type ProjectTask struct {
	ArchivedAt        *time.Time `json:"archived_at" gorm:"index"`
	ActualCompletedAt *time.Time `json:"actual_completed_at"`
	ID                uint       `json:"id" gorm:"primaryKey"`
	ProjectID         uint       `json:"project_id" gorm:"not null;index"`
	Title             string     `json:"title" gorm:"size:200;not null"`
	Description       string     `json:"description" gorm:"type:text"`
	Status            string     `json:"status" gorm:"size:20;not null"`
	SortOrder         int64      `json:"sort_order" gorm:"not null;default:0"`
	Priority          string     `json:"priority" gorm:"size:10;not null;default:normal"`
	PlanDate          *string    `json:"plan_date" gorm:"type:date;index"`
	DueDate           *string    `json:"due_date" gorm:"type:date"`
	CompletedAt       *time.Time `json:"completed_at"`
	IsNextAction      bool       `json:"is_next_action" gorm:"not null;default:false"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// SQLite's date scanner may return an RFC3339 timestamp for a DATE column.
// Keep due_date and plan_date as calendar dates after every database read.
func (task *ProjectTask) AfterFind(_ *gorm.DB) error {
	for _, value := range []**string{&task.DueDate, &task.PlanDate} {
		if *value == nil {
			continue
		}
		if parsed, err := time.Parse(time.RFC3339Nano, **value); err == nil {
			date := parsed.Format("2006-01-02")
			*value = &date
		}
	}
	return nil
}
