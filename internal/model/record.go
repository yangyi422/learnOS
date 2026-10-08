package model

import "time"

// LightweightRecord is retained context, independent from executable tasks.
// Nullable unique SourceInboxID prevents duplicate conversion without coupling
// record lifetime to Inbox deletion. Projects are referenced by real IDs.
type LightweightRecord struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	UserID        uint       `json:"user_id" gorm:"not null;index:idx_records_user_created,priority:1;uniqueIndex:idx_records_creation,priority:1"`
	CreationKey   *string    `json:"-" gorm:"size:80;uniqueIndex:idx_records_creation,priority:2"`
	Content       string     `json:"content" gorm:"type:text;not null"`
	ProjectID     *uint      `json:"project_id" gorm:"index"`
	SourceInboxID *uint      `json:"source_inbox_id" gorm:"uniqueIndex"`
	ExternalURL   string     `json:"external_url" gorm:"size:4096"`
	LinkName      string     `json:"link_name" gorm:"size:160"`
	CreatedAt     time.Time  `json:"created_at" gorm:"index:idx_records_user_created,priority:2,sort:desc"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ArchivedAt    *time.Time `json:"archived_at" gorm:"index"`
}
