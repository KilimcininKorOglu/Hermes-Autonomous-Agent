package git

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// GetFeatureBranchName generates a branch name for a feature
func (g *Git) GetFeatureBranchName(featureID, featureName string) string {
	name := SanitizeBranchName(featureName)

	// Truncate if too long
	if len(name) > 30 {
		name = name[:30]
	}

	// Remove trailing hyphens
	name = strings.TrimRight(name, "-")

	return fmt.Sprintf("feature/%s-%s", featureID, name)
}

// GetTaskBranchName generates a branch name for a task
func GetTaskBranchName(taskID, taskName string) string {
	name := SanitizeBranchName(taskName)

	// Truncate if too long
	if len(name) > 30 {
		name = name[:30]
	}

	// Remove trailing hyphens
	name = strings.TrimRight(name, "-")

	return fmt.Sprintf("task/%s-%s", taskID, name)
}

// SanitizeBranchName converts a name to a valid git branch name
func SanitizeBranchName(name string) string {
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

// CreateBranch creates a new branch and switches to it
func (g *Git) CreateBranch(name string) error {
	_, err := g.run("checkout", "-b", name)
	return err
}

// CheckoutBranch switches to an existing branch
func (g *Git) CheckoutBranch(name string) error {
	_, err := g.run("checkout", name)
	return err
}

// CreateFeatureBranch creates or switches to a feature branch
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

// EnsureOnFeatureBranch ensures we're on the feature branch
func (g *Git) EnsureOnFeatureBranch(featureID, featureName string) error {
	_, err := g.CreateFeatureBranch(featureID, featureName)
	return err
}

// DeleteBranch deletes a branch
func (g *Git) DeleteBranch(name string) error {
	_, err := g.run("branch", "-d", name)
	return err
}

// ForceDeleteBranch force deletes a branch
func (g *Git) ForceDeleteBranch(name string) error {
	_, err := g.run("branch", "-D", name)
	return err
}

// MergeBranch merges a branch into the current branch
func (g *Git) MergeBranch(branchName string) error {
	_, err := g.run("merge", branchName, "--no-edit")
	return err
}

// MergeFeatureBranch merges a feature branch into main and returns to main
func (g *Git) MergeFeatureBranch(featureID, featureName string) error {
	branchName := g.GetFeatureBranchName(featureID, featureName)
	
	// Check if branch exists
	if !g.BranchExists(branchName) {
		return nil // Nothing to merge
	}
	
	// Get main branch
	mainBranch := g.GetMainBranch()
	
	// Checkout main
	if err := g.CheckoutBranch(mainBranch); err != nil {
		return fmt.Errorf("failed to checkout %s: %w", mainBranch, err)
	}
	
	// Merge feature branch
	if err := g.MergeBranch(branchName); err != nil {
		return fmt.Errorf("failed to merge %s: %w", branchName, err)
	}
	
	return nil
}

// ListBranches returns all local branches
func (g *Git) ListBranches() ([]string, error) {
	output, err := g.run("branch", "--list", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}
	return strings.Split(output, "\n"), nil
}
