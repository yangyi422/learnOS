package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"learnos/internal/config"
	"learnos/internal/database"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type BackupService struct {
	db            *gorm.DB
	cfg           config.Config
	restartSignal chan<- struct{}
}

var (
	ErrBackupNotFound            = errors.New("backup not found")
	ErrRestoreConfirmationFailed = errors.New("restore confirmation failed")
)

type RestoreRequest struct {
	BackupName  string    `json:"backup_name"`
	RequestedAt time.Time `json:"requested_at"`
}

type RestoreRequestView struct {
	Backup         database.BackupInfo `json:"backup"`
	RequestedAt    time.Time           `json:"requested_at"`
	RestartPending bool                `json:"restart_pending"`
}

func NewBackupService(db *gorm.DB, cfg config.Config) *BackupService {
	return &BackupService{db: db, cfg: cfg}
}

func (s *BackupService) SetRestartSignal(signal chan<- struct{}) { s.restartSignal = signal }

func (s *BackupService) Create(ctx context.Context) (database.BackupInfo, error) {
	return database.CreateBackup(ctx, s.db, s.cfg.BackupDir, s.cfg.BackupRetentionCount)
}

func (s *BackupService) List(_ context.Context) ([]database.BackupInfo, error) {
	return database.ListBackups(s.cfg.BackupDir)
}

func (s *BackupService) Last() *database.BackupInfo {
	backups, err := database.ListBackups(s.cfg.BackupDir)
	if err != nil || len(backups) == 0 {
		return nil
	}
	return &backups[0]
}

func (s *BackupService) RequestRestore(ctx context.Context, backupName, confirmation string) (*RestoreRequestView, error) {
	backupName = strings.TrimSpace(backupName)
	if backupName == "" || filepath.Base(backupName) != backupName {
		return nil, ErrBackupNotFound
	}
	if strings.TrimSpace(confirmation) != "恢复 "+backupName {
		return nil, ErrRestoreConfirmationFailed
	}
	backups, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	var selected *database.BackupInfo
	for index := range backups {
		if backups[index].Name == backupName {
			selected = &backups[index]
			break
		}
	}
	if selected == nil {
		return nil, ErrBackupNotFound
	}
	if err := s.ValidateBackup(selected.Path); err != nil {
		return nil, fmt.Errorf("validate restore backup: %w", err)
	}
	// Pin the exact selected snapshot before recording the restart request.
	// Creating the mandatory pre-restore backup may enforce retention and prune
	// an older source backup, so the pending copy deliberately does not match
	// the normal backup filename pattern.
	if err := copyFileAtomically(selected.Path, pendingRestoreSourcePath(s.cfg)); err != nil {
		return nil, fmt.Errorf("stage restore backup: %w", err)
	}
	request := RestoreRequest{BackupName: selected.Name, RequestedAt: time.Now().UTC()}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.cfg.BackupDir, 0o755); err != nil {
		return nil, err
	}
	temporary, err := os.CreateTemp(s.cfg.BackupDir, ".restore-request-*.json")
	if err != nil {
		return nil, err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(payload); err != nil {
		_ = temporary.Close()
		return nil, err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return nil, err
	}
	if err := temporary.Close(); err != nil {
		return nil, err
	}
	if err := os.Rename(temporaryPath, restoreRequestPath(s.cfg)); err != nil {
		return nil, err
	}
	return &RestoreRequestView{Backup: *selected, RequestedAt: request.RequestedAt, RestartPending: true}, nil
}

func (s *BackupService) TriggerRestoreRestart() {
	if s.restartSignal == nil {
		return
	}
	select {
	case s.restartSignal <- struct{}{}:
	default:
	}
}

// ApplyPendingRestore runs before database.Open, when no SQLite connection is
// active. RestoreDatabase validates the selected backup and creates a
// pre-restore backup of the current database before the atomic replacement.
func ApplyPendingRestore(cfg config.Config) (bool, error) {
	marker := restoreRequestPath(cfg)
	payload, err := os.ReadFile(marker)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read pending restore request: %w", err)
	}
	var request RestoreRequest
	if err := json.Unmarshal(payload, &request); err != nil || filepath.Base(request.BackupName) != request.BackupName {
		archiveFailedRestore(marker)
		_ = os.Remove(pendingRestoreSourcePath(cfg))
		return false, fmt.Errorf("invalid pending restore request")
	}
	backupPath := pendingRestoreSourcePath(cfg)
	if _, err := RestoreDatabase(cfg.DatabasePath, backupPath, cfg.BackupDir, cfg.BackupRetentionCount); err != nil {
		archiveFailedRestore(marker)
		_ = os.Remove(backupPath)
		return false, err
	}
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}

func restoreRequestPath(cfg config.Config) string {
	return filepath.Join(cfg.BackupDir, ".restore-request.json")
}

func pendingRestoreSourcePath(cfg config.Config) string {
	return filepath.Join(cfg.BackupDir, ".pending-restore.db")
}

func copyFileAtomically(sourcePath, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	temporary, err := os.CreateTemp(filepath.Dir(targetPath), ".pending-restore-*.db")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := io.Copy(temporary, source); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, targetPath)
}

func archiveFailedRestore(marker string) {
	failed := marker + ".failed-" + time.Now().UTC().Format("20060102-150405")
	_ = os.Rename(marker, failed)
}

func (s *BackupService) ValidateBackup(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat backup: %w", err)
	}
	if info.IsDir() || filepath.Ext(path) != ".db" {
		return fmt.Errorf("backup must be a sqlite .db file")
	}
	return validateSQLiteFile(path)
}

func validateSQLiteFile(path string) error {
	values := url.Values{}
	values.Set("mode", "ro")
	values.Set("_foreign_keys", "on")
	db, err := gorm.Open(sqlite.Open("file:"+path+"?"+values.Encode()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return fmt.Errorf("open backup: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get backup connection: %w", err)
	}
	defer sqlDB.Close()
	var result string
	if err := db.Raw("PRAGMA integrity_check").Scan(&result).Error; err != nil {
		return fmt.Errorf("backup integrity check: %w", err)
	}
	if strings.TrimSpace(result) != "ok" {
		return fmt.Errorf("backup integrity check: %s", result)
	}
	return nil
}

// RestoreDatabase is intended for a stopped application. It validates the
// input first, creates a pre-restore backup of the current database, then
// atomically swaps a fully copied file into place.
func RestoreDatabase(targetPath, backupPath, backupDir string, retention int) (database.BackupInfo, error) {
	if err := validateSQLiteFile(backupPath); err != nil {
		return database.BackupInfo{}, err
	}
	var preRestore database.BackupInfo
	if _, err := os.Stat(targetPath); err == nil {
		currentDB, openErr := gorm.Open(sqlite.Open("file:"+targetPath+"?_busy_timeout=5000&_journal_mode=WAL"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if openErr != nil {
			return database.BackupInfo{}, fmt.Errorf("open current database: %w", openErr)
		}
		preRestore, openErr = database.CreateBackup(context.Background(), currentDB, backupDir, retention)
		if sqlDB, dbErr := currentDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		if openErr != nil {
			return database.BackupInfo{}, fmt.Errorf("pre-restore backup failed: %w", openErr)
		}
	} else if !os.IsNotExist(err) {
		return database.BackupInfo{}, fmt.Errorf("stat current database: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return preRestore, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(targetPath), ".learnos-restore-*.db")
	if err != nil {
		return preRestore, fmt.Errorf("create restore temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	source, err := os.Open(backupPath)
	if err != nil {
		_ = tmp.Close()
		return preRestore, err
	}
	if _, err := io.Copy(tmp, source); err != nil {
		source.Close()
		tmp.Close()
		return preRestore, fmt.Errorf("copy restore database: %w", err)
	}
	_ = source.Close()
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return preRestore, fmt.Errorf("sync restore database: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return preRestore, err
	}
	if err := os.Remove(targetPath + "-wal"); err != nil && !os.IsNotExist(err) {
		return preRestore, err
	}
	if err := os.Remove(targetPath + "-shm"); err != nil && !os.IsNotExist(err) {
		return preRestore, err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return preRestore, fmt.Errorf("replace database: %w", err)
	}
	if err := validateSQLiteFile(targetPath); err != nil {
		return preRestore, fmt.Errorf("restored database failed integrity check: %w", err)
	}
	return preRestore, nil
}
