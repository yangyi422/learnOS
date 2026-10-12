// Package librarymedia holds the small, shared file-store contract used by
// image uploads, export and backup. IDs and bytes are immutable after upload.
package librarymedia

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	_ "golang.org/x/image/webp"
)

const MaxBytes int64 = 10 << 20
const MaxPixels int64 = 24_000_000
const Prefix = "/api/v1/library/images/"

var IDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)
var References = regexp.MustCompile(`/api/v1/library/images/([a-f0-9]{32})\b`)
var FilePattern = regexp.MustCompile(`^[a-f0-9]{32}\.(png|jpg|webp)$`)

// Serialize bounded decodes and file mutation with snapshots/cleanup. This
// application intentionally runs as one process on a personal SQLite database.
var Mu sync.RWMutex
var ErrInvalid = errors.New("invalid image")

func Directory(databasePath string) string {
	return filepath.Join(filepath.Dir(databasePath), "uploads", "library")
}
func Validate(data []byte) (mime, ext string, width, height int, err error) {
	if len(data) == 0 || int64(len(data)) > MaxBytes {
		return "", "", 0, 0, ErrInvalid
	}
	cfg, format, e := image.DecodeConfig(bytes.NewReader(data))
	types := map[string][2]string{"png": {"image/png", "png"}, "jpeg": {"image/jpeg", "jpg"}, "webp": {"image/webp", "webp"}}
	kind, ok := types[format]
	if e != nil || !ok || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 12000 || cfg.Height > 12000 || int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return "", "", 0, 0, ErrInvalid
	}
	// Header-only checks are insufficient: reject truncated/corrupt image bodies.
	decoded, actual, e := image.Decode(bytes.NewReader(data))
	if e != nil || actual != format || decoded.Bounds().Dx() != cfg.Width || decoded.Bounds().Dy() != cfg.Height {
		return "", "", 0, 0, ErrInvalid
	}
	return kind[0], kind[1], cfg.Width, cfg.Height, nil
}
func Hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func Read(dir, name string) ([]byte, error) {
	if !FilePattern.MatchString(name) {
		return nil, ErrInvalid
	}
	// Do not follow a symlink planted in the persistent uploads directory.
	p := filepath.Join(dir, name)
	info, err := os.Lstat(p)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > MaxBytes {
		return nil, ErrInvalid
	}
	return os.ReadFile(p)
}
