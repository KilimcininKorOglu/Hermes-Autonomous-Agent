package ai

import (
	"context"
	"time"
)

// timeNow is a variable to allow mocking in tests
var timeNow = time.Now

// Provider defines the interface for AI providers
type Provider interface {
	Name() string
	IsAvailable() bool
	Execute(ctx context.Context, opts *ExecuteOptions) (*ExecuteResult, error)
	ExecuteStream(ctx context.Context, opts *ExecuteOptions) (<-chan StreamEvent, error)
}

// ExecuteOptions contains options for AI execution
type ExecuteOptions struct {
	Prompt       string
	WorkDir      string
	Tools        []string // Allowed tools: "Read", "Write", "Bash", etc.
	MaxTurns     int
	SystemPrompt string
	Timeout      int  // Timeout in seconds
	StreamOutput bool // Enable streaming
}

// ExecuteResult contains the result of AI execution
type ExecuteResult struct {
	Output    string
	Duration  float64
	Cost      float64
	TokensIn  int
	TokensOut int
	Success   bool
	Error     string
}

// StreamEvent represents a streaming event from AI
type StreamEvent struct {
	Type       string                 // "system", "text", "tool_use", "tool_result", "result", "error"
	Model      string                 // Model name (for system events)
	Text       string                 // Text content
	ToolName   string                 // Tool name (for tool_use/tool_result)
	ToolID     string                 // Tool call ID
	ToolInput  map[string]interface{} // Tool input parameters (for tool_use)
	ToolOutput string                 // Tool output (for tool_result)
	ToolError  string                 // Tool error message (for tool_result with error)
	Cost       float64                // Cost in USD
	Duration   float64                // Duration in seconds
}

// ToolTrace represents a single tool call trace
type ToolTrace struct {
	Name      string                 `json:"name"`
	ID        string                 `json:"id,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    string                 `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	StartTime int64                  `json:"startTime"`
	EndTime   int64                  `json:"endTime,omitempty"`
}

// SubagentTracer collects tool traces during AI execution
type SubagentTracer struct {
	Traces      []ToolTrace `json:"traces"`
	pendingTool *ToolTrace
}

// NewSubagentTracer creates a new tracer
func NewSubagentTracer() *SubagentTracer {
	return &SubagentTracer{
		Traces: make([]ToolTrace, 0),
	}
}

// ProcessEvent processes a stream event and updates traces
func (t *SubagentTracer) ProcessEvent(event StreamEvent) {
	switch event.Type {
	case "tool_use":
		// Start a new tool trace
		t.pendingTool = &ToolTrace{
			Name:      event.ToolName,
			ID:        event.ToolID,
			Input:     event.ToolInput,
			StartTime: timeNowUnix(),
		}
	case "tool_result":
		if t.pendingTool != nil {
			t.pendingTool.Output = event.ToolOutput
			t.pendingTool.Error = event.ToolError
			t.pendingTool.EndTime = timeNowUnix()
			t.Traces = append(t.Traces, *t.pendingTool)
			t.pendingTool = nil
		}
	}
}

// GetTraces returns all collected tool traces
func (t *SubagentTracer) GetTraces() []ToolTrace {
	return t.Traces
}

// timeNowUnix returns current unix timestamp in milliseconds
func timeNowUnix() int64 {
	return timeNow().UnixMilli()
}

// GetProvider returns a provider by name
func GetProvider(name string) Provider {
	switch name {
	case "claude":
		return NewClaudeProvider()
	case "droid":
		return NewDroidProvider()
	case "gemini":
		return NewGeminiProvider()
	case "opencode":
		return NewOpenCodeProvider()
	default:
		return nil
	}
}

// AutoDetectProvider finds an available provider (priority: claude > droid > opencode > gemini)
func AutoDetectProvider() Provider {
	claude := NewClaudeProvider()
	if claude.IsAvailable() {
		return claude
	}

	droid := NewDroidProvider()
	if droid.IsAvailable() {
		return droid
	}

	opencode := NewOpenCodeProvider()
	if opencode.IsAvailable() {
		return opencode
	}

	gemini := NewGeminiProvider()
	if gemini.IsAvailable() {
		return gemini
	}

	return nil
}

// GetAvailableProviders returns a list of available provider names
func GetAvailableProviders() []string {
	var providers []string

	if NewClaudeProvider().IsAvailable() {
		providers = append(providers, "claude")
	}
	if NewDroidProvider().IsAvailable() {
		providers = append(providers, "droid")
	}
	if NewOpenCodeProvider().IsAvailable() {
		providers = append(providers, "opencode")
	}
	if NewGeminiProvider().IsAvailable() {
		providers = append(providers, "gemini")
	}

	return providers
}
