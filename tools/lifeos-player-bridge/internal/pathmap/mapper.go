// Package pathmap converts a Jellyfin container path into a Windows drive path
// and refuses anything outside the configured mappings.
package pathmap

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	ErrNoMapping        = errors.New("no path mapping matched the media path")
	ErrTraversal        = errors.New("media path escapes its mapping")
	ErrExtension        = errors.New("media container extension is not allowed")
	ErrFileMissing      = errors.New("media file is missing on the mapped drive")
	ErrDriveUnavailable = errors.New("mapped drive is unavailable; check the SMB connection")
)

type Mapping struct {
	Source string
	Target string
}

type Mapper struct {
	mappings    []Mapping
	allowed     map[string]bool
	checkExists bool
}

func New(mappings []Mapping, allowedExtensions map[string]bool, checkExists bool) *Mapper {
	normalized := make([]Mapping, len(mappings))
	for i, mapping := range mappings {
		source := strings.TrimSpace(mapping.Source)
		if !strings.HasSuffix(source, "/") {
			source += "/"
		}
		target := strings.TrimSpace(mapping.Target)
		if !strings.HasSuffix(target, "\\") {
			target += "\\"
		}
		normalized[i] = Mapping{Source: source, Target: target}
	}
	return &Mapper{mappings: normalized, allowed: allowedExtensions, checkExists: checkExists}
}

// Map returns the Windows path for a Jellyfin container path.
func (m *Mapper) Map(containerPath string) (string, error) {
	if strings.TrimSpace(containerPath) == "" {
		return "", ErrNoMapping
	}
	cleaned := path.Clean(strings.ReplaceAll(containerPath, "\\", "/"))
	if !strings.HasPrefix(cleaned, "/") {
		return "", ErrNoMapping
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == ".." {
			return "", ErrTraversal
		}
	}
	lowerPath := strings.ToLower(cleaned)
	for _, mapping := range m.mappings {
		lowerSource := strings.ToLower(mapping.Source)
		// The trailing separator in lowerSource makes this a prefix boundary check:
		// /media/movies/ matches /media/movies/a.mkv but never /media/movies2/a.mkv.
		if lowerPath == strings.TrimSuffix(lowerSource, "/") {
			return "", ErrNoMapping
		}
		if !strings.HasPrefix(lowerPath, lowerSource) {
			continue
		}
		remainder := cleaned[len(mapping.Source):]
		if remainder == "" {
			return "", ErrNoMapping
		}
		candidate := filepath.Clean(mapping.Target + strings.ReplaceAll(remainder, "/", "\\"))
		if !strings.HasPrefix(strings.ToLower(candidate), strings.ToLower(mapping.Target)) {
			return "", ErrTraversal
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(candidate), "."))
		if len(m.allowed) > 0 && !m.allowed[ext] {
			return "", fmt.Errorf("%w: .%s", ErrExtension, ext)
		}
		if m.checkExists {
			if err := statMedia(candidate, mapping.Target); err != nil {
				return "", err
			}
		}
		return candidate, nil
	}
	return "", ErrNoMapping
}

func statMedia(candidate, targetRoot string) error {
	info, err := os.Stat(candidate)
	if err == nil {
		if info.IsDir() {
			return ErrFileMissing
		}
		return nil
	}
	if _, rootErr := os.Stat(targetRoot); rootErr != nil {
		return ErrDriveUnavailable
	}
	return ErrFileMissing
}