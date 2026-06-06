package llms

import (
	"testing"

	"github.com/openai/openai-go/v3/shared"
)

func TestDefaultReasoningEffort(t *testing.T) {
	config := LlmConfig{
		BaseUrl: "https://test.example.com",
		APIKey:  "test-key",
		Model:   "test-model",
	}
	if config.ReasoningEffort != "" {
		t.Errorf("expected empty ReasoningEffort by default, got '%s'", config.ReasoningEffort)
	}
}

func TestSetReasoningEffort(t *testing.T) {
	config := LlmConfig{
		BaseUrl:         "https://test.example.com",
		APIKey:          "test-key",
		Model:           "test-model",
		ReasoningEffort: shared.ReasoningEffortHigh,
	}
	if config.ReasoningEffort != shared.ReasoningEffortHigh {
		t.Errorf("expected ReasoningEffort 'high', got '%s'", config.ReasoningEffort)
	}
}