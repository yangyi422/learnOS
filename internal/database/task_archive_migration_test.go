package database

import (
	"context"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"learnos/internal/config"
	"learnos/internal/model"
	"learnos/internal/repository"
	"path/filepath"
	"testing"
)

func TestTaskArchiveMigrationPersistenceAndBackup(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{Environment: "production", DatabasePath: filepath.Join(dir, "tasks.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	old, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// Schema 26 task shape: nullable archive column must be added without backfill.
	if err = old.Exec("CREATE TABLE project_tasks (id integer PRIMARY KEY, project_id integer NOT NULL, title text NOT NULL, status text NOT NULL, completed_at datetime, actual_completed_at datetime)").Error; err != nil {
		t.Fatal(err)
	}
	old.AutoMigrate(&model.Project{}, &model.SystemMetadata{})
	old.Create(&model.SystemMetadata{ID: 1, SchemaVersion: 26})
	old.Create(&model.Project{ID: 1, UserID: 1, Title: "keep", Status: "active"})
	old.Exec("INSERT INTO project_tasks (id,project_id,title,status,completed_at,actual_completed_at) VALUES (1,1,'kept','done','2026-01-15 10:00:00','2026-01-15 10:00:00')")
	conn, _ := old.DB()
	conn.Close()
	db, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	r := repository.NewProjectRepository(db)
	task, err := r.GetTask(context.Background(), 1, 1)
	if err != nil || task.ArchivedAt != nil || task.CompletedAt == nil {
		t.Fatal(task, err)
	}
	completed := *task.CompletedAt
	if err = r.ArchiveTasks(context.Background(), 1, []uint{1}, true); err != nil {
		t.Fatal(err)
	}
	backup, err := CreateBackup(context.Background(), db, cfg.BackupDir, 5)
	if err != nil {
		t.Fatal(err)
	}
	copyDB, err := gorm.Open(sqlite.Open(backup.Path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var copy model.ProjectTask
	copyDB.First(&copy, 1)
	if copy.ArchivedAt == nil {
		t.Fatal("backup lost archive")
	}
	conn, _ = copyDB.DB()
	conn.Close()
	conn, _ = db.DB()
	conn.Close()
	db, err = Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ = db.DB()
	defer conn.Close()
	r = repository.NewProjectRepository(db)
	task, err = r.GetTask(context.Background(), 1, 1)
	if err != nil || task.ArchivedAt == nil || !task.CompletedAt.Equal(completed) || task.Title != "kept" {
		t.Fatal(task, err)
	}
}
