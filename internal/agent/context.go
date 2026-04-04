package agent

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadProjectContext reads CLAUDE.md, AGENTS.md, and .claude/ config from a workspace.
// Returns additional system prompt text to inject.
func LoadProjectContext(workDir string) string {
	var parts []string

	// ── 1. CLAUDE.md (project instructions) ──
	for _, name := range []string{"CLAUDE.md", "claude.md"} {
		content := readFileIfExists(filepath.Join(workDir, name))
		if content != "" {
			parts = append(parts, "## Project Instructions (CLAUDE.md)\n"+content)
			break
		}
	}

	// ── 2. AGENTS.md ──
	for _, name := range []string{"AGENTS.md", "agents.md"} {
		content := readFileIfExists(filepath.Join(workDir, name))
		if content != "" {
			parts = append(parts, "## Agent Instructions (AGENTS.md)\n"+content)
			break
		}
	}

	// ── 3. .claude/CLAUDE.md (user-level global instructions) ──
	home, _ := os.UserHomeDir()
	if home != "" {
		content := readFileIfExists(filepath.Join(home, ".claude", "CLAUDE.md"))
		if content != "" {
			parts = append(parts, "## Global Instructions (~/.claude/CLAUDE.md)\n"+content)
		}
	}

	// ── 4. .claude/settings.json (project-level settings) ──
	settings := readFileIfExists(filepath.Join(workDir, ".claude", "settings.json"))
	if settings != "" {
		parts = append(parts, "## Project Settings\n```json\n"+settings+"\n```")
	}

	// ── 5. README.md summary (first 50 lines for project context) ──
	for _, name := range []string{"README.md", "readme.md"} {
		content := readFileIfExists(filepath.Join(workDir, name))
		if content != "" {
			lines := strings.Split(content, "\n")
			if len(lines) > 50 {
				lines = lines[:50]
			}
			parts = append(parts, "## Project README (summary)\n"+strings.Join(lines, "\n"))
			break
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "\n\n--- PROJECT CONTEXT ---\n" + strings.Join(parts, "\n\n")
}

// LoadSkills reads .claude/skills/ directory and returns skill contents.
func LoadSkills(workDir string) []SkillInfo {
	var skills []SkillInfo

	// Project-level skills
	skillDir := filepath.Join(workDir, ".claude", "skills")
	skills = append(skills, loadSkillsFromDir(skillDir)...)

	// User-level skills
	home, _ := os.UserHomeDir()
	if home != "" {
		userSkillDir := filepath.Join(home, ".claude", "skills")
		skills = append(skills, loadSkillsFromDir(userSkillDir)...)
	}

	return skills
}

type SkillInfo struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Path    string `json:"path"`
}

func loadSkillsFromDir(dir string) []SkillInfo {
	var skills []SkillInfo

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check for SKILL.md inside the directory
			skillFile := filepath.Join(dir, entry.Name(), "SKILL.md")
			content := readFileIfExists(skillFile)
			if content != "" {
				skills = append(skills, SkillInfo{
					Name:    entry.Name(),
					Content: content,
					Path:    skillFile,
				})
			}
		} else if strings.HasSuffix(entry.Name(), ".md") {
			// Direct .md file as skill
			content := readFileIfExists(filepath.Join(dir, entry.Name()))
			if content != "" {
				name := strings.TrimSuffix(entry.Name(), ".md")
				skills = append(skills, SkillInfo{
					Name:    name,
					Content: content,
					Path:    filepath.Join(dir, entry.Name()),
				})
			}
		}
	}

	return skills
}

// LoadMCPConfig reads .mcp.json from the workspace.
func LoadMCPConfig(workDir string) string {
	for _, name := range []string{".mcp.json", "mcp.json"} {
		content := readFileIfExists(filepath.Join(workDir, name))
		if content != "" {
			return content
		}
	}
	return ""
}

func readFileIfExists(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(data))
	// Limit size to prevent huge files from flooding the prompt
	if len(content) > 20000 {
		content = content[:20000] + "\n... (truncated)"
	}
	return content
}
