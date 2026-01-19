package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// OpenCodeProvider implements Provider using OpenCode CLI
type OpenCodeProvider struct{}

// NewOpenCodeProvider creates a new OpenCode provider
func NewOpenCodeProvider() *OpenCodeProvider {
	return &OpenCodeProvider{}
}

// Name returns the provider name
func (p *OpenCodeProvider) Name() string {
	return "opencode"
}

// IsAvailable checks if OpenCode CLI is installed
func (p *OpenCodeProvider) IsAvailable() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}

// openCodeStreamEvent represents a JSON event from opencode stream output
// OpenCode event types: text, tool_use, tool_result, step_start, step_finish, error
type openCodeStreamEvent struct {
	Type string `json:"type"`
	Part struct {
		Text  string `json:"text,omitempty"`
		Tool  string `json:"tool,omitempty"`
		Name  string `json:"name,omitempty"`
		State struct {
			Input   map[string]interface{} `json:"input,omitempty"`
			Content string                 `json:"content,omitempty"`
			IsError bool                   `json:"isError,omitempty"`
			Error   string                 `json:"error,omitempty"`
		} `json:"state,omitempty"`
	} `json:"part,omitempty"`
	Error struct {
		Message string `json:"message,omitempty"`
	} `json:"error,omitempty"`
}

// Execute runs a prompt and returns the result
func (p *OpenCodeProvider) Execute(ctx context.Context, opts *ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	// Build command: opencode run --format json "<prompt>"
	args := []string{"run", "--format", "json"}

	// Add prompt as positional argument
	args = append(args, opts.Prompt)

	cmd := exec.CommandContext(ctx, "opencode", args...)

	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start opencode: %w", err)
	}

	result := &ExecuteResult{
		Success: true,
	}

	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		var event openCodeStreamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		switch event.Type {
		case "text":
			if event.Part.Text != "" {
				result.Output += event.Part.Text
			}
		case "error":
			if event.Error.Message != "" {
				result.Error = event.Error.Message
				result.Success = false
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		result.Success = false
		if result.Error == "" {
			result.Error = err.Error()
		}
	}

	result.Duration = time.Since(start).Seconds()

	return result, nil
}

// ExecuteStream runs a prompt with streaming output
func (p *OpenCodeProvider) ExecuteStream(ctx context.Context, opts *ExecuteOptions) (<-chan StreamEvent, error) {
	events := make(chan StreamEvent, 100)

	go func() {
		defer close(events)

		// Build command: opencode run --format json "<prompt>"
		args := []string{"run", "--format", "json"}
		args = append(args, opts.Prompt)

		cmd := exec.CommandContext(ctx, "opencode", args...)

		if opts.WorkDir != "" {
			cmd.Dir = opts.WorkDir
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

		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			var ocEvent openCodeStreamEvent
			if err := json.Unmarshal([]byte(line), &ocEvent); err != nil {
				continue
			}

			switch ocEvent.Type {
			case "text":
				if ocEvent.Part.Text != "" {
					events <- StreamEvent{
						Type: "text",
						Text: ocEvent.Part.Text,
					}
				}
			case "tool_use":
				toolName := ocEvent.Part.Tool
				if toolName == "" {
					toolName = ocEvent.Part.Name
				}
				events <- StreamEvent{
					Type:      "tool_use",
					ToolName:  toolName,
					ToolInput: ocEvent.Part.State.Input,
				}
			case "tool_result":
				if ocEvent.Part.State.IsError {
					events <- StreamEvent{
						Type:      "tool_result",
						ToolError: ocEvent.Part.State.Error,
					}
				} else {
					events <- StreamEvent{
						Type:       "tool_result",
						ToolOutput: ocEvent.Part.State.Content,
					}
				}
			case "error":
				events <- StreamEvent{
					Type: "error",
					Text: ocEvent.Error.Message,
				}
			}
		}

		if err := cmd.Wait(); err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
		}
	}()

	return events, nil
}
