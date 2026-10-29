package model

import (
	"strings"
	"time"
)

// Capture credentials are deliberately narrow: one scope, one endpoint. They
// exist so a third-party client (the uTools quick-capture plugin) can add inbox
// items without ever holding a session cookie or a password.
const (
	TokenScopeCaptureInbox = "capture:inbox"

	// tokenPrefix is stored separately so the UI can identify a credential
	// ("los_pat_3f9c…") without ever persisting the secret.
	TokenPrefixLength = 12
)

type APIToken struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	UserID     uint       `json:"user_id" gorm:"not null;index:idx_api_token_user,priority:1"`
	Name       string     `json:"name" gorm:"size:120;not null"`
	TokenHash  string     `json:"-" gorm:"size:128;not null;uniqueIndex"`
	Prefix     string     `json:"prefix" gorm:"size:32;not null"`
	Scopes     string     `json:"-" gorm:"size:200;not null"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"not null;index"`
	RevokedAt  *time.Time `json:"revoked_at" gorm:"index"`
}

// ScopesList splits the stored comma-separated scopes.
func (t APIToken) ScopesList() []string {
	parts := strings.Split(t.Scopes, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			scopes = append(scopes, trimmed)
		}
	}
	return scopes
}
