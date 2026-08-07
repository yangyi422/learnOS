package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"learnos/internal/ai"
	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrCourseNotFound        = errors.New("course not found")
	ErrCurrentLessonNotFound = errors.New("current lesson not found")
	ErrInvalidAnswer         = errors.New("invalid answer")
	ErrLessonNotCurrent      = errors.New("lesson is not the current lesson")
	ErrInvalidCourseID       = errors.New("invalid course id")
)

type CurrentLesson struct {
	Course *model.Course
	Unit   *model.CourseUnit
	Lesson *model.Lesson
}

type EvaluationResult = ai.EvaluationResult

type AnswerResult struct {
	TurnID                   uint                         `json:"turn_id"`
	Result                   string                       `json:"result"`
	Feedback                 string                       `json:"feedback"`
	Explanation              string                       `json:"explanation"`
	CorrectParts             []string                     `json:"correct_parts"`
	MissingParts             []string                     `json:"missing_parts"`
	Misconceptions           []ai.EvaluationMisconception `json:"misconceptions"`
	BoundaryConditions       []string                     `json:"boundary_conditions"`
	MasteryEvidence          []string                     `json:"mastery_evidence"`
	MasteryScore             float64                      `json:"mastery_score"`
	NeedsReview              bool                         `json:"needs_review"`
	EvaluationSource         string                       `json:"evaluation_source"`
	Provider                 string                       `json:"provider"`
	Model                    string                       `json:"model"`
	PromptVersion            string                       `json:"prompt_version"`
	DemonstratedLevel        string                       `json:"demonstrated_level"`
	UserUnderstandingSummary string                       `json:"user_understanding_summary"`
	CognitiveEvidence        []ai.EvaluationEvidence      `json:"cognitive_evidence"`
	CognitiveState           *CognitiveAnswerState        `json:"cognitive_state"`
}

type CognitiveAnswerState struct {
	CurrentLevel string `json:"current_level"`
	Status       string `json:"status"`
}

type LearningTurnView struct {
	ID                       uint                         `json:"id"`
	Question                 string                       `json:"question"`
	UserAnswer               string                       `json:"user_answer"`
	Result                   string                       `json:"result"`
	Feedback                 string                       `json:"feedback"`
	Explanation              string                       `json:"explanation"`
	CorrectParts             []string                     `json:"correct_parts"`
	MissingParts             []string                     `json:"missing_parts"`
	Misconceptions           []ai.EvaluationMisconception `json:"misconceptions"`
	BoundaryConditions       []string                     `json:"boundary_conditions"`
	MasteryEvidence          []string                     `json:"mastery_evidence"`
	MasteryScore             float64                      `json:"mastery_score"`
	NeedsReview              bool                         `json:"needs_review"`
	EvaluationSource         string                       `json:"evaluation_source"`
	Provider                 string                       `json:"provider"`
	Model                    string                       `json:"model"`
	PromptVersion            string                       `json:"prompt_version"`
	DemonstratedLevel        string                       `json:"demonstrated_level"`
	UserUnderstandingSummary string                       `json:"user_understanding_summary"`
	CognitiveEvidence        []ai.EvaluationEvidence      `json:"cognitive_evidence"`
	CreatedAt                time.Time                    `json:"created_at"`
}

type CourseService struct {
	courses   *repository.CourseRepository
	learning  *repository.LearningRepository
	provider  ai.AIProvider
	cognitive *CognitiveStateService
}

func NewCourseService(courses *repository.CourseRepository, learning *repository.LearningRepository, providers ...ai.AIProvider) *CourseService {
	provider := ai.AIProvider(ai.NewMockProvider())
	if len(providers) > 0 && providers[0] != nil {
		provider = providers[0]
	}
	return &CourseService{courses: courses, learning: learning, provider: provider}
}

func (s *CourseService) SetCognitiveStateService(cognitive *CognitiveStateService) {
	s.cognitive = cognitive
}

func (s *CourseService) List(ctx context.Context) ([]model.Course, error) {
	return s.courses.List(ctx)
}

func (s *CourseService) SeedStarterCourse(ctx context.Context) error {
	return s.courses.SeedStarterCourse(ctx)
}

func (s *CourseService) GetCurrentLesson(ctx context.Context, courseID uint) (*CurrentLesson, error) {
	course, err := s.findCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course.CurrentUnitID == nil || course.CurrentLessonID == nil {
		return nil, ErrCurrentLessonNotFound
	}

	unit, err := s.learning.FindUnitByID(ctx, *course.CurrentUnitID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCurrentLessonNotFound
		}
		return nil, err
	}
	lesson, err := s.learning.FindLessonByID(ctx, *course.CurrentLessonID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCurrentLessonNotFound
		}
		return nil, err
	}
	if unit.CourseID != course.ID || lesson.CourseID != course.ID || lesson.UnitID != unit.ID {
		return nil, ErrCurrentLessonNotFound
	}

	return &CurrentLesson{Course: course, Unit: unit, Lesson: lesson}, nil
}

func (s *CourseService) SubmitAnswer(ctx context.Context, courseID, lessonID uint, answer string) (*AnswerResult, error) {
	if lessonID == 0 {
		return nil, ErrInvalidAnswer
	}
	answer = strings.TrimSpace(answer)
	if answer == "" || len([]rune(answer)) > 5000 {
		return nil, ErrInvalidAnswer
	}

	current, err := s.GetCurrentLesson(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if current.Lesson.ID != lessonID {
		return nil, ErrLessonNotCurrent
	}
	target := current.Lesson.AssessmentTargetLevel
	if strings.TrimSpace(target) == "" {
		target = model.AssessmentTargetUnderstand
	}

	evaluation, meta, err := s.provider.EvaluateLessonAnswer(ctx, ai.EvaluationRequest{
		CourseName:            current.Course.Name,
		UnitTitle:             current.Unit.Title,
		LessonTitle:           current.Lesson.Title,
		CoreQuestion:          current.Lesson.CoreQuestion,
		ExpectedUnderstanding: current.Lesson.ExpectedUnderstanding,
		AssessmentTargetLevel: target,
		UserAnswer:            answer,
	})
	if err != nil {
		s.logInvalidEvaluation(meta, err)
		s.recordFailedEvaluation(ctx, courseID, lessonID, meta, err)
		return nil, err
	}
	evaluation, err = ai.NormalizeAndValidateForTarget(evaluation, target)
	if err != nil {
		invalidErr := fmt.Errorf("%w: %v", ai.ErrInvalidResponse, err)
		s.logInvalidEvaluation(meta, invalidErr)
		s.recordFailedEvaluation(ctx, courseID, lessonID, meta, invalidErr)
		return nil, invalidErr
	}
	var cognitiveTransition *CognitiveTransition
	if s.cognitive != nil {
		cognitiveTransition, err = s.cognitive.ApplyEvaluationToCognitiveState(ctx, courseID, lessonID, 0, target, evaluation)
		if err != nil {
			invalidErr := fmt.Errorf("%w: %v", ai.ErrInvalidResponse, err)
			s.logInvalidEvaluation(meta, invalidErr)
			s.recordFailedEvaluation(ctx, courseID, lessonID, meta, invalidErr)
			return nil, invalidErr
		}
	}

	mastery, err := s.learning.FindMasteryRecord(ctx, lessonID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if mastery == nil {
		mastery = &model.MasteryRecord{CourseID: courseID, LessonID: lessonID}
	}
	mastery.CourseID = courseID
	mastery.LessonID = lessonID
	mastery.AnswerCount++
	if evaluation.Result == "incorrect" || evaluation.Result == "insufficient" {
		mastery.IncorrectCount++
	}
	mastery.MasteryScore = evaluation.MasteryScore
	mastery.NeedsReview = evaluation.NeedsReview

	correctParts := marshalJSON(evaluation.CorrectParts)
	missingParts := marshalJSON(evaluation.MissingParts)
	misconceptionsJSON := marshalJSON(evaluation.Misconceptions)
	boundaryConditions := marshalJSON(evaluation.BoundaryConditions)
	masteryEvidence := marshalJSON(evaluation.MasteryEvidence)
	cognitiveEvidenceJSON := marshalJSON(evaluation.CognitiveEvidence)
	evaluationSource := "ai"
	if meta.Provider == "mock" {
		evaluationSource = "mock"
	}
	providerName, modelName, promptVersion := normalizedMeta(meta)
	turn := &model.LearningTurn{
		CourseID:                 courseID,
		UnitID:                   current.Unit.ID,
		LessonID:                 lessonID,
		TurnKind:                 model.TurnKindLessonAnswer,
		Question:                 current.Lesson.CoreQuestion,
		UserAnswer:               answer,
		Result:                   evaluation.Result,
		Feedback:                 evaluation.Feedback,
		Explanation:              evaluation.Explanation,
		CorrectParts:             correctParts,
		MissingParts:             missingParts,
		MisconceptionsJSON:       misconceptionsJSON,
		BoundaryConditions:       boundaryConditions,
		MasteryEvidence:          masteryEvidence,
		EvaluationSource:         evaluationSource,
		Provider:                 providerName,
		Model:                    modelName,
		PromptVersion:            promptVersion,
		DemonstratedLevel:        evaluation.DemonstratedLevel,
		UserUnderstandingSummary: evaluation.UserUnderstandingSummary,
		CognitiveEvidenceJSON:    cognitiveEvidenceJSON,
		MasteryScore:             evaluation.MasteryScore,
		NeedsReview:              evaluation.NeedsReview,
	}
	run := &model.AIEvaluationRun{
		CourseID:      courseID,
		LessonID:      lessonID,
		Provider:      providerName,
		Model:         modelName,
		PromptVersion: promptVersion,
		RunType:       "lesson_evaluation",
		Status:        model.AIEvaluationRunStatusSuccess,
		AttemptCount:  positiveOrDefault(meta.AttemptCount, 1),
		LatencyMS:     meta.LatencyMS,
		InputTokens:   meta.InputTokens,
		OutputTokens:  meta.OutputTokens,
		RawResponse:   meta.RawResponse,
	}
	var saveErr error
	observations := make([]model.MisconceptionObservation, 0, len(evaluation.Misconceptions))
	for _, misconception := range evaluation.Misconceptions {
		patterns := make([]model.ReasoningPattern, 0, len(misconception.ReasoningPatterns))
		for _, pattern := range misconception.ReasoningPatterns {
			patterns = append(patterns, model.ReasoningPattern{PatternKey: pattern.PatternKey, Explanation: pattern.Explanation})
		}
		observations = append(observations, model.MisconceptionObservation{
			OriginalUnderstanding: misconception.OriginalUnderstanding,
			CorrectUnderstanding:  misconception.CorrectUnderstanding,
			BoundaryNotes:         misconception.BoundaryNotes,
			Patterns:              patterns,
		})
	}
	if cognitiveTransition != nil {
		saveErr = s.learning.SaveEvaluationWithPhase6(ctx, turn, mastery, run, cognitiveTransition.State, cognitiveTransition.Evidence, cognitiveTransition.Event, observations)
	} else {
		saveErr = s.learning.SaveEvaluationWithPhase6(ctx, turn, mastery, run, nil, nil, nil, observations)
	}
	if saveErr != nil {
		s.recordFailedEvaluation(ctx, courseID, lessonID, meta, fmt.Errorf("persist evaluation"))
		return nil, saveErr
	}

	return &AnswerResult{
		TurnID:                   turn.ID,
		Result:                   evaluation.Result,
		Feedback:                 evaluation.Feedback,
		Explanation:              evaluation.Explanation,
		CorrectParts:             evaluation.CorrectParts,
		MissingParts:             evaluation.MissingParts,
		Misconceptions:           evaluation.Misconceptions,
		BoundaryConditions:       evaluation.BoundaryConditions,
		MasteryEvidence:          evaluation.MasteryEvidence,
		MasteryScore:             evaluation.MasteryScore,
		NeedsReview:              evaluation.NeedsReview,
		EvaluationSource:         evaluationSource,
		Provider:                 providerName,
		Model:                    modelName,
		PromptVersion:            promptVersion,
		DemonstratedLevel:        evaluation.DemonstratedLevel,
		UserUnderstandingSummary: evaluation.UserUnderstandingSummary,
		CognitiveEvidence:        evaluation.CognitiveEvidence,
		CognitiveState:           cognitiveTransitionState(cognitiveTransition),
	}, nil
}

func (s *CourseService) ListLearningTurns(ctx context.Context, courseID, limit uint) ([]LearningTurnView, error) {
	if _, err := s.findCourse(ctx, courseID); err != nil {
		return nil, err
	}
	turns, err := s.learning.ListLearningTurns(ctx, courseID, int(limit))
	if err != nil {
		return nil, err
	}
	views := make([]LearningTurnView, 0, len(turns))
	for _, turn := range turns {
		correctParts, err := decodeStrings(turn.CorrectParts)
		if err != nil {
			return nil, fmt.Errorf("decode correct parts: %w", err)
		}
		missingParts, err := decodeStrings(turn.MissingParts)
		if err != nil {
			return nil, fmt.Errorf("decode missing parts: %w", err)
		}
		misconceptions, err := decodeMisconceptions(turn.MisconceptionsJSON)
		if err != nil {
			return nil, fmt.Errorf("decode misconceptions: %w", err)
		}
		boundaryConditions, err := decodeStrings(turn.BoundaryConditions)
		if err != nil {
			return nil, fmt.Errorf("decode boundary conditions: %w", err)
		}
		masteryEvidence, err := decodeStrings(turn.MasteryEvidence)
		if err != nil {
			return nil, fmt.Errorf("decode mastery evidence: %w", err)
		}
		cognitiveEvidence, err := decodeEvaluationEvidence(turn.CognitiveEvidenceJSON)
		if err != nil {
			return nil, fmt.Errorf("decode cognitive evidence: %w", err)
		}
		views = append(views, LearningTurnView{
			ID:                       turn.ID,
			Question:                 turn.Question,
			UserAnswer:               turn.UserAnswer,
			Result:                   turn.Result,
			Feedback:                 turn.Feedback,
			Explanation:              turn.Explanation,
			CorrectParts:             correctParts,
			MissingParts:             missingParts,
			Misconceptions:           misconceptions,
			BoundaryConditions:       boundaryConditions,
			MasteryEvidence:          masteryEvidence,
			MasteryScore:             turn.MasteryScore,
			NeedsReview:              turn.NeedsReview,
			EvaluationSource:         turn.EvaluationSource,
			Provider:                 turn.Provider,
			Model:                    turn.Model,
			PromptVersion:            turn.PromptVersion,
			DemonstratedLevel:        turn.DemonstratedLevel,
			UserUnderstandingSummary: turn.UserUnderstandingSummary,
			CognitiveEvidence:        cognitiveEvidence,
			CreatedAt:                turn.CreatedAt,
		})
	}
	return views, nil
}

func (s *CourseService) findCourse(ctx context.Context, courseID uint) (*model.Course, error) {
	if courseID == 0 {
		return nil, ErrInvalidCourseID
	}
	course, err := s.courses.FindByID(ctx, courseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	return course, nil
}

func (s *CourseService) recordFailedEvaluation(ctx context.Context, courseID, lessonID uint, meta ai.ProviderMeta, err error) {
	providerName, modelName, promptVersion := normalizedMeta(meta)
	run := &model.AIEvaluationRun{
		CourseID:      courseID,
		LessonID:      lessonID,
		Provider:      providerName,
		Model:         modelName,
		PromptVersion: promptVersion,
		Status:        model.AIEvaluationRunStatusFailed,
		AttemptCount:  positiveOrDefault(meta.AttemptCount, 1),
		LatencyMS:     meta.LatencyMS,
		InputTokens:   meta.InputTokens,
		OutputTokens:  meta.OutputTokens,
		RawResponse:   meta.RawResponse,
		ErrorMessage:  ai.ErrorCode(err),
	}
	if saveErr := s.learning.SaveAIEvaluationRun(ctx, run); saveErr != nil {
		log.Printf("failed to save AI evaluation audit: %v", saveErr)
	}
}

func (s *CourseService) logInvalidEvaluation(meta ai.ProviderMeta, err error) {
	if !errors.Is(err, ai.ErrInvalidResponse) {
		return
	}
	providerName, modelName, promptVersion := normalizedMeta(meta)
	log.Printf("AI evaluation invalid: provider=%s model=%s prompt_version=%s attempt=%d validation_error=%s", providerName, modelName, promptVersion, positiveOrDefault(meta.AttemptCount, 1), ai.ErrorDetail(err))
}

func normalizedMeta(meta ai.ProviderMeta) (string, string, string) {
	providerName := strings.TrimSpace(meta.Provider)
	if providerName == "" {
		providerName = "unknown"
	}
	modelName := strings.TrimSpace(meta.Model)
	if modelName == "" {
		modelName = "unknown"
	}
	promptVersion := strings.TrimSpace(meta.PromptVersion)
	if promptVersion == "" {
		promptVersion = ai.PromptVersion
	}
	return providerName, modelName, promptVersion
}

func positiveOrDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func marshalJSON(value interface{}) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func decodeStrings(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	if values == nil {
		return []string{}, nil
	}
	return values, nil
}

func decodeMisconceptions(raw string) ([]ai.EvaluationMisconception, error) {
	if strings.TrimSpace(raw) == "" {
		return []ai.EvaluationMisconception{}, nil
	}
	var values []ai.EvaluationMisconception
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	if values == nil {
		return []ai.EvaluationMisconception{}, nil
	}
	return values, nil
}

func decodeEvaluationEvidence(raw string) ([]ai.EvaluationEvidence, error) {
	if strings.TrimSpace(raw) == "" {
		return []ai.EvaluationEvidence{}, nil
	}
	var values []ai.EvaluationEvidence
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	if values == nil {
		return []ai.EvaluationEvidence{}, nil
	}
	return values, nil
}

func cognitiveTransitionState(transition *CognitiveTransition) *CognitiveAnswerState {
	if transition == nil || transition.State == nil {
		return nil
	}
	return &CognitiveAnswerState{CurrentLevel: transition.State.CurrentLevel, Status: transition.State.Status}
}
