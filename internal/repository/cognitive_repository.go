package repository

import (
	"context"
	"errors"
	"fmt"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type CognitiveRepository struct {
	db *gorm.DB
}

func NewCognitiveRepository(db *gorm.DB) *CognitiveRepository {
	return &CognitiveRepository{db: db}
}

func (r *CognitiveRepository) GetByLessonID(ctx context.Context, courseID, lessonID uint) (*model.CognitiveState, error) {
	var state model.CognitiveState
	if err := r.db.WithContext(ctx).Where("course_id = ? AND lesson_id = ?", courseID, lessonID).First(&state).Error; err != nil {
		return nil, fmt.Errorf("find cognitive state: %w", err)
	}
	return &state, nil
}

func (r *CognitiveRepository) ListByCourse(ctx context.Context, courseID uint) ([]model.CognitiveState, error) {
	var states []model.CognitiveState
	if err := r.db.WithContext(ctx).Where("course_id = ?", courseID).Order("lesson_id ASC").Find(&states).Error; err != nil {
		return nil, fmt.Errorf("list cognitive states: %w", err)
	}
	return states, nil
}

func (r *CognitiveRepository) Upsert(ctx context.Context, state *model.CognitiveState) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return upsertCognitiveState(tx, state)
	})
}

func (r *CognitiveRepository) ListEvidenceByLesson(ctx context.Context, courseID, lessonID uint) ([]model.CognitiveEvidence, error) {
	var evidence []model.CognitiveEvidence
	if err := r.db.WithContext(ctx).
		Where("course_id = ? AND lesson_id = ?", courseID, lessonID).
		Order("created_at DESC, id DESC").Find(&evidence).Error; err != nil {
		return nil, fmt.Errorf("list cognitive evidence by lesson: %w", err)
	}
	return evidence, nil
}

func (r *CognitiveRepository) ListEvidenceByLearningTurn(ctx context.Context, turnID uint) ([]model.CognitiveEvidence, error) {
	var evidence []model.CognitiveEvidence
	if err := r.db.WithContext(ctx).
		Where("learning_turn_id = ?", turnID).
		Order("evidence_index ASC").Find(&evidence).Error; err != nil {
		return nil, fmt.Errorf("list cognitive evidence by turn: %w", err)
	}
	return evidence, nil
}

func (r *CognitiveRepository) CreateMany(ctx context.Context, evidence []model.CognitiveEvidence) error {
	if len(evidence) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&evidence).Error; err != nil {
		return fmt.Errorf("create cognitive evidence: %w", err)
	}
	return nil
}

func (r *CognitiveRepository) ListEventsByLesson(ctx context.Context, courseID, lessonID uint, limit int) ([]model.CognitiveStateEvent, error) {
	var events []model.CognitiveStateEvent
	query := r.db.WithContext(ctx).
		Where("course_id = ? AND lesson_id = ?", courseID, lessonID).
		Order("created_at DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&events).Error; err != nil {
		return nil, fmt.Errorf("list cognitive state events: %w", err)
	}
	return events, nil
}

func (r *CognitiveRepository) CreateEvent(ctx context.Context, event *model.CognitiveStateEvent) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create cognitive state event: %w", err)
	}
	return nil
}

func upsertCognitiveState(tx *gorm.DB, state *model.CognitiveState) error {
	var existing model.CognitiveState
	err := tx.Where("lesson_id = ?", state.LessonID).First(&existing).Error
	switch {
	case err == nil:
		state.ID = existing.ID
		state.CreatedAt = existing.CreatedAt
		if err := tx.Model(&existing).Updates(map[string]interface{}{
			"course_id":             state.CourseID,
			"current_level":         state.CurrentLevel,
			"status":                state.Status,
			"understanding_summary": state.UnderstandingSummary,
			"last_learning_turn_id": state.LastLearningTurnID,
			"last_evaluated_at":     state.LastEvaluatedAt,
		}).Error; err != nil {
			return fmt.Errorf("update cognitive state: %w", err)
		}
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := tx.Create(state).Error; err != nil {
			return fmt.Errorf("create cognitive state: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("find cognitive state for upsert: %w", err)
	}
}
