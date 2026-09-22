package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestMockDomainInitializationStagesSatisfyProgressiveBounds(t *testing.T) {
	provider := NewMockProvider()
	skeleton, _, err := provider.GenerateDomainSkeleton(context.Background(), DomainSkeletonRequest{DomainName: "摄影", LearningGoal: "建立基础理解", TargetDepth: "foundation"})
	if err != nil {
		t.Fatalf("generate skeleton: %v", err)
	}
	if _, err := ParseAndValidateDomainSkeleton(mustJSON(t, skeleton)); err != nil {
		t.Fatalf("validate skeleton: %v", err)
	}
	starter, _, err := provider.ExpandDomainStarter(context.Background(), DomainStarterRequest{Skeleton: skeleton, SelectedStarterKeys: skeleton.RecommendedStarterKeys})
	if err != nil {
		t.Fatalf("expand starter: %v", err)
	}
	if _, err := ParseAndValidateDomainStarter(mustJSON(t, starter), skeleton, skeleton.RecommendedStarterKeys); err != nil {
		t.Fatalf("validate starter: %v", err)
	}
	world, _, err := provider.GenerateInitialWorld(context.Background(), InitialWorldRequest{Starter: starter})
	if err != nil {
		t.Fatalf("generate initial world: %v", err)
	}
	if _, err := ParseAndValidateInitialWorld(mustJSON(t, world), starter); err != nil {
		t.Fatalf("validate initial world: %v", err)
	}
}

func TestDomainInitializationParserRejectsTrailingJSONAndInvalidCounts(t *testing.T) {
	if _, err := ParseAndValidateDomainSkeleton(`{"course":{"name":"x"},"blueprint":{"name":"x","domain":"x","units":[]},"recommended_starter_unit_keys":[]} {"unexpected":true}`); err == nil {
		t.Fatal("expected trailing JSON to be rejected")
	}
	if _, err := ParseAndValidateInitialWorld(`{"initial_lessons":[],"relations":[],"recommended_first_lesson_key":"missing"}`, DomainStarterResult{}); err == nil {
		t.Fatal("expected invalid initial world count to be rejected")
	}
}

func TestDomainInitializationParserAcceptsJSONCodeFence(t *testing.T) {
	provider := NewMockProvider()
	skeleton, _, err := provider.GenerateDomainSkeleton(context.Background(), DomainSkeletonRequest{DomainName: "摄影", LearningGoal: "建立基础理解", TargetDepth: "foundation"})
	if err != nil {
		t.Fatalf("generate skeleton: %v", err)
	}
	raw := mustJSON(t, skeleton)
	if _, err := ParseAndValidateDomainSkeleton("```json\n" + raw + "\n```"); err != nil {
		t.Fatalf("expected fenced JSON to be accepted: %v", err)
	}
}

func TestDeepSeekDomainRequestKeepsThinkingDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload struct {
			Thinking  map[string]string `json:"thinking"`
			MaxTokens int               `json:"max_tokens"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload.Thinking != nil {
			t.Errorf("domain initialization should keep the provider thinking default: %+v", payload.Thinking)
		}
		if payload.MaxTokens != 50000 {
			t.Errorf("domain initialization max_tokens = %d, want 50000", payload.MaxTokens)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":` + strconv.Quote(`{}`) + `}}]}`))
	}))
	defer server.Close()

	provider := NewDeepSeekProvider(server.Client(), server.URL, "test-key", DefaultModel, 0)
	if _, _, _, err := provider.domainRequestOnce(context.Background(), "system", "user"); err != nil {
		t.Fatalf("domain request: %v", err)
	}
}

func mustJSON(t *testing.T, value interface{}) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test value: %v", err)
	}
	return string(raw)
}
