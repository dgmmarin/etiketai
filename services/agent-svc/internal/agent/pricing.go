package agent

// ModelPricing holds per-token prices in USD for a given model.
type ModelPricing struct {
	InputPer1K  float64 // USD per 1000 input tokens
	OutputPer1K float64 // USD per 1000 output tokens
}

// pricing maps provider+model to USD per 1000 tokens.
// Sources: Anthropic pricing page (2025-Q1), OpenAI pricing page (2025-Q1).
var pricing = map[string]ModelPricing{
	// Anthropic Claude
	"claude:claude-opus-4-6":         {InputPer1K: 0.015, OutputPer1K: 0.075},
	"claude:claude-sonnet-4-6":       {InputPer1K: 0.003, OutputPer1K: 0.015},
	"claude:claude-haiku-4-5":        {InputPer1K: 0.00025, OutputPer1K: 0.00125},
	"claude:claude-3-5-sonnet-latest": {InputPer1K: 0.003, OutputPer1K: 0.015},
	"claude:claude-3-5-haiku-latest":  {InputPer1K: 0.00025, OutputPer1K: 0.00125},
	"claude:claude-3-opus-latest":     {InputPer1K: 0.015, OutputPer1K: 0.075},
	// OpenAI GPT
	"openai:gpt-4o":       {InputPer1K: 0.005, OutputPer1K: 0.015},
	"openai:gpt-4o-mini":  {InputPer1K: 0.00015, OutputPer1K: 0.0006},
	"openai:gpt-4-turbo":  {InputPer1K: 0.01, OutputPer1K: 0.03},
	// Ollama (on-premise — zero cost)
	"ollama:llava:7b":   {InputPer1K: 0, OutputPer1K: 0},
	"ollama:llava:13b":  {InputPer1K: 0, OutputPer1K: 0},
	"ollama:llava:34b":  {InputPer1K: 0, OutputPer1K: 0},
	"ollama:llama3.2":   {InputPer1K: 0, OutputPer1K: 0},
	"ollama:mistral":    {InputPer1K: 0, OutputPer1K: 0},
}

// ComputeCost returns the estimated USD cost for an agent call.
// tokensIn and tokensOut may both be 0 if the provider didn't return token counts
// (in that case totalTokens is split 60/40 as an approximation).
func ComputeCost(provider, model string, tokensIn, tokensOut, totalTokens int) float64 {
	key := provider + ":" + model
	p, ok := pricing[key]
	if !ok {
		// Fallback: try provider-only defaults
		switch provider {
		case "claude":
			p = ModelPricing{InputPer1K: 0.003, OutputPer1K: 0.015}
		case "openai":
			p = ModelPricing{InputPer1K: 0.005, OutputPer1K: 0.015}
		default:
			return 0
		}
	}

	if tokensIn == 0 && tokensOut == 0 && totalTokens > 0 {
		tokensIn = int(float64(totalTokens) * 0.6)
		tokensOut = totalTokens - tokensIn
	}

	return float64(tokensIn)/1000*p.InputPer1K + float64(tokensOut)/1000*p.OutputPer1K
}
