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

func TestAssetSchema23AdditiveMigrationAndSnapshot(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{Environment: "production", DatabasePath: filepath.Join(dir, "assets.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	old, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err = old.AutoMigrate(&model.SystemMetadata{}, &model.FinanceLedger{}, &model.FinanceTransaction{}, &model.FinanceBatch{}, &model.FinanceChange{}, &model.LifeEvent{}, &model.LifeGoal{}); err != nil {
		t.Fatal(err)
	}
	old.Create(&model.SystemMetadata{ID: 1, SchemaVersion: 22})
	old.Create(&model.FinanceLedger{UserID: 1, Name: "legacy"})
	old.Create(&model.FinanceTransaction{LedgerID: 1, ExternalID: "stable", AmountMinor: 1234})
	old.Create(&model.LifeEvent{UserID: 1, OccurredOn: "2025-01-01", Title: "kept event"})
	old.Create(&model.LifeGoal{UserID: 1, Title: "kept goal"})
	conn, _ := old.DB()
	conn.Close()
	db, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	backups, err := ListBackups(cfg.BackupDir)
	if err != nil || len(backups) != 1 {
		t.Fatal("missing pre-migration backup", backups, err)
	}
	var metadata model.SystemMetadata
	db.First(&metadata, 1)
	if metadata.SchemaVersion != model.CurrentSchemaVersion {
		t.Fatal(metadata)
	}
	zero := int64(0)
	asset := model.AssetSnapshot{UserID: 1, SnapshotDate: "2025-12-31", BankMinor: &zero, Version: 1, Note: "backup note"}
	if err = db.Create(&asset).Error; err != nil {
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
	for _, m := range []interface{}{&model.AssetSnapshot{}, &model.FinanceTransaction{}, &model.LifeEvent{}, &model.LifeGoal{}} {
		var count int64
		db.Model(m).Count(&count)
		if count != 1 {
			t.Fatal("repeat migration changed history", m, count)
		}
	}
	snapshot, err := gorm.Open(sqlite.Open(backup.Path), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := snapshot.DB()
	defer sc.Close()
	var archived model.AssetSnapshot
	if err = snapshot.First(&archived, asset.ID).Error; err != nil || archived.BankMinor == nil || *archived.BankMinor != 0 || archived.DebtMinor != nil || archived.Note != asset.Note {
		t.Fatal("backup lost zero/null", archived, err)
	}
}
