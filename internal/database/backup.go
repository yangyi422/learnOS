package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type BackupInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateBackup(ctx context.Context, db *gorm.DB, backupDir string, retention int) (BackupInfo, error) {
	if db == nil {
		return BackupInfo{}, fmt.Errorf("database is nil")
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return BackupInfo{}, fmt.Errorf("create backup directory: %w", err)
	}
	now := time.Now().UTC()
	name := fmt.Sprintf("learnos-%s.db", now.Format("20060102-150405"))
	path := filepath.Join(backupDir, name)
	for attempt := 0; ; attempt++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}
		if attempt > 99 {
			return BackupInfo{}, fmt.Errorf("unable to allocate backup filename")
		}
		name = fmt.Sprintf("learnos-%s-%02d.db", now.Format("20060102-150405"), attempt+1)
		path = filepath.Join(backupDir, name)
	}
	if err := db.WithContext(ctx).Exec("VACUUM INTO ?", path).Error; err != nil {
		return BackupInfo{}, fmt.Errorf("vacuum database into backup: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return BackupInfo{}, fmt.Errorf("stat backup: %w", err)
	}
	if retention > 0 {
		if err := RetainBackups(backupDir, retention); err != nil {
			return BackupInfo{}, err
		}
	}
	return BackupInfo{Name: name, Path: path, Size: info.Size(), CreatedAt: info.ModTime().UTC()}, nil
}

func ListBackups(backupDir string) ([]BackupInfo, error) {
	entries, err := os.ReadDir(backupDir)
	if os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}
	backups := make([]BackupInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "learnos-") || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil {
			return nil, fmt.Errorf("stat backup %s: %w", entry.Name(), statErr)
		}
		backups = append(backups, BackupInfo{Name: entry.Name(), Path: filepath.Join(backupDir, entry.Name()), Size: info.Size(), CreatedAt: info.ModTime().UTC()})
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].CreatedAt.After(backups[j].CreatedAt) })
	return backups, nil
}

func RetainBackups(backupDir string, retention int) error {
	if retention < 1 {
		return nil
	}
	backups, err := ListBackups(backupDir)
	if err != nil {
		return err
	}
	if len(backups) <= retention {
		return nil
	}
	for _, backup := range backups[retention:] {
		if err := os.Remove(backup.Path); err != nil {
			return fmt.Errorf("remove old backup %s: %w", backup.Name, err)
		}
	}
	return nil
}
