package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// ClaudeProvider implements Provider using Claude CLI
type ClaudeProvider struct{}

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{}
}

// Name returns the provider name
func (p *ClaudeProvider) Name() string {
	return "Claude"
}

// IsAvailable checks if Claude CLI is installed
func (p *ClaudeProvider) IsAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// claudeStreamEvent represents a JSON event from claude stream output
// Claude CLI stream-json format event types:
// - "assistant": AI responses with content[] containing text and tool_use blocks
// - "user": Tool results
// - "system": Hooks, init data
// - "result": Final result summary
type claudeStreamEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`
	Message struct {
		Role    string `json:"role,omitempty"`
		Content []struct {
			Type      string                 `json:"type"`
			Text      string                 `json:"text,omitempty"`
			ID        string                 `json:"id,omitempty"`
			Name      string                 `json:"name,omitempty"`
			Input     map[string]interface{} `json:"input,omitempty"`
			ToolUseID string                 `json:"tool_use_id,omitempty"`
			Content   string                 `json:"content,omitempty"`
			IsError   bool                   `json:"is_error,omitempty"`
		} `json:"content,omitempty"`
	} `json:"message,omitempty"`
	CostUSD    float64 `json:"cost_usd,omitempty"`
	DurationMs int64   `json:"duration_ms,omitempty"`
	Result     string  `json:"result,omitempty"`
}

// Execute runs a prompt and returns the result
func (p *ClaudeProvider) Execute(ctx context.Context, opts *ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	// Build command args
	args := []string{"--print", "--verbose", "--output-format", "stream-json", "--dangerously-skip-permissions"}

	cmd := exec.CommandContext(ctx, "claude", args...)

	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	// Get stdin pipe to send prompt
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	// Send prompt via stdin
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, opts.Prompt)
	}()

	result := &ExecuteResult{
		Success: true,
	}

	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		var event claudeStreamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		switch event.Type {
		case "assistant":
			for _, content := range event.Message.Content {
				if content.Type == "text" && content.Text != "" {
					result.Output += content.Text
				}
			}
		case "result":
			if event.Result != "" {
				result.Output = event.Result
			}
			result.Cost = event.CostUSD
			result.Duration = float64(event.DurationMs) / 1000
		}
	}

	if err := cmd.Wait(); err != nil {
		result.Success = false
		result.Error = err.Error()
	}

	if result.Duration == 0 {
		result.Duration = time.Since(start).Seconds()
	}

	return result, nil
}

// ExecuteStream runs a prompt with streaming output
func (p *ClaudeProvider) ExecuteStream(ctx context.Context, opts *ExecuteOptions) (<-chan StreamEvent, error) {
	events := make(chan StreamEvent, 100)

	go func() {
		defer close(events)

		// Build command args
		args := []string{"--print", "--verbose", "--output-format", "stream-json", "--dangerously-skip-permissions"}

		cmd := exec.CommandContext(ctx, "claude", args...)

		if opts.WorkDir != "" {
			cmd.Dir = opts.WorkDir
		}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
			return
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
			return
		}

		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
			return
		}

		// Send prompt via stdin
		go func() {
			defer stdin.Close()
			io.WriteString(stdin, opts.Prompt)
		}()

		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			var cEvent claudeStreamEvent
			if err := json.Unmarshal([]byte(line), &cEvent); err != nil {
				continue
			}

			switch cEvent.Type {
			case "system":
				events <- StreamEvent{
					Type:  "system",
					Model: cEvent.Subtype,
				}
			case "assistant":
				for _, content := range cEvent.Message.Content {
					switch content.Type {
					case "text":
						if content.Text != "" {
							events <- StreamEvent{
								Type: "text",
								Text: content.Text,
							}
						}
					case "tool_use":
						events <- StreamEvent{
							Type:      "tool_use",
							ToolName:  content.Name,
							ToolID:    content.ID,
							ToolInput: content.Input,
						}
					}
				}
			case "user":
				// Tool results come in user messages
				for _, content := range cEvent.Message.Content {
					if content.Type == "tool_result" {
						events <- StreamEvent{
							Type:       "tool_result",
							ToolID:     content.ToolUseID,
							ToolOutput: content.Content,
							ToolError:  func() string { if content.IsError { return content.Content } else { return "" } }(),
						}
					}
				}
			case "result":
				events <- StreamEvent{
					Type:     "result",
					Text:     cEvent.Result,
					Cost:     cEvent.CostUSD,
					Duration: float64(cEvent.DurationMs) / 1000,
				}
			}
		}

		if err := cmd.Wait(); err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
		}
	}()

	return events, nil
}
