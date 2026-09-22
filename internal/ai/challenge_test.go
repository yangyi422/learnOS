package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func validTransferChallengeEvaluation() ChallengeEvaluationResult {
	return ChallengeEvaluationResult{
		Result:            "mostly_correct",
		Feedback:          "反馈",
		Explanation:       "解释",
		DemonstratedLevel: "transfer",
		CognitiveEvidence: []EvaluationEvidence{{EvidenceType: "transfer", CognitiveLevel: "transfer", Polarity: "support", Description: "在陌生场景中使用了核心判断"}},
	}
}

func TestChallengeValidationAndPassRules(t *testing.T) {
	result, err := NormalizeAndValidateChallengeEvaluation(validTransferChallengeEvaluation(), "transfer", "transfer", 0)
	if err != nil {
		t.Fatalf("validate transfer evaluation: %v", err)
	}
	if !CalculateChallengePassed(result, "transfer", 0) {
		t.Fatal("valid transfer evidence should pass")
	}

	result.CognitiveEvidence[0].Polarity = "contradict"
	if CalculateChallengePassed(result, "transfer", 0) {
		t.Fatal("contradictory transfer evidence must fail")
	}

	result = validTransferChallengeEvaluation()
	if _, err := NormalizeAndValidateChallengeEvaluation(result, "transfer", "apply", 0); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected transfer level and evidence validation, got %v", err)
	}
}

func TestMisconceptionRecheckRequiresMatchingCorrection(t *testing.T) {
	result := ChallengeEvaluationResult{
		Result:            "correct",
		Feedback:          "反馈",
		Explanation:       "解释",
		DemonstratedLevel: "understand",
		CognitiveEvidence: []EvaluationEvidence{{EvidenceType: "concept_explanation", CognitiveLevel: "understand", Polarity: "support", Description: "解释了目标误区的边界"}},
		MisconceptionValidation: &MisconceptionValidation{
			TargetMisconceptionID: 7,
			Status:                "corrected",
			Evidence:              "新场景中明确否定绝对化判断",
		},
	}
	result, err := NormalizeAndValidateChallengeEvaluation(result, "misconception_recheck", "understand", 7)
	if err != nil || !CalculateChallengePassed(result, "misconception_recheck", 7) {
		t.Fatalf("valid recheck should pass: %v", err)
	}
	result.MisconceptionValidation.Status = "persists"
	if CalculateChallengePassed(result, "misconception_recheck", 7) {
		t.Fatal("persisting misconception must not pass")
	}
}

func TestReasoningPatternTaxonomyIsStrict(t *testing.T) {
	valid := validEvaluationJSON()
	valid = strings.Replace(valid, `"misconceptions":[]`, `"misconceptions":[{"original_understanding":"绝对化判断","correct_understanding":"需要结合场景","boundary_notes":"存在边界","reasoning_patterns":[{"pattern_key":"binary_thinking","explanation":"把复杂判断简化成二选一"}]}]`, 1)
	if _, err := ParseAndValidate(valid); err != nil {
		t.Fatalf("fixed reasoning pattern should be accepted: %v", err)
	}
	unsupported := strings.Replace(valid, "binary_thinking", "made_up_pattern", 1)
	if _, err := ParseAndValidate(unsupported); err == nil || !strings.Contains(ErrorDetail(err), "unsupported reasoning pattern") {
		t.Fatalf("unsupported reasoning pattern must be rejected: %v", err)
	}
}

func challengeEvaluationJSON(evidenceFields string) string {
	return `{"result":"mostly_correct","feedback":"反馈","explanation":"解释","demonstrated_level":"transfer","cognitive_evidence":[{"evidence_type":"transfer",` + evidenceFields + `,"polarity":"support","description":"迁移证据"}],"misconceptions":[],"misconception_validation":null}`
}

func validChallengeGenerationJSON() string {
	return `{"prompt":"挑战问题","scenario_context":"新场景","target_level":"transfer","evaluation_criteria":["标准"],"why_this_is_transfer":"原因","source_concepts":["概念"]}`
}

func TestChallengeEvidenceCognitiveLevelAliasNormalization(t *testing.T) {
	tests := []struct {
		name       string
		fields     string
		wantLevel  string
		shouldFail bool
	}{
		{name: "canonical cognitive_level", fields: `"cognitive_level":"transfer"`, wantLevel: "transfer"},
		{name: "level alias", fields: `"level":"transfer"`, wantLevel: "transfer"},
		{name: "matching fields", fields: `"cognitive_level":"transfer","level":"transfer"`, wantLevel: "transfer"},
		{name: "conflicting fields", fields: `"cognitive_level":"transfer","level":"understand"`, shouldFail: true},
		{name: "unknown field", fields: `"cognitive_level":"transfer","foo":"bar"`, shouldFail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parseChallengeEvaluation(challengeEvaluationJSON(test.fields), "transfer", "transfer", 0)
			if test.shouldFail {
				if !errors.Is(err, ErrInvalidResponse) {
					t.Fatalf("expected invalid response, got result=%+v err=%v", result, err)
				}
				return
			}
			if err != nil || result.CognitiveEvidence[0].CognitiveLevel != test.wantLevel {
				t.Fatalf("expected normalized cognitive level, result=%+v err=%v", result, err)
			}
		})
	}
}

func TestChallengePromptUsesOneStrictEvidenceSchema(t *testing.T) {
	prompt := BuildChallengeEvaluationSystemPrompt()
	for _, required := range []string{`"evidence_type"`, `"cognitive_level"`, `"polarity"`, `"description"`, "字段名必须严格为 cognitive_level", "不要使用 level", "不要使用 mastery_level", "合法 Transfer JSON 示例", "合法 Recheck JSON 示例"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("challenge evaluator prompt missing %q", required)
		}
	}
}

func TestValidNonTransferChallengeEvaluationFailsPassLogicOnly(t *testing.T) {
	content := strings.Replace(challengeEvaluationJSON(`"cognitive_level":"transfer"`), `"result":"mostly_correct"`, `"result":"partially_correct"`, 1)
	content = strings.Replace(content, `"demonstrated_level":"transfer"`, `"demonstrated_level":"understand"`, 1)
	content = strings.Replace(content, `"evidence_type":"transfer","cognitive_level":"transfer"`, `"evidence_type":"concept_explanation","cognitive_level":"understand"`, 1)
	result, err := parseChallengeEvaluation(content, "transfer", "transfer", 0)
	if err != nil {
		t.Fatalf("valid non-transfer evaluation was rejected: %v", err)
	}
	if CalculateChallengePassed(result, "transfer", 0) {
		t.Fatal("partially_correct understand evaluation must not pass transfer")
	}
}

func TestDeepSeekChallengeRetriesEmptyContentOnce(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		if calls.Load() == 1 {
			_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
			return
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":` + strconv.Quote(challengeEvaluationJSON(`"cognitive_level":"transfer"`)) + `}}]}`))
	}))
	defer server.Close()
	provider := NewDeepSeekProvider(server.Client(), server.URL, "test-key", DefaultModel, time.Second)
	result, meta, err := provider.EvaluateChallenge(context.Background(), ChallengeEvaluationRequest{ChallengeType: "transfer", TargetLevel: "transfer"})
	if err != nil || result.Result != "mostly_correct" || calls.Load() != 2 || meta.AttemptCount != 2 {
		t.Fatalf("expected one empty-content retry, result=%+v meta=%+v calls=%d err=%v", result, meta, calls.Load(), err)
	}
}

func TestDeepSeekChallengeReturnsInvalidAfterTwoEmptyResponses(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
	}))
	defer server.Close()
	provider := NewDeepSeekProvider(server.Client(), server.URL, "test-key", DefaultModel, time.Second)
	_, meta, err := provider.EvaluateChallenge(context.Background(), ChallengeEvaluationRequest{ChallengeType: "transfer", TargetLevel: "transfer"})
	if !errors.Is(err, ErrEmptyContent) || calls.Load() != 2 || meta.AttemptCount != 2 {
		t.Fatalf("expected two attempts and empty-content error, meta=%+v calls=%d err=%v", meta, calls.Load(), err)
	}
}

func TestDeepSeekChallengeGenerationRetriesEmptyContentAndSucceeds(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if calls.Add(1) == 1 {
			_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
			return
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":` + strconv.Quote(validChallengeGenerationJSON()) + `}}]}`))
	}))
	defer server.Close()
	provider := NewDeepSeekProvider(server.Client(), server.URL, "test-key", DefaultModel, 45*time.Second, 60*time.Second)
	result, meta, err := provider.GenerateChallenge(context.Background(), ChallengeGenerationRequest{ChallengeType: "transfer"})
	if err != nil || result.TargetLevel != "transfer" || calls.Load() != 2 || meta.AttemptCount != 2 {
		t.Fatalf("expected empty-content retry to succeed, result=%+v meta=%+v calls=%d err=%v", result, meta, calls.Load(), err)
	}
}

func TestDeepSeekChallengeGenerationDeadlineIsTimeoutNotValidation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if calls.Add(1) == 1 {
			_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
			return
		}
		select {
		case <-request.Context().Done():
		case <-time.After(time.Second):
		}
	}))
	defer func() {
		server.CloseClientConnections()
		server.Close()
	}()

	var logs bytes.Buffer
	oldWriter, oldFlags := log.Writer(), log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	}()

	provider := NewDeepSeekProvider(server.Client(), server.URL, "test-key", DefaultModel, 45*time.Second, 250*time.Millisecond)
	_, meta, err := provider.GenerateChallenge(context.Background(), ChallengeGenerationRequest{ChallengeType: "transfer"})
	if !errors.Is(err, ErrTimeout) || meta.AttemptCount != 2 || calls.Load() != 2 {
		t.Fatalf("expected generation timeout after retry, meta=%+v calls=%d err=%v", meta, calls.Load(), err)
	}
	output := logs.String()
	if !strings.Contains(output, "attempt=2") || !strings.Contains(output, "failure_type=timeout") || !strings.Contains(output, `error="context deadline exceeded"`) {
		t.Fatalf("timeout attempt was not classified correctly: %q", output)
	}
	if strings.Contains(output, "attempt=2") && strings.Contains(output, "validation_error=") {
		t.Fatalf("timeout attempt must not be logged as validation error: %q", output)
	}
}

func TestDeepSeekChallengeRequestUsesJSONOutputAndEnoughTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload struct {
			ResponseFormat responseFormat `json:"response_format"`
			MaxTokens      int            `json:"max_tokens"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode challenge request: %v", err)
		}
		if payload.ResponseFormat.Type != "json_object" || payload.MaxTokens != 24000 {
			t.Errorf("unexpected challenge request constraints: %+v", payload)
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
	}))
	defer server.Close()
	provider := NewDeepSeekProvider(server.Client(), server.URL, "test-key", DefaultModel, time.Second)
	_, _, _ = provider.EvaluateChallenge(context.Background(), ChallengeEvaluationRequest{ChallengeType: "transfer", TargetLevel: "transfer"})
}

func TestDeepSeekChallengeGenerationLogsSafeAttemptMetricsAndCompletion(t *testing.T) {
	const sensitivePrompt = "secret challenge context"
	const sensitiveResponse = "secret raw response"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"not-json-secret"}}]}`))
	}))
	defer server.Close()

	var logs bytes.Buffer
	oldWriter, oldFlags := log.Writer(), log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	}()

	provider := NewDeepSeekProvider(server.Client(), server.URL, "test-key", DefaultModel, time.Second)
	_, _, _ = provider.GenerateChallenge(context.Background(), ChallengeGenerationRequest{ChallengeType: "transfer", CoreQuestion: sensitivePrompt})
	output := logs.String()
	if strings.Contains(output, sensitivePrompt) || strings.Contains(output, sensitiveResponse) || strings.Contains(output, "test-key") || strings.Contains(output, "Authorization") {
		t.Fatalf("challenge generation logs leaked sensitive content: %q", output)
	}
	if !strings.Contains(output, "challenge generation: challenge_type=transfer provider=deepseek model=deepseek-v4-flash prompt_version=learnos-transfer-generator-v1 attempt=1") {
		t.Fatalf("missing attempt log: %q", output)
	}
	for _, field := range []string{"provider_latency_ms=", "prompt_chars=", "raw_content_chars=", "validation_status=failed", "validation_error=", "total_latency_ms="} {
		if !strings.Contains(output, field) {
			t.Fatalf("attempt log missing %q: %q", field, output)
		}
	}
	if strings.Contains(output, "challenge generation completed:") {
		t.Fatalf("failed generation must not log completion: %q", output)
	}
}

func TestDeepSeekChallengeGenerationLogsEachRetryAttempt(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if calls.Add(1) == 1 {
			_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"invalid"}}]}`))
			return
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"{\"prompt\":\"挑战问题\",\"scenario_context\":\"新场景\",\"target_level\":\"transfer\",\"evaluation_criteria\":[\"标准\"],\"why_this_is_transfer\":\"原因\",\"source_concepts\":[\"概念\"]}"}}]}`))
	}))
	defer server.Close()

	var logs bytes.Buffer
	oldWriter, oldFlags := log.Writer(), log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	}()

	provider := NewDeepSeekProvider(server.Client(), server.URL, "test-key", DefaultModel, time.Second)
	result, meta, err := provider.GenerateChallenge(context.Background(), ChallengeGenerationRequest{ChallengeType: "transfer"})
	if err != nil || result.TargetLevel != "transfer" || calls.Load() != 2 || meta.AttemptCount != 2 {
		t.Fatalf("expected successful retry, result=%+v meta=%+v calls=%d err=%v", result, meta, calls.Load(), err)
	}
	output := logs.String()
	if strings.Count(output, "challenge generation: challenge_type=") != 2 {
		t.Fatalf("expected one log per generation attempt: %q", output)
	}
	if !strings.Contains(output, "attempt=1") || !strings.Contains(output, "attempt=2") || !strings.Contains(output, "validation_status=failed") || !strings.Contains(output, "validation_status=success") {
		t.Fatalf("missing retry attempt statuses: %q", output)
	}
	if !strings.Contains(output, "challenge generation completed: challenge_type=transfer attempt_count=2 total_latency_ms=") {
		t.Fatalf("missing completion log: %q", output)
	}
}
