package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

func (p *DeepSeekProvider) GenerateChallenge(ctx context.Context, req ChallengeGenerationRequest) (ChallengeGenerationResult, ProviderMeta, error) {
	meta := ProviderMeta{Provider: "deepseek", Model: p.model, PromptVersion: TransferGeneratorPromptVersion}
	if p.apiKey == "" {
		return ChallengeGenerationResult{}, meta, newProviderError(ErrNotConfigured, nil)
	}
	requestContext, cancel := context.WithTimeout(ctx, p.challengeGenerationTimeoutDuration())
	defer cancel()
	started := time.Now()
	systemPrompt := BuildChallengeGenerationSystemPrompt()
	userPrompt := BuildChallengeGenerationUserPrompt(req)
	promptChars := int64(utf8.RuneCountInString(systemPrompt + userPrompt))
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		meta.AttemptCount = attempt
		providerStarted := time.Now()
		content, responseMeta, retry, err := p.requestJSON(requestContext, systemPrompt, userPrompt)
		providerLatency := time.Since(providerStarted).Milliseconds()
		meta.RawResponse, meta.InputTokens, meta.OutputTokens = responseMeta.RawResponse, responseMeta.InputTokens, responseMeta.OutputTokens
		if err == nil {
			result, validationErr := parseChallengeGeneration(content, req.ChallengeType)
			if validationErr == nil {
				logChallengeGenerationAttempt(req.ChallengeType, meta, attempt, providerLatency, promptChars, content, "success", nil, time.Since(started).Milliseconds())
				log.Printf("challenge generation completed: challenge_type=%s attempt_count=%d total_latency_ms=%d", req.ChallengeType, attempt, time.Since(started).Milliseconds())
				meta.LatencyMS = time.Since(started).Milliseconds()
				return result, meta, nil
			}
			err = validationErr
			retry = true
		}
		logChallengeGenerationAttempt(req.ChallengeType, meta, attempt, providerLatency, promptChars, content, "failed", err, time.Since(started).Milliseconds())
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
		return ChallengeGenerationResult{}, meta, newProviderError(ErrTimeout, requestContext.Err())
	}
	return ChallengeGenerationResult{}, meta, lastErr
}

func logChallengeGenerationAttempt(challengeType string, meta ProviderMeta, attempt int, providerLatencyMS, promptChars int64, content, validationStatus string, err error, totalLatencyMS int64) {
	logMessage := fmt.Sprintf("challenge generation: challenge_type=%s provider=%s model=%s prompt_version=%s attempt=%d provider_latency_ms=%d prompt_chars=%d raw_content_chars=%d validation_status=%s total_latency_ms=%d", challengeType, meta.Provider, meta.Model, meta.PromptVersion, attempt, providerLatencyMS, promptChars, int64(utf8.RuneCountInString(content)), validationStatus, totalLatencyMS)
	if validationStatus == "success" {
		log.Printf("%s", logMessage)
		return
	}
	failureType := challengeFailureType(err)
	if failureType == "validation_error" {
		log.Printf("%s failure_type=%s validation_error=%q", logMessage, failureType, ErrorDetail(err))
		return
	}
	log.Printf("%s failure_type=%s error=%q", logMessage, failureType, ErrorDetail(err))
}

func (p *DeepSeekProvider) EvaluateChallenge(ctx context.Context, req ChallengeEvaluationRequest) (ChallengeEvaluationResult, ProviderMeta, error) {
	meta := ProviderMeta{Provider: "deepseek", Model: p.model, PromptVersion: ChallengeEvaluatorPromptVersion}
	if p.apiKey == "" {
		return ChallengeEvaluationResult{}, meta, newProviderError(ErrNotConfigured, nil)
	}
	requestContext, cancel := context.WithTimeout(ctx, p.evaluationTimeout())
	defer cancel()
	started := time.Now()
	var lastErr error
	targetMisconceptionID := req.TargetMisconceptionID
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		meta.AttemptCount = attempt
		content, responseMeta, retry, err := p.requestJSON(requestContext, BuildChallengeEvaluationSystemPrompt(), BuildChallengeEvaluationUserPrompt(req))
		meta.RawResponse, meta.InputTokens, meta.OutputTokens = responseMeta.RawResponse, responseMeta.InputTokens, responseMeta.OutputTokens
		if err == nil {
			result, validationErr := parseChallengeEvaluation(content, req.ChallengeType, req.TargetLevel, targetMisconceptionID)
			if validationErr == nil {
				meta.LatencyMS = time.Since(started).Milliseconds()
				return result, meta, nil
			}
			err = validationErr
			retry = true
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
		return ChallengeEvaluationResult{}, meta, newProviderError(ErrTimeout, nil)
	}
	return ChallengeEvaluationResult{}, meta, lastErr
}

func (p *DeepSeekProvider) evaluationTimeout() time.Duration {
	if p.timeout > 0 {
		return p.timeout
	}
	return 45 * time.Second
}

func (p *DeepSeekProvider) challengeGenerationTimeoutDuration() time.Duration {
	if p.challengeGenerationTimeout > 0 {
		return p.challengeGenerationTimeout
	}
	return 60 * time.Second
}

func challengeFailureType(err error) string {
	switch {
	case errors.Is(err, ErrTimeout), errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, ErrEmptyContent):
		return "empty_content"
	case errors.Is(err, ErrInvalidResponse):
		return "validation_error"
	default:
		return "provider_error"
	}
}

func parseChallengeGeneration(content, challengeType string) (ChallengeGenerationResult, error) {
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	var result ChallengeGenerationResult
	if err := decoder.Decode(&result); err != nil {
		return ChallengeGenerationResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("decode challenge generation: %w", err))
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ChallengeGenerationResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("challenge generation JSON has trailing content"))
	}
	return NormalizeAndValidateChallengeGeneration(result, challengeType)
}

func parseChallengeEvaluation(content, challengeType, targetLevel string, targetMisconceptionID uint) (ChallengeEvaluationResult, error) {
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	var wire challengeEvaluationWire
	if err := decoder.Decode(&wire); err != nil {
		return ChallengeEvaluationResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("decode challenge evaluation: %w", err))
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ChallengeEvaluationResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("challenge evaluation JSON has trailing content"))
	}
	evidence, err := normalizeChallengeEvidence(wire.CognitiveEvidence)
	if err != nil {
		return ChallengeEvaluationResult{}, newProviderError(ErrInvalidResponse, err)
	}
	result := ChallengeEvaluationResult{
		Result: wire.Result, Feedback: wire.Feedback, Explanation: wire.Explanation,
		DemonstratedLevel: wire.DemonstratedLevel, CognitiveEvidence: evidence,
		Misconceptions: wire.Misconceptions, MisconceptionValidation: wire.MisconceptionValidation,
	}
	return NormalizeAndValidateChallengeEvaluation(result, challengeType, targetLevel, targetMisconceptionID)
}

// These wire types are deliberately private to Challenge Provider parsing. The
// business DTOs and persisted snapshots only expose cognitive_level.
type challengeEvaluationWire struct {
	Result                  string                    `json:"result"`
	Feedback                string                    `json:"feedback"`
	Explanation             string                    `json:"explanation"`
	DemonstratedLevel       string                    `json:"demonstrated_level"`
	CognitiveEvidence       []challengeEvidenceWire   `json:"cognitive_evidence"`
	Misconceptions          []EvaluationMisconception `json:"misconceptions"`
	MisconceptionValidation *MisconceptionValidation  `json:"misconception_validation"`
}

type challengeEvidenceWire struct {
	EvidenceType   string  `json:"evidence_type"`
	CognitiveLevel *string `json:"cognitive_level"`
	Level          *string `json:"level"`
	Polarity       string  `json:"polarity"`
	Description    string  `json:"description"`
}

func normalizeChallengeEvidence(wire []challengeEvidenceWire) ([]EvaluationEvidence, error) {
	if wire == nil {
		return []EvaluationEvidence{}, nil
	}
	evidence := make([]EvaluationEvidence, 0, len(wire))
	for index, item := range wire {
		cognitiveLevel := ""
		hasCognitiveLevel := item.CognitiveLevel != nil
		if item.CognitiveLevel != nil {
			cognitiveLevel = strings.TrimSpace(*item.CognitiveLevel)
		}
		if item.Level != nil {
			level := strings.TrimSpace(*item.Level)
			if hasCognitiveLevel && cognitiveLevel != level {
				return nil, fmt.Errorf("cognitive_evidence[%d] cognitive_level and level conflict", index)
			}
			if !hasCognitiveLevel {
				cognitiveLevel = level
			}
		}
		evidence = append(evidence, EvaluationEvidence{EvidenceType: item.EvidenceType, CognitiveLevel: cognitiveLevel, Polarity: item.Polarity, Description: item.Description})
	}
	return evidence, nil
}

func (p *DeepSeekProvider) requestJSON(ctx context.Context, systemPrompt, userPrompt string) (string, ProviderMeta, bool, error) {
	body, err := json.Marshal(chatCompletionRequest{
		Model:          p.model,
		Messages:       []chatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userPrompt}},
		ResponseFormat: responseFormat{Type: "json_object"},
		Temperature:    0.2,
		MaxTokens:      24000,
	})
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
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			cause := ctx.Err()
			if cause == nil {
				cause = err
			}
			return "", ProviderMeta{}, false, newProviderError(ErrTimeout, cause)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return "", ProviderMeta{}, false, ctx.Err()
		}
		return "", ProviderMeta{}, true, newProviderError(ErrNetworkError, err)
	}
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(readErr, context.DeadlineExceeded) || errors.Is(closeErr, context.DeadlineExceeded) {
			cause := ctx.Err()
			if cause == nil {
				cause = readErr
				if cause == nil {
					cause = closeErr
				}
			}
			return "", ProviderMeta{}, false, newProviderError(ErrTimeout, cause)
		}
		return "", ProviderMeta{}, true, newProviderError(ErrProvider, fmt.Errorf("read challenge provider response"))
	}
	var completion chatCompletionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return "", ProviderMeta{RawResponse: string(responseBody)}, true, newProviderError(ErrInvalidResponse, fmt.Errorf("decode challenge provider response"))
	}
	responseMeta := ProviderMeta{RawResponse: string(responseBody), InputTokens: completion.Usage.PromptTokens, OutputTokens: completion.Usage.CompletionTokens}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		code := ErrProvider
		if response.StatusCode == http.StatusTooManyRequests {
			code = ErrRateLimited
		}
		return "", responseMeta, response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500, &ProviderError{Code: code, StatusCode: response.StatusCode}
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		log.Printf("AI challenge provider empty content: provider=deepseek model=%s", p.model)
		return "", responseMeta, true, newProviderError(ErrEmptyContent, fmt.Errorf("empty challenge provider content"))
	}
	return completion.Choices[0].Message.Content, ProviderMeta{RawResponse: completion.Choices[0].Message.Content, InputTokens: responseMeta.InputTokens, OutputTokens: responseMeta.OutputTokens}, false, nil
}
