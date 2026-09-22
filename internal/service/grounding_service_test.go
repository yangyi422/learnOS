package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newGroundingFixture(t *testing.T) (*GroundingService, *gorm.DB, model.Course, model.CurriculumBlueprintLesson) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "grounding.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.Course{}, &model.CourseUnit{}, &model.Lesson{}, &model.LessonRelation{}, &model.LearningTurn{}, &model.MasteryRecord{}, &model.Misconception{}, &model.AIEvaluationRun{}, &model.CognitiveState{}, &model.CognitiveEvidence{}, &model.CognitiveStateEvent{}, &model.AssessmentChallenge{}, &model.ChallengeAttempt{}, &model.MisconceptionEvent{}, &model.MisconceptionPatternLink{}, &model.CurriculumBlueprint{}, &model.CurriculumBlueprintUnit{}, &model.CurriculumBlueprintLesson{}, &model.CurriculumBlueprintRelation{}, &model.CurriculumDraft{}, &model.KnowledgeSource{}, &model.SourceEvidence{}, &model.GroundingLink{}, &model.SourceCredibilityAssessment{}, &model.GroundingReviewEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	courses := repository.NewCourseRepository(db)
	if err := courses.SeedStarterCourse(context.Background()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var course model.Course
	if err := db.Where("name = ?", "营养学").First(&course).Error; err != nil {
		t.Fatalf("course: %v", err)
	}
	var blueprint model.CurriculumBlueprint
	if err := db.Where("course_id = ?", course.ID).First(&blueprint).Error; err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	var lesson model.CurriculumBlueprintLesson
	if err := db.Where("blueprint_id = ? AND importance = ?", blueprint.ID, model.CurriculumImportanceCore).First(&lesson).Error; err != nil {
		t.Fatalf("blueprint lesson: %v", err)
	}
	return NewGroundingService(repository.NewGroundingRepository(db), repository.NewCurriculumRepository(db), courses), db, course, lesson
}

func TestGroundingSourceEvidenceAndCredibilityLifecycle(t *testing.T) {
	service, db, course, blueprintLesson := newGroundingFixture(t)
	ctx := context.Background()
	year := 2024
	source, err := service.CreateSource(ctx, SourceInput{Title: "Demo Nutrition Reference", SourceType: model.KnowledgeSourceTypeCourseMaterial, PublicationYear: &year, VerificationStatus: model.KnowledgeSourceVerificationUnverified})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if _, err := service.CreateSource(ctx, SourceInput{Title: "Demo Nutrition Reference", SourceType: model.KnowledgeSourceTypeCourseMaterial, PublicationYear: &year}); !errors.Is(err, ErrSourceDuplicate) {
		t.Fatalf("expected duplicate source, got %v", err)
	}
	evidence, err := service.AddEvidence(ctx, source.ID, EvidenceInput{Locator: "section 1", Summary: "用于说明测试课程知识节点的摘要。"})
	if err != nil {
		t.Fatalf("add evidence: %v", err)
	}
	link, err := service.CreateLink(ctx, GroundingLinkInput{EvidenceID: evidence.ID, TargetType: model.GroundingTargetBlueprintLesson, TargetID: blueprintLesson.ID, Relation: model.GroundingRelationSupports, Strength: model.GroundingStrengthModerate})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}
	if link.Link.Status != model.GroundingLinkStatusProposed {
		t.Fatalf("new link must be proposed: %+v", link.Link)
	}
	if _, err := service.ReviewLink(ctx, link.Link.ID); err != nil {
		t.Fatalf("review link: %v", err)
	}
	var before model.CurriculumBlueprintLesson
	if err := db.First(&before, blueprintLesson.ID).Error; err != nil {
		t.Fatal(err)
	}
	if before.GroundingStatus != model.CurriculumGroundingPartially {
		t.Fatalf("expected partial without credibility, got %s", before.GroundingStatus)
	}
	assessment, err := service.CreateCredibility(ctx, source.ID, CredibilityInput{AuthorityScore: 15, MethodologyScore: 15, DirectnessScore: 15, RecencyScore: 10, IndependenceScore: 10})
	if err != nil {
		t.Fatalf("create credibility: %v", err)
	}
	if assessment.Status != model.CredibilityAssessmentStatusDraft || assessment.OverallScore != 65 {
		t.Fatalf("unexpected credibility: %+v", assessment)
	}
	if _, err := service.ReviewCredibility(ctx, assessment.ID); err != nil {
		t.Fatalf("review credibility: %v", err)
	}
	var after model.CurriculumBlueprintLesson
	if err := db.First(&after, blueprintLesson.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.GroundingStatus != model.CurriculumGroundingGrounded {
		t.Fatalf("expected grounded after review, got %s", after.GroundingStatus)
	}
	var turns int64
	if err := db.Model(&model.LearningTurn{}).Where("course_id = ?", course.ID).Count(&turns).Error; err != nil {
		t.Fatal(err)
	}
	if turns != 0 {
		t.Fatalf("grounding created learning turns: %d", turns)
	}
}

func TestGroundingConflictIsExplicit(t *testing.T) {
	service, db, _, blueprintLesson := newGroundingFixture(t)
	ctx := context.Background()
	source, _ := service.CreateSource(ctx, SourceInput{Title: "Conflict Reference", SourceType: model.KnowledgeSourceTypeCourseMaterial})
	evidence, _ := service.AddEvidence(ctx, source.ID, EvidenceInput{Summary: "summary"})
	first, _ := service.CreateLink(ctx, GroundingLinkInput{EvidenceID: evidence.ID, TargetType: model.GroundingTargetBlueprintLesson, TargetID: blueprintLesson.ID, Relation: model.GroundingRelationSupports, Strength: model.GroundingStrengthModerate})
	secondEvidence, _ := service.AddEvidence(ctx, source.ID, EvidenceInput{Summary: "contrary summary"})
	second, _ := service.CreateLink(ctx, GroundingLinkInput{EvidenceID: secondEvidence.ID, TargetType: model.GroundingTargetBlueprintLesson, TargetID: blueprintLesson.ID, Relation: model.GroundingRelationContradicts, Strength: model.GroundingStrengthModerate})
	if _, err := service.ReviewLink(ctx, first.Link.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReviewLink(ctx, second.Link.ID); err != nil {
		t.Fatal(err)
	}
	var stored model.CurriculumBlueprintLesson
	if err := db.First(&stored, blueprintLesson.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.GroundingStatus != model.CurriculumGroundingConflicted {
		t.Fatalf("expected conflicted, got %s", stored.GroundingStatus)
	}
}
