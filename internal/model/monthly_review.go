package model

import "time"

// Facts are immutable server-generated snapshots, including pending previews.
type MonthlyReviewFacts struct {
	ID                   uint       `json:"id" gorm:"primaryKey"`
	UserID               uint       `json:"-" gorm:"not null;index"`
	Month                string     `json:"month" gorm:"not null;index"`
	BaseVersion          uint       `json:"base_version"`
	FactsJSON            string     `json:"facts_json,omitempty" gorm:"type:text;not null"`
	GeneratedAt          time.Time  `json:"generated_at"`
	ConfirmedAt          *time.Time `json:"confirmed_at"`
	ArchivedPersonalJSON string     `json:"archived_personal_json,omitempty" gorm:"type:text"`
}
type MonthlyReview struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"-" gorm:"not null;uniqueIndex:idx_review_month,priority:1"`
	Month       string    `json:"month" gorm:"not null;uniqueIndex:idx_review_month,priority:2"`
	FactsID     uint      `json:"facts_id" gorm:"not null"`
	Memorable   string    `json:"memorable" gorm:"type:text"`
	Thoughts    string    `json:"thoughts" gorm:"type:text"`
	NextFocus   string    `json:"next_focus" gorm:"type:text"`
	ExternalURL string    `json:"external_url"`
	Version     uint      `json:"version" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
