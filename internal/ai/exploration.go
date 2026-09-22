package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ExplorationRequest struct {
	SourceCourseName string
	SourceLesson     string
	TargetCourseName string
	TargetLesson     string
	DirectionType    string
	ReasonCode       string
	ReasonData       map[string]interface{}
	Why              string
}

type ExplorationResult struct {
	Summary           string `json:"summary"`
	WhyWorthExploring string `json:"why_worth_exploring"`
}

func BuildExplorationSystemPrompt() string {
	return `你是 LearnOS 的探索方向编辑器。请根据给定的知识节点关系，生成简短、具体、不过度承诺的探索说明。只输出 JSON，不要输出 Markdown。JSON 必须包含 summary 和 why_worth_exploring 两个字符串字段。不要声称用户已经掌握目标知识，也不要建议修改课程主线。`
}

func BuildExplorationUserPrompt(req ExplorationRequest) string {
	reason, _ := json.Marshal(req.ReasonData)
	return fmt.Sprintf("源课程：%s\n源知识点：%s\n目标课程：%s\n目标知识点：%s\n方向类型：%s\n原因：%s\n原因数据：%s\n规则说明：%s\n请生成适合学习雷达展示的 summary（不超过80字）和 why_worth_exploring（不超过120字）。", req.SourceCourseName, req.SourceLesson, req.TargetCourseName, req.TargetLesson, req.DirectionType, req.ReasonCode, string(reason), req.Why)
}

func ParseAndValidateExploration(content string) (ExplorationResult, error) {
	var result ExplorationResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return ExplorationResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("decode exploration response"))
	}
	result.Summary = strings.TrimSpace(result.Summary)
	result.WhyWorthExploring = strings.TrimSpace(result.WhyWorthExploring)
	if result.Summary == "" || result.WhyWorthExploring == "" {
		return ExplorationResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("exploration response missing required fields"))
	}
	if len([]rune(result.Summary)) > 240 || len([]rune(result.WhyWorthExploring)) > 360 {
		return ExplorationResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("exploration response exceeds length limit"))
	}
	return result, nil
}

func (p *DeepSeekProvider) GenerateExploration(ctx context.Context, req ExplorationRequest) (ExplorationResult, ProviderMeta, error) {
	meta := ProviderMeta{Provider: "deepseek", Model: p.model, PromptVersion: ExplorationPromptVersion}
	if p.apiKey == "" {
		return ExplorationResult{}, meta, newProviderError(ErrNotConfigured, nil)
	}
	requestContext, cancel := context.WithTimeout(ctx, p.evaluationTimeout())
	defer cancel()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		meta.AttemptCount = attempt
		content, responseMeta, retry, err := p.requestJSON(requestContext, BuildExplorationSystemPrompt(), BuildExplorationUserPrompt(req))
		meta.RawResponse, meta.InputTokens, meta.OutputTokens = responseMeta.RawResponse, responseMeta.InputTokens, responseMeta.OutputTokens
		if err == nil {
			result, validationErr := ParseAndValidateExploration(content)
			if validationErr == nil {
				return result, meta, nil
			}
			err, retry = validationErr, true
		}
		lastErr = err
		if !retry || attempt == maxAttempts {
			break
		}
	}
	if requestContext.Err() != nil {
		if requestContext.Err() == context.DeadlineExceeded {
			return ExplorationResult{}, meta, newProviderError(ErrTimeout, nil)
		}
		return ExplorationResult{}, meta, requestContext.Err()
	}
	return ExplorationResult{}, meta, lastErr
}
