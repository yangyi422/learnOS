package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type ExplorationRepository struct {
	db *gorm.DB
}

func NewExplorationRepository(db *gorm.DB) *ExplorationRepository {
	return &ExplorationRepository{db: db}
}

func (r *ExplorationRepository) FindDirectionByID(ctx context.Context, id uint) (*model.ExplorationDirection, error) {
	var direction model.ExplorationDirection
	if err := r.db.WithContext(ctx).First(&direction, id).Error; err != nil {
		return nil, fmt.Errorf("find exploration direction: %w", err)
	}
	return &direction, nil
}

func (r *ExplorationRepository) FindDirectionByIdentity(ctx context.Context, courseID uint, sourceLessonID *uint, targetCourseID, targetLessonID uint, directionType string) (*model.ExplorationDirection, error) {
	var direction model.ExplorationDirection
	query := r.db.WithContext(ctx).
		Where("course_id = ? AND target_course_id = ? AND target_lesson_id = ? AND direction_type = ?", courseID, targetCourseID, targetLessonID, directionType)
	if sourceLessonID == nil {
		query = query.Where("source_lesson_id IS NULL")
	} else {
		query = query.Where("source_lesson_id = ?", *sourceLessonID)
	}
	if err := query.First(&direction).Error; err != nil {
		return nil, fmt.Errorf("find exploration direction by identity: %w", err)
	}
	return &direction, nil
}

func (r *ExplorationRepository) CreateDirection(ctx context.Context, direction *model.ExplorationDirection) error {
	if err := r.db.WithContext(ctx).Create(direction).Error; err != nil {
		return fmt.Errorf("create exploration direction: %w", err)
	}
	return nil
}

func (r *ExplorationRepository) UpdateDirection(ctx context.Context, direction *model.ExplorationDirection) error {
	if err := r.db.WithContext(ctx).Model(&model.ExplorationDirection{}).Where("id = ?", direction.ID).Updates(map[string]interface{}{
		"title":               direction.Title,
		"summary":             direction.Summary,
		"why_worth_exploring": direction.WhyWorthExploring,
		"score":               direction.Score,
		"reason_code":         direction.ReasonCode,
		"reason_data_json":    direction.ReasonDataJSON,
		"status":              direction.Status,
		"generated_by":        direction.GeneratedBy,
		"provider":            direction.Provider,
		"model":               direction.Model,
		"prompt_version":      direction.PromptVersion,
	}).Error; err != nil {
		return fmt.Errorf("update exploration direction: %w", err)
	}
	return nil
}

func (r *ExplorationRepository) UpdateDirectionStatus(ctx context.Context, id uint, status string) error {
	if err := r.db.WithContext(ctx).Model(&model.ExplorationDirection{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("update exploration direction status: %w", err)
	}
	return nil
}

func (r *ExplorationRepository) ListDirections(ctx context.Context, courseID uint, statuses []string, limit int) ([]model.ExplorationDirection, error) {
	var directions []model.ExplorationDirection
	query := r.db.WithContext(ctx).Where("course_id = ?", courseID).
		Order("score DESC, updated_at DESC, id DESC")
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&directions).Error; err != nil {
		return nil, fmt.Errorf("list exploration directions: %w", err)
	}
	return directions, nil
}

func (r *ExplorationRepository) ListDirectionHistory(ctx context.Context, courseID uint, limit int) ([]model.ExplorationDirection, error) {
	return r.ListDirections(ctx, courseID, []string{
		model.ExplorationDirectionSaved,
		model.ExplorationDirectionDismissed,
		model.ExplorationDirectionOpened,
	}, limit)
}

func (r *ExplorationRepository) FindOpenQuestionByDirection(ctx context.Context, directionID uint) (*model.ExplorationQuestion, error) {
	var question model.ExplorationQuestion
	if err := r.db.WithContext(ctx).Where("source_direction_id = ? AND status IN ?", directionID, []string{
		model.ExplorationQuestionOpen,
		model.ExplorationQuestionExploring,
		model.ExplorationQuestionLater,
		model.ExplorationQuestionResolved,
	}).Order("id DESC").First(&question).Error; err != nil {
		return nil, fmt.Errorf("find open exploration question: %w", err)
	}
	return &question, nil
}

func (r *ExplorationRepository) CreateQuestionAndSaveDirection(ctx context.Context, question *model.ExplorationQuestion, directionID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(question).Error; err != nil {
			return fmt.Errorf("create exploration question: %w", err)
		}
		if err := tx.Model(&model.ExplorationDirection{}).Where("id = ?", directionID).Updates(map[string]interface{}{
			"status": model.ExplorationDirectionSaved, "updated_at": time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("mark exploration direction saved: %w", err)
		}
		return nil
	})
}

func (r *ExplorationRepository) UndoQuestionSave(ctx context.Context, directionID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.ExplorationQuestion{}).
			Where("source_direction_id = ? AND status <> ?", directionID, model.ExplorationQuestionArchived).
			Updates(map[string]interface{}{"status": model.ExplorationQuestionArchived, "updated_at": time.Now()}).Error; err != nil {
			return fmt.Errorf("archive exploration question: %w", err)
		}
		if err := tx.Model(&model.ExplorationDirection{}).Where("id = ?", directionID).Updates(map[string]interface{}{
			"status": model.ExplorationDirectionActive, "updated_at": time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("restore exploration direction: %w", err)
		}
		return nil
	})
}

func (r *ExplorationRepository) CreateQuestion(ctx context.Context, question *model.ExplorationQuestion) error {
	if err := r.db.WithContext(ctx).Create(question).Error; err != nil {
		return fmt.Errorf("create exploration question: %w", err)
	}
	return nil
}

func (r *ExplorationRepository) FindQuestionByID(ctx context.Context, id uint) (*model.ExplorationQuestion, error) {
	var question model.ExplorationQuestion
	if err := r.db.WithContext(ctx).First(&question, id).Error; err != nil {
		return nil, fmt.Errorf("find exploration question: %w", err)
	}
	return &question, nil
}

func (r *ExplorationRepository) UpdateQuestionStatus(ctx context.Context, id uint, status string) error {
	if err := r.db.WithContext(ctx).Model(&model.ExplorationQuestion{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("update exploration question status: %w", err)
	}
	return nil
}

func (r *ExplorationRepository) UpdateQuestionPriority(ctx context.Context, id uint, priority string) error {
	if err := r.db.WithContext(ctx).Model(&model.ExplorationQuestion{}).Where("id = ?", id).Updates(map[string]interface{}{
		"priority": priority, "updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("update exploration question priority: %w", err)
	}
	return nil
}

func (r *ExplorationRepository) ListQuestions(ctx context.Context, courseID uint, status string, limit int, targetCourseID uint, priority string) ([]model.ExplorationQuestion, error) {
	var questions []model.ExplorationQuestion
	query := r.db.WithContext(ctx).Where("course_id = ?", courseID).Order("updated_at DESC, id DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if targetCourseID != 0 {
		query = query.Where("target_course_id = ?", targetCourseID)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("list exploration questions: %w", err)
	}
	return questions, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
