package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"learnos/internal/config"
	"learnos/internal/database"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackupServiceCreatesRetainsAndValidatesSQLiteBackup(t *testing.T) {
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "learnos.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE facts (id INTEGER PRIMARY KEY, value TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO facts(value) VALUES (?)", "kept").Error; err != nil {
		t.Fatal(err)
	}
	svc := NewBackupService(db, config.Config{BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 1})
	first, err := svc.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ValidateBackup(first.Path); err != nil {
		t.Fatalf("backup should validate: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	second, err := svc.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Path == second.Path {
		t.Fatal("backups created in one second must still have unique paths")
	}
	items, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Path != second.Path {
		t.Fatalf("retention did not keep newest backup: %+v", items)
	}
}

func TestConfirmedRestoreRequestCreatesSafetyBackupAndRestoresOnRestart(t *testing.T) {
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "learnos.db")
	backupDir := filepath.Join(dir, "backups")
	cfg := config.Config{DatabasePath: databasePath, BackupDir: backupDir, BackupRetentionCount: 10}
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE facts (id INTEGER PRIMARY KEY, value TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO facts(value) VALUES (?)", "before").Error; err != nil {
		t.Fatal(err)
	}
	service := NewBackupService(db, cfg)
	backup, err := service.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RequestRestore(context.Background(), backup.Name, "wrong confirmation"); !errors.Is(err, ErrRestoreConfirmationFailed) {
		t.Fatalf("expected confirmation failure, got %v", err)
	}
	if err := db.Exec("UPDATE facts SET value = ?", "after").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.RequestRestore(context.Background(), backup.Name, "恢复 "+backup.Name); err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	restored, err := ApplyPendingRestore(cfg)
	if err != nil || !restored {
		t.Fatalf("apply pending restore: restored=%v err=%v", restored, err)
	}
	checked, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var value string
	if err := checked.Raw("SELECT value FROM facts LIMIT 1").Scan(&value).Error; err != nil || value != "before" {
		t.Fatalf("unexpected restored value %q err=%v", value, err)
	}
	items, err := database.ListBackups(backupDir)
	if err != nil || len(items) < 2 {
		t.Fatalf("expected pre-restore safety backup, items=%+v err=%v", items, err)
	}
}

func TestPendingRestoreKeepsSelectedSnapshotWhenRetentionPrunesOriginal(t *testing.T) {
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "learnos.db")
	backupDir := filepath.Join(dir, "backups")
	cfg := config.Config{DatabasePath: databasePath, BackupDir: backupDir, BackupRetentionCount: 1}
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE facts (id INTEGER PRIMARY KEY, value TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO facts(value) VALUES (?)", "selected").Error; err != nil {
		t.Fatal(err)
	}
	svc := NewBackupService(db, cfg)
	selected, err := svc.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RequestRestore(context.Background(), selected.Name, "恢复 "+selected.Name); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE facts SET value = ?", "newer").Error; err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := svc.Create(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(selected.Path); !os.IsNotExist(err) {
		t.Fatalf("expected retention to prune selected source, stat err=%v", err)
	}
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	if restored, err := ApplyPendingRestore(cfg); err != nil || !restored {
		t.Fatalf("apply staged restore: restored=%v err=%v", restored, err)
	}
	checked, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var value string
	if err := checked.Raw("SELECT value FROM facts LIMIT 1").Scan(&value).Error; err != nil || value != "selected" {
		t.Fatalf("unexpected restored value %q err=%v", value, err)
	}
}
