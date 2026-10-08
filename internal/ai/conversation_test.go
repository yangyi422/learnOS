package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConversationProviderSingleCallAndContext(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.MaxTokens != 1800 || !strings.Contains(body.Messages[1].Content, "先前的问题") || !strings.Contains(body.Messages[1].Content, "换个例子") {
			t.Error("missing context or reply budget")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": `{"reply":"可以用日常例子说明。","intent":"confusion","understood":false,"evidence_quote":"","evidence_explanation":"","confidence":0}`}}}})
	}))
	defer server.Close()
	provider := NewDeepSeekProvider(server.Client(), server.URL, "test", "test", time.Second)
	result, _, err := provider.Converse(context.Background(), ConversationRequest{Message: "换个例子", History: []ConversationMessage{{User: "先前的问题", Assistant: "先前的解释"}}})
	if err != nil || result.Intent != "confusion" || calls != 1 {
		t.Fatal(result, err, calls)
	}
}
func TestConversationDropsInventedEvidenceAndAcceptsReply(t *testing.T) {
	result, err := ValidateConversation(ConversationResult{Reply: "这里是解释。", Intent: "answer", Understood: true, EvidenceQuote: "这是一段从未出现在用户输入中的完整解释与理解证据", EvidenceExplanation: "理解", Confidence: .9}, "我懂了")
	if err != nil || result.Understood || result.EvidenceQuote != "" {
		t.Fatal(result, err)
	}
}
