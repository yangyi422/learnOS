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
	ErrChallengeNotFound        = errors.New("challenge not found")
	ErrChallengeInvalid         = errors.New("invalid challenge")
	ErrChallengeNotEligible     = errors.New("challenge not eligible")
	ErrChallengeAlreadyAnswered = errors.New("challenge already answered")
	ErrChallengeAIUnavailable   = errors.New("challenge AI provider unavailable")
)

var nonSubstantiveChallengeAnswers = map[string]struct{}{
	"不知道":     {},
	"不清楚":     {},
	"不会":      {},
	"没想法":     {},
	"不知道怎么回答": {},
	"不太清楚":    {},
	"不确定":     {},
}

// IsNonSubstantiveChallengeAnswer identifies answers that provide no usable
// cognitive evidence. It intentionally uses an exact, conservative set so
// that short but meaningful incorrect answers still reach the evaluator.
func IsNonSubstantiveChallengeAnswer(answer string) bool {
	_, ok := nonSubstantiveChallengeAnswers[strings.TrimSpace(answer)]
	return ok
}

type ChallengeService struct {
	courses        *repository.CourseRepository
	learning       *repository.LearningRepository
	graphs         *repository.KnowledgeGraphRepository
	challenges     *repository.ChallengeRepository
	misconceptions *repository.MisconceptionRepository
	cognitive      *CognitiveStateService
	provider       ai.ChallengeProvider
}

func NewChallengeService(courses *repository.CourseRepository, learning *repository.LearningRepository, graphs *repository.KnowledgeGraphRepository, challenges *repository.ChallengeRepository, misconceptions *repository.MisconceptionRepository, cognitive *CognitiveStateService, provider ai.ChallengeProvider) *ChallengeService {
	return &ChallengeService{courses: courses, learning: learning, graphs: graphs, challenges: challenges, misconceptions: misconceptions, cognitive: cognitive, provider: provider}
}

type ChallengeView struct {
	ID                    uint     `json:"id"`
	CourseID              uint     `json:"course_id"`
	LessonID              uint     `json:"lesson_id"`
	ChallengeType         string   `json:"challenge_type"`
	TargetMisconceptionID *uint    `json:"target_misconception_id"`
	Prompt                string   `json:"prompt"`
	ScenarioContext       string   `json:"scenario_context"`
	EvaluationCriteria    []string `json:"evaluation_criteria"`
	WhyThisIsTransfer     string   `json:"why_this_is_transfer"`
	SourceConcepts        []string `json:"source_concepts"`
	TargetLevel           string   `json:"target_level"`
	Status                string   `json:"status"`
	Provider              string   `json:"provider"`
	Model                 string   `json:"model"`
	PromptVersion         string   `json:"prompt_version"`
	CreatedAt             string   `json:"created_at"`
}

type ChallengeAnswerResult struct {
	ChallengeID             uint                        `json:"challenge_id"`
	AttemptID               uint                        `json:"attempt_id"`
	LearningTurnID          uint                        `json:"learning_turn_id"`
	Result                  string                      `json:"result"`
	DemonstratedLevel       string                      `json:"demonstrated_level"`
	Passed                  bool                        `json:"passed"`
	Feedback                string                      `json:"feedback"`
	Explanation             string                      `json:"explanation"`
	CognitiveEvidence       []ai.EvaluationEvidence     `json:"cognitive_evidence"`
	CognitiveState          *CognitiveAnswerState       `json:"cognitive_state"`
	MisconceptionValidation *ai.MisconceptionValidation `json:"misconception_validation"`
}

func (s *ChallengeService) Generate(ctx context.Context, courseID, lessonID uint, challengeType string, misconceptionID *uint) (*ChallengeView, error) {
	if challengeType != model.ChallengeTypeTransfer && challengeType != model.ChallengeTypeMisconceptionRecheck {
		return nil, ErrChallengeInvalid
	}
	course, lesson, unit, err := s.courseLesson(ctx, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	var target *model.Misconception
	if challengeType == model.ChallengeTypeTransfer {
		if misconceptionID != nil {
			return nil, ErrChallengeInvalid
		}
		if s.cognitive == nil {
			return nil, ErrChallengeNotEligible
		}
		detail, detailErr := s.cognitive.GetLessonCognitiveState(ctx, courseID, lessonID)
		if detailErr != nil {
			return nil, detailErr
		}
		if cognitiveRank(detail.State.CurrentLevel) < cognitiveRank(model.CognitiveLevelUnderstand) {
			return nil, ErrChallengeNotEligible
		}
	} else {
		if misconceptionID == nil || *misconceptionID == 0 {
			return nil, ErrChallengeInvalid
		}
		target, err = s.misconceptions.FindByID(ctx, courseID, lessonID, *misconceptionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrChallengeInvalid
			}
			return nil, err
		}
		if target.Status != model.MisconceptionStatusActive {
			return nil, ErrChallengeInvalid
		}
	}
	if s.provider == nil {
		return nil, ErrChallengeAIUnavailable
	}
	prerequisites, relationContext := s.relationContext(ctx, courseID, lessonID)
	detail := &CognitiveStateDetail{State: CognitiveStateSummaryState{CurrentLevel: model.CognitiveLevelUnseen, Status: model.CognitiveStatusUnknown}}
	if s.cognitive != nil {
		if currentDetail, detailErr := s.cognitive.GetLessonCognitiveState(ctx, courseID, lessonID); detailErr == nil {
			detail = currentDetail
		}
	}
	active, _ := s.misconceptions.ListByLesson(ctx, courseID, lessonID)
	activeMisconceptions := make([]ai.EvaluationMisconception, 0)
	for _, item := range active {
		if item.Status == model.MisconceptionStatusActive {
			activeMisconceptions = append(activeMisconceptions, ai.EvaluationMisconception{OriginalUnderstanding: item.OriginalUnderstanding, CorrectUnderstanding: item.CorrectUnderstanding, BoundaryNotes: item.BoundaryNotes})
		}
	}
	var targetEvaluation *ai.EvaluationMisconception
	if target != nil {
		targetEvaluation = &ai.EvaluationMisconception{OriginalUnderstanding: target.OriginalUnderstanding, CorrectUnderstanding: target.CorrectUnderstanding, BoundaryNotes: target.BoundaryNotes}
	}
	req := ai.ChallengeGenerationRequest{CourseName: course.Name, UnitTitle: unit.Title, LessonTitle: lesson.Title, CoreQuestion: lesson.CoreQuestion, ExpectedUnderstanding: lesson.ExpectedUnderstanding, PrerequisiteLessonTitles: prerequisites, RelationContext: relationContext, CurrentCognitiveLevel: detail.State.CurrentLevel, UnderstandingSummary: detail.State.UnderstandingSummary, ActiveMisconceptions: activeMisconceptions, TargetMisconception: targetEvaluation, ChallengeType: challengeType}
	generated, meta, err := s.provider.GenerateChallenge(ctx, req)
	if err != nil {
		s.recordFailedRun(ctx, courseID, lessonID, meta, "challenge_generation", err)
		return nil, err
	}
	generated, err = ai.NormalizeAndValidateChallengeGeneration(generated, challengeType)
	if err != nil {
		s.recordFailedRun(ctx, courseID, lessonID, meta, "challenge_generation", err)
		return nil, fmt.Errorf("%w: %v", ai.ErrInvalidResponse, err)
	}
	criteriaJSON, _ := json.Marshal(generated.EvaluationCriteria)
	conceptsJSON, _ := json.Marshal(generated.SourceConcepts)
	challenge := &model.AssessmentChallenge{CourseID: courseID, LessonID: lessonID, ChallengeType: challengeType, Prompt: generated.Prompt, ScenarioContext: generated.ScenarioContext, EvaluationCriteriaJSON: string(criteriaJSON), WhyThisIsTransfer: generated.WhyThisIsTransfer, SourceConceptsJSON: string(conceptsJSON), TargetLevel: generated.TargetLevel, Status: model.ChallengeStatusPending, Provider: meta.Provider, Model: meta.Model, PromptVersion: meta.PromptVersion}
	if target != nil {
		challenge.TargetMisconceptionID = &target.ID
	}
	run := &model.AIEvaluationRun{Provider: meta.Provider, Model: meta.Model, PromptVersion: meta.PromptVersion, RunType: "challenge_generation", Status: model.AIEvaluationRunStatusSuccess, AttemptCount: positiveOrDefault(meta.AttemptCount, 1), LatencyMS: meta.LatencyMS, InputTokens: meta.InputTokens, OutputTokens: meta.OutputTokens, RawResponse: meta.RawResponse}
	if err := s.challenges.CreateWithRun(ctx, challenge, run); err != nil {
		return nil, err
	}
	return challengeView(challenge), nil
}

func (s *ChallengeService) Answer(ctx context.Context, courseID, challengeID uint, answer string) (*ChallengeAnswerResult, error) {
	answer = strings.TrimSpace(answer)
	if answer == "" || len([]rune(answer)) > 5000 {
		return nil, ErrInvalidAnswer
	}
	challenge, err := s.challenges.FindByID(ctx, courseID, challengeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChallengeNotFound
		}
		return nil, err
	}
	if challenge.Status != model.ChallengeStatusPending {
		return nil, ErrChallengeAlreadyAnswered
	}
	course, lesson, unit, err := s.courseLesson(ctx, courseID, challenge.LessonID)
	if err != nil {
		return nil, err
	}
	var target *model.Misconception
	if challenge.ChallengeType == model.ChallengeTypeMisconceptionRecheck {
		if challenge.TargetMisconceptionID == nil {
			return nil, ErrChallengeInvalid
		}
		target, err = s.misconceptions.FindByID(ctx, courseID, challenge.LessonID, *challenge.TargetMisconceptionID)
		if err != nil {
			return nil, ErrChallengeInvalid
		}
		if target.Status != model.MisconceptionStatusActive {
			return nil, ErrChallengeInvalid
		}
	}
	criteria := []string{}
	_ = json.Unmarshal([]byte(challenge.EvaluationCriteriaJSON), &criteria)
	targetEval := (*ai.EvaluationMisconception)(nil)
	targetID := uint(0)
	if target != nil {
		targetEval = &ai.EvaluationMisconception{OriginalUnderstanding: target.OriginalUnderstanding, CorrectUnderstanding: target.CorrectUnderstanding, BoundaryNotes: target.BoundaryNotes}
		targetID = target.ID
	}
	var evaluation ai.ChallengeEvaluationResult
	var meta ai.ProviderMeta
	if challenge.ChallengeType == model.ChallengeTypeTransfer && IsNonSubstantiveChallengeAnswer(answer) {
		evaluation = ai.ChallengeEvaluationResult{
			Result:            "insufficient",
			Feedback:          "这次回答没有提供足够信息来验证知识迁移。",
			Explanation:       "迁移验证需要把已有知识应用到当前新场景中。本次回答没有展示判断过程，因此暂时无法形成迁移证据。",
			DemonstratedLevel: model.CognitiveLevelExposed,
			CognitiveEvidence: []ai.EvaluationEvidence{},
			Misconceptions:    []ai.EvaluationMisconception{},
		}
		meta = ai.ProviderMeta{Provider: "local", Model: "non_substantive_guard", PromptVersion: ai.ChallengeEvaluatorPromptVersion}
	} else {
		if s.provider == nil {
			return nil, ErrChallengeAIUnavailable
		}
		evaluation, meta, err = s.provider.EvaluateChallenge(ctx, ai.ChallengeEvaluationRequest{ChallengeType: challenge.ChallengeType, CourseName: course.Name, UnitTitle: unit.Title, LessonTitle: lesson.Title, CoreQuestion: lesson.CoreQuestion, ExpectedUnderstanding: lesson.ExpectedUnderstanding, ChallengePrompt: challenge.Prompt, ScenarioContext: challenge.ScenarioContext, EvaluationCriteria: criteria, TargetLevel: challenge.TargetLevel, TargetMisconception: targetEval, TargetMisconceptionID: targetID, UserAnswer: answer})
		if err != nil {
			s.recordFailedRun(ctx, courseID, challenge.LessonID, meta, "challenge_evaluation", err)
			return nil, err
		}
	}
	evaluation, err = ai.NormalizeAndValidateChallengeEvaluation(evaluation, challenge.ChallengeType, challenge.TargetLevel, targetID)
	if err != nil {
		s.recordFailedRun(ctx, courseID, challenge.LessonID, meta, "challenge_evaluation", err)
		return nil, fmt.Errorf("%w: %v", ai.ErrInvalidResponse, err)
	}
	passed := ai.CalculateChallengePassed(evaluation, challenge.ChallengeType, targetID)
	requiresReview := s.challengeRequiresReview(ctx, courseID, challenge.LessonID, evaluation)
	transition, err := s.cognitive.BuildChallengeTransition(ctx, courseID, challenge.LessonID, 0, challenge.ChallengeType, evaluation, passed, requiresReview)
	if err != nil {
		return nil, err
	}
	turn := &model.LearningTurn{CourseID: courseID, UnitID: unit.ID, LessonID: lesson.ID, TurnKind: turnKind(challenge.ChallengeType), ChallengeID: &challenge.ID, Question: challenge.Prompt, UserAnswer: answer, Result: evaluation.Result, Feedback: evaluation.Feedback, Explanation: evaluation.Explanation, MisconceptionsJSON: marshalJSON(evaluation.Misconceptions), CognitiveEvidenceJSON: marshalJSON(evaluation.CognitiveEvidence), EvaluationSource: sourceFromMeta(meta), Provider: meta.Provider, Model: meta.Model, PromptVersion: meta.PromptVersion, DemonstratedLevel: evaluation.DemonstratedLevel, MasteryScore: 0, NeedsReview: requiresReview}
	validationJSON := marshalJSON(evaluation.MisconceptionValidation)
	attempt := &model.ChallengeAttempt{CourseID: courseID, LessonID: lesson.ID, ChallengeID: challenge.ID, UserAnswer: answer, Result: evaluation.Result, DemonstratedLevel: evaluation.DemonstratedLevel, Passed: passed, Feedback: evaluation.Feedback, EvidenceJSON: marshalJSON(evaluation.CognitiveEvidence), MisconceptionValidationJSON: validationJSON}
	updates := []model.Misconception{}
	events := []model.MisconceptionEvent{}
	observations := make([]model.MisconceptionObservation, 0, len(evaluation.Misconceptions))
	for _, misconception := range evaluation.Misconceptions {
		patterns := make([]model.ReasoningPattern, 0, len(misconception.ReasoningPatterns))
		for _, pattern := range misconception.ReasoningPatterns {
			patterns = append(patterns, model.ReasoningPattern{PatternKey: pattern.PatternKey, Explanation: pattern.Explanation})
		}
		observations = append(observations, model.MisconceptionObservation{OriginalUnderstanding: misconception.OriginalUnderstanding, CorrectUnderstanding: misconception.CorrectUnderstanding, BoundaryNotes: misconception.BoundaryNotes, Patterns: patterns})
	}
	if target != nil {
		if passed && evaluation.MisconceptionValidation != nil && evaluation.MisconceptionValidation.Status == "corrected" {
			now := timeNow()
			target.Status = model.MisconceptionStatusResolved
			target.ResolvedAt = &now
			target.ResolvedByChallengeID = &challenge.ID
			updates = append(updates, *target)
			events = append(events, model.MisconceptionEvent{MisconceptionID: target.ID, EventType: model.MisconceptionEventResolved, Notes: evaluation.MisconceptionValidation.Evidence})
		} else if evaluation.MisconceptionValidation != nil && evaluation.MisconceptionValidation.Status == "persists" {
			events = append(events, model.MisconceptionEvent{MisconceptionID: target.ID, EventType: model.MisconceptionEventObserved, Notes: evaluation.MisconceptionValidation.Evidence})
		}
	}
	run := &model.AIEvaluationRun{CourseID: courseID, LessonID: lesson.ID, Provider: meta.Provider, Model: meta.Model, PromptVersion: meta.PromptVersion, RunType: "challenge_evaluation", Status: model.AIEvaluationRunStatusSuccess, AttemptCount: positiveOrDefault(meta.AttemptCount, 1), LatencyMS: meta.LatencyMS, InputTokens: meta.InputTokens, OutputTokens: meta.OutputTokens, RawResponse: meta.RawResponse}
	if err := s.challenges.SaveChallengeResultWithObservations(ctx, turn, attempt, challenge, transition.State, transition.Evidence, transition.Event, updates, events, nil, observations, run); err != nil {
		return nil, err
	}
	return &ChallengeAnswerResult{ChallengeID: challenge.ID, AttemptID: attempt.ID, LearningTurnID: turn.ID, Result: evaluation.Result, DemonstratedLevel: evaluation.DemonstratedLevel, Passed: passed, Feedback: evaluation.Feedback, Explanation: evaluation.Explanation, CognitiveEvidence: evaluation.CognitiveEvidence, CognitiveState: cognitiveTransitionState(transition), MisconceptionValidation: evaluation.MisconceptionValidation}, nil
}

func (s *ChallengeService) courseLesson(ctx context.Context, courseID, lessonID uint) (*model.Course, *model.Lesson, *model.CourseUnit, error) {
	if courseID == 0 || lessonID == 0 {
		return nil, nil, nil, ErrInvalidCourseID
	}
	course, err := s.courses.FindByID(ctx, courseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, ErrCourseNotFound
		}
		return nil, nil, nil, err
	}
	lesson, err := s.graphs.FindLessonByCourse(ctx, courseID, lessonID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, ErrLessonNotInCourse
		}
		return nil, nil, nil, err
	}
	unit, err := s.learning.FindUnitByID(ctx, lesson.UnitID)
	if err != nil {
		return nil, nil, nil, err
	}
	return course, lesson, unit, nil
}

func (s *ChallengeService) relationContext(ctx context.Context, courseID, lessonID uint) ([]string, []string) {
	relations, err := s.graphs.ListRelationsForLesson(ctx, courseID, lessonID)
	if err != nil {
		return []string{}, []string{}
	}
	lessons, err := s.graphs.ListLessonsByCourse(ctx, courseID)
	if err != nil {
		return []string{}, []string{}
	}
	titles := map[uint]string{}
	for _, lesson := range lessons {
		titles[lesson.ID] = lesson.Title
	}
	prerequisites, contextItems := []string{}, []string{}
	for _, relation := range relations {
		neighbor := relation.FromLessonID
		if neighbor == lessonID {
			neighbor = relation.ToLessonID
		}
		text := fmt.Sprintf("%s: %s", relation.RelationType, titles[neighbor])
		contextItems = append(contextItems, text)
		if relation.RelationType == model.LessonRelationPrerequisite && relation.ToLessonID == lessonID {
			prerequisites = append(prerequisites, titles[relation.FromLessonID])
		}
	}
	return prerequisites, contextItems
}

func (s *ChallengeService) challengeRequiresReview(ctx context.Context, courseID, lessonID uint, evaluation ai.ChallengeEvaluationResult) bool {
	for _, evidence := range evaluation.CognitiveEvidence {
		if evidence.Polarity == model.CognitiveEvidenceContradict {
			return true
		}
	}
	active, err := s.misconceptions.ListByLesson(ctx, courseID, lessonID)
	if err != nil {
		return false
	}
	for _, observed := range evaluation.Misconceptions {
		for _, existing := range active {
			if existing.Status == model.MisconceptionStatusActive && strings.TrimSpace(existing.OriginalUnderstanding) == strings.TrimSpace(observed.OriginalUnderstanding) && strings.TrimSpace(existing.CorrectUnderstanding) == strings.TrimSpace(observed.CorrectUnderstanding) {
				return true
			}
		}
	}
	if evaluation.MisconceptionValidation != nil && evaluation.MisconceptionValidation.Status == "persists" {
		return true
	}
	return false
}

func (s *ChallengeService) recordFailedRun(ctx context.Context, courseID, lessonID uint, meta ai.ProviderMeta, runType string, err error) {
	if errors.Is(err, ai.ErrInvalidResponse) {
		log.Printf("AI challenge evaluation invalid: provider=%s model=%s prompt_version=%s attempt=%d validation_error=%s", meta.Provider, meta.Model, meta.PromptVersion, positiveOrDefault(meta.AttemptCount, 1), ai.ErrorDetail(err))
	}
	_ = s.learning.SaveAIEvaluationRun(ctx, &model.AIEvaluationRun{CourseID: courseID, LessonID: lessonID, Provider: meta.Provider, Model: meta.Model, PromptVersion: meta.PromptVersion, RunType: runType, Status: model.AIEvaluationRunStatusFailed, AttemptCount: positiveOrDefault(meta.AttemptCount, 1), LatencyMS: meta.LatencyMS, InputTokens: meta.InputTokens, OutputTokens: meta.OutputTokens, RawResponse: meta.RawResponse, ErrorMessage: ai.ErrorCode(err)})
}

func challengeView(challenge *model.AssessmentChallenge) *ChallengeView {
	criteria, _ := decodeStrings(challenge.EvaluationCriteriaJSON)
	concepts, _ := decodeStrings(challenge.SourceConceptsJSON)
	return &ChallengeView{ID: challenge.ID, CourseID: challenge.CourseID, LessonID: challenge.LessonID, ChallengeType: challenge.ChallengeType, TargetMisconceptionID: challenge.TargetMisconceptionID, Prompt: challenge.Prompt, ScenarioContext: challenge.ScenarioContext, EvaluationCriteria: criteria, WhyThisIsTransfer: challenge.WhyThisIsTransfer, SourceConcepts: concepts, TargetLevel: challenge.TargetLevel, Status: challenge.Status, Provider: challenge.Provider, Model: challenge.Model, PromptVersion: challenge.PromptVersion, CreatedAt: challenge.CreatedAt.Format(time.RFC3339)}
}

func turnKind(challengeType string) string {
	if challengeType == model.ChallengeTypeMisconceptionRecheck {
		return model.TurnKindMisconceptionRecheck
	}
	return model.TurnKindTransferChallenge
}
func sourceFromMeta(meta ai.ProviderMeta) string {
	if meta.Provider == "local" {
		return "local"
	}
	if meta.Provider == "mock" {
		return "mock"
	}
	return "ai"
}
func timeNow() time.Time { return time.Now() }
