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

// DroidProvider implements Provider using Factory Droid CLI
type DroidProvider struct{}

// NewDroidProvider creates a new Droid provider
func NewDroidProvider() *DroidProvider {
	return &DroidProvider{}
}

// Name returns the provider name
func (p *DroidProvider) Name() string {
	return "droid"
}

// IsAvailable checks if Droid CLI is installed
func (p *DroidProvider) IsAvailable() bool {
	_, err := exec.LookPath("droid")
	return err == nil
}

// droidStreamEvent represents a JSON event from droid stream output
type droidStreamEvent struct {
	Type       string                 `json:"type"`
	Subtype    string                 `json:"subtype,omitempty"`
	Model      string                 `json:"model,omitempty"`
	Role       string                 `json:"role,omitempty"`
	Text       string                 `json:"text,omitempty"`
	ToolName   string                 `json:"toolName,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	DurationMs int64                  `json:"durationMs,omitempty"`
	NumTurns   int                    `json:"numTurns,omitempty"`
	FinalText  string                 `json:"finalText,omitempty"`
}

// Execute runs a prompt and returns the result
func (p *DroidProvider) Execute(ctx context.Context, opts *ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	// Write prompt to temp file
	tmpFile, err := os.CreateTemp("", "hermes-droid-*.md")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(opts.Prompt); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write prompt: %w", err)
	}
	tmpFile.Close()

	// Build command
	args := []string{"exec", "--skip-permissions-unsafe", "--file", tmpFile.Name()}

	// Add output format for parsing
	args = append(args, "--output-format", "stream-json")

	cmd := exec.CommandContext(ctx, "droid", args...)

	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Redirect stderr to prevent blocking
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start droid: %w", err)
	}

	// Parse stream output
	result := &ExecuteResult{
		Success: true,
	}

	scanner := bufio.NewScanner(stdout)
	// Increase buffer size for large JSON lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max token size

	for scanner.Scan() {
		line := scanner.Text()
		var event droidStreamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		switch event.Type {
		case "message":
			if event.Role == "assistant" && event.Text != "" {
				result.Output += event.Text
			}
		case "completion":
			if event.FinalText != "" {
				result.Output = event.FinalText
			}
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
func (p *DroidProvider) ExecuteStream(ctx context.Context, opts *ExecuteOptions) (<-chan StreamEvent, error) {
	events := make(chan StreamEvent, 100)

	go func() {
		defer close(events)

		// Write prompt to temp file
		tmpFile, err := os.CreateTemp("", "hermes-droid-*.md")
		if err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
			return
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(opts.Prompt); err != nil {
			tmpFile.Close()
			events <- StreamEvent{Type: "error", Text: err.Error()}
			return
		}
		tmpFile.Close()

		// Build command
		args := []string{"exec", "--skip-permissions-unsafe", "--file", tmpFile.Name(), "--output-format", "stream-json"}

		cmd := exec.CommandContext(ctx, "droid", args...)

		if opts.WorkDir != "" {
			cmd.Dir = opts.WorkDir
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
			return
		}

		// Redirect stderr to prevent blocking
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
			return
		}

		scanner := bufio.NewScanner(stdout)
		// Increase buffer size for large JSON lines
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024) // 1MB max token size

		for scanner.Scan() {
			line := scanner.Text()
			var dEvent droidStreamEvent
			if err := json.Unmarshal([]byte(line), &dEvent); err != nil {
				continue
			}

			switch dEvent.Type {
			case "system":
				events <- StreamEvent{
					Type:  "system",
					Model: dEvent.Model,
				}
			case "message":
				if dEvent.Role == "assistant" && dEvent.Text != "" {
					events <- StreamEvent{
						Type: "text",
						Text: dEvent.Text,
					}
				}
			case "tool_call":
				events <- StreamEvent{
					Type:     "tool_use",
					ToolName: dEvent.ToolName,
				}
			case "tool_result":
				events <- StreamEvent{
					Type:     "tool_result",
					ToolName: dEvent.ToolName,
				}
			case "completion":
				events <- StreamEvent{
					Type:     "result",
					Text:     dEvent.FinalText,
					Duration: float64(dEvent.DurationMs) / 1000,
				}
			}
		}

		if err := cmd.Wait(); err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
		}
	}()

	return events, nil
}
