package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Plan represents a structured implementation plan.
type Plan struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Steps     []PlanStep `json:"steps"`
	Status    string     `json:"status"` // "draft", "approved", "executing", "completed"
	CreatedAt time.Time  `json:"createdAt"`
}

type PlanStep struct {
	Index       int    `json:"index"`
	Description string `json:"description"`
	Files       string `json:"files"`
	Status      string `json:"status"` // "pending", "in_progress", "completed"
}

var activePlan *Plan

// PlanMode tool definitions
func PlanToolDefs() []PlanToolDef {
	return []PlanToolDef{
		{Name: "EnterPlanMode", Desc: "Start planning before implementation. Creates a structured plan for user approval."},
		{Name: "ExitPlanMode", Desc: "Submit the plan for approval and begin implementation."},
	}
}

type PlanToolDef struct {
	Name string
	Desc string
}

// IsPlanMode returns true if a plan is active and not yet approved.
func IsPlanMode() bool {
	return activePlan != nil && activePlan.Status == "draft"
}

// CreatePlan starts a new plan.
func CreatePlan(title string) *Plan {
	activePlan = &Plan{
		ID:        fmt.Sprintf("plan_%d", time.Now().Unix()),
		Title:     title,
		Status:    "draft",
		CreatedAt: time.Now(),
	}
	return activePlan
}

// ApprovePlan marks the plan as approved for execution.
func ApprovePlan() string {
	if activePlan == nil {
		return "No active plan."
	}
	activePlan.Status = "approved"
	return fmt.Sprintf("Plan '%s' approved with %d steps. Implementation can begin.", activePlan.Title, len(activePlan.Steps))
}

// GetActivePlan returns the current plan.
func GetActivePlan() *Plan {
	return activePlan
}

// ── Context Compression ──

// CompressContext summarizes a conversation to reduce token usage.
func CompressContext(messages []map[string]string) string {
	if len(messages) < 6 {
		return "" // no need to compress short conversations
	}

	var summary strings.Builder
	summary.WriteString("## Conversation Summary\n\n")

	toolsUsed := make(map[string]int)
	filesModified := []string{}
	keyDecisions := []string{}

	for _, m := range messages {
		role := m["role"]
		content := m["content"]

		if role == "tool" {
			name := m["toolName"]
			toolsUsed[name]++
			if name == "Write" || name == "Edit" {
				var args struct{ FilePath string `json:"file_path"` }
				json.Unmarshal([]byte(content), &args)
				if args.FilePath != "" {
					filesModified = append(filesModified, args.FilePath)
				}
			}
		}

		// Extract key decisions from assistant messages
		if role == "assistant" && len(content) > 100 {
			// Take first sentence as key point
			sentence := content
			if idx := strings.Index(content, ". "); idx > 0 && idx < 200 {
				sentence = content[:idx+1]
			}
			if len(sentence) > 200 {
				sentence = sentence[:200] + "..."
			}
			keyDecisions = append(keyDecisions, sentence)
		}
	}

	// Tools summary
	if len(toolsUsed) > 0 {
		summary.WriteString("**Tools used**: ")
		for name, count := range toolsUsed {
			summary.WriteString(fmt.Sprintf("%s(%d) ", name, count))
		}
		summary.WriteString("\n")
	}

	// Files modified
	if len(filesModified) > 0 {
		unique := uniqueStrings(filesModified)
		summary.WriteString("**Files modified**: " + strings.Join(unique, ", ") + "\n")
	}

	// Key decisions (last 5)
	if len(keyDecisions) > 5 {
		keyDecisions = keyDecisions[len(keyDecisions)-5:]
	}
	if len(keyDecisions) > 0 {
		summary.WriteString("\n**Key points**:\n")
		for _, d := range keyDecisions {
			summary.WriteString("- " + d + "\n")
		}
	}

	return summary.String()
}

// EstimateTokens roughly estimates token count from text.
func EstimateTokens(text string) int {
	return len(text) / 4
}

func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
