package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"learnos/internal/model"
)

type MediaRepository struct{ db *gorm.DB }

func NewMediaRepository(db *gorm.DB) *MediaRepository {
	return &MediaRepository{db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})}
}
func (r *MediaRepository) Upsert(ctx context.Context, items []model.MediaItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "source"}, {Name: "source_id"}, {Name: "external_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"type", "title", "description", "authors", "genres", "tags", "year", "cover_id", "updated_at"}),
		}).Create(&items).Error
	})
}
func (r *MediaRepository) List(ctx context.Context, user uint, kind, source, search string, page int) ([]model.MediaItem, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.MediaItem{}).Where("user_id = ?", user)
	if kind != "" {
		q = q.Where("type = ?", kind)
	}
	if source != "" {
		q = q.Where("source = ?", source)
	}
	if search != "" {
		search = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(search)
		q = q.Where("title LIKE ? ESCAPE '!'", "%"+search+"%")
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	items := make([]model.MediaItem, 0)
	err := q.Order("id DESC").Offset((page - 1) * 24).Limit(24).Find(&items).Error
	return items, count, err
}
