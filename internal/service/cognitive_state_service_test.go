package service

import (
	"context"
	"errors"
	"testing"

	"learnos/internal/ai"
	"learnos/internal/model"
)

func TestCalculateTurnLevelHonorsResultAndTargetCaps(t *testing.T) {
	tests := []struct {
		name         string
		result       string
		demonstrated string
		target       string
		want         string
	}{
		{name: "incorrect exposed", result: "incorrect", demonstrated: "understand", target: model.AssessmentTargetUnderstand, want: model.CognitiveLevelExposed},
		{name: "partially recognize", result: "partially_correct", demonstrated: "apply", target: model.AssessmentTargetApply, want: model.CognitiveLevelRecognize},
		{name: "understand target", result: "mostly_correct", demonstrated: "understand", target: model.AssessmentTargetUnderstand, want: model.CognitiveLevelUnderstand},
		{name: "apply target accepts understand", result: "correct", demonstrated: "understand", target: model.AssessmentTargetApply, want: model.CognitiveLevelUnderstand},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateTurnLevel(ai.EvaluationResult{Result: tt.result, DemonstratedLevel: tt.demonstrated}, tt.target)
			if err != nil {
				t.Fatalf("calculate turn level: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestValidateCognitiveEvidenceCaps(t *testing.T) {
	tests := []struct {
		evidenceType string
		level        string
		valid        bool
	}{
		{evidenceType: model.CognitiveEvidenceRecognition, level: model.CognitiveLevelRecognize, valid: true},
		{evidenceType: model.CognitiveEvidenceRecognition, level: model.CognitiveLevelUnderstand},
		{evidenceType: model.CognitiveEvidenceConceptExplanation, level: model.CognitiveLevelUnderstand, valid: true},
		{evidenceType: model.CognitiveEvidenceBoundaryAwareness, level: model.CognitiveLevelApply},
		{evidenceType: model.CognitiveEvidenceApplication, level: model.CognitiveLevelApply, valid: true},
		{evidenceType: model.CognitiveEvidenceTransfer, level: model.CognitiveLevelTransfer, valid: true},
	}
	for _, tt := range tests {
		t.Run(tt.evidenceType+"/"+tt.level, func(t *testing.T) {
			evidence := []ai.EvaluationEvidence{{
				EvidenceType: tt.evidenceType, CognitiveLevel: tt.level,
				Polarity: model.CognitiveEvidenceSupport, Description: "具体证据",
			}}
			err := ValidateCognitiveEvidence(evidence, tt.level)
			if (err == nil) != tt.valid {
				t.Fatalf("expected valid=%t, got error=%v", tt.valid, err)
			}
		})
	}
}

func TestCognitiveStateKeepsHighestLevelAndTracksReviewRecovery(t *testing.T) {
	fixture := newServiceTestFixture(t, newFakeProvider(validFakeResult()))
	ctx := context.Background()

	if _, err := fixture.service.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "第一次回答"); err != nil {
		t.Fatalf("submit first answer: %v", err)
	}
	var state model.CognitiveState
	if err := fixture.db.Where("lesson_id = ?", fixture.lesson.ID).First(&state).Error; err != nil {
		t.Fatalf("find cognitive state: %v", err)
	}
	if state.CurrentLevel != model.CognitiveLevelUnderstand || state.Status != model.CognitiveStatusStable || state.UnderstandingSummary == "" || state.LastLearningTurnID == nil {
		t.Fatalf("unexpected first cognitive state: %+v", state)
	}
	firstSummary := state.UnderstandingSummary

	fixture.provider.result.Result = "insufficient"
	fixture.provider.result.DemonstratedLevel = model.CognitiveLevelExposed
	fixture.provider.result.UserUnderstandingSummary = "本次回答没有提供足够信息。"
	fixture.provider.result.CognitiveEvidence = []ai.EvaluationEvidence{}
	if _, err := fixture.service.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "不知道"); err != nil {
		t.Fatalf("submit insufficient answer: %v", err)
	}
	if err := fixture.db.Where("lesson_id = ?", fixture.lesson.ID).First(&state).Error; err != nil {
		t.Fatalf("reload cognitive state: %v", err)
	}
	if state.CurrentLevel != model.CognitiveLevelUnderstand || state.Status != model.CognitiveStatusNeedsReview || state.UnderstandingSummary != firstSummary {
		t.Fatalf("insufficient answer downgraded or replaced summary: %+v", state)
	}

	fixture.provider.result = validFakeResult()
	if _, err := fixture.service.SubmitAnswer(ctx, fixture.course.ID, fixture.lesson.ID, "再次解释"); err != nil {
		t.Fatalf("submit recovery answer: %v", err)
	}
	if err := fixture.db.Where("lesson_id = ?", fixture.lesson.ID).First(&state).Error; err != nil {
		t.Fatalf("reload recovered state: %v", err)
	}
	if state.CurrentLevel != model.CognitiveLevelUnderstand || state.Status != model.CognitiveStatusStable {
		t.Fatalf("expected stable understand after recovery: %+v", state)
	}

	var evidenceCount, eventCount int64
	fixture.db.Model(&model.CognitiveEvidence{}).Where("lesson_id = ?", fixture.lesson.ID).Count(&evidenceCount)
	fixture.db.Model(&model.CognitiveStateEvent{}).Where("lesson_id = ?", fixture.lesson.ID).Count(&eventCount)
	if evidenceCount != 4 || eventCount != 3 {
		t.Fatalf("expected evidence/event for every successful turn, got evidence=%d events=%d", evidenceCount, eventCount)
	}
	var events []model.CognitiveStateEvent
	if err := fixture.db.Where("lesson_id = ?", fixture.lesson.ID).Order("created_at DESC, id DESC").Find(&events).Error; err != nil {
		t.Fatalf("list cognitive events: %v", err)
	}
	if len(events) != 3 || events[0].ToStatus != model.CognitiveStatusStable || events[1].ToStatus != model.CognitiveStatusNeedsReview || events[2].FromLevel != model.CognitiveLevelUnseen {
		t.Fatalf("unexpected cognitive timeline: %+v", events)
	}
}

func TestFirstInsufficientAnswerCreatesExposedDevelopingState(t *testing.T) {
	result := validFakeResult()
	result.Result = "insufficient"
	result.DemonstratedLevel = model.CognitiveLevelExposed
	result.UserUnderstandingSummary = "本次回答没有提供足够信息。"
	result.CognitiveEvidence = []ai.EvaluationEvidence{}
	fixture := newServiceTestFixture(t, newFakeProvider(result))
	if _, err := fixture.service.SubmitAnswer(context.Background(), fixture.course.ID, fixture.lesson.ID, "不知道"); err != nil {
		t.Fatalf("submit insufficient answer: %v", err)
	}
	var state model.CognitiveState
	if err := fixture.db.Where("lesson_id = ?", fixture.lesson.ID).First(&state).Error; err != nil {
		t.Fatalf("find cognitive state: %v", err)
	}
	if state.CurrentLevel != model.CognitiveLevelExposed || state.Status != model.CognitiveStatusDeveloping {
		t.Fatalf("expected exposed/developing, got %+v", state)
	}
}

func TestContradictionEvidenceIsPersisted(t *testing.T) {
	result := validFakeResult()
	result.CognitiveEvidence = append(result.CognitiveEvidence, ai.EvaluationEvidence{
		EvidenceType: model.CognitiveEvidenceContradiction, CognitiveLevel: model.CognitiveLevelUnderstand,
		Polarity: model.CognitiveEvidenceContradict, Description: "仍然把没有口渴等同于没有补水风险",
	})
	fixture := newServiceTestFixture(t, newFakeProvider(result))
	if _, err := fixture.service.SubmitAnswer(context.Background(), fixture.course.ID, fixture.lesson.ID, "存在矛盾的回答"); err != nil {
		t.Fatalf("submit contradiction answer: %v", err)
	}
	var evidence model.CognitiveEvidence
	if err := fixture.db.Where("polarity = ?", model.CognitiveEvidenceContradict).First(&evidence).Error; err != nil {
		t.Fatalf("find contradiction evidence: %v", err)
	}
	if evidence.EvidenceType != model.CognitiveEvidenceContradiction || evidence.CognitiveLevel != model.CognitiveLevelUnderstand {
		t.Fatalf("unexpected contradiction evidence: %+v", evidence)
	}
	var state model.CognitiveState
	if err := fixture.db.Where("lesson_id = ?", fixture.lesson.ID).First(&state).Error; err != nil {
		t.Fatalf("find cognitive state: %v", err)
	}
	if state.Status != model.CognitiveStatusNeedsReview {
		t.Fatalf("expected contradiction to require review, got %+v", state)
	}
}

func TestSubmitAnswerRejectsAILevelAboveAssessmentTarget(t *testing.T) {
	result := validFakeResult()
	result.DemonstratedLevel = model.CognitiveLevelApply
	fixture := newServiceTestFixture(t, newFakeProvider(result))
	_, err := fixture.service.SubmitAnswer(context.Background(), fixture.course.ID, fixture.lesson.ID, "超出目标的回答")
	if !errors.Is(err, ai.ErrInvalidResponse) {
		t.Fatalf("expected invalid AI response, got %v", err)
	}
	var turns, states, evidence, events int64
	fixture.db.Model(&model.LearningTurn{}).Count(&turns)
	fixture.db.Model(&model.CognitiveState{}).Count(&states)
	fixture.db.Model(&model.CognitiveEvidence{}).Count(&evidence)
	fixture.db.Model(&model.CognitiveStateEvent{}).Count(&events)
	if turns != 0 || states != 0 || evidence != 0 || events != 0 {
		t.Fatalf("invalid AI response left formal cognitive data: turns=%d states=%d evidence=%d events=%d", turns, states, evidence, events)
	}
}
