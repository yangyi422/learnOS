package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
)

var (
	ErrAPITokenNotFound = errors.New("api token not found")
	ErrAPITokenRevoked  = errors.New("api token revoked")
	ErrAPITokenExpired  = errors.New("api token expired")
)

type APITokenRepository struct{ db *gorm.DB }

func NewAPITokenRepository(db *gorm.DB) *APITokenRepository { return &APITokenRepository{db: db} }

func (r *APITokenRepository) Create(ctx context.Context, token *model.APIToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("create api token: %w", err)
	}
	return nil
}

// ListByUser never returns TokenHash: the secret exists only in the response of
// the create call.
func (r *APITokenRepository) ListByUser(ctx context.Context, userID uint) ([]model.APIToken, error) {
	var tokens []model.APIToken
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC, id DESC").Find(&tokens).Error; err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}
	return tokens, nil
}

// FindByHash returns the token row regardless of state so callers can tell
// "unknown" from "revoked"/"expired" without leaking which one to outsiders.
func (r *APITokenRepository) FindByHash(ctx context.Context, tokenHash string) (*model.APIToken, error) {
	var token model.APIToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAPITokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read api token: %w", err)
	}
	return &token, nil
}

func (r *APITokenRepository) Revoke(ctx context.Context, userID, id uint) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&model.APIToken{}).
		Where("user_id = ? AND id = ? AND revoked_at IS NULL", userID, id).
		Updates(map[string]interface{}{"revoked_at": now})
	if result.Error != nil {
		return fmt.Errorf("revoke api token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAPITokenNotFound
	}
	return nil
}

func (r *APITokenRepository) TouchLastUsed(ctx context.Context, id uint) {
	// Best effort: a failed bookkeeping write must not fail the capture.
	now := time.Now().UTC()
	_ = r.db.WithContext(ctx).Model(&model.APIToken{}).Where("id = ?", id).
		Updates(map[string]interface{}{"last_used_at": now}).Error
}
