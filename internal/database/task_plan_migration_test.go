package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"learnos/internal/config"
	"learnos/internal/model"
	"path/filepath"
	"testing"
	"time"
)

func TestTaskPlanMigrationPreservesHistoryAndBackup(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{Environment: "production", DatabasePath: filepath.Join(dir, "legacy.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	old, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = old.AutoMigrate(&model.Project{}, &model.ProjectTask{}, &model.SystemMetadata{}); err != nil {
		t.Fatal(err)
	}
	if err = old.Migrator().DropColumn(&model.ProjectTask{}, "plan_date"); err != nil {
		t.Fatal(err)
	}
	old.Create(&model.SystemMetadata{ID: 1, SchemaVersion: 29})
	old.Create(&model.Project{ID: 1, UserID: 1, Title: "work", Status: "active", Context: "work"})
	if err = old.Exec(`INSERT INTO project_tasks(id,project_id,title,status,sort_order,priority,due_date,completed_at,actual_completed_at,archived_at,updated_at) VALUES(81,1,'kept','done',3072,'high','2026-11-01','2026-10-19 12:00:00','2026-10-19 12:00:00','2026-10-20 12:00:00','2026-10-20 12:00:00')`).Error; err != nil {
		t.Fatal(err)
	}
	conn, _ := old.DB()
	conn.Close()
	for i := 0; i < 2; i++ {
		db, e := Open(cfg)
		if e != nil {
			t.Fatal(e)
		}
		var task model.ProjectTask
		if e = db.First(&task, 81).Error; e != nil {
			t.Fatal(e)
		}
		if task.PlanDate != nil || task.ProjectID != 1 || task.Status != "done" || task.SortOrder != 3072 || task.ArchivedAt == nil || task.CompletedAt == nil || task.ActualCompletedAt == nil || *task.DueDate != "2026-11-01" || !task.CompletedAt.Equal(time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)) {
			t.Fatalf("lost facts %+v", task)
		}
		if !db.Migrator().HasIndex(&model.ProjectTask{}, "idx_project_tasks_plan_date") {
			t.Fatal("missing plan index")
		}
		if e = db.Exec(`INSERT INTO project_tasks(project_id,title,status,priority) VALUES(1,'default','inbox','normal')`).Error; e != nil {
			t.Fatal(e)
		}
		var row model.ProjectTask
		db.Where("title = ?", "default").First(&row)
		if row.PlanDate != nil {
			t.Fatal("non-null default")
		}
		if i == 0 {
			if e = db.Exec(`INSERT INTO project_tasks(id,project_id,title,status,priority,plan_date) VALUES(181,1,'new plan','inbox','normal','2026-10-22')`).Error; e != nil {
				t.Fatal(e)
			}
		} else {
			var scheduled model.ProjectTask
			if e = db.First(&scheduled, 181).Error; e != nil || scheduled.PlanDate == nil || *scheduled.PlanDate != "2026-10-22" {
				t.Fatal("repeat migration changed plan", scheduled, e)
			}
		}
		conn, _ = db.DB()
		conn.Close()
	}
	backups, err := ListBackups(cfg.BackupDir)
	if err != nil || len(backups) != 1 {
		t.Fatal(backups, err)
	}
	copyDB, err := gorm.Open(sqlite.Open(backups[0].Path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { conn, _ := copyDB.DB(); conn.Close() }()
	if copyDB.Migrator().HasColumn("project_tasks", "plan_date") {
		t.Fatal("backup after migration")
	}
}
