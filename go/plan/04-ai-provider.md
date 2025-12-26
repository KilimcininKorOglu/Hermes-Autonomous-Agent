# Phase 04: AI Provider Integration

## Goal

Implement AI provider abstraction using `claude-code-sdk-go` for Claude and direct CLI for fallback.

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
    ExecuteStream(ctx context.Context, opts *ExecuteOptions, handler StreamHandler) error
}

type ExecuteOptions struct {
    Prompt       string
    Content      string
    Timeout      int
    MaxRetries   int
    StreamOutput bool
    WorkDir      string
}

type ExecuteResult struct {
    Output   string
    Duration float64
    Cost     float64
    Success  bool
    Error    string
}

type StreamHandler func(event StreamEvent)

type StreamEvent struct {
    Type    string // "init", "text", "tool", "tool_done", "result", "error"
    Model   string
    Text    string
    Tool    string
    Cost    float64
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
    // Try claude first
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
    
    // Build options for SDK
    sdkOpts := []claudecode.Option{
        claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions),
    }
    
    if opts.WorkDir != "" {
        sdkOpts = append(sdkOpts, claudecode.WithCwd(opts.WorkDir))
    }
    
    // Build prompt with content
    prompt := opts.Prompt
    if opts.Content != "" {
        prompt = opts.Prompt + "\n\n---\nCONTENT:\n---\n\n" + opts.Content
    }
    
    // Execute query
    result, err := claudecode.Query(ctx, prompt, sdkOpts...)
    if err != nil {
        return &ExecuteResult{
            Success: false,
            Error:   err.Error(),
        }, err
    }
    
    // Collect output from messages
    var output string
    for _, msg := range result {
        if textMsg, ok := msg.(*claudecode.AssistantMessage); ok {
            for _, content := range textMsg.Content {
                if textContent, ok := content.(*claudecode.TextContent); ok {
                    output += textContent.Text
                }
            }
        }
        if resultMsg, ok := msg.(*claudecode.ResultMessage); ok {
            output = resultMsg.Result
        }
    }
    
    return &ExecuteResult{
        Output:   output,
        Duration: time.Since(start).Seconds(),
        Success:  true,
    }, nil
}

func (p *ClaudeProvider) ExecuteStream(ctx context.Context, opts *ExecuteOptions, handler StreamHandler) error {
    // Build options for SDK
    sdkOpts := []claudecode.Option{
        claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions),
    }
    
    if opts.WorkDir != "" {
        sdkOpts = append(sdkOpts, claudecode.WithCwd(opts.WorkDir))
    }
    
    // Build prompt with content
    prompt := opts.Prompt
    if opts.Content != "" {
        prompt = opts.Prompt + "\n\n---\nCONTENT:\n---\n\n" + opts.Content
    }
    
    // Execute with streaming
    messages, err := claudecode.Query(ctx, prompt, sdkOpts...)
    if err != nil {
        handler(StreamEvent{Type: "error", Text: err.Error()})
        return err
    }
    
    // Process messages
    for _, msg := range messages {
        switch m := msg.(type) {
        case *claudecode.SystemMessage:
            handler(StreamEvent{
                Type:  "init",
                Model: m.Model,
            })
        case *claudecode.AssistantMessage:
            for _, content := range m.Content {
                if textContent, ok := content.(*claudecode.TextContent); ok {
                    handler(StreamEvent{
                        Type: "text",
                        Text: textContent.Text,
                    })
                }
                if toolContent, ok := content.(*claudecode.ToolUseContent); ok {
                    handler(StreamEvent{
                        Type: "tool",
                        Tool: toolContent.Name,
                    })
                }
            }
        case *claudecode.ToolResultMessage:
            handler(StreamEvent{
                Type: "tool_done",
            })
        case *claudecode.ResultMessage:
            handler(StreamEvent{
                Type:     "result",
                Text:     m.Result,
                Cost:     m.TotalCostUSD,
                Duration: float64(m.DurationMs) / 1000,
            })
        }
    }
    
    return nil
}
```

### 4.3 Retry Logic

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
}

func ExecuteWithRetry(ctx context.Context, provider Provider, opts *ExecuteOptions, retryConfig *RetryConfig) (*ExecuteResult, error) {
    var lastErr error
    
    for attempt := 1; attempt <= retryConfig.MaxRetries; attempt++ {
        result, err := provider.Execute(ctx, opts)
        if err == nil && result.Success {
            return result, nil
        }
        
        lastErr = err
        
        if attempt < retryConfig.MaxRetries {
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(retryConfig.Delay):
            }
        }
    }
    
    return nil, lastErr
}
```

### 4.4 Stream Output Display

```go
// internal/ai/stream.go
package ai

import (
    "fmt"
    "github.com/fatih/color"
)

var (
    initColor     = color.New(color.FgHiBlack)
    textColor     = color.New(color.FgWhite)
    toolColor     = color.New(color.FgYellow)
    toolDoneColor = color.New(color.FgHiYellow)
    resultColor   = color.New(color.FgGreen)
    costColor     = color.New(color.FgCyan)
    errorColor    = color.New(color.FgRed)
)

func DefaultStreamHandler(event StreamEvent) {
    switch event.Type {
    case "init":
        initColor.Printf("[Session] Model: %s\n", event.Model)
    case "text":
        textColor.Print(event.Text)
    case "tool":
        toolColor.Printf("\n[Tool: %s]", event.Tool)
    case "tool_done":
        toolDoneColor.Print("[Done]")
    case "result":
        fmt.Println()
        resultColor.Print("[Complete] ")
        costColor.Printf("Duration: %.1fs | Cost: $%.4f\n", event.Duration, event.Cost)
    case "error":
        errorColor.Printf("\n[Error] %s\n", event.Text)
    }
}
```

## Files to Create

| File | Description |
|------|-------------|
| `internal/ai/provider.go` | Provider interface |
| `internal/ai/claude.go` | Claude implementation |
| `internal/ai/retry.go` | Retry logic |
| `internal/ai/stream.go` | Stream output display |
| `internal/ai/provider_test.go` | Tests |

## Acceptance Criteria

- [ ] Claude provider executes prompts
- [ ] Stream output displays correctly
- [ ] Retry logic works
- [ ] Timeout handling works
- [ ] IsAvailable() returns correct value
- [ ] All tests pass
