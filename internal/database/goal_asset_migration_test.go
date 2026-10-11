package database

import (
	"context"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"learnos/internal/config"
	"learnos/internal/model"
	"path/filepath"
	"testing"
)

func TestGoalAssetSchema24MigrationBackupAndCascade(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{Environment: "production", DatabasePath: filepath.Join(dir, "archive.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	old, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err = old.AutoMigrate(&model.SystemMetadata{}, &model.LifeGoal{}, &model.LifeGoalEntry{}, &model.LifeEvent{}, &model.AssetSnapshot{}, &model.FinanceTransaction{}); err != nil {
		t.Fatal(err)
	}
	old.Create(&model.SystemMetadata{ID: 1, SchemaVersion: 23})
	goal := model.LifeGoal{UserID: 1, Title: "kept"}
	old.Create(&goal)
	n := int64(0)
	asset := model.AssetSnapshot{UserID: 1, SnapshotDate: "2026-01-01", BankMinor: &n, Version: 1}
	old.Create(&asset)
	old.Create(&model.LifeGoalEntry{GoalID: goal.ID, OccurredOn: "2026-01-01", Content: "kept"})
	old.Create(&model.FinanceTransaction{LedgerID: 1, ExternalID: "kept", AmountMinor: 123})
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
	ref := model.GoalAssetReference{GoalID: goal.ID, AssetSnapshotID: asset.ID, JudgmentDate: "2026-10-10", Note: "context", Version: 1}
	if err = db.Omit("Goal", "Snapshot").Create(&ref).Error; err != nil {
		t.Fatal(err)
	}
	backup, err := CreateBackup(context.Background(), db, cfg.BackupDir, 5)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ = db.DB()
	conn.Close()
	db, err = Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ = db.DB()
	defer conn.Close()
	for _, m := range []interface{}{&model.LifeGoal{}, &model.LifeGoalEntry{}, &model.AssetSnapshot{}, &model.FinanceTransaction{}, &model.GoalAssetReference{}} {
		var count int64
		if err = db.Model(m).Count(&count).Error; err != nil || count != 1 {
			t.Fatal(m, count, err)
		}
	}
	saved, err := gorm.Open(sqlite.Open(backup.Path), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := saved.DB()
	defer sc.Close()
	var archived model.GoalAssetReference
	if err = saved.First(&archived, ref.ID).Error; err != nil || archived.Note != ref.Note {
		t.Fatal(archived, err)
	}
	if err = db.Delete(&model.LifeGoal{}, goal.ID).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.GoalAssetReference{}).Count(&count)
	if count != 0 {
		t.Fatal("dangling reference", count)
	}
	if err = db.First(&asset, asset.ID).Error; err != nil {
		t.Fatal("asset deleted", err)
	}
}
