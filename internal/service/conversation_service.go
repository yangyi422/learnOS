package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"learnos/internal/ai"
	"learnos/internal/model"

	"gorm.io/gorm"
)

var conversationLocks sync.Map

func lockConversation(courseID, lessonID uint) (func(), bool) {
	value, _ := conversationLocks.LoadOrStore(fmt.Sprintf("%d:%d", courseID, lessonID), make(chan struct{}, 1))
	lock := value.(chan struct{})
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, true
	default:
		return nil, false
	}
}

type ConversationView struct {
	Turns               []model.LearningTurn `json:"turns"`
	HasNext             bool                 `json:"has_next"`
	Position            int                  `json:"position"`
	Total               int                  `json:"total"`
	CompletionSuggested bool                 `json:"completion_suggested"`
}

func (s *CourseService) GetConversation(ctx context.Context, courseID, lessonID uint) (*ConversationView, error) {
	if _, err := s.GetLessonForLearning(ctx, courseID, lessonID); err != nil {
		return nil, err
	}
	turns, err := s.learning.LessonConversation(ctx, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	suggested := false
	for _, turn := range turns {
		if turn.TurnKind == "conversation" && turn.Result == "ready" && turn.EvaluationSource == "ai" {
			suggested = true
		}
	}
	position, total, hasNext, err := s.learning.LessonPosition(ctx, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	return &ConversationView{Turns: turns, CompletionSuggested: suggested, Position: position, Total: total, HasNext: hasNext}, nil
}
func (s *CourseService) Converse(ctx context.Context, courseID, lessonID uint, message, key string) (*ConversationView, error) {
	current, err := s.GetLessonForLearning(ctx, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	message = strings.TrimSpace(message)
	idempotencyKey, err := normalizeIdempotencyKey(key)
	if err != nil || idempotencyKey == nil || message == "" || len([]rune(message)) > 5000 {
		return nil, ErrInvalidAnswer
	}
	unlock, ok := lockConversation(courseID, lessonID)
	if !ok {
		return nil, ErrIdempotencyConflict
	}
	defer unlock()
	turn, err := s.learning.FindLearningTurnByIdempotencyKey(ctx, courseID, *idempotencyKey)
	if err == nil {
		if turn.LessonID != lessonID || turn.TurnKind != "conversation" || turn.UserAnswer != message {
			return nil, ErrIdempotencyConflict
		}
		if turn.Result != "pending" {
			return s.GetConversation(ctx, courseID, lessonID)
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	} else {
		turn = &model.LearningTurn{CourseID: courseID, UnitID: current.Unit.ID, LessonID: lessonID, TurnKind: "conversation", IdempotencyKey: idempotencyKey, Question: current.Lesson.CoreQuestion, UserAnswer: message, Result: "pending", EvaluationSource: "ai"}
		if err := s.learning.CreateConversation(ctx, turn); err != nil {
			return nil, ErrIdempotencyConflict
		}
	}
	history, err := s.learning.LessonConversation(ctx, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	messages := []ai.ConversationMessage{}
	for _, previous := range history {
		if previous.ID < turn.ID && previous.Feedback != "" {
			messages = append(messages, ai.ConversationMessage{User: previous.UserAnswer, Assistant: previous.Feedback})
		}
	}
	// Bound model context; persisted history remains complete.
	if len(messages) > 16 {
		messages = messages[len(messages)-16:]
	}
	provider, ok := s.provider.(ai.ConversationProvider)
	if !ok {
		return nil, ai.ErrNotConfigured
	}
	result, meta, err := provider.Converse(ctx, ai.ConversationRequest{Title: current.Lesson.Title, Content: current.Lesson.Content, Objective: current.Lesson.ExpectedUnderstanding, Question: current.Lesson.CoreQuestion, History: messages, Message: message})
	if err != nil {
		s.recordConversationFailure(ctx, courseID, lessonID, meta, err)
		return nil, err
	}
	result, err = ai.ValidateConversation(result, message)
	if err != nil {
		s.recordConversationFailure(ctx, courseID, lessonID, meta, err)
		return nil, err
	}
	// Save after HTTP cancellation too: a retry can recover the original reply.
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	turn.Feedback = result.Reply
	turn.Result = "replied"
	turn.Provider = meta.Provider
	turn.Model = meta.Model
	turn.PromptVersion = meta.PromptVersion
	turn.EvaluationSource = "ai"
	if meta.Provider == "mock" {
		turn.EvaluationSource = "mock"
		result.Understood = false
	}
	var evidence *model.CognitiveEvidence
	if result.Understood {
		turn.Result = "ready"
		turn.EvidenceUsedJSON = marshalJSON([]string{result.EvidenceQuote})
		turn.Confidence = result.Confidence
		turn.UserUnderstandingSummary = result.EvidenceExplanation
		turn.DemonstratedLevel = "understand"
		evidence = &model.CognitiveEvidence{CourseID: courseID, LessonID: lessonID, EvidenceIndex: 0, EvidenceType: "concept_explanation", CognitiveLevel: "understand", Polarity: "support", Description: result.EvidenceQuote + "\n" + result.EvidenceExplanation, Source: "ai"}
	}
	if err := s.learning.SaveConversationReply(saveCtx, turn, evidence, &model.AIEvaluationRun{CourseID: courseID, LessonID: lessonID, Provider: meta.Provider, Model: meta.Model, PromptVersion: meta.PromptVersion, RunType: "lesson_conversation", Status: model.AIEvaluationRunStatusSuccess, AttemptCount: 1, LatencyMS: meta.LatencyMS, InputTokens: meta.InputTokens, OutputTokens: meta.OutputTokens}); err != nil {
		return nil, err
	}
	return s.GetConversation(saveCtx, courseID, lessonID)
}
func (s *CourseService) AdvanceConversation(ctx context.Context, courseID, lessonID uint, action string) (*CurrentLesson, error) {
	current, err := s.GetLessonForLearning(ctx, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	unlock, ok := lockConversation(courseID, lessonID)
	if !ok {
		return nil, ErrIdempotencyConflict
	}
	defer unlock()
	status := model.LessonStatusSkipped
	if action == "complete" {
		view, err := s.GetConversation(ctx, courseID, lessonID)
		if err != nil {
			return nil, err
		}
		if !view.CompletionSuggested {
			return nil, ErrInvalidAnswer
		}
		status = model.LessonStatusCompleted
	} else if action == "continue" {
		if current.Lesson.Status != model.LessonStatusCompleted && current.Lesson.Status != model.LessonStatusSkipped {
			return nil, ErrInvalidAnswer
		}
		status = current.Lesson.Status
	} else if action != "skip" {
		return nil, ErrInvalidAnswer
	}
	next, err := s.learning.AdvanceLesson(ctx, courseID, lessonID, status)
	if errors.Is(err, gorm.ErrInvalidData) {
		return nil, ErrIdempotencyConflict
	}
	if err != nil {
		return nil, err
	}
	return s.GetLessonForLearning(ctx, courseID, next.ID)
}

func (s *CourseService) RetryConversation(ctx context.Context, courseID, lessonID, turnID uint) (*ConversationView, error) {
	if _, err := s.GetLessonForLearning(ctx, courseID, lessonID); err != nil {
		return nil, err
	}
	turn, err := s.learning.FindLearningTurnByID(ctx, turnID)
	if err != nil || turn.CourseID != courseID || turn.LessonID != lessonID || turn.TurnKind != "conversation" || turn.IdempotencyKey == nil {
		return nil, ErrInvalidAnswer
	}
	return s.Converse(ctx, courseID, lessonID, turn.UserAnswer, *turn.IdempotencyKey)
}

func (s *CourseService) recordConversationFailure(ctx context.Context, courseID, lessonID uint, meta ai.ProviderMeta, cause error) {
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_ = s.learning.SaveAIEvaluationRun(saveCtx, &model.AIEvaluationRun{CourseID: courseID, LessonID: lessonID, Provider: meta.Provider, Model: meta.Model, PromptVersion: meta.PromptVersion, RunType: "lesson_conversation", Status: model.AIEvaluationRunStatusFailed, AttemptCount: 1, ErrorMessage: ai.ErrorCode(cause)})
}
