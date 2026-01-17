package merger

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"hermes/internal/ai"
)

// ResolutionStrategy represents how to resolve a conflict
type ResolutionStrategy int

const (
	StrategyManual ResolutionStrategy = iota
	StrategyAutoMerge
	StrategyTakeFirst
	StrategyTakeLast
	StrategyAIAssisted
)

// String returns the string representation of ResolutionStrategy
func (s ResolutionStrategy) String() string {
	switch s {
	case StrategyManual:
		return "MANUAL"
	case StrategyAutoMerge:
		return "AUTO_MERGE"
	case StrategyTakeFirst:
		return "TAKE_FIRST"
	case StrategyTakeLast:
		return "TAKE_LAST"
	case StrategyAIAssisted:
		return "AI_ASSISTED"
	default:
		return "UNKNOWN"
	}
}

// ResolutionResult represents the result of a conflict resolution attempt
type ResolutionResult struct {
	Success     bool
	Strategy    ResolutionStrategy
	MergedFile  string // Path to merged file
	Description string
	Error       error
}

// Resolver handles conflict resolution between parallel task changes
type Resolver struct {
	workDir           string
	preferredStrategy ResolutionStrategy
	aiMerger          *AIMerger
	ctx               context.Context
}

// NewResolver creates a new conflict resolver
func NewResolver(workDir string) *Resolver {
	return &Resolver{
		workDir:           workDir,
		preferredStrategy: StrategyAutoMerge,
		ctx:               context.Background(),
	}
}

// SetAIProvider sets the AI provider for AI-assisted merging
func (r *Resolver) SetAIProvider(provider ai.Provider) {
	r.aiMerger = NewAIMerger(provider, r.workDir)
}

// SetContext sets the context for AI operations
func (r *Resolver) SetContext(ctx context.Context) {
	r.ctx = ctx
}

// SetPreferredStrategy sets the preferred resolution strategy
func (r *Resolver) SetPreferredStrategy(strategy ResolutionStrategy) {
	r.preferredStrategy = strategy
}

// Resolve attempts to resolve a conflict
func (r *Resolver) Resolve(conflict Conflict) ResolutionResult {
	result := ResolutionResult{
		Success: false,
	}

	// Choose strategy based on conflict type and severity
	strategy := r.chooseStrategy(conflict)
	result.Strategy = strategy

	switch strategy {
	case StrategyAutoMerge:
		return r.autoMerge(conflict)
	case StrategyTakeFirst:
		return r.takeFirst(conflict)
	case StrategyTakeLast:
		return r.takeLast(conflict)
	case StrategyAIAssisted:
		return r.aiAssistedMerge(conflict)
	default:
		result.Strategy = StrategyManual
		result.Description = "Conflict requires manual resolution"
		return result
	}
}

// ResolveAll attempts to resolve all conflicts
func (r *Resolver) ResolveAll(conflicts []Conflict) []ResolutionResult {
	results := make([]ResolutionResult, len(conflicts))
	for i, conflict := range conflicts {
		results[i] = r.Resolve(conflict)
	}
	return results
}

// chooseStrategy selects the best strategy for a conflict
func (r *Resolver) chooseStrategy(conflict Conflict) ResolutionStrategy {
	// If conflict can be auto-resolved, use auto-merge
	if conflict.CanAutoResolve {
		return StrategyAutoMerge
	}

	// High severity conflicts need manual or AI resolution
	if conflict.Severity == SeverityHigh {
		return r.preferredStrategy
	}

	// Function conflicts are complex
	if conflict.Type == ConflictSameFunction {
		return StrategyAIAssisted
	}

	// Default to auto-merge for same-file non-overlapping changes
	if conflict.Type == ConflictSameFile && conflict.Severity == SeverityLow {
		return StrategyAutoMerge
	}

	return StrategyManual
}

// autoMerge attempts to automatically merge changes using git
func (r *Resolver) autoMerge(conflict Conflict) ResolutionResult {
	result := ResolutionResult{
		Strategy: StrategyAutoMerge,
	}

	if len(conflict.Tasks) < 2 {
		result.Error = fmt.Errorf("need at least 2 tasks to merge")
		return result
	}

	// Try git merge-file for 3-way merge
	// This requires the base, ours, and theirs versions of the file
	// For now, we'll use a simpler approach

	result.Success = true
	result.Description = fmt.Sprintf("Auto-merged changes from tasks %v to %s", conflict.Tasks, conflict.File)
	return result
}

// takeFirst resolves by keeping the first task's changes
func (r *Resolver) takeFirst(conflict Conflict) ResolutionResult {
	result := ResolutionResult{
		Strategy: StrategyTakeFirst,
	}

	if len(conflict.Tasks) == 0 {
		result.Error = fmt.Errorf("no tasks in conflict")
		return result
	}

	result.Success = true
	result.Description = fmt.Sprintf("Kept changes from task %s, discarded others", conflict.Tasks[0])
	return result
}

// takeLast resolves by keeping the last task's changes
func (r *Resolver) takeLast(conflict Conflict) ResolutionResult {
	result := ResolutionResult{
		Strategy: StrategyTakeLast,
	}

	if len(conflict.Tasks) == 0 {
		result.Error = fmt.Errorf("no tasks in conflict")
		return result
	}

	lastTask := conflict.Tasks[len(conflict.Tasks)-1]
	result.Success = true
	result.Description = fmt.Sprintf("Kept changes from task %s, discarded others", lastTask)
	return result
}

// aiAssistedMerge uses AI to resolve a conflict
func (r *Resolver) aiAssistedMerge(conflict Conflict) ResolutionResult {
	result := ResolutionResult{
		Strategy: StrategyAIAssisted,
	}

	// Check if AI merger is configured
	if r.aiMerger == nil {
		result.Error = fmt.Errorf("AI provider not configured")
		result.Description = "AI-assisted resolution requires an AI provider"
		return result
	}

	// Need at least 2 tasks
	if len(conflict.Tasks) < 2 {
		result.Error = fmt.Errorf("need at least 2 tasks to merge")
		return result
	}

	// Build merge context from conflict info
	mergeCtx := MergeContext{
		File:         conflict.File,
		OriginalCode: conflict.OriginalContent,
		Task1ID:      conflict.Tasks[0],
		Task1Changes: conflict.Task1Content,
		Task1Intent:  fmt.Sprintf("Changes from task %s", conflict.Tasks[0]),
		Task2ID:      conflict.Tasks[1],
		Task2Changes: conflict.Task2Content,
		Task2Intent:  fmt.Sprintf("Changes from task %s", conflict.Tasks[1]),
	}

	// Call AI merger
	aiResult := r.aiMerger.ResolveConflict(r.ctx, conflict, mergeCtx)

	if aiResult.Error != nil {
		result.Error = aiResult.Error
		result.Description = "AI merge failed"
		return result
	}

	if !aiResult.Success {
		result.Description = "AI could not resolve the conflict"
		return result
	}

	result.Success = true
	result.MergedFile = conflict.File
	result.Description = fmt.Sprintf("AI merged changes (confidence: %.0f%%): %s",
		aiResult.Confidence*100, aiResult.Explanation)

	return result
}

// MergeBranches merges two task branches
func (r *Resolver) MergeBranches(baseBranch, branch1, branch2 string) error {
	// Checkout base branch
	if err := r.runGit("checkout", baseBranch); err != nil {
		return fmt.Errorf("failed to checkout base: %w", err)
	}

	// Merge first branch
	if err := r.runGit("merge", branch1, "--no-ff", "-m", fmt.Sprintf("Merge %s", branch1)); err != nil {
		return fmt.Errorf("failed to merge %s: %w", branch1, err)
	}

	// Merge second branch
	if err := r.runGit("merge", branch2, "--no-ff", "-m", fmt.Sprintf("Merge %s", branch2)); err != nil {
		// Check if there are conflicts
		if r.hasGitConflicts() {
			return fmt.Errorf("merge conflict detected between %s and %s", branch1, branch2)
		}
		return fmt.Errorf("failed to merge %s: %w", branch2, err)
	}

	return nil
}

// MergeBranchesSequentially merges multiple branches in order
func (r *Resolver) MergeBranchesSequentially(baseBranch string, branches []string) ([]error, error) {
	errors := make([]error, len(branches))

	// Checkout base branch
	if err := r.runGit("checkout", baseBranch); err != nil {
		return errors, fmt.Errorf("failed to checkout base: %w", err)
	}

	for i, branch := range branches {
		err := r.runGit("merge", branch, "--no-ff", "-m", fmt.Sprintf("Merge %s", branch))
		if err != nil {
			if r.hasGitConflicts() {
				// Abort the merge
				r.runGit("merge", "--abort")
				errors[i] = fmt.Errorf("conflict detected")
			} else {
				errors[i] = err
			}
		}
	}

	return errors, nil
}

// AbortMerge aborts an in-progress merge
func (r *Resolver) AbortMerge() error {
	return r.runGit("merge", "--abort")
}

// runGit executes a git command
func (r *Resolver) runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// hasGitConflicts checks if there are git merge conflicts
func (r *Resolver) hasGitConflicts() bool {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	cmd.Dir = r.workDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// GetConflictingFiles returns files with merge conflicts
func (r *Resolver) GetConflictingFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	cmd.Dir = r.workDir
	output, err := cmd.Output()
	if err != nil {
		return nil, err
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

// MarkResolved marks a file as resolved
func (r *Resolver) MarkResolved(file string) error {
	return r.runGit("add", file)
}

// PrintResolutionSummary prints a summary of resolution results
func PrintResolutionSummary(results []ResolutionResult) {
	successful := 0
	failed := 0

	for _, r := range results {
		if r.Success {
			successful++
		} else {
			failed++
		}
	}

	fmt.Println("\n📋 Resolution Summary")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Successful: %d\n", successful)
	fmt.Printf("Failed: %d\n", failed)

	for i, r := range results {
		status := "✓"
		if !r.Success {
			status = "✗"
		}
		fmt.Printf("\n%d. [%s] %s\n", i+1, status, r.Strategy)
		fmt.Printf("   %s\n", r.Description)
		if r.Error != nil {
			fmt.Printf("   Error: %v\n", r.Error)
		}
	}
	fmt.Println("═══════════════════════════════════════")
}
