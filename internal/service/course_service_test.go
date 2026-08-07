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

type fakeProvider struct {
	result ai.EvaluationResult
	meta   ai.ProviderMeta
	err    error
	calls  int
}

func (p *fakeProvider) EvaluateLessonAnswer(_ context.Context, _ ai.EvaluationRequest) (ai.EvaluationResult, ai.ProviderMeta, error) {
	p.calls++
	return p.result, p.meta, p.err
}

type serviceTestFixture struct {
	db       *gorm.DB
	service  *CourseService
	course   model.Course
	lesson   model.Lesson
	provider *fakeProvider
}

func newServiceTestFixture(t *testing.T, provider *fakeProvider) serviceTestFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Course{}, &model.CourseUnit{}, &model.Lesson{}, &model.LessonRelation{}, &model.LearningTurn{},
		&model.MasteryRecord{}, &model.Misconception{}, &model.AIEvaluationRun{}, &model.CognitiveState{}, &model.CognitiveEvidence{}, &model.CognitiveStateEvent{},
		&model.AssessmentChallenge{}, &model.ChallengeAttempt{}, &model.MisconceptionEvent{}, &model.MisconceptionPatternLink{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	courses := repository.NewCourseRepository(db)
	learning := repository.NewLearningRepository(db)
	service := NewCourseService(courses, learning, provider)
	graphs := repository.NewKnowledgeGraphRepository(db)
	cognitive := repository.NewCognitiveRepository(db)
	service.SetCognitiveStateService(NewCognitiveStateService(courses, graphs, cognitive))
	if err := service.SeedStarterCourse(context.Background()); err != nil {
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
	return serviceTestFixture{db: db, service: service, course: course, lesson: lesson, provider: provider}
}

func validFakeResult() ai.EvaluationResult {
	return ai.EvaluationResult{
		Result:      "mostly_correct",
		Feedback:    "反馈",
		Explanation: "解释",
		CorrectParts: []string{
			"正确点",
		},
		MissingParts: []string{
			"缺失点",
		},
		Misconceptions: []ai.EvaluationMisconception{
			{
				OriginalUnderstanding: "把不口渴理解为一定不缺水",
				CorrectUnderstanding:  "口渴不是所有场景下唯一的补水依据",
				BoundaryNotes:         "高温和运动等场景需要额外考虑",
			},
		},
		BoundaryConditions:       []string{"日常低强度环境仍可参考口渴"},
		MasteryEvidence:          []string{"识别了单一信号的局限"},
		MasteryScore:             0.78,
		NeedsReview:              true,
		DemonstratedLevel:        "understand",
		UserUnderstandingSummary: "用户能解释口渴不是唯一依据，并能指出环境和身体状态会影响判断。",
		CognitiveEvidence: []ai.EvaluationEvidence{
			{EvidenceType: "recognition", CognitiveLevel: "recognize", Polarity: "support", Description: "识别单一口渴信号的局限"},
			{EvidenceType: "boundary_awareness", CognitiveLevel: "understand", Polarity: "support", Description: "指出环境和身体状态会影响判断"},
		},
	}
}

func newFakeProvider(result ai.EvaluationResult) *fakeProvider {
	return &fakeProvider{
		result: result,
		meta: ai.ProviderMeta{
			Provider:      "deepseek",
			Model:         "deepseek-v4-flash",
			PromptVersion: ai.PromptVersion,
			RawResponse:   `{"result":"mostly_correct"}`,
			AttemptCount:  1,
		},
	}
}

func TestSubmitAnswerPersistsAIResultAuditMasteryAndMisconception(t *testing.T) {
	fixture := newServiceTestFixture(t, newFakeProvider(validFakeResult()))
	if _, err := fixture.service.SubmitAnswer(context.Background(), fixture.course.ID, fixture.lesson.ID, "不能只看口渴，还要结合环境和身体状态。"); err != nil {
		t.Fatalf("submit answer: %v", err)
	}
	var turn model.LearningTurn
	if err := fixture.db.First(&turn).Error; err != nil {
		t.Fatalf("find learning turn: %v", err)
	}
	if turn.EvaluationSource != "ai" || turn.Provider != "deepseek" || turn.PromptVersion != ai.PromptVersion || turn.Explanation == "" {
		t.Fatalf("unexpected learning turn snapshot: %+v", turn)
	}
	var mastery model.MasteryRecord
	if err := fixture.db.First(&mastery).Error; err != nil {
		t.Fatalf("find mastery record: %v", err)
	}
	if mastery.AnswerCount != 1 || mastery.MasteryScore != 0.78 || !mastery.NeedsReview {
		t.Fatalf("unexpected mastery record: %+v", mastery)
	}
	var misconceptionCount int64
	if err := fixture.db.Model(&model.Misconception{}).Count(&misconceptionCount).Error; err != nil {
		t.Fatalf("count misconceptions: %v", err)
	}
	if misconceptionCount != 1 {
		t.Fatalf("expected one misconception, got %d", misconceptionCount)
	}
	var run model.AIEvaluationRun
	if err := fixture.db.First(&run).Error; err != nil {
		t.Fatalf("find evaluation run: %v", err)
	}
	if run.Status != model.AIEvaluationRunStatusSuccess || run.LearningTurnID == nil || *run.LearningTurnID != turn.ID {
		t.Fatalf("unexpected evaluation run: %+v", run)
	}

	if _, err := fixture.service.SubmitAnswer(context.Background(), fixture.course.ID, fixture.lesson.ID, "再次回答"); err != nil {
		t.Fatalf("submit duplicate misconception: %v", err)
	}
	if err := fixture.db.Model(&model.Misconception{}).Count(&misconceptionCount).Error; err != nil {
		t.Fatalf("count misconceptions after duplicate: %v", err)
	}
	if misconceptionCount != 1 {
		t.Fatalf("duplicate misconception was persisted: %d", misconceptionCount)
	}
}

func TestSubmitAnswerProviderFailureDoesNotChangeLearningState(t *testing.T) {
	provider := newFakeProvider(validFakeResult())
	provider.err = ai.ErrTimeout
	fixture := newServiceTestFixture(t, provider)
	if _, err := fixture.service.SubmitAnswer(context.Background(), fixture.course.ID, fixture.lesson.ID, "回答失败测试"); !errors.Is(err, ai.ErrTimeout) {
		t.Fatalf("expected timeout, got %v", err)
	}
	var turnCount, masteryCount, successRunCount, failedRunCount int64
	fixture.db.Model(&model.LearningTurn{}).Count(&turnCount)
	fixture.db.Model(&model.MasteryRecord{}).Count(&masteryCount)
	fixture.db.Model(&model.AIEvaluationRun{}).Where("status = ?", model.AIEvaluationRunStatusSuccess).Count(&successRunCount)
	fixture.db.Model(&model.AIEvaluationRun{}).Where("status = ?", model.AIEvaluationRunStatusFailed).Count(&failedRunCount)
	if turnCount != 0 || masteryCount != 0 || successRunCount != 0 || failedRunCount != 1 {
		t.Fatalf("unexpected state after provider failure: turns=%d mastery=%d success_runs=%d failed_runs=%d", turnCount, masteryCount, successRunCount, failedRunCount)
	}
}

func TestSubmitAnswerRejectsInvalidProviderResultWithoutLearningState(t *testing.T) {
	provider := newFakeProvider(validFakeResult())
	provider.result.Result = "not_allowed"
	fixture := newServiceTestFixture(t, provider)
	if _, err := fixture.service.SubmitAnswer(context.Background(), fixture.course.ID, fixture.lesson.ID, "非法结果测试"); !errors.Is(err, ai.ErrInvalidResponse) {
		t.Fatalf("expected invalid response, got %v", err)
	}
	var turnCount, masteryCount, failedRunCount int64
	fixture.db.Model(&model.LearningTurn{}).Count(&turnCount)
	fixture.db.Model(&model.MasteryRecord{}).Count(&masteryCount)
	fixture.db.Model(&model.AIEvaluationRun{}).Where("status = ?", model.AIEvaluationRunStatusFailed).Count(&failedRunCount)
	if turnCount != 0 || masteryCount != 0 || failedRunCount != 1 {
		t.Fatalf("unexpected state after invalid response: turns=%d mastery=%d failed_runs=%d", turnCount, masteryCount, failedRunCount)
	}
}

func TestSubmitAnswerRejectsNonCurrentLessonBeforeProvider(t *testing.T) {
	provider := newFakeProvider(validFakeResult())
	fixture := newServiceTestFixture(t, provider)
	if _, err := fixture.service.SubmitAnswer(context.Background(), fixture.course.ID, fixture.lesson.ID+100, "不应调用 Provider"); !errors.Is(err, ErrLessonNotCurrent) {
		t.Fatalf("expected current lesson error, got %v", err)
	}
	if provider.calls != 0 {
		t.Fatalf("provider was called %d times", provider.calls)
	}
}

func TestIncorrectAndInsufficientIncreaseIncorrectCount(t *testing.T) {
	for _, result := range []string{"incorrect", "insufficient"} {
		t.Run(result, func(t *testing.T) {
			provider := newFakeProvider(validFakeResult())
			provider.result.Result = result
			fixture := newServiceTestFixture(t, provider)
			if _, err := fixture.service.SubmitAnswer(context.Background(), fixture.course.ID, fixture.lesson.ID, "足够长的回答"); err != nil {
				t.Fatalf("submit answer: %v", err)
			}
			var mastery model.MasteryRecord
			if err := fixture.db.First(&mastery).Error; err != nil {
				t.Fatalf("find mastery: %v", err)
			}
			if mastery.IncorrectCount != 1 {
				t.Fatalf("expected incorrect count 1, got %d", mastery.IncorrectCount)
			}
		})
	}
}
