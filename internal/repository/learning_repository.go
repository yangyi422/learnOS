package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type LearningRepository struct {
	db *gorm.DB
}

func NewLearningRepository(db *gorm.DB) *LearningRepository {
	return &LearningRepository{db: db}
}

func (r *LearningRepository) FindUnitByID(ctx context.Context, id uint) (*model.CourseUnit, error) {
	var unit model.CourseUnit
	if err := r.db.WithContext(ctx).First(&unit, id).Error; err != nil {
		return nil, fmt.Errorf("find course unit by id: %w", err)
	}
	return &unit, nil
}

func (r *LearningRepository) FindLessonByID(ctx context.Context, id uint) (*model.Lesson, error) {
	var lesson model.Lesson
	if err := r.db.WithContext(ctx).First(&lesson, id).Error; err != nil {
		return nil, fmt.Errorf("find lesson by id: %w", err)
	}
	return &lesson, nil
}

func (r *LearningRepository) ListLearningTurns(ctx context.Context, courseID uint, limit int) ([]model.LearningTurn, error) {
	var turns []model.LearningTurn
	if err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&turns).Error; err != nil {
		return nil, fmt.Errorf("list learning turns: %w", err)
	}
	return turns, nil
}

func (r *LearningRepository) FindMasteryRecord(ctx context.Context, lessonID uint) (*model.MasteryRecord, error) {
	var mastery model.MasteryRecord
	if err := r.db.WithContext(ctx).Where("lesson_id = ?", lessonID).First(&mastery).Error; err != nil {
		return nil, fmt.Errorf("find mastery record: %w", err)
	}
	return &mastery, nil
}

func (r *LearningRepository) SaveLearningTurnAndMastery(ctx context.Context, turn *model.LearningTurn, mastery *model.MasteryRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(turn).Error; err != nil {
			return fmt.Errorf("create learning turn: %w", err)
		}

		var existing model.MasteryRecord
		err := tx.Where("lesson_id = ?", mastery.LessonID).First(&existing).Error
		switch {
		case err == nil:
			mastery.ID = existing.ID
			mastery.CreatedAt = existing.CreatedAt
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"course_id":       mastery.CourseID,
				"mastery_score":   mastery.MasteryScore,
				"answer_count":    mastery.AnswerCount,
				"incorrect_count": mastery.IncorrectCount,
				"needs_review":    mastery.NeedsReview,
				"next_review_at":  mastery.NextReviewAt,
			}).Error; err != nil {
				return fmt.Errorf("update mastery record: %w", err)
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(mastery).Error; err != nil {
				return fmt.Errorf("create mastery record: %w", err)
			}
		default:
			return fmt.Errorf("find mastery record: %w", err)
		}
		return nil
	})
}

func (r *LearningRepository) SaveEvaluation(ctx context.Context, turn *model.LearningTurn, mastery *model.MasteryRecord, misconceptions []model.Misconception, run *model.AIEvaluationRun) error {
	return r.saveEvaluation(ctx, turn, mastery, misconceptions, run, nil, nil, nil)
}

func (r *LearningRepository) SaveEvaluationWithCognitiveState(ctx context.Context, turn *model.LearningTurn, mastery *model.MasteryRecord, misconceptions []model.Misconception, run *model.AIEvaluationRun, state *model.CognitiveState, evidence []model.CognitiveEvidence, event *model.CognitiveStateEvent) error {
	return r.saveEvaluation(ctx, turn, mastery, misconceptions, run, state, evidence, event)
}

func (r *LearningRepository) SaveEvaluationWithPhase6(ctx context.Context, turn *model.LearningTurn, mastery *model.MasteryRecord, run *model.AIEvaluationRun, state *model.CognitiveState, evidence []model.CognitiveEvidence, event *model.CognitiveStateEvent, observations []model.MisconceptionObservation) error {
	return r.saveEvaluationWithObservations(ctx, turn, mastery, nil, run, state, evidence, event, observations)
}

func (r *LearningRepository) saveEvaluation(ctx context.Context, turn *model.LearningTurn, mastery *model.MasteryRecord, misconceptions []model.Misconception, run *model.AIEvaluationRun, state *model.CognitiveState, evidence []model.CognitiveEvidence, event *model.CognitiveStateEvent) error {
	return r.saveEvaluationWithObservations(ctx, turn, mastery, misconceptions, run, state, evidence, event, nil)
}

func (r *LearningRepository) saveEvaluationWithObservations(ctx context.Context, turn *model.LearningTurn, mastery *model.MasteryRecord, misconceptions []model.Misconception, run *model.AIEvaluationRun, state *model.CognitiveState, evidence []model.CognitiveEvidence, event *model.CognitiveStateEvent, observations []model.MisconceptionObservation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(turn).Error; err != nil {
			return fmt.Errorf("create learning turn: %w", err)
		}

		var existingMastery model.MasteryRecord
		err := tx.Where("lesson_id = ?", mastery.LessonID).First(&existingMastery).Error
		switch {
		case err == nil:
			if err := tx.Model(&existingMastery).Updates(map[string]interface{}{
				"course_id":       mastery.CourseID,
				"mastery_score":   mastery.MasteryScore,
				"answer_count":    mastery.AnswerCount,
				"incorrect_count": mastery.IncorrectCount,
				"needs_review":    mastery.NeedsReview,
				"next_review_at":  mastery.NextReviewAt,
			}).Error; err != nil {
				return fmt.Errorf("update mastery record: %w", err)
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(mastery).Error; err != nil {
				return fmt.Errorf("create mastery record: %w", err)
			}
		default:
			return fmt.Errorf("find mastery record: %w", err)
		}

		for _, misconception := range misconceptions {
			var existingMisconception model.Misconception
			err := tx.Where(
				"course_id = ? AND lesson_id = ? AND original_understanding = ? AND correct_understanding = ? AND status = ?",
				misconception.CourseID,
				misconception.LessonID,
				misconception.OriginalUnderstanding,
				misconception.CorrectUnderstanding,
				"active",
			).First(&existingMisconception).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				if err := tx.Create(&misconception).Error; err != nil {
					return fmt.Errorf("create misconception: %w", err)
				}
			case err == nil:
				if err := tx.Model(&existingMisconception).Update("boundary_notes", misconception.BoundaryNotes).Error; err != nil {
					return fmt.Errorf("update misconception: %w", err)
				}
			default:
				return fmt.Errorf("find misconception: %w", err)
			}
		}
		if observations != nil {
			turnID := turn.ID
			for _, observation := range observations {
				original := strings.TrimSpace(observation.OriginalUnderstanding)
				correct := strings.TrimSpace(observation.CorrectUnderstanding)
				var existing model.Misconception
				err := tx.Where("course_id = ? AND lesson_id = ? AND original_understanding = ? AND correct_understanding = ?", turn.CourseID, turn.LessonID, original, correct).First(&existing).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					existing = model.Misconception{CourseID: turn.CourseID, LessonID: turn.LessonID, OriginalUnderstanding: original, CorrectUnderstanding: correct, BoundaryNotes: observation.BoundaryNotes, Status: model.MisconceptionStatusActive}
					if err := tx.Create(&existing).Error; err != nil {
						return fmt.Errorf("create misconception observation: %w", err)
					}
					misEvent := model.MisconceptionEvent{CourseID: turn.CourseID, LessonID: turn.LessonID, MisconceptionID: existing.ID, LearningTurnID: &turnID, EventType: model.MisconceptionEventObserved, Notes: "普通 Lesson 评价发现新误区"}
					if err := tx.Create(&misEvent).Error; err != nil {
						return fmt.Errorf("create misconception observed event: %w", err)
					}
				} else if err != nil {
					return fmt.Errorf("find misconception observation: %w", err)
				} else {
					if err := tx.Model(&existing).Update("boundary_notes", observation.BoundaryNotes).Error; err != nil {
						return fmt.Errorf("update misconception observation: %w", err)
					}
					if existing.Status == model.MisconceptionStatusResolved {
						if err := tx.Model(&existing).Updates(map[string]interface{}{"status": model.MisconceptionStatusActive, "resolved_at": nil, "resolved_by_learning_turn_id": nil, "resolved_by_challenge_id": nil}).Error; err != nil {
							return fmt.Errorf("reopen misconception: %w", err)
						}
						misEvent := model.MisconceptionEvent{CourseID: turn.CourseID, LessonID: turn.LessonID, MisconceptionID: existing.ID, LearningTurnID: &turnID, EventType: model.MisconceptionEventReopened, Notes: "已解决误区在普通 Lesson 评价中再次出现"}
						if err := tx.Create(&misEvent).Error; err != nil {
							return fmt.Errorf("create misconception reopened event: %w", err)
						}
					} else {
						misEvent := model.MisconceptionEvent{CourseID: turn.CourseID, LessonID: turn.LessonID, MisconceptionID: existing.ID, LearningTurnID: &turnID, EventType: model.MisconceptionEventObserved, Notes: "普通 Lesson 评价再次发现该误区"}
						if err := tx.Create(&misEvent).Error; err != nil {
							return fmt.Errorf("create misconception observed event: %w", err)
						}
					}
				}
				var current model.Misconception
				if err := tx.Where("course_id = ? AND lesson_id = ? AND original_understanding = ? AND correct_understanding = ?", turn.CourseID, turn.LessonID, original, correct).First(&current).Error; err != nil {
					return fmt.Errorf("reload misconception observation: %w", err)
				}
				for _, pattern := range observation.Patterns {
					link := model.MisconceptionPatternLink{MisconceptionID: current.ID, PatternKey: pattern.PatternKey, Explanation: pattern.Explanation, Source: "ai", PromptVersion: turn.PromptVersion}
					var existingLink model.MisconceptionPatternLink
					linkErr := tx.Where("misconception_id = ? AND pattern_key = ?", link.MisconceptionID, link.PatternKey).First(&existingLink).Error
					if errors.Is(linkErr, gorm.ErrRecordNotFound) {
						if err := tx.Create(&link).Error; err != nil {
							return fmt.Errorf("create misconception pattern link: %w", err)
						}
					} else if linkErr != nil {
						return fmt.Errorf("find misconception pattern link: %w", linkErr)
					}
				}
			}
		}

		if run != nil {
			run.LearningTurnID = &turn.ID
			if err := tx.Create(run).Error; err != nil {
				return fmt.Errorf("create AI evaluation run: %w", err)
			}
		}

		if state != nil {
			turnID := turn.ID
			state.LastLearningTurnID = &turnID
			if err := upsertCognitiveState(tx, state); err != nil {
				return err
			}
			for index := range evidence {
				evidence[index].CourseID = turn.CourseID
				evidence[index].LessonID = turn.LessonID
				evidence[index].LearningTurnID = turn.ID
				evidence[index].EvidenceIndex = index
				if err := tx.Create(&evidence[index]).Error; err != nil {
					return fmt.Errorf("create cognitive evidence: %w", err)
				}
			}
			if event != nil {
				event.CourseID = turn.CourseID
				event.LessonID = turn.LessonID
				event.LearningTurnID = turn.ID
				if err := tx.Create(event).Error; err != nil {
					return fmt.Errorf("create cognitive state event: %w", err)
				}
			}
		}
		return nil
	})
}

func (r *LearningRepository) SaveAIEvaluationRun(ctx context.Context, run *model.AIEvaluationRun) error {
	if err := r.db.WithContext(ctx).Create(run).Error; err != nil {
		return fmt.Errorf("create AI evaluation run: %w", err)
	}
	return nil
}
