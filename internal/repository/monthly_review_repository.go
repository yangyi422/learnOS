package repository

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"learnos/internal/model"
	"time"
)

var ErrReviewConflict = errors.New("monthly review changed")

type MonthlyReviewRepository struct{ db *gorm.DB }

func NewMonthlyReviewRepository(db *gorm.DB) *MonthlyReviewRepository {
	return &MonthlyReviewRepository{db.Session(&gorm.Session{Logger: logger.Discard})}
}
func (r *MonthlyReviewRepository) Read(ctx context.Context, fn func(*gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
func (r *MonthlyReviewRepository) Get(ctx context.Context, user uint, month string) (model.MonthlyReview, error) {
	var row model.MonthlyReview
	err := r.db.WithContext(ctx).Where("user_id = ? AND month = ?", user, month).First(&row).Error
	return row, err
}
func (r *MonthlyReviewRepository) Facts(ctx context.Context, user, id uint) (model.MonthlyReviewFacts, error) {
	var row model.MonthlyReviewFacts
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", user, id).First(&row).Error
	return row, err
}
func (r *MonthlyReviewRepository) Draft(ctx context.Context, row model.MonthlyReviewFacts) (model.MonthlyReviewFacts, error) {
	err := r.db.WithContext(ctx).Create(&row).Error
	return row, err
}
func (r *MonthlyReviewRepository) List(ctx context.Context, user uint) ([]model.MonthlyReview, error) {
	rows := []model.MonthlyReview{}
	err := r.db.WithContext(ctx).Where("user_id = ?", user).Order("month DESC").Find(&rows).Error
	return rows, err
}
func (r *MonthlyReviewRepository) History(ctx context.Context, user uint, month string) ([]model.MonthlyReviewFacts, error) {
	rows := []model.MonthlyReviewFacts{}
	err := r.db.WithContext(ctx).Select("id,user_id,month,base_version,generated_at,confirmed_at").Where("user_id = ? AND month = ? AND confirmed_at IS NOT NULL", user, month).Order("confirmed_at DESC, id DESC").Find(&rows).Error
	return rows, err
}
func (r *MonthlyReviewRepository) Save(ctx context.Context, row model.MonthlyReview) (model.MonthlyReview, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var facts model.MonthlyReviewFacts
		if err := tx.Where("id = ? AND user_id = ? AND month = ?", row.FactsID, row.UserID, row.Month).First(&facts).Error; err != nil {
			return err
		}
		var old model.MonthlyReview
		err := tx.Where("user_id = ? AND month = ?", row.UserID, row.Month).First(&old).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		exists := err == nil
		if exists && old.FactsID == row.FactsID && old.Memorable == row.Memorable && old.Thoughts == row.Thoughts && old.NextFocus == row.NextFocus && old.ExternalURL == row.ExternalURL && (row.Version == old.Version || row.Version+1 == old.Version) {
			row = old
			return nil
		}
		if (!exists && row.Version != 0) || (exists && old.Version != row.Version) {
			return ErrReviewConflict
		}
		if (!exists || old.FactsID != row.FactsID) && (facts.BaseVersion != row.Version || facts.ConfirmedAt != nil || time.Since(facts.GeneratedAt) > 24*time.Hour) {
			return ErrReviewConflict
		}
		if exists && old.FactsID != row.FactsID {
			personal, _ := json.Marshal(old)
			if err := tx.Model(&model.MonthlyReviewFacts{}).Where("id = ? AND user_id = ?", old.FactsID, row.UserID).Update("archived_personal_json", string(personal)).Error; err != nil {
				return err
			}
		}
		if !exists {
			row.Version = 1
			q := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "month"}}, DoNothing: true}).Create(&row)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected != 1 {
				return ErrReviewConflict
			}
		} else {
			row.ID = old.ID
			row.CreatedAt = old.CreatedAt
			q := tx.Model(&model.MonthlyReview{}).Where("id = ? AND user_id = ? AND version = ?", old.ID, row.UserID, row.Version).Updates(map[string]interface{}{"facts_id": row.FactsID, "memorable": row.Memorable, "thoughts": row.Thoughts, "next_focus": row.NextFocus, "external_url": row.ExternalURL, "version": row.Version + 1, "updated_at": time.Now().UTC()})
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected != 1 {
				return ErrReviewConflict
			}
			if err := tx.First(&row, old.ID).Error; err != nil {
				return err
			}
		}
		if facts.ConfirmedAt == nil {
			return tx.Model(&facts).Update("confirmed_at", time.Now().UTC()).Error
		}
		return nil
	})
	return row, err
}
