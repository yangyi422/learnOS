package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	AppName                              string
	AppVersion                           string
	Environment                          string
	DemoSeedEnabled                      bool
	Address                              string
	DataDir                              string
	DatabasePath                         string
	BackupDir                            string
	BackupRetentionCount                 int
	Username                             string
	PasswordHash                         string
	AIProvider                           string
	DeepSeekAPIKey                       string
	DeepSeekBaseURL                      string
	DeepSeekModel                        string
	AITimeoutSeconds                     int
	ChallengeGenerationTimeoutSeconds    int
	CurriculumDraftTimeoutSeconds        int
	SourceCredibilityTimeoutSeconds      int
	DomainSkeletonTimeoutSeconds         int
	DomainStarterBlueprintTimeoutSeconds int
	DomainInitialWorldTimeoutSeconds     int
}

func Load() (Config, error) {
	environment := envOrDefault("APP_ENV", "development")
	defaultAddress := ":8080"
	if strings.EqualFold(environment, "development") {
		defaultAddress = "127.0.0.1:8080"
	}
	demoSeedEnabled := strings.EqualFold(environment, "development")
	if raw := strings.TrimSpace(os.Getenv("DEMO_SEED_ENABLED")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, errors.New("DEMO_SEED_ENABLED must be true or false")
		}
		demoSeedEnabled = parsed
	}
	dataDir := strings.TrimSpace(os.Getenv("APP_DATA_DIR"))
	if dataDir == "" {
		if strings.EqualFold(environment, "production") {
			dataDir = "./data/prod"
		} else {
			dataDir = "./data/dev"
		}
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return Config{}, err
	}
	backupDir := envOrDefault("BACKUP_DIR", filepath.Join(dataDir, "backups"))
	retention := 20
	if raw := strings.TrimSpace(os.Getenv("BACKUP_RETENTION_COUNT")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return Config{}, errors.New("BACKUP_RETENTION_COUNT must be a positive integer")
		}
		retention = parsed
	}

	cfg := Config{
		AppName:                              envOrDefault("APP_NAME", "LearnOS"),
		AppVersion:                           envOrDefault("APP_VERSION", "0.1.0"),
		Environment:                          environment,
		DemoSeedEnabled:                      demoSeedEnabled,
		Address:                              envOrDefault("APP_ADDR", defaultAddress),
		DataDir:                              dataDir,
		DatabasePath:                         filepath.Join(dataDir, "learnos.db"),
		BackupDir:                            backupDir,
		BackupRetentionCount:                 retention,
		Username:                             strings.TrimSpace(os.Getenv("APP_USERNAME")),
		PasswordHash:                         strings.TrimSpace(os.Getenv("APP_PASSWORD_HASH")),
		AIProvider:                           envOrDefault("AI_PROVIDER", "mock"),
		DeepSeekAPIKey:                       strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")),
		DeepSeekBaseURL:                      envOrDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		DeepSeekModel:                        envOrDefault("DEEPSEEK_MODEL", "deepseek-v4-flash"),
		AITimeoutSeconds:                     45,
		ChallengeGenerationTimeoutSeconds:    60,
		CurriculumDraftTimeoutSeconds:        60,
		SourceCredibilityTimeoutSeconds:      45,
		DomainSkeletonTimeoutSeconds:         120,
		DomainStarterBlueprintTimeoutSeconds: 60,
		DomainInitialWorldTimeoutSeconds:     180,
	}

	if (cfg.Username == "") != (cfg.PasswordHash == "") {
		return Config{}, errors.New("APP_USERNAME and APP_PASSWORD_HASH must be configured together")
	}
	if cfg.Development() {
		host, _, err := net.SplitHostPort(cfg.Address)
		if err != nil || host != "127.0.0.1" {
			return Config{}, errors.New("APP_ADDR must bind to 127.0.0.1 in development")
		}
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
	if rawTimeout := strings.TrimSpace(os.Getenv("AI_CHALLENGE_GENERATION_TIMEOUT_SECONDS")); rawTimeout != "" {
		timeout, err := strconv.Atoi(rawTimeout)
		if err != nil || timeout <= 0 {
			return Config{}, errors.New("AI_CHALLENGE_GENERATION_TIMEOUT_SECONDS must be a positive integer")
		}
		cfg.ChallengeGenerationTimeoutSeconds = timeout
	}
	if rawTimeout := strings.TrimSpace(os.Getenv("AI_CURRICULUM_DRAFT_TIMEOUT_SECONDS")); rawTimeout != "" {
		timeout, err := strconv.Atoi(rawTimeout)
		if err != nil || timeout <= 0 {
			return Config{}, errors.New("AI_CURRICULUM_DRAFT_TIMEOUT_SECONDS must be a positive integer")
		}
		cfg.CurriculumDraftTimeoutSeconds = timeout
	}
	if rawTimeout := strings.TrimSpace(os.Getenv("AI_SOURCE_CREDIBILITY_TIMEOUT_SECONDS")); rawTimeout != "" {
		timeout, err := strconv.Atoi(rawTimeout)
		if err != nil || timeout <= 0 {
			return Config{}, errors.New("AI_SOURCE_CREDIBILITY_TIMEOUT_SECONDS must be a positive integer")
		}
		cfg.SourceCredibilityTimeoutSeconds = timeout
	}
	for _, item := range []struct {
		env    string
		target *int
	}{
		{"AI_DOMAIN_SKELETON_TIMEOUT_SECONDS", &cfg.DomainSkeletonTimeoutSeconds},
		{"AI_DOMAIN_STARTER_BLUEPRINT_TIMEOUT_SECONDS", &cfg.DomainStarterBlueprintTimeoutSeconds},
		{"AI_DOMAIN_INITIAL_WORLD_TIMEOUT_SECONDS", &cfg.DomainInitialWorldTimeoutSeconds},
	} {
		if rawTimeout := strings.TrimSpace(os.Getenv(item.env)); rawTimeout != "" {
			timeout, err := strconv.Atoi(rawTimeout)
			if err != nil || timeout <= 0 {
				return Config{}, fmt.Errorf("%s must be a positive integer", item.env)
			}
			*item.target = timeout
		}
	}

	return cfg, nil
}

func (c Config) Production() bool {
	return strings.EqualFold(c.Environment, "production")
}

func (c Config) Development() bool {
	return strings.EqualFold(c.Environment, "development")
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
