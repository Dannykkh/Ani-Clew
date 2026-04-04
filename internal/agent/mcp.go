package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// MCPServerConfig represents an MCP server from .mcp.json
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// MCPConfig represents the .mcp.json file structure
type MCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPConnection holds a running MCP server process.
type MCPConnection struct {
	Name    string
	Config  MCPServerConfig
	Process *exec.Cmd
	Running bool
}

var mcpConnections = make(map[string]*MCPConnection)

// ParseMCPConfig reads and parses .mcp.json
func ParseMCPConfig(configJSON string) (*MCPConfig, error) {
	if configJSON == "" {
		return nil, fmt.Errorf("no MCP config")
	}
	var cfg MCPConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("parse .mcp.json: %w", err)
	}
	return &cfg, nil
}

// ListMCPServers returns available MCP server names and their commands.
func ListMCPServers(workDir string) []map[string]string {
	configJSON := LoadMCPConfig(workDir)
	if configJSON == "" {
		return nil
	}
	cfg, err := ParseMCPConfig(configJSON)
	if err != nil {
		return nil
	}

	var servers []map[string]string
	for name, srv := range cfg.MCPServers {
		status := "stopped"
		if conn, ok := mcpConnections[name]; ok && conn.Running {
			status = "running"
		}
		servers = append(servers, map[string]string{
			"name":    name,
			"command": srv.Command + " " + strings.Join(srv.Args, " "),
			"status":  status,
		})
	}
	return servers
}

// StartMCPServer starts an MCP server process.
func StartMCPServer(name string, workDir string) (string, error) {
	configJSON := LoadMCPConfig(workDir)
	cfg, err := ParseMCPConfig(configJSON)
	if err != nil {
		return "", err
	}

	srv, ok := cfg.MCPServers[name]
	if !ok {
		return "", fmt.Errorf("MCP server '%s' not found in config", name)
	}

	cmd := exec.Command(srv.Command, srv.Args...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()
	for k, v := range srv.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start MCP server '%s': %w", name, err)
	}

	mcpConnections[name] = &MCPConnection{
		Name: name, Config: srv, Process: cmd, Running: true,
	}

	return fmt.Sprintf("MCP server '%s' started (PID %d)", name, cmd.Process.Pid), nil
}

// StopMCPServer stops a running MCP server.
func StopMCPServer(name string) string {
	conn, ok := mcpConnections[name]
	if !ok || !conn.Running {
		return fmt.Sprintf("MCP server '%s' is not running", name)
	}
	conn.Process.Process.Kill()
	conn.Running = false
	return fmt.Sprintf("MCP server '%s' stopped", name)
}

// ── Sub-Agent System ──

// SubAgent represents a spawned sub-agent for parallel work.
type SubAgent struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Task     string `json:"task"`
	Status   string `json:"status"` // "running", "completed", "failed"
	Result   string `json:"result"`
}

var subAgents = make(map[string]*SubAgent)
var subAgentCounter = 0

// SpawnSubAgent creates a conceptual sub-agent (executed sequentially in current impl).
func SpawnSubAgent(name, task string) *SubAgent {
	subAgentCounter++
	id := fmt.Sprintf("agent-%d", subAgentCounter)
	agent := &SubAgent{
		ID: id, Name: name, Task: task, Status: "running",
	}
	subAgents[id] = agent
	return agent
}

// CompleteSubAgent marks a sub-agent as done.
func CompleteSubAgent(id, result string) {
	if a, ok := subAgents[id]; ok {
		a.Status = "completed"
		a.Result = result
	}
}

// ListSubAgents returns all sub-agents.
func ListSubAgents() []*SubAgent {
	var result []*SubAgent
	for _, a := range subAgents {
		result = append(result, a)
	}
	return result
}
