package model

import "time"

// References retain context, never a second copy of financial amounts.
type GoalAssetReference struct {
	ID              uint          `json:"id" gorm:"primaryKey"`
	GoalID          uint          `json:"goal_id" gorm:"not null;uniqueIndex:idx_goal_asset,priority:1"`
	AssetSnapshotID uint          `json:"asset_snapshot_id" gorm:"not null;uniqueIndex:idx_goal_asset,priority:2;index"`
	Note            string        `json:"note" gorm:"type:text"`
	JudgmentDate    string        `json:"judgment_date" gorm:"not null"`
	Version         uint          `json:"version" gorm:"not null"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Goal            LifeGoal      `json:"-" gorm:"foreignKey:GoalID;constraint:OnDelete:CASCADE"`
	Snapshot        AssetSnapshot `json:"-" gorm:"foreignKey:AssetSnapshotID;constraint:OnDelete:RESTRICT"`
}
