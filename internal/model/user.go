package model

import "time"

type UserRole string

const (
	UserRoleAdmin = UserRole("admin")
	UserRoleUser  = UserRole("user")
)

type UserStatus string

const (
	UserStatusActive  = UserStatus("active")
	UserStatusBlocked = UserStatus("blocked")
)

type User struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	Username     string     `json:"username" gorm:"size:120;not null;uniqueIndex"`
	DisplayName  string     `json:"display_name" gorm:"size:120;not null"`
	PasswordHash string     `json:"-" gorm:"type:text;not null"`
	Role         UserRole   `json:"role" gorm:"size:32;not null;index"`
	Status       UserStatus `json:"status" gorm:"size:32;not null;index"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Session struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	TokenHash string    `json:"-" gorm:"size:128;not null;uniqueIndex"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at"`
}
