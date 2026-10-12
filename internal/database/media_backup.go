package database

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"learnos/internal/librarymedia"
	"learnos/internal/model"
)

const FullBackupScope = "database_and_library_images"

type MediaBackupFile struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type MediaBackupManifest struct {
	Format   string            `json:"format"`
	Scope    string            `json:"scope"`
	Database MediaBackupFile   `json:"database"`
	Images   []MediaBackupFile `json:"images"`
}

func FileSHA256(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return "", e
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func SnapshotImages(path string) ([]model.LibraryImage, error) {
	db, e := gorm.Open(sqlite.Open("file:"+filepath.ToSlash(path)+"?mode=ro"), &gorm.Config{Logger: logger.Discard})
	if e != nil {
		return nil, e
	}
	conn, e := db.DB()
	if e != nil {
		return nil, e
	}
	defer conn.Close()
	if !db.Migrator().HasTable(&model.LibraryImage{}) {
		return []model.LibraryImage{}, nil
	}
	var rows []model.LibraryImage
	e = db.Find(&rows).Error
	return rows, e
}

// Full snapshots contain the immutable files present in the SQLite snapshot,
// including pending uploads; no recursive walk can accidentally include secrets.
func CreateMediaBackup(ctx context.Context, db *gorm.DB, databasePath, backupDir string, retention int) (BackupInfo, error) {
	librarymedia.Mu.RLock()
	defer librarymedia.Mu.RUnlock()
	if !db.Migrator().HasTable(&model.LibraryImage{}) {
		return CreateBackup(ctx, db, backupDir, retention)
	}
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return BackupInfo{}, err
	}
	stage, err := os.MkdirTemp(backupDir, ".media-backup-")
	if err != nil {
		return BackupInfo{}, err
	}
	defer os.RemoveAll(stage)
	snap, err := CreateBackup(ctx, db, stage, 0)
	if err != nil {
		return BackupInfo{}, err
	}
	rows, err := SnapshotImages(snap.Path)
	if err != nil {
		return BackupInfo{}, err
	}
	if len(rows) > 20000 {
		return BackupInfo{}, fmt.Errorf("too many images for full backup")
	}
	hash, err := FileSHA256(snap.Path)
	if err != nil {
		return BackupInfo{}, err
	}
	manifest := MediaBackupManifest{Format: "lifeos-backup-v1", Scope: FullBackupScope, Database: MediaBackupFile{Name: "learnos.db", SHA256: hash, Size: snap.Size}, Images: []MediaBackupFile{}}
	output := filepath.Join(stage, "bundle.zip")
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return BackupInfo{}, err
	}
	defer f.Close()
	z := zip.NewWriter(f)
	defer z.Close()
	write := func(name string, r io.Reader) error {
		w, e := z.Create(name)
		if e != nil {
			return e
		}
		_, e = io.Copy(w, r)
		return e
	}
	databaseFile, err := os.Open(snap.Path)
	if err != nil {
		return BackupInfo{}, err
	}
	err = write("learnos.db", databaseFile)
	databaseFile.Close()
	if err != nil {
		return BackupInfo{}, err
	}
	for _, row := range rows {
		if err = ctx.Err(); err != nil {
			return BackupInfo{}, err
		}
		if !librarymedia.IDPattern.MatchString(row.ID) || row.StorageName[:min(len(row.StorageName), 32)] != row.ID {
			return BackupInfo{}, fmt.Errorf("invalid image metadata")
		}
		data, e := librarymedia.Read(librarymedia.Directory(databasePath), row.StorageName)
		if e != nil || int64(len(data)) != row.Size || librarymedia.Hash(data) != row.SHA256 {
			return BackupInfo{}, fmt.Errorf("image missing or corrupt; full backup was not created")
		}
		name := "assets/" + row.StorageName
		if err = write(name, bytes.NewReader(data)); err != nil {
			return BackupInfo{}, err
		}
		manifest.Images = append(manifest.Images, MediaBackupFile{Name: name, SHA256: row.SHA256, Size: row.Size})
	}
	meta, err := json.Marshal(manifest)
	if err != nil {
		return BackupInfo{}, err
	}
	if err = write("manifest.json", bytes.NewReader(meta)); err != nil {
		return BackupInfo{}, err
	}
	if err = z.Close(); err != nil {
		return BackupInfo{}, err
	}
	if err = f.Sync(); err != nil {
		return BackupInfo{}, err
	}
	if err = f.Close(); err != nil {
		return BackupInfo{}, err
	}
	name := fmt.Sprintf("learnos-%s.zip", time.Now().UTC().Format("20060102-150405.000000000"))
	path := filepath.Join(backupDir, name)
	if err = os.Rename(output, path); err != nil {
		return BackupInfo{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return BackupInfo{}, err
	}
	if err = RetainBackups(backupDir, retention); err != nil {
		return BackupInfo{}, err
	}
	return BackupInfo{Name: name, Path: path, Size: info.Size(), CreatedAt: info.ModTime().UTC(), Scope: FullBackupScope}, nil
}
