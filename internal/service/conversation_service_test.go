package service

import (
	"context"
	"errors"
	"testing"

	"learnos/internal/ai"
	"learnos/internal/model"

	"gorm.io/gorm"
)

type conversationFake struct {
	*fakeProvider
	result  ai.ConversationResult
	err     error
	calls   int
	history []ai.ConversationMessage
}

func (p *conversationFake) Converse(_ context.Context, req ai.ConversationRequest) (ai.ConversationResult, ai.ProviderMeta, error) {
	p.calls++
	p.history = req.History
	return p.result, ai.ProviderMeta{Provider: "deepseek", Model: "test"}, p.err
}
func conversationFixture(t *testing.T) (serviceTestFixture, *conversationFake) {
	f := newServiceTestFixture(t, newFakeProvider(validFakeResult()))
	provider := &conversationFake{fakeProvider: f.provider, result: ai.ConversationResult{Reply: "换个例子解释这个概念。", Intent: "question"}}
	f.service.provider = provider
	return f, provider
}
func TestConversationRetryRestoreAndContext(t *testing.T) {
	f, p := conversationFixture(t)
	ctx := context.Background()
	p.err = ai.ErrTimeout
	if _, err := f.service.Converse(ctx, f.course.ID, f.lesson.ID, "为什么会这样？", "chat-1"); !errors.Is(err, ai.ErrTimeout) {
		t.Fatal(err)
	}
	view, err := f.service.GetConversation(ctx, f.course.ID, f.lesson.ID)
	if err != nil || len(view.Turns) != 1 || view.Turns[0].Result != "pending" {
		t.Fatalf("pending message lost: %+v %v", view, err)
	}
	p.err = nil
	view, err = f.service.RetryConversation(ctx, f.course.ID, f.lesson.ID, view.Turns[0].ID)
	if err != nil || len(view.Turns) != 1 || view.Turns[0].Feedback == "" {
		t.Fatalf("retry: %+v %v", view, err)
	}
	calls := p.calls
	if _, err = f.service.Converse(ctx, f.course.ID, f.lesson.ID, "为什么会这样？", "chat-1"); err != nil || p.calls != calls {
		t.Fatal("replay called model", err)
	}
	if _, err = f.service.Converse(ctx, f.course.ID, f.lesson.ID, "其他输入", "chat-1"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatal(err)
	}
	if _, err = f.service.Converse(ctx, f.course.ID, f.lesson.ID, "还是不理解，请换个比喻", "chat-2"); err != nil || len(p.history) != 1 {
		t.Fatal("missing context", err)
	}
	var count int64
	f.db.Model(&model.MasteryRecord{}).Where("lesson_id = ?", f.lesson.ID).Count(&count)
	if count != 0 {
		t.Fatal("question raised mastery")
	}
	facts, err := f.service.courses.ProgressFacts(ctx, f.course.ID)
	if err != nil || facts.CoveredLessonCount != 0 || buildCourseView(f.course, facts).LearningStatus != "in_progress" {
		t.Fatal("chat completed learning implicitly", facts, err)
	}
}
func TestConversationCompletionEvidenceAndStableSuggestion(t *testing.T) {
	f, p := conversationFixture(t)
	ctx := context.Background()
	if _, err := f.service.AdvanceConversation(ctx, f.course.ID, f.lesson.ID, "complete"); !errors.Is(err, ErrInvalidAnswer) {
		t.Fatal("completion without evidence", err)
	}
	p.result = ai.ConversationResult{Reply: "本课目标基本达成。", Intent: "answer", Understood: true, EvidenceQuote: "我懂了", EvidenceExplanation: "同意", Confidence: .9}
	view, err := f.service.Converse(ctx, f.course.ID, f.lesson.ID, "我懂了", "short")
	if err != nil || view.CompletionSuggested {
		t.Fatal("self report counted", err)
	}
	message := "基础代谢是维持生命活动所需的最低能量消耗，运动和消化的额外消耗不属于基础代谢。"
	p.result.EvidenceQuote = message
	p.result.EvidenceExplanation = "区分了基础代谢和其他能量消耗。"
	view, err = f.service.Converse(ctx, f.course.ID, f.lesson.ID, message, "evidence")
	if err != nil || !view.CompletionSuggested {
		t.Fatal("evidence not saved", err)
	}
	p.result = ai.ConversationResult{Reply: "可以继续聊。", Intent: "exploration"}
	view, err = f.service.Converse(ctx, f.course.ID, f.lesson.ID, "还有哪些相关知识？", "extension")
	if err != nil || !view.CompletionSuggested {
		t.Fatal("unstable suggestion", err)
	}
	var evidenceCount int64
	f.db.Model(&model.CognitiveEvidence{}).Where("lesson_id = ?", f.lesson.ID).Count(&evidenceCount)
	if evidenceCount != 1 {
		t.Fatal("evidence missing", evidenceCount)
	}
	next, err := f.service.AdvanceConversation(ctx, f.course.ID, f.lesson.ID, "complete")
	if err != nil || next.Lesson.ID == f.lesson.ID {
		t.Fatal("did not advance", err)
	}
	replay, err := f.service.AdvanceConversation(ctx, f.course.ID, f.lesson.ID, "complete")
	if err != nil || replay.Lesson.ID != next.Lesson.ID {
		t.Fatal("replay advanced twice", err)
	}
	var old model.Lesson
	f.db.First(&old, f.lesson.ID)
	if old.Status != model.LessonStatusCompleted {
		t.Fatal(old.Status)
	}
	var masteryCount int64
	f.db.Model(&model.MasteryRecord{}).Count(&masteryCount)
	if masteryCount != 0 {
		t.Fatal("completion raised mastery")
	}
}
func TestConversationSkipKeepsMasteryAndRejectsStaleAdvance(t *testing.T) {
	f, _ := conversationFixture(t)
	ctx := context.Background()
	next, err := f.service.AdvanceConversation(ctx, f.course.ID, f.lesson.ID, "skip")
	if err != nil {
		t.Fatal(err)
	}
	var lesson model.Lesson
	f.db.First(&lesson, f.lesson.ID)
	if lesson.Status != model.LessonStatusSkipped {
		t.Fatal(lesson.Status)
	}
	facts, err := f.service.courses.ProgressFacts(ctx, f.course.ID)
	if err != nil || facts.CoveredLessonCount == 0 || facts.MasteryPointTotal != 0 {
		t.Fatal("skip conflated progress and mastery", facts, err)
	}
	replay, err := f.service.AdvanceConversation(ctx, f.course.ID, f.lesson.ID, "skip")
	if err != nil || replay.Lesson.ID != next.Lesson.ID {
		t.Fatal("skip replay changed current", err)
	}
	if _, err := f.service.AdvanceConversation(ctx, f.course.ID, f.lesson.ID, "complete"); err == nil {
		t.Fatal("skip changed to complete")
	}
}
func TestConversationInvalidReplyRetainsPendingMessage(t *testing.T) {
	f, p := conversationFixture(t)
	p.result.Reply = ""
	if _, err := f.service.Converse(context.Background(), f.course.ID, f.lesson.ID, "一个问题", "bad"); !errors.Is(err, ai.ErrInvalidResponse) {
		t.Fatal(err)
	}
	view, err := f.service.GetConversation(context.Background(), f.course.ID, f.lesson.ID)
	if err != nil || view.CompletionSuggested || view.Turns[0].Result != "pending" {
		t.Fatal("invalid response changed state", err)
	}
}

func TestConversationAdvanceRollbackAndLegacyCompletion(t *testing.T) {
	f, _ := conversationFixture(t)
	ctx := context.Background()
	callback := "test:fail_course_update"
	if err := f.db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "courses" {
			tx.AddError(errors.New("injected persistence failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AdvanceConversation(ctx, f.course.ID, f.lesson.ID, "skip"); err == nil {
		t.Fatal("expected rollback")
	}
	f.db.Callback().Update().Remove(callback)
	var lesson model.Lesson
	var course model.Course
	f.db.First(&lesson, f.lesson.ID)
	f.db.First(&course, f.course.ID)
	if lesson.Status != f.lesson.Status || *course.CurrentLessonID != f.lesson.ID {
		t.Fatal("advance was not atomic")
	}
	f.db.Model(&lesson).Update("status", model.LessonStatusCompleted)
	if _, err := f.service.AdvanceConversation(ctx, f.course.ID, f.lesson.ID, "continue"); err != nil {
		t.Fatal("legacy completed lesson cannot continue", err)
	}
}

func TestConversationLastLessonKeepsTerminalPosition(t *testing.T) {
	f, _ := conversationFixture(t)
	ctx := context.Background()
	if err := f.db.Model(&model.Lesson{}).Where("course_id = ? AND id <> ?", f.course.ID, f.lesson.ID).Update("status", model.LessonStatusSkipped).Error; err != nil {
		t.Fatal(err)
	}
	last, err := f.service.AdvanceConversation(ctx, f.course.ID, f.lesson.ID, "skip")
	if err != nil || last.Lesson.ID != f.lesson.ID || last.Lesson.Status != model.LessonStatusSkipped {
		t.Fatal(last, err)
	}
	view, err := f.service.GetConversation(ctx, f.course.ID, f.lesson.ID)
	if err != nil || view.HasNext {
		t.Fatal("last lesson offers nonexistent next", err)
	}
	replay, err := f.service.AdvanceConversation(ctx, f.course.ID, f.lesson.ID, "skip")
	if err != nil || replay.Lesson.ID != f.lesson.ID {
		t.Fatal(replay, err)
	}
}
