// Package config loads and validates the bridge configuration. Validation is
// strict on purpose: a misconfigured mapping must fail before a player launch.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var ErrInvalid = errors.New("invalid configuration")

var windowsTargetPattern = regexp.MustCompile(`^[A-Za-z]:\\`)

type Mapping struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

type Config struct {
	Jellyfin struct {
		BaseURL          string   `yaml:"base_url"`
		UserID           string   `yaml:"user_id"`
		CredentialTarget string   `yaml:"credential_target"`
		AllowedLibraries []string `yaml:"allowed_libraries"`
		TimeoutSeconds   int      `yaml:"timeout_seconds"`
	} `yaml:"jellyfin"`
	Player struct {
		Executable string `yaml:"executable"`
		Type       string `yaml:"type"`
	} `yaml:"player"`
	PathMappings []Mapping `yaml:"path_mappings"`
	Playback     struct {
		CheckFileExists   bool     `yaml:"check_file_exists"`
		AllowedExtensions []string `yaml:"allowed_extensions"`
	} `yaml:"playback"`
}	

const DefaultCredentialTarget = "lifeos-player-bridge/jellyfin"

var defaultExtensions = []string{"mkv", "mp4", "avi", "mov", "m4v", "ts", "wmv"}

// DefaultPath is where install.ps1 places the configuration.
func DefaultPath() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.Getenv("APPDATA")
	}
	return filepath.Join(base, "LifeOS", "PlayerBridge", "config.yaml")
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if err := cfg.normalize(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) normalize() error {
	c.Jellyfin.BaseURL = strings.TrimRight(strings.TrimSpace(c.Jellyfin.BaseURL), "/")
	parsed, err := url.Parse(c.Jellyfin.BaseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%w: jellyfin.base_url must be an http(s) URL", ErrInvalid)
	}
	if strings.TrimSpace(c.Jellyfin.UserID) == "" {
		return fmt.Errorf("%w: jellyfin.user_id is required (dedicated non-admin Jellyfin user)", ErrInvalid)
	}
	if strings.TrimSpace(c.Jellyfin.CredentialTarget) == "" {
		c.Jellyfin.CredentialTarget = DefaultCredentialTarget
	}
	if c.Jellyfin.TimeoutSeconds <= 0 {
		c.Jellyfin.TimeoutSeconds = 20
	}
	if len(c.Jellyfin.AllowedLibraries) == 0 {
		c.Jellyfin.AllowedLibraries = []string{"movies"}
	}
	for i := range c.Jellyfin.AllowedLibraries {
		c.Jellyfin.AllowedLibraries[i] = strings.ToLower(strings.TrimSpace(c.Jellyfin.AllowedLibraries[i]))
	}
	if strings.TrimSpace(c.Player.Type) == "" {
		c.Player.Type = "potplayer"
	}
	if !strings.EqualFold(c.Player.Type, "potplayer") {
		return fmt.Errorf("%w: player.type must be potplayer", ErrInvalid)
	}
	if strings.TrimSpace(c.Player.Executable) == "" {
		c.Player.Executable = "auto"
	}
	if len(c.PathMappings) == 0 {
		return fmt.Errorf("%w: path_mappings must contain at least one mapping", ErrInvalid)
	}
	if len(c.Playback.AllowedExtensions) == 0 {
		c.Playback.AllowedExtensions = append([]string(nil), defaultExtensions...)
	}
	for i := range c.Playback.AllowedExtensions {
		c.Playback.AllowedExtensions[i] = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(c.Playback.AllowedExtensions[i]), "."))
	}
	for i := range c.PathMappings {
		mapping := &c.PathMappings[i]
		mapping.Source = strings.TrimSpace(mapping.Source)
		mapping.Target = strings.TrimSpace(mapping.Target)
		if !strings.HasPrefix(mapping.Source, "/") {
			return fmt.Errorf("%w: path_mappings[%d].source must be an absolute container path", ErrInvalid, i)
		}
		if !strings.HasSuffix(mapping.Source, "/") {
			mapping.Source += "/"
		}
		if !windowsTargetPattern.MatchString(mapping.Target) {
			return fmt.Errorf("%w: path_mappings[%d].target must be a Windows drive path like Z:\\Movies\\", ErrInvalid, i)
		}
		if !strings.HasSuffix(mapping.Target, "\\") {
			mapping.Target += "\\"
		}
	}
	return nil
}

// Extensions returns the allowlist as a set of lowercase suffixes without dots.
func (c *Config) Extensions() map[string]bool {
	set := make(map[string]bool, len(c.Playback.AllowedExtensions))
	for _, ext := range c.Playback.AllowedExtensions {
		set[ext] = true
	}
	return set
}