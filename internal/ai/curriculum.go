package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type CurriculumMissingLesson struct {
	Key                   string `json:"key"`
	UnitKey               string `json:"unit_key"`
	Title                 string `json:"title"`
	Summary               string `json:"summary"`
	Importance            string `json:"importance"`
	ContentRole           string `json:"content_role"`
	DepthLevel            int    `json:"depth_level"`
	AssessmentTargetLevel string `json:"assessment_target_level"`
}

type CurriculumDraftRequest struct {
	CourseName        string
	Scope             string
	Limit             int
	Missing           []CurriculumMissingLesson
	RuleChangeSetJSON string
}

type CurriculumDraftResult struct {
	Summary       string `json:"summary"`
	ChangeSetJSON string `json:"-"`
}

type curriculumDraftPayload struct {
	Summary           string          `json:"summary"`
	NewUnits          json.RawMessage `json:"new_units"`
	NewLessons        json.RawMessage `json:"new_lessons"`
	NewRelations      json.RawMessage `json:"new_relations"`
	BlueprintMappings json.RawMessage `json:"blueprint_mappings"`
}

func BuildCurriculumDraftSystemPrompt() string {
	return `你是 LearnOS 的课程结构草案编辑器。你的任务是针对课程蓝图中缺失的核心知识，生成最小、可审核的课程扩充草案。只输出 JSON，不删除或修改已有课程节点，不处理用户认知状态，不生成来源、引用或搜索结果。所有新 Lesson 必须来自给定的 missing blueprint lessons。`
}

func BuildCurriculumDraftUserPrompt(req CurriculumDraftRequest) string {
	missing, _ := json.Marshal(req.Missing)
	return fmt.Sprintf("课程：%s\n范围：%s\n最多新增：%d\n缺失蓝图节点：%s\n规则候选 ChangeSet：%s\n请只返回 JSON：包含 summary、new_units、new_lessons、new_relations、blueprint_mappings。", req.CourseName, req.Scope, req.Limit, string(missing), req.RuleChangeSetJSON)
}

func ParseAndValidateCurriculumDraft(content string) (CurriculumDraftResult, error) {
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	var payload curriculumDraftPayload
	if err := decoder.Decode(&payload); err != nil {
		return CurriculumDraftResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("decode curriculum draft JSON"))
	}
	if strings.TrimSpace(payload.Summary) == "" || len([]rune(payload.Summary)) > 2000 {
		return CurriculumDraftResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("invalid curriculum draft summary"))
	}
	if payload.NewUnits == nil {
		payload.NewUnits = json.RawMessage("[]")
	}
	if payload.NewLessons == nil {
		payload.NewLessons = json.RawMessage("[]")
	}
	if payload.NewRelations == nil {
		payload.NewRelations = json.RawMessage("[]")
	}
	if payload.BlueprintMappings == nil {
		payload.BlueprintMappings = json.RawMessage("[]")
	}
	raw, err := json.Marshal(map[string]json.RawMessage{
		"new_units": payload.NewUnits, "new_lessons": payload.NewLessons,
		"new_relations": payload.NewRelations, "blueprint_mappings": payload.BlueprintMappings,
	})
	if err != nil {
		return CurriculumDraftResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("encode curriculum change set"))
	}
	return CurriculumDraftResult{Summary: strings.TrimSpace(payload.Summary), ChangeSetJSON: string(raw)}, nil
}

func (p *MockProvider) GenerateCurriculumDraft(_ context.Context, req CurriculumDraftRequest) (CurriculumDraftResult, ProviderMeta, error) {
	return CurriculumDraftResult{
		Summary:       fmt.Sprintf("规则筛选出 %d 个缺失的课程知识节点，生成可人工审核的扩充草案。", len(req.Missing)),
		ChangeSetJSON: req.RuleChangeSetJSON,
	}, ProviderMeta{Provider: "mock", Model: "mock", PromptVersion: CurriculumDraftPromptVersion, AttemptCount: 1}, nil
}

func (p *DeepSeekProvider) GenerateCurriculumDraft(ctx context.Context, req CurriculumDraftRequest) (CurriculumDraftResult, ProviderMeta, error) {
	meta := ProviderMeta{Provider: "deepseek", Model: p.model, PromptVersion: CurriculumDraftPromptVersion}
	if p.apiKey == "" {
		return CurriculumDraftResult{}, meta, newProviderError(ErrNotConfigured, nil)
	}
	requestContext, cancel := context.WithTimeout(ctx, p.curriculumDraftTimeoutDuration())
	defer cancel()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		meta.AttemptCount = attempt
		content, responseMeta, retry, err := p.requestJSON(requestContext, BuildCurriculumDraftSystemPrompt(), BuildCurriculumDraftUserPrompt(req))
		meta.RawResponse, meta.InputTokens, meta.OutputTokens = responseMeta.RawResponse, responseMeta.InputTokens, responseMeta.OutputTokens
		if err == nil {
			result, validationErr := ParseAndValidateCurriculumDraft(content)
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
	if requestContext.Err() == context.DeadlineExceeded {
		return CurriculumDraftResult{}, meta, newProviderError(ErrTimeout, nil)
	}
	if requestContext.Err() != nil {
		return CurriculumDraftResult{}, meta, requestContext.Err()
	}
	return CurriculumDraftResult{}, meta, lastErr
}

func (p *DeepSeekProvider) curriculumDraftTimeoutDuration() time.Duration {
	if p.curriculumDraftTimeout > 0 {
		return p.curriculumDraftTimeout
	}
	return 60 * time.Second
}
