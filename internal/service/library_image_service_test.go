package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/librarymedia"
	"learnos/internal/model"
	"learnos/internal/repository"
)

func syntheticImage(t *testing.T, format string) []byte {
	t.Helper()
	if format == "webp" {
		data, e := os.ReadFile("testdata/synthetic.webp")
		if e != nil {
			t.Fatal(e)
		}
		return data
	}
	im := image.NewRGBA(image.Rect(0, 0, 3, 2))
	im.Set(0, 0, color.RGBA{R: 25, G: 110, B: 230, A: 255})
	var b bytes.Buffer
	var e error
	if format == "png" {
		e = png.Encode(&b, im)
	} else {
		e = jpeg.Encode(&b, im, nil)
	}
	if e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func imageTestServices(t *testing.T) (*LibraryImageService, *LibraryService, *gorm.DB, config.Config) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Config{Environment: "production", DatabasePath: filepath.Join(dir, "learnos.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 10}
	db, e := database.Open(cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { conn, _ := db.DB(); conn.Close() })
	images := NewLibraryImageService(repository.NewLibraryImageRepository(db), cfg.DatabasePath, 2<<30)
	return images, NewLibraryService(repository.NewLibraryRepository(db), images), db, cfg
}
func TestLibraryImageFormatsValidationPersistenceAndIsolation(t *testing.T) {
	images, _, db, cfg := imageTestServices(t)
	ctx := context.Background()
	for _, format := range []string{"png", "jpeg", "webp"} {
		data := syntheticImage(t, format)
		row, e := images.Upload(ctx, 1, "../../unsafe\\photo.svg", bytes.NewReader(data))
		if e != nil {
			t.Fatal(format, e)
		}
		if row.ID == "" || row.Width != 3 || row.Height != 2 || row.OriginalName != "photo.svg" || row.URL != librarymedia.Prefix+row.ID {
			t.Fatal(row)
		}
		if _, _, e = images.Read(ctx, 2, row.ID); e == nil {
			t.Fatal("foreign image exposed")
		}
		if _, _, e = images.Read(ctx, 1, "../../learnos.db"); e == nil {
			t.Fatal("path traversal")
		}
		readService := NewLibraryImageService(repository.NewLibraryImageRepository(db), cfg.DatabasePath, 0)
		_, saved, e := readService.Read(ctx, 1, row.ID)
		if e != nil || !bytes.Equal(saved, data) {
			t.Fatal("refresh lost bytes", e)
		}
	}
	for _, data := range [][]byte{[]byte("<svg xmlns='http://www.w3.org/2000/svg'></svg>"), []byte("<?php system($_GET['x']); ?>"), syntheticImage(t, "png")[:25]} {
		if _, e := images.Upload(ctx, 1, "fake.png", bytes.NewReader(data)); !errors.Is(e, librarymedia.ErrInvalid) {
			t.Fatal("accepted fake/truncated image", e)
		}
	}
	if _, e := images.Upload(ctx, 1, "huge.png", bytes.NewReader(make([]byte, librarymedia.MaxBytes+1))); !errors.Is(e, ErrImageSize) {
		t.Fatal(e)
	}
	for _, dims := range [][2]uint32{{12001, 2}, {6000, 6000}} {
		large := append([]byte(nil), syntheticImage(t, "png")...)
		binary.BigEndian.PutUint32(large[16:20], dims[0])
		binary.BigEndian.PutUint32(large[20:24], dims[1])
		binary.BigEndian.PutUint32(large[29:33], crc32.ChecksumIEEE(large[12:29]))
		if _, e := images.Upload(ctx, 1, "pixels.png", bytes.NewReader(large)); !errors.Is(e, librarymedia.ErrInvalid) {
			t.Fatal("oversized pixels accepted", e)
		}
	}
	smallQuota := NewLibraryImageService(repository.NewLibraryImageRepository(db), cfg.DatabasePath, 1)
	if _, e := smallQuota.Upload(ctx, 1, "full.png", bytes.NewReader(syntheticImage(t, "png"))); !errors.Is(e, ErrImageQuota) {
		t.Fatal(e)
	}
	files, _ := os.ReadDir(librarymedia.Directory(cfg.DatabasePath))
	if len(files) != 3 {
		t.Fatal("failed uploads leaked files", len(files))
	}
}
func TestLibraryImageSharedReferencesExportCleanupAndMissingFile(t *testing.T) {
	images, library, db, cfg := imageTestServices(t)
	ctx := context.Background()
	row, e := images.Upload(ctx, 1, "map.png", bytes.NewReader(syntheticImage(t, "png")))
	if e != nil {
		t.Fatal(e)
	}
	col, e := library.SaveCollection(ctx, 1, 0, libraryCollectionInput("Synthetic", "wiki"))
	if e != nil {
		t.Fatal(e)
	}
	body := "![map](" + row.URL + ")\n![again](" + row.URL + ")\n![external](https://example.com/test.png)"
	page, e := library.SavePage(ctx, 1, col.ID, 0, LibraryPageInput{Title: "Map", Body: body})
	if e != nil {
		t.Fatal(e)
	}
	second, e := library.SavePage(ctx, 1, col.ID, 0, LibraryPageInput{Title: "Shared", Body: body})
	if e != nil {
		t.Fatal(e)
	}
	section, e := library.SaveSection(ctx, 1, col.ID, 0, "Group")
	if e != nil {
		t.Fatal(e)
	}
	page, e = library.SavePage(ctx, 1, col.ID, page.ID, LibraryPageInput{Title: "Renamed", Body: body, SectionID: &section.ID, Version: page.Version})
	if e != nil {
		t.Fatal(e)
	}
	data, e := library.Export(ctx, 1, col.ID)
	if e != nil {
		t.Fatal(e)
	}
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		t.Fatal(e)
	}
	imageCount := 0
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "assets/") {
			imageCount++
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			r.Close()
			if !bytes.Equal(b, syntheticImage(t, "png")) {
				t.Fatal("export image differs")
			}
		}
		if strings.HasPrefix(f.Name, "pages/") {
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			r.Close()
			if !strings.Contains(string(b), "../assets/"+row.StorageName) || strings.Contains(string(b), row.URL) || !strings.Contains(string(b), "https://example.com/test.png") {
				t.Fatal("nonportable export", string(b))
			}
		}
	}
	if imageCount != 1 {
		t.Fatal("duplicate packaged image", imageCount)
	}
	unchanged, _ := library.Page(ctx, 1, col.ID, page.ID)
	if unchanged.Body != body {
		t.Fatal("export mutated source")
	}
	old := time.Now().Add(-8 * 24 * time.Hour)
	db.Model(&model.LibraryImage{}).Where("id = ?", row.ID).Update("created_at", old)
	if e = library.DeletePage(ctx, 1, col.ID, page.ID, page.Version); e != nil {
		t.Fatal(e)
	}
	if n, e := images.Cleanup(ctx, 1); e != nil || n != 0 {
		t.Fatal("deleted shared image", n, e)
	}
	pending, e := images.Upload(ctx, 1, "pending.png", bytes.NewReader(syntheticImage(t, "png")))
	if e != nil {
		t.Fatal(e)
	}
	if n, e := images.Cleanup(ctx, 1); e != nil || n != 0 {
		t.Fatal("deleted fresh pending upload")
	}
	db.Model(&model.LibraryImage{}).Where("id = ?", pending.ID).Update("created_at", old)
	if n, e := images.Cleanup(ctx, 1); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if _, _, e := images.Read(ctx, 1, pending.ID); e == nil {
		t.Fatal("orphan retained")
	}
	if e = os.Remove(filepath.Join(librarymedia.Directory(cfg.DatabasePath), row.StorageName)); e != nil {
		t.Fatal(e)
	}
	if _, e = library.Export(ctx, 1, col.ID); !errors.Is(e, ErrImageMissing) {
		t.Fatal("silent incomplete export", e)
	}
	if _, e = library.SavePage(ctx, 2, col.ID, 0, LibraryPageInput{Title: "Foreign", Body: body}); e == nil {
		t.Fatal("foreign reference accepted")
	}
	_ = second
}
func TestImageDatabaseFailureRemovesUncommittedFile(t *testing.T) {
	images, _, db, cfg := imageTestServices(t)
	if e := db.Callback().Create().Before("gorm:create").Register("fail_image", func(tx *gorm.DB) {
		if tx.Statement.Table == "library_images" {
			tx.AddError(errors.New("synthetic write failure"))
		}
	}); e != nil {
		t.Fatal(e)
	}
	if _, e := images.Upload(context.Background(), 1, "test.png", bytes.NewReader(syntheticImage(t, "png"))); e == nil {
		t.Fatal("upload falsely succeeded")
	}
	files, _ := os.ReadDir(librarymedia.Directory(cfg.DatabasePath))
	if len(files) != 0 {
		t.Fatal("uncommitted file leaked")
	}
}

func TestLibraryPageCoversShareImagesExportAndSurviveBackup(t *testing.T) {
	images, library, db, cfg := imageTestServices(t)
	ctx := context.Background()
	original, e := images.Upload(ctx, 1, "body.png", bytes.NewReader(syntheticImage(t, "png")))
	if e != nil {
		t.Fatal(e)
	}
	cover, e := images.Upload(ctx, 1, "cover.jpg", bytes.NewReader(syntheticImage(t, "jpeg")))
	if e != nil {
		t.Fatal(e)
	}
	col, e := library.SaveCollection(ctx, 1, 0, libraryCollectionInput("cover test", "recipe"))
	if e != nil {
		t.Fatal(e)
	}
	body := "![body](" + original.URL + ")"
	page, e := library.SavePage(ctx, 1, col.ID, 0, LibraryPageInput{Title: "Cover", Body: body, CoverURL: cover.URL})
	if e != nil {
		t.Fatal(e)
	}
	second, e := library.SavePage(ctx, 1, col.ID, 0, LibraryPageInput{Title: "Shared cover", CoverURL: cover.URL})
	if e != nil {
		t.Fatal(e)
	}
	db.Model(&model.LibraryImage{}).Where("id IN ?", []string{original.ID, cover.ID}).Update("created_at", time.Now().Add(-8*24*time.Hour))
	if n, e := images.Cleanup(ctx, 1); e != nil || n != 0 {
		t.Fatal("cover or body reference not protected", n, e)
	}
	if _, e = images.ResponseBytes(ctx, cover, LibraryCardVariant); e != nil {
		t.Fatal(e)
	}
	data, e := library.Export(ctx, 1, col.ID)
	if e != nil {
		t.Fatal(e)
	}
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		t.Fatal(e)
	}
	packaged := map[string][]byte{}
	for _, f := range z.File {
		r, _ := f.Open()
		b, _ := io.ReadAll(r)
		r.Close()
		packaged[f.Name] = b
	}
	if !bytes.Equal(packaged["assets/"+cover.StorageName], syntheticImage(t, "jpeg")) || !strings.Contains(string(packaged[fmt.Sprintf("pages/%d.md", second.ID)]), "![封面](../assets/"+cover.StorageName+")") {
		t.Fatal("cover-only image missing from offline export")
	}
	backup, e := NewBackupService(db, cfg).Create(ctx)
	if e != nil {
		t.Fatal(e)
	}
	stage, e := stageMediaBackup(backup.Path)
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(stage)
	if b, e := os.ReadFile(filepath.Join(stage, "assets", cover.StorageName)); e != nil || !bytes.Equal(b, syntheticImage(t, "jpeg")) {
		t.Fatal("backup lost cover", e)
	}
	snapshot, e := gorm.Open(sqlite.Open(filepath.Join(stage, "learnos.db")), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	var saved model.LibraryPage
	snapshot.First(&saved, second.ID)
	conn, _ := snapshot.DB()
	conn.Close()
	if saved.CoverURL != cover.URL {
		t.Fatal("backup metadata lost cover reference")
	}
	page, e = library.SavePage(ctx, 1, col.ID, page.ID, LibraryPageInput{Title: page.Title, Body: body, CoverURL: original.URL, Version: page.Version})
	if e != nil {
		t.Fatal(e)
	}
	page, e = library.SavePage(ctx, 1, col.ID, page.ID, LibraryPageInput{Title: page.Title, Body: body, Version: page.Version})
	if e != nil {
		t.Fatal(e)
	}
	if page.Body != body || page.CoverURL != "" {
		t.Fatal("clearing cover changed body")
	}
	if n, e := images.Cleanup(ctx, 1); e != nil || n != 0 {
		t.Fatal("clearing/replacing cover deleted shared image", n, e)
	}
	if e = library.DeletePage(ctx, 1, col.ID, second.ID, second.Version); e != nil {
		t.Fatal(e)
	}
	if _, _, e = images.Read(ctx, 1, cover.ID); e != nil {
		t.Fatal("deleting page immediately deleted image", e)
	}
	page, e = library.SavePage(ctx, 1, col.ID, page.ID, LibraryPageInput{Title: page.Title, Body: body, CoverURL: "https://example.com/cover.jpg", Version: page.Version})
	if e != nil || page.CoverURL != "https://example.com/cover.jpg" {
		t.Fatal("external cover rejected", e)
	}
	if _, e = library.SavePage(ctx, 2, col.ID, 0, LibraryPageInput{Title: "foreign", CoverURL: cover.URL}); e == nil {
		t.Fatal("foreign image accepted")
	}
}
