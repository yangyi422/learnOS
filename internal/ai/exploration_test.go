package ai

import (
	"context"
	"testing"
)

func TestParseAndValidateExploration(t *testing.T) {
	result, err := ParseAndValidateExploration(`{"summary":"从饮水信号连接到因果判断。","why_worth_exploring":"它能帮助你比较单一信号在不同领域中的局限。"}`)
	if err != nil {
		t.Fatalf("parse exploration response: %v", err)
	}
	if result.Summary == "" || result.WhyWorthExploring == "" {
		t.Fatalf("expected validated exploration text: %+v", result)
	}
	if _, err := ParseAndValidateExploration(`{"summary":""}`); err == nil {
		t.Fatal("missing exploration text should fail validation")
	}
}

func TestMockProviderGeneratesExplorationText(t *testing.T) {
	result, meta, err := NewMockProvider().GenerateExploration(context.Background(), ExplorationRequest{SourceLesson: "口渴信号", TargetLesson: "相关不等于因果", Why: "帮助建立通用判断框架。"})
	if err != nil {
		t.Fatalf("mock exploration generation: %v", err)
	}
	if result.Summary == "" || result.WhyWorthExploring == "" || meta.PromptVersion != ExplorationPromptVersion {
		t.Fatalf("unexpected mock exploration result: %+v %+v", result, meta)
	}
}
