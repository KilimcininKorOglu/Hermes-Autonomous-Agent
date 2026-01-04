package analyzer

import (
	"regexp"
	"strings"
)

var (
	hermesStatusRegex = regexp.MustCompile(`---HERMES_STATUS---\s*([\s\S]*?)\s*---END_HERMES_STATUS---`)
	statusRegex       = regexp.MustCompile(`STATUS:\s*(\w+)`)
	exitSignalRegex   = regexp.MustCompile(`EXIT_SIGNAL:\s*(true|false)`)
	workTypeRegex     = regexp.MustCompile(`WORK_TYPE:\s*(\w+)`)
	recommendRegex    = regexp.MustCompile(`RECOMMENDATION:\s*(.+)`)

	completionKeywords = []string{
		"done", "complete", "finished", "implemented",
		"all tasks complete", "project complete",
	}

	testOnlyPatterns = []string{
		"npm test", "pytest", "go test", "jest",
		"running tests", "test passed", "tests passed",
	}

	noWorkPatterns = []string{
		"nothing to do", "no changes needed",
		"already implemented", "already exists",
	}

	implementationPatterns = []string{
		"created", "modified", "updated", "added",
		"func ", "function ", "class ", "def ",
	}
)

// ResponseAnalyzer analyzes AI responses
type ResponseAnalyzer struct{}

// NewResponseAnalyzer creates a new response analyzer
func NewResponseAnalyzer() *ResponseAnalyzer {
	return &ResponseAnalyzer{}
}

// Analyze analyzes an AI response and returns the result
func (a *ResponseAnalyzer) Analyze(output string) *AnalysisResult {
	result := &AnalysisResult{
		OutputLength: len(output),
		HasProgress:  true, // Assume progress by default
	}

	outputLower := strings.ToLower(output)

	// Parse HERMES_STATUS block
	a.parseStatusBlock(output, result)

	// Detect completion keywords
	for _, kw := range completionKeywords {
		if strings.Contains(outputLower, kw) {
			result.CompletionKeyword = kw
			result.Confidence += 0.2
			break
		}
	}

	// Detect test-only loop
	hasTestPattern := false
	hasImplementation := false

	for _, pattern := range testOnlyPatterns {
		if strings.Contains(outputLower, pattern) {
			hasTestPattern = true
			break
		}
	}

	// Check for implementation work
	for _, pattern := range implementationPatterns {
		if strings.Contains(outputLower, pattern) {
			hasImplementation = true
			break
		}
	}

	result.IsTestOnly = hasTestPattern && !hasImplementation

	// Detect no-work patterns
	for _, pattern := range noWorkPatterns {
		if strings.Contains(outputLower, pattern) {
			result.HasProgress = false
			break
		}
	}

	// Count errors
	result.ErrorCount = strings.Count(outputLower, "error")
	result.IsStuck = result.ErrorCount > 5

	// Determine status flags based on parsed status
	switch result.Status {
	case "COMPLETE":
		result.IsComplete = true
	case "BLOCKED":
		result.IsBlocked = true
		result.HasProgress = false // Blocked means no progress
	case "AT_RISK":
		result.IsAtRisk = true
	case "PAUSED":
		result.IsPaused = true
		result.HasProgress = false // Paused means no progress
	}

	// Also check for completion via exit signal or keywords
	if result.ExitSignal || result.CompletionKeyword != "" {
		result.IsComplete = true
	}

	// Determine progress
	if !result.HasProgress {
		// Already set to false by no-work patterns or blocked/paused
	} else if result.IsComplete || hasImplementation {
		result.HasProgress = true
	} else if result.OutputLength < 100 || result.IsTestOnly {
		result.HasProgress = false
	}

	// Calculate final confidence
	if result.ExitSignal {
		result.Confidence = 1.0
	} else if result.Status == "COMPLETE" {
		result.Confidence = 0.9
	} else if result.Status == "BLOCKED" {
		result.Confidence = 0.9
	} else if result.Status == "AT_RISK" {
		result.Confidence = 0.8
	} else if result.Status == "PAUSED" {
		result.Confidence = 0.9
	} else if result.CompletionKeyword != "" {
		result.Confidence = 0.7
	}

	return result
}

func (a *ResponseAnalyzer) parseStatusBlock(output string, result *AnalysisResult) {
	matches := hermesStatusRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		return
	}

	block := matches[1]

	if m := statusRegex.FindStringSubmatch(block); len(m) > 1 {
		result.Status = m[1]
	}

	if m := exitSignalRegex.FindStringSubmatch(block); len(m) > 1 {
		result.ExitSignal = m[1] == "true"
	}

	if m := workTypeRegex.FindStringSubmatch(block); len(m) > 1 {
		result.WorkType = m[1]
	}

	if m := recommendRegex.FindStringSubmatch(block); len(m) > 1 {
		result.Recommendation = strings.TrimSpace(m[1])
	}
}

// HasStatusBlock checks if the output contains a HERMES_STATUS block
func (a *ResponseAnalyzer) HasStatusBlock(output string) bool {
	return hermesStatusRegex.MatchString(output)
}

// ExtractStatusBlock extracts the status block from output
func (a *ResponseAnalyzer) ExtractStatusBlock(output string) string {
	matches := hermesStatusRegex.FindStringSubmatch(output)
	if len(matches) >= 1 {
		return matches[0]
	}
	return ""
}
