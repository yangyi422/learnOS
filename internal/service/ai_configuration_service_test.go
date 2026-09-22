package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"learnos/internal/ai"
	"learnos/internal/config"
	"learnos/internal/model"
	"learnos/internal/repository"
)

func TestAIConfigurationUpdateActivatesDeepSeekWithoutReturningKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.AIConfiguration{}, &model.AIEvaluationRun{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	cfg := config.Config{AIProvider: "mock", DeepSeekBaseURL: ai.DefaultBaseURL, DeepSeekModel: ai.DefaultModel, AITimeoutSeconds: 1}
	runtime := ai.NewRuntimeProvider(ai.NewMockProvider())
	service := NewAIConfigurationService(repository.NewAIConfigurationRepository(db), cfg, runtime)

	view, err := service.Update(context.Background(), UpdateAIConfigurationRequest{Provider: "deepseek", APIKey: "test-secret-key", BaseURL: "https://example.test", Model: "test-model"})
	if err != nil {
		t.Fatalf("update AI configuration: %v", err)
	}
	if !view.APIKeyConfigured || view.Provider != "deepseek" || view.BaseURL != "https://example.test" || view.Model != "test-model" {
		t.Fatalf("unexpected configuration view: %+v", view)
	}
	if _, ok := runtime.CurrentProvider().(*ai.DeepSeekProvider); !ok {
		t.Fatal("expected runtime provider to be DeepSeek")
	}

	loaded, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("get AI configuration: %v", err)
	}
	if !loaded.APIKeyConfigured {
		t.Fatalf("expected configured key: %+v", loaded)
	}
	var stored model.AIConfiguration
	if err := db.First(&stored, 1).Error; err != nil {
		t.Fatalf("read stored configuration: %v", err)
	}
	if stored.APIKey != "test-secret-key" {
		t.Fatal("stored API key was not persisted")
	}
}

func TestAIConnectionUsesConfiguredProviderWithoutExposingKey(t *testing.T) {
	const secret = "connection-test-secret"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/models" || request.Header.Get("Authorization") != "Bearer "+secret {
			t.Fatalf("unexpected connection request: path=%s authorization=%q", request.URL.Path, request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIConfiguration{}, &model.AIEvaluationRun{}); err != nil {
		t.Fatal(err)
	}
	service := NewAIConfigurationService(repository.NewAIConfigurationRepository(db), config.Config{AITimeoutSeconds: 2}, ai.NewRuntimeProvider(ai.NewMockProvider()))
	if _, err := service.Update(context.Background(), UpdateAIConfigurationRequest{Provider: "deepseek", APIKey: secret, BaseURL: server.URL, Model: "test-model"}); err != nil {
		t.Fatal(err)
	}
	result, err := service.TestConnection(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" || result.Model != "test-model" || result.Provider != "deepseek" {
		t.Fatalf("unexpected test result: %+v", result)
	}
}

func TestAIConfigurationKeepsExistingKeyWhenInputIsBlank(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.AIConfiguration{}, &model.AIEvaluationRun{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	cfg := config.Config{AIProvider: "mock", DeepSeekBaseURL: ai.DefaultBaseURL, DeepSeekModel: ai.DefaultModel}
	runtime := ai.NewRuntimeProvider(ai.NewMockProvider())
	service := NewAIConfigurationService(repository.NewAIConfigurationRepository(db), cfg, runtime)
	if _, err := service.Update(context.Background(), UpdateAIConfigurationRequest{Provider: "deepseek", APIKey: "keep-me"}); err != nil {
		t.Fatalf("initial update: %v", err)
	}
	if _, err := service.Update(context.Background(), UpdateAIConfigurationRequest{Provider: "deepseek"}); err != nil {
		t.Fatalf("blank key update should retain key: %v", err)
	}
	var stored model.AIConfiguration
	if err := db.First(&stored, 1).Error; err != nil {
		t.Fatalf("read stored configuration: %v", err)
	}
	if stored.APIKey != "keep-me" {
		t.Fatalf("existing key was overwritten: %q", stored.APIKey)
	}
}
