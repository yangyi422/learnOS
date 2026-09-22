package service

import (
	"context"
	"path/filepath"
	"testing"

	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/model"
)

func TestConsistencyCheckValidatesMasteryAndEvidenceScope(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(config.Config{
		DatabasePath:         filepath.Join(dir, "consistency.db"),
		BackupDir:            filepath.Join(dir, "backups"),
		BackupRetentionCount: 3,
		AppVersion:           "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	first := model.Course{Name: "first", Status: model.CourseStatusLearning}
	second := model.Course{Name: "second", Status: model.CourseStatusLearning}
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	unit := model.CourseUnit{CourseID: first.ID, Title: "unit", Status: model.CourseUnitStatusLearning}
	if err := db.Create(&unit).Error; err != nil {
		t.Fatal(err)
	}
	lesson := model.Lesson{CourseID: first.ID, UnitID: unit.ID, Title: "lesson", CoreQuestion: "why", Status: model.LessonStatusLearning, ContentRole: model.ContentRoleCore}
	if err := db.Create(&lesson).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&first).Updates(map[string]interface{}{"current_unit_id": unit.ID, "current_lesson_id": lesson.ID}).Error; err != nil {
		t.Fatal(err)
	}

	turn := model.LearningTurn{CourseID: first.ID, UnitID: unit.ID, LessonID: lesson.ID, Question: "q", UserAnswer: "a", Result: "partial"}
	if err := db.Create(&turn).Error; err != nil {
		t.Fatal(err)
	}
	badMastery := model.MasteryRecord{CourseID: second.ID, LessonID: lesson.ID, MasteryScore: 1.2}
	if err := db.Create(&badMastery).Error; err != nil {
		t.Fatal(err)
	}
	badEvidence := model.CognitiveEvidence{CourseID: second.ID, LessonID: lesson.ID, LearningTurnID: turn.ID, EvidenceIndex: 0, EvidenceType: model.CognitiveEvidenceRecognition, CognitiveLevel: model.CognitiveLevelRecognize, Polarity: model.CognitiveEvidenceSupport, Description: "evidence", Source: "test"}
	if err := db.Create(&badEvidence).Error; err != nil {
		t.Fatal(err)
	}

	report, err := NewConsistencyService(db).Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Healthy {
		t.Fatal("expected inconsistent report")
	}
	want := map[string]bool{
		"MASTERY_COURSE_MISMATCH":           false,
		"MASTERY_SCORE_INVALID":             false,
		"COGNITIVE_EVIDENCE_SCOPE_MISMATCH": false,
	}
	for _, issue := range report.Errors {
		if _, ok := want[issue.Code]; ok {
			want[issue.Code] = true
		}
	}
	for code, found := range want {
		if !found {
			t.Errorf("expected issue %s, got %+v", code, report.Errors)
		}
	}
}

func TestConsistencyCheckAcceptsAlignedCourseGraphAndMastery(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(config.Config{DatabasePath: filepath.Join(dir, "healthy.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 3, AppVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	course := model.Course{Name: "course", Status: model.CourseStatusLearning}
	if err := db.Create(&course).Error; err != nil {
		t.Fatal(err)
	}
	unit := model.CourseUnit{CourseID: course.ID, Title: "unit", Status: model.CourseUnitStatusLearning}
	if err := db.Create(&unit).Error; err != nil {
		t.Fatal(err)
	}
	lesson := model.Lesson{CourseID: course.ID, UnitID: unit.ID, Title: "lesson", CoreQuestion: "why", Status: model.LessonStatusLearning, ContentRole: model.ContentRoleFoundation}
	if err := db.Create(&lesson).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&course).Updates(map[string]interface{}{"current_unit_id": unit.ID, "current_lesson_id": lesson.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.MasteryRecord{CourseID: course.ID, LessonID: lesson.ID, MasteryScore: 0.6}).Error; err != nil {
		t.Fatal(err)
	}

	report, err := NewConsistencyService(db).Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !report.Healthy {
		t.Fatalf("expected healthy report, got %+v", report.Errors)
	}
}
