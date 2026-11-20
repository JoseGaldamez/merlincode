package pricing

import (
	"strings"
	"time"

	"merlincode/internal/domain"
)

// Rates are USD per million tokens. Only the published text inference rates for
// models in Merlin's catalog are included. Account credits and discounts are not observable.
type rates struct {
	input, output, cacheRead, cacheWriteShort, cacheWriteLong float64
}

func Estimate(provider, model string, result *domain.ChatCompletionResult, requestedAt time.Time) *int64 {
	usage := result.Usage
	if usage == nil || result.TokensPrompt < 0 || result.TokensPrompt > 2_000_000 ||
		result.TokensCompletion < 0 || result.TokensCompletion > 2_000_000 ||
		usage.CacheRead < 0 || usage.CacheRead > result.TokensPrompt ||
		usage.CacheWriteShort < 0 || usage.CacheWriteShort > result.TokensPrompt ||
		usage.CacheWriteLong < 0 || usage.CacheWriteLong > result.TokensPrompt {
		return nil
	}
	uncached := result.TokensPrompt - usage.CacheRead - usage.CacheWriteShort - usage.CacheWriteLong
	if uncached < 0 {
		return nil
	}

	if !usage.OccurredAt.IsZero() {
		requestedAt = usage.OccurredAt
	}
	rate, multiplier := modelRates(provider, strings.ToLower(model), usage.ServiceTier, result.TokensPrompt, requestedAt.UTC())
	if multiplier == 0 {
		return nil
	}
	total := float64(uncached)*rate.input + float64(result.TokensCompletion)*rate.output +
		float64(usage.CacheRead)*rate.cacheRead + float64(usage.CacheWriteShort)*rate.cacheWriteShort +
		float64(usage.CacheWriteLong)*rate.cacheWriteLong
	nanos := int64(total*multiplier*1000 + 0.5)
	return &nanos
}

func modelRates(provider, model, tier string, input int64, at time.Time) (rates, float64) {
	var rate rates
	multiplier := 1.0
	switch provider {
	case "openai":
		switch model {
		case "gpt-6.2-sol":
			rate = rates{2, 10, 0.10, 2.50, 5}
		case "gpt-6-luna":
			rate = rates{0.10, 0.50, 0.01, 0.125, 0.25}
		}
		if input > 272_000 {
			rate = rates{rate.input * 2, rate.output * 1.5, rate.cacheRead * 2, rate.cacheWriteShort * 2, rate.cacheWriteLong * 2}
		}
		switch tier {
		case "", "auto", "default", "standard":
		case "priority", "fast":
			multiplier = 2
		case "flex":
			multiplier = 0.5
		case "ultrafast":
			multiplier = 3
			if !at.Before(time.Date(2027, time.March, 1, 8, 0, 0, 0, time.UTC)) {
				multiplier = 6
			}
		default:
			return rates{}, 0
		}
	case "anthropic":
		if tier != "" && tier != "standard" && tier != "default" {
			return rates{}, 0
		}
		switch model {
		case "claude-sonnet-5-5":
			rate = rates{2, 10, 0.10, 2.50, 4}
		case "claude-opus-5-5":
			rate = rates{4, 20, 0.20, 5, 8}
		case "claude-fable-5-2":
			rate = rates{8, 40, 0.20, 10, 16}
		case "claude-haiku-5-5":
			rate = rates{0.10, 0.50, 0.01, 0.125, 0.20}
			if input > 100_000 {
				rate = rates{0.50, 2.50, 0.05, 0.625, 1}
			}
		}
	case "google":
		switch model {
		case "gemini-4-hydrogen":
			rate = rates{input: 1.5, output: 7.5, cacheRead: 0.075}
		case "gemini-4-neon":
			rate = rates{input: 3, output: 15, cacheRead: 0.15}
		case "gemini-4-carbon":
			rate = rates{input: 2, output: 10, cacheRead: 0.10}
			if !at.Before(time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)) {
				multiplier = 2
			}
		case "gemini-3.5-flash-lite":
			rate = rates{input: 0.30, output: 2.50, cacheRead: 0.03}
		}
	case "deepseek":
		switch model {
		case "deepseek-flash":
			rate = rates{input: 0.30, output: 1.20, cacheRead: 0.006}
		case "deepseek-v4-pro":
			rate = rates{input: 1.32, output: 3.96, cacheRead: 0.044}
		}
		// The regular UTC schedule is observable; the provider's holiday overrides are not.
		hour := at.Hour()
		weekday := at.Weekday() >= time.Monday && at.Weekday() <= time.Friday
		if !weekday || !(hour >= 1 && hour < 4 || hour >= 6 && hour < 10) {
			multiplier = 0.5
		}
	}
	if rate.input == 0 {
		return rates{}, 0
	}
	return rate, multiplier
}
