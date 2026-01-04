package isolation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hermes/internal/git"
)

// Workspace represents an isolated workspace for a task
type Workspace struct {
	TaskID   string
	TaskName string
	BasePath string // Original repository path
	WorkPath string // Isolated workspace path (git worktree)
	Branch   string
}

// NewWorkspace creates a new workspace configuration
func NewWorkspace(taskID, basePath string) *Workspace {
	branchName := fmt.Sprintf("task/%s", taskID)
	// Create worktree in project directory instead of temp
	workPath := filepath.Join(basePath, ".hermes", "worktrees", fmt.Sprintf("wt-%s", taskID))

	return &Workspace{
		TaskID:   taskID,
		BasePath: basePath,
		WorkPath: workPath,
		Branch:   branchName,
	}
}

// NewWorkspaceWithName creates a new workspace configuration with task name
func NewWorkspaceWithName(taskID, taskName, basePath string) *Workspace {
	var branchName string
	if taskName != "" {
		branchName = git.GetTaskBranchName(taskID, taskName)
	} else {
		branchName = fmt.Sprintf("task/%s", taskID)
	}
	// Create worktree in project directory instead of temp
	workPath := filepath.Join(basePath, ".hermes", "worktrees", fmt.Sprintf("wt-%s", taskID))

	return &Workspace{
		TaskID:   taskID,
		TaskName: taskName,
		BasePath: basePath,
		WorkPath: workPath,
		Branch:   branchName,
	}
}

// Setup creates the isolated workspace using git worktree
func (w *Workspace) Setup() error {
	// Check if worktree already exists
	if _, err := os.Stat(w.WorkPath); err == nil {
		// Remove existing worktree
		if err := w.Cleanup(); err != nil {
			return fmt.Errorf("failed to cleanup existing worktree: %w", err)
		}
	}

	// Get current branch to use as base
	baseBranch, err := w.getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Create new branch for the task
	if err := w.createBranch(baseBranch); err != nil {
		// Branch might already exist, try to continue
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create branch: %w", err)
		}
	}

	// Create worktree
	cmd := exec.Command("git", "worktree", "add", w.WorkPath, w.Branch)
	cmd.Dir = w.BasePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w: %s", err, string(output))
	}

	return nil
}

// SetupShared creates a workspace using the shared repository (no isolation)
// This is faster but doesn't provide isolation
func (w *Workspace) SetupShared() error {
	w.WorkPath = w.BasePath
	return nil
}

// Cleanup removes the isolated workspace
func (w *Workspace) Cleanup() error {
	// Remove worktree
	cmd := exec.Command("git", "worktree", "remove", w.WorkPath, "--force")
	cmd.Dir = w.BasePath
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try manual removal if git worktree remove fails
		if err := os.RemoveAll(w.WorkPath); err != nil {
			return fmt.Errorf("failed to remove worktree: %s", string(output))
		}
	}

	// Prune worktrees
	cmd = exec.Command("git", "worktree", "prune")
	cmd.Dir = w.BasePath
	cmd.Run() // Ignore errors

	return nil
}

// CleanupBranch removes the task branch
func (w *Workspace) CleanupBranch() error {
	cmd := exec.Command("git", "branch", "-D", w.Branch)
	cmd.Dir = w.BasePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete branch: %w: %s", err, string(output))
	}
	return nil
}

// GetChanges returns the files changed in this workspace
func (w *Workspace) GetChanges() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = w.WorkPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get changes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// GetDiff returns the git diff for changes in this workspace
func (w *Workspace) GetDiff() (string, error) {
	cmd := exec.Command("git", "diff", "HEAD")
	cmd.Dir = w.WorkPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}
	return string(output), nil
}

// CommitChanges commits all changes in the workspace
func (w *Workspace) CommitChanges(message string) error {
	// Stage all changes
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = w.WorkPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage changes: %w: %s", err, string(output))
	}

	// Check if there are changes to commit
	cmd = exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = w.WorkPath
	if err := cmd.Run(); err == nil {
		// No changes to commit
		return nil
	}

	// Commit
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = w.WorkPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit: %w: %s", err, string(output))
	}

	return nil
}

// PushChanges pushes changes to remote
func (w *Workspace) PushChanges() error {
	cmd := exec.Command("git", "push", "-u", "origin", w.Branch)
	cmd.Dir = w.WorkPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push: %w: %s", err, string(output))
	}
	return nil
}

// HasUncommittedChanges returns true if there are uncommitted changes
func (w *Workspace) HasUncommittedChanges() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = w.WorkPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// getCurrentBranch returns the current branch name
func (w *Workspace) getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = w.BasePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// createBranch creates a new branch from the current HEAD
func (w *Workspace) createBranch(baseBranch string) error {
	cmd := exec.Command("git", "branch", w.Branch, baseBranch)
	cmd.Dir = w.BasePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// GetBranch returns the branch name for this workspace
func (w *Workspace) GetBranch() string {
	return w.Branch
}

// GetWorkPath returns the workspace path
func (w *Workspace) GetWorkPath() string {
	return w.WorkPath
}

// IsIsolated returns true if this workspace is isolated (using worktree)
func (w *Workspace) IsIsolated() bool {
	return w.WorkPath != w.BasePath
}
