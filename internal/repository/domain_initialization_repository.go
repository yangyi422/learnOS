package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"learnos/internal/auth"
	"learnos/internal/model"
)

type DomainInitializationRepository struct{ db *gorm.DB }

func NewDomainInitializationRepository(db *gorm.DB) *DomainInitializationRepository {
	return &DomainInitializationRepository{db: db}
}

func (r *DomainInitializationRepository) Create(ctx context.Context, draft *model.DomainInitializationDraft) error {
	if principal, ok := auth.PrincipalFromContext(ctx); ok && draft.UserID == 0 {
		draft.UserID = principal.UserID
	}
	if err := r.db.WithContext(ctx).Create(draft).Error; err != nil {
		return fmt.Errorf("create domain initialization draft: %w", err)
	}
	return nil
}

func (r *DomainInitializationRepository) Find(ctx context.Context, id uint) (*model.DomainInitializationDraft, error) {
	var draft model.DomainInitializationDraft
	query := r.db.WithContext(ctx)
	if principal, ok := auth.PrincipalFromContext(ctx); ok {
		query = query.Where("user_id = ?", principal.UserID)
	}
	if err := query.First(&draft, id).Error; err != nil {
		return nil, fmt.Errorf("find domain initialization draft: %w", err)
	}
	return &draft, nil
}

func (r *DomainInitializationRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now().UTC()
	query := r.db.WithContext(ctx).Model(&model.DomainInitializationDraft{}).Where("id = ?", id)
	if principal, ok := auth.PrincipalFromContext(ctx); ok {
		query = query.Where("user_id = ?", principal.UserID)
	}
	if err := query.Updates(updates).Error; err != nil {
		return fmt.Errorf("update domain initialization draft: %w", err)
	}
	return nil
}

func (r *DomainInitializationRepository) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *DomainInitializationRepository) FindCourseByName(ctx context.Context, name string) (*model.Course, error) {
	var course model.Course
	name = strings.TrimSpace(name)
	query := r.db.WithContext(ctx).Where("LOWER(TRIM(name)) = LOWER(?)", name)
	if principal, ok := auth.PrincipalFromContext(ctx); ok {
		query = query.Where("user_id = ?", principal.UserID)
	}
	if err := query.First(&course).Error; err != nil {
		return nil, err
	}
	return &course, nil
}
