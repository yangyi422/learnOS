// Package player locates PotPlayer and launches it with one verified file path.
package player

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	ErrNotFound       = errors.New("PotPlayer executable not found")
	ErrLaunchFailed   = errors.New("PotPlayer launch failed")
)

// commonPaths lists the usual PotPlayer install locations, newest name first.
func commonPaths() []string {
	var paths []string
	for _, root := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramW6432")} {
		if root == "" {
			continue
		}
		for _, name := range []string{"PotPlayerMini64.exe", "PotPlayerMini.exe"} {
			paths = append(paths, filepath.Join(root, "DAUM", "PotPlayer", name))
		}
	}
	return paths
}

// Detect returns the player executable to use. "auto" (or empty) searches the
// common install locations; anything else must be an existing file.
func Detect(configured string) (string, error) {
	if configured != "" && !equalFold(configured, "auto") {
		if info, err := os.Stat(configured); err == nil && !info.IsDir() {
			return configured, nil
		}
		return "", fmt.Errorf("%w: %s", ErrNotFound, configured)
	}
	for _, candidate := range commonPaths() {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", ErrNotFound
}

// Launch starts the player with the media path as a single argument. No shell is
// involved and the bridge does not wait for the player to exit.
func Launch(executable, mediaPath string) error {
	if executable == "" || mediaPath == "" {
		return ErrLaunchFailed
	}
	command := exec.Command(executable, mediaPath)
	command.Dir = filepath.Dir(executable)
	if err := command.Start(); err != nil {
		return fmt.Errorf("%w: %v", ErrLaunchFailed, err)
	}
	// Release the handle so the bridge can exit immediately.
	if command.Process != nil {
		_ = command.Process.Release()
	}
	return nil
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}