package ai

import (
	"context"
)

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
	Type     string  // "system", "assistant", "tool_use", "tool_result", "result", "error"
	Model    string
	Text     string
	ToolName string
	ToolID   string
	Cost     float64
	Duration float64
}

// GetProvider returns a provider by name
func GetProvider(name string) Provider {
	switch name {
	case "claude":
		return NewClaudeProvider()
	case "droid":
		return NewDroidProvider()
	default:
		return nil
	}
}

// AutoDetectProvider finds an available provider (priority: claude > droid)
func AutoDetectProvider() Provider {
	claude := NewClaudeProvider()
	if claude.IsAvailable() {
		return claude
	}

	droid := NewDroidProvider()
	if droid.IsAvailable() {
		return droid
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

	return providers
}
