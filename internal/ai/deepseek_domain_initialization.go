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

func (p *DeepSeekProvider) GenerateDomainSkeleton(ctx context.Context, req DomainSkeletonRequest) (DomainSkeletonResult, ProviderMeta, error) {
	meta := ProviderMeta{Provider: "deepseek", Model: p.model, PromptVersion: DomainSkeletonPromptVersion}
	content, meta, err := p.domainRequestWithRetry(ctx, meta, p.domainSkeletonTimeout, "skeleton", BuildDomainSkeletonSystemPrompt(), BuildDomainSkeletonUserPrompt(req), func(raw string) (interface{}, error) { return ParseAndValidateDomainSkeleton(raw) })
	if err != nil {
		return DomainSkeletonResult{}, meta, err
	}
	result, err := ParseAndValidateDomainSkeleton(content)
	if err != nil {
		return DomainSkeletonResult{}, meta, err
	}
	return result, meta, nil
}

func (p *DeepSeekProvider) ExpandDomainStarter(ctx context.Context, req DomainStarterRequest) (DomainStarterResult, ProviderMeta, error) {
	meta := ProviderMeta{Provider: "deepseek", Model: p.model, PromptVersion: DomainStarterPromptVersion}
	content, meta, err := p.domainRequestWithRetry(ctx, meta, p.domainStarterTimeout, "starter_blueprint", BuildDomainStarterSystemPrompt(), BuildDomainStarterUserPrompt(req), func(raw string) (interface{}, error) {
		return ParseAndValidateDomainStarter(raw, req.Skeleton, req.SelectedStarterKeys)
	})
	if err != nil {
		return DomainStarterResult{}, meta, err
	}
	result, err := ParseAndValidateDomainStarter(content, req.Skeleton, req.SelectedStarterKeys)
	if err != nil {
		return DomainStarterResult{}, meta, err
	}
	return result, meta, nil
}

func (p *DeepSeekProvider) GenerateInitialWorld(ctx context.Context, req InitialWorldRequest) (InitialWorldResult, ProviderMeta, error) {
	meta := ProviderMeta{Provider: "deepseek", Model: p.model, PromptVersion: DomainInitialWorldPromptVersion}
	content, meta, err := p.domainRequestWithRetry(ctx, meta, p.domainInitialWorldTimeout, "initial_world", BuildInitialWorldSystemPrompt(), BuildInitialWorldUserPrompt(req), func(raw string) (interface{}, error) { return ParseAndValidateInitialWorld(raw, req.Starter) })
	if err != nil {
		return InitialWorldResult{}, meta, err
	}
	result, err := ParseAndValidateInitialWorld(content, req.Starter)
	if err != nil {
		return InitialWorldResult{}, meta, err
	}
	return result, meta, nil
}

func (p *DeepSeekProvider) ExpandBlueprintUnit(ctx context.Context, req BlueprintUnitExpansionRequest) (BlueprintUnitExpansionResult, ProviderMeta, error) {
	meta := ProviderMeta{Provider: "deepseek", Model: p.model, PromptVersion: BlueprintUnitExpansionPromptVersion}
	content, meta, err := p.domainRequestWithRetry(ctx, meta, p.domainStarterTimeout, "blueprint_unit_expansion", BuildBlueprintUnitExpansionSystemPrompt(), BuildBlueprintUnitExpansionUserPrompt(req), func(raw string) (interface{}, error) {
		return ParseAndValidateBlueprintUnitExpansion(raw, req.ExistingLessonKeys)
	})
	if err != nil {
		return BlueprintUnitExpansionResult{}, meta, err
	}
	result, err := ParseAndValidateBlueprintUnitExpansion(content, req.ExistingLessonKeys)
	if err != nil {
		return BlueprintUnitExpansionResult{}, meta, err
	}
	return result, meta, nil
}

func (p *DeepSeekProvider) domainRequestWithRetry(ctx context.Context, meta ProviderMeta, timeout time.Duration, stage, systemPrompt, userPrompt string, validate func(string) (interface{}, error)) (string, ProviderMeta, error) {
	if p.apiKey == "" {
		return "", meta, newProviderError(ErrNotConfigured, nil)
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	requestContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := time.Now()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		meta.AttemptCount = attempt
		attemptUserPrompt := userPrompt
		if attempt > 1 {
			attemptUserPrompt += "\n这是一次校正重试。上一次输出未通过服务端 JSON 结构校验；请严格按照系统消息中的字段白名单和数量限制重新生成，只返回一个 JSON object。"
		}
		content, responseMeta, retry, err := p.domainRequestOnce(requestContext, systemPrompt, attemptUserPrompt)
		meta.RawResponse, meta.InputTokens, meta.OutputTokens = responseMeta.RawResponse, responseMeta.InputTokens, responseMeta.OutputTokens
		if err == nil {
			if _, validationErr := validate(content); validationErr == nil {
				meta.LatencyMS = time.Since(started).Milliseconds()
				return content, meta, nil
			} else {
				err = validationErr
				retry = true
			}
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
		return "", meta, newProviderError(ErrTimeout, requestContext.Err())
	}
	if errors.Is(requestContext.Err(), context.Canceled) {
		return "", meta, requestContext.Err()
	}
	return "", meta, lastErr
}

func (p *DeepSeekProvider) domainRequestOnce(ctx context.Context, systemPrompt, userPrompt string) (string, ProviderMeta, bool, error) {
	body, err := json.Marshal(chatCompletionRequest{Model: p.model, Messages: []chatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userPrompt}}, ResponseFormat: responseFormat{Type: "json_object"}, Temperature: 0.2, MaxTokens: 50000})
	if err != nil {
		return "", ProviderMeta{}, false, newProviderError(ErrProvider, err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", ProviderMeta{}, false, newProviderError(ErrProvider, err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+p.apiKey)
	response, err := p.client.Do(request)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", ProviderMeta{}, false, newProviderError(ErrTimeout, nil)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return "", ProviderMeta{}, false, ctx.Err()
		}
		return "", ProviderMeta{}, true, newProviderError(ErrNetworkError, err)
	}
	defer response.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 3<<20))
	if readErr != nil {
		return "", ProviderMeta{}, true, newProviderError(ErrNetworkError, fmt.Errorf("read domain provider response"))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		code := ErrProvider
		if response.StatusCode == http.StatusTooManyRequests {
			code = ErrRateLimited
		}
		return "", ProviderMeta{}, response.StatusCode == 429 || response.StatusCode >= 500, &ProviderError{Code: code, StatusCode: response.StatusCode}
	}
	var completion chatCompletionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return "", ProviderMeta{RawResponse: string(responseBody)}, true, newProviderError(ErrInvalidResponse, fmt.Errorf("decode domain provider response"))
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return "", ProviderMeta{RawResponse: string(responseBody)}, true, newProviderError(ErrEmptyContent, fmt.Errorf("empty domain provider content"))
	}
	return completion.Choices[0].Message.Content, ProviderMeta{RawResponse: completion.Choices[0].Message.Content, InputTokens: completion.Usage.PromptTokens, OutputTokens: completion.Usage.CompletionTokens}, false, nil
}
