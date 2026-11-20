package google

import "merlincode/internal/domain"

type geminiUsage struct {
	PromptTokens   int64 `json:"promptTokenCount"`
	ToolTokens     int64 `json:"toolUsePromptTokenCount"`
	CachedTokens   int64 `json:"cachedContentTokenCount"`
	ResponseTokens int64 `json:"candidatesTokenCount"`
	ThoughtTokens  int64 `json:"thoughtsTokenCount"`
}

func (u *geminiUsage) apply(result *domain.ChatCompletionResult) {
	if u != nil {
		result.TokensPrompt = u.PromptTokens + u.ToolTokens
		result.TokensCompletion = u.ResponseTokens + u.ThoughtTokens
		result.Usage = &domain.ModelUsage{CacheRead: u.CachedTokens}
	}
}
