package ai

import (
	"context"
	"fmt"
	"strings"

	"hermes/internal/task"
)

const statusBlockMarker = "---HERMES_STATUS---"

const statusReminderPrompt = `You did not include the required HERMES_STATUS block in your response.

This block is MANDATORY. Please provide ONLY the status block now, based on your work:

If you completed the task successfully:
` + "```" + `
---HERMES_STATUS---
STATUS: COMPLETE
EXIT_SIGNAL: true
RECOMMENDATION: Move to next task
---END_HERMES_STATUS---
` + "```" + `

If you are blocked:
` + "```" + `
---HERMES_STATUS---
STATUS: BLOCKED
EXIT_SIGNAL: false
RECOMMENDATION: <describe what is blocking>
---END_HERMES_STATUS---
` + "```" + `

If still in progress:
` + "```" + `
---HERMES_STATUS---
STATUS: IN_PROGRESS
EXIT_SIGNAL: false
RECOMMENDATION: <describe next steps>
---END_HERMES_STATUS---
` + "```" + `

Output ONLY the status block, nothing else.`

// TaskExecutor executes tasks using an AI provider
type TaskExecutor struct {
	provider Provider
	workDir  string
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(provider Provider, workDir string) *TaskExecutor {
	return &TaskExecutor{
		provider: provider,
		workDir:  workDir,
	}
}

// ExecuteTask executes a single task
func (e *TaskExecutor) ExecuteTask(ctx context.Context, t *task.Task, promptContent string, streamOutput bool) (*ExecuteResult, error) {
	prompt := e.buildTaskPrompt(t, promptContent)

	opts := &ExecuteOptions{
		Prompt:       prompt,
		WorkDir:      e.workDir,
		Tools:        []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
		StreamOutput: streamOutput,
	}

	var result *ExecuteResult
	var err error

	if streamOutput {
		result, err = e.executeWithStreaming(ctx, opts)
	} else {
		result, err = e.provider.Execute(ctx, opts)
	}

	if err != nil {
		return nil, err
	}

	// Check if HERMES_STATUS block is present
	if !strings.Contains(result.Output, statusBlockMarker) {
		// Ask AI to provide the status block
		statusResult, statusErr := e.requestStatusBlock(ctx)
		if statusErr == nil && strings.Contains(statusResult.Output, statusBlockMarker) {
			// Append status block to original output
			result.Output = result.Output + "\n\n" + statusResult.Output
		}
	}

	return result, nil
}

// requestStatusBlock asks AI to provide the missing status block
func (e *TaskExecutor) requestStatusBlock(ctx context.Context) (*ExecuteResult, error) {
	opts := &ExecuteOptions{
		Prompt:  statusReminderPrompt,
		WorkDir: e.workDir,
		Tools:   []string{}, // No tools needed for status block
	}

	return e.provider.Execute(ctx, opts)
}

// executeWithStreaming executes with real-time output to console
func (e *TaskExecutor) executeWithStreaming(ctx context.Context, opts *ExecuteOptions) (*ExecuteResult, error) {
	events, err := e.provider.ExecuteStream(ctx, opts)
	if err != nil {
		return nil, err
	}

	var output string
	for event := range events {
		switch event.Type {
		case "text":
			fmt.Print(event.Text)
			output += event.Text
		case "error":
			return &ExecuteResult{Success: false, Output: output, Error: event.Text}, nil
		case "done":
			fmt.Println()
		}
	}

	return &ExecuteResult{Success: true, Output: output}, nil
}

// ExecuteTaskStream executes a task with streaming output
func (e *TaskExecutor) ExecuteTaskStream(ctx context.Context, t *task.Task, promptContent string) (<-chan StreamEvent, error) {
	prompt := e.buildTaskPrompt(t, promptContent)

	opts := &ExecuteOptions{
		Prompt:  prompt,
		WorkDir: e.workDir,
		Tools:   []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
	}

	return e.provider.ExecuteStream(ctx, opts)
}

// ExecutePrompt executes a raw prompt without task context
func (e *TaskExecutor) ExecutePrompt(ctx context.Context, prompt string, taskID string) (*ExecuteResult, error) {
	opts := &ExecuteOptions{
		Prompt:  prompt,
		WorkDir: e.workDir,
		Tools:   []string{"Read"}, // Limited tools for merge operations
	}

	return e.provider.Execute(ctx, opts)
}

func (e *TaskExecutor) buildTaskPrompt(t *task.Task, promptContent string) string {
	return fmt.Sprintf(`%s

## Current Task: %s

**Task:** %s: %s

**Files to Touch:**
%s

**Success Criteria:**
%s

Complete this task and output the HERMES_STATUS block when done:

` + "```" + `
---HERMES_STATUS---
STATUS: COMPLETE
EXIT_SIGNAL: true
RECOMMENDATION: Move to next task
---END_HERMES_STATUS---
` + "```",
		promptContent,
		t.ID,
		t.ID, t.Name,
		formatFiles(t.FilesToTouch),
		formatCriteria(t.SuccessCriteria),
	)
}

func formatFiles(files []string) string {
	if len(files) == 0 {
		return "- (none specified)"
	}
	result := ""
	for _, f := range files {
		result += fmt.Sprintf("- %s\n", f)
	}
	return result
}

func formatCriteria(criteria []string) string {
	if len(criteria) == 0 {
		return "- (none specified)"
	}
	result := ""
	for _, c := range criteria {
		result += fmt.Sprintf("- %s\n", c)
	}
	return result
}
