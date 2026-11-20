package anthropic

import "merlincode/internal/domain"

type anthropicUsage struct {
	InputTokens   *int64 `json:"input_tokens"`
	OutputTokens  *int64 `json:"output_tokens"`
	CacheRead     *int64 `json:"cache_read_input_tokens"`
	CacheWrites   *int64 `json:"cache_creation_input_tokens"`
	CacheCreation *struct {
		Short int64 `json:"ephemeral_5m_input_tokens"`
		Long  int64 `json:"ephemeral_1h_input_tokens"`
	} `json:"cache_creation"`
	ServiceTier string `json:"service_tier"`
}

// Anthropic transmits cumulative usage across message_start and message_delta.
func (u *anthropicUsage) merge(update *anthropicUsage) {
	if update == nil {
		return
	}
	if update.InputTokens != nil {
		u.InputTokens = update.InputTokens
	}
	if update.OutputTokens != nil {
		u.OutputTokens = update.OutputTokens
	}
	if update.CacheRead != nil {
		u.CacheRead = update.CacheRead
	}
	if update.CacheWrites != nil {
		u.CacheWrites = update.CacheWrites
	}
	if update.CacheCreation != nil {
		u.CacheCreation = update.CacheCreation
	}
	if update.ServiceTier != "" {
		u.ServiceTier = update.ServiceTier
	}
}

func (u anthropicUsage) apply(result *domain.ChatCompletionResult) {
	usage := domain.ModelUsage{ServiceTier: u.ServiceTier}
	if u.CacheRead != nil {
		usage.CacheRead = *u.CacheRead
	}
	if u.CacheCreation != nil {
		usage.CacheWriteShort = u.CacheCreation.Short
		usage.CacheWriteLong = u.CacheCreation.Long
	}
	if u.CacheWrites != nil {
		usage.CacheWriteShort = max(usage.CacheWriteShort, *u.CacheWrites-usage.CacheWriteLong)
	}
	result.TokensPrompt = usage.CacheRead + usage.CacheWriteShort + usage.CacheWriteLong
	if u.InputTokens != nil {
		result.TokensPrompt += *u.InputTokens
	}
	if u.OutputTokens != nil {
		result.TokensCompletion = *u.OutputTokens
	}
	if u.InputTokens != nil && u.OutputTokens != nil {
		result.Usage = &usage
	}
}
