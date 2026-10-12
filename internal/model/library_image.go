package model

import "time"

type LibraryImage struct {
	ID           string    `json:"id" gorm:"primaryKey;size:32"`
	UserID       uint      `json:"-" gorm:"not null;index"`
	StorageName  string    `json:"-" gorm:"not null;uniqueIndex"`
	MIME         string    `json:"mime"`
	Size         int64     `json:"size"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	SHA256       string    `json:"-"`
	OriginalName string    `json:"original_name"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
	URL          string    `json:"url" gorm:"-"`
}
