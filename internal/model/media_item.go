package model

import "time"

type MediaItem struct {
	Library     string    `json:"library"`
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"-" gorm:"not null;uniqueIndex:media_origin"`
	Source      string    `json:"Source" gorm:"not null;uniqueIndex:media_origin"`
	SourceID    string    `json:"-" gorm:"not null;uniqueIndex:media_origin"`
	// external_id is the Jellyfin/Calibre item id. The player bridge needs it to
	// resolve PlaybackInfo, and it is not a credential: authorization stays on the
	// Jellyfin side, gated by the bridge library allowlist.
	ExternalID  string    `json:"external_id" gorm:"not null;uniqueIndex:media_origin"`
	Type        string    `json:"Type"`
	Title       string    `json:"Title"`
	Description string    `json:"Description"`
	Authors     []string  `json:"Authors" gorm:"serializer:json"`
	Genres      []string  `json:"Genres" gorm:"serializer:json"`
	Tags        []string  `json:"Tags" gorm:"serializer:json"`
	Year        int       `json:"Year"`
	CoverID     string    `json:"cover_id"`
	Favorite    bool      `json:"favorite"`
	Notes       string    `json:"notes"`
	UpdatedAt   time.Time `json:"updated_at"`
}
