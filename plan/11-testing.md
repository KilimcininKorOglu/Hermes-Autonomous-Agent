# Phase 11: Testing

## Goal

Comprehensive unit and integration tests for all packages.

## Testing Strategy

### Unit Tests

Each package gets `*_test.go` files with:

- Table-driven tests
- Mock implementations where needed
- Edge case coverage

### Integration Tests

End-to-end tests that:

- Use real file system (temp directories)
- Test full workflows
- Verify CLI output

## Test Files

### 11.1 Config Tests

```go
// internal/config/config_test.go
package config

import (
    "os"
    "path/filepath"
    "testing"
)

func TestDefaultConfig(t *testing.T) {
    cfg := DefaultConfig()
    
    if cfg.AI.Planning != "claude" {
        t.Errorf("expected planning=claude, got %s", cfg.AI.Planning)
    }
    if cfg.AI.Coding != "droid" {
        t.Errorf("expected coding=droid, got %s", cfg.AI.Coding)
    }
    if cfg.AI.Timeout != 300 {
        t.Errorf("expected timeout=300, got %d", cfg.AI.Timeout)
    }
}

func TestLoadConfig(t *testing.T) {
    // Create temp directory
    tmpDir, err := os.MkdirTemp("", "hermes-test-*")
    if err != nil {
        t.Fatal(err)
    }
    defer os.RemoveAll(tmpDir)
    
    // Create .hermes directory
    hermesDir := filepath.Join(tmpDir, ".hermes")
    os.MkdirAll(hermesDir, 0755)
    
    // Create config file
    configContent := `{"ai": {"timeout": 600}}`
    os.WriteFile(filepath.Join(hermesDir, "config.json"), []byte(configContent), 0644)
    
    // Load and verify
    cfg, err := Load(tmpDir)
    if err != nil {
        t.Fatal(err)
    }
    
    if cfg.AI.Timeout != 600 {
        t.Errorf("expected timeout=600, got %d", cfg.AI.Timeout)
    }
    // Default values should still be present
    if cfg.AI.Planning != "claude" {
        t.Errorf("expected planning=claude, got %s", cfg.AI.Planning)
    }
}

func TestGetAIForTask(t *testing.T) {
    cfg := DefaultConfig()
    
    tests := []struct {
        taskType string
        override string
        expected string
    }{
        {"planning", "", "claude"},
        {"coding", "", "droid"},
        {"planning", "droid", "droid"},
        {"coding", "claude", "claude"},
        {"planning", "auto", "claude"},
    }
    
    for _, tt := range tests {
        result := GetAIForTask(tt.taskType, tt.override, cfg)
        if result != tt.expected {
            t.Errorf("GetAIForTask(%s, %s) = %s, want %s",
                tt.taskType, tt.override, result, tt.expected)
        }
    }
}
```

### 11.2 Task Tests

```go
// internal/task/reader_test.go
package task

import (
    "os"
    "path/filepath"
    "testing"
)

func TestParseFeature(t *testing.T) {
    content := `# Feature 1: User Authentication
**Feature ID:** F001
**Status:** NOT_STARTED

### T001: Create login endpoint
**Status:** NOT_STARTED
**Priority:** P1
**Files to Touch:** api/auth.go, handlers/login.go
**Dependencies:** None
**Success Criteria:**
- Endpoint accepts username/password
- Returns JWT token on success

### T002: Add password hashing
**Status:** NOT_STARTED
**Priority:** P1
**Files to Touch:** utils/crypto.go
**Dependencies:** T001
**Success Criteria:**
- Use bcrypt for hashing
`

    feature, err := parseFeature(content, "001-user-auth.md")
    if err != nil {
        t.Fatal(err)
    }
    
    if feature.ID != "F001" {
        t.Errorf("expected ID=F001, got %s", feature.ID)
    }
    if feature.Name != "User Authentication" {
        t.Errorf("expected name='User Authentication', got %s", feature.Name)
    }
    if len(feature.Tasks) != 2 {
        t.Errorf("expected 2 tasks, got %d", len(feature.Tasks))
    }
    
    task1 := feature.Tasks[0]
    if task1.ID != "T001" {
        t.Errorf("expected task ID=T001, got %s", task1.ID)
    }
    if task1.Priority != PriorityP1 {
        t.Errorf("expected priority=P1, got %s", task1.Priority)
    }
    if len(task1.FilesToTouch) != 2 {
        t.Errorf("expected 2 files, got %d", len(task1.FilesToTouch))
    }
    if len(task1.SuccessCriteria) != 2 {
        t.Errorf("expected 2 criteria, got %d", len(task1.SuccessCriteria))
    }
    
    task2 := feature.Tasks[1]
    if len(task2.Dependencies) != 1 || task2.Dependencies[0] != "T001" {
        t.Errorf("expected dependency=T001, got %v", task2.Dependencies)
    }
}

func TestGetProgress(t *testing.T) {
    // Create temp directory with task files
    tmpDir, _ := os.MkdirTemp("", "hermes-test-*")
    defer os.RemoveAll(tmpDir)
    
    tasksDir := filepath.Join(tmpDir, ".hermes", "tasks")
    os.MkdirAll(tasksDir, 0755)
    
    content := `# Feature 1: Test
**Feature ID:** F001
**Status:** IN_PROGRESS

### T001: Task 1
**Status:** COMPLETED
**Priority:** P1

### T002: Task 2
**Status:** IN_PROGRESS
**Priority:** P1

### T003: Task 3
**Status:** NOT_STARTED
**Priority:** P2
`
    os.WriteFile(filepath.Join(tasksDir, "001-test.md"), []byte(content), 0644)
    
    reader := NewReader(tmpDir)
    progress, err := reader.GetProgress()
    if err != nil {
        t.Fatal(err)
    }
    
    if progress.Total != 3 {
        t.Errorf("expected total=3, got %d", progress.Total)
    }
    if progress.Completed != 1 {
        t.Errorf("expected completed=1, got %d", progress.Completed)
    }
    if progress.InProgress != 1 {
        t.Errorf("expected inProgress=1, got %d", progress.InProgress)
    }
    if progress.NotStarted != 1 {
        t.Errorf("expected notStarted=1, got %d", progress.NotStarted)
    }
}
```

### 11.3 Circuit Breaker Tests

```go
// internal/circuit/breaker_test.go
package circuit

import (
    "os"
    "testing"
)

func TestCircuitBreakerStates(t *testing.T) {
    tmpDir, _ := os.MkdirTemp("", "hermes-test-*")
    defer os.RemoveAll(tmpDir)
    
    breaker := New(tmpDir)
    breaker.Initialize()
    
    // Initial state should be CLOSED
    state, _ := breaker.GetState()
    if state.State != StateClosed {
        t.Errorf("expected CLOSED, got %s", state.State)
    }
    
    // Add no-progress results
    breaker.AddLoopResult(false, false, 1)
    state, _ = breaker.GetState()
    if state.State != StateClosed {
        t.Errorf("after 1 no-progress: expected CLOSED, got %s", state.State)
    }
    
    breaker.AddLoopResult(false, false, 2)
    state, _ = breaker.GetState()
    if state.State != StateHalfOpen {
        t.Errorf("after 2 no-progress: expected HALF_OPEN, got %s", state.State)
    }
    
    breaker.AddLoopResult(false, false, 3)
    state, _ = breaker.GetState()
    if state.State != StateOpen {
        t.Errorf("after 3 no-progress: expected OPEN, got %s", state.State)
    }
    
    // Can't execute when OPEN
    canExecute, _ := breaker.CanExecute()
    if canExecute {
        t.Error("should not be able to execute when OPEN")
    }
}

func TestCircuitBreakerRecovery(t *testing.T) {
    tmpDir, _ := os.MkdirTemp("", "hermes-test-*")
    defer os.RemoveAll(tmpDir)
    
    breaker := New(tmpDir)
    breaker.Initialize()
    
    // Get to HALF_OPEN
    breaker.AddLoopResult(false, false, 1)
    breaker.AddLoopResult(false, false, 2)
    
    state, _ := breaker.GetState()
    if state.State != StateHalfOpen {
        t.Fatalf("expected HALF_OPEN, got %s", state.State)
    }
    
    // Progress should recover to CLOSED
    breaker.AddLoopResult(true, false, 3)
    state, _ = breaker.GetState()
    if state.State != StateClosed {
        t.Errorf("after progress: expected CLOSED, got %s", state.State)
    }
}

func TestCircuitBreakerReset(t *testing.T) {
    tmpDir, _ := os.MkdirTemp("", "hermes-test-*")
    defer os.RemoveAll(tmpDir)
    
    breaker := New(tmpDir)
    breaker.Initialize()
    
    // Get to OPEN
    breaker.AddLoopResult(false, false, 1)
    breaker.AddLoopResult(false, false, 2)
    breaker.AddLoopResult(false, false, 3)
    
    // Reset
    breaker.Reset("Test reset")
    
    state, _ := breaker.GetState()
    if state.State != StateClosed {
        t.Errorf("after reset: expected CLOSED, got %s", state.State)
    }
}
```

### 11.4 Git Tests

```go
// internal/git/git_test.go
package git

import (
    "testing"
)

func TestSanitizeBranchName(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"User Authentication", "user-authentication"},
        {"Add API endpoint", "add-api-endpoint"},
        {"Fix bug #123", "fix-bug-123"},
        {"Feature: New UI", "feature-new-ui"},
        {"Multiple   Spaces", "multiple-spaces"},
    }
    
    for _, tt := range tests {
        result := sanitizeBranchName(tt.input)
        if result != tt.expected {
            t.Errorf("sanitizeBranchName(%q) = %q, want %q",
                tt.input, result, tt.expected)
        }
    }
}

func TestGetFeatureBranchName(t *testing.T) {
    g := New(".")
    
    name := g.GetFeatureBranchName("F001", "User Authentication")
    expected := "feature/F001-user-authentication"
    
    if name != expected {
        t.Errorf("got %q, want %q", name, expected)
    }
}
```

### 11.5 Analyzer Tests

```go
// internal/analyzer/response_test.go
package analyzer

import (
    "testing"
)

func TestAnalyzeHermesStatus(t *testing.T) {
    output := `
Some work was done here.

---HERMES_STATUS---
STATUS: COMPLETE
EXIT_SIGNAL: true
RECOMMENDATION: Move to next task
---END_HERMES_STATUS---
`

    analyzer := NewResponseAnalyzer()
    result := analyzer.Analyze(output)
    
    if !result.ExitSignal {
        t.Error("expected ExitSignal=true")
    }
    if result.Status != "COMPLETE" {
        t.Errorf("expected Status=COMPLETE, got %s", result.Status)
    }
    if result.Confidence != 1.0 {
        t.Errorf("expected Confidence=1.0, got %f", result.Confidence)
    }
}

func TestAnalyzeCompletionKeywords(t *testing.T) {
    tests := []struct {
        output   string
        complete bool
    }{
        {"The task is done.", true},
        {"Implementation complete.", true},
        {"All tasks finished.", true},
        {"Still working on it.", false},
        {"In progress...", false},
    }
    
    analyzer := NewResponseAnalyzer()
    
    for _, tt := range tests {
        result := analyzer.Analyze(tt.output)
        if result.IsComplete != tt.complete {
            t.Errorf("Analyze(%q).IsComplete = %v, want %v",
                tt.output, result.IsComplete, tt.complete)
        }
    }
}

func TestAnalyzeTestOnly(t *testing.T) {
    testOnly := `Running npm test...
All tests passed.
`
    
    withImpl := `Running npm test...
Created new file: src/app.js
All tests passed.
`
    
    analyzer := NewResponseAnalyzer()
    
    result1 := analyzer.Analyze(testOnly)
    if !result1.IsTestOnly {
        t.Error("expected IsTestOnly=true for test-only output")
    }
    
    result2 := analyzer.Analyze(withImpl)
    if result2.IsTestOnly {
        t.Error("expected IsTestOnly=false when implementation present")
    }
}
```

## Running Tests

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test -v ./internal/task/...

# Run with race detector
go test -race ./...
```

## Files to Create

| File | Description |
|------|-------------|
| `internal/config/config_test.go` | Config tests |
| `internal/task/reader_test.go` | Task reader tests |
| `internal/task/parser_test.go` | Parser tests |
| `internal/circuit/breaker_test.go` | Circuit breaker tests |
| `internal/git/git_test.go` | Git tests |
| `internal/analyzer/response_test.go` | Analyzer tests |
| `internal/prompt/injector_test.go` | Prompt tests |

## Acceptance Criteria

- [ ] All tests pass
- [ ] Coverage > 80%
- [ ] No race conditions
- [ ] Edge cases covered
- [ ] Integration tests work
