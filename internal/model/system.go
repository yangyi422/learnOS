package model

import "time"

const CurrentSchemaVersion = 15

// SystemMetadata is a singleton record describing the application schema that
// last completed startup migration. It contains no user secrets.
type SystemMetadata struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SchemaVersion  int       `gorm:"not null" json:"schema_version"`
	LastMigratedAt time.Time `gorm:"not null" json:"last_migrated_at"`
	AppVersion     string    `gorm:"size:32;not null" json:"app_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AIConfiguration is the singleton runtime AI configuration. APIKey is never
// serialized in API responses; it is only read by the configured provider.
type AIConfiguration struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Provider  string    `gorm:"size:32;not null" json:"provider"`
	APIKey    string    `gorm:"type:text" json:"-"`
	BaseURL   string    `gorm:"size:512;not null" json:"base_url"`
	Model     string    `gorm:"size:128;not null" json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
