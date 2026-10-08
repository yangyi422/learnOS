package repository

import (
	"context"
	"errors"
	"fmt"
	"learnos/internal/links"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInboxCaptureConflict  = errors.New("capture key reused with different content")
	ErrInboxItemNotFound     = errors.New("inbox item not found")
	ErrInboxAlreadyProcessed = errors.New("inbox item already processed")
	ErrInboxAlreadyArchived  = errors.New("inbox item already archived")
)

type InboxRepository struct{ db *gorm.DB }

func NewInboxRepository(db *gorm.DB) *InboxRepository { return &InboxRepository{db: db} }

func (r *InboxRepository) Create(ctx context.Context, item *model.InboxItem) error {
	query := r.db.WithContext(ctx)
	if item.CaptureKey != nil {
		query = query.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "capture_key"}}, DoNothing: true})
	}
	result := query.Create(item)
	if result.Error != nil {
		return fmt.Errorf("create inbox item: %w", result.Error)
	}
	if result.RowsAffected == 0 && item.CaptureKey != nil {
		var existing model.InboxItem
		if err := r.db.WithContext(ctx).Where("user_id = ? AND capture_key = ?", item.UserID, *item.CaptureKey).First(&existing).Error; err != nil {
			return err
		}
		if existing.Content != item.Content {
			return ErrInboxCaptureConflict
		}
		*item = existing
	}
	return nil
}

func (r *InboxRepository) List(ctx context.Context, userID uint, status string) ([]model.InboxItem, map[string]int64, error) {
	var items []model.InboxItem
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at DESC, id DESC").Find(&items).Error; err != nil {
		return nil, nil, fmt.Errorf("list inbox items: %w", err)
	}
	counts := map[string]int64{model.InboxStatusInbox: 0, model.InboxStatusProcessed: 0, model.InboxStatusArchived: 0}
	type countRow struct {
		Status string
		Total  int64
	}
	var rows []countRow
	if err := r.db.WithContext(ctx).Model(&model.InboxItem{}).Select("status, COUNT(*) AS total").Where("user_id = ?", userID).Group("status").Scan(&rows).Error; err != nil {
		return nil, nil, fmt.Errorf("count inbox items: %w", err)
	}
	for _, row := range rows {
		counts[row.Status] = row.Total
	}
	return items, counts, nil
}

func (r *InboxRepository) Find(ctx context.Context, userID, itemID uint) (*model.InboxItem, error) {
	var item model.InboxItem
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", itemID, userID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInboxItemNotFound
		}
		return nil, fmt.Errorf("find inbox item: %w", err)
	}
	return &item, nil
}

func (r *InboxRepository) UpdateContent(ctx context.Context, userID, itemID uint, content string) (*model.InboxItem, error) {
	var item model.InboxItem
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", itemID, userID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInboxItemNotFound
			}
			return err
		}
		if item.Status != model.InboxStatusInbox {
			if item.Status == model.InboxStatusProcessed {
				return ErrInboxAlreadyProcessed
			}
			return ErrInboxAlreadyArchived
		}
		if err := tx.Model(&item).Updates(map[string]interface{}{"content": content, "source_type": sourceType(content), "source_url": sourceURL(content)}).Error; err != nil {
			return err
		}
		return tx.First(&item, itemID).Error
	})
	return &item, err
}

func (r *InboxRepository) Archive(ctx context.Context, userID, itemID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item model.InboxItem
		if err := tx.Where("id = ? AND user_id = ?", itemID, userID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInboxItemNotFound
			}
			return err
		}
		if item.Status == model.InboxStatusArchived {
			return ErrInboxAlreadyArchived
		}
		now := time.Now().UTC()
		return tx.Model(&item).Updates(map[string]interface{}{"status": model.InboxStatusArchived, "archived_at": now}).Error
	})
}

func (r *InboxRepository) Delete(ctx context.Context, userID, itemID uint) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", itemID, userID).Delete(&model.InboxItem{})
	if result.Error != nil {
		return fmt.Errorf("delete inbox item: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrInboxItemNotFound
	}
	return nil
}

func (r *InboxRepository) ConvertToTask(ctx context.Context, userID, itemID uint, input model.ProjectTask, projectID uint, projects *ProjectRepository) (*model.InboxItem, *model.ProjectTask, error) {
	var item model.InboxItem
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", itemID, userID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInboxItemNotFound
			}
			return err
		}
		if item.Status == model.InboxStatusArchived {
			return ErrInboxAlreadyArchived
		}
		if item.Status != model.InboxStatusInbox {
			return ErrInboxAlreadyProcessed
		}

		now := time.Now().UTC()
		claim := tx.Model(&model.InboxItem{}).Where("id = ? AND user_id = ? AND status = ?", itemID, userID, model.InboxStatusInbox).
			Updates(map[string]interface{}{"status": model.InboxStatusProcessed, "processed_at": now})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected != 1 {
			return ErrInboxAlreadyProcessed
		}
		if err := projects.CreateTaskInTx(tx, userID, projectID, &input, true); err != nil {
			return err
		}
		if err := tx.Model(&model.InboxItem{}).Where("id = ? AND user_id = ? AND status = ?", itemID, userID, model.InboxStatusProcessed).
			Updates(map[string]interface{}{"processed_to_type": model.InboxTargetTask, "processed_to_id": input.ID, "processed_at": now}).Error; err != nil {
			return err
		}
		if err := tx.First(&item, itemID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &item, &input, nil
}

func sourceType(content string) string {
	if sourceURL(content) != "" {
		return model.InboxSourceURL
	}
	return model.InboxSourceManual
}

func sourceURL(content string) string {
	if content != "" && links.SafeExternal(content) {
		return content
	}
	return ""
}
