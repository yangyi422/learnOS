// Package pathmap converts a Jellyfin container path into a Windows drive or UNC path
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
	configErr   error
}

func New(mappings []Mapping, allowedExtensions map[string]bool, checkExists bool) *Mapper {
	normalized := make([]Mapping, len(mappings))
	var configErr error
	for i, mapping := range mappings {
		source, err := NormalizeSource(mapping.Source)
		if err != nil {
			configErr = fmt.Errorf("%w: %v", ErrTraversal, err)
		}
		target, err := NormalizeTarget(mapping.Target)
		if err != nil {
			configErr = fmt.Errorf("%w: %v", ErrTraversal, err)
		}
		normalized[i] = Mapping{Source: source, Target: target}
	}
	return &Mapper{mappings: normalized, allowed: allowedExtensions, checkExists: checkExists, configErr: configErr}
}

// Map returns the Windows path for a Jellyfin container path.
func (m *Mapper) Map(containerPath string) (string, error) {
	if m.configErr != nil {
		return "", m.configErr
	}
	if strings.TrimSpace(containerPath) == "" {
		return "", ErrNoMapping
	}
	containerPath = strings.ReplaceAll(containerPath, "\\", "/")
	if hasTraversal(containerPath, "/") {
		return "", ErrTraversal
	}
	cleaned := path.Clean(containerPath)
	if !strings.HasPrefix(cleaned, "/") {
		return "", ErrNoMapping
	}
	for _, mapping := range m.mappings {
		// The trailing separator in Source makes this a prefix boundary check:
		// /media/movies/ matches /media/movies/a.mkv but never /media/movies2/a.mkv.
		if strings.EqualFold(cleaned, strings.TrimSuffix(mapping.Source, "/")) {
			return "", ErrNoMapping
		}
		if len(cleaned) < len(mapping.Source) || !strings.EqualFold(cleaned[:len(mapping.Source)], mapping.Source) {
			continue
		}
		remainder := cleaned[len(mapping.Source):]
		if remainder == "" {
			return "", ErrNoMapping
		}
		for _, segment := range strings.Split(remainder, "/") {
			if !safeWindowsSegment(segment) {
				return "", ErrTraversal
			}
		}
		candidate := filepath.Join(mapping.Target, filepath.FromSlash(remainder))
		relative, err := filepath.Rel(mapping.Target, candidate)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
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
