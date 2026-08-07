package config

import "testing"

func TestLoadDefaultsToMockProvider(t *testing.T) {
	t.Setenv("APP_DATA_DIR", t.TempDir())
	t.Setenv("APP_USERNAME", "")
	t.Setenv("APP_PASSWORD_HASH", "")
	t.Setenv("AI_PROVIDER", "mock")
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("AI_TIMEOUT_SECONDS", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AIProvider != "mock" || cfg.AITimeoutSeconds != 45 {
		t.Fatalf("unexpected AI defaults: %+v", cfg)
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
