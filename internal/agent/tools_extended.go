package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/aniclew/aniclew/internal/types"
)

// ExtendedToolDefs returns additional tool definitions beyond the base 6.
func ExtendedToolDefs() []types.ToolDef {
	return []types.ToolDef{
		// ── Web Tools ──
		{
			Name:        "WebSearch",
			Description: "Search the web and return top results. Use for finding documentation, APIs, solutions.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Search query"}
				},
				"required": ["query"]
			}`),
		},
		{
			Name:        "WebFetch",
			Description: "Fetch a webpage URL and return its text content.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"url": {"type": "string", "description": "URL to fetch"},
					"prompt": {"type": "string", "description": "What to extract from the page"}
				},
				"required": ["url"]
			}`),
		},
		// ── Git Tools ──
		{
			Name:        "Git",
			Description: "Run git commands. Safe commands (status, diff, log, branch, show, blame) run directly. Mutating commands (add, commit, push, reset) require confirm:true.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": {"type": "string", "description": "Git subcommand: status, diff, log, branch, add, commit, push, show, blame, stash"},
					"args": {"type": "string", "description": "Additional arguments"},
					"confirm": {"type": "boolean", "description": "Required true for mutating commands"}
				},
				"required": ["command"]
			}`),
		},
		// ── Directory Listing ──
		{
			Name:        "LS",
			Description: "List directory contents with file sizes and types.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "Directory path (default: working directory)"}
				}
			}`),
		},
		// ── Task Management ──
		{
			Name:        "TaskCreate",
			Description: "Create a task to track work progress. Returns task ID.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"subject": {"type": "string", "description": "Brief task title"},
					"description": {"type": "string", "description": "What needs to be done"}
				},
				"required": ["subject"]
			}`),
		},
		{
			Name:        "TaskUpdate",
			Description: "Update a task's status: pending, in_progress, completed.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"id": {"type": "string", "description": "Task ID"},
					"status": {"type": "string", "description": "New status: pending, in_progress, completed"}
				},
				"required": ["id", "status"]
			}`),
		},
		{
			Name:        "TaskList",
			Description: "List all active tasks and their status.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		// ── Notebook ──
		{
			Name:        "NotebookRead",
			Description: "Read a Jupyter notebook (.ipynb) and display its cells.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {"type": "string", "description": "Path to .ipynb file"}
				},
				"required": ["file_path"]
			}`),
		},
		{
			Name:        "NotebookEdit",
			Description: "Edit a cell in a Jupyter notebook.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {"type": "string", "description": "Path to .ipynb file"},
					"cell_index": {"type": "integer", "description": "Cell index (0-based)"},
					"new_source": {"type": "string", "description": "New cell content"}
				},
				"required": ["file_path", "cell_index", "new_source"]
			}`),
		},
	}
}

// ExecuteExtendedTool handles the extended tools.
func ExecuteExtendedTool(name string, input json.RawMessage, workDir string) (string, bool, bool) {
	switch name {
	case "WebSearch":
		r, e := executeWebSearch(input)
		return r, e, true
	case "WebFetch":
		r, e := executeWebFetch(input)
		return r, e, true
	case "Git":
		r, e := executeGit(input, workDir)
		return r, e, true
	case "LS":
		r, e := executeLS(input, workDir)
		return r, e, true
	case "TaskCreate":
		r, e := executeTaskCreate(input)
		return r, e, true
	case "TaskUpdate":
		r, e := executeTaskUpdate(input)
		return r, e, true
	case "TaskList":
		r, e := executeTaskList()
		return r, e, true
	case "NotebookRead":
		r, e := executeNotebookRead(input, workDir)
		return r, e, true
	case "NotebookEdit":
		r, e := executeNotebookEdit(input, workDir)
		return r, e, true
	default:
		return "", false, false // not handled
	}
}

// ── Web Search ──

func executeWebSearch(input json.RawMessage) (string, bool) {
	var args struct{ Query string `json:"query"` }
	json.Unmarshal(input, &args)

	searchURL := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(args.Query)
	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "AniClew/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Search failed: %v", err), true
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Extract results from DuckDuckGo HTML
	var results []string
	re := regexp.MustCompile(`<a rel="nofollow" class="result__a" href="([^"]+)"[^>]*>(.+?)</a>`)
	matches := re.FindAllStringSubmatch(html, 5)
	for i, m := range matches {
		title := stripHTML(m[2])
		href := m[1]
		results = append(results, fmt.Sprintf("%d. %s\n   %s", i+1, title, href))
	}

	if len(results) == 0 {
		// Fallback: try snippet extraction
		re2 := regexp.MustCompile(`<a class="result__snippet"[^>]*>(.+?)</a>`)
		snippets := re2.FindAllStringSubmatch(html, 5)
		for i, s := range snippets {
			results = append(results, fmt.Sprintf("%d. %s", i+1, stripHTML(s[1])))
		}
	}

	if len(results) == 0 {
		return "No results found for: " + args.Query, false
	}
	return fmt.Sprintf("Search results for \"%s\":\n\n%s", args.Query, strings.Join(results, "\n\n")), false
}

// ── Web Fetch ──

func executeWebFetch(input json.RawMessage) (string, bool) {
	var args struct {
		URL    string `json:"url"`
		Prompt string `json:"prompt"`
	}
	json.Unmarshal(input, &args)

	req, _ := http.NewRequest("GET", args.URL, nil)
	req.Header.Set("User-Agent", "AniClew/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Fetch failed: %v", err), true
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	text := stripHTML(string(body))

	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	if len(text) > 10000 {
		text = text[:10000] + "\n... (truncated)"
	}

	result := fmt.Sprintf("Content from %s:\n\n%s", args.URL, text)
	if args.Prompt != "" {
		result = fmt.Sprintf("Prompt: %s\n\n%s", args.Prompt, result)
	}
	return result, false
}

// ── Git ──

var safeGitCommands = map[string]bool{
	"status": true, "diff": true, "log": true, "branch": true,
	"show": true, "blame": true, "stash": true, "remote": true, "tag": true,
}

func executeGit(input json.RawMessage, workDir string) (string, bool) {
	var args struct {
		Command string `json:"command"`
		Args    string `json:"args"`
		Confirm bool   `json:"confirm"`
	}
	json.Unmarshal(input, &args)

	if !safeGitCommands[args.Command] && !args.Confirm {
		return fmt.Sprintf("Git '%s' is a mutating command. Set confirm:true to execute.", args.Command), true
	}

	cmdArgs := []string{args.Command}
	if args.Args != "" {
		cmdArgs = append(cmdArgs, strings.Fields(args.Args)...)
	}

	// Safety: block force push and destructive resets
	fullCmd := args.Command + " " + args.Args
	if strings.Contains(fullCmd, "--force") || strings.Contains(fullCmd, "reset --hard") {
		return "Blocked: force push and hard reset are not allowed for safety.", true
	}

	cmd := exec.Command("git", cmdArgs...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	result := string(out)
	if err != nil {
		result += "\n[git error: " + err.Error() + "]"
	}
	if len(result) > 30000 {
		result = result[:30000] + "\n... (truncated)"
	}
	return result, err != nil
}

// ── LS ──

func executeLS(input json.RawMessage, workDir string) (string, bool) {
	var args struct{ Path string `json:"path"` }
	json.Unmarshal(input, &args)

	dir := workDir
	if args.Path != "" {
		dir = resolvePath(args.Path, workDir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Sprintf("Error listing %s: %v", dir, err), true
	}

	var lines []string
	for _, e := range entries {
		info, _ := e.Info()
		typeStr := "file"
		size := ""
		if e.IsDir() {
			typeStr = "dir "
		} else if info != nil {
			size = formatSize(info.Size())
		}
		lines = append(lines, fmt.Sprintf("  %s %8s  %s", typeStr, size, e.Name()))
	}
	return fmt.Sprintf("%s (%d items):\n%s", dir, len(entries), strings.Join(lines, "\n")), false
}

// ── Task Management (in-memory) ──

type taskItem struct {
	ID          string `json:"id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

var taskStore = make(map[string]*taskItem)
var taskCounter = 0

func executeTaskCreate(input json.RawMessage) (string, bool) {
	var args struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
	}
	json.Unmarshal(input, &args)

	taskCounter++
	id := fmt.Sprintf("task-%d", taskCounter)
	taskStore[id] = &taskItem{
		ID: id, Subject: args.Subject, Description: args.Description, Status: "pending",
	}
	return fmt.Sprintf("Task %s created: %s", id, args.Subject), false
}

func executeTaskUpdate(input json.RawMessage) (string, bool) {
	var args struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	json.Unmarshal(input, &args)

	task, ok := taskStore[args.ID]
	if !ok {
		return fmt.Sprintf("Task %s not found", args.ID), true
	}
	task.Status = args.Status
	return fmt.Sprintf("Task %s updated to: %s", args.ID, args.Status), false
}

func executeTaskList() (string, bool) {
	if len(taskStore) == 0 {
		return "No tasks.", false
	}
	var lines []string
	for _, t := range taskStore {
		icon := "⬜"
		switch t.Status {
		case "in_progress":
			icon = "🔄"
		case "completed":
			icon = "✅"
		}
		lines = append(lines, fmt.Sprintf("%s %s [%s] %s", icon, t.ID, t.Status, t.Subject))
	}
	return strings.Join(lines, "\n"), false
}

// ── Notebook (.ipynb) ──

type notebookFile struct {
	Cells    []notebookCell         `json:"cells"`
	Metadata map[string]interface{} `json:"metadata"`
	Nbformat int                    `json:"nbformat"`
}

type notebookCell struct {
	CellType string        `json:"cell_type"`
	Source   []string      `json:"source"`
	Outputs  []interface{} `json:"outputs,omitempty"`
}

func executeNotebookRead(input json.RawMessage, workDir string) (string, bool) {
	var args struct{ FilePath string `json:"file_path"` }
	json.Unmarshal(input, &args)

	path := resolvePath(args.FilePath, workDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error reading notebook: %v", err), true
	}

	var nb notebookFile
	if err := json.Unmarshal(data, &nb); err != nil {
		return fmt.Sprintf("Error parsing notebook: %v", err), true
	}

	var lines []string
	for i, cell := range nb.Cells {
		source := strings.Join(cell.Source, "")
		lines = append(lines, fmt.Sprintf("--- Cell %d [%s] ---\n%s", i, cell.CellType, source))
	}
	return fmt.Sprintf("Notebook: %s (%d cells)\n\n%s", args.FilePath, len(nb.Cells), strings.Join(lines, "\n\n")), false
}

func executeNotebookEdit(input json.RawMessage, workDir string) (string, bool) {
	var args struct {
		FilePath  string `json:"file_path"`
		CellIndex int    `json:"cell_index"`
		NewSource string `json:"new_source"`
	}
	json.Unmarshal(input, &args)

	path := resolvePath(args.FilePath, workDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error reading notebook: %v", err), true
	}

	var nb notebookFile
	if err := json.Unmarshal(data, &nb); err != nil {
		return fmt.Sprintf("Error parsing notebook: %v", err), true
	}

	if args.CellIndex < 0 || args.CellIndex >= len(nb.Cells) {
		return fmt.Sprintf("Cell index %d out of range (0-%d)", args.CellIndex, len(nb.Cells)-1), true
	}

	// Split source into lines (ipynb stores source as array of lines)
	lines := strings.Split(args.NewSource, "\n")
	for i := range lines {
		if i < len(lines)-1 {
			lines[i] += "\n"
		}
	}
	nb.Cells[args.CellIndex].Source = lines

	out, _ := json.MarshalIndent(nb, "", " ")
	os.WriteFile(path, out, 0644)
	return fmt.Sprintf("Cell %d updated in %s", args.CellIndex, args.FilePath), false
}

// ── Helpers ──

func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, "")
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/1024/1024)
}
