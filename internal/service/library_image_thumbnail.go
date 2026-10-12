package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/image/draw"
	"learnos/internal/librarymedia"
	"learnos/internal/model"
)

// One fixed, versioned rendition bounds both disk usage and decoding work.
// Originals remain immutable; this cache can be removed and regenerated.
const LibraryCardVariant = "card-v1"
const thumbnailMaxBytes = 512 << 10

func (s *LibraryImageService) thumbnailPath(row model.LibraryImage) string {
	return filepath.Join(s.dir, "thumbnails", LibraryCardVariant, row.ID+"-"+row.SHA256+".jpg")
}

// Metadata always checks ownership and the original's presence before an HTTP
// cache validator can return 304. It neither reads nor hashes image contents.
func (s *LibraryImageService) Metadata(ctx context.Context, user uint, id string) (model.LibraryImage, error) {
	librarymedia.Mu.RLock()
	defer librarymedia.Mu.RUnlock()
	if !librarymedia.IDPattern.MatchString(id) {
		return model.LibraryImage{}, ErrImageMissing
	}
	row, err := s.repo.Get(ctx, user, id)
	if err != nil {
		return row, err
	}
	if !librarymedia.FilePattern.MatchString(row.StorageName) {
		return row, ErrImageMissing
	}
	info, err := os.Lstat(filepath.Join(s.dir, row.StorageName))
	if err != nil || !info.Mode().IsRegular() || info.Size() != row.Size {
		return row, ErrImageMissing
	}
	return row, nil
}

func readThumbnail(path string) ([]byte, error) {
	for _, directory := range []string{filepath.Dir(filepath.Dir(path)), filepath.Dir(path)} {
		info, err := os.Lstat(directory)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, ErrImageMissing
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > thumbnailMaxBytes {
		return nil, ErrImageMissing
	}
	data, err := os.ReadFile(path)
	if err == nil && (len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 || data[len(data)-2] != 0xff || data[len(data)-1] != 0xd9) {
		err = ErrImageMissing
	}
	return data, err
}

// ResponseBytes uses the already-authorized metadata (no second DB query).
// Only a cache miss decodes the bounded original. Concurrent misses recheck
// under a context-aware semaphore; they never create duplicate renditions.
func (s *LibraryImageService) ResponseBytes(ctx context.Context, row model.LibraryImage, variant string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if variant == "" {
		librarymedia.Mu.RLock()
		defer librarymedia.Mu.RUnlock()
		data, err := librarymedia.Read(s.dir, row.StorageName)
		if err != nil || int64(len(data)) != row.Size || librarymedia.Hash(data) != row.SHA256 {
			return nil, ErrImageMissing
		}
		return data, nil
	}
	if variant != LibraryCardVariant {
		return nil, ErrLibraryInput
	}
	path := s.thumbnailPath(row)
	librarymedia.Mu.RLock()
	cached, cacheErr := readThumbnail(path)
	librarymedia.Mu.RUnlock()
	if cacheErr == nil {
		return cached, nil
	}
	select {
	case s.thumbnailGate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-s.thumbnailGate }()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	librarymedia.Mu.RLock()
	defer librarymedia.Mu.RUnlock()
	if data, err := readThumbnail(path); err == nil {
		return data, nil
	}
	data, err := librarymedia.Read(s.dir, row.StorageName)
	if err != nil || int64(len(data)) != row.Size || librarymedia.Hash(data) != row.SHA256 {
		return nil, ErrImageMissing
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 12000 || cfg.Height > 12000 || int64(cfg.Width)*int64(cfg.Height) > librarymedia.MaxPixels {
		return nil, ErrImageMissing
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrImageMissing
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	width, height := cfg.Width, cfg.Height
	if width > 640 || height > 640 {
		if width >= height {
			height = max(1, height*640/width)
			width = 640
		} else {
			width = max(1, width*640/height)
			height = 640
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	// White backdrop for transparent PNGs; originals retain their transparency.
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	var out bytes.Buffer
	if err = jpeg.Encode(&out, dst, &jpeg.Options{Quality: 80}); err != nil {
		return nil, err
	}
	rendered := retainJPEGOrientation(out.Bytes(), jpegOrientation(data))
	if len(rendered) > thumbnailMaxBytes {
		return nil, ErrImageSize
	}
	// Refuse symlinked cache directories; original store rules remain unchanged.
	dir := filepath.Dir(path)
	for _, directory := range []string{filepath.Dir(dir), dir} {
		if err = os.Mkdir(directory, 0750); err != nil && !os.IsExist(err) {
			return nil, err
		}
		info, e := os.Lstat(directory)
		if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, ErrImageMissing
		}
	}
	if err = s.trimThumbnailCache(dir, int64(len(rendered))); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(dir, ".pending-")
	if err != nil {
		return nil, err
	}
	temporary := f.Name()
	defer os.Remove(temporary)
	if _, err = f.Write(rendered); err != nil {
		f.Close()
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if err = os.Rename(temporary, path); err != nil {
		return nil, err
	}
	return rendered, nil
}

// Derived files have a separate, bounded budget (at most 256 MiB or a quarter
// of the configured original-image quota). Eviction never touches originals.
func (s *LibraryImageService) trimThumbnailCache(dir string, incoming int64) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type cached struct {
		name string
		info os.FileInfo
	}
	files := []cached{}
	var total int64
	for _, entry := range entries {
		name := entry.Name()
		parts := strings.Split(strings.TrimSuffix(name, ".jpg"), "-")
		if !strings.HasSuffix(name, ".jpg") || len(parts) != 2 || !librarymedia.IDPattern.MatchString(parts[0]) || len(parts[1]) != 64 {
			continue
		}
		info, e := entry.Info()
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			continue
		}
		total += info.Size()
		files = append(files, cached{name, info})
	}
	budget := min(int64(256<<20), max(int64(thumbnailMaxBytes), s.quota/4))
	sort.Slice(files, func(i, j int) bool { return files[i].info.ModTime().Before(files[j].info.ModTime()) })
	for _, file := range files {
		if total+incoming <= budget {
			break
		}
		if err = os.Remove(filepath.Join(dir, file.name)); err != nil {
			return err
		}
		total -= file.info.Size()
	}
	return nil
}
