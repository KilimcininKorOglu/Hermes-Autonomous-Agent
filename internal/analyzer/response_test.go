package analyzer

import (
	"testing"
)

func TestAnalyzeStatusBlock(t *testing.T) {
	a := NewResponseAnalyzer()

	output := `
Some AI output here...

---HERMES_STATUS---
STATUS: COMPLETE
EXIT_SIGNAL: true
WORK_TYPE: implementation
RECOMMENDATION: Move to next task
---END_HERMES_STATUS---
`

	result := a.Analyze(output)

	if result.Status != "COMPLETE" {
		t.Errorf("expected status COMPLETE, got %s", result.Status)
	}
	if !result.ExitSignal {
		t.Error("expected ExitSignal = true")
	}
	if result.WorkType != "implementation" {
		t.Errorf("expected WorkType 'implementation', got %s", result.WorkType)
	}
	if result.Recommendation != "Move to next task" {
		t.Errorf("expected Recommendation 'Move to next task', got %s", result.Recommendation)
	}
	if !result.IsComplete {
		t.Error("expected IsComplete = true")
	}
	if result.Confidence != 1.0 {
		t.Errorf("expected Confidence = 1.0, got %f", result.Confidence)
	}
}

func TestAnalyzeInProgress(t *testing.T) {
	a := NewResponseAnalyzer()

	output := `
---HERMES_STATUS---
STATUS: IN_PROGRESS
EXIT_SIGNAL: false
RECOMMENDATION: Continue implementing
---END_HERMES_STATUS---
`

	result := a.Analyze(output)

	if result.Status != "IN_PROGRESS" {
		t.Errorf("expected status IN_PROGRESS, got %s", result.Status)
	}
	if result.ExitSignal {
		t.Error("expected ExitSignal = false")
	}
	if result.IsComplete {
		t.Error("expected IsComplete = false")
	}
}

func TestAnalyzeCompletionKeywords(t *testing.T) {
	a := NewResponseAnalyzer()

	tests := []struct {
		output   string
		complete bool
	}{
		{"The task is done and working", true},
		{"Implementation is complete", true},
		{"All finished successfully", true},
		{"Feature implemented correctly", true},
		{"Still working on it", false},
	}

	for _, tt := range tests {
		result := a.Analyze(tt.output)
		if result.IsComplete != tt.complete {
			t.Errorf("Analyze(%q): expected IsComplete=%v, got %v",
				tt.output, tt.complete, result.IsComplete)
		}
	}
}

func TestAnalyzeTestOnly(t *testing.T) {
	a := NewResponseAnalyzer()

	// Test-only output (no implementation)
	testOnlyOutput := "Running tests... npm test... All tests passed."
	result := a.Analyze(testOnlyOutput)
	if !result.IsTestOnly {
		t.Error("expected IsTestOnly = true for test-only output")
	}

	// Output with implementation
	implementOutput := "Created new function handleLogin... npm test... Tests passed."
	result = a.Analyze(implementOutput)
	if result.IsTestOnly {
		t.Error("expected IsTestOnly = false when implementation is present")
	}
}

func TestAnalyzeNoProgress(t *testing.T) {
	a := NewResponseAnalyzer()

	noProgressOutput := "There is nothing to do, already implemented."
	result := a.Analyze(noProgressOutput)
	if result.HasProgress {
		t.Error("expected HasProgress = false for no-progress output")
	}
}

func TestAnalyzeErrorCount(t *testing.T) {
	a := NewResponseAnalyzer()

	output := "Error: file not found\nError: not available\nError: compile failed\nError: test failed\nError: runtime failure\nError: another issue"
	result := a.Analyze(output)

	if result.ErrorCount != 6 {
		t.Errorf("expected ErrorCount = 6, got %d", result.ErrorCount)
	}
	if !result.IsStuck {
		t.Error("expected IsStuck = true when error count > 5")
	}
}

func TestHasStatusBlock(t *testing.T) {
	a := NewResponseAnalyzer()

	withBlock := "Output\n---HERMES_STATUS---\nSTATUS: COMPLETE\n---END_HERMES_STATUS---"
	withoutBlock := "Just some output without status block"

	if !a.HasStatusBlock(withBlock) {
		t.Error("expected HasStatusBlock = true")
	}
	if a.HasStatusBlock(withoutBlock) {
		t.Error("expected HasStatusBlock = false")
	}
}

func TestExtractStatusBlock(t *testing.T) {
	a := NewResponseAnalyzer()

	output := "Prefix\n---HERMES_STATUS---\nSTATUS: COMPLETE\nEXIT_SIGNAL: true\n---END_HERMES_STATUS---\nSuffix"

	block := a.ExtractStatusBlock(output)

	if block == "" {
		t.Error("expected non-empty status block")
	}
	if block != "---HERMES_STATUS---\nSTATUS: COMPLETE\nEXIT_SIGNAL: true\n---END_HERMES_STATUS---" {
		t.Errorf("unexpected block content: %s", block)
	}
}

func TestAnalyzeOutputLength(t *testing.T) {
	a := NewResponseAnalyzer()

	shortOutput := "OK"
	longOutput := "This is a much longer output with detailed implementation work including created new functions and modified existing code to implement the feature correctly."

	shortResult := a.Analyze(shortOutput)
	longResult := a.Analyze(longOutput)

	if shortResult.OutputLength != 2 {
		t.Errorf("expected OutputLength = 2, got %d", shortResult.OutputLength)
	}
	if longResult.OutputLength <= 100 {
		t.Error("expected OutputLength > 100 for long output")
	}
}

func TestAnalyzeProgress(t *testing.T) {
	a := NewResponseAnalyzer()

	tests := []struct {
		name     string
		output   string
		progress bool
	}{
		{
			name:     "implementation work",
			output:   "Created new file auth.go with login function",
			progress: true,
		},
		{
			name:     "modified code",
			output:   "Modified the handler to support new feature",
			progress: true,
		},
		{
			name:     "complete status",
			output:   "---HERMES_STATUS---\nSTATUS: COMPLETE\nEXIT_SIGNAL: true\n---END_HERMES_STATUS---",
			progress: true,
		},
		{
			name:     "nothing to do",
			output:   "Nothing to do here",
			progress: false,
		},
		{
			name:     "already exists",
			output:   "The feature already exists in the codebase",
			progress: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.Analyze(tt.output)
			if result.HasProgress != tt.progress {
				t.Errorf("expected HasProgress=%v, got %v", tt.progress, result.HasProgress)
			}
		})
	}
}
