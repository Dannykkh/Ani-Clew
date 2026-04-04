package translate

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aniclew/aniclew/internal/types"
)

// Translator converts OpenAI stream chunks into Anthropic SSE events.
type Translator struct {
	model       string
	messageID   string
	blockIndex  int
	textOpen    bool
	toolCalls   map[int]string // index -> tool ID
	outTokens   int
}

func NewTranslator(model string) *Translator {
	return &Translator{
		model:     model,
		messageID: generateID("msg_proxy_"),
		toolCalls: make(map[int]string),
	}
}

// Start yields the message_start event.
func (t *Translator) Start() types.SSEEvent {
	return types.SSEEvent{
		Type: "message_start",
		Message: &types.SSEMessage{
			ID:      t.messageID,
			Type:    "message",
			Role:    "assistant",
			Model:   t.model,
			Content: json.RawMessage(`[]`),
			Usage:   &types.SSEUsage{InputTokens: 0, OutputTokens: 0},
		},
	}
}

// Translate converts one OpenAI chunk into zero or more Anthropic events.
func (t *Translator) Translate(chunk types.OAIStreamChunk) []types.SSEEvent {
	var events []types.SSEEvent

	if chunk.Usage != nil {
		t.outTokens = chunk.Usage.CompletionTokens
	}

	if len(chunk.Choices) == 0 {
		return events
	}
	choice := chunk.Choices[0]
	delta := choice.Delta

	// ── Text content ──
	if delta.Content != nil && *delta.Content != "" {
		if !t.textOpen {
			idx := t.blockIndex
			events = append(events, types.SSEEvent{
				Type:         "content_block_start",
				Index:        &idx,
				ContentBlock: mustMarshal(map[string]string{"type": "text", "text": ""}),
			})
			t.textOpen = true
		}
		idx := t.blockIndex
		events = append(events, types.SSEEvent{
			Type:  "content_block_delta",
			Index: &idx,
			Delta: mustMarshal(map[string]string{"type": "text_delta", "text": *delta.Content}),
		})
	}

	// ── Tool calls ──
	for _, tc := range delta.ToolCalls {
		if t.textOpen {
			events = append(events, t.closeTextBlock()...)
		}

		// New tool call
		if tc.ID != "" && tc.Function != nil && tc.Function.Name != "" {
			t.toolCalls[tc.Index] = tc.ID
			idx := t.blockIndex
			events = append(events, types.SSEEvent{
				Type:  "content_block_start",
				Index: &idx,
				ContentBlock: mustMarshal(map[string]string{
					"type": "tool_use", "id": tc.ID, "name": tc.Function.Name, "input": "",
				}),
			})
		}

		// Argument delta
		if tc.Function != nil && tc.Function.Arguments != "" {
			idx := t.blockIndex
			events = append(events, types.SSEEvent{
				Type:  "content_block_delta",
				Index: &idx,
				Delta: mustMarshal(map[string]string{
					"type": "input_json_delta", "partial_json": tc.Function.Arguments,
				}),
			})
		}
	}

	// ── Finish ──
	if choice.FinishReason != nil {
		events = append(events, t.Finish(*choice.FinishReason)...)
	}

	return events
}

// End produces final events if stream ended without explicit finish.
func (t *Translator) End() []types.SSEEvent {
	var events []types.SSEEvent
	if t.textOpen {
		events = append(events, t.closeTextBlock()...)
	}
	events = append(events, types.SSEEvent{
		Type:  "message_delta",
		Delta: mustMarshal(map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}),
		Usage: &types.SSEUsage{OutputTokens: t.outTokens},
	})
	events = append(events, types.SSEEvent{Type: "message_stop"})
	return events
}

func (t *Translator) Finish(reason string) []types.SSEEvent {
	var events []types.SSEEvent
	if t.textOpen {
		events = append(events, t.closeTextBlock()...)
	}
	// Close any open tool blocks
	for range t.toolCalls {
		idx := t.blockIndex
		events = append(events, types.SSEEvent{Type: "content_block_stop", Index: &idx})
		t.blockIndex++
	}
	t.toolCalls = make(map[int]string)

	stopReason := mapFinishReason(reason)
	events = append(events, types.SSEEvent{
		Type:  "message_delta",
		Delta: mustMarshal(map[string]any{"stop_reason": stopReason, "stop_sequence": nil}),
		Usage: &types.SSEUsage{OutputTokens: t.outTokens},
	})
	events = append(events, types.SSEEvent{Type: "message_stop"})
	return events
}

func (t *Translator) closeTextBlock() []types.SSEEvent {
	idx := t.blockIndex
	t.blockIndex++
	t.textOpen = false
	return []types.SSEEvent{{Type: "content_block_stop", Index: &idx}}
}

func mapFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	default:
		return "end_turn"
	}
}

func generateID(prefix string) string {
	b := make([]byte, 18)
	rand.Read(b)
	return fmt.Sprintf("%s%s", prefix, base64.RawURLEncoding.EncodeToString(b))
}
