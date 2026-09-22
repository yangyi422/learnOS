package database

import (
	"path/filepath"
	"testing"

	"learnos/internal/config"
	"learnos/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOpenCreatesPreMigrationBackupForExistingDatabase(t *testing.T) {
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "learnos.db")
	legacy, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := legacy.AutoMigrate(&model.Course{}); err != nil {
		t.Fatal(err)
	}
	legacyCourse := model.Course{Name: "迁移保留测试", Status: model.CourseStatusLearning}
	if err := legacy.Create(&legacyCourse).Error; err != nil {
		t.Fatal(err)
	}
	legacySQL, err := legacy.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = legacySQL.Close()

	db, err := Open(config.Config{AppVersion: "test", DatabasePath: databasePath, BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var metadata model.SystemMetadata
	if err := db.First(&metadata, 1).Error; err != nil {
		t.Fatal(err)
	}
	if metadata.SchemaVersion != model.CurrentSchemaVersion {
		t.Fatalf("unexpected schema version: %d", metadata.SchemaVersion)
	}
	var preserved model.Course
	if err := db.First(&preserved, legacyCourse.ID).Error; err != nil || preserved.Name != legacyCourse.Name {
		t.Fatalf("migration did not preserve existing data: course=%+v err=%v", preserved, err)
	}
	columns := map[any][]string{
		&model.LearningTurn{}:        {"idempotency_key", "evidence_used_json", "confidence", "uncertainty", "recommended_next_action", "transfer_challenge_eligible", "mastery_score_before", "mastery_score_after"},
		&model.AssessmentChallenge{}: {"idempotency_key"},
		&model.ChallengeAttempt{}:    {"idempotency_key", "explanation"},
		&model.CourseUnit{}:          {"blueprint_unit_id"},
		&model.Misconception{}:       {"review_status", "user_note"},
		&model.ExplorationQuestion{}: {"priority"},
	}
	for table, names := range columns {
		for _, name := range names {
			if !db.Migrator().HasColumn(table, name) {
				t.Fatalf("schema %d migration is missing %T.%s", model.CurrentSchemaVersion, table, name)
			}
		}
	}
	backups, err := ListBackups(filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected one pre-migration backup, got %d", len(backups))
	}
}

func TestBackfillCourseUnitBlueprintIDsUsesAppliedLessonIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "backfill.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Course{}, &model.CourseUnit{}, &model.Lesson{}, &model.CurriculumBlueprint{}, &model.CurriculumBlueprintUnit{}, &model.CurriculumBlueprintLesson{}); err != nil {
		t.Fatal(err)
	}
	course := model.Course{Name: "稳定关联", Status: model.CourseStatusLearning}
	if err := db.Create(&course).Error; err != nil {
		t.Fatal(err)
	}
	unit := model.CourseUnit{CourseID: course.ID, Title: "标题可以不同", SortOrder: 1}
	if err := db.Create(&unit).Error; err != nil {
		t.Fatal(err)
	}
	lesson := model.Lesson{CourseID: course.ID, UnitID: unit.ID, Title: "Lesson", Status: model.LessonStatusPending, ContentRole: model.ContentRoleCore, DepthLevel: 1}
	if err := db.Create(&lesson).Error; err != nil {
		t.Fatal(err)
	}
	blueprint := model.CurriculumBlueprint{CourseID: &course.ID, Name: "Blueprint", Status: model.CurriculumBlueprintStatusActive}
	if err := db.Create(&blueprint).Error; err != nil {
		t.Fatal(err)
	}
	blueprintUnit := model.CurriculumBlueprintUnit{BlueprintID: blueprint.ID, Key: "stable-key", Title: "完全不同标题", SortOrder: 1}
	if err := db.Create(&blueprintUnit).Error; err != nil {
		t.Fatal(err)
	}
	blueprintLesson := model.CurriculumBlueprintLesson{BlueprintID: blueprint.ID, BlueprintUnitID: blueprintUnit.ID, Key: "lesson-key", Title: "Blueprint Lesson", AppliedLessonID: &lesson.ID, ContentRole: model.ContentRoleCore, DepthLevel: 1}
	if err := db.Create(&blueprintLesson).Error; err != nil {
		t.Fatal(err)
	}
	if err := backfillCourseUnitBlueprintIDs(db); err != nil {
		t.Fatal(err)
	}
	var reloaded model.CourseUnit
	if err := db.First(&reloaded, unit.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.BlueprintUnitID == nil || *reloaded.BlueprintUnitID != blueprintUnit.ID {
		t.Fatalf("stable blueprint unit link was not backfilled: %+v", reloaded)
	}
}
