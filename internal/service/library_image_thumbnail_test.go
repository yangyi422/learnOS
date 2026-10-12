package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"learnos/internal/librarymedia"
)

func TestLibraryCardThumbnailHistoricalFormatsAndConcurrentCache(t *testing.T) {
	images, _, _, cfg := imageTestServices(t)
	ctx := context.Background()
	for _, format := range []string{"png", "jpeg", "webp"} {
		original := syntheticImage(t, format)
		row, err := images.Upload(ctx, 1, "test", bytes.NewReader(original))
		if err != nil {
			t.Fatal(err)
		}
		// A new service instance must handle images uploaded before this feature.
		fresh := NewLibraryImageService(images.repo, cfg.DatabasePath, 0)
		if _, err = fresh.Metadata(ctx, 2, row.ID); err == nil {
			t.Fatal("foreign metadata exposed")
		}
		data, err := fresh.ResponseBytes(ctx, row, LibraryCardVariant)
		if err != nil {
			t.Fatal(err)
		}
		decoded, kind, err := image.Decode(bytes.NewReader(data))
		if err != nil || kind != "jpeg" || decoded.Bounds().Dx() != 3 || decoded.Bounds().Dy() != 2 {
			t.Fatal("bad rendition", kind, err)
		}
		if _, saved, err := fresh.Read(ctx, 1, row.ID); err != nil || !bytes.Equal(original, saved) {
			t.Fatal("original changed", err)
		}
	}
	source := image.NewRGBA(image.Rect(0, 0, 1200, 900))
	source.Set(1, 1, color.RGBA{R: 255, A: 255})
	var original bytes.Buffer
	if err := png.Encode(&original, source); err != nil {
		t.Fatal(err)
	}
	row, err := images.Upload(ctx, 1, "large.png", bytes.NewReader(original.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, e := images.ResponseBytes(ctx, row, LibraryCardVariant)
			if e != nil {
				t.Error(e)
				return
			}
			decoded, _, e := image.Decode(bytes.NewReader(data))
			if e != nil || decoded.Bounds().Dx() != 640 || decoded.Bounds().Dy() != 480 {
				t.Error("bad thumbnail size", e)
			}
		}()
	}
	wg.Wait()
	entries, err := os.ReadDir(filepath.Dir(images.thumbnailPath(row)))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if entry.Name() == filepath.Base(images.thumbnailPath(row)) {
			count++
		}
		if len(entry.Name()) > 0 && entry.Name()[0] == '.' {
			t.Fatal("unfinished cache file")
		}
	}
	if count != 1 {
		t.Fatal("duplicate cache files", count)
	}
	first, err := os.Stat(images.thumbnailPath(row))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = images.ResponseBytes(ctx, row, LibraryCardVariant); err != nil {
		t.Fatal(err)
	}
	second, _ := os.Stat(images.thumbnailPath(row))
	if !first.ModTime().Equal(second.ModTime()) {
		t.Fatal("cache unnecessarily regenerated")
	}
	if err = os.WriteFile(images.thumbnailPath(row), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = images.ResponseBytes(ctx, row, LibraryCardVariant); err != nil {
		t.Fatal("bad derived cache was not regenerated", err)
	}
	if _, err = images.ResponseBytes(ctx, row, "arbitrary-size"); err == nil {
		t.Fatal("unbounded variant accepted")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	os.Remove(images.thumbnailPath(row))
	if _, err = images.ResponseBytes(cancelled, row, LibraryCardVariant); err == nil {
		t.Fatal("cancelled generation")
	}
	if _, err = os.Stat(images.thumbnailPath(row)); !os.IsNotExist(err) {
		t.Fatal("cancelled cache was committed")
	}
	// Even a cached image cannot validate against a missing original.
	os.Remove(filepath.Join(librarymedia.Directory(cfg.DatabasePath), row.StorageName))
	if _, err = images.Metadata(ctx, 1, row.ID); err == nil {
		t.Fatal("missing original validated")
	}
}

func TestLibraryCardThumbnailCacheBudgetAndCleanup(t *testing.T) {
	images, _, db, _ := imageTestServices(t)
	ctx := context.Background()
	row, err := images.Upload(ctx, 1, "test.png", bytes.NewReader(syntheticImage(t, "png")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = images.ResponseBytes(ctx, row, LibraryCardVariant); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(images.thumbnailPath(row))
	images.quota = 1 // Still permits one bounded rendition; never touches originals.
	other := filepath.Join(dir, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb-"+row.SHA256+".jpg")
	if err = os.WriteFile(other, make([]byte, thumbnailMaxBytes), 0600); err != nil {
		t.Fatal(err)
	}
	if err = images.trimThumbnailCache(dir, thumbnailMaxBytes); err != nil {
		t.Fatal(err)
	}
	if _, _, err = images.Read(ctx, 1, row.ID); err != nil {
		t.Fatal("cache pruning touched original", err)
	}
	if _, err = images.ResponseBytes(ctx, row, LibraryCardVariant); err != nil {
		t.Fatal(err)
	}
	if err = db.Exec("UPDATE library_images SET created_at = datetime('now', '-8 days') WHERE id = ?", row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if count, err := images.Cleanup(ctx, 1); err != nil || count != 1 {
		t.Fatal("cleanup", count, err)
	}
	if _, err = os.Stat(images.thumbnailPath(row)); !os.IsNotExist(err) {
		t.Fatal("orphan thumbnail retained")
	}
}

func TestLibraryCardThumbnailRetainsJPEGOrientation(t *testing.T) {
	images, _, _, _ := imageTestServices(t)
	for orientation := uint16(2); orientation <= 8; orientation++ {
		original := retainJPEGOrientation(syntheticImage(t, "jpeg"), orientation)
		row, err := images.Upload(context.Background(), 1, "oriented.jpg", bytes.NewReader(original))
		if err != nil {
			t.Fatal(err)
		}
		thumbnail, err := images.ResponseBytes(context.Background(), row, LibraryCardVariant)
		if err != nil {
			t.Fatal(err)
		}
		if jpegOrientation(thumbnail) != orientation {
			t.Fatal("JPEG direction lost", orientation)
		}
		if _, _, err = image.Decode(bytes.NewReader(thumbnail)); err != nil {
			t.Fatal("invalid EXIF rendition", err)
		}
	}
	for _, data := range [][]byte{nil, {0xff, 0xd8, 0xff}, {0xff, 0xd8, 0xff, 0xe1, 0xff, 0xff}, []byte("Exif\x00\x00"), syntheticImage(t, "png")} {
		if jpegOrientation(data) != 1 {
			t.Fatal("unsafe orientation parsing")
		}
	}
}
