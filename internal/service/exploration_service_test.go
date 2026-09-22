package service

import (
	"context"
	"testing"

	"learnos/internal/model"
	"learnos/internal/repository"
)

func TestExplorationQuestionSaveFilterAndUndo(t *testing.T) {
	fixture := newChallengeFixture(t)
	if err := fixture.db.AutoMigrate(&model.ExplorationDirection{}, &model.ExplorationQuestion{}); err != nil {
		t.Fatal(err)
	}
	direction := model.ExplorationDirection{
		CourseID: fixture.course.ID, SourceLessonID: &fixture.lesson.ID,
		TargetCourseID: fixture.course.ID, TargetLessonID: fixture.lesson.ID,
		DirectionType: model.ExplorationDirectionAdjacent, Title: "方向", WhyWorthExploring: "测试保存与撤销",
		ReasonCode: ReasonCodeGraphNeighbor, Status: model.ExplorationDirectionActive,
	}
	if err := fixture.db.Create(&direction).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewExplorationRepository(fixture.db)
	service := NewExplorationService(fixture.courses, repository.NewKnowledgeGraphRepository(fixture.db), fixture.learning, repository.NewCognitiveRepository(fixture.db), fixture.misconception, repo)

	question, err := service.AddQuestion(context.Background(), fixture.course.ID, direction.ID)
	if err != nil {
		t.Fatal(err)
	}
	if question.Priority != model.ExplorationPriorityNormal {
		t.Fatalf("default priority = %q", question.Priority)
	}
	if _, err := service.SetQuestionPriority(context.Background(), fixture.course.ID, question.ID, model.ExplorationPriorityHigh); err != nil {
		t.Fatal(err)
	}
	items, err := service.ListQuestions(context.Background(), fixture.course.ID, "", 10, ExplorationQuestionFilter{TargetCourseID: fixture.course.ID, Priority: model.ExplorationPriorityHigh})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != question.ID {
		t.Fatalf("question filters returned %+v", items)
	}

	if _, err := service.UndoQuestion(context.Background(), fixture.course.ID, direction.ID); err != nil {
		t.Fatal(err)
	}
	var saved model.ExplorationDirection
	if err := fixture.db.First(&saved, direction.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != model.ExplorationDirectionActive {
		t.Fatalf("direction status = %q", saved.Status)
	}
	var archived model.ExplorationQuestion
	if err := fixture.db.First(&archived, question.ID).Error; err != nil {
		t.Fatal(err)
	}
	if archived.Status != model.ExplorationQuestionArchived {
		t.Fatalf("question status = %q", archived.Status)
	}
}
