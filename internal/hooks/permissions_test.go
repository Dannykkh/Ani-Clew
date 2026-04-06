package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCapturePermissions_Defaults(t *testing.T) {
	dir := t.TempDir()
	snap := CapturePermissions(dir)

	// Default mode
	if snap.Mode != "default" {
		t.Errorf("Expected mode 'default', got %q", snap.Mode)
	}

	// Default allow rules: Read, Glob, Grep
	readAllowed := snap.Decide("Read", "")
	if readAllowed != "allow" {
		t.Errorf("Read should be allowed by default, got %q", readAllowed)
	}

	// Unknown tool should be "ask"
	bashDecision := snap.Decide("Bash", "ls")
	if bashDecision != "ask" {
		t.Errorf("Bash should be 'ask' by default, got %q", bashDecision)
	}
}

func TestCapturePermissions_ProjectRules(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	settings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []map[string]string{
				{"tool": "Bash", "pattern": "npm test"},
			},
			"deny": []map[string]string{
				{"tool": "Bash", "pattern": "rm -rf"},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	snap := CapturePermissions(dir)

	// npm test should be allowed
	if snap.Decide("Bash", "npm test") != "allow" {
		t.Error("Bash 'npm test' should be allowed")
	}

	// rm -rf should be denied
	if snap.Decide("Bash", "rm -rf /") != "deny" {
		t.Error("Bash 'rm -rf /' should be denied")
	}
}

func TestCapturePermissions_ModeOverride(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	settings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"mode": "bypassPermissions",
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	snap := CapturePermissions(dir)

	if snap.Mode != "bypassPermissions" {
		t.Errorf("Expected mode 'bypassPermissions', got %q", snap.Mode)
	}

	// Everything should be allowed in bypass mode
	if snap.Decide("Bash", "rm -rf /") != "allow" {
		t.Error("bypassPermissions should allow everything")
	}
}

func TestPersistAllowRule(t *testing.T) {
	dir := t.TempDir()

	// Persist a rule
	err := PersistAllowRule(dir, "Bash", "npm test")
	if err != nil {
		t.Fatalf("PersistAllowRule failed: %v", err)
	}

	// Read the file and verify
	data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("Failed to read settings: %v", err)
	}

	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	perms := settings["permissions"].(map[string]interface{})
	allowList := perms["allow"].([]interface{})
	if len(allowList) != 1 {
		t.Fatalf("Expected 1 allow rule, got %d", len(allowList))
	}

	rule := allowList[0].(map[string]interface{})
	if rule["tool"] != "Bash" || rule["pattern"] != "npm test" {
		t.Errorf("Rule mismatch: %v", rule)
	}

	// Now capture and verify it's loaded
	snap := CapturePermissions(dir)
	if snap.Decide("Bash", "npm test") != "allow" {
		t.Error("Persisted allow rule should work in next capture")
	}
}

func TestPersistDenyRule(t *testing.T) {
	dir := t.TempDir()

	err := PersistDenyRule(dir, "Bash", "sudo")
	if err != nil {
		t.Fatalf("PersistDenyRule failed: %v", err)
	}

	snap := CapturePermissions(dir)
	if snap.Decide("Bash", "sudo rm") != "deny" {
		t.Error("Persisted deny rule should block 'sudo rm'")
	}
}

func TestDenyWinsOverAllow(t *testing.T) {
	dir := t.TempDir()

	// Add both allow and deny for same tool
	PersistAllowRule(dir, "Bash", "npm")
	PersistDenyRule(dir, "Bash", "npm")

	snap := CapturePermissions(dir)
	// Deny should win (checked first)
	if snap.Decide("Bash", "npm install") != "deny" {
		t.Error("Deny should win over allow")
	}
}
