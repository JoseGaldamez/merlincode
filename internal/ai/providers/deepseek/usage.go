package deepseek

import (
	"time"

	"merlincode/internal/domain"
)

type deepSeekUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	CachedTokens     int64 `json:"prompt_cache_hit_tokens"`
}

func (u *deepSeekUsage) apply(result *domain.ChatCompletionResult, created int64) {
	if u != nil {
		result.TokensPrompt = u.PromptTokens
		result.TokensCompletion = u.CompletionTokens
		result.Usage = &domain.ModelUsage{CacheRead: u.CachedTokens}
		if created > 0 {
			result.Usage.OccurredAt = time.Unix(created, 0)
		}
	}
}
