package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"learnos/internal/ai"
	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrInvalidCognitiveState  = errors.New("invalid cognitive state")
	ErrCognitiveStateNotFound = errors.New("cognitive state not found")
)

type CognitiveStateService struct {
	courses   *repository.CourseRepository
	graphs    *repository.KnowledgeGraphRepository
	cognitive *repository.CognitiveRepository
}

func NewCognitiveStateService(courses *repository.CourseRepository, graphs *repository.KnowledgeGraphRepository, cognitive *repository.CognitiveRepository) *CognitiveStateService {
	return &CognitiveStateService{courses: courses, graphs: graphs, cognitive: cognitive}
}

type CognitiveStateSummary struct {
	LessonID             uint   `json:"lesson_id"`
	CurrentLevel         string `json:"current_level"`
	Status               string `json:"status"`
	UnderstandingSummary string `json:"understanding_summary"`
	EvidenceCount        int    `json:"evidence_count"`
}

type CourseCognitiveStates struct {
	CourseID uint                    `json:"course_id"`
	States   []CognitiveStateSummary `json:"states"`
}

type CognitiveStateDetailLesson struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

type CognitiveStateDetail struct {
	Lesson   CognitiveStateDetailLesson  `json:"lesson"`
	State    CognitiveStateSummaryState  `json:"state"`
	Evidence []model.CognitiveEvidence   `json:"evidence"`
	Timeline []model.CognitiveStateEvent `json:"timeline"`
}

type CognitiveStateSummaryState struct {
	CurrentLevel         string     `json:"current_level"`
	Status               string     `json:"status"`
	UnderstandingSummary string     `json:"understanding_summary"`
	LastEvaluatedAt      *time.Time `json:"last_evaluated_at"`
}

type CognitiveTransition struct {
	State     *model.CognitiveState
	Evidence  []model.CognitiveEvidence
	Event     *model.CognitiveStateEvent
	TurnLevel string
}

func (s *CognitiveStateService) BuildChallengeTransition(ctx context.Context, courseID, lessonID, turnID uint, challengeType string, evaluation ai.ChallengeEvaluationResult, passed, requiresReview bool) (*CognitiveTransition, error) {
	previous, err := s.cognitive.GetByLessonID(ctx, courseID, lessonID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		previous = nil
	}
	fromLevel, fromStatus := model.CognitiveLevelUnseen, model.CognitiveStatusUnknown
	if previous != nil {
		fromLevel, fromStatus = previous.CurrentLevel, previous.Status
	}
	currentLevel := fromLevel
	if passed && cognitiveRank(evaluation.DemonstratedLevel) > cognitiveRank(currentLevel) {
		currentLevel = evaluation.DemonstratedLevel
	}
	toStatus := fromStatus
	if requiresReview && cognitiveRank(fromLevel) >= cognitiveRank(model.CognitiveLevelUnderstand) {
		toStatus = model.CognitiveStatusNeedsReview
	}
	if passed && cognitiveRank(currentLevel) >= cognitiveRank(model.CognitiveLevelUnderstand) {
		toStatus = model.CognitiveStatusStable
	}
	if previous == nil && currentLevel != model.CognitiveLevelUnseen && !passed {
		toStatus = model.CognitiveStatusDeveloping
	}
	if currentLevel == model.CognitiveLevelUnseen {
		toStatus = model.CognitiveStatusUnknown
	}
	if previous == nil && !passed && !requiresReview {
		return &CognitiveTransition{TurnLevel: evaluation.DemonstratedLevel}, nil
	}
	turnIDCopy := turnID
	state := &model.CognitiveState{CourseID: courseID, LessonID: lessonID, CurrentLevel: currentLevel, Status: toStatus, LastLearningTurnID: &turnIDCopy}
	evidence := make([]model.CognitiveEvidence, 0, len(evaluation.CognitiveEvidence))
	for index, item := range evaluation.CognitiveEvidence {
		if challengeType == model.ChallengeTypeTransfer && !passed && item.EvidenceType == model.CognitiveEvidenceTransfer && item.Polarity == model.CognitiveEvidenceSupport {
			continue
		}
		evidence = append(evidence, model.CognitiveEvidence{CourseID: courseID, LessonID: lessonID, LearningTurnID: turnID, EvidenceIndex: index, EvidenceType: item.EvidenceType, CognitiveLevel: item.CognitiveLevel, Polarity: item.Polarity, Description: item.Description, Source: "ai"})
	}
	event := &model.CognitiveStateEvent{CourseID: courseID, LessonID: lessonID, LearningTurnID: turnID, FromLevel: fromLevel, ToLevel: currentLevel, FromStatus: fromStatus, ToStatus: toStatus, Reason: fmt.Sprintf("%s challenge：passed=%t，状态为 %s。", challengeType, passed, toStatus)}
	return &CognitiveTransition{State: state, Evidence: evidence, Event: event, TurnLevel: evaluation.DemonstratedLevel}, nil
}

func (s *CognitiveStateService) GetCourseCognitiveStates(ctx context.Context, courseID uint) (*CourseCognitiveStates, error) {
	if courseID == 0 {
		return nil, ErrInvalidCourseID
	}
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	lessons, err := s.graphs.ListLessonsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	states, err := s.cognitive.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	stateByLesson := make(map[uint]model.CognitiveState, len(states))
	for _, state := range states {
		stateByLesson[state.LessonID] = state
	}
	result := &CourseCognitiveStates{CourseID: courseID, States: make([]CognitiveStateSummary, 0, len(lessons))}
	for _, lesson := range lessons {
		state := stateByLesson[lesson.ID]
		summary := cognitiveStateSummary(state, lesson.ID)
		evidence, err := s.cognitive.ListEvidenceByLesson(ctx, courseID, lesson.ID)
		if err != nil {
			return nil, err
		}
		summary.EvidenceCount = len(evidence)
		result.States = append(result.States, summary)
	}
	return result, nil
}

func (s *CognitiveStateService) GetLessonCognitiveState(ctx context.Context, courseID, lessonID uint) (*CognitiveStateDetail, error) {
	if courseID == 0 {
		return nil, ErrInvalidCourseID
	}
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	lesson, err := s.graphs.FindLessonByCourse(ctx, courseID, lessonID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLessonNotInCourse
		}
		return nil, err
	}
	state, err := s.cognitive.GetByLessonID(ctx, courseID, lessonID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		state = nil
	}
	evidence, err := s.cognitive.ListEvidenceByLesson(ctx, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	timeline, err := s.cognitive.ListEventsByLesson(ctx, courseID, lessonID, 20)
	if err != nil {
		return nil, err
	}
	stateSummary := cognitiveStateSummaryState(state)
	return &CognitiveStateDetail{
		Lesson: CognitiveStateDetailLesson{ID: lesson.ID, Title: lesson.Title},
		State:  stateSummary, Evidence: evidence, Timeline: timeline,
	}, nil
}

func (s *CognitiveStateService) BuildTransition(ctx context.Context, courseID, lessonID, turnID uint, target string, evaluation ai.EvaluationResult) (*CognitiveTransition, error) {
	if err := ValidateAssessmentTarget(target); err != nil {
		return nil, err
	}
	turnLevel, err := CalculateTurnLevel(evaluation, target)
	if err != nil {
		return nil, err
	}
	if err := ValidateCognitiveEvidence(evaluation.CognitiveEvidence, turnLevel); err != nil {
		return nil, err
	}
	previous, err := s.cognitive.GetByLessonID(ctx, courseID, lessonID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		previous = nil
	}
	fromLevel, fromStatus := model.CognitiveLevelUnseen, model.CognitiveStatusUnknown
	understandingSummary := ""
	if previous != nil {
		fromLevel = previous.CurrentLevel
		fromStatus = previous.Status
		understandingSummary = previous.UnderstandingSummary
	}
	currentLevel := maxCognitiveLevel(fromLevel, turnLevel)
	toStatus := CalculateCognitiveStatus(fromLevel, fromStatus, currentLevel, turnLevel, evaluation.Result, evaluation.CognitiveEvidence)
	if cognitiveRank(turnLevel) >= cognitiveRank(model.CognitiveLevelUnderstand) && strings.TrimSpace(evaluation.UserUnderstandingSummary) != "" {
		understandingSummary = strings.TrimSpace(evaluation.UserUnderstandingSummary)
	}
	now := time.Now()
	turnIDCopy := turnID
	state := &model.CognitiveState{
		CourseID: courseID, LessonID: lessonID, CurrentLevel: currentLevel, Status: toStatus,
		UnderstandingSummary: understandingSummary, LastLearningTurnID: &turnIDCopy, LastEvaluatedAt: &now,
	}
	evidence := make([]model.CognitiveEvidence, 0, len(evaluation.CognitiveEvidence))
	for index, item := range evaluation.CognitiveEvidence {
		evidence = append(evidence, model.CognitiveEvidence{
			CourseID: courseID, LessonID: lessonID, LearningTurnID: turnID, EvidenceIndex: index,
			EvidenceType: item.EvidenceType, CognitiveLevel: item.CognitiveLevel, Polarity: item.Polarity,
			Description: item.Description, Source: "ai",
		})
	}
	event := &model.CognitiveStateEvent{
		CourseID: courseID, LessonID: lessonID, LearningTurnID: turnID,
		FromLevel: fromLevel, ToLevel: currentLevel, FromStatus: fromStatus, ToStatus: toStatus,
		UnderstandingSummary: evaluation.UserUnderstandingSummary,
		Reason:               cognitiveEventReason(evaluation.Result, turnLevel, toStatus),
	}
	return &CognitiveTransition{State: state, Evidence: evidence, Event: event, TurnLevel: turnLevel}, nil
}

// ApplyEvaluationToCognitiveState validates and calculates the logical state
// transition. The learning repository persists the returned transition in the
// same transaction as the LearningTurn, so a failed write cannot be partial.
func (s *CognitiveStateService) ApplyEvaluationToCognitiveState(ctx context.Context, courseID, lessonID, turnID uint, target string, evaluation ai.EvaluationResult) (*CognitiveTransition, error) {
	return s.BuildTransition(ctx, courseID, lessonID, turnID, target, evaluation)
}

func ValidateAssessmentTarget(target string) error {
	if cognitiveRank(target) < cognitiveRank(model.CognitiveLevelRecognize) || cognitiveRank(target) > cognitiveRank(model.CognitiveLevelTransfer) {
		return fmt.Errorf("%w: unsupported assessment target %q", ErrInvalidCognitiveState, target)
	}
	return nil
}

func CalculateTurnLevel(evaluation ai.EvaluationResult, target string) (string, error) {
	if err := ValidateAssessmentTarget(target); err != nil {
		return "", err
	}
	if cognitiveRank(evaluation.DemonstratedLevel) < cognitiveRank(model.CognitiveLevelExposed) {
		return "", fmt.Errorf("%w: invalid demonstrated level", ErrInvalidCognitiveState)
	}
	if cognitiveRank(evaluation.DemonstratedLevel) > cognitiveRank(target) {
		return "", fmt.Errorf("%w: demonstrated level exceeds target", ErrInvalidCognitiveState)
	}
	resultCap := model.CognitiveLevelUnderstand
	switch evaluation.Result {
	case "insufficient", "incorrect":
		resultCap = model.CognitiveLevelExposed
	case "partially_correct":
		resultCap = model.CognitiveLevelRecognize
	case "mostly_correct", "correct":
		return evaluation.DemonstratedLevel, nil
	default:
		return "", fmt.Errorf("%w: unsupported result %q", ErrInvalidCognitiveState, evaluation.Result)
	}
	if cognitiveRank(evaluation.DemonstratedLevel) < cognitiveRank(resultCap) {
		return evaluation.DemonstratedLevel, nil
	}
	return resultCap, nil
}

func ValidateCognitiveEvidence(evidence []ai.EvaluationEvidence, turnLevel string) error {
	if len(evidence) > 8 {
		return fmt.Errorf("%w: too many evidence items", ErrInvalidCognitiveState)
	}
	hasSupport := false
	for _, item := range evidence {
		if item.Description == "" || item.EvidenceType == "" || item.CognitiveLevel == "" || item.Polarity == "" {
			return fmt.Errorf("%w: incomplete cognitive evidence", ErrInvalidCognitiveState)
		}
		maxRank := evidenceMaxRank(item.EvidenceType)
		if maxRank < 0 || cognitiveRank(item.CognitiveLevel) < cognitiveRank(model.CognitiveLevelExposed) || cognitiveRank(item.CognitiveLevel) > maxRank {
			return fmt.Errorf("%w: evidence type %q cannot support level %q", ErrInvalidCognitiveState, item.EvidenceType, item.CognitiveLevel)
		}
		if item.Polarity != model.CognitiveEvidenceSupport && item.Polarity != model.CognitiveEvidenceContradict {
			return fmt.Errorf("%w: invalid evidence polarity", ErrInvalidCognitiveState)
		}
		if item.Polarity == model.CognitiveEvidenceSupport && cognitiveRank(item.CognitiveLevel) >= cognitiveRank(turnLevel) {
			hasSupport = true
		}
	}
	if cognitiveRank(turnLevel) >= cognitiveRank(model.CognitiveLevelRecognize) && !hasSupport {
		return fmt.Errorf("%w: demonstrated level has no supporting evidence", ErrInvalidCognitiveState)
	}
	return nil
}

func CalculateCognitiveStatus(previousLevel, previousStatus, currentLevel, turnLevel, result string, evidence []ai.EvaluationEvidence) string {
	if currentLevel == model.CognitiveLevelUnseen {
		return model.CognitiveStatusUnknown
	}
	negative := result == "incorrect" || result == "insufficient" || result == "partially_correct"
	for _, item := range evidence {
		if item.Polarity == model.CognitiveEvidenceContradict {
			negative = true
		}
	}
	if cognitiveRank(previousLevel) >= cognitiveRank(model.CognitiveLevelUnderstand) && negative {
		return model.CognitiveStatusNeedsReview
	}
	if cognitiveRank(currentLevel) >= cognitiveRank(model.CognitiveLevelUnderstand) {
		if negative || cognitiveRank(turnLevel) < cognitiveRank(previousLevel) {
			return model.CognitiveStatusNeedsReview
		}
		return model.CognitiveStatusStable
	}
	return model.CognitiveStatusDeveloping
}

func cognitiveStateSummary(state model.CognitiveState, lessonID uint) CognitiveStateSummary {
	if state.LessonID == 0 {
		return CognitiveStateSummary{LessonID: lessonID, CurrentLevel: model.CognitiveLevelUnseen, Status: model.CognitiveStatusUnknown}
	}
	return CognitiveStateSummary{LessonID: state.LessonID, CurrentLevel: state.CurrentLevel, Status: state.Status, UnderstandingSummary: state.UnderstandingSummary}
}

func cognitiveStateSummaryState(state *model.CognitiveState) CognitiveStateSummaryState {
	if state == nil {
		return CognitiveStateSummaryState{CurrentLevel: model.CognitiveLevelUnseen, Status: model.CognitiveStatusUnknown}
	}
	return CognitiveStateSummaryState{CurrentLevel: state.CurrentLevel, Status: state.Status, UnderstandingSummary: state.UnderstandingSummary, LastEvaluatedAt: state.LastEvaluatedAt}
}

func maxCognitiveLevel(left, right string) string {
	if cognitiveRank(right) > cognitiveRank(left) {
		return right
	}
	return left
}

func cognitiveRank(level string) int {
	switch level {
	case model.CognitiveLevelUnseen:
		return 0
	case model.CognitiveLevelExposed:
		return 1
	case model.CognitiveLevelRecognize:
		return 2
	case model.CognitiveLevelUnderstand:
		return 3
	case model.CognitiveLevelApply:
		return 4
	case model.CognitiveLevelTransfer:
		return 5
	default:
		return -1
	}
}

func evidenceMaxRank(evidenceType string) int {
	switch evidenceType {
	case model.CognitiveEvidenceRecognition:
		return 2
	case model.CognitiveEvidenceConceptExplanation, model.CognitiveEvidenceBoundaryAwareness, model.CognitiveEvidenceContradiction:
		return 3
	case model.CognitiveEvidenceApplication:
		return 4
	case model.CognitiveEvidenceTransfer:
		return 5
	default:
		return -1
	}
}

func cognitiveEventReason(result, turnLevel, status string) string {
	return fmt.Sprintf("本次 %s 评价验证到 %s，状态更新为 %s。", result, turnLevel, status)
}
