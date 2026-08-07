package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"learnos/internal/ai"
	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type phase6Provider struct {
	lessonResult   ai.EvaluationResult
	fallback       *ai.MockProvider
	failTransfer   bool
	timeout        bool
	challengeCalls int
}

func (p *phase6Provider) EvaluateLessonAnswer(context.Context, ai.EvaluationRequest) (ai.EvaluationResult, ai.ProviderMeta, error) {
	raw := p.lessonResult
	return raw, ai.ProviderMeta{Provider: "test", Model: "test", PromptVersion: ai.PromptVersion, AttemptCount: 1}, nil
}

func (p *phase6Provider) GenerateChallenge(ctx context.Context, req ai.ChallengeGenerationRequest) (ai.ChallengeGenerationResult, ai.ProviderMeta, error) {
	return p.fallback.GenerateChallenge(ctx, req)
}

func (p *phase6Provider) EvaluateChallenge(ctx context.Context, req ai.ChallengeEvaluationRequest) (ai.ChallengeEvaluationResult, ai.ProviderMeta, error) {
	p.challengeCalls++
	meta := ai.ProviderMeta{Provider: "deepseek", Model: "test", PromptVersion: ai.ChallengeEvaluatorPromptVersion, AttemptCount: 1}
	if p.timeout {
		return ai.ChallengeEvaluationResult{}, meta, ai.ErrTimeout
	}
	if p.failTransfer && req.ChallengeType == model.ChallengeTypeTransfer {
		return ai.ChallengeEvaluationResult{
			Result: "insufficient", Feedback: "回答不足", Explanation: "没有完成新场景判断", DemonstratedLevel: "exposed",
		}, meta, nil
	}
	return p.fallback.EvaluateChallenge(ctx, req)
}

type challengeFixture struct {
	db            *gorm.DB
	courses       *repository.CourseRepository
	learning      *repository.LearningRepository
	courseService *CourseService
	service       *ChallengeService
	course        model.Course
	lesson        model.Lesson
	provider      *phase6Provider
	misconception *repository.MisconceptionRepository
}

func newChallengeFixture(t *testing.T) challengeFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "phase6.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Course{}, &model.CourseUnit{}, &model.Lesson{}, &model.LessonRelation{}, &model.LearningTurn{},
		&model.MasteryRecord{}, &model.Misconception{}, &model.AIEvaluationRun{}, &model.CognitiveState{},
		&model.CognitiveEvidence{}, &model.CognitiveStateEvent{}, &model.AssessmentChallenge{}, &model.ChallengeAttempt{},
		&model.MisconceptionEvent{}, &model.MisconceptionPatternLink{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	courses := repository.NewCourseRepository(db)
	learning := repository.NewLearningRepository(db)
	graphs := repository.NewKnowledgeGraphRepository(db)
	cognitive := repository.NewCognitiveRepository(db)
	challenges := repository.NewChallengeRepository(db)
	misconceptions := repository.NewMisconceptionRepository(db)
	provider := &phase6Provider{lessonResult: validFakeResult(), fallback: ai.NewMockProvider()}
	courseService := NewCourseService(courses, learning, provider)
	cognitiveService := NewCognitiveStateService(courses, graphs, cognitive)
	courseService.SetCognitiveStateService(cognitiveService)
	if err := courseService.SeedStarterCourse(context.Background()); err != nil {
		t.Fatalf("seed course: %v", err)
	}
	var course model.Course
	if err := db.Where("name = ?", "营养学").First(&course).Error; err != nil {
		t.Fatalf("find course: %v", err)
	}
	var lesson model.Lesson
	if err := db.First(&lesson, *course.CurrentLessonID).Error; err != nil {
		t.Fatalf("find lesson: %v", err)
	}
	return challengeFixture{
		db: db, courses: courses, learning: learning, courseService: courseService,
		service: NewChallengeService(courses, learning, graphs, challenges, misconceptions, cognitiveService, provider),
		course:  course, lesson: lesson, provider: provider, misconception: misconceptions,
	}
}

func TestTransferChallengeAndCorrectionLifecycle(t *testing.T) {
	fixture := newChallengeFixture(t)
	ctx := context.Background()
	if _, err := fixture.courseService.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "口渴只是信号，还要结合环境和身体状态"); err != nil {
		t.Fatalf("seed cognitive state: %v", err)
	}
	challenge, err := fixture.service.Generate(ctx, fixture.course.ID, fixture.lesson.ID, model.ChallengeTypeTransfer, nil)
	if err != nil {
		t.Fatalf("generate transfer challenge: %v", err)
	}
	answer, err := fixture.service.Answer(ctx, fixture.course.ID, challenge.ID, "高温散步后即使不口渴，也要结合年龄、出汗和环境判断")
	if err != nil {
		t.Fatalf("answer transfer challenge: %v", err)
	}
	if !answer.Passed || answer.DemonstratedLevel != model.CognitiveLevelTransfer {
		t.Fatalf("expected passed transfer answer: %+v", answer)
	}
	if fixture.provider.challengeCalls != 1 {
		t.Fatalf("substantive complete answer must call provider once, calls=%d", fixture.provider.challengeCalls)
	}
	var state model.CognitiveState
	if err := fixture.db.Where("course_id = ? AND lesson_id = ?", fixture.course.ID, fixture.lesson.ID).First(&state).Error; err != nil || state.CurrentLevel != model.CognitiveLevelTransfer {
		t.Fatalf("expected transfer cognitive state, state=%+v err=%v", state, err)
	}

	misconceptions, err := fixture.misconception.ListByLesson(ctx, fixture.course.ID, fixture.lesson.ID)
	if err != nil || len(misconceptions) != 1 {
		t.Fatalf("expected one active misconception: %v %#v", err, misconceptions)
	}
	recheck, err := fixture.service.Generate(ctx, fixture.course.ID, fixture.lesson.ID, model.ChallengeTypeMisconceptionRecheck, &misconceptions[0].ID)
	if err != nil {
		t.Fatalf("generate recheck: %v", err)
	}
	correction, err := fixture.service.Answer(ctx, fixture.course.ID, recheck.ID, "不口渴不等于一定不缺水，仍要结合出汗、温度和身体状态")
	if err != nil {
		t.Fatalf("answer recheck: %v", err)
	}
	if !correction.Passed || correction.MisconceptionValidation == nil || correction.MisconceptionValidation.Status != "corrected" {
		t.Fatalf("expected correction: %+v", correction)
	}
	misconceptions, _ = fixture.misconception.ListByLesson(ctx, fixture.course.ID, fixture.lesson.ID)
	if misconceptions[0].Status != model.MisconceptionStatusResolved {
		t.Fatalf("expected resolved misconception: %+v", misconceptions[0])
	}
	var resolvedEvents []model.MisconceptionEvent
	fixture.db.Where("misconception_id = ?", misconceptions[0].ID).Find(&resolvedEvents)
	if len(resolvedEvents) < 2 {
		t.Fatalf("expected observed and resolved events: %#v", resolvedEvents)
	}

	// A later ordinary Lesson observation reopens the same row instead of creating a duplicate.
	if _, err := fixture.courseService.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "口渴是唯一依据"); err != nil {
		t.Fatalf("reobserve misconception: %v", err)
	}
	misconceptions, _ = fixture.misconception.ListByLesson(ctx, fixture.course.ID, fixture.lesson.ID)
	if len(misconceptions) != 1 || misconceptions[0].Status != model.MisconceptionStatusActive {
		t.Fatalf("expected one reopened active misconception: %#v", misconceptions)
	}
	var reopened int64
	fixture.db.Model(&model.MisconceptionEvent{}).Where("misconception_id = ? AND event_type = ?", misconceptions[0].ID, model.MisconceptionEventReopened).Count(&reopened)
	if reopened != 1 {
		t.Fatalf("expected one reopened event, got %d", reopened)
	}
}

func TestFailedTransferDoesNotDowngradeOrReviewStableState(t *testing.T) {
	fixture := newChallengeFixture(t)
	ctx := context.Background()
	if _, err := fixture.courseService.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "口渴只是信号，还要结合环境和身体状态"); err != nil {
		t.Fatalf("seed cognitive state: %v", err)
	}
	challenge, err := fixture.service.Generate(ctx, fixture.course.ID, fixture.lesson.ID, model.ChallengeTypeTransfer, nil)
	if err != nil {
		t.Fatalf("generate transfer challenge: %v", err)
	}
	fixture.provider.failTransfer = true
	answer, err := fixture.service.Answer(ctx, fixture.course.ID, challenge.ID, "不知道")
	if err != nil {
		t.Fatalf("answer failed transfer: %v", err)
	}
	if answer.Passed {
		t.Fatal("insufficient transfer must fail")
	}
	if fixture.provider.challengeCalls != 0 {
		t.Fatalf("non-substantive answer must not call provider, calls=%d", fixture.provider.challengeCalls)
	}
	var state model.CognitiveState
	fixture.db.Where("course_id = ? AND lesson_id = ?", fixture.course.ID, fixture.lesson.ID).First(&state)
	if state.CurrentLevel != model.CognitiveLevelUnderstand || state.Status != model.CognitiveStatusStable {
		t.Fatalf("failed transfer changed stable state: %+v", state)
	}
}

func TestNonSubstantiveTransferAnswerIsPersistedWithoutEvidence(t *testing.T) {
	fixture := newChallengeFixture(t)
	ctx := context.Background()
	if _, err := fixture.courseService.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "口渴只是信号，还要结合环境和身体状态"); err != nil {
		t.Fatalf("seed cognitive state: %v", err)
	}
	challenge, err := fixture.service.Generate(ctx, fixture.course.ID, fixture.lesson.ID, model.ChallengeTypeTransfer, nil)
	if err != nil {
		t.Fatalf("generate challenge: %v", err)
	}
	result, err := fixture.service.Answer(ctx, fixture.course.ID, challenge.ID, " 不知道 ")
	if err != nil {
		t.Fatalf("answer non-substantive challenge: %v", err)
	}
	if result.Result != "insufficient" || result.Passed || result.DemonstratedLevel != model.CognitiveLevelExposed {
		t.Fatalf("unexpected guard result: %+v", result)
	}
	if result.Feedback != "这次回答没有提供足够信息来验证知识迁移。" || result.Explanation != "迁移验证需要把已有知识应用到当前新场景中。本次回答没有展示判断过程，因此暂时无法形成迁移证据。" {
		t.Fatalf("unexpected guard feedback: %+v", result)
	}
	if fixture.provider.challengeCalls != 0 {
		t.Fatalf("non-substantive answer must not call provider, calls=%d", fixture.provider.challengeCalls)
	}
	var storedChallenge model.AssessmentChallenge
	if err := fixture.db.First(&storedChallenge, challenge.ID).Error; err != nil || storedChallenge.Status != model.ChallengeStatusAnswered {
		t.Fatalf("challenge was not answered: %+v err=%v", storedChallenge, err)
	}
	var attempt model.ChallengeAttempt
	if err := fixture.db.Where("challenge_id = ?", challenge.ID).First(&attempt).Error; err != nil || attempt.Result != "insufficient" || attempt.Passed {
		t.Fatalf("attempt was not persisted: %+v err=%v", attempt, err)
	}
	var turn model.LearningTurn
	if err := fixture.db.Where("challenge_id = ?", challenge.ID).First(&turn).Error; err != nil || turn.UserAnswer != "不知道" {
		t.Fatalf("learning turn was not persisted: %+v err=%v", turn, err)
	}
	var evidenceCount int64
	fixture.db.Model(&model.CognitiveEvidence{}).Where("learning_turn_id = ?", turn.ID).Count(&evidenceCount)
	if evidenceCount != 0 {
		t.Fatalf("non-substantive answer created cognitive evidence: %d", evidenceCount)
	}
	var misconceptionCount int64
	fixture.db.Model(&model.MisconceptionEvent{}).Where("challenge_id = ?", challenge.ID).Count(&misconceptionCount)
	if misconceptionCount != 0 {
		t.Fatalf("non-substantive answer created misconception events: %d", misconceptionCount)
	}
}

func TestNonSubstantiveTransferKeepsTransferStableState(t *testing.T) {
	fixture := newChallengeFixture(t)
	ctx := context.Background()
	if _, err := fixture.courseService.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "口渴只是信号，还要结合环境和身体状态"); err != nil {
		t.Fatalf("seed cognitive state: %v", err)
	}
	first, err := fixture.service.Generate(ctx, fixture.course.ID, fixture.lesson.ID, model.ChallengeTypeTransfer, nil)
	if err != nil {
		t.Fatalf("generate first challenge: %v", err)
	}
	if _, err := fixture.service.Answer(ctx, fixture.course.ID, first.ID, "高温散步后即使不口渴，也要结合环境和身体状态判断"); err != nil {
		t.Fatalf("answer first challenge: %v", err)
	}
	second, err := fixture.service.Generate(ctx, fixture.course.ID, fixture.lesson.ID, model.ChallengeTypeTransfer, nil)
	if err != nil {
		t.Fatalf("generate second challenge: %v", err)
	}
	result, err := fixture.service.Answer(ctx, fixture.course.ID, second.ID, "不知道")
	if err != nil || result.Passed {
		t.Fatalf("unexpected non-substantive transfer result: %+v err=%v", result, err)
	}
	var state model.CognitiveState
	fixture.db.Where("course_id = ? AND lesson_id = ?", fixture.course.ID, fixture.lesson.ID).First(&state)
	if state.CurrentLevel != model.CognitiveLevelTransfer || state.Status != model.CognitiveStatusStable {
		t.Fatalf("transfer stable state changed: %+v", state)
	}
}

func TestSubstantiveWrongAnswerStillCallsProvider(t *testing.T) {
	fixture := newChallengeFixture(t)
	ctx := context.Background()
	if _, err := fixture.courseService.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "口渴只是信号，还要结合环境和身体状态"); err != nil {
		t.Fatalf("seed cognitive state: %v", err)
	}
	challenge, err := fixture.service.Generate(ctx, fixture.course.ID, fixture.lesson.ID, model.ChallengeTypeTransfer, nil)
	if err != nil {
		t.Fatalf("generate challenge: %v", err)
	}
	if _, err := fixture.service.Answer(ctx, fixture.course.ID, challenge.ID, "他不口渴，所以肯定不缺水"); err != nil {
		t.Fatalf("answer substantive wrong challenge: %v", err)
	}
	if fixture.provider.challengeCalls != 1 {
		t.Fatalf("substantive wrong answer must call provider once, calls=%d", fixture.provider.challengeCalls)
	}
}

func TestSubstantiveAnswerProviderTimeoutRemainsTimeout(t *testing.T) {
	fixture := newChallengeFixture(t)
	ctx := context.Background()
	if _, err := fixture.courseService.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "口渴只是信号，还要结合环境和身体状态"); err != nil {
		t.Fatalf("seed cognitive state: %v", err)
	}
	challenge, err := fixture.service.Generate(ctx, fixture.course.ID, fixture.lesson.ID, model.ChallengeTypeTransfer, nil)
	if err != nil {
		t.Fatalf("generate challenge: %v", err)
	}
	fixture.provider.timeout = true
	if _, err := fixture.service.Answer(ctx, fixture.course.ID, challenge.ID, "高温环境下即使不口渴，也需要结合出汗和身体状态判断"); !errors.Is(err, ai.ErrTimeout) {
		t.Fatalf("substantive provider timeout was changed: %v", err)
	}
	if fixture.provider.challengeCalls != 1 {
		t.Fatalf("timeout answer should call provider once, calls=%d", fixture.provider.challengeCalls)
	}
	var storedChallenge model.AssessmentChallenge
	fixture.db.First(&storedChallenge, challenge.ID)
	if storedChallenge.Status != model.ChallengeStatusPending {
		t.Fatalf("timed out challenge should remain pending: %+v", storedChallenge)
	}
}

func TestIsNonSubstantiveChallengeAnswerIsConservative(t *testing.T) {
	for _, answer := range []string{"不知道", "不清楚", "不会", "没想法", "不知道怎么回答", "不太清楚", "不确定", " 不知道 "} {
		if !IsNonSubstantiveChallengeAnswer(answer) {
			t.Errorf("expected non-substantive answer: %q", answer)
		}
	}
	for _, answer := range []string{"他不口渴，所以肯定不缺水", "不知道为什么，因为高温", "不口渴"} {
		if IsNonSubstantiveChallengeAnswer(answer) {
			t.Errorf("unexpectedly blocked substantive answer: %q", answer)
		}
	}
}
