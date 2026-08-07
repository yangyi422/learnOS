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
)

func (p *DeepSeekProvider) GenerateChallenge(ctx context.Context, req ChallengeGenerationRequest) (ChallengeGenerationResult, ProviderMeta, error) {
	meta := ProviderMeta{Provider: "deepseek", Model: p.model, PromptVersion: TransferGeneratorPromptVersion}
	if p.apiKey == "" {
		return ChallengeGenerationResult{}, meta, newProviderError(ErrNotConfigured, nil)
	}
	requestContext, cancel := context.WithTimeout(ctx, p.challengeTimeout())
	defer cancel()
	started := time.Now()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		meta.AttemptCount = attempt
		content, responseMeta, retry, err := p.requestJSON(requestContext, BuildChallengeGenerationSystemPrompt(), BuildChallengeGenerationUserPrompt(req))
		meta.RawResponse, meta.InputTokens, meta.OutputTokens = responseMeta.RawResponse, responseMeta.InputTokens, responseMeta.OutputTokens
		if err == nil {
			result, validationErr := parseChallengeGeneration(content, req.ChallengeType)
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
		select {
		case <-requestContext.Done():
		case <-time.After(100 * time.Millisecond):
		}
	}
	meta.LatencyMS = time.Since(started).Milliseconds()
	if errors.Is(requestContext.Err(), context.DeadlineExceeded) {
		return ChallengeGenerationResult{}, meta, newProviderError(ErrTimeout, nil)
	}
	return ChallengeGenerationResult{}, meta, lastErr
}

func (p *DeepSeekProvider) EvaluateChallenge(ctx context.Context, req ChallengeEvaluationRequest) (ChallengeEvaluationResult, ProviderMeta, error) {
	meta := ProviderMeta{Provider: "deepseek", Model: p.model, PromptVersion: ChallengeEvaluatorPromptVersion}
	if p.apiKey == "" {
		return ChallengeEvaluationResult{}, meta, newProviderError(ErrNotConfigured, nil)
	}
	requestContext, cancel := context.WithTimeout(ctx, p.challengeTimeout())
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
		select {
		case <-requestContext.Done():
		case <-time.After(100 * time.Millisecond):
		}
	}
	meta.LatencyMS = time.Since(started).Milliseconds()
	if errors.Is(requestContext.Err(), context.DeadlineExceeded) {
		return ChallengeEvaluationResult{}, meta, newProviderError(ErrTimeout, nil)
	}
	return ChallengeEvaluationResult{}, meta, lastErr
}

func (p *DeepSeekProvider) challengeTimeout() time.Duration {
	if p.timeout > 0 {
		return p.timeout
	}
	return 45 * time.Second
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
		MaxTokens:      2400,
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
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", ProviderMeta{}, false, newProviderError(ErrTimeout, nil)
		}
		return "", ProviderMeta{}, true, newProviderError(ErrProvider, err)
	}
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		return "", ProviderMeta{}, true, newProviderError(ErrProvider, fmt.Errorf("read challenge provider response"))
	}
	var completion chatCompletionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return "", ProviderMeta{RawResponse: string(responseBody)}, true, newProviderError(ErrInvalidResponse, fmt.Errorf("decode challenge provider response"))
	}
	responseMeta := ProviderMeta{RawResponse: string(responseBody), InputTokens: completion.Usage.PromptTokens, OutputTokens: completion.Usage.CompletionTokens}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", responseMeta, response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500, &ProviderError{Code: ErrProvider, StatusCode: response.StatusCode}
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		log.Printf("AI challenge provider empty content: provider=deepseek model=%s", p.model)
		return "", responseMeta, true, newProviderError(ErrInvalidResponse, fmt.Errorf("empty challenge provider content"))
	}
	return completion.Choices[0].Message.Content, ProviderMeta{RawResponse: completion.Choices[0].Message.Content, InputTokens: responseMeta.InputTokens, OutputTokens: responseMeta.OutputTokens}, false, nil
}
