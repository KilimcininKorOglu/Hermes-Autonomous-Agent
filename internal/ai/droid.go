package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
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
	return "Droid"
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
	ToolID     string                 `json:"toolId,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Output     string                 `json:"output,omitempty"`
	IsError    bool                   `json:"isError,omitempty"`
	Error      string                 `json:"error,omitempty"`
	DurationMs int64                  `json:"durationMs,omitempty"`
	NumTurns   int                    `json:"numTurns,omitempty"`
	FinalText  string                 `json:"finalText,omitempty"`
}

// buildDroidCommand creates the droid command with pseudo-TTY wrapper if needed.
// Droid uses Ink-based UI which requires a TTY. On Unix systems, we use the
// `script` command to create a pseudo-TTY, similar to Ralph-TUI's approach.
func buildDroidCommand(ctx context.Context, droidArgs []string, workDir string) *exec.Cmd {
	// Build the base droid command string
	droidCmd := "droid " + strings.Join(droidArgs, " ")

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS: script -q /dev/null sh -c "droid exec ..."
		cmd = exec.CommandContext(ctx, "script", "-q", "/dev/null", "sh", "-c", droidCmd)
	case "linux":
		// Linux: script -q -c "droid exec ..." /dev/null
		cmd = exec.CommandContext(ctx, "script", "-q", "-c", droidCmd, "/dev/null")
	default:
		// Windows and others: run droid directly (no pseudo-TTY available)
		cmd = exec.CommandContext(ctx, "droid", droidArgs...)
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	// Set environment variables to signal non-interactive mode to Ink-based CLIs
	cmd.Env = append(os.Environ(),
		"CI=true",
		"TERM=dumb",
		"NO_COLOR=1",
		"FORCE_COLOR=0",
		"INK_DISABLE_INPUT=1",
	)

	return cmd
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
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Build command args
	args := []string{"exec", "--skip-permissions-unsafe", "--file", tmpFile.Name(), "--output-format", "stream-json"}

	// Build command with pseudo-TTY wrapper
	cmd := buildDroidCommand(ctx, args, opts.WorkDir)

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
		if err := tmpFile.Close(); err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
			return
		}

		// Build command args
		args := []string{"exec", "--skip-permissions-unsafe", "--file", tmpFile.Name(), "--output-format", "stream-json"}

		// Build command with pseudo-TTY wrapper
		cmd := buildDroidCommand(ctx, args, opts.WorkDir)

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
					Type:      "tool_use",
					ToolName:  dEvent.ToolName,
					ToolID:    dEvent.ToolID,
					ToolInput: dEvent.Parameters,
				}
			case "tool_result":
				events <- StreamEvent{
					Type:       "tool_result",
					ToolName:   dEvent.ToolName,
					ToolID:     dEvent.ToolID,
					ToolOutput: dEvent.Output,
					ToolError:  dEvent.Error,
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
