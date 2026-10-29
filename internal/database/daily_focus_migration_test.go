package database

import (
	"path/filepath"
	"testing"
	"time"

	"learnos/internal/config"
	"learnos/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestDailyFocusMigrationIsIdempotentAndPreservesWorkspace migrates a schema 30
// database to 31 twice. The migration only creates daily_focuses: existing
// projects and tasks keep every fact and no historical focus is generated.
func TestDailyFocusMigrationIsIdempotentAndPreservesWorkspace(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{Environment: "production", AppVersion: "test", DatabasePath: filepath.Join(dir, "legacy.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	legacy, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := legacy.AutoMigrate(&model.Project{}, &model.ProjectTask{}, &model.SystemMetadata{}); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Create(&model.SystemMetadata{ID: 1, SchemaVersion: 30}).Error; err != nil {
		t.Fatal(err)
	}
	if err := legacy.Create(&model.Project{ID: 1, UserID: 1, Title: "工作项目", Status: "active", Context: "work"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := legacy.Exec(`INSERT INTO project_tasks(id, project_id, title, status, sort_order, priority, plan_date, due_date, completed_at, archived_at, updated_at)
		VALUES(81, 1, '既有任务', 'done', 3072, 'high', '2026-10-21', '2026-10-23', '2026-10-20 12:00:00', '2026-10-21 08:00:00', '2026-10-21 08:00:00')`).Error; err != nil {
		t.Fatal(err)
	}
	connection, err := legacy.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}

	for attempt := 0; attempt < 2; attempt++ {
		db, err := Open(cfg)
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		var metadata model.SystemMetadata
		if err := db.First(&metadata, 1).Error; err != nil {
			t.Fatal(err)
		}
		if model.CurrentSchemaVersion != 33 {
			t.Fatalf("schema constant: %d", model.CurrentSchemaVersion)
		}
		if metadata.SchemaVersion != model.CurrentSchemaVersion {
			t.Fatalf("schema version: %d", metadata.SchemaVersion)
		}
		if !db.Migrator().HasTable(&model.DailyFocus{}) {
			t.Fatal("daily_focuses table missing")
		}
		var tableCount int64
		if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", "daily_focuses").Scan(&tableCount).Error; err != nil || tableCount != 1 {
			t.Fatalf("daily_focuses table name: count=%d err=%v", tableCount, err)
		}
		for _, column := range []string{"user_id", "focus_date", "context", "task_id", "revision", "created_at", "updated_at"} {
			if !db.Migrator().HasColumn(&model.DailyFocus{}, column) {
				t.Fatalf("daily_focuses.%s missing", column)
			}
		}
		if !db.Migrator().HasIndex(&model.DailyFocus{}, "idx_daily_focus_scope") {
			t.Fatal("daily focus scope index missing")
		}
		var focusCount int64
		if err := db.Model(&model.DailyFocus{}).Count(&focusCount).Error; err != nil {
			t.Fatal(err)
		}
		// The first open must not backfill any historical focus. The second open
		// sees only the row this test inserted deliberately.
		if focusCount != int64(attempt) {
			t.Fatalf("attempt %d: generated history focuses: %d", attempt, focusCount)
		}

		var project model.Project
		if err := db.First(&project, 1).Error; err != nil {
			t.Fatal(err)
		}
		if project.Title != "工作项目" || project.Context != "work" || project.Status != "active" || project.UserID != 1 {
			t.Fatalf("project changed by migration: %+v", project)
		}
		var taskCount int64
		if err := db.Model(&model.ProjectTask{}).Count(&taskCount).Error; err != nil || taskCount != 1 {
			t.Fatalf("task count changed: %d %v", taskCount, err)
		}
		var task model.ProjectTask
		if err := db.First(&task, 81).Error; err != nil {
			t.Fatal(err)
		}
		if task.Title != "既有任务" || task.Status != "done" || task.SortOrder != 3072 || task.Priority != "high" {
			t.Fatalf("task facts changed: %+v", task)
		}
		if task.PlanDate == nil || *task.PlanDate != "2026-10-21" || task.DueDate == nil || *task.DueDate != "2026-10-23" {
			t.Fatalf("task dates changed: %+v", task)
		}
		if task.ArchivedAt == nil || task.CompletedAt == nil || !task.CompletedAt.Equal(time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)) {
			t.Fatalf("task timestamps changed: %+v", task)
		}

		if attempt == 0 {
			focus := model.DailyFocus{UserID: 1, FocusDate: "2026-10-21", Context: "work", TaskID: &task.ID, Revision: 1}
			if err := db.Create(&focus).Error; err != nil {
				t.Fatalf("insert focus: %v", err)
			}
		} else {
			var focus model.DailyFocus
			if err := db.Where("user_id = ? AND focus_date = ? AND context = ?", 1, "2026-10-21", "work").First(&focus).Error; err != nil {
				t.Fatal(err)
			}
			if focus.TaskID == nil || *focus.TaskID != 81 || focus.Revision != 1 || focus.FocusDate != "2026-10-21" {
				t.Fatalf("repeat migration changed focus: %+v", focus)
			}
			duplicate := model.DailyFocus{UserID: 1, FocusDate: "2026-10-21", Context: "work", Revision: 2}
			if err := db.Create(&duplicate).Error; err == nil {
				t.Fatal("duplicate day/scene focus accepted after migration")
			}
		}
		connection, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		if err := connection.Close(); err != nil {
			t.Fatal(err)
		}
	}

	backups, err := ListBackups(cfg.BackupDir)
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one pre-migration backup, got %d: %v", len(backups), err)
	}
}
