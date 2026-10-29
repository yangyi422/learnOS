package model

import "time"

// Tool keys are part of the API surface: the desktop/web UI addresses a tool's
// saved state by key instead of embedding its data in the library tables.
const (
	ToolKeyFEFWPlanner = "fefw-planner"
)

// ToolState stores one JSON document per user and tool. The payload is opaque to
// the backend (the tool owns its shape and version), so schema changes inside a
// tool never require a migration here.
type ToolState struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_tool_state_user_key,priority:1"`
	ToolKey   string    `json:"tool_key" gorm:"size:64;not null;uniqueIndex:idx_tool_state_user_key,priority:2"`
	StateJSON string    `json:"state_json" gorm:"type:text;not null"`
	Revision  int64     `json:"revision" gorm:"not null;default:1"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
