package kairos

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)


// GitStatus holds the current git state of a project.
type GitStatus struct {
	Branch       string   `json:"branch"`
	Staged       []string `json:"staged"`
	Modified     []string `json:"modified"`
	Untracked    []string `json:"untracked"`
	AheadBy      int      `json:"aheadBy"`
	LastCommit   string   `json:"lastCommit"`
	LastCommitAt string   `json:"lastCommitAt"`
}

// CheckGitStatus runs git commands in the given directory and returns status.
func CheckGitStatus(workDir string) (*GitStatus, error) {
	if workDir == "" {
		return nil, fmt.Errorf("no workDir")
	}

	// Check if it's a git repo
	if err := gitCmd(workDir, "rev-parse", "--git-dir"); err != nil {
		return nil, fmt.Errorf("not a git repo")
	}

	status := &GitStatus{}

	// Branch name
	if out, err := gitOutput(workDir, "branch", "--show-current"); err == nil {
		status.Branch = strings.TrimSpace(out)
	}

	// Porcelain status
	if out, err := gitOutput(workDir, "status", "--porcelain"); err == nil {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			xy := line[:2]
			file := strings.TrimSpace(line[2:])
			// Rename arrows
			if idx := strings.Index(file, " -> "); idx >= 0 {
				file = file[idx+4:]
			}

			if xy[0] != ' ' && xy[0] != '?' {
				status.Staged = append(status.Staged, file)
			}
			if xy[1] == 'M' || xy[1] == 'D' {
				status.Modified = append(status.Modified, file)
			}
			if xy == "??" {
				status.Untracked = append(status.Untracked, file)
			}
		}
	}

	// Ahead count
	if out, err := gitOutput(workDir, "rev-list", "--count", "@{u}..HEAD"); err == nil {
		fmt.Sscanf(strings.TrimSpace(out), "%d", &status.AheadBy)
	}

	// Last commit
	if out, err := gitOutput(workDir, "log", "-1", "--format=%s|||%ci"); err == nil {
		parts := strings.SplitN(strings.TrimSpace(out), "|||", 2)
		if len(parts) == 2 {
			status.LastCommit = parts[0]
			status.LastCommitAt = parts[1]
		}
	}

	return status, nil
}

// GitWatchSummary creates a human-readable summary of git changes.
func GitWatchSummary(prev, curr *GitStatus) string {
	if curr == nil {
		return "Not a git repository"
	}

	var parts []string

	total := len(curr.Staged) + len(curr.Modified) + len(curr.Untracked)
	if total == 0 {
		return fmt.Sprintf("[%s] Clean working tree. Last: %s", curr.Branch, curr.LastCommit)
	}

	parts = append(parts, fmt.Sprintf("[%s]", curr.Branch))

	if len(curr.Staged) > 0 {
		parts = append(parts, fmt.Sprintf("staged:%d", len(curr.Staged)))
	}
	if len(curr.Modified) > 0 {
		parts = append(parts, fmt.Sprintf("modified:%d(%s)", len(curr.Modified), joinMax(curr.Modified, 3)))
	}
	if len(curr.Untracked) > 0 {
		parts = append(parts, fmt.Sprintf("untracked:%d", len(curr.Untracked)))
	}
	if curr.AheadBy > 0 {
		parts = append(parts, fmt.Sprintf("ahead:%d (unpushed)", curr.AheadBy))
	}

	// Detect new changes since previous check
	if prev != nil {
		newMod := diffSlice(curr.Modified, prev.Modified)
		if len(newMod) > 0 {
			parts = append(parts, fmt.Sprintf("NEW: %s", joinMax(newMod, 5)))
		}
	}

	return strings.Join(parts, " ")
}

// ── Helpers ──

func gitCmd(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = filepath.FromSlash(dir)
	return cmd.Run()
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = filepath.FromSlash(dir)
	out, err := cmd.Output()
	return string(out), err
}

func joinMax(items []string, max int) string {
	if len(items) <= max {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:max], ", ") + fmt.Sprintf(" +%d", len(items)-max)
}

func diffSlice(curr, prev []string) []string {
	prevSet := make(map[string]bool, len(prev))
	for _, p := range prev {
		prevSet[p] = true
	}
	var diff []string
	for _, c := range curr {
		if !prevSet[c] {
			diff = append(diff, c)
		}
	}
	return diff
}

// RunGitWatch is a built-in task that checks git status and logs changes.
func (d *Daemon) RunGitWatch() {
	d.mu.RLock()
	workDir := d.workDir
	d.mu.RUnlock()

	curr, err := CheckGitStatus(workDir)
	if err != nil {
		d.addLog("git-watch", err.Error())
		return
	}

	prev := d.lastGitStatus
	summary := GitWatchSummary(prev, curr)
	d.mu.Lock()
	d.lastGitStatus = curr
	d.mu.Unlock()

	d.addLog("git-watch", summary)

	// Notify if new changes detected
	totalPrev := 0
	if prev != nil {
		totalPrev = len(prev.Staged) + len(prev.Modified) + len(prev.Untracked)
	}
	totalCurr := len(curr.Staged) + len(curr.Modified) + len(curr.Untracked)
	if totalCurr > totalPrev && d.notifier != nil {
		d.notifier.Send(Notification{
			Type:    "git-change",
			Title:   "Git changes detected",
			Message: summary,
			Project: filepath.Base(workDir),
		})
	}
}

// LastGitStatus returns the most recent git status.
func (d *Daemon) LastGitStatus() *GitStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.lastGitStatus
}

// AutoGitWatch creates a default git-watch task.
func AutoGitWatchTask() Task {
	return Task{
		ID:          "git-watch",
		Type:        "git-watch",
		Description: "Monitor git changes",
		Enabled:     true,
		CreatedAt:   time.Now(),
	}
}
