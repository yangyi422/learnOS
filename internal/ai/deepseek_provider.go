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
	client  *http.Client
	baseURL string
	apiKey  string
	model   string
	timeout time.Duration
}

func NewDeepSeekProvider(client *http.Client, baseURL, apiKey, model string, timeout time.Duration) *DeepSeekProvider {
	if client == nil {
		client = &http.Client{}
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}
	if strings.TrimSpace(model) == "" {
		model = DefaultModel
	}
	return &DeepSeekProvider{
		client:  client,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  strings.TrimSpace(apiKey),
		model:   model,
		timeout: timeout,
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
		select {
		case <-requestContext.Done():
			break
		case <-time.After(100 * time.Millisecond):
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

func (p *DeepSeekProvider) evaluateOnce(ctx context.Context, req EvaluationRequest) (EvaluationResult, ProviderMeta, bool, error) {
	body, err := json.Marshal(chatCompletionRequest{
		Model: p.model,
		Messages: []chatMessage{
			{Role: "system", Content: BuildEvaluationSystemPrompt()},
			{Role: "user", Content: BuildEvaluationUserPrompt(req)},
		},
		ResponseFormat: responseFormat{Type: "json_object"},
		Temperature:    0.2,
		MaxTokens:      2000,
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
		return EvaluationResult{}, ProviderMeta{}, true, newProviderError(ErrProvider, err)
	}
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		return EvaluationResult{}, ProviderMeta{}, true, newProviderError(ErrProvider, fmt.Errorf("read provider response"))
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		retry := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
		return EvaluationResult{}, ProviderMeta{}, retry, &ProviderError{Code: ErrProvider, StatusCode: response.StatusCode}
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
