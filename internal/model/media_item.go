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
	Rating      float64   `json:"Rating,omitempty"`
	CoverID     string    `json:"cover_id"`
	// Episode columns: zero values for movies, series and books. Season/Episode
	// numbers are 1-based except specials (season 0), so 0 is a valid episode number.
	SeriesExternalID string `json:"series_external_id,omitempty"`
	SeasonExternalID string `json:"season_external_id,omitempty"`
	SeasonNumber     int    `json:"season_number,omitempty"`
	EpisodeNumber    int    `json:"episode_number,omitempty"`
	PremiereDate     string `json:"premiere_date,omitempty"`
	RuntimeSeconds   int    `json:"runtime_seconds,omitempty"`
	Favorite    bool      `json:"favorite"`
	Notes       string    `json:"notes"`
	UpdatedAt   time.Time `json:"updated_at"`
}
