package database

import (
	"context"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"learnos/internal/config"
	"learnos/internal/model"
	"path/filepath"
	"testing"
)

func TestLifeSchema21MigrationAndBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "life.db")
	legacy, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = legacy.AutoMigrate(&model.SystemMetadata{}, &model.InboxItem{}, &model.LightweightRecord{}); err != nil {
		t.Fatal(err)
	}
	legacy.Create(&model.SystemMetadata{ID: 1, SchemaVersion: 20})
	legacy.Create(&model.InboxItem{UserID: 1, Content: "旧收集", Status: "inbox", SourceType: "manual"})
	legacy.Create(&model.LightweightRecord{UserID: 1, Content: "旧记录"})
	sql, _ := legacy.DB()
	sql.Close()
	cfg := config.Config{DatabasePath: path, BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	db, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var inbox model.InboxItem
	var record model.LightweightRecord
	db.First(&inbox, 1)
	db.First(&record, 1)
	if inbox.Content != "旧收集" || record.Content != "旧记录" {
		t.Fatal(inbox, record)
	}
	goal := model.LifeGoal{UserID: 1, Title: "目标", Status: "active"}
	if err = db.Create(&goal).Error; err != nil {
		t.Fatal(err)
	}
	event := model.LifeEvent{UserID: 1, Title: "过去事件", OccurredOn: "2000-01-02", GoalID: &goal.ID, Milestone: true}
	if err = db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	entry := model.LifeGoalEntry{GoalID: goal.ID, OccurredOn: "2026-10-08", Content: "阶段记录"}
	db.Create(&entry)
	backup, err := CreateBackup(context.Background(), db, cfg.BackupDir, 5)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	conn.Close()
	db, err = Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ = db.DB()
	defer conn.Close()
	var reloaded model.LifeEvent
	if err = db.First(&reloaded, event.ID).Error; err != nil || reloaded.OccurredOn != "2000-01-02" {
		t.Fatal(reloaded, err)
	}
	archived, err := gorm.Open(sqlite.Open(backup.Path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	backupConn, _ := archived.DB()
	defer backupConn.Close()
	for _, table := range []any{&model.LifeEvent{}, &model.LifeGoal{}, &model.LifeGoalEntry{}} {
		var count int64
		if err = archived.Model(table).Count(&count).Error; err != nil || count != 1 {
			t.Fatal(table, count, err)
		}
	}
	if !db.Migrator().HasIndex(&model.LifeEvent{}, "idx_life_event_source") || !db.Migrator().HasIndex(&model.LifeGoalEntry{}, "idx_life_entry_key") {
		t.Fatal("missing identity constraints")
	}
	var metadata model.SystemMetadata
	db.First(&metadata, 1)
	if metadata.SchemaVersion != 21 {
		t.Fatal(metadata)
	}
}
