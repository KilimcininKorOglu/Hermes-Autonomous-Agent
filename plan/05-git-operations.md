# Phase 05: Git Operations

## Goal

Implement Git branch management and commit operations.

## PowerShell Reference

```powershell
# From lib/GitBranchManager.ps1
# - Get-FeatureBranchName
# - Get-MainBranch
# - Test-GitRepository
# - Get-CurrentBranch
# - Test-WorkingTreeClean
# - Test-BranchExists
# - New-FeatureBranch
# - Invoke-GitCommit
```

## Go Implementation

### 5.1 Git Package

```go
// internal/git/git.go
package git

import (
    "fmt"
    "os/exec"
    "regexp"
    "strings"
    "unicode"
)

type Git struct {
    workDir string
}

func New(workDir string) *Git {
    return &Git{workDir: workDir}
}

func (g *Git) run(args ...string) (string, error) {
    cmd := exec.Command("git", args...)
    cmd.Dir = g.workDir
    output, err := cmd.CombinedOutput()
    return strings.TrimSpace(string(output)), err
}

func (g *Git) IsRepository() bool {
    _, err := g.run("rev-parse", "--git-dir")
    return err == nil
}

func (g *Git) GetCurrentBranch() (string, error) {
    return g.run("rev-parse", "--abbrev-ref", "HEAD")
}

func (g *Git) GetMainBranch() string {
    // Check if 'main' exists
    if _, err := g.run("rev-parse", "--verify", "main"); err == nil {
        return "main"
    }
    return "master"
}

func (g *Git) IsWorkingTreeClean() bool {
    output, _ := g.run("status", "--porcelain")
    return output == ""
}

func (g *Git) HasStagedChanges() bool {
    output, _ := g.run("diff", "--cached", "--name-only")
    return output != ""
}

func (g *Git) BranchExists(branch string) bool {
    _, err := g.run("rev-parse", "--verify", branch)
    return err == nil
}

func (g *Git) IsMergeInProgress() bool {
    _, err := g.run("rev-parse", "-q", "--verify", "MERGE_HEAD")
    return err == nil
}

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
```

### 5.2 Branch Management

```go
// internal/git/branch.go
package git

import (
    "fmt"
    "regexp"
    "strings"
    "unicode"
)

func (g *Git) GetFeatureBranchName(featureID, featureName string) string {
    // Sanitize feature name
    name := sanitizeBranchName(featureName)
    
    // Truncate if too long
    if len(name) > 30 {
        name = name[:30]
    }
    
    // Remove trailing hyphens
    name = strings.TrimRight(name, "-")
    
    return fmt.Sprintf("feature/%s-%s", featureID, name)
}

func sanitizeBranchName(name string) string {
    // Convert to lowercase
    name = strings.ToLower(name)
    
    // Replace spaces and special chars with hyphens
    var result strings.Builder
    for _, r := range name {
        if unicode.IsLetter(r) || unicode.IsDigit(r) {
            result.WriteRune(r)
        } else if r == ' ' || r == '-' || r == '_' {
            result.WriteRune('-')
        }
    }
    
    // Collapse multiple hyphens
    re := regexp.MustCompile(`-+`)
    return re.ReplaceAllString(result.String(), "-")
}

func (g *Git) CreateBranch(name string) error {
    _, err := g.run("checkout", "-b", name)
    return err
}

func (g *Git) CheckoutBranch(name string) error {
    _, err := g.run("checkout", name)
    return err
}

func (g *Git) CreateFeatureBranch(featureID, featureName string) (string, error) {
    branchName := g.GetFeatureBranchName(featureID, featureName)
    
    // Check if already on feature branch
    current, _ := g.GetCurrentBranch()
    if current == branchName {
        return branchName, nil
    }
    
    // Create or checkout
    if g.BranchExists(branchName) {
        return branchName, g.CheckoutBranch(branchName)
    }
    
    return branchName, g.CreateBranch(branchName)
}

func (g *Git) EnsureOnFeatureBranch(featureID, featureName string) error {
    _, err := g.CreateFeatureBranch(featureID, featureName)
    return err
}
```

### 5.3 Commit Operations

```go
// internal/git/commit.go
package git

import "fmt"

func (g *Git) StageAll() error {
    _, err := g.run("add", "-A")
    return err
}

func (g *Git) StageFiles(files ...string) error {
    args := append([]string{"add"}, files...)
    _, err := g.run(args...)
    return err
}

func (g *Git) Commit(message string) error {
    _, err := g.run("commit", "-m", message)
    return err
}

func (g *Git) CommitTask(taskID, taskName string) error {
    message := fmt.Sprintf("feat(%s): %s", taskID, taskName)
    return g.Commit(message)
}

func (g *Git) CommitFeature(featureID, featureName string) error {
    message := fmt.Sprintf("feat(%s): complete %s", featureID, featureName)
    return g.Commit(message)
}

func (g *Git) GetLastCommitMessage() (string, error) {
    return g.run("log", "-1", "--pretty=%B")
}

func (g *Git) HasUncommittedChanges() bool {
    return !g.IsWorkingTreeClean()
}
```

## Files to Create

| File | Description |
|------|-------------|
| `internal/git/git.go` | Core git operations |
| `internal/git/branch.go` | Branch management |
| `internal/git/commit.go` | Commit operations |
| `internal/git/git_test.go` | Tests |

## Acceptance Criteria

- [ ] Detect git repository
- [ ] Get current branch
- [ ] Create feature branches
- [ ] Sanitize branch names
- [ ] Stage and commit changes
- [ ] Check working tree status
- [ ] All tests pass
