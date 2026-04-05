package api

// ModelPricing holds per-million-token pricing.
type ModelPricing struct {
	InputPerMillion       float64 // $/M input tokens
	OutputPerMillion      float64 // $/M output tokens
	CacheReadPerMillion   float64 // $/M cache read tokens
	CacheWritePerMillion  float64 // $/M cache write tokens
}

// TokenUsage tracks token consumption for a request.
type TokenUsage struct {
	InputTokens              int `json:"inputTokens"`
	OutputTokens             int `json:"outputTokens"`
	CacheCreationInputTokens int `json:"cacheCreationInputTokens"`
	CacheReadInputTokens     int `json:"cacheReadInputTokens"`
}

// PricingTable maps model IDs to pricing.
// Updated as of 2026-04.
var PricingTable = map[string]ModelPricing{
	// Anthropic
	"claude-opus-4-6-20250205":    {InputPerMillion: 15, OutputPerMillion: 75, CacheReadPerMillion: 1.5, CacheWritePerMillion: 18.75},
	"claude-sonnet-4-6-20250217":  {InputPerMillion: 3, OutputPerMillion: 15, CacheReadPerMillion: 0.3, CacheWritePerMillion: 3.75},
	"claude-haiku-4-5-20251001":   {InputPerMillion: 0.8, OutputPerMillion: 4, CacheReadPerMillion: 0.08, CacheWritePerMillion: 1},
	"claude-opus-4-20250514":      {InputPerMillion: 15, OutputPerMillion: 75, CacheReadPerMillion: 1.5, CacheWritePerMillion: 18.75},
	"claude-sonnet-4-20250514":    {InputPerMillion: 3, OutputPerMillion: 15, CacheReadPerMillion: 0.3, CacheWritePerMillion: 3.75},

	// OpenAI
	"gpt-5.4":          {InputPerMillion: 2.5, OutputPerMillion: 10},
	"gpt-5.4-mini":     {InputPerMillion: 0.4, OutputPerMillion: 1.6},
	"gpt-4.1":          {InputPerMillion: 2, OutputPerMillion: 8},
	"gpt-4o":           {InputPerMillion: 2.5, OutputPerMillion: 10},
	"gpt-4o-mini":      {InputPerMillion: 0.15, OutputPerMillion: 0.6},
	"o3":               {InputPerMillion: 10, OutputPerMillion: 40},
	"o4-mini":          {InputPerMillion: 1.1, OutputPerMillion: 4.4},

	// Gemini
	"gemini-3-pro-preview":   {InputPerMillion: 1.25, OutputPerMillion: 5},
	"gemini-3-flash-preview": {InputPerMillion: 0.075, OutputPerMillion: 0.3},
	"gemini-2.5-pro":         {InputPerMillion: 1.25, OutputPerMillion: 5},
	"gemini-2.5-flash":       {InputPerMillion: 0.075, OutputPerMillion: 0.3},

	// Groq (pass-through pricing)
	"llama-3.3-70b-versatile": {InputPerMillion: 0.59, OutputPerMillion: 0.79},
	"qwen/qwen3-32b":         {InputPerMillion: 0.29, OutputPerMillion: 0.39},

	// Ollama (free, local)
	"qwen3:8b":  {InputPerMillion: 0, OutputPerMillion: 0},
	"qwen3:14b": {InputPerMillion: 0, OutputPerMillion: 0},
	"qwen3:32b": {InputPerMillion: 0, OutputPerMillion: 0},
	"qwen3:72b": {InputPerMillion: 0, OutputPerMillion: 0},
}

// CalculateCost computes the USD cost for a request.
func CalculateCost(model string, usage TokenUsage) float64 {
	pricing, ok := PricingTable[model]
	if !ok {
		// Fallback: rough estimate
		return float64(usage.InputTokens+usage.OutputTokens) / 1_000_000 * 5
	}

	cost := 0.0
	cost += float64(usage.InputTokens) / 1_000_000 * pricing.InputPerMillion
	cost += float64(usage.OutputTokens) / 1_000_000 * pricing.OutputPerMillion
	cost += float64(usage.CacheReadInputTokens) / 1_000_000 * pricing.CacheReadPerMillion
	cost += float64(usage.CacheCreationInputTokens) / 1_000_000 * pricing.CacheWritePerMillion

	return cost
}

// EstimateInputTokens roughly estimates token count from text length.
func EstimateInputTokens(textLength int) int {
	return textLength / 4 // ~4 chars per token average
}
