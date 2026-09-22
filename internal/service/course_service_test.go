package service

import (
	"context"
	"errors"
	"math"
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
		&model.CurriculumBlueprint{}, &model.CurriculumBlueprintUnit{}, &model.CurriculumBlueprintLesson{}, &model.CurriculumBlueprintRelation{}, &model.CurriculumDraft{},
		&model.KnowledgeSource{}, &model.SourceEvidence{}, &model.GroundingLink{}, &model.SourceCredibilityAssessment{}, &model.GroundingReviewEvent{},
		&model.ExplorationDirection{}, &model.ExplorationQuestion{}, &model.DomainInitializationDraft{},
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

func TestDeleteCourseRemovesCourseGraphAndKeepsSharedSources(t *testing.T) {
	fixture := newServiceTestFixture(t, newFakeProvider(validFakeResult()))
	var otherCourse model.Course
	if err := fixture.db.Where("name = ?", "逻辑与科学思维").First(&otherCourse).Error; err != nil {
		t.Fatalf("find other course: %v", err)
	}
	turn := model.LearningTurn{CourseID: fixture.course.ID, UnitID: *fixture.course.CurrentUnitID, LessonID: fixture.lesson.ID, Question: "问题", UserAnswer: "回答", Result: "mostly_correct"}
	mastery := model.MasteryRecord{CourseID: fixture.course.ID, LessonID: fixture.lesson.ID, MasteryScore: 0.5}
	source := model.KnowledgeSource{Title: "共享来源", SourceType: model.KnowledgeSourceTypeTextbook}
	evidence := model.SourceEvidence{SourceID: 1, EvidenceType: model.SourceEvidenceTypeSummary, Summary: "摘要"}
	if err := fixture.db.Create(&turn).Error; err != nil {
		t.Fatalf("create learning turn: %v", err)
	}
	if err := fixture.db.Create(&mastery).Error; err != nil {
		t.Fatalf("create mastery record: %v", err)
	}
	if err := fixture.db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	evidence.SourceID = source.ID
	if err := fixture.db.Create(&evidence).Error; err != nil {
		t.Fatalf("create source evidence: %v", err)
	}
	if err := fixture.db.Create(&model.GroundingLink{EvidenceID: evidence.ID, TargetType: model.GroundingTargetLesson, TargetID: fixture.lesson.ID, Relation: model.GroundingRelationSupports, Strength: model.GroundingStrengthModerate, Status: model.GroundingLinkStatusReviewed, CreatedBy: model.GroundingCreatedByManual}).Error; err != nil {
		t.Fatalf("create grounding link: %v", err)
	}
	if err := fixture.db.Create(&model.ExplorationDirection{CourseID: fixture.course.ID, TargetCourseID: otherCourse.ID, TargetLessonID: 1, DirectionType: model.ExplorationDirectionCrossDomain, Title: "方向", WhyWorthExploring: "原因"}).Error; err != nil {
		t.Fatalf("create exploration direction: %v", err)
	}
	if err := fixture.db.Create(&model.DomainInitializationDraft{DomainName: fixture.course.Name, LearningGoal: fixture.course.Goal, TargetDepth: "systematic", Status: model.DomainInitializationStatusApplied, AppliedCourseID: &fixture.course.ID}).Error; err != nil {
		t.Fatalf("create domain draft: %v", err)
	}

	if err := fixture.service.Delete(context.Background(), fixture.course.ID); err != nil {
		t.Fatalf("delete course: %v", err)
	}
	if _, err := fixture.service.GetCurrentLesson(context.Background(), fixture.course.ID); !errors.Is(err, ErrCourseNotFound) {
		t.Fatalf("deleted course should be unavailable, got %v", err)
	}
	for _, item := range []struct {
		name  string
		model interface{}
	}{
		{"course units", &model.CourseUnit{}}, {"lessons", &model.Lesson{}}, {"lesson relations", &model.LessonRelation{}},
		{"learning turns", &model.LearningTurn{}}, {"mastery records", &model.MasteryRecord{}}, {"blueprints", &model.CurriculumBlueprint{}},
		{"grounding links", &model.GroundingLink{}}, {"exploration directions", &model.ExplorationDirection{}}, {"domain drafts", &model.DomainInitializationDraft{}},
	} {
		var count int64
		query := fixture.db.Model(item.model).Where("course_id = ?", fixture.course.ID)
		if item.name == "blueprints" {
			query = fixture.db.Model(item.model).Where("course_id = ?", fixture.course.ID)
		}
		if item.name == "grounding links" {
			query = fixture.db.Model(item.model).Where("target_type = ? AND target_id = ?", model.GroundingTargetLesson, fixture.lesson.ID)
		}
		if item.name == "domain drafts" {
			query = fixture.db.Model(item.model).Where("applied_course_id = ?", fixture.course.ID)
		}
		if item.name == "exploration directions" {
			query = fixture.db.Model(item.model).Where("course_id = ? OR target_course_id = ?", fixture.course.ID, fixture.course.ID)
		}
		if err := query.Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", item.name, err)
		}
		if count != 0 {
			t.Fatalf("deleted course left %s: %d", item.name, count)
		}
	}
	var sourceCount int64
	if err := fixture.db.Model(&model.KnowledgeSource{}).Count(&sourceCount).Error; err != nil || sourceCount != 1 {
		t.Fatalf("shared source was deleted: count=%d err=%v", sourceCount, err)
	}
	var remainingCourses int64
	if err := fixture.db.Model(&model.Course{}).Count(&remainingCourses).Error; err != nil || remainingCourses != 2 {
		t.Fatalf("other courses were affected: count=%d err=%v", remainingCourses, err)
	}
}

func TestSetCurrentLessonChangesOnlyMainlinePointer(t *testing.T) {
	fixture := newServiceTestFixture(t, newFakeProvider(validFakeResult()))
	var target model.Lesson
	if err := fixture.db.Where("course_id = ? AND id <> ?", fixture.course.ID, fixture.lesson.ID).Order("id ASC").First(&target).Error; err != nil {
		t.Fatalf("find target lesson: %v", err)
	}
	var turnsBefore, statesBefore int64
	fixture.db.Model(&model.LearningTurn{}).Where("course_id = ?", fixture.course.ID).Count(&turnsBefore)
	fixture.db.Model(&model.CognitiveState{}).Where("course_id = ?", fixture.course.ID).Count(&statesBefore)
	if _, err := fixture.service.SetCurrentLesson(context.Background(), fixture.course.ID, target.ID); err != nil {
		t.Fatalf("set current lesson: %v", err)
	}
	var course model.Course
	if err := fixture.db.First(&course, fixture.course.ID).Error; err != nil {
		t.Fatalf("reload course: %v", err)
	}
	if course.CurrentLessonID == nil || *course.CurrentLessonID != target.ID {
		t.Fatalf("current lesson was not switched: %v", course.CurrentLessonID)
	}
	var turnsAfter, statesAfter int64
	fixture.db.Model(&model.LearningTurn{}).Where("course_id = ?", fixture.course.ID).Count(&turnsAfter)
	fixture.db.Model(&model.CognitiveState{}).Where("course_id = ?", fixture.course.ID).Count(&statesAfter)
	if turnsBefore != turnsAfter || statesBefore != statesAfter {
		t.Fatalf("switch created learning data: turns %d/%d states %d/%d", turnsBefore, turnsAfter, statesBefore, statesAfter)
	}
}

func TestBuildCourseViewSeparatesGenerationCoverageAndMastery(t *testing.T) {
	course := model.Course{Status: model.CourseStatusLearning, Progress: 99}
	view := buildCourseView(course, repository.CourseProgressFacts{
		LessonCount: 10, BlueprintLessonCount: 8, GeneratedLessonCount: 4,
		CoveredLessonCount: 3, MasteryPointTotal: 240,
	})
	if view.GenerationStatus != "in_progress" || view.GenerationProgress != 50 {
		t.Fatalf("unexpected generation values: %+v", view)
	}
	if view.LearningStatus != "in_progress" || view.CoverageProgress != 30 {
		t.Fatalf("unexpected learning values: %+v", view)
	}
	if view.MasteryStatus != model.CognitiveStatusDeveloping || view.MasteryProgress != 24 {
		t.Fatalf("unexpected mastery values: %+v", view)
	}
	if view.Progress != 99 {
		t.Fatalf("legacy progress must remain compatibility-only, got %d", view.Progress)
	}
}

func TestListRepairsOnlyInitializingCourseWithValidCurrentLesson(t *testing.T) {
	fixture := newServiceTestFixture(t, newFakeProvider(validFakeResult()))
	valid := model.Course{Name: "valid initializing", Status: model.CourseStatusInitializing}
	if err := fixture.db.Create(&valid).Error; err != nil {
		t.Fatalf("create valid course: %v", err)
	}
	unit := model.CourseUnit{CourseID: valid.ID, Title: "unit", Status: model.CourseUnitStatusLearning}
	if err := fixture.db.Create(&unit).Error; err != nil {
		t.Fatalf("create unit: %v", err)
	}
	lesson := model.Lesson{CourseID: valid.ID, UnitID: unit.ID, Title: "lesson", CoreQuestion: "question", Status: model.LessonStatusLearning, ContentRole: model.ContentRoleCore, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand}
	if err := fixture.db.Create(&lesson).Error; err != nil {
		t.Fatalf("create lesson: %v", err)
	}
	if err := fixture.db.Model(&valid).Updates(map[string]interface{}{"current_unit_id": unit.ID, "current_lesson_id": lesson.ID}).Error; err != nil {
		t.Fatalf("set current pointers: %v", err)
	}
	invalid := model.Course{Name: "invalid initializing", Status: model.CourseStatusInitializing}
	if err := fixture.db.Create(&invalid).Error; err != nil {
		t.Fatalf("create invalid course: %v", err)
	}
	if _, err := fixture.service.List(context.Background()); err != nil {
		t.Fatalf("list courses: %v", err)
	}
	if err := fixture.db.First(&valid, valid.ID).Error; err != nil || valid.Status != model.CourseStatusLearning {
		t.Fatalf("valid course was not repaired: status=%s err=%v", valid.Status, err)
	}
	if err := fixture.db.First(&invalid, invalid.ID).Error; err != nil || invalid.Status != model.CourseStatusInitializing {
		t.Fatalf("course without content must remain initializing: status=%s err=%v", invalid.Status, err)
	}
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
		BoundaryConditions:        []string{"日常低强度环境仍可参考口渴"},
		MasteryEvidence:           []string{"识别了单一信号的局限"},
		EvidenceUsed:              []string{"回答指出不能只依赖口渴"},
		Confidence:                0.84,
		Uncertainty:               "尚未验证陌生场景迁移",
		RecommendedNextAction:     "修正缺口后完成迁移挑战",
		TransferChallengeEligible: true,
		MasteryScore:              0.78,
		NeedsReview:               true,
		DemonstratedLevel:         "understand",
		UserUnderstandingSummary:  "用户能解释口渴不是唯一依据，并能指出环境和身体状态会影响判断。",
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

func TestSubmitAnswerIsIdempotentAndKeepsEvaluationHistory(t *testing.T) {
	provider := newFakeProvider(validFakeResult())
	fixture := newServiceTestFixture(t, provider)
	ctx := context.Background()
	answer := "不能只看口渴，还要结合环境和身体状态。"

	first, err := fixture.service.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, answer, "answer-request-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := fixture.service.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, answer, "answer-request-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.TurnID != second.TurnID || provider.calls != 1 {
		t.Fatalf("duplicate submission was evaluated twice: first=%d second=%d calls=%d", first.TurnID, second.TurnID, provider.calls)
	}
	var turns, evidence int64
	fixture.db.Model(&model.LearningTurn{}).Count(&turns)
	fixture.db.Model(&model.CognitiveEvidence{}).Count(&evidence)
	if turns != 1 || evidence != 2 {
		t.Fatalf("duplicate submission changed history: turns=%d evidence=%d", turns, evidence)
	}
	if _, err := fixture.service.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "不同回答", "answer-request-1"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("reused key with different payload error = %v", err)
	}
}

func TestMasteryScoreAggregatesAcrossAnswerVersions(t *testing.T) {
	provider := newFakeProvider(validFakeResult())
	provider.result.MasteryScore = 0.8
	fixture := newServiceTestFixture(t, provider)
	ctx := context.Background()
	if _, err := fixture.service.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "第一个完整回答", "answer-version-1"); err != nil {
		t.Fatal(err)
	}
	provider.result.MasteryScore = 0.4
	if _, err := fixture.service.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "第二个修正回答", "answer-version-2"); err != nil {
		t.Fatal(err)
	}
	var mastery model.MasteryRecord
	if err := fixture.db.Where("lesson_id = ?", fixture.lesson.ID).First(&mastery).Error; err != nil {
		t.Fatal(err)
	}
	if math.Abs(mastery.MasteryScore-0.6) > 1e-9 || mastery.AnswerCount != 2 {
		t.Fatalf("mastery should aggregate answer history: %+v", mastery)
	}
	var turns []model.LearningTurn
	fixture.db.Order("id ASC").Find(&turns)
	if len(turns) != 2 || math.Abs(turns[1].MasteryScoreBefore-0.8) > 1e-9 || math.Abs(turns[1].MasteryScoreAfter-0.6) > 1e-9 {
		t.Fatalf("mastery snapshots missing: %+v", turns)
	}
	var evidenceCount, eventCount int64
	fixture.db.Model(&model.CognitiveEvidence{}).Where("lesson_id = ?", fixture.lesson.ID).Count(&evidenceCount)
	fixture.db.Model(&model.CognitiveStateEvent{}).Where("lesson_id = ?", fixture.lesson.ID).Count(&eventCount)
	if evidenceCount != 4 || eventCount != 2 {
		t.Fatalf("a later evaluation overwrote historical evidence: evidence=%d events=%d", evidenceCount, eventCount)
	}
}
