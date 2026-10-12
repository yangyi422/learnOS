package service

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"learnos/internal/database"
	"learnos/internal/librarymedia"
)

// A bounded, validated staging area is disposable. No archive path is ever
// concatenated to a live uploads directory, and the live directory is not cleared.
func stageMediaBackup(source string) (string, error) {
	z, err := zip.OpenReader(source)
	if err != nil {
		return "", fmt.Errorf("invalid full backup ZIP")
	}
	defer z.Close()
	if len(z.File) > 20002 {
		return "", fmt.Errorf("too many backup entries")
	}
	entries := map[string]*zip.File{}
	var total uint64
	for _, f := range z.File {
		if entries[f.Name] != nil || f.Mode()&os.ModeSymlink != 0 || f.FileInfo().IsDir() || (f.Name != "learnos.db" && f.Name != "manifest.json" && !(strings.HasPrefix(f.Name, "assets/") && librarymedia.FilePattern.MatchString(strings.TrimPrefix(f.Name, "assets/")))) {
			return "", fmt.Errorf("unsafe or duplicate backup path")
		}
		limit := uint64(librarymedia.MaxBytes)
		if f.Name == "learnos.db" {
			limit = 8 << 30
		}
		if f.Name == "manifest.json" {
			limit = 8 << 20
		}
		if f.UncompressedSize64 > limit {
			return "", fmt.Errorf("backup entry too large")
		}
		total += f.UncompressedSize64
		if total > 32<<30 {
			return "", fmt.Errorf("backup too large")
		}
		entries[f.Name] = f
	}
	manifestFile := entries["manifest.json"]
	if manifestFile == nil || entries["learnos.db"] == nil {
		return "", fmt.Errorf("missing full backup manifest or database")
	}
	r, err := manifestFile.Open()
	if err != nil {
		return "", err
	}
	payload, err := io.ReadAll(io.LimitReader(r, 8<<20+1))
	r.Close()
	if err != nil || len(payload) > 8<<20 {
		return "", fmt.Errorf("invalid manifest")
	}
	var manifest database.MediaBackupManifest
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&manifest) != nil || decoder.Decode(new(any)) != io.EOF || manifest.Format != "lifeos-backup-v1" || manifest.Scope != database.FullBackupScope || manifest.Database.Name != "learnos.db" {
		return "", fmt.Errorf("unsupported full backup format")
	}
	expected := map[string]database.MediaBackupFile{"learnos.db": manifest.Database}
	for _, item := range manifest.Images {
		if !strings.HasPrefix(item.Name, "assets/") || !librarymedia.FilePattern.MatchString(strings.TrimPrefix(item.Name, "assets/")) || expected[item.Name].Name != "" {
			return "", fmt.Errorf("invalid image manifest")
		}
		expected[item.Name] = item
	}
	if len(expected)+1 != len(entries) {
		return "", fmt.Errorf("unexpected backup files")
	}
	stage, err := os.MkdirTemp("", "learnos-media-restore-")
	if err != nil {
		return "", err
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(stage)
		}
	}()
	for name, item := range expected {
		f := entries[name]
		if f == nil || item.Size < 0 || uint64(item.Size) != f.UncompressedSize64 || len(item.SHA256) != 64 {
			return "", fmt.Errorf("backup manifest size mismatch")
		}
		path := filepath.Join(stage, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return "", err
		}
		out, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return "", e
		}
		in, e := f.Open()
		if e != nil {
			out.Close()
			return "", e
		}
		n, e := io.Copy(out, io.LimitReader(in, item.Size+1))
		in.Close()
		if e == nil {
			e = out.Sync()
		}
		closeErr := out.Close()
		if e != nil || closeErr != nil || n != item.Size {
			return "", fmt.Errorf("invalid backup entry")
		}
		hash, e := database.FileSHA256(path)
		if e != nil || hash != item.SHA256 {
			return "", fmt.Errorf("backup checksum mismatch")
		}
	}
	dbPath := filepath.Join(stage, "learnos.db")
	if err = validateSQLiteFile(dbPath); err != nil {
		return "", err
	}
	rows, err := database.SnapshotImages(dbPath)
	if err != nil {
		return "", err
	}
	if len(rows) != len(manifest.Images) {
		return "", fmt.Errorf("image metadata mismatch")
	}
	for _, row := range rows {
		if !librarymedia.IDPattern.MatchString(row.ID) || !strings.HasPrefix(row.StorageName, row.ID+".") {
			return "", fmt.Errorf("invalid stored image ID")
		}
		entry, ok := expected["assets/"+row.StorageName]
		if !ok || entry.Size != row.Size || entry.SHA256 != row.SHA256 {
			return "", fmt.Errorf("missing backup image")
		}
		data, e := librarymedia.Read(filepath.Join(stage, "assets"), row.StorageName)
		if e != nil {
			return "", e
		}
		mime, ext, w, h, e := librarymedia.Validate(data)
		if e != nil || mime != row.MIME || row.StorageName != row.ID+"."+ext || w != row.Width || h != row.Height {
			return "", fmt.Errorf("invalid backup image content")
		}
	}
	success = true
	return stage, nil
}
func isMediaBackup(path string) bool {
	f, e := os.Open(path)
	if e != nil {
		return false
	}
	defer f.Close()
	var header [4]byte
	_, e = io.ReadFull(f, header[:])
	return e == nil && bytes.Equal(header[:], []byte{'P', 'K', 3, 4})
}

// Merge immutable images before the SQLite swap. Existing files are never
// overwritten or deleted, including when restoring a database-only old backup.
// A failed restore may add harmless unreferenced files, but cannot erase images.
func RestoreWithMedia(target, source, backupDir string, retention int) (database.BackupInfo, error) {
	if !isMediaBackup(source) {
		return RestoreDatabase(target, source, backupDir, retention)
	}
	// Restore runs with the application stopped, so no file mutation can race.
	stage, err := stageMediaBackup(source)
	if err != nil {
		return database.BackupInfo{}, err
	}
	defer os.RemoveAll(stage)
	dir := librarymedia.Directory(target)
	if err = os.MkdirAll(dir, 0750); err != nil {
		return database.BackupInfo{}, err
	}
	entries, err := os.ReadDir(filepath.Join(stage, "assets"))
	if os.IsNotExist(err) {
		entries = nil
	} else if err != nil {
		return database.BackupInfo{}, err
	}
	// Preflight every existing ID before making any changes.
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if info, e := os.Lstat(path); e == nil {
			if !info.Mode().IsRegular() {
				return database.BackupInfo{}, fmt.Errorf("restore image path is not a regular file")
			}
			original, e := database.FileSHA256(path)
			if e != nil {
				return database.BackupInfo{}, e
			}
			restored, e := database.FileSHA256(filepath.Join(stage, "assets", entry.Name()))
			if e != nil || original != restored {
				return database.BackupInfo{}, fmt.Errorf("existing image ID conflicts with backup")
			}
		} else if !os.IsNotExist(e) {
			return database.BackupInfo{}, e
		}
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(e) {
			continue
		}
		if e != nil {
			return database.BackupInfo{}, e
		}
		data, e := librarymedia.Read(filepath.Join(stage, "assets"), entry.Name())
		if e == nil {
			_, e = f.Write(data)
		}
		if e == nil {
			e = f.Sync()
		}
		closeErr := f.Close()
		if e != nil || closeErr != nil {
			os.Remove(path)
			return database.BackupInfo{}, fmt.Errorf("restore image write failed")
		}
	}
	return RestoreDatabase(target, filepath.Join(stage, "learnos.db"), backupDir, retention)
}
