package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) List(ctx context.Context) ([]model.User, error) {
	var users []model.User
	if err := r.db.WithContext(ctx).Order("created_at ASC, id ASC").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) Update(ctx context.Context, id uint, values map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(values).Error
}

func (r *UserRepository) CreateSession(ctx context.Context, session *model.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *UserRepository) FindSession(ctx context.Context, tokenHash string) (*model.Session, error) {
	var session model.Session
	if err := r.db.WithContext(ctx).Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now().UTC()).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *UserRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&model.Session{}).Error
}

func (r *UserRepository) Bootstrap(ctx context.Context, username, displayName, passwordHash string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = model.User{Username: username, DisplayName: displayName, PasswordHash: passwordHash, Role: model.UserRoleAdmin, Status: model.UserStatusActive}
		if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) AssignUnownedCourses(ctx context.Context, userID uint) error {
	if err := r.db.WithContext(ctx).Model(&model.Course{}).Where("user_id = 0").Update("user_id", userID).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&model.DomainInitializationDraft{}).Where("user_id = 0").Update("user_id", userID).Error
}
