package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aniclew/aniclew/internal/types"
)

// SubAgentTask represents a task assigned to a sub-agent.
type SubAgentTask struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Instruction string    `json:"instruction"`
	Files       []string  `json:"files"`       // files this agent owns
	Status      string    `json:"status"`       // "pending", "running", "completed", "failed"
	Result      string    `json:"result"`
	StartedAt   time.Time `json:"startedAt,omitempty"`
	FinishedAt  time.Time `json:"finishedAt,omitempty"`
	ToolCalls   int       `json:"toolCalls"`
}

// SubAgentManager manages parallel sub-agents.
type SubAgentManager struct {
	mu       sync.RWMutex
	tasks    map[string]*SubAgentTask
	provider types.Provider
	model    string
	workDir  string
	counter  int
}

func NewSubAgentManager(provider types.Provider, model, workDir string) *SubAgentManager {
	return &SubAgentManager{
		tasks:    make(map[string]*SubAgentTask),
		provider: provider,
		model:    model,
		workDir:  workDir,
	}
}

// Spawn creates and starts a sub-agent in a separate goroutine.
func (m *SubAgentManager) Spawn(name, instruction string, files []string) *SubAgentTask {
	m.mu.Lock()
	m.counter++
	id := fmt.Sprintf("sub-%d", m.counter)
	task := &SubAgentTask{
		ID:          id,
		Name:        name,
		Instruction: instruction,
		Files:       files,
		Status:      "pending",
	}
	m.tasks[id] = task
	m.mu.Unlock()

	// Start in goroutine
	go m.run(task)

	return task
}

// SpawnMultiple creates multiple sub-agents and runs them in parallel.
func (m *SubAgentManager) SpawnMultiple(tasks []struct {
	Name        string
	Instruction string
	Files       []string
}) []*SubAgentTask {
	var results []*SubAgentTask
	for _, t := range tasks {
		results = append(results, m.Spawn(t.Name, t.Instruction, t.Files))
	}
	return results
}

// Wait blocks until all sub-agents are done.
func (m *SubAgentManager) Wait(timeout time.Duration) {
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			log.Println("[SubAgent] Timeout waiting for agents")
			return
		default:
			if m.AllDone() {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// AllDone returns true if all tasks are completed or failed.
func (m *SubAgentManager) AllDone() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.tasks {
		if t.Status == "pending" || t.Status == "running" {
			return false
		}
	}
	return true
}

// GetTasks returns all sub-agent tasks.
func (m *SubAgentManager) GetTasks() []*SubAgentTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*SubAgentTask
	for _, t := range m.tasks {
		result = append(result, t)
	}
	return result
}

// GetTask returns a specific task.
func (m *SubAgentManager) GetTask(id string) *SubAgentTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tasks[id]
}

// run executes a sub-agent's task using its own agent loop.
func (m *SubAgentManager) run(task *SubAgentTask) {
	m.mu.Lock()
	task.Status = "running"
	task.StartedAt = time.Now()
	m.mu.Unlock()

	log.Printf("[SubAgent] %s started: %s", task.ID, task.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Build sub-agent system prompt
	// Note: /no_think disables qwen3's reasoning mode for direct output
	sysPrompt := fmt.Sprintf(`/no_think
You are a sub-agent named "%s" working on a specific task.
You have access to tools: Bash, Read, Write, Edit, Glob, Grep, Git, LS.
You ONLY work on these files: %s
Do NOT modify files outside your assigned scope.

Your task: %s

Be efficient. Complete the task and report what you did.`,
		task.Name, strings.Join(task.Files, ", "), task.Instruction)

	req := &types.MessagesRequest{
		Model:     m.model,
		System:    mustJSON([]map[string]string{{"type": "text", "text": sysPrompt}}),
		Messages:  []types.Message{{Role: "user", Content: mustJSON(task.Instruction)}},
		Tools:     AllToolDefs(m.workDir),
		MaxTokens: 8192,
	}

	// Mini agent loop (max 10 iterations)
	var finalResponse string
	toolCalls := 0

	for i := 0; i < 10; i++ {
		ch, err := m.provider.StreamMessage(ctx, req, nil)
		if err != nil {
			m.mu.Lock()
			task.Status = "failed"
			task.Result = fmt.Sprintf("Error: %v", err)
			task.FinishedAt = time.Now()
			m.mu.Unlock()
			log.Printf("[SubAgent] %s failed: %v", task.ID, err)
			return
		}

		var textContent string
		var toolUses []toolUseBlock

		for event := range ch {
			switch event.Type {
			case "content_block_delta":
				var delta struct {
					Type        string `json:"type"`
					Text        string `json:"text"`
					PartialJSON string `json:"partial_json"`
				}
				json.Unmarshal(event.Delta, &delta)
				if delta.Type == "text_delta" {
					textContent += delta.Text
				}
			case "content_block_start":
				var block struct {
					Type string `json:"type"`
					ID   string `json:"id"`
					Name string `json:"name"`
				}
				json.Unmarshal(event.ContentBlock, &block)
				if block.Type == "tool_use" {
					toolUses = append(toolUses, toolUseBlock{ID: block.ID, Name: block.Name})
				}
			}
		}

		// No tool calls → done
		if len(toolUses) == 0 {
			finalResponse = textContent
			break
		}

		// Execute tools
		var assistantContent []map[string]interface{}
		if textContent != "" {
			assistantContent = append(assistantContent, map[string]interface{}{"type": "text", "text": textContent})
		}

		var toolResults []map[string]interface{}
		for _, tu := range toolUses {
			toolCalls++

			// File ownership check
			if !m.isAllowedFile(task, tu.Name, tu.Input) {
				toolResults = append(toolResults, map[string]interface{}{
					"type": "tool_result", "tool_use_id": tu.ID,
					"content": "Permission denied: file outside your scope", "is_error": true,
				})
				continue
			}

			result, isError := ExecuteTool(tu.Name, tu.Input, m.workDir)
			assistantContent = append(assistantContent, map[string]interface{}{
				"type": "tool_use", "id": tu.ID, "name": tu.Name, "input": json.RawMessage(tu.InputRaw),
			})
			toolResults = append(toolResults, map[string]interface{}{
				"type": "tool_result", "tool_use_id": tu.ID,
				"content": result, "is_error": isError,
			})
		}

		// Update messages for next iteration
		req.Messages = append(req.Messages,
			types.Message{Role: "assistant", Content: mustJSON(assistantContent)},
			types.Message{Role: "user", Content: mustJSON(toolResults)},
		)
	}

	m.mu.Lock()
	task.Status = "completed"
	task.Result = finalResponse
	task.ToolCalls = toolCalls
	task.FinishedAt = time.Now()
	m.mu.Unlock()

	log.Printf("[SubAgent] %s completed: %d tool calls, %.1fs",
		task.ID, toolCalls, time.Since(task.StartedAt).Seconds())
}

// isAllowedFile checks if a tool call targets files within the agent's scope.
func (m *SubAgentManager) isAllowedFile(task *SubAgentTask, toolName string, input json.RawMessage) bool {
	if len(task.Files) == 0 {
		return true // no restriction
	}

	// Only check file-modifying tools
	if toolName != "Write" && toolName != "Edit" {
		return true
	}

	var args struct{ FilePath string `json:"file_path"` }
	json.Unmarshal(input, &args)
	if args.FilePath == "" {
		return true
	}

	// Check if file matches any allowed pattern
	for _, allowed := range task.Files {
		if strings.Contains(args.FilePath, allowed) || matchGlob(args.FilePath, allowed) {
			return true
		}
	}

	return false
}

func matchGlob(path, pattern string) bool {
	// Simple glob: "src/**" matches "src/foo/bar.go"
	if strings.HasSuffix(pattern, "**") {
		prefix := strings.TrimSuffix(pattern, "**")
		return strings.HasPrefix(path, prefix)
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}
	return path == pattern
}
