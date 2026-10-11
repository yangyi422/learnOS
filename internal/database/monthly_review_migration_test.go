package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"learnos/internal/config"
	"learnos/internal/model"
	"path/filepath"
	"testing"
	"time"
)

func TestMonthlyReviewSchema25MigrationPreservesUnknownCompletion(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{Environment: "production", DatabasePath: filepath.Join(dir, "monthly.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	old, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err = old.AutoMigrate(&model.SystemMetadata{}, &model.Project{}, &model.ProjectTask{}, &model.Course{}, &model.Lesson{}, &model.LifeEvent{}, &model.LifeGoal{}, &model.LifeGoalEntry{}, &model.AssetSnapshot{}, &model.GoalAssetReference{}, &model.FinanceTransaction{}); err != nil {
		t.Fatal(err)
	}
	old.Create(&model.SystemMetadata{ID: 1, SchemaVersion: 24})
	p := model.Project{UserID: 1, Title: "legacy completed", Status: "completed"}
	old.Create(&p)
	now := time.Now().UTC()
	old.Create(&model.ProjectTask{ProjectID: p.ID, Title: "inferred old task", Status: "done", CompletedAt: &now})
	old.Create(&model.Course{UserID: 1, Name: "legacy course", Status: model.CourseStatusCompleted})
	old.Create(&model.Lesson{CourseID: 1, Title: "legacy lesson", Status: model.LessonStatusCompleted})
	for _, change := range []string{"ALTER TABLE project_tasks DROP COLUMN actual_completed_at", "ALTER TABLE projects DROP COLUMN completed_at", "ALTER TABLE courses DROP COLUMN completed_at", "ALTER TABLE lessons DROP COLUMN completed_at"} {
		if err = old.Exec(change).Error; err != nil {
			t.Fatal(err)
		}
	}
	conn, _ := old.DB()
	conn.Close()
	db, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	backups, err := ListBackups(cfg.BackupDir)
	if err != nil || len(backups) != 1 {
		t.Fatal(backups, err)
	}
	var meta model.SystemMetadata
	db.First(&meta, 1)
	if meta.SchemaVersion != model.CurrentSchemaVersion || !db.Migrator().HasTable(&model.MonthlyReview{}) || !db.Migrator().HasTable(&model.MonthlyReviewFacts{}) {
		t.Fatal(meta)
	}
	var task model.ProjectTask
	db.First(&task, 1)
	if task.CompletedAt == nil || task.ActualCompletedAt != nil || task.Title != "inferred old task" {
		t.Fatal("invented completion", task)
	}
	var project model.Project
	db.First(&project, p.ID)
	var course model.Course
	db.First(&course, 1)
	var lesson model.Lesson
	db.First(&lesson, 1)
	if project.CompletedAt != nil || course.CompletedAt != nil || lesson.CompletedAt != nil {
		t.Fatal("fabricated historical dates")
	}
	conn, _ = db.DB()
	conn.Close()
	db, err = Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ = db.DB()
	defer conn.Close()
	db.First(&task, 1)
	if task.ActualCompletedAt != nil {
		t.Fatal("repeat migration inferred dates")
	}
}
