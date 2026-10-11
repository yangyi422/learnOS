package repository

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"learnos/internal/model"
	"time"
)

var ErrAssetDateExists = errors.New("asset snapshot date already exists")
var ErrAssetChanged = errors.New("asset snapshot changed")

type AssetRepository struct{ db *gorm.DB }

func NewAssetRepository(db *gorm.DB) *AssetRepository {
	return &AssetRepository{db.Session(&gorm.Session{Logger: logger.Discard})}
}
func (r *AssetRepository) List(ctx context.Context, user uint) ([]model.AssetSnapshot, error) {
	rows := []model.AssetSnapshot{}
	err := r.db.WithContext(ctx).Where("user_id = ?", user).Order("snapshot_date DESC, id DESC").Find(&rows).Error
	return rows, err
}
func (r *AssetRepository) Get(ctx context.Context, user, id uint) (model.AssetSnapshot, error) {
	var row model.AssetSnapshot
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", user, id).First(&row).Error
	return row, err
}
func (r *AssetRepository) Save(ctx context.Context, row model.AssetSnapshot) (model.AssetSnapshot, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.ID == 0 {
			row.Version = 1
			q := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "snapshot_date"}}, DoNothing: true}).Create(&row)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected != 1 {
				return ErrAssetDateExists
			}
			return nil
		}
		var old model.AssetSnapshot
		if err := tx.Where("user_id = ? AND id = ?", row.UserID, row.ID).First(&old).Error; err != nil {
			return err
		}
		if old.Version != row.Version {
			return ErrAssetChanged
		}
		var collision int64
		if err := tx.Model(&model.AssetSnapshot{}).Where("user_id = ? AND snapshot_date = ? AND id <> ?", row.UserID, row.SnapshotDate, row.ID).Count(&collision).Error; err != nil {
			return err
		}
		if collision > 0 {
			return ErrAssetDateExists
		}
		q := tx.Model(&model.AssetSnapshot{}).Where("id = ? AND user_id = ? AND version = ?", row.ID, row.UserID, row.Version).Updates(map[string]interface{}{
			"snapshot_date": row.SnapshotDate, "bank_minor": row.BankMinor, "platform_minor": row.PlatformMinor, "investment_minor": row.InvestmentMinor, "other_minor": row.OtherMinor, "debt_minor": row.DebtMinor, "note": row.Note, "version": row.Version + 1, "updated_at": time.Now().UTC(),
		})
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 {
			return ErrAssetChanged
		}
		return tx.First(&row, row.ID).Error
	})
	return row, err
}
func (r *AssetRepository) Delete(ctx context.Context, user, id, version uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.AssetSnapshot
		if err := tx.Where("user_id = ? AND id = ?", user, id).First(&row).Error; err != nil {
			return err
		}
		if row.Version != version {
			return ErrAssetChanged
		}
		// The FK also protects against direct deletes and concurrent references.
		if tx.Migrator().HasTable(&model.GoalAssetReference{}) {
			var count int64
			if err := tx.Model(&model.GoalAssetReference{}).Where("asset_snapshot_id = ?", id).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return ErrAssetReferenced
			}
		}
		return tx.Where("user_id = ? AND id = ? AND version = ?", user, id, version).Delete(&model.AssetSnapshot{}).Error
	})
}
