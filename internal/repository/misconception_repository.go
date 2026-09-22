package repository

import (
	"context"
	"fmt"
	"strings"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type MisconceptionRepository struct{ db *gorm.DB }

func NewMisconceptionRepository(db *gorm.DB) *MisconceptionRepository {
	return &MisconceptionRepository{db: db}
}

func (r *MisconceptionRepository) FindByID(ctx context.Context, courseID, lessonID, id uint) (*model.Misconception, error) {
	var misconception model.Misconception
	if err := r.db.WithContext(ctx).Where("course_id = ? AND lesson_id = ? AND id = ?", courseID, lessonID, id).First(&misconception).Error; err != nil {
		return nil, fmt.Errorf("find misconception: %w", err)
	}
	return &misconception, nil
}

func (r *MisconceptionRepository) FindByCourseID(ctx context.Context, courseID, id uint) (*model.Misconception, error) {
	var misconception model.Misconception
	if err := r.db.WithContext(ctx).Where("course_id = ? AND id = ?", courseID, id).First(&misconception).Error; err != nil {
		return nil, fmt.Errorf("find misconception in course: %w", err)
	}
	return &misconception, nil
}

func (r *MisconceptionRepository) Review(ctx context.Context, misconception *model.Misconception, reviewStatus, note, eventType string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Misconception{}).Where("id = ? AND course_id = ?", misconception.ID, misconception.CourseID).Updates(map[string]interface{}{
			"review_status": reviewStatus,
			"user_note":     note,
		}).Error; err != nil {
			return fmt.Errorf("review misconception: %w", err)
		}
		event := model.MisconceptionEvent{
			CourseID: misconception.CourseID, LessonID: misconception.LessonID, MisconceptionID: misconception.ID,
			EventType: eventType, Notes: note,
		}
		if err := tx.Create(&event).Error; err != nil {
			return fmt.Errorf("create misconception review event: %w", err)
		}
		return nil
	})
}

func (r *MisconceptionRepository) FindByIdentity(ctx context.Context, courseID, lessonID uint, original, correct string) (*model.Misconception, error) {
	var misconception model.Misconception
	err := r.db.WithContext(ctx).Where("course_id = ? AND lesson_id = ? AND original_understanding = ? AND correct_understanding = ?", courseID, lessonID, strings.TrimSpace(original), strings.TrimSpace(correct)).First(&misconception).Error
	if err != nil {
		return nil, fmt.Errorf("find misconception by identity: %w", err)
	}
	return &misconception, nil
}

func (r *MisconceptionRepository) ListByCourse(ctx context.Context, courseID uint) ([]model.Misconception, error) {
	var misconceptions []model.Misconception
	if err := r.db.WithContext(ctx).Where("course_id = ?", courseID).Order("created_at ASC, id ASC").Find(&misconceptions).Error; err != nil {
		return nil, fmt.Errorf("list misconceptions: %w", err)
	}
	return misconceptions, nil
}

func (r *MisconceptionRepository) ListByLesson(ctx context.Context, courseID, lessonID uint) ([]model.Misconception, error) {
	var misconceptions []model.Misconception
	if err := r.db.WithContext(ctx).Where("course_id = ? AND lesson_id = ?", courseID, lessonID).Order("created_at ASC, id ASC").Find(&misconceptions).Error; err != nil {
		return nil, fmt.Errorf("list lesson misconceptions: %w", err)
	}
	return misconceptions, nil
}

func (r *MisconceptionRepository) ListEvents(ctx context.Context, courseID uint) ([]model.MisconceptionEvent, error) {
	var events []model.MisconceptionEvent
	if err := r.db.WithContext(ctx).Where("course_id = ?", courseID).Order("created_at ASC, id ASC").Find(&events).Error; err != nil {
		return nil, fmt.Errorf("list misconception events: %w", err)
	}
	return events, nil
}

func (r *MisconceptionRepository) ListEventsByMisconception(ctx context.Context, courseID, misconceptionID uint) ([]model.MisconceptionEvent, error) {
	var events []model.MisconceptionEvent
	if err := r.db.WithContext(ctx).Where("course_id = ? AND misconception_id = ?", courseID, misconceptionID).Order("created_at DESC, id DESC").Find(&events).Error; err != nil {
		return nil, fmt.Errorf("list misconception history: %w", err)
	}
	return events, nil
}

func (r *MisconceptionRepository) ListPatternLinks(ctx context.Context, misconceptionIDs []uint) ([]model.MisconceptionPatternLink, error) {
	var links []model.MisconceptionPatternLink
	if len(misconceptionIDs) == 0 {
		return []model.MisconceptionPatternLink{}, nil
	}
	if err := r.db.WithContext(ctx).Where("misconception_id IN ?", misconceptionIDs).Order("pattern_key ASC, id ASC").Find(&links).Error; err != nil {
		return nil, fmt.Errorf("list misconception pattern links: %w", err)
	}
	return links, nil
}
