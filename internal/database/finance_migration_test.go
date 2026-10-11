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

func TestFinanceSchema22MigrationAndBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "finance.db")
	old, e := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Discard})
	if e != nil {
		t.Fatal(e)
	}
	if e = old.AutoMigrate(&model.SystemMetadata{}, &model.LifeEvent{}, &model.LifeGoal{}, &model.LifeGoalEntry{}); e != nil {
		t.Fatal(e)
	}
	old.Create(&model.SystemMetadata{ID: 1, SchemaVersion: 21})
	old.Create(&model.LifeEvent{UserID: 1, Title: "旧生活经历", OccurredOn: "2000-01-01"})
	conn, _ := old.DB()
	conn.Close()
	cfg := config.Config{Environment: "production", DatabasePath: path, BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	db, e := Open(cfg)
	if e != nil {
		t.Fatal(e)
	}
	var event model.LifeEvent
	db.First(&event, 1)
	if event.Title != "旧生活经历" {
		t.Fatal("lost legacy", event)
	}
	var meta model.SystemMetadata
	db.First(&meta, 1)
	if meta.SchemaVersion != model.CurrentSchemaVersion {
		t.Fatal(meta)
	}
	for _, m := range []interface{}{&model.FinanceLedger{}, &model.FinanceTransaction{}, &model.FinanceBatch{}, &model.FinanceChange{}} {
		if !db.Migrator().HasTable(m) {
			t.Fatal("missing finance table")
		}
	}
	ledger := model.FinanceLedger{UserID: 1, Name: "个人"}
	if e = db.Create(&ledger).Error; e != nil {
		t.Fatal(e)
	}
	row := model.FinanceTransaction{LedgerID: ledger.ID, ExternalID: "id-a", AmountMinor: 1234, OccurredAt: "2026-01-01 00:00:00", Currency: "CNY", RawJSON: `{"备注":"original"}`, TagsJSON: `["日常"]`}
	if e = db.Create(&row).Error; e != nil {
		t.Fatal(e)
	}
	batch := model.FinanceBatch{LedgerID: ledger.ID, Status: "imported", RowCount: 1}
	db.Create(&batch)
	db.Create(&model.FinanceChange{BatchID: batch.ID, TransactionID: row.ID, AfterJSON: "snapshot"})
	backup, e := CreateBackup(context.Background(), db, cfg.BackupDir, 5)
	if e != nil {
		t.Fatal(e)
	}
	conn, _ = db.DB()
	conn.Close()
	db, e = Open(cfg)
	if e != nil {
		t.Fatal(e)
	}
	conn, _ = db.DB()
	defer conn.Close()
	var count int64
	db.Model(&model.FinanceTransaction{}).Count(&count)
	if count != 1 {
		t.Fatal("repeat migration", count)
	}
	archived, e := gorm.Open(sqlite.Open(backup.Path), &gorm.Config{Logger: logger.Discard})
	if e != nil {
		t.Fatal(e)
	}
	bc, _ := archived.DB()
	defer bc.Close()
	var restored model.FinanceTransaction
	if e = archived.First(&restored, row.ID).Error; e != nil || restored.RawJSON != row.RawJSON || restored.AmountMinor != 1234 {
		t.Fatal("backup incomplete", restored, e)
	}
	archived.Model(&model.FinanceChange{}).Count(&count)
	if count != 1 {
		t.Fatal("missing rollback snapshot", count)
	}
}
