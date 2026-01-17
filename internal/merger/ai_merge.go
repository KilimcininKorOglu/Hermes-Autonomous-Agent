package merger

import (
	"context"
	"fmt"
	"strings"

	"hermes/internal/ai"
)

// AIMerger uses AI to resolve complex merge conflicts
type AIMerger struct {
	provider ai.Provider
	workDir  string
}

// MergeContext provides context for AI-assisted merge
type MergeContext struct {
	File           string
	OriginalCode   string
	Task1ID        string
	Task1Changes   string
	Task1Intent    string
	Task2ID        string
	Task2Changes   string
	Task2Intent    string
}

// MergeResult represents the result of an AI merge
type MergeResult struct {
	Success     bool
	MergedCode  string
	Explanation string
	Confidence  float64
	Error       error
}

// NewAIMerger creates a new AI merger
func NewAIMerger(provider ai.Provider, workDir string) *AIMerger {
	return &AIMerger{
		provider: provider,
		workDir:  workDir,
	}
}

// ResolveConflict uses AI to resolve a merge conflict
func (m *AIMerger) ResolveConflict(ctx context.Context, conflict Conflict, mergeCtx MergeContext) MergeResult {
	result := MergeResult{}

	// Build the merge prompt
	prompt := m.buildMergePrompt(mergeCtx)

	// Execute AI request
	output, err := m.executeAI(ctx, prompt)
	if err != nil {
		result.Error = fmt.Errorf("AI merge failed: %w", err)
		return result
	}

	// Parse the AI response
	mergedCode, explanation, confidence := m.parseResponse(output)

	result.Success = mergedCode != ""
	result.MergedCode = mergedCode
	result.Explanation = explanation
	result.Confidence = confidence

	return result
}

// buildMergePrompt creates the prompt for AI-assisted merge
func (m *AIMerger) buildMergePrompt(ctx MergeContext) string {
	prompt := fmt.Sprintf(`You are merging code changes from two parallel tasks that modified the same file.

File: %s

## Original Code
%s

## Task 1: %s
Intent: %s
Changes:
%s

## Task 2: %s
Intent: %s
Changes:
%s

## Instructions
1. Analyze both changes and understand their intent
2. Create a merged version that preserves BOTH intents
3. Resolve any conflicts intelligently
4. Maintain code correctness and consistency

## Output Format
Provide your response in the following format:

MERGED_CODE_START
[Your merged code here]
MERGED_CODE_END

EXPLANATION:
[Brief explanation of how you merged the changes]

CONFIDENCE: [0.0-1.0]
`,
		ctx.File,
		ctx.OriginalCode,
		ctx.Task1ID, ctx.Task1Intent, ctx.Task1Changes,
		ctx.Task2ID, ctx.Task2Intent, ctx.Task2Changes,
	)

	return prompt
}

// executeAI runs the AI merge request
func (m *AIMerger) executeAI(ctx context.Context, prompt string) (string, error) {
	if m.provider == nil {
		return "", fmt.Errorf("no AI provider configured")
	}

	// Create a merge-specific executor
	executor := ai.NewTaskExecutor(m.provider, m.workDir)

	// Create a temporary task for the merge
	mergeTask := struct {
		ID          string
		Name        string
		Description string
	}{
		ID:          "MERGE",
		Name:        "AI-Assisted Merge",
		Description: "Resolving merge conflicts",
	}

	// Execute with the prompt
	result, err := executor.ExecutePrompt(ctx, prompt, mergeTask.ID)
	if err != nil {
		return "", err
	}

	return result.Output, nil
}

// parseResponse extracts the merged code from AI output
func (m *AIMerger) parseResponse(output string) (code, explanation string, confidence float64) {
	// Extract merged code
	if startIdx := strings.Index(output, "MERGED_CODE_START"); startIdx != -1 {
		if endIdx := strings.Index(output, "MERGED_CODE_END"); endIdx != -1 && endIdx > startIdx {
			codeStart := startIdx + len("MERGED_CODE_START")
			code = strings.TrimSpace(output[codeStart:endIdx])
		}
	}

	// Extract explanation
	if expIdx := strings.Index(output, "EXPLANATION:"); expIdx != -1 {
		expEnd := strings.Index(output[expIdx:], "CONFIDENCE:")
		if expEnd == -1 {
			expEnd = len(output) - expIdx
		}
		explanation = strings.TrimSpace(output[expIdx+len("EXPLANATION:"):expIdx+expEnd])
	}

	// Extract confidence
	if confIdx := strings.Index(output, "CONFIDENCE:"); confIdx != -1 {
		confStr := strings.TrimSpace(output[confIdx+len("CONFIDENCE:"):])
		if len(confStr) > 0 {
			// Parse first number found
			for _, c := range confStr {
				if c >= '0' && c <= '9' || c == '.' {
					continue
				}
				confStr = confStr[:strings.IndexRune(confStr, c)]
				break
			}
			fmt.Sscanf(confStr, "%f", &confidence)
		}
	}

	// Default confidence if not found
	if confidence == 0 && code != "" {
		confidence = 0.7
	}

	return
}

// MergeMultipleChanges merges changes from multiple tasks
func (m *AIMerger) MergeMultipleChanges(ctx context.Context, file string, original string, changes []TaskMergeInfo) MergeResult {
	if len(changes) < 2 {
		return MergeResult{
			Error: fmt.Errorf("need at least 2 changes to merge"),
		}
	}

	// For multiple tasks, merge pairwise
	current := original
	var lastExplanation string

	for i := 1; i < len(changes); i++ {
		mergeCtx := MergeContext{
			File:         file,
			OriginalCode: current,
			Task1ID:      changes[i-1].TaskID,
			Task1Changes: changes[i-1].Diff,
			Task1Intent:  changes[i-1].Intent,
			Task2ID:      changes[i].TaskID,
			Task2Changes: changes[i].Diff,
			Task2Intent:  changes[i].Intent,
		}

		result := m.ResolveConflict(ctx, Conflict{File: file}, mergeCtx)
		if !result.Success {
			return result
		}

		current = result.MergedCode
		lastExplanation += fmt.Sprintf("\nMerge %d: %s", i, result.Explanation)
	}

	return MergeResult{
		Success:     true,
		MergedCode:  current,
		Explanation: lastExplanation,
		Confidence:  0.8,
	}
}

// TaskMergeInfo contains information about a task's changes for merging
type TaskMergeInfo struct {
	TaskID string
	Diff   string
	Intent string
}

// ValidateMerge checks if the merged code is valid
func (m *AIMerger) ValidateMerge(ctx context.Context, file, mergedCode string) (bool, string, error) {
	// Basic validation: check for obvious issues
	
	// Check for conflict markers
	if strings.Contains(mergedCode, "<<<<<<<") || 
	   strings.Contains(mergedCode, "=======") || 
	   strings.Contains(mergedCode, ">>>>>>>") {
		return false, "Merged code contains conflict markers", nil
	}

	// Check for empty result
	if strings.TrimSpace(mergedCode) == "" {
		return false, "Merged code is empty", nil
	}

	// TODO: Add syntax validation based on file type
	// For Go files, we could use go/parser
	// For now, we just do basic checks

	return true, "Validation passed", nil
}

// AnalyzeSemanticConflict uses AI to detect semantic conflicts
func (m *AIMerger) AnalyzeSemanticConflict(ctx context.Context, file string, changes []TaskMergeInfo) (*SemanticConflictResult, error) {
	if len(changes) < 2 {
		return nil, fmt.Errorf("need at least 2 changes to analyze")
	}

	prompt := m.buildSemanticAnalysisPrompt(file, changes)
	output, err := m.executeAI(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return m.parseSemanticAnalysis(output), nil
}

// SemanticConflictResult represents the result of semantic conflict analysis
type SemanticConflictResult struct {
	HasConflict bool
	Severity    int // 1-3
	Description string
	Suggestion  string
}

func (m *AIMerger) buildSemanticAnalysisPrompt(file string, changes []TaskMergeInfo) string {
	var changesDesc strings.Builder
	for i, c := range changes {
		changesDesc.WriteString(fmt.Sprintf("\n## Task %d: %s\nIntent: %s\nChanges:\n%s\n",
			i+1, c.TaskID, c.Intent, c.Diff))
	}

	return fmt.Sprintf(`Analyze these code changes for semantic conflicts:

File: %s
%s

A semantic conflict occurs when changes are syntactically compatible but logically incompatible.
Examples:
- One task adds logging, another removes it
- Different tasks modify the same business logic differently
- Contradicting configuration changes

Output format:
HAS_CONFLICT: [true/false]
SEVERITY: [1-3]
DESCRIPTION: [description of conflict if any]
SUGGESTION: [how to resolve]
`, file, changesDesc.String())
}

func (m *AIMerger) parseSemanticAnalysis(output string) *SemanticConflictResult {
	result := &SemanticConflictResult{}

	// Parse has conflict
	if strings.Contains(strings.ToLower(output), "has_conflict: true") {
		result.HasConflict = true
	}

	// Parse severity
	if idx := strings.Index(output, "SEVERITY:"); idx != -1 {
		fmt.Sscanf(output[idx+len("SEVERITY:"):], "%d", &result.Severity)
	}

	// Parse description
	if idx := strings.Index(output, "DESCRIPTION:"); idx != -1 {
		end := strings.Index(output[idx:], "SUGGESTION:")
		if end == -1 {
			end = len(output) - idx
		}
		result.Description = strings.TrimSpace(output[idx+len("DESCRIPTION:"):idx+end])
	}

	// Parse suggestion
	if idx := strings.Index(output, "SUGGESTION:"); idx != -1 {
		result.Suggestion = strings.TrimSpace(output[idx+len("SUGGESTION:"):])
	}

	return result
}
