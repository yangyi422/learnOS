package config

import (
	"strings"
	"testing"
)

func TestLoadDefaultsToMockProvider(t *testing.T) {
	t.Setenv("APP_DATA_DIR", t.TempDir())
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_ADDR", "")
	t.Setenv("APP_USERNAME", "")
	t.Setenv("APP_PASSWORD_HASH", "")
	t.Setenv("AI_PROVIDER", "mock")
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("AI_TIMEOUT_SECONDS", "")
	t.Setenv("AI_CHALLENGE_GENERATION_TIMEOUT_SECONDS", "")
	t.Setenv("AI_CURRICULUM_DRAFT_TIMEOUT_SECONDS", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AIProvider != "mock" || cfg.AITimeoutSeconds != 45 || cfg.ChallengeGenerationTimeoutSeconds != 60 || cfg.CurriculumDraftTimeoutSeconds != 60 || cfg.DomainInitialWorldTimeoutSeconds != 180 {
		t.Fatalf("unexpected AI defaults: %+v", cfg)
	}
	if cfg.Address != "127.0.0.1:8080" || !cfg.Development() {
		t.Fatalf("unexpected development defaults: %+v", cfg)
	}
}

func TestLoadRejectsNonLoopbackDevelopmentAddress(t *testing.T) {
	t.Setenv("APP_DATA_DIR", t.TempDir())
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_ADDR", "0.0.0.0:8080")
	t.Setenv("APP_USERNAME", "")
	t.Setenv("APP_PASSWORD_HASH", "")
	err := func() error {
		_, err := Load()
		return err
	}()
	if err == nil || !strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("expected loopback address validation error, got %v", err)
	}
}

func TestLoadChallengeGenerationTimeoutOverride(t *testing.T) {
	t.Setenv("APP_DATA_DIR", t.TempDir())
	t.Setenv("APP_USERNAME", "")
	t.Setenv("APP_PASSWORD_HASH", "")
	t.Setenv("AI_PROVIDER", "mock")
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("AI_TIMEOUT_SECONDS", "45")
	t.Setenv("AI_CHALLENGE_GENERATION_TIMEOUT_SECONDS", "75")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AITimeoutSeconds != 45 || cfg.ChallengeGenerationTimeoutSeconds != 75 {
		t.Fatalf("timeouts are not independent: %+v", cfg)
	}
}

func TestLoadRejectsInvalidChallengeGenerationTimeout(t *testing.T) {
	t.Setenv("APP_DATA_DIR", t.TempDir())
	t.Setenv("APP_USERNAME", "")
	t.Setenv("APP_PASSWORD_HASH", "")
	t.Setenv("AI_PROVIDER", "mock")
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("AI_CHALLENGE_GENERATION_TIMEOUT_SECONDS", "0")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid challenge generation timeout error")
	}
}

func TestLoadDeepSeekRequiresAPIKey(t *testing.T) {
	t.Setenv("APP_DATA_DIR", t.TempDir())
	t.Setenv("APP_USERNAME", "")
	t.Setenv("APP_PASSWORD_HASH", "")
	t.Setenv("AI_PROVIDER", "deepseek")
	t.Setenv("DEEPSEEK_API_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected missing DeepSeek API key error")
	}
}
