package bridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// devAPIBase is only a development fallback for a Go backend running on this
// machine. Real deployments configure their base in desktop.config.json or the
// LIFEOS_API_BASE environment variable; nothing is hardcoded here.
const devAPIBase = "http://127.0.0.1:8080"

const configFileName = "desktop.config.json"

type Config struct {
	APIBase string `json:"apiBase"`
	// source records where the base came from, for the startup log and /healthz.
	Source string `json:"-"`
}

// LoadConfig resolves the upstream API base in this order:
//  1. LIFEOS_API_BASE
//  2. desktop.config.json next to the executable
//  3. desktop.config.json in the working directory
//  4. embedded: the desktop.config.json compiled into the binary at build time
//  5. the local development fallback
//
// The embedded default is what makes the release a single portable .exe; the
// file and env overrides stay available for other machines without a rebuild.
func LoadConfig(embedded []byte) (Config, error) {
	if raw, ok := os.LookupEnv("LIFEOS_API_BASE"); ok && strings.TrimSpace(raw) != "" {
		base, err := NormalizeAPIBase(raw)
		return Config{APIBase: base, Source: "env:LIFEOS_API_BASE"}, err
	}

	dirs := make([]string, 0, 2)
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}

	for _, dir := range dirs {
		path := filepath.Join(dir, configFileName)
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return Config{}, fmt.Errorf("read %s: %w", path, err)
		}
		var parsed Config
		if err := json.Unmarshal(data, &parsed); err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", path, err)
		}
		base, err := NormalizeAPIBase(parsed.APIBase)
		if err != nil {
			return Config{}, fmt.Errorf("%s: %w", path, err)
		}
		return Config{APIBase: base, Source: path}, nil
	}

	if len(embedded) > 0 {
		var parsed Config
		if err := json.Unmarshal(embedded, &parsed); err != nil {
			return Config{}, fmt.Errorf("embedded %s: %w", configFileName, err)
		}
		base, err := NormalizeAPIBase(parsed.APIBase)
		if err != nil {
			return Config{}, fmt.Errorf("embedded %s: %w", configFileName, err)
		}
		return Config{APIBase: base, Source: "embedded default"}, nil
	}

	base, err := NormalizeAPIBase(devAPIBase)
	return Config{APIBase: base, Source: "built-in development fallback"}, err
}

// normalizeAPIBase validates the upstream and strips anything that is not an
// origin: the proxy target must never contain a path, query, fragment or
// userinfo, so a request cannot be steered somewhere else.
func NormalizeAPIBase(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("apiBase is empty")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("apiBase %q is not a URL: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("apiBase %q must use http or https", raw)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("apiBase %q has no host", raw)
	}
	if parsed.User != nil {
		return "", errors.New("apiBase must not embed credentials")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("apiBase must not contain a query or fragment")
	}
	if path := strings.TrimSuffix(parsed.Path, "/"); path != "" {
		return "", fmt.Errorf("apiBase %q must be an origin without a path", raw)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}
