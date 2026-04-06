package agent

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// Plugin represents a loaded plugin.
type Plugin struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	Tools       []PluginTool      `json:"tools,omitempty"`
	Hooks       []PluginHook      `json:"hooks,omitempty"`
	Commands    []PluginCommand   `json:"commands,omitempty"`
	AgentTypes  []PluginAgentType `json:"agents,omitempty"`
}

// PluginTool is a tool provided by a plugin.
type PluginTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Command     string `json:"command"` // shell command to execute
}

// PluginHook is a hook registered by a plugin.
type PluginHook struct {
	Event   string `json:"event"`   // pre_tool_use, post_tool_use, etc.
	Command string `json:"command"`
}

// PluginCommand is a slash command added by a plugin.
type PluginCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Command     string `json:"command"`
}

// PluginAgentType is a custom agent type from a plugin.
type PluginAgentType struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"systemPrompt"`
	Tools        []string `json:"tools"`
}

// PluginManager loads and manages plugins.
type PluginManager struct {
	plugins []Plugin
	dirs    []string // directories to scan
}

// NewPluginManager creates a plugin manager.
func NewPluginManager(dirs ...string) *PluginManager {
	return &PluginManager{dirs: dirs}
}

// LoadAll scans plugin directories and loads plugin manifests.
func (pm *PluginManager) LoadAll() {
	pm.plugins = nil

	for _, dir := range pm.dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if !e.IsDir() {
				continue
			}

			manifestPath := filepath.Join(dir, e.Name(), "plugin.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				continue
			}

			var plugin Plugin
			if err := json.Unmarshal(data, &plugin); err != nil {
				log.Printf("[Plugins] Failed to parse %s: %v", manifestPath, err)
				continue
			}

			pm.plugins = append(pm.plugins, plugin)
			log.Printf("[Plugins] Loaded: %s v%s (%d tools, %d hooks, %d commands)",
				plugin.Name, plugin.Version,
				len(plugin.Tools), len(plugin.Hooks), len(plugin.Commands))
		}
	}
}

// GetPlugins returns all loaded plugins.
func (pm *PluginManager) GetPlugins() []Plugin {
	return pm.plugins
}

// GetAllTools returns tools from all plugins.
func (pm *PluginManager) GetAllTools() []PluginTool {
	var tools []PluginTool
	for _, p := range pm.plugins {
		tools = append(tools, p.Tools...)
	}
	return tools
}

// GetAllHooks returns hooks from all plugins.
func (pm *PluginManager) GetAllHooks() []PluginHook {
	var hooks []PluginHook
	for _, p := range pm.plugins {
		hooks = append(hooks, p.Hooks...)
	}
	return hooks
}

// GetAllCommands returns commands from all plugins.
func (pm *PluginManager) GetAllCommands() []PluginCommand {
	var commands []PluginCommand
	for _, p := range pm.plugins {
		commands = append(commands, p.Commands...)
	}
	return commands
}
