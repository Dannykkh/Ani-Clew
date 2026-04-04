package translate

import "github.com/aniclew/aniclew/internal/types"

// ToOpenAI converts an Anthropic Messages request to an OpenAI Chat Completions request.
func ToOpenAI(req *types.MessagesRequest, model string) types.OAIChatRequest {
	var msgs []types.OAIMessage

	// System prompt
	if sys := SystemToOAI(req.System); sys != nil {
		msgs = append(msgs, *sys)
	}

	// Messages
	msgs = append(msgs, MessagesToOAI(req.Messages)...)

	result := types.OAIChatRequest{
		Model:         model,
		Messages:      msgs,
		Stream:        true,
		StreamOptions: &types.StreamOpts{IncludeUsage: true},
		MaxTokens:     req.MaxTokens,
	}

	// Temperature
	if req.Temperature != nil && len(req.Thinking) == 0 {
		result.Temperature = req.Temperature
	}

	// Tools
	if tools := ToolDefsToOAI(req.Tools); tools != nil {
		result.Tools = tools
	}

	// Tool choice
	if tc := ToolChoiceToOAI(req.ToolChoice); tc != nil {
		result.ToolChoice = tc
	}

	return result
}
