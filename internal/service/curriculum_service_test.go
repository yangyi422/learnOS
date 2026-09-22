package service

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"

	"learnos/internal/ai"
	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type countingCurriculumProvider struct {
	base  *ai.MockProvider
	calls atomic.Int32
}

type retryingCurriculumProvider struct {
	base  *ai.MockProvider
	calls atomic.Int32
}

func (p *retryingCurriculumProvider) GenerateCurriculumDraft(ctx context.Context, req ai.CurriculumDraftRequest) (ai.CurriculumDraftResult, ai.ProviderMeta, error) {
	if p.calls.Add(1) == 1 {
		return ai.CurriculumDraftResult{}, ai.ProviderMeta{Provider: "test", Model: "timeout", PromptVersion: "test"}, ai.ErrTimeout
	}
	return p.base.GenerateCurriculumDraft(ctx, req)
}

type invalidCurriculumProvider struct{}

func (invalidCurriculumProvider) GenerateCurriculumDraft(context.Context, ai.CurriculumDraftRequest) (ai.CurriculumDraftResult, ai.ProviderMeta, error) {
	return ai.CurriculumDraftResult{ChangeSetJSON: `{not-json`}, ai.ProviderMeta{Provider: "test", Model: "invalid", PromptVersion: "test"}, nil
}

func (p *countingCurriculumProvider) GenerateCurriculumDraft(ctx context.Context, req ai.CurriculumDraftRequest) (ai.CurriculumDraftResult, ai.ProviderMeta, error) {
	p.calls.Add(1)
	return p.base.GenerateCurriculumDraft(ctx, req)
}

func newCurriculumFixture(t *testing.T) (*CurriculumService, *gorm.DB, model.Course) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "curriculum.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Course{}, &model.CourseUnit{}, &model.Lesson{}, &model.LessonRelation{}, &model.LearningTurn{},
		&model.MasteryRecord{}, &model.Misconception{}, &model.AIEvaluationRun{}, &model.CognitiveState{}, &model.CognitiveEvidence{}, &model.CognitiveStateEvent{},
		&model.AssessmentChallenge{}, &model.ChallengeAttempt{}, &model.MisconceptionEvent{}, &model.MisconceptionPatternLink{},
		&model.CurriculumBlueprint{}, &model.CurriculumBlueprintUnit{}, &model.CurriculumBlueprintLesson{}, &model.CurriculumBlueprintRelation{}, &model.CurriculumDraft{},
		&model.KnowledgeSource{}, &model.SourceEvidence{}, &model.GroundingLink{}, &model.SourceCredibilityAssessment{}, &model.GroundingReviewEvent{},
	); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	courses := repository.NewCourseRepository(db)
	if err := courses.SeedStarterCourse(context.Background()); err != nil {
		t.Fatalf("seed course: %v", err)
	}
	var course model.Course
	if err := db.Where("name = ?", "营养学").First(&course).Error; err != nil {
		t.Fatalf("find course: %v", err)
	}
	return NewCurriculumService(courses, repository.NewCurriculumRepository(db)), db, course
}

func TestCurriculumCoverageUsesBlueprintOnly(t *testing.T) {
	service, _, course := newCurriculumFixture(t)
	coverage, err := service.GetCoverage(context.Background(), course.ID)
	if err != nil {
		t.Fatalf("get coverage: %v", err)
	}
	if coverage.Metrics.CoreTotal == 0 || coverage.Metrics.CoreMissing == 0 {
		t.Fatalf("expected provisional blueprint with missing core lessons: %+v", coverage.Metrics)
	}
	for _, unit := range coverage.Units {
		for _, lesson := range unit.Lessons {
			if lesson.State == "covered" && lesson.AppliedLesson == nil {
				t.Fatal("covered blueprint lesson must have an applied lesson")
			}
		}
	}
}

func TestCurriculumDraftApplyIsTransactionalAndPreservesMainline(t *testing.T) {
	service, db, course := newCurriculumFixture(t)
	beforeCurrent := *course.CurrentLessonID
	var beforeLessons int64
	if err := db.Model(&model.Lesson{}).Where("course_id = ?", course.ID).Count(&beforeLessons).Error; err != nil {
		t.Fatalf("count lessons: %v", err)
	}
	draft, err := service.GenerateDraft(context.Background(), course.ID, "missing_core", 1)
	if err != nil {
		t.Fatalf("generate draft: %v", err)
	}
	if draft.Draft.Status != model.CurriculumDraftStatusDraft || len(draft.ChangeSet.NewLessons) != 1 {
		t.Fatalf("unexpected draft: %+v", draft)
	}
	if _, err := service.ApplyDraft(context.Background(), course.ID, draft.Draft.ID); err != nil {
		t.Fatalf("apply draft: %v", err)
	}
	var after model.Course
	if err := db.First(&after, course.ID).Error; err != nil {
		t.Fatalf("reload course: %v", err)
	}
	if after.CurrentLessonID == nil || *after.CurrentLessonID != beforeCurrent {
		t.Fatalf("apply changed current lesson: before=%d after=%v", beforeCurrent, after.CurrentLessonID)
	}
	var afterLessons int64
	if err := db.Model(&model.Lesson{}).Where("course_id = ?", course.ID).Count(&afterLessons).Error; err != nil {
		t.Fatalf("count lessons after apply: %v", err)
	}
	if afterLessons != beforeLessons+1 {
		t.Fatalf("expected one new lesson, before=%d after=%d", beforeLessons, afterLessons)
	}
	if _, err := service.ApplyDraft(context.Background(), course.ID, draft.Draft.ID); !errors.Is(err, ErrCurriculumDraftApplied) {
		t.Fatalf("expected second apply to be rejected, got %v", err)
	}
}

func TestCurriculumDraftUsesProviderButDoesNotApply(t *testing.T) {
	service, db, course := newCurriculumFixture(t)
	service.SetCurriculumDraftProvider(ai.NewMockProvider())
	var before int64
	if err := db.Model(&model.Lesson{}).Where("course_id = ?", course.ID).Count(&before).Error; err != nil {
		t.Fatalf("count lessons: %v", err)
	}
	draft, err := service.GenerateDraft(context.Background(), course.ID, "missing_core", 1)
	if err != nil {
		t.Fatalf("generate provider draft: %v", err)
	}
	var after int64
	if err := db.Model(&model.Lesson{}).Where("course_id = ?", course.ID).Count(&after).Error; err != nil {
		t.Fatalf("count lessons after draft: %v", err)
	}
	if before != after || draft.Draft.GeneratedBy != model.CurriculumGeneratedByAI {
		t.Fatalf("provider draft changed formal course or metadata: before=%d after=%d draft=%+v", before, after, draft.Draft)
	}
}

func TestCurriculumDraftForUnitCanContinueWithRecommendedLessons(t *testing.T) {
	service, db, course := newCurriculumFixture(t)
	var blueprint model.CurriculumBlueprint
	if err := db.Where("course_id = ? AND status = ?", course.ID, model.CurriculumBlueprintStatusActive).First(&blueprint).Error; err != nil {
		t.Fatalf("find blueprint: %v", err)
	}
	var unit model.CurriculumBlueprintUnit
	if err := db.Where("blueprint_id = ? AND key = ?", blueprint.ID, "nutrition.foundation").First(&unit).Error; err != nil {
		t.Fatalf("find blueprint unit: %v", err)
	}

	first, err := service.GenerateDraftForUnit(context.Background(), course.ID, unit.ID, "missing_core", 5)
	if err != nil {
		t.Fatalf("generate first unit draft: %v", err)
	}
	if _, err := service.ApplyDraft(context.Background(), course.ID, first.Draft.ID); err != nil {
		t.Fatalf("apply first unit draft: %v", err)
	}

	continued, err := service.GenerateDraftForUnit(context.Background(), course.ID, unit.ID, "missing_recommended", 5)
	if err != nil {
		t.Fatalf("generate continued unit draft: %v", err)
	}
	if len(continued.ChangeSet.NewLessons) != 1 || continued.ChangeSet.NewLessons[0].BlueprintLessonKey != "nutrition.foundation.digestion_absorption" {
		t.Fatalf("expected the remaining recommended lesson, got %+v", continued.ChangeSet.NewLessons)
	}
}

func TestCurriculumDraftForUnitIsIdempotentWhilePending(t *testing.T) {
	service, db, course := newCurriculumFixture(t)
	provider := &countingCurriculumProvider{base: ai.NewMockProvider()}
	service.SetCurriculumDraftProvider(provider)
	var blueprint model.CurriculumBlueprint
	if err := db.Where("course_id = ? AND status = ?", course.ID, model.CurriculumBlueprintStatusActive).First(&blueprint).Error; err != nil {
		t.Fatalf("find blueprint: %v", err)
	}
	var unit model.CurriculumBlueprintUnit
	if err := db.Where("blueprint_id = ? AND key = ?", blueprint.ID, "nutrition.foundation").First(&unit).Error; err != nil {
		t.Fatalf("find blueprint unit: %v", err)
	}
	first, err := service.GenerateDraftForUnit(context.Background(), course.ID, unit.ID, "missing_core", 5)
	if err != nil {
		t.Fatalf("generate first draft: %v", err)
	}
	second, err := service.GenerateDraftForUnit(context.Background(), course.ID, unit.ID, "missing_core", 5)
	if err != nil {
		t.Fatalf("generate second draft: %v", err)
	}
	if first.Draft.ID != second.Draft.ID {
		t.Fatalf("pending draft IDs differ: %d/%d", first.Draft.ID, second.Draft.ID)
	}
	if calls := provider.calls.Load(); calls != 1 {
		t.Fatalf("provider calls = %d, want 1", calls)
	}
	var count int64
	if err := db.Model(&model.CurriculumDraft{}).Where("course_id = ? AND blueprint_unit_id = ? AND status = ?", course.ID, unit.ID, model.CurriculumDraftStatusDraft).Count(&count).Error; err != nil {
		t.Fatalf("count pending drafts: %v", err)
	}
	if count != 1 {
		t.Fatalf("pending draft count = %d, want 1", count)
	}
}

func TestCurriculumDraftTimeoutWritesNoPartialDataAndRetrySucceeds(t *testing.T) {
	service, db, course := newCurriculumFixture(t)
	provider := &retryingCurriculumProvider{base: ai.NewMockProvider()}
	service.SetCurriculumDraftProvider(provider)
	var lessonsBefore int64
	if err := db.Model(&model.Lesson{}).Where("course_id = ?", course.ID).Count(&lessonsBefore).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := service.GenerateDraft(context.Background(), course.ID, "missing_core", 1); !errors.Is(err, ai.ErrTimeout) {
		t.Fatalf("expected timeout, got %v", err)
	}
	var failed model.CurriculumDraft
	if err := db.Where("course_id = ?", course.ID).Order("id DESC").First(&failed).Error; err != nil {
		t.Fatal(err)
	}
	if failed.Status != model.CurriculumDraftStatusRejected {
		t.Fatalf("failed placeholder status = %s, want rejected", failed.Status)
	}
	var lessonsAfterFailure int64
	if err := db.Model(&model.Lesson{}).Where("course_id = ?", course.ID).Count(&lessonsAfterFailure).Error; err != nil {
		t.Fatal(err)
	}
	if lessonsAfterFailure != lessonsBefore {
		t.Fatalf("timeout wrote formal lessons: before=%d after=%d", lessonsBefore, lessonsAfterFailure)
	}

	retried, err := service.GenerateDraft(context.Background(), course.ID, "missing_core", 1)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if retried.Draft.Status != model.CurriculumDraftStatusDraft || retried.Draft.ID == failed.ID {
		t.Fatalf("retry did not create a completed independent draft: %+v", retried.Draft)
	}
	if provider.calls.Load() != 2 {
		t.Fatalf("provider calls = %d, want 2", provider.calls.Load())
	}
}

func TestCurriculumDraftParseFailureRejectsPlaceholderAndWritesNoLesson(t *testing.T) {
	service, db, course := newCurriculumFixture(t)
	service.SetCurriculumDraftProvider(invalidCurriculumProvider{})
	var lessonsBefore int64
	if err := db.Model(&model.Lesson{}).Where("course_id = ?", course.ID).Count(&lessonsBefore).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.GenerateDraft(context.Background(), course.ID, "missing_core", 1); err == nil {
		t.Fatal("expected provider payload validation failure")
	}
	var failed model.CurriculumDraft
	if err := db.Where("course_id = ?", course.ID).Order("id DESC").First(&failed).Error; err != nil {
		t.Fatal(err)
	}
	if failed.Status != model.CurriculumDraftStatusRejected {
		t.Fatalf("status = %s", failed.Status)
	}
	var lessonsAfter int64
	if err := db.Model(&model.Lesson{}).Where("course_id = ?", course.ID).Count(&lessonsAfter).Error; err != nil {
		t.Fatal(err)
	}
	if lessonsAfter != lessonsBefore {
		t.Fatalf("invalid AI payload wrote formal lessons: before=%d after=%d", lessonsBefore, lessonsAfter)
	}
}
