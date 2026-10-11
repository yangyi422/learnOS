package repository

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"learnos/internal/model"
	"time"
)

var ErrAssetReferenced = errors.New("asset snapshot referenced by goals")

func referenceParents(tx *gorm.DB, user, goal, asset uint) error {
	if err := validateLifeGoal(tx, user, &goal); err != nil {
		return err
	}
	var row model.AssetSnapshot
	return tx.Where("user_id = ? AND id = ?", user, asset).First(&row).Error
}
func (r *AssetRepository) GoalReferences(ctx context.Context, user, goal uint) ([]model.GoalAssetReference, error) {
	rows := []model.GoalAssetReference{}
	tx := r.db.WithContext(ctx)
	if err := validateLifeGoal(tx, user, &goal); err != nil {
		return rows, err
	}
	err := tx.Joins("JOIN asset_snapshots a ON a.id = goal_asset_references.asset_snapshot_id AND a.user_id = ?", user).Where("goal_asset_references.goal_id = ?", goal).Order("a.snapshot_date DESC, goal_asset_references.id DESC").Find(&rows).Error
	return rows, err
}
func (r *AssetRepository) ReferencingGoals(ctx context.Context, user, asset uint) ([]model.LifeGoal, error) {
	rows := []model.LifeGoal{}
	if _, err := r.Get(ctx, user, asset); err != nil {
		return rows, err
	}
	err := r.db.WithContext(ctx).Model(&model.LifeGoal{}).Select("life_goals.*").Joins("JOIN goal_asset_references r ON r.goal_id = life_goals.id").Where("life_goals.user_id = ? AND r.asset_snapshot_id = ?", user, asset).Order("life_goals.id DESC").Find(&rows).Error
	return rows, err
}
func (r *AssetRepository) SaveReference(ctx context.Context, user uint, row model.GoalAssetReference, create bool) (model.GoalAssetReference, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := referenceParents(tx, user, row.GoalID, row.AssetSnapshotID); err != nil {
			return err
		}
		if create {
			row.Version = 1
			q := tx.Omit("Goal", "Snapshot").Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "goal_id"}, {Name: "asset_snapshot_id"}}, DoNothing: true}).Create(&row)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				var old model.GoalAssetReference
				if err := tx.Where("goal_id = ? AND asset_snapshot_id = ?", row.GoalID, row.AssetSnapshotID).First(&old).Error; err != nil {
					return err
				}
				if old.Note != row.Note || old.JudgmentDate != row.JudgmentDate {
					return ErrLifeConflict
				}
				row = old
			}
			return nil
		}
		q := tx.Model(&model.GoalAssetReference{}).Where("goal_id = ? AND asset_snapshot_id = ? AND version = ?", row.GoalID, row.AssetSnapshotID, row.Version).Updates(map[string]interface{}{"note": row.Note, "judgment_date": row.JudgmentDate, "version": row.Version + 1, "updated_at": time.Now().UTC()})
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 {
			return ErrAssetChanged
		}
		return tx.Where("goal_id = ? AND asset_snapshot_id = ?", row.GoalID, row.AssetSnapshotID).First(&row).Error
	})
	return row, err
}
func (r *AssetRepository) DeleteReference(ctx context.Context, user, goal, asset, version uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := referenceParents(tx, user, goal, asset); err != nil {
			return err
		}
		q := tx.Where("goal_id = ? AND asset_snapshot_id = ? AND version = ?", goal, asset, version).Delete(&model.GoalAssetReference{})
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 {
			return ErrAssetChanged
		}
		return nil
	})
}
