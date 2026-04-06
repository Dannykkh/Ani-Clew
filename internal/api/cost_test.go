package api

import (
	"math"
	"testing"
)

func TestCalculateCost_Anthropic(t *testing.T) {
	usage := TokenUsage{
		InputTokens:  1000,
		OutputTokens: 500,
	}

	cost := CalculateCost("claude-sonnet-4-6-20250217", usage)
	// Sonnet: $3/M input, $15/M output
	expected := (1000.0/1_000_000)*3 + (500.0/1_000_000)*15
	if math.Abs(cost-expected) > 0.0001 {
		t.Errorf("Expected %.6f, got %.6f", expected, cost)
	}
}

func TestCalculateCost_WithCache(t *testing.T) {
	usage := TokenUsage{
		InputTokens:              500,
		OutputTokens:             200,
		CacheReadInputTokens:     1000,
		CacheCreationInputTokens: 300,
	}

	cost := CalculateCost("claude-opus-4-6-20250205", usage)
	// Opus: $15/M input, $75/M output, $1.5/M cache_read, $18.75/M cache_write
	expected := (500.0/1e6)*15 + (200.0/1e6)*75 + (1000.0/1e6)*1.5 + (300.0/1e6)*18.75
	if math.Abs(cost-expected) > 0.0001 {
		t.Errorf("Expected %.6f, got %.6f", expected, cost)
	}
}

func TestCalculateCost_Ollama(t *testing.T) {
	usage := TokenUsage{InputTokens: 10000, OutputTokens: 5000}
	cost := CalculateCost("qwen3:14b", usage)
	if cost != 0 {
		t.Errorf("Ollama should be free, got %.6f", cost)
	}
}

func TestCalculateCost_Unknown(t *testing.T) {
	usage := TokenUsage{InputTokens: 1000, OutputTokens: 500}
	cost := CalculateCost("unknown-model-xyz", usage)
	// Fallback: (input+output) / 1M * 5
	expected := (1500.0 / 1_000_000) * 5
	if math.Abs(cost-expected) > 0.0001 {
		t.Errorf("Unknown model fallback expected %.6f, got %.6f", expected, cost)
	}
}
