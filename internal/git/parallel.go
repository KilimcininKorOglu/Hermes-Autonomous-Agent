package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ParallelBranchManager manages branches for parallel task execution
type ParallelBranchManager struct {
	git        *Git
	baseBranch string
	branches   map[string]string // taskID -> branchName
	worktrees  map[string]string // taskID -> worktree path
}

// NewParallelBranchManager creates a new parallel branch manager
func NewParallelBranchManager(git *Git) *ParallelBranchManager {
	baseBranch, _ := git.GetCurrentBranch()
	return &ParallelBranchManager{
		git:        git,
		baseBranch: baseBranch,
		branches:   make(map[string]string),
		worktrees:  make(map[string]string),
	}
}

// GetBaseBranch returns the base branch name
func (m *ParallelBranchManager) GetBaseBranch() string {
	return m.baseBranch
}

// CreateTaskBranch creates a new branch for a task
func (m *ParallelBranchManager) CreateTaskBranch(taskID string, taskName ...string) (string, error) {
	var branchName string
	if len(taskName) > 0 && taskName[0] != "" {
		branchName = GetTaskBranchName(taskID, taskName[0])
	} else {
		branchName = fmt.Sprintf("task/%s", taskID)
	}

	// Check if branch already exists
	if m.git.BranchExists(branchName) {
		m.branches[taskID] = branchName
		return branchName, nil
	}

	// Create branch from base
	_, err := m.git.run("branch", branchName, m.baseBranch)
	if err != nil {
		return "", fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	m.branches[taskID] = branchName
	return branchName, nil
}

// CreateWorktree creates a git worktree for a task
func (m *ParallelBranchManager) CreateWorktree(taskID string) (string, error) {
	// Ensure branch exists
	branchName, ok := m.branches[taskID]
	if !ok {
		var err error
		branchName, err = m.CreateTaskBranch(taskID)
		if err != nil {
			return "", err
		}
	}

	// Create worktree path
	worktreePath := filepath.Join(os.TempDir(), fmt.Sprintf("wt-%s", taskID))

	// Remove existing worktree if present
	if _, err := os.Stat(worktreePath); err == nil {
		m.RemoveWorktree(taskID)
	}

	// Create worktree
	_, err := m.git.run("worktree", "add", worktreePath, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %w", err)
	}

	m.worktrees[taskID] = worktreePath
	return worktreePath, nil
}

// RemoveWorktree removes a git worktree for a task
func (m *ParallelBranchManager) RemoveWorktree(taskID string) error {
	worktreePath, ok := m.worktrees[taskID]
	if !ok {
		return nil
	}

	// Remove worktree
	_, err := m.git.run("worktree", "remove", worktreePath, "--force")
	if err != nil {
		// Try manual removal
		os.RemoveAll(worktreePath)
	}

	// Prune worktrees
	m.git.run("worktree", "prune")

	delete(m.worktrees, taskID)
	return nil
}

// GetWorktreePath returns the worktree path for a task
func (m *ParallelBranchManager) GetWorktreePath(taskID string) (string, bool) {
	path, ok := m.worktrees[taskID]
	return path, ok
}

// MergeBranch merges a task branch back to base
func (m *ParallelBranchManager) MergeBranch(taskID string) error {
	branchName, ok := m.branches[taskID]
	if !ok {
		return fmt.Errorf("no branch found for task %s", taskID)
	}

	// Checkout base branch
	_, err := m.git.run("checkout", m.baseBranch)
	if err != nil {
		return fmt.Errorf("failed to checkout base branch: %w", err)
	}

	// Merge task branch
	_, err = m.git.run("merge", branchName, "--no-ff", "-m", fmt.Sprintf("Merge task %s", taskID))
	if err != nil {
		return fmt.Errorf("failed to merge branch %s: %w", branchName, err)
	}

	return nil
}

// MergeBranches merges multiple task branches back to base in order
func (m *ParallelBranchManager) MergeBranches(taskIDs []string) error {
	for _, taskID := range taskIDs {
		if err := m.MergeBranch(taskID); err != nil {
			return err
		}
	}
	return nil
}

// GetConflicts returns conflicting files between two branches
func (m *ParallelBranchManager) GetConflicts(taskID1, taskID2 string) ([]string, error) {
	branch1, ok := m.branches[taskID1]
	if !ok {
		return nil, fmt.Errorf("no branch found for task %s", taskID1)
	}
	branch2, ok := m.branches[taskID2]
	if !ok {
		return nil, fmt.Errorf("no branch found for task %s", taskID2)
	}

	// Get files changed in each branch relative to base
	files1, err := m.getChangedFiles(branch1)
	if err != nil {
		return nil, err
	}
	files2, err := m.getChangedFiles(branch2)
	if err != nil {
		return nil, err
	}

	// Find intersection
	fileSet := make(map[string]bool)
	for _, f := range files1 {
		fileSet[f] = true
	}

	var conflicts []string
	for _, f := range files2 {
		if fileSet[f] {
			conflicts = append(conflicts, f)
		}
	}

	return conflicts, nil
}

// getChangedFiles returns files changed in a branch relative to base
func (m *ParallelBranchManager) getChangedFiles(branch string) ([]string, error) {
	output, err := m.git.run("diff", "--name-only", m.baseBranch+"..."+branch)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// DeleteTaskBranch deletes a task branch
func (m *ParallelBranchManager) DeleteTaskBranch(taskID string) error {
	// Remove worktree first if exists
	m.RemoveWorktree(taskID)

	branchName, ok := m.branches[taskID]
	if !ok {
		return nil
	}

	_, err := m.git.run("branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}

	delete(m.branches, taskID)
	return nil
}

// Cleanup removes all task branches and worktrees
func (m *ParallelBranchManager) Cleanup() error {
	// Remove all worktrees
	for taskID := range m.worktrees {
		m.RemoveWorktree(taskID)
	}

	// Checkout base branch
	m.git.run("checkout", m.baseBranch)

	// Delete all task branches
	for taskID := range m.branches {
		m.DeleteTaskBranch(taskID)
	}

	return nil
}

// ListWorktrees returns list of hermes worktrees
func (m *ParallelBranchManager) ListWorktrees() ([]string, error) {
	output, err := m.git.run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			if strings.Contains(path, "wt-") {
				worktrees = append(worktrees, path)
			}
		}
	}
	return worktrees, nil
}
