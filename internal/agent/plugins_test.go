package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPluginLoad(t *testing.T) {
	dir := t.TempDir()

	// Create a test plugin
	pluginDir := filepath.Join(dir, "test-plugin")
	os.MkdirAll(pluginDir, 0755)

	manifest := Plugin{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Author:      "tester",
		Commands: []PluginCommand{
			{Name: "greet", Description: "Say hello", Command: "echo hello"},
		},
		Hooks: []PluginHook{
			{Event: "pre_tool_use", Command: "echo pre"},
		},
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"), data, 0644)

	// Load
	pm := NewPluginManager(dir)
	pm.LoadAll()

	plugins := pm.GetPlugins()
	if len(plugins) != 1 {
		t.Fatalf("Expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "test-plugin" {
		t.Errorf("Expected name 'test-plugin', got %q", plugins[0].Name)
	}

	cmds := pm.GetAllCommands()
	if len(cmds) != 1 || cmds[0].Name != "greet" {
		t.Errorf("Expected 1 command 'greet', got %v", cmds)
	}

	hooks := pm.GetAllHooks()
	if len(hooks) != 1 || hooks[0].Event != "pre_tool_use" {
		t.Errorf("Expected 1 hook, got %v", hooks)
	}
}

func TestPluginEmptyDir(t *testing.T) {
	pm := NewPluginManager(t.TempDir())
	pm.LoadAll()
	if len(pm.GetPlugins()) != 0 {
		t.Error("Empty dir should load 0 plugins")
	}
}
