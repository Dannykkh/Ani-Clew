package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)


// PermissionRule defines an allow/deny rule.
type PermissionRule struct {
	Tool    string `json:"tool"`              // tool name or "*"
	Pattern string `json:"pattern,omitempty"` // command pattern (for bash)
	Source  string `json:"source"`            // where the rule came from
}

// PermissionSnapshot is an immutable set of rules captured at session start.
type PermissionSnapshot struct {
	AllowRules []PermissionRule `json:"allowRules"`
	DenyRules  []PermissionRule `json:"denyRules"`
	Mode       string           `json:"mode"` // "default", "bypassPermissions", "dontAsk", "acceptEdits"
}

// PersistAllowRule saves an "always allow" decision for future sessions.
func PersistAllowRule(workDir string, tool string, pattern string) error {
	path := filepath.Join(workDir, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(path), 0755)

	// Load existing
	data, _ := os.ReadFile(path)
	var settings map[string]interface{}
	if len(data) > 0 {
		json.Unmarshal(data, &settings)
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	// Add permission
	perms, _ := settings["permissions"].(map[string]interface{})
	if perms == nil {
		perms = make(map[string]interface{})
	}
	allowList, _ := perms["allow"].([]interface{})
	allowList = append(allowList, map[string]interface{}{
		"tool": tool, "pattern": pattern,
	})
	perms["allow"] = allowList
	settings["permissions"] = perms

	out, _ := json.MarshalIndent(settings, "", "  ")
	return os.WriteFile(path, out, 0644)
}

// PersistDenyRule saves an "always deny" decision.
func PersistDenyRule(workDir string, tool string, pattern string) error {
	path := filepath.Join(workDir, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(path), 0755)

	data, _ := os.ReadFile(path)
	var settings map[string]interface{}
	if len(data) > 0 {
		json.Unmarshal(data, &settings)
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	perms, _ := settings["permissions"].(map[string]interface{})
	if perms == nil {
		perms = make(map[string]interface{})
	}
	denyList, _ := perms["deny"].([]interface{})
	denyList = append(denyList, map[string]interface{}{
		"tool": tool, "pattern": pattern,
	})
	perms["deny"] = denyList
	settings["permissions"] = perms

	out, _ := json.MarshalIndent(settings, "", "  ")
	return os.WriteFile(path, out, 0644)
}

// Decide returns "allow", "deny", or "ask" for a tool+input combination.
func (ps *PermissionSnapshot) Decide(toolName string, input string) string {
	switch ps.Mode {
	case "bypassPermissions":
		return "allow"
	case "dontAsk":
		return "deny"
	case "acceptEdits":
		if toolName == "Write" || toolName == "Edit" {
			return "allow"
		}
	}

	// Check deny rules first (deny wins over allow)
	for _, rule := range ps.DenyRules {
		if matchRule(rule, toolName, input) {
			return "deny"
		}
	}

	// Check allow rules
	for _, rule := range ps.AllowRules {
		if matchRule(rule, toolName, input) {
			return "allow"
		}
	}

	return "ask"
}

func matchRule(rule PermissionRule, toolName, input string) bool {
	if rule.Tool != "*" && rule.Tool != toolName {
		return false
	}
	if rule.Pattern != "" && !strings.Contains(input, rule.Pattern) {
		return false
	}
	return true
}

// CapturePermissions reads rules from the 6-level cascade and creates a snapshot.
// Priority: CLI flags > org policy > local settings > user settings > project settings > defaults
func CapturePermissions(workDir string) PermissionSnapshot {
	snap := PermissionSnapshot{Mode: "default"}

	// Level 6: Defaults (most permissive for built-in read tools)
	snap.AllowRules = append(snap.AllowRules,
		PermissionRule{Tool: "Read", Source: "default"},
		PermissionRule{Tool: "Glob", Source: "default"},
		PermissionRule{Tool: "Grep", Source: "default"},
	)

	// Level 5: Project settings (.claude/settings.json in project)
	loadPermissionRules(filepath.Join(workDir, ".claude", "settings.json"), "project", &snap)

	// Level 4: User settings (~/.claude/settings.json)
	home, _ := os.UserHomeDir()
	loadPermissionRules(filepath.Join(home, ".claude", "settings.json"), "user", &snap)

	// Level 3: Local settings (.claude/settings-local.json — git-ignored)
	loadPermissionRules(filepath.Join(workDir, ".claude", "settings-local.json"), "local", &snap)

	// Level 2: Org policy (if exists)
	loadPermissionRules(filepath.Join(home, ".claude", "policy.json"), "policy", &snap)

	// Level 1: CLI flags (handled by caller, not file-based)
	// Applied via snap.Mode override

	return snap
}

func loadPermissionRules(path, source string, snap *PermissionSnapshot) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var settings struct {
		Permissions struct {
			Allow []struct {
				Tool    string `json:"tool"`
				Pattern string `json:"pattern"`
			} `json:"allow"`
			Deny []struct {
				Tool    string `json:"tool"`
				Pattern string `json:"pattern"`
			} `json:"deny"`
			Mode string `json:"mode"`
		} `json:"permissions"`
	}

	if json.Unmarshal(data, &settings) != nil {
		return
	}

	for _, a := range settings.Permissions.Allow {
		snap.AllowRules = append(snap.AllowRules, PermissionRule{
			Tool: a.Tool, Pattern: a.Pattern, Source: source,
		})
	}

	for _, d := range settings.Permissions.Deny {
		snap.DenyRules = append(snap.DenyRules, PermissionRule{
			Tool: d.Tool, Pattern: d.Pattern, Source: source,
		})
	}

	// Higher priority source overrides mode
	if settings.Permissions.Mode != "" {
		snap.Mode = settings.Permissions.Mode
	}
}
