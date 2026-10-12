package repository

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"learnos/internal/model"
	"time"
)

type LibraryImageRepository struct{ db *gorm.DB }

func NewLibraryImageRepository(db *gorm.DB) *LibraryImageRepository {
	return &LibraryImageRepository{db.Session(&gorm.Session{Logger: logger.Discard})}
}
func (r *LibraryImageRepository) Create(ctx context.Context, row *model.LibraryImage) error {
	return r.db.WithContext(ctx).Create(row).Error
}
func (r *LibraryImageRepository) Get(ctx context.Context, user uint, id string) (model.LibraryImage, error) {
	var row model.LibraryImage
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, user).First(&row).Error
	return row, err
}
func (r *LibraryImageRepository) Usage(ctx context.Context, user uint) (int64, error) {
	var size int64
	err := r.db.WithContext(ctx).Model(&model.LibraryImage{}).Where("user_id = ?", user).Select("COALESCE(SUM(size),0)").Scan(&size).Error
	return size, err
}
func (r *LibraryImageRepository) Old(ctx context.Context, user uint, before time.Time) ([]model.LibraryImage, error) {
	var rows []model.LibraryImage
	err := r.db.WithContext(ctx).Where("user_id = ? AND created_at < ?", user, before).Find(&rows).Error
	return rows, err
}
func (r *LibraryImageRepository) References(ctx context.Context) ([]string, error) {
	// Conservative scan across all collections: a shared reference never loses its file.
	var pages []model.LibraryPage
	var collections []model.LibraryCollection
	if err := r.db.WithContext(ctx).Select("body", "cover_url").Find(&pages).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Select("cover_url").Find(&collections).Error; err != nil {
		return nil, err
	}
	texts := []string{}
	for _, p := range pages {
		texts = append(texts, p.Body, p.CoverURL)
	}
	for _, c := range collections {
		texts = append(texts, c.CoverURL)
	}
	return texts, nil
}
func (r *LibraryImageRepository) Delete(ctx context.Context, user uint, id string) error {
	return r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, user).Delete(&model.LibraryImage{}).Error
}

func (r *LibraryImageRepository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.LibraryImage{}).Count(&n).Error
	return n, err
}
func (r *LibraryImageRepository) StorageNames(ctx context.Context) (map[string]bool, error) {
	var rows []model.LibraryImage
	if err := r.db.WithContext(ctx).Select("storage_name").Find(&rows).Error; err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, r := range rows {
		names[r.StorageName] = true
	}
	return names, nil
}
