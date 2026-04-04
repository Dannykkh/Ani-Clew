package agent

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

type DangerLevel string

const (
	DangerSafe      DangerLevel = "safe"
	DangerModerate  DangerLevel = "moderate"
	DangerDangerous DangerLevel = "dangerous"
)

type PermissionConfig struct {
	AutoApprove     string   `json:"autoApprove"` // "safe", "moderate", "all", "none"
	BlockedPaths    []string `json:"blockedPaths"`
	BlockedCommands []string `json:"blockedCommands"`
}

func DefaultPermissionConfig() PermissionConfig {
	return PermissionConfig{
		AutoApprove: "safe",
		BlockedPaths: []string{
			"/etc/passwd", "/etc/shadow", ".ssh/", ".aws/credentials",
			".env", ".env.local", ".env.production",
		},
		BlockedCommands: []string{
			"rm -rf /", "mkfs", "dd if=", ":(){ :|:& };:", "> /dev/sda",
			"shutdown", "reboot",
		},
	}
}

var dangerousBashPatterns = []string{
	"rm -rf", "rm -r /", "sudo rm", "chmod 777", "mkfs", "dd if=",
	":(){ :|:& };:", "> /dev/sda", "shutdown", "reboot", "kill -9 1",
	"pkill -9", "> /dev/null 2>&1 &", "curl | sh", "wget | sh",
}

var moderateBashPatterns = []string{
	"rm ", "mv ", "cp -r", "git push", "git reset --hard",
	"npm publish", "docker rm", "pip install", "apt install",
	"brew install", "chmod", "chown",
}

// ClassifyDanger returns the danger level for a tool call.
func ClassifyDanger(toolName string, input json.RawMessage) (DangerLevel, string) {
	switch toolName {
	case "Bash":
		var args struct{ Command string `json:"command"` }
		json.Unmarshal(input, &args)
		cmd := strings.ToLower(args.Command)

		for _, p := range dangerousBashPatterns {
			if strings.Contains(cmd, p) {
				return DangerDangerous, "Dangerous command: " + p
			}
		}
		for _, p := range moderateBashPatterns {
			if strings.Contains(cmd, p) {
				return DangerModerate, "Potentially risky: " + p
			}
		}
		return DangerSafe, ""

	case "Write":
		return DangerModerate, "Creating/overwriting file"

	case "Git":
		var args struct {
			Command string `json:"command"`
			Args    string `json:"args"`
		}
		json.Unmarshal(input, &args)
		full := args.Command + " " + args.Args

		if strings.Contains(full, "--force") || strings.Contains(full, "reset --hard") {
			return DangerDangerous, "Destructive git operation"
		}
		if args.Command == "push" || args.Command == "commit" || args.Command == "add" {
			return DangerModerate, "Git mutating command"
		}
		return DangerSafe, ""

	case "Edit":
		return DangerSafe, ""

	case "Read", "Glob", "Grep", "LS", "WebSearch", "WebFetch",
		"TaskCreate", "TaskUpdate", "TaskList",
		"NotebookRead":
		return DangerSafe, ""

	case "NotebookEdit":
		return DangerModerate, "Modifying notebook"

	default:
		return DangerModerate, "Unknown tool"
	}
}

// CheckPath validates that a file path is safe to access.
func CheckPath(path string, workDir string, cfg PermissionConfig) (bool, string) {
	// Block known sensitive paths
	for _, blocked := range cfg.BlockedPaths {
		if strings.Contains(path, blocked) {
			return false, "Blocked path: " + blocked
		}
	}

	// Block path traversal outside workDir
	absPath, err := filepath.Abs(path)
	if err == nil {
		absWork, _ := filepath.Abs(workDir)
		if !strings.HasPrefix(absPath, absWork) {
			// Allow home directory and /tmp
			if !strings.HasPrefix(absPath, "/tmp") && !strings.Contains(absPath, ".claude-proxy") {
				return false, "Path outside workspace: " + path
			}
		}
	}

	return true, ""
}

// CheckPermission determines if a tool call should be allowed.
func CheckPermission(toolName string, input json.RawMessage, workDir string, cfg PermissionConfig) (bool, string, DangerLevel) {
	level, reason := ClassifyDanger(toolName, input)

	// Check blocked commands for Bash
	if toolName == "Bash" {
		var args struct{ Command string `json:"command"` }
		json.Unmarshal(input, &args)
		for _, blocked := range cfg.BlockedCommands {
			if strings.Contains(strings.ToLower(args.Command), strings.ToLower(blocked)) {
				return false, "Blocked command: " + blocked, DangerDangerous
			}
		}
	}

	// Check file paths
	if toolName == "Read" || toolName == "Write" || toolName == "Edit" {
		var args struct{ FilePath string `json:"file_path"` }
		json.Unmarshal(input, &args)
		if args.FilePath != "" {
			if ok, msg := CheckPath(args.FilePath, workDir, cfg); !ok {
				return false, msg, DangerDangerous
			}
		}
	}

	// Auto-approve check
	switch cfg.AutoApprove {
	case "all":
		return true, "", level
	case "none":
		return false, "Manual approval required", level
	case "moderate":
		if level == DangerDangerous {
			return false, reason, level
		}
		return true, "", level
	default: // "safe"
		if level != DangerSafe {
			return false, reason, level
		}
		return true, "", level
	}
}
