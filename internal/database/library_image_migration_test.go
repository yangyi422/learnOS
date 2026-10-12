package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"learnos/internal/config"
	"learnos/internal/model"
	"path/filepath"
	"testing"
)

func TestLibraryImagesMigration28PreservesPagesAndRepeats(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{Environment: "production", DatabasePath: filepath.Join(dir, "learnos.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	old, e := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	if e = old.AutoMigrate(&model.SystemMetadata{}, &model.LibraryCollection{}, &model.LibrarySection{}, &model.LibraryPage{}); e != nil {
		t.Fatal(e)
	}
	old.Create(&model.SystemMetadata{ID: 1, SchemaVersion: 27})
	col := model.LibraryCollection{UserID: 1, Name: "preserved", Template: "wiki", DisplayMode: "wiki"}
	old.Create(&col)
	page := model.LibraryPage{CollectionID: col.ID, Title: "kept", Body: "external ![](https://example.com/a.png)", Version: 1}
	old.Create(&page)
	conn, _ := old.DB()
	conn.Close()
	for i := 0; i < 2; i++ {
		db, e := Open(cfg)
		if e != nil {
			t.Fatal(e)
		}
		if !db.Migrator().HasTable(&model.LibraryImage{}) {
			t.Fatal("missing image table")
		}
		var saved model.LibraryPage
		db.First(&saved, page.ID)
		if saved.Body != page.Body || saved.Version != page.Version {
			t.Fatal("migration changed page")
		}
		conn, _ = db.DB()
		conn.Close()
	}
	backups, e := ListBackups(cfg.BackupDir)
	if e != nil || len(backups) != 1 || backups[0].Scope != "database_only" {
		t.Fatal(backups, e)
	}
}
