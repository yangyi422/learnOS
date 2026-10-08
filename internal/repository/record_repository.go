package repository

import (
	"context"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RecordRepository struct{ db *gorm.DB }

func NewRecordRepository(db *gorm.DB) *RecordRepository { return &RecordRepository{db: db} }
func (r *InboxRepository) Records() *RecordRepository   { return NewRecordRepository(r.db) }

func validateRecordProject(tx *gorm.DB, userID uint, projectID *uint) error {
	if projectID == nil {
		return nil
	}
	var project model.Project
	return tx.Where("id = ? AND user_id = ?", *projectID, userID).First(&project).Error
}
func (r *RecordRepository) List(ctx context.Context, userID uint, projectID *uint, archived bool) ([]model.LightweightRecord, error) {
	records := []model.LightweightRecord{}
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	}
	if archived {
		query = query.Where("archived_at IS NOT NULL")
	} else {
		query = query.Where("archived_at IS NULL")
	}
	err := query.Order("created_at DESC, id DESC").Find(&records).Error
	return records, err
}
func (r *RecordRepository) Find(ctx context.Context, userID, id uint) (*model.LightweightRecord, error) {
	var record model.LightweightRecord
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&record).Error
	return &record, err
}
func (r *RecordRepository) Save(ctx context.Context, userID uint, id uint, input model.LightweightRecord) (*model.LightweightRecord, error) {
	var saved model.LightweightRecord
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := validateRecordProject(tx, userID, input.ProjectID); err != nil {
			return err
		}
		if id == 0 {
			input.UserID = userID
			query := tx
			if input.CreationKey != nil {
				query = query.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "creation_key"}}, DoNothing: true})
			}
			result := query.Create(&input)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 && input.CreationKey != nil {
				if err := tx.Where("user_id = ? AND creation_key = ?", userID, *input.CreationKey).First(&saved).Error; err != nil {
					return err
				}
				sameProject := saved.ProjectID == nil && input.ProjectID == nil || saved.ProjectID != nil && input.ProjectID != nil && *saved.ProjectID == *input.ProjectID
				if saved.Content != input.Content || !sameProject || saved.ExternalURL != input.ExternalURL || saved.LinkName != input.LinkName {
					return ErrInboxCaptureConflict
				}
				return nil
			}
			saved = input
			return nil
		}
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&saved).Error; err != nil {
			return err
		}
		if err := tx.Model(&saved).Updates(map[string]interface{}{"content": input.Content, "project_id": input.ProjectID, "external_url": input.ExternalURL, "link_name": input.LinkName}).Error; err != nil {
			return err
		}
		return tx.First(&saved, id).Error
	})
	return &saved, err
}
func (r *RecordRepository) Archive(ctx context.Context, userID, id uint, archived bool) error {
	var value *time.Time
	if archived {
		now := time.Now().UTC()
		value = &now
	}
	result := r.db.WithContext(ctx).Model(&model.LightweightRecord{}).Where("id = ? AND user_id = ?", id, userID).Updates(map[string]interface{}{"archived_at": value})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *RecordRepository) Delete(ctx context.Context, userID, id uint) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.LightweightRecord{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *RecordRepository) ConvertInbox(ctx context.Context, userID, itemID uint, input model.LightweightRecord) (*model.InboxItem, *model.LightweightRecord, error) {
	var item model.InboxItem
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", itemID, userID).First(&item).Error; err != nil {
			return err
		}
		if item.Status == model.InboxStatusArchived {
			return ErrInboxAlreadyArchived
		}
		if item.Status != model.InboxStatusInbox {
			return ErrInboxAlreadyProcessed
		}
		if err := validateRecordProject(tx, userID, input.ProjectID); err != nil {
			return err
		}
		now := time.Now().UTC()
		claim := tx.Model(&model.InboxItem{}).Where("id = ? AND user_id = ? AND status = ?", itemID, userID, model.InboxStatusInbox).Updates(map[string]interface{}{"status": model.InboxStatusProcessed, "processed_at": now})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected != 1 {
			return ErrInboxAlreadyProcessed
		}
		input.UserID = userID
		input.CreationKey = nil
		input.SourceInboxID = &itemID
		if err := tx.Create(&input).Error; err != nil {
			return err
		}
		if err := tx.Model(&item).Updates(map[string]interface{}{"processed_to_type": model.InboxTargetRecord, "processed_to_id": input.ID}).Error; err != nil {
			return err
		}
		return tx.First(&item, itemID).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return &item, &input, nil
}
