package agent

// AgentType defines a specialized agent role.
type AgentType struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	SystemPrompt string  `json:"systemPrompt"`
	Tools       []string `json:"tools"`       // allowed tools (empty = all)
	ReadOnly    bool     `json:"readOnly"`    // if true, no write tools
	Model       string   `json:"model"`       // override model (empty = inherit)
}

// BuiltinAgentTypes returns the predefined agent types.
func BuiltinAgentTypes() map[string]AgentType {
	return map[string]AgentType{
		"explorer": {
			Name:        "explorer",
			Description: "Fast codebase exploration — finds files, reads code, searches patterns",
			SystemPrompt: `You are an Explorer agent. Your job is to quickly find information in the codebase.
Use Glob to find files, Grep to search content, Read to read files.
Do NOT modify any files. Report findings concisely.`,
			Tools:    []string{"Read", "Glob", "Grep", "Bash", "LS"},
			ReadOnly: true,
		},
		"researcher": {
			Name:        "researcher",
			Description: "Deep research — analyzes architecture, traces data flow, understands patterns",
			SystemPrompt: `You are a Researcher agent. Deeply analyze code architecture, trace data flows, and understand design patterns.
Read multiple files to understand relationships. Report with file paths and line numbers.
Do NOT modify any files.`,
			Tools:    []string{"Read", "Glob", "Grep", "Bash", "LS"},
			ReadOnly: true,
		},
		"planner": {
			Name:        "planner",
			Description: "Implementation planning — designs approach, identifies files to modify, creates plan",
			SystemPrompt: `You are a Planner agent. Design implementation plans.
1. Analyze the current codebase structure
2. Identify which files need to be created or modified
3. Define step-by-step implementation plan
4. Note potential risks and dependencies
Do NOT modify any files. Output a structured plan.`,
			Tools:    []string{"Read", "Glob", "Grep", "Bash", "LS"},
			ReadOnly: true,
		},
		"coder": {
			Name:        "coder",
			Description: "Implementation — writes and edits code, runs tests",
			SystemPrompt: `You are a Coder agent. Implement changes precisely.
- Read files before editing
- Make minimal, focused changes
- Run tests after changes when possible
- Follow existing code patterns`,
			Tools: []string{}, // all tools
		},
		"reviewer": {
			Name:        "reviewer",
			Description: "Code review — checks quality, security, and correctness",
			SystemPrompt: `You are a Code Reviewer agent. Review code for:
1. Correctness — does it do what it should?
2. Security — any vulnerabilities? (OWASP top 10)
3. Performance — any bottlenecks?
4. Style — follows project conventions?
5. Tests — adequate coverage?
Report issues with file paths and line numbers. Suggest fixes.
Do NOT modify any files.`,
			Tools:    []string{"Read", "Glob", "Grep", "Bash"},
			ReadOnly: true,
		},
		"tester": {
			Name:        "tester",
			Description: "Test execution — runs tests, analyzes failures, writes test cases",
			SystemPrompt: `You are a Tester agent. Run tests and analyze results.
- Execute test suites
- Analyze failures and identify root causes
- Write new test cases for untested code
- Report coverage gaps`,
			Tools: []string{}, // all tools
		},
	}
}

// GetAgentType returns an agent type by name, or nil if not found.
func GetAgentType(name string) *AgentType {
	types := BuiltinAgentTypes()
	if t, ok := types[name]; ok {
		return &t
	}
	return nil
}

// LoadCustomAgentTypes reads agent definitions from project .claude/agents/ directory.
func LoadCustomAgentTypes(workDir string) map[string]AgentType {
	// TODO: parse .claude/agents/*.md files for custom agent definitions
	// Each file defines: name, description, system prompt, tools, model
	return nil
}
