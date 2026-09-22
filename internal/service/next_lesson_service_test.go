package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"learnos/internal/ai"
	"learnos/internal/model"
	"learnos/internal/repository"

	"sync"
	"sync/atomic"
)

type blockingBlueprintExpansionProvider struct {
	base    *ai.MockProvider
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
	once    sync.Once
	err     error
}

func (p *blockingBlueprintExpansionProvider) ExpandBlueprintUnit(ctx context.Context, req ai.BlueprintUnitExpansionRequest) (ai.BlueprintUnitExpansionResult, ai.ProviderMeta, error) {
	p.calls.Add(1)
	p.once.Do(func() { close(p.started) })
	if p.release != nil {
		select {
		case <-p.release:
		case <-ctx.Done():
			return ai.BlueprintUnitExpansionResult{}, ai.ProviderMeta{}, ctx.Err()
		}
	}
	if p.err != nil {
		return ai.BlueprintUnitExpansionResult{}, ai.ProviderMeta{}, p.err
	}
	return p.base.ExpandBlueprintUnit(ctx, req)
}

func TestNextLessonPrefersCurrentUnitAndReportsPrerequisiteState(t *testing.T) {
	_, db, course := newCurriculumFixture(t)
	courses := repository.NewCourseRepository(db)
	graphs := repository.NewKnowledgeGraphRepository(db)
	curriculum := repository.NewCurriculumRepository(db)
	cognitive := repository.NewCognitiveRepository(db)
	learning := repository.NewLearningRepository(db)
	service := NewNextLessonService(courses, graphs, curriculum, cognitive, learning)

	view, err := service.Recommend(context.Background(), course.ID)
	if err != nil {
		t.Fatalf("recommend next lesson: %v", err)
	}
	if view.Recommended == nil || view.Recommended.UnitTitle != course.CurrentUnit {
		t.Fatalf("expected current-unit recommendation: %+v", view)
	}
	if view.Recommended.PrerequisitesSatisfied {
		t.Fatal("expected prerequisite to be unsatisfied before evidence")
	}

	var lessons []model.Lesson
	if err := db.Where("course_id = ?", course.ID).Find(&lessons).Error; err != nil {
		t.Fatalf("list lessons: %v", err)
	}
	for _, lesson := range lessons {
		if err := db.Create(&model.CognitiveState{CourseID: course.ID, LessonID: lesson.ID, CurrentLevel: model.CognitiveLevelUnderstand, Status: model.CognitiveStatusStable}).Error; err != nil {
			t.Fatalf("create prerequisite evidence: %v", err)
		}
	}
	view, err = service.Recommend(context.Background(), course.ID)
	if err != nil {
		t.Fatalf("recommend after evidence: %v", err)
	}
	if view.Recommended == nil || !view.Recommended.PrerequisitesSatisfied {
		t.Fatalf("expected satisfied prerequisite: %+v", view)
	}
}

func TestNextLessonDoesNotRecommendPreviousCurrentUnitNodes(t *testing.T) {
	_, db, course := newCurriculumFixture(t)
	var lastLesson model.Lesson
	if err := db.Where("course_id = ? AND unit_id = ?", course.ID, *course.CurrentUnitID).Order("sort_order DESC, id DESC").First(&lastLesson).Error; err != nil {
		t.Fatalf("find last current-unit lesson: %v", err)
	}
	if err := db.Model(&model.Course{}).Where("id = ?", course.ID).Updates(map[string]interface{}{"current_lesson_id": lastLesson.ID}).Error; err != nil {
		t.Fatalf("move current lesson: %v", err)
	}
	service := NewNextLessonService(repository.NewCourseRepository(db), repository.NewKnowledgeGraphRepository(db), repository.NewCurriculumRepository(db), repository.NewCognitiveRepository(db), repository.NewLearningRepository(db))
	view, err := service.Recommend(context.Background(), course.ID)
	if err != nil {
		t.Fatalf("recommend from exhausted unit: %v", err)
	}
	if view.Recommended != nil && view.Recommended.UnitID == lastLesson.UnitID && view.Recommended.ID != lastLesson.ID {
		t.Fatalf("recommended a previous node in exhausted unit: %+v", view.Recommended)
	}
}

func TestBlueprintUnitExpansionPersistsBlueprintOnlyAndIsIdempotent(t *testing.T) {
	service, db, course := newCurriculumFixture(t)
	service.SetDomainInitializationProvider(ai.NewMockProvider())
	var unit model.CurriculumBlueprintUnit
	if err := db.Order("sort_order ASC").First(&unit).Error; err != nil {
		t.Fatalf("find blueprint unit: %v", err)
	}
	if err := db.Model(&model.CurriculumBlueprintUnit{}).Where("id = ?", unit.ID).Update("expansion_status", model.CurriculumUnitUnexpanded).Error; err != nil {
		t.Fatalf("mark unit unexpanded: %v", err)
	}
	var formalBefore, blueprintBefore int64
	db.Model(&model.Lesson{}).Where("course_id = ?", course.ID).Count(&formalBefore)
	db.Model(&model.CurriculumBlueprintLesson{}).Where("blueprint_unit_id = ?", unit.ID).Count(&blueprintBefore)
	if _, err := service.ExpandBlueprintUnit(context.Background(), course.ID, unit.ID); err != nil {
		t.Fatalf("expand blueprint unit: %v", err)
	}
	var stored model.CurriculumBlueprintUnit
	if err := db.First(&stored, unit.ID).Error; err != nil || stored.ExpansionStatus != model.CurriculumUnitExpanded {
		t.Fatalf("unit was not marked expanded: %+v err=%v", stored, err)
	}
	var blueprintAfter, formalAfter int64
	db.Model(&model.CurriculumBlueprintLesson{}).Where("blueprint_unit_id = ?", unit.ID).Count(&blueprintAfter)
	db.Model(&model.Lesson{}).Where("course_id = ?", course.ID).Count(&formalAfter)
	if blueprintAfter-blueprintBefore < 5 || blueprintAfter-blueprintBefore > 10 || formalBefore != formalAfter {
		t.Fatalf("unexpected expansion counts: blueprint %d/%d formal %d/%d", blueprintBefore, blueprintAfter, formalBefore, formalAfter)
	}
	if _, err := service.ExpandBlueprintUnit(context.Background(), course.ID, unit.ID); err != nil {
		t.Fatalf("idempotent expansion failed: %v", err)
	}
	var blueprintRepeat int64
	db.Model(&model.CurriculumBlueprintLesson{}).Where("blueprint_unit_id = ?", unit.ID).Count(&blueprintRepeat)
	if blueprintRepeat != blueprintAfter {
		t.Fatalf("idempotent expansion duplicated lessons: %d/%d", blueprintAfter, blueprintRepeat)
	}
}

func TestBlueprintUnitExpansionConcurrentRequestsCreateOneBatch(t *testing.T) {
	service, db, course := newCurriculumFixture(t)
	provider := &blockingBlueprintExpansionProvider{base: ai.NewMockProvider(), started: make(chan struct{}), release: make(chan struct{})}
	service.SetDomainInitializationProvider(provider)
	var unit model.CurriculumBlueprintUnit
	if err := db.Order("sort_order ASC").First(&unit).Error; err != nil {
		t.Fatalf("find blueprint unit: %v", err)
	}
	if err := db.Model(&model.CurriculumBlueprintUnit{}).Where("id = ?", unit.ID).Update("expansion_status", model.CurriculumUnitUnexpanded).Error; err != nil {
		t.Fatalf("mark unit unexpanded: %v", err)
	}
	var countBefore int64
	if err := db.Model(&model.CurriculumBlueprintLesson{}).Where("blueprint_unit_id = ?", unit.ID).Count(&countBefore).Error; err != nil {
		t.Fatalf("count lessons before expansion: %v", err)
	}
	firstDone := make(chan error, 1)
	go func() {
		_, err := service.ExpandBlueprintUnit(context.Background(), course.ID, unit.ID)
		firstDone <- err
	}()
	select {
	case <-provider.started:
	case <-time.After(2 * time.Second):
		t.Fatal("first expansion did not reach provider")
	}
	secondDone := make(chan error, 1)
	go func() {
		_, err := service.ExpandBlueprintUnit(context.Background(), course.ID, unit.ID)
		secondDone <- err
	}()
	select {
	case err := <-secondDone:
		if !errors.Is(err, ErrCurriculumUnitExpansionInProgress) {
			t.Fatalf("second expansion error = %v, want in-progress", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second expansion did not return while first request was running")
	}
	close(provider.release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first expansion failed: %v", err)
	}
	if calls := provider.calls.Load(); calls != 1 {
		t.Fatalf("provider calls = %d, want 1", calls)
	}
	var count int64
	if err := db.Model(&model.CurriculumBlueprintLesson{}).Where("blueprint_unit_id = ?", unit.ID).Count(&count).Error; err != nil {
		t.Fatalf("count expanded lessons: %v", err)
	}
	if count != countBefore+7 {
		t.Fatalf("expanded lesson count = %d, want %d existing + 7 new", count, countBefore)
	}
}

func TestBlueprintUnitExpansionTimeoutLeavesUnitUnexpanded(t *testing.T) {
	service, db, course := newCurriculumFixture(t)
	service.SetDomainInitializationProvider(&blockingBlueprintExpansionProvider{base: ai.NewMockProvider(), err: ai.ErrTimeout, started: make(chan struct{})})
	var unit model.CurriculumBlueprintUnit
	if err := db.Order("sort_order ASC").First(&unit).Error; err != nil {
		t.Fatalf("find blueprint unit: %v", err)
	}
	if err := db.Model(&model.CurriculumBlueprintUnit{}).Where("id = ?", unit.ID).Update("expansion_status", model.CurriculumUnitUnexpanded).Error; err != nil {
		t.Fatalf("mark unit unexpanded: %v", err)
	}
	var countBefore int64
	if err := db.Model(&model.CurriculumBlueprintLesson{}).Where("blueprint_unit_id = ?", unit.ID).Count(&countBefore).Error; err != nil {
		t.Fatalf("count lessons before expansion: %v", err)
	}
	if _, err := service.ExpandBlueprintUnit(context.Background(), course.ID, unit.ID); !errors.Is(err, ai.ErrTimeout) {
		t.Fatalf("expansion error = %v, want timeout", err)
	}
	var stored model.CurriculumBlueprintUnit
	if err := db.First(&stored, unit.ID).Error; err != nil {
		t.Fatalf("reload unit: %v", err)
	}
	if stored.ExpansionStatus != model.CurriculumUnitUnexpanded {
		t.Fatalf("unit status = %q, want unexpanded", stored.ExpansionStatus)
	}
	var count int64
	if err := db.Model(&model.CurriculumBlueprintLesson{}).Where("blueprint_unit_id = ?", unit.ID).Count(&count).Error; err != nil {
		t.Fatalf("count lessons: %v", err)
	}
	if count != countBefore {
		t.Fatalf("timeout changed blueprint lesson count from %d to %d", countBefore, count)
	}
}
