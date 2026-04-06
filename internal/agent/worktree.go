package agent

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree manages git worktree isolation for parallel agents.
type Worktree struct {
	Name     string `json:"name"`
	Path     string `json:"path"`     // absolute path to worktree
	Branch   string `json:"branch"`   // branch name
	ParentDir string `json:"parentDir"` // main repo dir
}

// CreateWorktree creates an isolated git worktree for an agent.
func CreateWorktree(repoDir string, name string) (*Worktree, error) {
	// Verify it's a git repo
	if err := runGit(repoDir, "rev-parse", "--git-dir"); err != nil {
		return nil, fmt.Errorf("not a git repo: %s", repoDir)
	}

	// Create worktree directory under .claude/.worktrees/
	worktreeBase := filepath.Join(repoDir, ".claude", ".worktrees")
	os.MkdirAll(worktreeBase, 0755)

	safeName := sanitizeName(name)
	worktreePath := filepath.Join(worktreeBase, safeName)
	branchName := fmt.Sprintf("aniclew/%s", safeName)

	// Remove existing worktree if any
	if _, err := os.Stat(worktreePath); err == nil {
		RemoveWorktree(repoDir, worktreePath)
	}

	// Create new branch from HEAD
	runGit(repoDir, "branch", "-D", branchName) // ignore error if doesn't exist
	if err := runGit(repoDir, "worktree", "add", "-b", branchName, worktreePath); err != nil {
		return nil, fmt.Errorf("git worktree add failed: %w", err)
	}

	log.Printf("[Worktree] Created: %s at %s (branch: %s)", name, worktreePath, branchName)

	return &Worktree{
		Name:      name,
		Path:      worktreePath,
		Branch:    branchName,
		ParentDir: repoDir,
	}, nil
}

// HasChanges checks if the worktree has uncommitted changes.
func (w *Worktree) HasChanges() bool {
	out, err := runGitOutput(w.Path, "status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// GetDiff returns the diff of changes in the worktree.
func (w *Worktree) GetDiff() string {
	out, _ := runGitOutput(w.Path, "diff", "--stat")
	return out
}

// RemoveWorktree removes a git worktree safely.
func RemoveWorktree(repoDir string, worktreePath string) error {
	// Try git worktree remove first
	if err := runGit(repoDir, "worktree", "remove", "--force", worktreePath); err != nil {
		// Fallback: manual cleanup
		log.Printf("[Worktree] git remove failed, cleaning manually: %v", err)
		os.RemoveAll(worktreePath)
	}

	// Prune stale worktree entries
	runGit(repoDir, "worktree", "prune")

	log.Printf("[Worktree] Removed: %s", worktreePath)
	return nil
}

// ListWorktrees returns all worktrees for a repo.
func ListWorktrees(repoDir string) []Worktree {
	out, err := runGitOutput(repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil
	}

	var worktrees []Worktree
	var current Worktree

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = Worktree{}
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			current.Path = line[9:]
			current.ParentDir = repoDir
		}
		if strings.HasPrefix(line, "branch ") {
			current.Branch = line[7:]
			current.Name = filepath.Base(current.Path)
		}
	}
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees
}

// WorktreeNotice generates the context notice for a child agent in a worktree.
func WorktreeNotice(parentDir, worktreePath string) string {
	return fmt.Sprintf(`You are working in an isolated git worktree.
- Parent repo: %s
- Your worktree: %s
- Your changes are isolated and won't affect the parent's files.
- Re-read files before editing — they may differ from the parent context.
- When done, your changes can be reviewed and merged separately.`, parentDir, worktreePath)
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, name)
	return strings.Trim(name, "-")
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runGitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}
