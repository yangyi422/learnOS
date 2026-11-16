package model

import "time"

type MediaItem struct {
	Library     string    `json:"library"`
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"-" gorm:"not null;uniqueIndex:media_origin"`
	Source      string    `json:"Source" gorm:"not null;uniqueIndex:media_origin"`
	SourceID    string    `json:"-" gorm:"not null;uniqueIndex:media_origin"`
	ExternalID  string    `json:"-" gorm:"not null;uniqueIndex:media_origin"`
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
