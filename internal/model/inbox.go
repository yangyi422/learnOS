package model

import "time"

const (
	InboxStatusInbox     = "inbox"
	InboxStatusProcessed = "processed"
	InboxStatusArchived  = "archived"
	InboxSourceManual    = "manual"
	InboxSourceURL       = "url"
	InboxSourceSystem    = "system"
	InboxTargetTask      = "task"
	InboxTargetRecord    = "record"
)

type InboxItem struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	UserID          uint       `json:"user_id" gorm:"not null;index:idx_inbox_user_status_created,priority:1;uniqueIndex:idx_inbox_capture,priority:1"`
	CaptureKey      *string    `json:"-" gorm:"size:80;uniqueIndex:idx_inbox_capture,priority:2"`
	Content         string     `json:"content" gorm:"type:text;not null"`
	Status          string     `json:"status" gorm:"size:20;not null;default:inbox;index:idx_inbox_user_status_created,priority:2"`
	SourceType      string     `json:"source_type" gorm:"size:20;not null;default:manual"`
	SourceURL       string     `json:"source_url" gorm:"size:2048"`
	ProcessedToType string     `json:"processed_to_type" gorm:"size:32"`
	ProcessedToID   *uint      `json:"processed_to_id"`
	CreatedAt       time.Time  `json:"created_at" gorm:"index:idx_inbox_user_status_created,priority:3,sort:desc"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ProcessedAt     *time.Time `json:"processed_at"`
	ArchivedAt      *time.Time `json:"archived_at"`
}
