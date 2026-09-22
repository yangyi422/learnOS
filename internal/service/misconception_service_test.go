package service

import (
	"context"
	"testing"

	"learnos/internal/model"
	"learnos/internal/repository"
)

func TestMisconceptionPatternRequiresRepeatedOrConfirmedEvidence(t *testing.T) {
	fixture := newChallengeFixture(t)
	ctx := context.Background()
	misconception := model.Misconception{
		CourseID: fixture.course.ID, LessonID: fixture.lesson.ID,
		OriginalUnderstanding: "只看一个因素", CorrectUnderstanding: "需要结合多个因素",
		Status: model.MisconceptionStatusActive, ReviewStatus: model.MisconceptionReviewAIInferred,
	}
	if err := fixture.db.Create(&misconception).Error; err != nil {
		t.Fatal(err)
	}
	if err := fixture.db.Create(&model.MisconceptionPatternLink{MisconceptionID: misconception.ID, PatternKey: "single_factor_reasoning", Source: "ai"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := fixture.db.Create(&model.MisconceptionEvent{CourseID: fixture.course.ID, LessonID: fixture.lesson.ID, MisconceptionID: misconception.ID, EventType: model.MisconceptionEventObserved, Notes: "first"}).Error; err != nil {
		t.Fatal(err)
	}

	service := NewMisconceptionService(fixture.courses, repository.NewKnowledgeGraphRepository(fixture.db), fixture.misconception, fixture.learning)
	network, err := service.GetNetwork(ctx, fixture.course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(network.Patterns) != 0 || len(network.Misconceptions) != 1 || network.Misconceptions[0].StableEvidence {
		t.Fatalf("single AI observation must remain an unconfirmed clue: %+v", network)
	}

	view, err := service.Review(ctx, fixture.course.ID, misconception.ID, "confirm", "这个判断符合我的回答")
	if err != nil {
		t.Fatal(err)
	}
	if view.ReviewStatus != model.MisconceptionReviewUserConfirmed || !view.StableEvidence {
		t.Fatalf("confirmed misconception should become stable evidence: %+v", view)
	}
	network, err = service.GetNetwork(ctx, fixture.course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(network.Patterns) != 1 || network.Patterns[0].Key != "single_factor_reasoning" {
		t.Fatalf("confirmed evidence should participate in pattern aggregation: %+v", network.Patterns)
	}
}

func TestMisconceptionCorrectReviewRequiresUserExplanation(t *testing.T) {
	fixture := newChallengeFixture(t)
	item := model.Misconception{CourseID: fixture.course.ID, LessonID: fixture.lesson.ID, OriginalUnderstanding: "旧理解", CorrectUnderstanding: "AI 建议", Status: model.MisconceptionStatusActive}
	if err := fixture.db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	service := NewMisconceptionService(fixture.courses, repository.NewKnowledgeGraphRepository(fixture.db), fixture.misconception, fixture.learning)
	if _, err := service.Review(context.Background(), fixture.course.ID, item.ID, "correct", "   "); err == nil {
		t.Fatal("correcting an AI inference without an explanation should fail")
	}
}
