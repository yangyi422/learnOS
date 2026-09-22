package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxAttempts = 2

type DeepSeekProvider struct {
	client                     *http.Client
	baseURL                    string
	apiKey                     string
	model                      string
	timeout                    time.Duration
	challengeGenerationTimeout time.Duration
	curriculumDraftTimeout     time.Duration
	domainSkeletonTimeout      time.Duration
	domainStarterTimeout       time.Duration
	domainInitialWorldTimeout  time.Duration
}

// NewDeepSeekProvider keeps the ordinary AI timeout as its fifth argument.
// The optional sixth and seventh arguments are independent Challenge and Curriculum Draft budgets.
func NewDeepSeekProvider(client *http.Client, baseURL, apiKey, model string, timeout time.Duration, challengeGenerationTimeout ...time.Duration) *DeepSeekProvider {
	if client == nil {
		client = &http.Client{}
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}
	if strings.TrimSpace(model) == "" {
		model = DefaultModel
	}
	generationTimeout := 60 * time.Second
	if len(challengeGenerationTimeout) > 0 && challengeGenerationTimeout[0] > 0 {
		generationTimeout = challengeGenerationTimeout[0]
	}
	curriculumTimeout := 60 * time.Second
	if len(challengeGenerationTimeout) > 1 && challengeGenerationTimeout[1] > 0 {
		curriculumTimeout = challengeGenerationTimeout[1]
	}
	domainSkeletonTimeout := 120 * time.Second
	if len(challengeGenerationTimeout) > 2 && challengeGenerationTimeout[2] > 0 {
		domainSkeletonTimeout = challengeGenerationTimeout[2]
	}
	domainStarterTimeout := 60 * time.Second
	if len(challengeGenerationTimeout) > 3 && challengeGenerationTimeout[3] > 0 {
		domainStarterTimeout = challengeGenerationTimeout[3]
	}
	domainInitialWorldTimeout := 180 * time.Second
	if len(challengeGenerationTimeout) > 4 && challengeGenerationTimeout[4] > 0 {
		domainInitialWorldTimeout = challengeGenerationTimeout[4]
	}
	return &DeepSeekProvider{
		client:                     client,
		baseURL:                    strings.TrimRight(baseURL, "/"),
		apiKey:                     strings.TrimSpace(apiKey),
		model:                      model,
		timeout:                    timeout,
		challengeGenerationTimeout: generationTimeout,
		curriculumDraftTimeout:     curriculumTimeout,
		domainSkeletonTimeout:      domainSkeletonTimeout,
		domainStarterTimeout:       domainStarterTimeout,
		domainInitialWorldTimeout:  domainInitialWorldTimeout,
	}
}

type chatCompletionRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	ResponseFormat responseFormat `json:"response_format"`
	Temperature    float64        `json:"temperature"`
	MaxTokens      int            `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (p *DeepSeekProvider) EvaluateLessonAnswer(ctx context.Context, req EvaluationRequest) (EvaluationResult, ProviderMeta, error) {
	meta := ProviderMeta{Provider: "deepseek", Model: p.model, PromptVersion: PromptVersion}
	if p.apiKey == "" {
		return EvaluationResult{}, meta, newProviderError(ErrNotConfigured, nil)
	}
	if p.timeout <= 0 {
		p.timeout = 45 * time.Second
	}
	requestContext, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	started := time.Now()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		meta.AttemptCount = attempt
		result, attemptMeta, retry, err := p.evaluateOnce(requestContext, req)
		meta.RawResponse = attemptMeta.RawResponse
		meta.InputTokens = attemptMeta.InputTokens
		meta.OutputTokens = attemptMeta.OutputTokens
		if err == nil {
			meta.LatencyMS = time.Since(started).Milliseconds()
			return result, meta, nil
		}
		lastErr = err
		if !retry || attempt == maxAttempts {
			break
		}
		if !waitForRetry(requestContext) {
			break
		}
	}
	meta.LatencyMS = time.Since(started).Milliseconds()
	if errors.Is(requestContext.Err(), context.DeadlineExceeded) {
		return EvaluationResult{}, meta, newProviderError(ErrTimeout, nil)
	}
	if errors.Is(requestContext.Err(), context.Canceled) {
		return EvaluationResult{}, meta, requestContext.Err()
	}
	return EvaluationResult{}, meta, lastErr
}

func waitForRetry(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(100 * time.Millisecond):
		return true
	}
}

func (p *DeepSeekProvider) evaluateOnce(ctx context.Context, req EvaluationRequest) (EvaluationResult, ProviderMeta, bool, error) {
	body, err := json.Marshal(chatCompletionRequest{
		Model: p.model,
		Messages: []chatMessage{
			{Role: "system", Content: BuildEvaluationSystemPrompt()},
			{Role: "user", Content: BuildEvaluationUserPrompt(req)},
		},
		ResponseFormat: responseFormat{Type: "json_object"},
		Temperature:    0.2,
		MaxTokens:      20000,
	})
	if err != nil {
		return EvaluationResult{}, ProviderMeta{}, false, newProviderError(ErrProvider, err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return EvaluationResult{}, ProviderMeta{}, false, newProviderError(ErrProvider, err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+p.apiKey)
	response, err := p.client.Do(httpRequest)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return EvaluationResult{}, ProviderMeta{}, false, newProviderError(ErrTimeout, nil)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return EvaluationResult{}, ProviderMeta{}, false, ctx.Err()
		}
		return EvaluationResult{}, ProviderMeta{}, true, newProviderError(ErrNetworkError, err)
	}
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		return EvaluationResult{}, ProviderMeta{}, true, newProviderError(ErrProvider, fmt.Errorf("read provider response"))
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		retry := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
		code := ErrProvider
		if response.StatusCode == http.StatusTooManyRequests {
			code = ErrRateLimited
		}
		return EvaluationResult{}, ProviderMeta{}, retry, &ProviderError{Code: code, StatusCode: response.StatusCode}
	}
	var completion chatCompletionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return EvaluationResult{}, ProviderMeta{RawResponse: string(responseBody)}, true, newProviderError(ErrInvalidResponse, fmt.Errorf("decode provider response"))
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return EvaluationResult{}, ProviderMeta{RawResponse: string(responseBody), InputTokens: completion.Usage.PromptTokens, OutputTokens: completion.Usage.CompletionTokens}, true, newProviderError(ErrInvalidResponse, fmt.Errorf("empty provider content"))
	}
	content := completion.Choices[0].Message.Content
	result, err := ParseAndValidateForTarget(content, req.AssessmentTargetLevel)
	if err != nil {
		return EvaluationResult{}, ProviderMeta{RawResponse: content, InputTokens: completion.Usage.PromptTokens, OutputTokens: completion.Usage.CompletionTokens}, true, err
	}
	return result, ProviderMeta{
		RawResponse:  content,
		InputTokens:  completion.Usage.PromptTokens,
		OutputTokens: completion.Usage.CompletionTokens,
	}, false, nil
}
