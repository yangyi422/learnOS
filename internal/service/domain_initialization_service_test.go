package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"learnos/internal/ai"
	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDomainInitializationFixture(t *testing.T, provider ai.DomainInitializationProvider) (*gorm.DB, *DomainInitializationService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "domain.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Course{}, &model.CourseUnit{}, &model.Lesson{}, &model.LessonRelation{}, &model.CurriculumBlueprint{}, &model.CurriculumBlueprintUnit{}, &model.CurriculumBlueprintLesson{}, &model.CurriculumBlueprintRelation{}, &model.DomainInitializationDraft{}, &model.LearningTurn{}, &model.CognitiveState{}, &model.CognitiveEvidence{}, &model.CognitiveStateEvent{}, &model.Misconception{}, &model.AssessmentChallenge{}, &model.ChallengeAttempt{}, &model.ExplorationDirection{}, &model.ExplorationQuestion{}); err != nil {
		t.Fatal(err)
	}
	service := NewDomainInitializationService(repository.NewDomainInitializationRepository(db))
	service.SetProvider(provider)
	return db, service
}

func TestDomainInitializationRejectsBlankInputWithoutPersistingData(t *testing.T) {
	db, service := newDomainInitializationFixture(t, ai.NewMockProvider())

	_, err := service.CreateSkeleton(context.Background(), CreateDomainRequest{
		DomainName:   " \t\n ",
		LearningGoal: "  ",
		TargetDepth:  "systematic",
	})
	if !errors.Is(err, ErrDomainInputInvalid) {
		t.Fatalf("blank input error = %v, want ErrDomainInputInvalid", err)
	}

	for name, target := range map[string]interface{}{
		"domain drafts": &model.DomainInitializationDraft{},
		"courses":       &model.Course{},
		"blueprints":    &model.CurriculumBlueprint{},
		"lessons":       &model.Lesson{},
	} {
		var count int64
		if err := db.Model(target).Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", name, err)
		}
		if count != 0 {
			t.Fatalf("blank input persisted %d %s", count, name)
		}
	}
}

func TestDomainInitializationValidatesInputBoundaries(t *testing.T) {
	_, service := newDomainInitializationFixture(t, ai.NewMockProvider())
	tests := []CreateDomainRequest{
		{DomainName: "学", LearningGoal: "建立完整而可靠的基础理解", TargetDepth: "systematic"},
		{DomainName: strings.Repeat("学", 51), LearningGoal: "建立完整而可靠的基础理解", TargetDepth: "systematic"},
		{DomainName: "测试领域", LearningGoal: "不足十字", TargetDepth: "systematic"},
		{DomainName: "测试领域", LearningGoal: "建立完整而可靠的基础理解", TargetDepth: "invalid"},
	}
	for _, request := range tests {
		if _, err := service.CreateSkeleton(context.Background(), request); !errors.Is(err, ErrDomainInputInvalid) {
			t.Fatalf("request %+v error = %v, want ErrDomainInputInvalid", request, err)
		}
	}
}

func TestDomainInitializationStagesPersistBeforeApplyAndKeepPersonalWorldEmpty(t *testing.T) {
	db, service := newDomainInitializationFixture(t, ai.NewMockProvider())
	ctx := context.Background()
	view, err := service.CreateSkeleton(ctx, CreateDomainRequest{DomainName: "测试领域", LearningGoal: "建立可用且完整的基础理解", TargetDepth: "systematic"})
	if err != nil {
		t.Fatal(err)
	}
	var courses int64
	db.Model(&model.Course{}).Count(&courses)
	if courses != 0 || len(view.Skeleton.Blueprint.Units) != 6 {
		t.Fatalf("skeleton should not create course: courses=%d view=%+v", courses, view)
	}
	view, err = service.ExpandStarter(ctx, view.Draft.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if view.Starter == nil || len(view.Starter.ExpandedUnits) != 1 {
		t.Fatalf("starter was not persisted: %+v", view)
	}
	view, err = service.GenerateInitialWorld(ctx, view.Draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.InitialWorld == nil || len(view.InitialWorld.InitialLessons) != 3 {
		t.Fatalf("initial world was not persisted: %+v", view)
	}
	var units, blueprints, lessons int64
	db.Model(&model.CourseUnit{}).Count(&units)
	db.Model(&model.CurriculumBlueprint{}).Count(&blueprints)
	db.Model(&model.Lesson{}).Count(&lessons)
	if blueprints != 0 || lessons != 0 {
		t.Fatalf("apply-only records were created too early: blueprints=%d lessons=%d", blueprints, lessons)
	}
	view, err = service.Apply(ctx, view.Draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Draft.Status != model.DomainInitializationStatusApplied || view.Draft.AppliedCourseID == nil {
		t.Fatalf("apply did not finish: %+v", view.Draft)
	}
	db.Model(&model.Course{}).Count(&courses)
	db.Model(&model.CourseUnit{}).Count(&units)
	db.Model(&model.CurriculumBlueprint{}).Count(&blueprints)
	db.Model(&model.Lesson{}).Count(&lessons)
	if courses != 1 || units != 6 || blueprints != 1 || lessons != 3 {
		t.Fatalf("unexpected applied counts: courses=%d units=%d blueprints=%d lessons=%d", courses, units, blueprints, lessons)
	}
	var course model.Course
	if err := db.First(&course).Error; err != nil {
		t.Fatal(err)
	}
	if course.CurrentUnitID == nil || course.CurrentLessonID == nil {
		t.Fatalf("applied course has no current focus: %+v", course)
	}
	var currentUnit model.CourseUnit
	if err := db.First(&currentUnit, *course.CurrentUnitID).Error; err != nil {
		t.Fatalf("current unit is not a formal CourseUnit: %v", err)
	}
	if currentUnit.CourseID != course.ID {
		t.Fatalf("current unit belongs to another course: %+v", currentUnit)
	}
	var currentLesson model.Lesson
	if err := db.First(&currentLesson, *course.CurrentLessonID).Error; err != nil {
		t.Fatal(err)
	}
	if currentLesson.CourseID != course.ID || currentLesson.UnitID != currentUnit.ID {
		t.Fatalf("current lesson is not attached to current CourseUnit: lesson=%+v unit=%+v", currentLesson, currentUnit)
	}
	var turns, states, evidence, misconceptions, challenges, attempts, directions, questions int64
	db.Model(&model.LearningTurn{}).Count(&turns)
	db.Model(&model.CognitiveState{}).Count(&states)
	db.Model(&model.CognitiveEvidence{}).Count(&evidence)
	db.Model(&model.Misconception{}).Count(&misconceptions)
	db.Model(&model.AssessmentChallenge{}).Count(&challenges)
	db.Model(&model.ChallengeAttempt{}).Count(&attempts)
	db.Model(&model.ExplorationDirection{}).Count(&directions)
	db.Model(&model.ExplorationQuestion{}).Count(&questions)
	if turns != 0 || states != 0 || evidence != 0 || misconceptions != 0 || challenges != 0 || attempts != 0 || directions != 0 || questions != 0 {
		t.Fatalf("apply polluted personal world: turns=%d states=%d evidence=%d misconceptions=%d challenges=%d attempts=%d directions=%d questions=%d", turns, states, evidence, misconceptions, challenges, attempts, directions, questions)
	}
	if _, err := service.Apply(ctx, view.Draft.ID); err != nil {
		t.Fatalf("repeated apply should be idempotent: %v", err)
	}
}

type failingInitialWorldProvider struct{ *ai.MockProvider }

func (p failingInitialWorldProvider) GenerateInitialWorld(context.Context, ai.InitialWorldRequest) (ai.InitialWorldResult, ai.ProviderMeta, error) {
	return ai.InitialWorldResult{}, ai.ProviderMeta{}, ai.ErrTimeout
}

func TestDomainInitializationStageThreeFailureKeepsStageOneAndTwo(t *testing.T) {
	db, service := newDomainInitializationFixture(t, failingInitialWorldProvider{ai.NewMockProvider()})
	view, err := service.CreateSkeleton(context.Background(), CreateDomainRequest{DomainName: "失败保留测试", LearningGoal: "测试阶段失败时的持久化行为", TargetDepth: "foundation"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.ExpandStarter(context.Background(), view.Draft.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GenerateInitialWorld(context.Background(), view.Draft.ID); !errors.Is(err, ai.ErrTimeout) {
		t.Fatalf("expected timeout, got %v", err)
	}
	persisted, err := service.Get(context.Background(), view.Draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Draft.Status != model.DomainInitializationStatusStarterExpanded || persisted.Skeleton == nil || persisted.Starter == nil || persisted.InitialWorld != nil {
		t.Fatalf("stage data was not preserved correctly: %+v", persisted)
	}
	var count int64
	db.Model(&model.Course{}).Count(&count)
	if count != 0 {
		t.Fatalf("failed stage must not create course: %d", count)
	}
}

func applyTestDomain(t *testing.T, service *DomainInitializationService, name string) model.Course {
	t.Helper()
	ctx := context.Background()
	view, err := service.CreateSkeleton(ctx, CreateDomainRequest{DomainName: name, LearningGoal: "建立独立且完整的长期学习路径", TargetDepth: "systematic"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.ExpandStarter(ctx, view.Draft.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.GenerateInitialWorld(ctx, view.Draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.Apply(ctx, view.Draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	course, err := service.drafts.FindCourseByName(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	if view.Draft.AppliedCourseID == nil || course.ID != *view.Draft.AppliedCourseID {
		t.Fatalf("applied course mismatch: course=%+v draft=%+v", course, view.Draft)
	}
	return *course
}

func TestDomainInitializationSupportsMultipleIndependentCourses(t *testing.T) {
	db, service := newDomainInitializationFixture(t, ai.NewMockProvider())
	first := applyTestDomain(t, service, "Golang")

	if _, err := service.CreateSkeleton(context.Background(), CreateDomainRequest{DomainName: " goLANG ", LearningGoal: "验证规范化后的重复领域名称", TargetDepth: "foundation"}); !errors.Is(err, ErrDomainAlreadyExists) {
		t.Fatalf("case-insensitive duplicate should be rejected, got %v", err)
	}

	second := applyTestDomain(t, service, "营养学")
	if first.ID == second.ID {
		t.Fatalf("independent domains reused course id: first=%d second=%d", first.ID, second.ID)
	}

	var courseCount, blueprintCount, lessonCount int64
	db.Model(&model.Course{}).Count(&courseCount)
	db.Model(&model.CurriculumBlueprint{}).Count(&blueprintCount)
	db.Model(&model.Lesson{}).Count(&lessonCount)
	if courseCount != 2 || blueprintCount != 2 || lessonCount != 6 {
		t.Fatalf("unexpected multi-course counts: courses=%d blueprints=%d lessons=%d", courseCount, blueprintCount, lessonCount)
	}

	for _, course := range []model.Course{first, second} {
		if course.CurrentUnitID == nil || course.CurrentLessonID == nil {
			t.Fatalf("course has no independent current lesson: %+v", course)
		}
		var unit model.CourseUnit
		if err := db.First(&unit, *course.CurrentUnitID).Error; err != nil {
			t.Fatal(err)
		}
		var lesson model.Lesson
		if err := db.First(&lesson, *course.CurrentLessonID).Error; err != nil {
			t.Fatal(err)
		}
		if unit.CourseID != course.ID || lesson.CourseID != course.ID || lesson.UnitID != unit.ID {
			t.Fatalf("current lesson crossed course boundary: course=%d unit=%+v lesson=%+v", course.ID, unit, lesson)
		}
	}

	view, err := service.CreateSkeleton(context.Background(), CreateDomainRequest{DomainName: "心理学", LearningGoal: "阶段失败不影响已有课程", TargetDepth: "overview"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ExpandStarter(context.Background(), view.Draft.ID, nil); err != nil {
		t.Fatal(err)
	}
	db.Model(&model.Course{}).Count(&courseCount)
	if courseCount != 2 {
		t.Fatalf("unapplied draft changed course count: %d", courseCount)
	}
	if _, err := service.GenerateInitialWorld(context.Background(), view.Draft.ID); err != nil {
		t.Fatal(err)
	}
	db.Model(&model.Course{}).Count(&courseCount)
	if courseCount != 2 {
		t.Fatalf("stage three draft changed course count: %d", courseCount)
	}
	if _, err := service.Apply(context.Background(), view.Draft.ID); err != nil {
		t.Fatal(err)
	}
	db.Model(&model.Course{}).Count(&courseCount)
	if courseCount != 3 {
		t.Fatalf("third independent course was not applied: %d", courseCount)
	}
}

func TestDomainInitializationApplyRejectsNormalizedDuplicate(t *testing.T) {
	db, service := newDomainInitializationFixture(t, ai.NewMockProvider())
	ctx := context.Background()
	view, err := service.CreateSkeleton(ctx, CreateDomainRequest{DomainName: "Golang", LearningGoal: "验证 Apply 重复保护", TargetDepth: "foundation"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.ExpandStarter(ctx, view.Draft.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.GenerateInitialWorld(ctx, view.Draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Course{Name: " golang ", Status: model.CourseStatusLearning}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.Apply(ctx, view.Draft.ID); !errors.Is(err, ErrDomainAlreadyExists) {
		t.Fatalf("apply should reject normalized duplicate, got %v", err)
	}
	var courseCount int64
	db.Model(&model.Course{}).Count(&courseCount)
	if courseCount != 1 {
		t.Fatalf("duplicate apply changed course count: %d", courseCount)
	}
}
