package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	AppName      string
	Environment  string
	Address      string
	DataDir      string
	DatabasePath string
	Username     string
	PasswordHash string
}

func Load() (Config, error) {
	dataDir := envOrDefault("APP_DATA_DIR", "./data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppName:      envOrDefault("APP_NAME", "LearnOS"),
		Environment:  envOrDefault("APP_ENV", "development"),
		Address:      envOrDefault("APP_ADDR", ":8080"),
		DataDir:      dataDir,
		DatabasePath: filepath.Join(dataDir, "learnos.db"),
		Username:     strings.TrimSpace(os.Getenv("APP_USERNAME")),
		PasswordHash: strings.TrimSpace(os.Getenv("APP_PASSWORD_HASH")),
	}

	if (cfg.Username == "") != (cfg.PasswordHash == "") {
		return Config{}, errors.New("APP_USERNAME and APP_PASSWORD_HASH must be configured together")
	}

	return cfg, nil
}

func (c Config) Production() bool {
	return strings.EqualFold(c.Environment, "production")
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
