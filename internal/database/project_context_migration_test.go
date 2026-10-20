package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"learnos/internal/config"
	"learnos/internal/model"
	"path/filepath"
	"testing"
)

func TestProjectContextMigrationPreservesFactsAndBackup(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{Environment: "production", DatabasePath: filepath.Join(dir, "legacy.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	old, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = old.Exec(`CREATE TABLE projects (id integer PRIMARY KEY, user_id integer NOT NULL, title text NOT NULL, status text NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	if err = old.AutoMigrate(&model.ProjectTask{}, &model.SystemMetadata{}); err != nil {
		t.Fatal(err)
	}
	old.Create(&model.SystemMetadata{ID: 1, SchemaVersion: 28})
	old.Exec(`INSERT INTO projects(id,user_id,title,status) VALUES(71,1,'工作也不猜测','active')`)
	if err = old.Exec(`INSERT INTO project_tasks(id,project_id,title,status,sort_order,priority,completed_at,actual_completed_at) VALUES(81,71,'保留','done',3072,'high','2026-10-19 12:00:00','2026-10-19 12:00:00')`).Error; err != nil {
		t.Fatal(err)
	}
	conn, _ := old.DB()
	conn.Close()
	for i := 0; i < 2; i++ {
		db, e := Open(cfg)
		if e != nil {
			t.Fatal(e)
		}
		var p model.Project
		if e = db.First(&p, 71).Error; e != nil || p.Context != "unclassified" {
			t.Fatal(p, e)
		}
		var task model.ProjectTask
		if e = db.First(&task, 81).Error; e != nil || task.ProjectID != 71 || task.Status != "done" || task.SortOrder != 3072 || task.CompletedAt == nil || task.ActualCompletedAt == nil {
			t.Fatal(task, e)
		}
		if !db.Migrator().HasIndex(&model.Project{}, "idx_projects_user_context") {
			t.Fatal("missing context index")
		}
		if e = db.Exec(`INSERT INTO projects(user_id,title,status) VALUES(1,'default','active')`).Error; e != nil {
			t.Fatal(e)
		}
		var defaults model.Project
		db.Where("title = ?", "default").First(&defaults)
		if defaults.Context != "unclassified" {
			t.Fatal("missing SQL default")
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
	if copyDB.Migrator().HasColumn("projects", "context") {
		t.Fatal("backup was taken after migration")
	}
	var count int64
	copyDB.Table("project_tasks").Where("id=81 AND sort_order=3072 AND status='done'").Count(&count)
	if count != 1 {
		t.Fatal("backup lost original task")
	}
}
