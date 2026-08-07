package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type ChallengeRepository struct {
	db *gorm.DB
}

func NewChallengeRepository(db *gorm.DB) *ChallengeRepository { return &ChallengeRepository{db: db} }

func (r *ChallengeRepository) Create(ctx context.Context, challenge *model.AssessmentChallenge) error {
	if err := r.db.WithContext(ctx).Create(challenge).Error; err != nil {
		return fmt.Errorf("create assessment challenge: %w", err)
	}
	return nil
}

func (r *ChallengeRepository) CreateWithRun(ctx context.Context, challenge *model.AssessmentChallenge, run *model.AIEvaluationRun) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(challenge).Error; err != nil {
			return fmt.Errorf("create assessment challenge: %w", err)
		}
		if run != nil {
			run.CourseID, run.LessonID = challenge.CourseID, challenge.LessonID
			if err := tx.Create(run).Error; err != nil {
				return fmt.Errorf("create challenge generation audit: %w", err)
			}
		}
		return nil
	})
}

func (r *ChallengeRepository) FindByID(ctx context.Context, courseID, challengeID uint) (*model.AssessmentChallenge, error) {
	var challenge model.AssessmentChallenge
	if err := r.db.WithContext(ctx).Where("course_id = ? AND id = ?", courseID, challengeID).First(&challenge).Error; err != nil {
		return nil, fmt.Errorf("find assessment challenge: %w", err)
	}
	return &challenge, nil
}

func (r *ChallengeRepository) ListByLesson(ctx context.Context, courseID, lessonID uint) ([]model.AssessmentChallenge, error) {
	var challenges []model.AssessmentChallenge
	if err := r.db.WithContext(ctx).Where("course_id = ? AND lesson_id = ?", courseID, lessonID).Order("created_at DESC, id DESC").Find(&challenges).Error; err != nil {
		return nil, fmt.Errorf("list assessment challenges: %w", err)
	}
	return challenges, nil
}

func (r *ChallengeRepository) FindAttempt(ctx context.Context, challengeID uint) (*model.ChallengeAttempt, error) {
	var attempt model.ChallengeAttempt
	if err := r.db.WithContext(ctx).Where("challenge_id = ?", challengeID).First(&attempt).Error; err != nil {
		return nil, fmt.Errorf("find challenge attempt: %w", err)
	}
	return &attempt, nil
}

func (r *ChallengeRepository) SaveChallengeResult(ctx context.Context, turn *model.LearningTurn, attempt *model.ChallengeAttempt, challenge *model.AssessmentChallenge, state *model.CognitiveState, evidence []model.CognitiveEvidence, stateEvent *model.CognitiveStateEvent, misconceptionUpdates []model.Misconception, misconceptionEvents []model.MisconceptionEvent, patternLinks []model.MisconceptionPatternLink, run *model.AIEvaluationRun) error {
	return r.saveChallengeResult(ctx, turn, attempt, challenge, state, evidence, stateEvent, misconceptionUpdates, misconceptionEvents, patternLinks, nil, run)
}

func (r *ChallengeRepository) SaveChallengeResultWithObservations(ctx context.Context, turn *model.LearningTurn, attempt *model.ChallengeAttempt, challenge *model.AssessmentChallenge, state *model.CognitiveState, evidence []model.CognitiveEvidence, stateEvent *model.CognitiveStateEvent, misconceptionUpdates []model.Misconception, misconceptionEvents []model.MisconceptionEvent, patternLinks []model.MisconceptionPatternLink, observations []model.MisconceptionObservation, run *model.AIEvaluationRun) error {
	return r.saveChallengeResult(ctx, turn, attempt, challenge, state, evidence, stateEvent, misconceptionUpdates, misconceptionEvents, patternLinks, observations, run)
}

func (r *ChallengeRepository) saveChallengeResult(ctx context.Context, turn *model.LearningTurn, attempt *model.ChallengeAttempt, challenge *model.AssessmentChallenge, state *model.CognitiveState, evidence []model.CognitiveEvidence, stateEvent *model.CognitiveStateEvent, misconceptionUpdates []model.Misconception, misconceptionEvents []model.MisconceptionEvent, patternLinks []model.MisconceptionPatternLink, observations []model.MisconceptionObservation, run *model.AIEvaluationRun) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(turn).Error; err != nil {
			return fmt.Errorf("create challenge learning turn: %w", err)
		}
		attempt.CourseID = turn.CourseID
		attempt.LessonID = turn.LessonID
		attempt.LearningTurnID = turn.ID
		if err := tx.Create(attempt).Error; err != nil {
			return fmt.Errorf("create challenge attempt: %w", err)
		}
		updated := tx.Model(&model.AssessmentChallenge{}).Where("id = ? AND course_id = ? AND status = ?", challenge.ID, challenge.CourseID, model.ChallengeStatusPending).Update("status", model.ChallengeStatusAnswered)
		if updated.Error != nil {
			return fmt.Errorf("mark challenge answered: %w", updated.Error)
		}
		if updated.RowsAffected != 1 {
			return fmt.Errorf("challenge is no longer pending")
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
					return fmt.Errorf("create challenge cognitive evidence: %w", err)
				}
			}
			if stateEvent != nil {
				stateEvent.CourseID = turn.CourseID
				stateEvent.LessonID = turn.LessonID
				stateEvent.LearningTurnID = turn.ID
				if err := tx.Create(stateEvent).Error; err != nil {
					return fmt.Errorf("create challenge cognitive state event: %w", err)
				}
			}
		}
		for index := range misconceptionUpdates {
			misconception := &misconceptionUpdates[index]
			if misconception.Status == model.MisconceptionStatusResolved {
				misconception.ResolvedByLearningTurnID = &turn.ID
			}
			if err := tx.Model(&model.Misconception{}).Where("id = ? AND course_id = ? AND lesson_id = ?", misconception.ID, turn.CourseID, turn.LessonID).Updates(map[string]interface{}{
				"status": misconception.Status, "resolved_at": misconception.ResolvedAt,
				"resolved_by_learning_turn_id": misconception.ResolvedByLearningTurnID,
				"resolved_by_challenge_id":     misconception.ResolvedByChallengeID,
				"boundary_notes":               misconception.BoundaryNotes,
			}).Error; err != nil {
				return fmt.Errorf("update misconception: %w", err)
			}
		}
		for index := range misconceptionEvents {
			event := &misconceptionEvents[index]
			event.CourseID = turn.CourseID
			event.LessonID = turn.LessonID
			event.LearningTurnID = &turn.ID
			event.ChallengeID = &challenge.ID
			if err := tx.Create(event).Error; err != nil {
				return fmt.Errorf("create misconception event: %w", err)
			}
		}
		for index := range patternLinks {
			link := &patternLinks[index]
			var existing model.MisconceptionPatternLink
			err := tx.Where("misconception_id = ? AND pattern_key = ?", link.MisconceptionID, link.PatternKey).First(&existing).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				if err := tx.Create(link).Error; err != nil {
					return fmt.Errorf("create misconception pattern link: %w", err)
				}
			case err == nil:
				if err := tx.Model(&existing).Updates(map[string]interface{}{"explanation": link.Explanation, "source": link.Source, "prompt_version": link.PromptVersion}).Error; err != nil {
					return fmt.Errorf("update misconception pattern link: %w", err)
				}
			default:
				return fmt.Errorf("find misconception pattern link: %w", err)
			}
		}
		if err := persistChallengeMisconceptionObservations(tx, turn, challenge, observations); err != nil {
			return err
		}
		if run != nil {
			run.LearningTurnID = &turn.ID
			if err := tx.Create(run).Error; err != nil {
				return fmt.Errorf("create challenge AI evaluation run: %w", err)
			}
		}
		return nil
	})
}

func persistChallengeMisconceptionObservations(tx *gorm.DB, turn *model.LearningTurn, challenge *model.AssessmentChallenge, observations []model.MisconceptionObservation) error {
	if len(observations) == 0 {
		return nil
	}
	turnID := turn.ID
	challengeID := challenge.ID
	for _, observation := range observations {
		original := strings.TrimSpace(observation.OriginalUnderstanding)
		correct := strings.TrimSpace(observation.CorrectUnderstanding)
		var existing model.Misconception
		err := tx.Where("course_id = ? AND lesson_id = ? AND original_understanding = ? AND correct_understanding = ?", turn.CourseID, turn.LessonID, original, correct).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			existing = model.Misconception{CourseID: turn.CourseID, LessonID: turn.LessonID, OriginalUnderstanding: original, CorrectUnderstanding: correct, BoundaryNotes: observation.BoundaryNotes, Status: model.MisconceptionStatusActive}
			if err := tx.Create(&existing).Error; err != nil {
				return fmt.Errorf("create challenge misconception: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("find challenge misconception: %w", err)
		} else {
			if err := tx.Model(&existing).Updates(map[string]interface{}{"boundary_notes": observation.BoundaryNotes, "status": model.MisconceptionStatusActive, "resolved_at": nil, "resolved_by_learning_turn_id": nil, "resolved_by_challenge_id": nil}).Error; err != nil {
				return fmt.Errorf("update challenge misconception: %w", err)
			}
		}
		eventType := model.MisconceptionEventObserved
		notes := "挑战评价再次发现该误区"
		if existing.ID != 0 && existing.Status == model.MisconceptionStatusResolved {
			eventType = model.MisconceptionEventReopened
			notes = "已解决误区在挑战评价中再次出现"
		}
		event := model.MisconceptionEvent{CourseID: turn.CourseID, LessonID: turn.LessonID, MisconceptionID: existing.ID, LearningTurnID: &turnID, ChallengeID: &challengeID, EventType: eventType, Notes: notes}
		if err := tx.Create(&event).Error; err != nil {
			return fmt.Errorf("create challenge misconception event: %w", err)
		}
		for _, pattern := range observation.Patterns {
			link := model.MisconceptionPatternLink{MisconceptionID: existing.ID, PatternKey: pattern.PatternKey, Explanation: pattern.Explanation, Source: "ai", PromptVersion: turn.PromptVersion}
			var current model.MisconceptionPatternLink
			linkErr := tx.Where("misconception_id = ? AND pattern_key = ?", link.MisconceptionID, link.PatternKey).First(&current).Error
			if errors.Is(linkErr, gorm.ErrRecordNotFound) {
				if err := tx.Create(&link).Error; err != nil {
					return fmt.Errorf("create challenge pattern link: %w", err)
				}
			} else if linkErr == nil {
				if err := tx.Model(&current).Updates(map[string]interface{}{"explanation": link.Explanation, "source": link.Source, "prompt_version": link.PromptVersion}).Error; err != nil {
					return fmt.Errorf("update challenge pattern link: %w", err)
				}
			} else {
				return fmt.Errorf("find challenge pattern link: %w", linkErr)
			}
		}
	}
	return nil
}
