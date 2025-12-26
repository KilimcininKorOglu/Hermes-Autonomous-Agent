package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Git provides git operations
type Git struct {
	workDir string
}

// New creates a new Git instance
func New(workDir string) *Git {
	return &Git{workDir: workDir}
}

// run executes a git command and returns the output
func (g *Git) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.workDir
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// IsRepository checks if the directory is a git repository
func (g *Git) IsRepository() bool {
	_, err := g.run("rev-parse", "--git-dir")
	return err == nil
}

// GetCurrentBranch returns the current branch name
func (g *Git) GetCurrentBranch() (string, error) {
	return g.run("rev-parse", "--abbrev-ref", "HEAD")
}

// GetMainBranch returns the main branch name (main or master)
func (g *Git) GetMainBranch() string {
	if _, err := g.run("rev-parse", "--verify", "main"); err == nil {
		return "main"
	}
	return "master"
}

// IsWorkingTreeClean returns true if there are no uncommitted changes
func (g *Git) IsWorkingTreeClean() bool {
	output, _ := g.run("status", "--porcelain")
	return output == ""
}

// HasStagedChanges returns true if there are staged changes
func (g *Git) HasStagedChanges() bool {
	output, _ := g.run("diff", "--cached", "--name-only")
	return output != ""
}

// HasUncommittedChanges returns true if there are any uncommitted changes
func (g *Git) HasUncommittedChanges() bool {
	return !g.IsWorkingTreeClean()
}

// BranchExists checks if a branch exists
func (g *Git) BranchExists(branch string) bool {
	_, err := g.run("rev-parse", "--verify", branch)
	return err == nil
}

// IsMergeInProgress checks if a merge is in progress
func (g *Git) IsMergeInProgress() bool {
	_, err := g.run("rev-parse", "-q", "--verify", "MERGE_HEAD")
	return err == nil
}

// GetCommitsSinceMain returns the number of commits since main branch
func (g *Git) GetCommitsSinceMain() (int, error) {
	main := g.GetMainBranch()
	output, err := g.run("rev-list", "--count", fmt.Sprintf("%s..HEAD", main))
	if err != nil {
		return 0, err
	}
	var count int
	fmt.Sscanf(output, "%d", &count)
	return count, nil
}

// GetStatus returns git status output
func (g *Git) GetStatus() (string, error) {
	return g.run("status", "--short")
}

// GetDiff returns git diff output
func (g *Git) GetDiff() (string, error) {
	return g.run("diff")
}

// GetDiffCached returns staged changes diff
func (g *Git) GetDiffCached() (string, error) {
	return g.run("diff", "--cached")
}
