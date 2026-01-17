# Phase 08: Response Analysis

## Goal

Analyze AI responses to detect completion, progress, and issues.

## PowerShell Reference

```powershell
# From lib/ResponseAnalyzer.ps1
# - Invoke-ResponseAnalysis
# - Detect HERMES_STATUS block
# - Detect completion keywords
# - Detect test-only loops
# - Calculate confidence score
```

## Go Implementation

### 8.1 Analysis Types

```go
// internal/analyzer/types.go
package analyzer

type AnalysisResult struct {
    HasProgress       bool    `json:"has_progress"`
    IsComplete        bool    `json:"is_complete"`
    IsTestOnly        bool    `json:"is_test_only"`
    IsStuck           bool    `json:"is_stuck"`
    ExitSignal        bool    `json:"exit_signal"`
    Status            string  `json:"status"`
    WorkType          string  `json:"work_type"`
    Recommendation    string  `json:"recommendation"`
    Confidence        float64 `json:"confidence"`
    OutputLength      int     `json:"output_length"`
    ErrorCount        int     `json:"error_count"`
    CompletionKeyword string  `json:"completion_keyword"`
}

type ExitSignals struct {
    Signals      []SignalEntry `json:"signals"`
    TestOnlyRuns int           `json:"test_only_runs"`
    DoneSignals  int           `json:"done_signals"`
}

type SignalEntry struct {
    LoopNumber int    `json:"loop_number"`
    Signal     string `json:"signal"`
    Timestamp  string `json:"timestamp"`
}
```

### 8.2 Response Analyzer

```go
// internal/analyzer/response.go
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
)

type ResponseAnalyzer struct{}

func NewResponseAnalyzer() *ResponseAnalyzer {
    return &ResponseAnalyzer{}
}

func (a *ResponseAnalyzer) Analyze(output string) *AnalysisResult {
    result := &AnalysisResult{
        OutputLength: len(output),
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
    implementationPatterns := []string{
        "created", "modified", "updated", "added",
        "func ", "function ", "class ", "def ",
    }
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
    
    // Determine if complete
    result.IsComplete = result.ExitSignal || 
        result.Status == "COMPLETE" || 
        result.CompletionKeyword != ""
    
    // Determine progress
    result.HasProgress = result.IsComplete || 
        hasImplementation || 
        (result.OutputLength > 100 && !result.IsTestOnly)
    
    // Calculate final confidence
    if result.ExitSignal {
        result.Confidence = 1.0
    } else if result.Status == "COMPLETE" {
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
```

### 8.3 Feature Analyzer

```go
// internal/analyzer/feature.go
package analyzer

import (
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
    
    "hermes/internal/task"
)

type FeatureAnalyzer struct {
    basePath string
}

func NewFeatureAnalyzer(basePath string) *FeatureAnalyzer {
    return &FeatureAnalyzer{basePath: basePath}
}

func (a *FeatureAnalyzer) GetHighestFeatureID() (int, error) {
    reader := task.NewReader(a.basePath)
    files, err := reader.GetFeatureFiles()
    if err != nil {
        return 0, err
    }
    
    highest := 0
    re := regexp.MustCompile(`F(\d+)`)
    
    for _, file := range files {
        feature, err := reader.ReadFeature(file)
        if err != nil {
            continue
        }
        
        if m := re.FindStringSubmatch(feature.ID); len(m) > 1 {
            if n, _ := strconv.Atoi(m[1]); n > highest {
                highest = n
            }
        }
    }
    
    return highest, nil
}

func (a *FeatureAnalyzer) GetHighestTaskID() (int, error) {
    reader := task.NewReader(a.basePath)
    tasks, err := reader.GetAllTasks()
    if err != nil {
        return 0, err
    }
    
    highest := 0
    re := regexp.MustCompile(`T(\d+)`)
    
    for _, t := range tasks {
        if m := re.FindStringSubmatch(t.ID); len(m) > 1 {
            if n, _ := strconv.Atoi(m[1]); n > highest {
                highest = n
            }
        }
    }
    
    return highest, nil
}

func (a *FeatureAnalyzer) GetNextIDs() (featureID int, taskID int, error) {
    fid, err := a.GetHighestFeatureID()
    if err != nil {
        return 0, 0, err
    }
    
    tid, err := a.GetHighestTaskID()
    if err != nil {
        return 0, 0, err
    }
    
    return fid + 1, tid + 1, nil
}
```

## Files to Create

| File | Description |
|------|-------------|
| `internal/analyzer/types.go` | Analysis types |
| `internal/analyzer/response.go` | Response analyzer |
| `internal/analyzer/feature.go` | Feature analyzer |
| `internal/analyzer/response_test.go` | Tests |

## Acceptance Criteria

- [ ] Parse HERMES_STATUS block
- [ ] Detect completion keywords
- [ ] Detect test-only loops
- [ ] Count errors
- [ ] Calculate confidence score
- [ ] Get highest feature/task IDs
- [ ] All tests pass
