package model

import "time"

type Project struct {
	ID             uint          `json:"id" gorm:"primaryKey"`
	UserID         uint          `json:"user_id" gorm:"not null;index"`
	Title          string        `json:"title" gorm:"size:160;not null"`
	Description    string        `json:"description" gorm:"type:text"`
	Status         string        `json:"status" gorm:"size:20;not null;index"`
	Icon           string        `json:"icon" gorm:"size:32"`
	Accent         string        `json:"accent" gorm:"size:32"`
	ArchivedAt     *time.Time    `json:"archived_at"`
	Tasks          []ProjectTask `json:"tasks" gorm:"foreignKey:ProjectID"`
	OpenTaskCount  int64         `json:"open_task_count" gorm:"-"`
	DoingTaskCount int64         `json:"doing_task_count" gorm:"-"`
	DoneTaskCount  int64         `json:"done_task_count" gorm:"-"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type ProjectTask struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	ProjectID    uint       `json:"project_id" gorm:"not null;index"`
	Title        string     `json:"title" gorm:"size:200;not null"`
	Description  string     `json:"description" gorm:"type:text"`
	Status       string     `json:"status" gorm:"size:20;not null"`
	SortOrder    int64      `json:"sort_order" gorm:"not null;default:0"`
	Priority     string     `json:"priority" gorm:"size:10;not null;default:normal"`
	DueDate      *string    `json:"due_date" gorm:"type:date"`
	CompletedAt  *time.Time `json:"completed_at"`
	IsNextAction bool       `json:"is_next_action" gorm:"not null;default:false"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
