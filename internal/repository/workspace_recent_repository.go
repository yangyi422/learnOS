package repository

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

// Bounded projections of existing facts: no body, judgments or financial amounts.
type WorkspaceRecentRepository struct{ db *gorm.DB }

func NewWorkspaceRecentRepository(db *gorm.DB) *WorkspaceRecentRepository {
	return &WorkspaceRecentRepository{db.Session(&gorm.Session{Logger: logger.Discard})}
}

type RecentLibraryPage struct {
	ID             uint      `json:"id"`
	CollectionID   uint      `json:"collection_id"`
	Title          string    `json:"title"`
	CollectionName string    `json:"collection_name"`
	DisplayMode    string    `json:"display_mode"`
	CoverURL       string    `json:"cover_url"`
	UpdatedAt      time.Time `json:"updated_at"`
}
type RecentLifeEvent struct {
	ID            uint   `json:"id"`
	Title         string `json:"title"`
	OccurredOn    string `json:"occurred_on"`
	PrimaryDomain string `json:"primary_domain"`
}
type RecentGoalEntry struct {
	ID         uint   `json:"id"`
	GoalID     uint   `json:"goal_id"`
	GoalTitle  string `json:"goal_title"`
	Content    string `json:"content"`
	OccurredOn string `json:"occurred_on"`
}

func (r *WorkspaceRecentRepository) Library(ctx context.Context, user uint) ([]RecentLibraryPage, error) {
	rows := []RecentLibraryPage{}
	err := r.db.WithContext(ctx).Table("library_pages p").
		Select("p.id, p.collection_id, p.title, c.name AS collection_name, c.display_mode, p.cover_url, p.updated_at").
		Joins("JOIN library_collections c ON c.id = p.collection_id").Where("c.user_id = ?", user).
		Order("p.updated_at DESC, p.id DESC").Limit(3).Scan(&rows).Error
	return rows, err
}
func (r *WorkspaceRecentRepository) Events(ctx context.Context, user uint) ([]RecentLifeEvent, error) {
	rows := []RecentLifeEvent{}
	err := r.db.WithContext(ctx).Table("life_events").Select("id, title, occurred_on, primary_domain").Where("user_id = ?", user).
		Order("occurred_on DESC, id DESC").Limit(2).Scan(&rows).Error
	return rows, err
}
func (r *WorkspaceRecentRepository) Goals(ctx context.Context, user uint) ([]RecentGoalEntry, error) {
	rows := []RecentGoalEntry{}
	err := r.db.WithContext(ctx).Table("life_goal_entries e").
		Select("e.id, e.goal_id, g.title AS goal_title, substr(e.content, 1, 100) AS content, e.occurred_on").
		Joins("JOIN life_goals g ON g.id = e.goal_id").Where("g.user_id = ?", user).
		Order("e.occurred_on DESC, e.id DESC").Limit(1).Scan(&rows).Error
	return rows, err
}
func (r *WorkspaceRecentRepository) AssetDate(ctx context.Context, user uint) (string, error) {
	var rows []struct{ SnapshotDate string }
	err := r.db.WithContext(ctx).Table("asset_snapshots").Select("snapshot_date").Where("user_id = ?", user).
		Order("snapshot_date DESC, id DESC").Limit(1).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return "", err
	}
	return rows[0].SnapshotDate, nil
}
