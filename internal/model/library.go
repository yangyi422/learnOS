package model

import "time"

// A collection owns root pages and one optional level of sections.
type LibraryCollection struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"-" gorm:"not null;index"`
	Name        string    `json:"name"`
	Description string    `json:"description" gorm:"type:text"`
	Category    string    `json:"category"`
	Template    string    `json:"template"`
	DisplayMode string    `json:"display_mode"`
	CoverURL    string    `json:"cover_url"`
	Version     uint      `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type LibrarySection struct {
	ID           uint              `json:"id" gorm:"primaryKey"`
	CollectionID uint              `json:"collection_id" gorm:"not null;index"`
	Collection   LibraryCollection `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	Name         string            `json:"name"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}
type LibrarySource struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	Note string `json:"note"`
}
type LibraryPage struct {
	ID           uint              `json:"id" gorm:"primaryKey"`
	CollectionID uint              `json:"collection_id" gorm:"not null;index"`
	Collection   LibraryCollection `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	SectionID    *uint             `json:"section_id" gorm:"index"`
	Section      *LibrarySection   `json:"-" gorm:"constraint:OnDelete:RESTRICT"`
	Title        string            `json:"title"`
	Body         string            `json:"body" gorm:"type:text"`
	Tags         []string          `json:"tags" gorm:"serializer:json;type:text"`
	Sources      []LibrarySource   `json:"sources" gorm:"serializer:json;type:text"`
	CoverURL     string            `json:"cover_url"`
	Version      uint              `json:"version"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}
