package pricing

import (
	"testing"
	"time"

	"merlincode/internal/domain"
)

func TestEstimate(t *testing.T) {
	peak := time.Date(2026, time.November, 20, 2, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name, provider, model string
		usage                 domain.ModelUsage
		expected              int64
	}{
		{"OpenAI cache writes", "openai", "gpt-6.2-sol",
			domain.ModelUsage{CacheRead: 100, CacheWriteShort: 200, CacheWriteLong: 300}, 3_810_000},
		{"Anthropic cache writes", "anthropic", "claude-sonnet-5-5",
			domain.ModelUsage{CacheRead: 100, CacheWriteShort: 200, CacheWriteLong: 300}, 3_510_000},
		{"Gemini paid rate", "google", "gemini-4-hydrogen",
			domain.ModelUsage{CacheRead: 100}, 2_107_500},
		{"DeepSeek peak", "deepseek", "deepseek-flash",
			domain.ModelUsage{CacheRead: 100}, 390_600},
		{"DeepSeek off-peak by provider time", "deepseek", "deepseek-flash",
			domain.ModelUsage{CacheRead: 100, OccurredAt: peak.Add(10 * time.Hour)}, 195_300},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.ChatCompletionResult{TokensPrompt: 1000, TokensCompletion: 100, Usage: &tt.usage}
			if got := Estimate(tt.provider, tt.model, &result, peak); got == nil || *got != tt.expected {
				t.Fatalf("expected %d nanos USD, got %v", tt.expected, got)
			}
		})
	}
	if Estimate("openai", "gpt-6.2-sol", &domain.ChatCompletionResult{}, peak) != nil {
		t.Fatal("missing provider usage must not be valued at zero")
	}
}
