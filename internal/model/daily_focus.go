package model

import (
	"time"

	"gorm.io/gorm"
)

// DailyFocus is the single user-chosen focus task for one natural day and one
// scene. task_id intentionally has no SQLite foreign key: deleting a task must
// not silently rewrite focus history. Every read re-validates ownership, scene
// and availability, and a stale reference is reported as state=missing.
type DailyFocus struct {
	// UserID, FocusDate and Context form the natural scope of a focus. The
	// composite unique index is the only invariant that prevents a second
	// focus for the same day and scene.
	UserID    uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_daily_focus_scope,priority:1"`
	ID        uint      `json:"id" gorm:"primaryKey"`
	FocusDate string    `json:"focus_date" gorm:"type:date;not null;uniqueIndex:idx_daily_focus_scope,priority:2"`
	Context   string    `json:"context" gorm:"size:10;not null;uniqueIndex:idx_daily_focus_scope,priority:3"`
	TaskID    *uint     `json:"task_id"`
	Revision  int       `json:"revision" gorm:"not null;default:1"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName pins the contract table name. The default GORM pluralizer treats a
// word ending in "s" as already plural and would create "daily_focus".
func (DailyFocus) TableName() string { return "daily_focuses" }

// SQLite may return a DATE column as a timestamp; keep focus_date a calendar
// date so API responses and revision compare-and-set stay stable.
func (focus *DailyFocus) AfterFind(_ *gorm.DB) error {
	if parsed, err := time.Parse(time.RFC3339Nano, focus.FocusDate); err == nil {
		focus.FocusDate = parsed.Format("2006-01-02")
	}
	return nil
}
