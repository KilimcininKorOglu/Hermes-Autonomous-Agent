package scheduler

import (
	"fmt"
	"os/exec"
	"strings"
)

// Rollback provides rollback functionality for parallel execution
type Rollback struct {
	workDir    string
	snapshots  map[string]string // taskID -> commit hash before task
	baseBranch string
}

// NewRollback creates a new rollback manager
func NewRollback(workDir string) *Rollback {
	baseBranch, _ := getCurrentBranch(workDir)
	return &Rollback{
		workDir:    workDir,
		snapshots:  make(map[string]string),
		baseBranch: baseBranch,
	}
}

// SaveSnapshot saves the current state before a task
func (r *Rollback) SaveSnapshot(taskID string) error {
	commitHash, err := getCurrentCommit(r.workDir)
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}
	r.snapshots[taskID] = commitHash
	return nil
}

// RollbackTask reverts changes made by a specific task
func (r *Rollback) RollbackTask(taskID string) error {
	commitHash, ok := r.snapshots[taskID]
	if !ok {
		return fmt.Errorf("no snapshot found for task %s", taskID)
	}

	// Reset to the snapshot
	return runGitCommand(r.workDir, "reset", "--hard", commitHash)
}

// RollbackBatch reverts all tasks in a batch
func (r *Rollback) RollbackBatch(taskIDs []string) error {
	if len(taskIDs) == 0 {
		return nil
	}

	// Find earliest snapshot
	var earliestCommit string
	for _, taskID := range taskIDs {
		if commit, ok := r.snapshots[taskID]; ok {
			if earliestCommit == "" {
				earliestCommit = commit
			}
		}
	}

	if earliestCommit == "" {
		return fmt.Errorf("no snapshots found for batch")
	}

	return runGitCommand(r.workDir, "reset", "--hard", earliestCommit)
}

// RollbackAll reverts all changes to the initial state
func (r *Rollback) RollbackAll() error {
	// Get the earliest snapshot
	var earliestCommit string
	for _, commit := range r.snapshots {
		if earliestCommit == "" {
			earliestCommit = commit
		}
	}

	if earliestCommit == "" {
		return fmt.Errorf("no snapshots available")
	}

	return runGitCommand(r.workDir, "reset", "--hard", earliestCommit)
}

// CleanupTaskBranches removes all task branches
func (r *Rollback) CleanupTaskBranches() error {
	// List all task branches
	output, err := runGitCommandOutput(r.workDir, "branch", "--list", "task/*")
	if err != nil {
		return err
	}

	branches := strings.Split(strings.TrimSpace(output), "\n")
	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		branch = strings.TrimPrefix(branch, "* ")
		if branch != "" && strings.HasPrefix(branch, "task/") {
			runGitCommand(r.workDir, "branch", "-D", branch)
		}
	}

	return nil
}

// CleanupWorktrees removes all hermes worktrees
func (r *Rollback) CleanupWorktrees() error {
	// List worktrees
	output, err := runGitCommandOutput(r.workDir, "worktree", "list", "--porcelain")
	if err != nil {
		return err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") && strings.Contains(line, "wt-") {
			path := strings.TrimPrefix(line, "worktree ")
			runGitCommand(r.workDir, "worktree", "remove", path, "--force")
		}
	}

	// Prune
	runGitCommand(r.workDir, "worktree", "prune")

	return nil
}

// Cleanup performs full cleanup after parallel execution
func (r *Rollback) Cleanup() error {
	// First, checkout base branch
	if r.baseBranch != "" {
		runGitCommand(r.workDir, "checkout", r.baseBranch)
	}

	// Remove worktrees
	if err := r.CleanupWorktrees(); err != nil {
		return err
	}

	// Remove branches
	return r.CleanupTaskBranches()
}

// GetSnapshot returns the commit hash for a task snapshot
func (r *Rollback) GetSnapshot(taskID string) (string, bool) {
	commit, ok := r.snapshots[taskID]
	return commit, ok
}

// HasSnapshots returns true if there are any snapshots
func (r *Rollback) HasSnapshots() bool {
	return len(r.snapshots) > 0
}

// GetBaseBranch returns the base branch name
func (r *Rollback) GetBaseBranch() string {
	return r.baseBranch
}

// PrintStatus prints the rollback status
func (r *Rollback) PrintStatus() {
	fmt.Println("\n🔄 Rollback Status")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Base Branch: %s\n", r.baseBranch)
	fmt.Printf("Snapshots: %d\n", len(r.snapshots))
	
	if len(r.snapshots) > 0 {
		fmt.Println("\nTask Snapshots:")
		for taskID, commit := range r.snapshots {
			fmt.Printf("  %s: %s\n", taskID, commit[:8])
		}
	}
	fmt.Println("═══════════════════════════════════════")
}

// Helper functions

func getCurrentBranch(workDir string) (string, error) {
	return runGitCommandOutput(workDir, "rev-parse", "--abbrev-ref", "HEAD")
}

func getCurrentCommit(workDir string) (string, error) {
	return runGitCommandOutput(workDir, "rev-parse", "HEAD")
}

func runGitCommand(workDir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

func runGitCommandOutput(workDir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
