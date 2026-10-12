package service

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"learnos/internal/database"
	"learnos/internal/librarymedia"
	"learnos/internal/model"
	"learnos/internal/repository"
)

func TestFullBackupRestoreImagesAndLegacyDatabaseCompatibility(t *testing.T) {
	images, library, db, cfg := imageTestServices(t)
	ctx := context.Background()
	row, e := images.Upload(ctx, 1, "image.png", bytes.NewReader(syntheticImage(t, "png")))
	if e != nil {
		t.Fatal(e)
	}
	col, e := library.SaveCollection(ctx, 1, 0, libraryCollectionInput("kept", "wiki"))
	if e != nil {
		t.Fatal(e)
	}
	page, e := library.SavePage(ctx, 1, col.ID, 0, LibraryPageInput{Title: "kept", Body: "![image](" + row.URL + ")"})
	if e != nil {
		t.Fatal(e)
	}
	svc := NewBackupService(db, cfg)
	backup, e := svc.Create(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if backup.Scope != database.FullBackupScope || !strings.HasSuffix(backup.Name, ".zip") {
		t.Fatal(backup)
	}
	if e = svc.ValidateBackup(backup.Path); e != nil {
		t.Fatal(e)
	}
	db.Model(&model.LibraryPage{}).Where("id = ?", page.ID).Update("title", "changed")
	if _, e = svc.RequestRestore(ctx, backup.Name, "恢复 "+backup.Name); e != nil {
		t.Fatal(e)
	}
	conn, _ := db.DB()
	conn.Close()
	// Simulate migration to a new server: move bytes away, then restore from ZIP.
	originalDir := librarymedia.Directory(cfg.DatabasePath)
	if e = os.Rename(originalDir, originalDir+"-held"); e != nil {
		t.Fatal(e)
	}
	if restored, e := ApplyPendingRestore(cfg); e != nil || !restored {
		t.Fatal(restored, e)
	}
	restored, e := database.Open(cfg)
	if e != nil {
		t.Fatal(e)
	}
	conn, _ = restored.DB()
	defer conn.Close()
	check := NewLibraryImageService(repository.NewLibraryImageRepository(restored), cfg.DatabasePath, 0)
	_, data, e := check.Read(ctx, 1, row.ID)
	if e != nil || !bytes.Equal(data, syntheticImage(t, "png")) {
		t.Fatal("restored image missing", e)
	}
	var saved model.LibraryPage
	restored.First(&saved, page.ID)
	if saved.Title != "kept" || !strings.Contains(saved.Body, row.URL) {
		t.Fatal(saved)
	}
	old, e := database.CreateBackup(ctx, restored, cfg.BackupDir, 20)
	if e != nil {
		t.Fatal(e)
	}
	if old.Scope != "database_only" {
		t.Fatal(old)
	}
	conn.Close()
	if _, e = RestoreWithMedia(cfg.DatabasePath, old.Path, cfg.BackupDir, 20); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(originalDir, row.StorageName)); e != nil {
		t.Fatal("legacy restore removed images", e)
	}
}
func TestFullRestoreRejectsTraversalCorruptionAndCollisionWithoutClearingImages(t *testing.T) {
	images, _, db, cfg := imageTestServices(t)
	ctx := context.Background()
	row, e := images.Upload(ctx, 1, "image.png", bytes.NewReader(syntheticImage(t, "png")))
	if e != nil {
		t.Fatal(e)
	}
	backup, e := NewBackupService(db, cfg).Create(ctx)
	if e != nil {
		t.Fatal(e)
	}
	mutate := func(name string, transform func(string, []byte) (string, []byte)) string {
		t.Helper()
		source, e := zip.OpenReader(backup.Path)
		if e != nil {
			t.Fatal(e)
		}
		defer source.Close()
		path := filepath.Join(t.TempDir(), name+".zip")
		out, e := os.Create(path)
		if e != nil {
			t.Fatal(e)
		}
		z := zip.NewWriter(out)
		for _, f := range source.File {
			r, _ := f.Open()
			body, _ := io.ReadAll(r)
			r.Close()
			n, b := transform(f.Name, body)
			w, _ := z.Create(n)
			w.Write(b)
		}
		z.Close()
		out.Close()
		return path
	}
	traversal := mutate("traversal", func(n string, b []byte) (string, []byte) {
		if strings.HasPrefix(n, "assets/") {
			n = "../outside.png"
		}
		return n, b
	})
	corrupt := mutate("corrupt", func(n string, b []byte) (string, []byte) {
		if strings.HasPrefix(n, "assets/") {
			b = []byte("broken")
		}
		return n, b
	})
	for _, path := range []string{traversal, corrupt} {
		if _, e = RestoreWithMedia(cfg.DatabasePath, path, cfg.BackupDir, 20); e == nil {
			t.Fatal("invalid restore accepted")
		}
		if _, b, e := images.Read(ctx, 1, row.ID); e != nil || !bytes.Equal(b, syntheticImage(t, "png")) {
			t.Fatal("failed restore lost image", e)
		}
	}
	// A conflicting existing ID must not be silently overwritten.
	conn, _ := db.DB()
	conn.Close()
	path := filepath.Join(librarymedia.Directory(cfg.DatabasePath), row.StorageName)
	os.WriteFile(path, []byte("conflict"), 0600)
	if _, e = RestoreWithMedia(cfg.DatabasePath, backup.Path, cfg.BackupDir, 20); e == nil {
		t.Fatal("image collision overwritten")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "conflict" {
		t.Fatal("collision changed")
	}
	reopened, e := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	conn, _ = reopened.DB()
	conn.Close()
}
