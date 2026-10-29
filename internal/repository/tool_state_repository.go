package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// ErrToolStateNotFound means this user has no saved document for the tool yet.
	ErrToolStateNotFound = errors.New("tool state not found")
	// ErrToolStateConflict means the client wrote against a stale revision.
	ErrToolStateConflict = errors.New("tool state revision conflict")
)

type ToolStateRepository struct{ db *gorm.DB }

func NewToolStateRepository(db *gorm.DB) *ToolStateRepository { return &ToolStateRepository{db: db} }

func (r *ToolStateRepository) Get(ctx context.Context, userID uint, toolKey string) (*model.ToolState, error) {
	var row model.ToolState
	err := r.db.WithContext(ctx).Where("user_id = ? AND tool_key = ?", userID, toolKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrToolStateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read tool state: %w", err)
	}
	return &row, nil
}

// Save writes the document with optimistic concurrency. expectedRevision == 0
// creates the first revision; any other value only succeeds when it still
// matches the stored revision, so two windows editing at once cannot silently
// overwrite each other.
func (r *ToolStateRepository) Save(ctx context.Context, userID uint, toolKey, stateJSON string, expectedRevision int64) (*model.ToolState, error) {
	now := time.Now().UTC()
	if expectedRevision > 0 {
		result := r.db.WithContext(ctx).Model(&model.ToolState{}).
			Where("user_id = ? AND tool_key = ? AND revision = ?", userID, toolKey, expectedRevision).
			Updates(map[string]interface{}{"state_json": stateJSON, "revision": expectedRevision + 1, "updated_at": now})
		if result.Error != nil {
			return nil, fmt.Errorf("update tool state: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return nil, ErrToolStateConflict
		}
		return r.Get(ctx, userID, toolKey)
	}

	row := &model.ToolState{UserID: userID, ToolKey: toolKey, StateJSON: stateJSON, Revision: 1}
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "tool_key"}}, DoNothing: true}).
		Create(row)
	if result.Error != nil {
		return nil, fmt.Errorf("create tool state: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		// Another tab created it first; the caller must re-read and retry.
		return nil, ErrToolStateConflict
	}
	return row, nil
}
