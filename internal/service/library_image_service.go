package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"learnos/internal/librarymedia"
	"learnos/internal/model"
	"learnos/internal/repository"
)

var ErrImageSize = errors.New("image exceeds 10 MiB")
var ErrImageQuota = errors.New("image storage quota exceeded")
var ErrImageMissing = errors.New("image missing or corrupt")

type LibraryImageService struct {
	repo  *repository.LibraryImageRepository
	dir   string
	quota int64
}

func NewLibraryImageService(repo *repository.LibraryImageRepository, databasePath string, quota int64) *LibraryImageService {
	if quota <= 0 {
		quota = 2 << 30
	}
	return &LibraryImageService{repo, librarymedia.Directory(databasePath), quota}
}
func (s *LibraryImageService) Upload(ctx context.Context, user uint, name string, reader io.Reader) (model.LibraryImage, error) {
	librarymedia.Mu.Lock()
	defer librarymedia.Mu.Unlock()
	data, err := io.ReadAll(io.LimitReader(reader, librarymedia.MaxBytes+1))
	if err != nil {
		return model.LibraryImage{}, err
	}
	if int64(len(data)) > librarymedia.MaxBytes {
		return model.LibraryImage{}, ErrImageSize
	}
	mime, ext, w, h, err := librarymedia.Validate(data)
	if err != nil {
		return model.LibraryImage{}, err
	}
	used, err := s.repo.Usage(ctx, user)
	if err != nil {
		return model.LibraryImage{}, err
	}
	if used+int64(len(data)) > s.quota {
		return model.LibraryImage{}, ErrImageQuota
	}
	count, err := s.repo.Count(ctx)
	if err != nil {
		return model.LibraryImage{}, err
	}
	if count >= 20000 {
		return model.LibraryImage{}, ErrImageQuota
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil && !os.IsNotExist(err) {
		return model.LibraryImage{}, err
	}
	var diskSize int64
	for _, entry := range entries {
		info, e := entry.Info()
		if e != nil {
			return model.LibraryImage{}, e
		}
		if info.Mode().IsRegular() {
			diskSize += info.Size()
		}
	}
	if diskSize+int64(len(data)) > s.quota {
		return model.LibraryImage{}, ErrImageQuota
	}
	token := make([]byte, 16)
	if _, err = rand.Read(token); err != nil {
		return model.LibraryImage{}, err
	}
	id := hex.EncodeToString(token)
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	if len([]rune(name)) > 200 {
		name = string([]rune(name)[:200])
	}
	row := model.LibraryImage{ID: id, UserID: user, StorageName: id + "." + ext, MIME: mime, Size: int64(len(data)), Width: w, Height: h, SHA256: librarymedia.Hash(data), OriginalName: name, URL: librarymedia.Prefix + id}
	if err = os.MkdirAll(s.dir, 0750); err != nil {
		return model.LibraryImage{}, err
	}
	path := filepath.Join(s.dir, row.StorageName)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return model.LibraryImage{}, err
	}
	success := false
	defer func() {
		if !success {
			_ = os.Remove(path)
		}
	}()
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return model.LibraryImage{}, err
	}
	if closeErr != nil {
		return model.LibraryImage{}, closeErr
	}
	if err = s.repo.Create(ctx, &row); err != nil {
		return model.LibraryImage{}, err
	}
	success = true
	return row, nil
}

// ReadLocked is used under the shared store lock by export and request handlers.
func (s *LibraryImageService) ReadLocked(ctx context.Context, user uint, id string) (model.LibraryImage, []byte, error) {
	if !librarymedia.IDPattern.MatchString(id) {
		return model.LibraryImage{}, nil, ErrImageMissing
	}
	row, err := s.repo.Get(ctx, user, id)
	if err != nil {
		return row, nil, err
	}
	data, err := librarymedia.Read(s.dir, row.StorageName)
	if err != nil || int64(len(data)) != row.Size || librarymedia.Hash(data) != row.SHA256 {
		return row, nil, ErrImageMissing
	}
	row.URL = librarymedia.Prefix + row.ID
	return row, data, nil
}
func (s *LibraryImageService) Read(ctx context.Context, user uint, id string) (model.LibraryImage, []byte, error) {
	librarymedia.Mu.RLock()
	defer librarymedia.Mu.RUnlock()
	return s.ReadLocked(ctx, user, id)
}
func (s *LibraryImageService) ValidateReferences(ctx context.Context, user uint, texts ...string) error {
	seen := map[string]bool{}
	for _, text := range texts {
		for _, m := range librarymedia.References.FindAllStringSubmatch(text, -1) {
			if !seen[m[1]] {
				if _, _, err := s.ReadLocked(ctx, user, m[1]); err != nil {
					return ErrImageMissing
				}
				seen[m[1]] = true
			}
		}
	}
	return nil
}
func (s *LibraryImageService) Cleanup(ctx context.Context, user uint) (int, error) {
	librarymedia.Mu.Lock()
	defer librarymedia.Mu.Unlock()
	texts, err := s.repo.References(ctx)
	if err != nil {
		return 0, err
	}
	used := map[string]bool{}
	for _, text := range texts {
		for _, m := range librarymedia.References.FindAllStringSubmatch(text, -1) {
			used[m[1]] = true
		}
	}
	rows, err := s.repo.Old(ctx, user, time.Now().UTC().Add(-7*24*time.Hour))
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, row := range rows {
		if used[row.ID] {
			continue
		}
		// Remove metadata first: an interrupted file removal leaves an inert orphan,
		// never a live image record with missing bytes. No referenced row is touched.
		if err = s.repo.Delete(ctx, user, row.ID); err != nil {
			return removed, err
		}
		if librarymedia.FilePattern.MatchString(row.StorageName) {
			if err = os.Remove(filepath.Join(s.dir, row.StorageName)); err != nil && !os.IsNotExist(err) {
				return removed, err
			}
		}
		removed++
	}
	names, err := s.repo.StorageNames(ctx)
	if err != nil {
		return removed, err
	}
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return removed, nil
	}
	if err != nil {
		return removed, err
	}
	for _, entry := range entries {
		if !librarymedia.FilePattern.MatchString(entry.Name()) || names[entry.Name()] {
			continue
		}
		info, e := entry.Info()
		if e != nil {
			return removed, e
		}
		if !info.Mode().IsRegular() || !info.ModTime().Before(time.Now().Add(-7*24*time.Hour)) {
			continue
		}
		if err = os.Remove(filepath.Join(s.dir, entry.Name())); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
