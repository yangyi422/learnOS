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

func TestLibrarySchema26RepeatMigrationAndBackup(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{Environment: "production", DatabasePath: filepath.Join(dir, "library.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	old, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	old.AutoMigrate(&model.SystemMetadata{}, &model.LifeEvent{}, &model.FinanceLedger{}, &model.FinanceTransaction{})
	old.Create(&model.SystemMetadata{ID: 1, SchemaVersion: 25})
	old.Create(&model.LifeEvent{UserID: 1, Title: "已有经历", OccurredOn: "2026-01-01"})
	ledger := model.FinanceLedger{UserID: 1, Name: "保留账本"}
	old.Create(&ledger)
	old.Create(&model.FinanceTransaction{LedgerID: ledger.ID, ExternalID: "preserved", AmountMinor: 1234})
	conn, _ := old.DB()
	conn.Close()
	db, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasTable(&model.LibraryCollection{}) || !db.Migrator().HasTable(&model.LibrarySection{}) || !db.Migrator().HasTable(&model.LibraryPage{}) {
		t.Fatal("missing tables")
	}
	backups, err := ListBackups(cfg.BackupDir)
	if err != nil || len(backups) != 1 {
		t.Fatal(backups, err)
	}
	collection := model.LibraryCollection{UserID: 1, Name: "专题", Template: "wiki", DisplayMode: "wiki", Version: 1}
	if err = db.Create(&collection).Error; err != nil {
		t.Fatal(err)
	}
	page := model.LibraryPage{CollectionID: collection.ID, Title: "页面", Body: "正文", Version: 1}
	if err = db.Omit("Collection", "Section").Create(&page).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = CreateBackup(context.Background(), db, cfg.BackupDir, 5); err != nil {
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
	var saved model.LibraryPage
	db.First(&saved, page.ID)
	var tx model.FinanceTransaction
	db.First(&tx, 1)
	var event model.LifeEvent
	db.First(&event, 1)
	if saved.Body != "正文" || tx.AmountMinor != 1234 || event.Title != "已有经历" {
		t.Fatal("migration changed content")
	}
	var meta model.SystemMetadata
	db.First(&meta, 1)
	if meta.SchemaVersion != model.CurrentSchemaVersion {
		t.Fatal(meta)
	}
}
