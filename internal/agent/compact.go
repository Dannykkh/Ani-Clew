package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/aniclew/aniclew/internal/types"
)

const (
	// Reserve 20K tokens for output
	maxOutputReserve = 20000
	// Auto-compact when this close to limit
	compactMargin = 13000
	// Circuit breaker: stop after N consecutive failures
	maxCompactFailures = 3
)

// CompactConfig holds compaction settings.
type CompactConfig struct {
	ContextWindow   int // model's context window
	CompactFailures int // consecutive failure count
}

// ShouldCompact checks if auto-compaction should trigger.
func ShouldCompact(cfg CompactConfig, currentTokens int) bool {
	if cfg.CompactFailures >= maxCompactFailures {
		return false // circuit breaker
	}
	effective := cfg.ContextWindow - maxOutputReserve
	threshold := effective - compactMargin
	return currentTokens > threshold
}

// CompactMessages summarizes older messages to reduce token count.
// Uses the LLM itself to create a summary.
func CompactMessages(ctx context.Context, provider types.Provider, model string, messages []types.Message) ([]types.Message, error) {
	if len(messages) <= 2 {
		return messages, nil // nothing to compact
	}

	log.Printf("[Compact] Compacting %d messages", len(messages))

	// Keep the last 4 messages (recent context)
	keepCount := 4
	if keepCount > len(messages) {
		keepCount = len(messages)
	}
	oldMessages := messages[:len(messages)-keepCount]
	recentMessages := messages[len(messages)-keepCount:]

	// Build summary of old messages
	var summaryParts []string
	for _, msg := range oldMessages {
		var content string
		json.Unmarshal(msg.Content, &content)
		if content == "" {
			// Try array format
			var blocks []struct{ Type, Text string }
			json.Unmarshal(msg.Content, &blocks)
			for _, b := range blocks {
				content += b.Text
			}
		}
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		if content != "" {
			summaryParts = append(summaryParts, fmt.Sprintf("[%s]: %s", msg.Role, content))
		}
	}

	if len(summaryParts) == 0 {
		return messages, nil
	}

	// Ask LLM to summarize
	summaryPrompt := "Summarize this conversation history concisely. Keep key decisions, file paths, and action items:\n\n" +
		strings.Join(summaryParts, "\n")

	summaryContent, _ := json.Marshal(summaryPrompt)
	req := &types.MessagesRequest{
		Model: model,
		Messages: []types.Message{
			{Role: "user", Content: summaryContent},
		},
		MaxTokens: 2000,
	}

	ch, err := provider.StreamMessage(ctx, req, nil)
	if err != nil {
		return messages, fmt.Errorf("compact summary failed: %w", err)
	}

	var summary string
	for event := range ch {
		if event.Type == "content_block_delta" && event.Delta != nil {
			var delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			json.Unmarshal(event.Delta, &delta)
			summary += delta.Text
		}
		if event.Type == "message_stop" {
			break
		}
	}

	if summary == "" {
		return messages, fmt.Errorf("compact produced empty summary")
	}

	// Build compacted message list: summary + recent
	compactedContent, _ := json.Marshal("[Conversation Summary]\n" + summary)
	compacted := []types.Message{
		{Role: "user", Content: compactedContent},
		{Role: "assistant", Content: json.RawMessage(`"Understood. I have the conversation context from the summary above. Continuing..."`)},
	}
	compacted = append(compacted, recentMessages...)

	log.Printf("[Compact] Reduced %d → %d messages", len(messages), len(compacted))
	return compacted, nil
}

// EstimateMessageTokens gives a rough token count for messages.
func EstimateMessageTokens(messages []types.Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content) / 4 // rough: 4 chars per token
	}
	return total
}
