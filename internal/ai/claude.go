package ai

import (
	"context"
	"os/exec"
	"time"

	claudecode "github.com/severity1/claude-code-sdk-go"
)

// ClaudeProvider implements Provider using claude-code-sdk-go
type ClaudeProvider struct{}

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{}
}

// Name returns the provider name
func (p *ClaudeProvider) Name() string {
	return "claude"
}

// IsAvailable checks if Claude CLI is installed
func (p *ClaudeProvider) IsAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// Execute runs a prompt and returns the result
func (p *ClaudeProvider) Execute(ctx context.Context, opts *ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	// Build SDK options
	sdkOpts := p.buildOptions(opts)

	// Execute query using SDK - returns MessageIterator
	iter, err := claudecode.Query(ctx, opts.Prompt, sdkOpts...)
	if err != nil {
		return &ExecuteResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start).Seconds(),
		}, err
	}

	// Process messages to extract result
	result := &ExecuteResult{
		Duration: time.Since(start).Seconds(),
		Success:  true,
	}

	// Iterate through messages
	for {
		msg, err := iter.Next(ctx)
		if err != nil {
			break // No more messages or error
		}
		p.processMessage(msg, result)
	}

	return result, nil
}

// ExecuteStream runs a prompt with streaming output
func (p *ClaudeProvider) ExecuteStream(ctx context.Context, opts *ExecuteOptions) (<-chan StreamEvent, error) {
	events := make(chan StreamEvent, 100)

	go func() {
		defer close(events)

		sdkOpts := p.buildOptions(opts)

		// Use WithClient for streaming
		err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
			// Send query
			if err := client.Query(ctx, opts.Prompt); err != nil {
				return err
			}

			// Receive response
			iter := client.ReceiveResponse(ctx)
			for {
				msg, err := iter.Next(ctx)
				if err != nil {
					break
				}
				p.processStreamMessage(msg, events)
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

func (p *ClaudeProvider) buildOptions(opts *ExecuteOptions) []claudecode.Option {
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

	return sdkOpts
}

func (p *ClaudeProvider) processMessage(msg claudecode.Message, result *ExecuteResult) {
	switch m := msg.(type) {
	case *claudecode.AssistantMessage:
		for _, block := range m.Content {
			if tb, ok := block.(*claudecode.TextBlock); ok {
				result.Output += tb.Text
			}
		}
	case *claudecode.ResultMessage:
		if m.Result != nil {
			result.Output = *m.Result
		}
		if m.TotalCostUSD != nil {
			result.Cost = *m.TotalCostUSD
		}
		result.Duration = float64(m.DurationMs) / 1000
	}
}

func (p *ClaudeProvider) processStreamMessage(msg claudecode.Message, events chan<- StreamEvent) {
	switch m := msg.(type) {
	case *claudecode.SystemMessage:
		events <- StreamEvent{
			Type:  "system",
			Model: m.Subtype,
		}
	case *claudecode.AssistantMessage:
		for _, block := range m.Content {
			switch b := block.(type) {
			case *claudecode.TextBlock:
				events <- StreamEvent{
					Type: "text",
					Text: b.Text,
				}
			case *claudecode.ToolUseBlock:
				events <- StreamEvent{
					Type:     "tool_use",
					ToolName: b.Name,
				}
			}
		}
	case *claudecode.ResultMessage:
		var text string
		var cost float64
		if m.Result != nil {
			text = *m.Result
		}
		if m.TotalCostUSD != nil {
			cost = *m.TotalCostUSD
		}
		events <- StreamEvent{
			Type:     "result",
			Text:     text,
			Cost:     cost,
			Duration: float64(m.DurationMs) / 1000,
		}
	}
}
