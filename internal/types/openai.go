package types

import "encoding/json"

// ── OpenAI Messages ──

type OAIMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content,omitempty"`
	ToolCalls  []OAIToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type OAIContentPart struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	ImageURL *OAIImageURL `json:"image_url,omitempty"`
}

type OAIImageURL struct {
	URL string `json:"url"`
}

// ── OpenAI Tools ──

type OAIToolDef struct {
	Type     string      `json:"type"` // "function"
	Function OAIFunction `json:"function"`
}

type OAIFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type OAIToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"` // "function"
	Function OAIFunctionCall `json:"function"`
}

type OAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ── OpenAI Request ──

type OAIChatRequest struct {
	Model         string          `json:"model"`
	Messages      []OAIMessage    `json:"messages"`
	Tools         []OAIToolDef    `json:"tools,omitempty"`
	ToolChoice    json.RawMessage `json:"tool_choice,omitempty"`
	MaxTokens     int             `json:"max_tokens,omitempty"`
	Temperature   *float64        `json:"temperature,omitempty"`
	Stream        bool            `json:"stream"`
	StreamOptions *StreamOpts     `json:"stream_options,omitempty"`
}

type StreamOpts struct {
	IncludeUsage bool `json:"include_usage"`
}

// ── OpenAI Streaming Response ──

type OAIStreamChunk struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []OAIStreamChoice `json:"choices"`
	Usage   *OAIUsage         `json:"usage,omitempty"`
}

type OAIStreamChoice struct {
	Index        int             `json:"index"`
	Delta        OAIStreamDelta  `json:"delta"`
	FinishReason *string         `json:"finish_reason"`
}

type OAIStreamDelta struct {
	Role      string              `json:"role,omitempty"`
	Content   *string             `json:"content,omitempty"`
	ToolCalls []OAIStreamToolCall `json:"tool_calls,omitempty"`
}

type OAIStreamToolCall struct {
	Index    int              `json:"index"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function *OAIStreamFnCall `json:"function,omitempty"`
}

type OAIStreamFnCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type OAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
