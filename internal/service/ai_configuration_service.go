package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
	"learnos/internal/ai"
	"learnos/internal/config"
	"learnos/internal/model"
	"learnos/internal/repository"
)

var (
	ErrAIConfigurationInvalid = errors.New("invalid AI configuration")
	ErrAIConnectionFailed     = errors.New("AI connection failed")
)

type AIConfigurationService struct {
	repository *repository.AIConfigurationRepository
	cfg        config.Config
	runtime    *ai.RuntimeProvider
}

type AIConfigurationView struct {
	Provider             string     `json:"provider"`
	APIKeyConfigured     bool       `json:"api_key_configured"`
	BaseURL              string     `json:"base_url"`
	Model                string     `json:"model"`
	EffectiveProvider    string     `json:"effective_provider"`
	EffectiveModel       string     `json:"effective_model"`
	LastSuccessfulCallAt *time.Time `json:"last_successful_call_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type AIConnectionTestView struct {
	Status    string    `json:"status"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	CheckedAt time.Time `json:"checked_at"`
	LatencyMS int64     `json:"latency_ms"`
}

type UpdateAIConfigurationRequest struct {
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	ClearAPIKey bool   `json:"clear_api_key"`
	BaseURL     string `json:"base_url"`
	Model       string `json:"model"`
}

func NewAIConfigurationService(repo *repository.AIConfigurationRepository, cfg config.Config, runtime *ai.RuntimeProvider) *AIConfigurationService {
	return &AIConfigurationService{repository: repo, cfg: cfg, runtime: runtime}
}

// LoadPersisted activates the saved configuration during application startup.
// A missing row means the environment configuration remains active.
func (s *AIConfigurationService) LoadPersisted(ctx context.Context) error {
	configuration, err := s.repository.Find(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read AI configuration: %w", err)
	}
	provider, err := s.buildProvider(configuration.Provider, configuration.APIKey, configuration.BaseURL, configuration.Model)
	if err != nil {
		return err
	}
	s.runtime.SetProvider(provider)
	return nil
}

func (s *AIConfigurationService) Get(ctx context.Context) (*AIConfigurationView, error) {
	configuration, err := s.repository.Find(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		view := &AIConfigurationView{Provider: s.cfg.AIProvider, APIKeyConfigured: s.cfg.DeepSeekAPIKey != "", BaseURL: s.cfg.DeepSeekBaseURL, Model: s.cfg.DeepSeekModel}
		return s.enrichView(ctx, view)
	}
	if err != nil {
		return nil, fmt.Errorf("read AI configuration: %w", err)
	}
	return s.enrichView(ctx, configurationView(configuration))
}

func (s *AIConfigurationService) Update(ctx context.Context, request UpdateAIConfigurationRequest) (*AIConfigurationView, error) {
	existing, err := s.repository.Find(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("read AI configuration: %w", err)
	}
	providerName := strings.ToLower(strings.TrimSpace(request.Provider))
	if providerName == "" {
		providerName = "mock"
	}
	baseURL := strings.TrimSpace(request.BaseURL)
	if baseURL == "" {
		baseURL = s.cfg.DeepSeekBaseURL
	}
	modelName := strings.TrimSpace(request.Model)
	if modelName == "" {
		modelName = s.cfg.DeepSeekModel
	}
	apiKey := strings.TrimSpace(request.APIKey)
	if apiKey == "" && existing != nil && !request.ClearAPIKey {
		apiKey = existing.APIKey
	}
	if providerName == "deepseek" && apiKey == "" {
		return nil, ErrAIConfigurationInvalid
	}
	provider, err := s.buildProvider(providerName, apiKey, baseURL, modelName)
	if err != nil {
		return nil, err
	}
	configuration := &model.AIConfiguration{ID: 1, Provider: providerName, APIKey: apiKey, BaseURL: baseURL, Model: modelName}
	if err := s.repository.Save(ctx, configuration); err != nil {
		return nil, fmt.Errorf("save AI configuration: %w", err)
	}
	s.runtime.SetProvider(provider)
	return s.Get(ctx)
}

func (s *AIConfigurationService) TestConnection(ctx context.Context) (*AIConnectionTestView, error) {
	providerName, apiKey, baseURL, modelName, err := s.effectiveConfiguration(ctx)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	if providerName == "mock" {
		return &AIConnectionTestView{Status: "ok", Provider: "mock", Model: "mock", CheckedAt: time.Now().UTC(), LatencyMS: time.Since(started).Milliseconds()}, nil
	}
	if providerName != "deepseek" || strings.TrimSpace(apiKey) == "" {
		return nil, ErrAIConfigurationInvalid
	}
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, ErrAIConfigurationInvalid
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/models", nil)
	if err != nil {
		return nil, ErrAIConfigurationInvalid
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: time.Duration(s.cfg.AITimeoutSeconds) * time.Second}
	if client.Timeout <= 0 || client.Timeout > 30*time.Second {
		client.Timeout = 15 * time.Second
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, ErrAIConnectionFailed
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, ErrAIConnectionFailed
	}
	return &AIConnectionTestView{Status: "ok", Provider: providerName, Model: modelName, CheckedAt: time.Now().UTC(), LatencyMS: time.Since(started).Milliseconds()}, nil
}

func (s *AIConfigurationService) effectiveConfiguration(ctx context.Context) (provider, apiKey, baseURL, modelName string, err error) {
	configuration, findErr := s.repository.Find(ctx)
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		return s.cfg.AIProvider, s.cfg.DeepSeekAPIKey, s.cfg.DeepSeekBaseURL, s.cfg.DeepSeekModel, nil
	}
	if findErr != nil {
		return "", "", "", "", findErr
	}
	return configuration.Provider, configuration.APIKey, configuration.BaseURL, configuration.Model, nil
}

func (s *AIConfigurationService) enrichView(ctx context.Context, view *AIConfigurationView) (*AIConfigurationView, error) {
	view.EffectiveProvider = view.Provider
	view.EffectiveModel = view.Model
	if view.Provider == "mock" {
		view.EffectiveModel = "mock"
	}
	lastSuccess, err := s.repository.LastSuccessfulCallAt(ctx)
	if err != nil {
		return nil, fmt.Errorf("read latest successful AI call: %w", err)
	}
	view.LastSuccessfulCallAt = lastSuccess
	return view, nil
}

func (s *AIConfigurationService) buildProvider(providerName, apiKey, baseURL, modelName string) (ai.AIProvider, error) {
	switch providerName {
	case "mock":
		return ai.NewMockProvider(), nil
	case "deepseek":
		if strings.TrimSpace(apiKey) == "" {
			return nil, ErrAIConfigurationInvalid
		}
		return ai.NewDeepSeekProvider(&http.Client{}, baseURL, apiKey, modelName,
			time.Duration(s.cfg.AITimeoutSeconds)*time.Second,
			time.Duration(s.cfg.ChallengeGenerationTimeoutSeconds)*time.Second,
			time.Duration(s.cfg.CurriculumDraftTimeoutSeconds)*time.Second,
			time.Duration(s.cfg.DomainSkeletonTimeoutSeconds)*time.Second,
			time.Duration(s.cfg.DomainStarterBlueprintTimeoutSeconds)*time.Second,
			time.Duration(s.cfg.DomainInitialWorldTimeoutSeconds)*time.Second), nil
	default:
		return nil, ErrAIConfigurationInvalid
	}
}

func configurationView(configuration *model.AIConfiguration) *AIConfigurationView {
	return &AIConfigurationView{Provider: configuration.Provider, APIKeyConfigured: strings.TrimSpace(configuration.APIKey) != "", BaseURL: configuration.BaseURL, Model: configuration.Model, UpdatedAt: configuration.UpdatedAt}
}
