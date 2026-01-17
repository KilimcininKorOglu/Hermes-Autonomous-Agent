# Phase 04: AI Provider Integration

## Goal

Implement AI provider abstraction using `github.com/severity1/claude-code-sdk-go` SDK.

## SDK Overview

The `claude-code-sdk-go` SDK provides programmatic access to Claude Code CLI:

- Executes Claude CLI as subprocess
- Handles streaming JSON output
- Manages permissions, tools, and sessions
- No API key needed - uses installed Claude CLI

## PowerShell Reference

```powershell
# From lib/AIProvider.ps1
# - Invoke-AIWithTimeout
# - Read-AIStreamOutput
# - Invoke-AIWithRetry
# - Invoke-TaskExecution
```

## Go Implementation

### 4.1 Provider Interface

```go
// internal/ai/provider.go
package ai

import (
    "context"
)

type Provider interface {
    Name() string
    IsAvailable() bool
    Execute(ctx context.Context, opts *ExecuteOptions) (*ExecuteResult, error)
    ExecuteStream(ctx context.Context, opts *ExecuteOptions) (<-chan StreamEvent, error)
}

type ExecuteOptions struct {
    Prompt       string
    WorkDir      string
    Tools        []string  // Allowed tools: "Read", "Write", "Bash", etc.
    MaxTurns     int
    SystemPrompt string
}

type ExecuteResult struct {
    Output      string
    Duration    float64
    Cost        float64
    TokensIn    int
    TokensOut   int
    Success     bool
    Error       string
}

type StreamEvent struct {
    Type     string  // "system", "assistant", "tool_use", "tool_result", "result", "error"
    Model    string
    Text     string
    ToolName string
    ToolID   string
    Cost     float64
    Duration float64
}

func GetProvider(name string) Provider {
    switch name {
    case "claude":
        return NewClaudeProvider()
    default:
        return nil
    }
}

func AutoDetectProvider() Provider {
    claude := NewClaudeProvider()
    if claude.IsAvailable() {
        return claude
    }
    return nil
}
```

### 4.2 Claude Provider (using SDK)

```go
// internal/ai/claude.go
package ai

import (
    "context"
    "os/exec"
    "time"

    claudecode "github.com/severity1/claude-code-sdk-go"
)

type ClaudeProvider struct{}

func NewClaudeProvider() *ClaudeProvider {
    return &ClaudeProvider{}
}

func (p *ClaudeProvider) Name() string {
    return "claude"
}

func (p *ClaudeProvider) IsAvailable() bool {
    _, err := exec.LookPath("claude")
    return err == nil
}

func (p *ClaudeProvider) Execute(ctx context.Context, opts *ExecuteOptions) (*ExecuteResult, error) {
    start := time.Now()

    // Build SDK options
    sdkOpts := []claudecode.Option{
        claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions),
    }

    if opts.WorkDir != "" {
        sdkOpts = append(sdkOpts, claudecode.WithCwd(opts.WorkDir))
    }

    if len(opts.Tools) > 0 {
        sdkOpts = append(sdkOpts, claudecode.WithAllowedTools(opts.Tools...))
    }

    if opts.MaxTurns > 0 {
        sdkOpts = append(sdkOpts, claudecode.WithMaxTurns(opts.MaxTurns))
    }

    if opts.SystemPrompt != "" {
        sdkOpts = append(sdkOpts, claudecode.WithSystemPrompt(opts.SystemPrompt))
    }

    // Execute query using SDK
    messages, err := claudecode.Query(ctx, opts.Prompt, sdkOpts...)
    if err != nil {
        return &ExecuteResult{
            Success: false,
            Error:   err.Error(),
        }, err
    }

    // Process messages to extract result
    result := &ExecuteResult{
        Duration: time.Since(start).Seconds(),
        Success:  true,
    }

    for _, msg := range messages {
        switch m := msg.(type) {
        case claudecode.AssistantMessage:
            for _, content := range m.Content {
                switch c := content.(type) {
                case claudecode.TextContent:
                    result.Output += c.Text
                }
            }
        case claudecode.ResultMessage:
            result.Output = m.Result
            result.Cost = m.TotalCostUSD
            result.Duration = float64(m.DurationMs) / 1000
        }
    }

    return result, nil
}

func (p *ClaudeProvider) ExecuteStream(ctx context.Context, opts *ExecuteOptions) (<-chan StreamEvent, error) {
    events := make(chan StreamEvent, 100)

    go func() {
        defer close(events)

        // Build SDK options
        sdkOpts := []claudecode.Option{
            claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions),
        }

        if opts.WorkDir != "" {
            sdkOpts = append(sdkOpts, claudecode.WithCwd(opts.WorkDir))
        }

        if len(opts.Tools) > 0 {
            sdkOpts = append(sdkOpts, claudecode.WithAllowedTools(opts.Tools...))
        }

        if opts.MaxTurns > 0 {
            sdkOpts = append(sdkOpts, claudecode.WithMaxTurns(opts.MaxTurns))
        }

        // Use WithClient for streaming
        err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
            msgChan, err := client.Query(ctx, opts.Prompt)
            if err != nil {
                return err
            }

            for msg := range msgChan {
                switch m := msg.(type) {
                case claudecode.SystemMessage:
                    events <- StreamEvent{
                        Type:  "system",
                        Model: m.Model,
                    }
                case claudecode.AssistantMessage:
                    for _, content := range m.Content {
                        switch c := content.(type) {
                        case claudecode.TextContent:
                            events <- StreamEvent{
                                Type: "assistant",
                                Text: c.Text,
                            }
                        case claudecode.ToolUseContent:
                            events <- StreamEvent{
                                Type:     "tool_use",
                                ToolName: c.Name,
                                ToolID:   c.ID,
                            }
                        }
                    }
                case claudecode.ToolResultMessage:
                    events <- StreamEvent{
                        Type:   "tool_result",
                        ToolID: m.ToolUseID,
                    }
                case claudecode.ResultMessage:
                    events <- StreamEvent{
                        Type:     "result",
                        Text:     m.Result,
                        Cost:     m.TotalCostUSD,
                        Duration: float64(m.DurationMs) / 1000,
                    }
                }
            }
            return nil
        }, sdkOpts...)

        if err != nil {
            events <- StreamEvent{
                Type: "error",
                Text: err.Error(),
            }
        }
    }()

    return events, nil
}
```

### 4.3 Task Executor

```go
// internal/ai/executor.go
package ai

import (
    "context"
    "fmt"
    "time"

    "hermes/internal/task"
)

type TaskExecutor struct {
    provider Provider
    workDir  string
}

func NewTaskExecutor(provider Provider, workDir string) *TaskExecutor {
    return &TaskExecutor{
        provider: provider,
        workDir:  workDir,
    }
}

func (e *TaskExecutor) ExecuteTask(ctx context.Context, t *task.Task, promptContent string) (*ExecuteResult, error) {
    // Build prompt with task context
    prompt := fmt.Sprintf(`%s

## Current Task: %s

**Task:** %s: %s

**Files to Touch:**
%s

**Success Criteria:**
%s

Complete this task and output the HERMES_STATUS block when done.`,
        promptContent,
        t.ID,
        t.ID, t.Name,
        formatFiles(t.FilesToTouch),
        formatCriteria(t.SuccessCriteria),
    )

    opts := &ExecuteOptions{
        Prompt:  prompt,
        WorkDir: e.workDir,
        Tools:   []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
    }

    return e.provider.Execute(ctx, opts)
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
```

### 4.4 Retry Logic

```go
// internal/ai/retry.go
package ai

import (
    "context"
    "time"
)

type RetryConfig struct {
    MaxRetries int
    Delay      time.Duration
    MaxDelay   time.Duration
}

func DefaultRetryConfig() *RetryConfig {
    return &RetryConfig{
        MaxRetries: 3,
        Delay:      5 * time.Second,
        MaxDelay:   60 * time.Second,
    }
}

func ExecuteWithRetry(ctx context.Context, provider Provider, opts *ExecuteOptions, cfg *RetryConfig) (*ExecuteResult, error) {
    if cfg == nil {
        cfg = DefaultRetryConfig()
    }

    var lastErr error
    delay := cfg.Delay

    for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
        result, err := provider.Execute(ctx, opts)
        if err == nil && result.Success {
            return result, nil
        }

        lastErr = err
        if result != nil && result.Error != "" {
            lastErr = fmt.Errorf("%s", result.Error)
        }

        // Don't wait after last attempt
        if attempt < cfg.MaxRetries {
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(delay):
            }

            // Exponential backoff
            delay = delay * 2
            if delay > cfg.MaxDelay {
                delay = cfg.MaxDelay
            }
        }
    }

    return nil, fmt.Errorf("failed after %d attempts: %w", cfg.MaxRetries, lastErr)
}
```

### 4.5 Stream Display

```go
// internal/ai/display.go
package ai

import (
    "fmt"

    "github.com/fatih/color"
)

var (
    systemColor  = color.New(color.FgHiBlack)
    textColor    = color.New(color.FgWhite)
    toolColor    = color.New(color.FgYellow)
    resultColor  = color.New(color.FgGreen)
    costColor    = color.New(color.FgCyan)
    errorColor   = color.New(color.FgRed)
)

type StreamDisplay struct {
    showTools bool
    showCost  bool
}

func NewStreamDisplay(showTools, showCost bool) *StreamDisplay {
    return &StreamDisplay{
        showTools: showTools,
        showCost:  showCost,
    }
}

func (d *StreamDisplay) Handle(event StreamEvent) {
    switch event.Type {
    case "system":
        systemColor.Printf("[Model: %s]\n", event.Model)

    case "assistant":
        textColor.Print(event.Text)

    case "tool_use":
        if d.showTools {
            toolColor.Printf("\n[Tool: %s]", event.ToolName)
        }

    case "tool_result":
        if d.showTools {
            toolColor.Print(" [Done]")
        }

    case "result":
        fmt.Println()
        if d.showCost {
            resultColor.Print("[Complete] ")
            costColor.Printf("%.1fs | $%.4f\n", event.Duration, event.Cost)
        }

    case "error":
        errorColor.Printf("\n[Error] %s\n", event.Text)
    }
}
```

## SDK Usage Examples

### Simple Query

```go
// One-shot query
messages, err := claudecode.Query(ctx, "Explain this code",
    claudecode.WithCwd("/project"),
    claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions),
)
```

### With Tools

```go
// Allow file operations
messages, err := claudecode.Query(ctx, "Refactor main.go",
    claudecode.WithAllowedTools("Read", "Write", "Edit"),
    claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions),
)
```

### Streaming with Client

```go
// Real-time streaming
err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
    msgChan, err := client.Query(ctx, "Build the feature")
    if err != nil {
        return err
    }

    for msg := range msgChan {
        // Process each message as it arrives
        fmt.Printf("Received: %T\n", msg)
    }
    return nil
}, claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions))
```

### Multi-turn Conversation

```go
// Maintain context across queries
err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
    // First query
    client.Query(ctx, "Read the codebase structure")

    // Follow-up with context
    client.Query(ctx, "Now implement the feature based on what you learned")

    return nil
})
```

## Files to Create

| File | Description |
|------|-------------|
| `internal/ai/provider.go` | Provider interface |
| `internal/ai/claude.go` | Claude SDK implementation |
| `internal/ai/executor.go` | Task executor |
| `internal/ai/retry.go` | Retry logic |
| `internal/ai/display.go` | Stream display |
| `internal/ai/provider_test.go` | Tests |

## Dependencies

```go
// go.mod
require github.com/severity1/claude-code-sdk-go v0.4.0
```

## Acceptance Criteria

- [ ] Claude provider executes prompts via SDK
- [ ] Stream output displays correctly
- [ ] Retry logic with exponential backoff works
- [ ] Task executor builds proper prompts
- [ ] IsAvailable() checks for claude CLI
- [ ] All tools (Read, Write, Edit, Bash, etc.) work
- [ ] All tests pass
