package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aniclew/aniclew/internal/types"
)

const baseSystemPrompt = `You are AniClew, an expert coding assistant.

## Tools: Bash, Read, Write, Edit, Glob, Grep, Git, LS, WebSearch, WebFetch, TaskCreate/Update/List, NotebookRead/Edit, Screenshot, MouseClick, TypeText, OpenApp, FileManager, Clipboard

## Rules
- Read files BEFORE modifying them
- Use Glob/Grep to find files instead of guessing paths
- Use Edit (not Write) to modify existing files
- Run tests after changes when possible
- For git: use Git tool (not Bash)
- Keep changes minimal and focused
- Be concise`

var langInstructions = map[string]string{
	"ko": "\n\nIMPORTANT: Always respond in Korean (한국어). Code and file paths stay in English, but all explanations, comments to the user, and descriptions must be in Korean.",
	"en": "\n\nIMPORTANT: Always respond in English.",
	"ja": "\n\nIMPORTANT: Always respond in Japanese (日本語). Code and file paths stay in English, but all explanations must be in Japanese.",
	"zh": "\n\nIMPORTANT: Always respond in Chinese (中文). Code and file paths stay in English, but all explanations must be in Chinese.",
	"auto": "", // no language instruction — let the model follow the user's language
}

func buildSystemPrompt(responseLang string) string {
	instruction := langInstructions[responseLang]
	if instruction == "" {
		instruction = langInstructions["auto"]
	}
	return baseSystemPrompt + instruction
}

// Event is sent to the client via SSE during the agent loop.
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// RunLoop executes the agent loop: prompt → LLM → tool_use → execute → repeat.
func RunLoop(
	ctx context.Context,
	provider types.Provider,
	model string,
	userMessages []types.Message,
	workDir string,
	responseLang string,
	eventCh chan<- Event,
) {
	defer close(eventCh)

	messages := make([]types.Message, len(userMessages))
	copy(messages, userMessages)
	tools := AllToolDefs(workDir)

	maxIterations := 25 // increased from 20

	// ── Detect project type ──
	project := DetectProject(workDir)
	projectPrompt := project.ToPrompt()
	eventCh <- Event{Type: "status", Data: fmt.Sprintf("Project: %s (%s, %d files)", project.Name, project.Type, project.FileCount)}

	// ── Load project context (CLAUDE.md, AGENTS.md, skills) ──
	projectCtx := LoadProjectContext(workDir)
	skills := LoadSkills(workDir)
	mcpConfig := LoadMCPConfig(workDir)

	// ── Process slash commands ──
	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		var lastText string
		json.Unmarshal(lastMsg.Content, &lastText)
		if IsSlashCommand(lastText) {
			processed, err := ProcessSlashCommand(lastText, skills)
			if err != nil {
				eventCh <- Event{Type: "error", Data: err.Error()}
				return
			}
			// Direct output commands — don't send to LLM
			if processed == "[CLEAR_CHAT]" || processed == "[SHOW_MODEL_SELECTOR]" {
				eventCh <- Event{Type: "command", Data: processed}
				return
			}
			if processed == "[COMPACT_CONTEXT]" {
				eventCh <- Event{Type: "status", Data: "Compressing context..."}
			}
			// /help → return directly, no LLM needed
			if strings.HasPrefix(lastText, "/help") {
				eventCh <- Event{Type: "text", Data: processed}
				eventCh <- Event{Type: "done", Data: nil}
				return
			}
			// Replace last message with processed skill prompt
			messages[len(messages)-1] = types.Message{
				Role:    "user",
				Content: mustJSON(processed),
			}
			eventCh <- Event{Type: "status", Data: "Skill loaded: " + lastText}
		}
	}

	// ── Connect MCP servers ──
	if mcpConfig != "" {
		count, _ := ConnectMCPServers(workDir)
		if count > 0 {
			eventCh <- Event{Type: "status", Data: fmt.Sprintf("Connected to %d MCP servers", count)}
		}
	}

	skillText := ""
	if len(skills) > 0 {
		skillText = "\n\n--- AVAILABLE SKILLS ---\n"
		for _, s := range skills {
			skillText += fmt.Sprintf("\n### Skill: %s\n%s\n", s.Name, s.Content)
		}
		eventCh <- Event{Type: "status", Data: fmt.Sprintf("Loaded %d skills", len(skills))}
	}
	if projectCtx != "" {
		eventCh <- Event{Type: "status", Data: "Project context loaded (CLAUDE.md)"}
	}
	if mcpConfig != "" {
		eventCh <- Event{Type: "status", Data: "MCP config detected"}
	}

	for i := 0; i < maxIterations; i++ {
		// ── Context compression: estimate tokens and compress if needed ──
		totalChars := len(projectCtx) + len(skillText)
		for _, m := range messages {
			totalChars += len(m.Content)
		}
		estimatedTokens := totalChars / 4

		if estimatedTokens > 80000 && len(messages) > 6 {
			eventCh <- Event{Type: "status", Data: fmt.Sprintf("Context large (~%dk tokens), compressing...", estimatedTokens/1000)}
			// Keep first 2 messages (system context) and last 4 messages, summarize the middle
			if len(messages) > 8 {
				middle := make([]map[string]string, 0)
				for _, m := range messages[2 : len(messages)-4] {
					var text string
					json.Unmarshal(m.Content, &text)
					middle = append(middle, map[string]string{"role": m.Role, "content": text})
				}
				summary := CompressContext(middle)
				summaryMsg := types.Message{
					Role:    "user",
					Content: mustJSON("[Context Summary]\n" + summary),
				}
				compressed := make([]types.Message, 0)
				compressed = append(compressed, messages[:2]...)
				compressed = append(compressed, summaryMsg)
				compressed = append(compressed, messages[len(messages)-4:]...)
				messages = compressed
				eventCh <- Event{Type: "status", Data: fmt.Sprintf("Compressed to %d messages", len(messages))}
			}
		}

		// Build request with full context
		sysPrompt := buildSystemPrompt(responseLang) + projectPrompt + projectCtx + skillText
		req := &types.MessagesRequest{
			Model:    model,
			System:   mustJSON([]map[string]string{{"type": "text", "text": sysPrompt}}),
			Messages: messages,
			Tools:    tools,
			MaxTokens: 8192,
		}

		// Call LLM (with retry)
		eventCh <- Event{Type: "status", Data: fmt.Sprintf("Thinking... (iteration %d/%d, ~%dk tokens)", i+1, maxIterations, estimatedTokens/1000)}

		var ch <-chan types.SSEEvent
		var err error
		for retry := 0; retry < 3; retry++ {
			ch, err = provider.StreamMessage(ctx, req, nil)
			if err == nil {
				break
			}
			if retry < 2 {
				eventCh <- Event{Type: "status", Data: fmt.Sprintf("Retrying... (%d/3): %s", retry+1, err.Error())}
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
				}
			}
		}
		if err != nil {
			eventCh <- Event{Type: "error", Data: fmt.Sprintf("Failed after 3 retries: %s", err.Error())}
			return
		}

		// Collect response
		var textContent string
		var toolUses []toolUseBlock
		currentText := ""
		var currentTool *toolUseBlock

		for event := range ch {
			switch event.Type {
			case "content_block_start":
				var block struct {
					Type string `json:"type"`
					ID   string `json:"id"`
					Name string `json:"name"`
				}
				json.Unmarshal(event.ContentBlock, &block)

				if block.Type == "thinking" {
					// Thinking block — stream to UI
					eventCh <- Event{Type: "status", Data: "Thinking..."}
				} else if block.Type == "text" {
					currentText = ""
				} else if block.Type == "tool_use" {
					currentTool = &toolUseBlock{ID: block.ID, Name: block.Name}
					eventCh <- Event{Type: "tool_start", Data: map[string]string{
						"id": block.ID, "name": block.Name,
					}}
				}

			case "content_block_delta":
				var delta struct {
					Type        string `json:"type"`
					Text        string `json:"text"`
					PartialJSON string `json:"partial_json"`
				}
				json.Unmarshal(event.Delta, &delta)

				if delta.Type == "thinking_delta" {
					// Stream thinking to UI as dimmed text
					var thinkDelta struct{ Thinking string `json:"thinking"` }
					json.Unmarshal(event.Delta, &thinkDelta)
					if thinkDelta.Thinking != "" {
						eventCh <- Event{Type: "thinking", Data: thinkDelta.Thinking}
					}
				} else if delta.Type == "text_delta" {
					currentText += delta.Text
					eventCh <- Event{Type: "text", Data: delta.Text}
				} else if delta.Type == "input_json_delta" && currentTool != nil {
					currentTool.InputRaw += delta.PartialJSON
				}

			case "content_block_stop":
				if currentTool != nil {
					currentTool.Input = json.RawMessage(currentTool.InputRaw)
					toolUses = append(toolUses, *currentTool)
					currentTool = nil
				} else if currentText != "" {
					textContent += currentText
				}

			case "message_stop":
				// done with this LLM call
			}
		}

		// ── No tool calls → done ──
		if len(toolUses) == 0 {
			eventCh <- Event{Type: "done", Data: map[string]interface{}{
				"iterations":     i + 1,
				"estimatedTokens": estimatedTokens,
				"project":        project.Type,
			}}
			return
		}

		// ── Build assistant message with tool_use blocks ──
		var assistantContent []map[string]interface{}
		if textContent != "" {
			assistantContent = append(assistantContent, map[string]interface{}{
				"type": "text", "text": textContent,
			})
		}
		for _, tu := range toolUses {
			assistantContent = append(assistantContent, map[string]interface{}{
				"type": "tool_use", "id": tu.ID, "name": tu.Name, "input": json.RawMessage(tu.InputRaw),
			})
		}
		messages = append(messages, types.Message{
			Role:    "assistant",
			Content: mustJSON(assistantContent),
		})

		// ── Execute tools and collect results ──
		var toolResults []map[string]interface{}
		for _, tu := range toolUses {
			log.Printf("[Agent] Executing: %s", tu.Name)

			// ── Permission check ──
			permCfg := DefaultPermissionConfig()
			permCfg.AutoApprove = "moderate" // allow safe + moderate by default
			allowed, permReason, dangerLevel := CheckPermission(tu.Name, tu.Input, workDir, permCfg)

			// Show tool input to client
			var inputPreview interface{}
			json.Unmarshal(tu.Input, &inputPreview)
			eventCh <- Event{Type: "tool_input", Data: map[string]interface{}{
				"id": tu.ID, "name": tu.Name, "input": inputPreview,
				"danger": string(dangerLevel),
			}}

			if !allowed {
				eventCh <- Event{Type: "tool_result", Data: map[string]interface{}{
					"id": tu.ID, "name": tu.Name,
					"result": fmt.Sprintf("[BLOCKED] %s", permReason), "isError": true,
				}}
				toolResults = append(toolResults, map[string]interface{}{
					"type": "tool_result", "tool_use_id": tu.ID,
					"content": fmt.Sprintf("Permission denied: %s", permReason), "is_error": true,
				})
				continue
			}

			result, isError := ExecuteTool(tu.Name, tu.Input, workDir)

			// Send result to client
			eventCh <- Event{Type: "tool_result", Data: map[string]interface{}{
				"id": tu.ID, "name": tu.Name, "result": truncateStr(result, 2000), "isError": isError,
			}}

			toolResults = append(toolResults, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": tu.ID,
				"content":     result,
				"is_error":    isError,
			})
		}

		// ── Add tool results as user message ──
		messages = append(messages, types.Message{
			Role:    "user",
			Content: mustJSON(toolResults),
		})

		eventCh <- Event{Type: "status", Data: fmt.Sprintf("Iteration %d/%d — %d tools executed", i+1, maxIterations, len(toolUses))}
	}

	eventCh <- Event{Type: "error", Data: "Max iterations reached"}
}

type toolUseBlock struct {
	ID       string
	Name     string
	InputRaw string
	Input    json.RawMessage
}

func truncateStr(s string, max int) string {
	if len(s) <= max { return s }
	return s[:max] + "..."
}

func mustJSON(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
