package model

import "time"

// NULL means unrecorded; an explicit zero remains a verified recorded amount.
type AssetSnapshot struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"-" gorm:"not null;uniqueIndex:idx_asset_date,priority:1"`
	SnapshotDate    string    `json:"snapshot_date" gorm:"not null;uniqueIndex:idx_asset_date,priority:2"`
	BankMinor       *int64    `json:"bank_minor,string"`
	PlatformMinor   *int64    `json:"platform_minor,string"`
	InvestmentMinor *int64    `json:"investment_minor,string"`
	OtherMinor      *int64    `json:"other_minor,string"`
	DebtMinor       *int64    `json:"debt_minor,string"`
	Note            string    `json:"note" gorm:"type:text"`
	Version         uint      `json:"version" gorm:"not null"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
