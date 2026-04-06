package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadClaudeHooks(t *testing.T) {
	dir := t.TempDir()

	// Create .claude/settings.json with hooks
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"pre_tool_use": []map[string]interface{}{
				{"command": "echo pre-tool", "timeout": 5},
			},
			"post_tool_use": []map[string]interface{}{
				{"command": "echo post-tool"},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	// Load hooks
	registry := NewRegistry()
	registry.Load(dir, "claude")

	hooks := registry.GetHooks()
	if len(hooks) != 2 {
		t.Fatalf("Expected 2 hooks, got %d", len(hooks))
	}

	// Check types
	preHooks := 0
	postHooks := 0
	for _, h := range hooks {
		if h.Type == HookPreToolUse {
			preHooks++
			if h.Command != "echo pre-tool" {
				t.Errorf("Expected 'echo pre-tool', got %q", h.Command)
			}
			if h.Timeout != 5 {
				t.Errorf("Expected timeout 5, got %d", h.Timeout)
			}
		}
		if h.Type == HookPostToolUse {
			postHooks++
		}
	}
	if preHooks != 1 {
		t.Errorf("Expected 1 pre hook, got %d", preHooks)
	}
	if postHooks != 1 {
		t.Errorf("Expected 1 post hook, got %d", postHooks)
	}
}

func TestHookExecution(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	// Manually add a hook
	registry.hooks = []Hook{
		{Type: HookPreToolUse, Command: "echo hello_hook", Timeout: 5, Source: "test"},
	}
	registry.workDir = dir

	results := registry.Execute(HookPreToolUse, nil)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Blocked {
		t.Error("Hook should not block")
	}
	if results[0].Output != "hello_hook" {
		t.Errorf("Expected 'hello_hook', got %q", results[0].Output)
	}
}

func TestHookBlocking(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	// Hook that exits non-zero = blocks
	registry.hooks = []Hook{
		{Type: HookPreToolUse, Command: "exit 1", Timeout: 5, Source: "test"},
	}
	registry.workDir = dir

	blocked, _ := registry.IsBlocked(HookPreToolUse, nil)
	if !blocked {
		t.Error("Hook with exit 1 should block")
	}
}

func TestHookTimeout(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	// Hook that times out
	registry.hooks = []Hook{
		{Type: HookPreToolUse, Command: "sleep 10", Timeout: 1, Source: "test"},
	}
	registry.workDir = dir

	results := registry.Execute(HookPreToolUse, nil)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Error != "timeout" {
		t.Errorf("Expected 'timeout' error, got %q", results[0].Error)
	}
	// Timeout should NOT block (just warn)
	if results[0].Blocked {
		t.Error("Timeout should not block")
	}
}

func TestHookSkillSourceFiltering(t *testing.T) {
	dir := t.TempDir()

	// Create claude hooks
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"session_start": []map[string]interface{}{
				{"command": "echo claude-start"},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	// Load with "codex" source → should NOT load claude hooks
	registry := NewRegistry()
	registry.Load(dir, "codex")
	if len(registry.GetHooks()) != 0 {
		t.Errorf("Codex source should not load claude hooks, got %d", len(registry.GetHooks()))
	}

	// Load with "all" source → should load claude hooks
	registry2 := NewRegistry()
	registry2.Load(dir, "all")
	if len(registry2.GetHooks()) != 1 {
		t.Errorf("All source should load claude hooks, got %d", len(registry2.GetHooks()))
	}

	// Load with "none" → nothing
	registry3 := NewRegistry()
	registry3.Load(dir, "none")
	if len(registry3.GetHooks()) != 0 {
		t.Errorf("None source should load 0, got %d", len(registry3.GetHooks()))
	}
}

func TestHookEnvironment(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	registry.hooks = []Hook{
		{Type: HookPreToolUse, Command: "echo $TOOL_NAME", Timeout: 5, Source: "test"},
	}
	registry.workDir = dir

	results := registry.Execute(HookPreToolUse, map[string]string{
		"TOOL_NAME": "Bash",
	})
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Output != "Bash" {
		t.Errorf("Expected 'Bash' from env, got %q", results[0].Output)
	}
}
