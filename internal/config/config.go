package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	AppName          string
	Environment      string
	Address          string
	DataDir          string
	DatabasePath     string
	Username         string
	PasswordHash     string
	AIProvider       string
	DeepSeekAPIKey   string
	DeepSeekBaseURL  string
	DeepSeekModel    string
	AITimeoutSeconds int
}

func Load() (Config, error) {
	dataDir := envOrDefault("APP_DATA_DIR", "./data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppName:          envOrDefault("APP_NAME", "LearnOS"),
		Environment:      envOrDefault("APP_ENV", "development"),
		Address:          envOrDefault("APP_ADDR", ":8080"),
		DataDir:          dataDir,
		DatabasePath:     filepath.Join(dataDir, "learnos.db"),
		Username:         strings.TrimSpace(os.Getenv("APP_USERNAME")),
		PasswordHash:     strings.TrimSpace(os.Getenv("APP_PASSWORD_HASH")),
		AIProvider:       envOrDefault("AI_PROVIDER", "mock"),
		DeepSeekAPIKey:   strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")),
		DeepSeekBaseURL:  envOrDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		DeepSeekModel:    envOrDefault("DEEPSEEK_MODEL", "deepseek-v4-flash"),
		AITimeoutSeconds: 45,
	}

	if (cfg.Username == "") != (cfg.PasswordHash == "") {
		return Config{}, errors.New("APP_USERNAME and APP_PASSWORD_HASH must be configured together")
	}

	cfg.AIProvider = strings.ToLower(strings.TrimSpace(cfg.AIProvider))
	if cfg.AIProvider != "mock" && cfg.AIProvider != "deepseek" {
		return Config{}, fmt.Errorf("AI_PROVIDER must be mock or deepseek")
	}
	if cfg.AIProvider == "deepseek" && cfg.DeepSeekAPIKey == "" {
		return Config{}, errors.New("DEEPSEEK_API_KEY is required when AI_PROVIDER=deepseek")
	}
	if rawTimeout := strings.TrimSpace(os.Getenv("AI_TIMEOUT_SECONDS")); rawTimeout != "" {
		timeout, err := strconv.Atoi(rawTimeout)
		if err != nil || timeout <= 0 {
			return Config{}, errors.New("AI_TIMEOUT_SECONDS must be a positive integer")
		}
		cfg.AITimeoutSeconds = timeout
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
