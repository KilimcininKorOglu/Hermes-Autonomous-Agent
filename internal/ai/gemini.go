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

// GeminiProvider implements Provider using Google Gemini CLI
type GeminiProvider struct{}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{}
}

// Name returns the provider name
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// IsAvailable checks if Gemini CLI is installed
func (p *GeminiProvider) IsAvailable() bool {
	_, err := exec.LookPath("gemini")
	return err == nil
}

// geminiJSONResponse represents the JSON response from gemini CLI
type geminiJSONResponse struct {
	SessionID string `json:"session_id"`
	Response  string `json:"response"`
	Stats     struct {
		Models map[string]struct {
			API struct {
				TotalRequests  int `json:"totalRequests"`
				TotalErrors    int `json:"totalErrors"`
				TotalLatencyMs int `json:"totalLatencyMs"`
			} `json:"api"`
			Tokens struct {
				Input      int `json:"input"`
				Prompt     int `json:"prompt"`
				Candidates int `json:"candidates"`
				Total      int `json:"total"`
				Cached     int `json:"cached"`
				Thoughts   int `json:"thoughts"`
			} `json:"tokens"`
		} `json:"models"`
	} `json:"stats"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Code    int    `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

// geminiStreamEvent represents a streaming event from gemini
// Types: init, message, tool_use, tool_result, error, result
type geminiStreamEvent struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Model     string `json:"model,omitempty"`
	// For message events
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
	Delta   bool   `json:"delta,omitempty"`
	// For tool events
	ToolName   string                 `json:"tool_name,omitempty"`
	ToolID     string                 `json:"tool_id,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Output     string                 `json:"output,omitempty"`
	// For result events
	Stats struct {
		TotalTokens  int `json:"total_tokens,omitempty"`
		InputTokens  int `json:"input_tokens,omitempty"`
		OutputTokens int `json:"output_tokens,omitempty"`
		DurationMs   int `json:"duration_ms,omitempty"`
		ToolCalls    int `json:"tool_calls,omitempty"`
	} `json:"stats,omitempty"`
}

// Execute runs a prompt and returns the result
func (p *GeminiProvider) Execute(ctx context.Context, opts *ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	// Write prompt to temp file for large prompts
	tmpFile, err := os.CreateTemp("", "hermes-gemini-*.md")
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

	// Build command - use headless mode with JSON output
	// gemini -p "prompt" --output-format json --yolo (auto-approve)
	args := []string{
		"-p", fmt.Sprintf("Read %s and follow the instructions.", tmpFile.Name()),
		"--output-format", "json",
		"--yolo", // Auto-approve all actions
	}

	cmd := exec.CommandContext(ctx, "gemini", args...)

	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	output, err := cmd.Output()
	if err != nil {
		// Try to parse error from stderr
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ExecuteResult{
				Success:  false,
				Error:    string(exitErr.Stderr),
				Duration: time.Since(start).Seconds(),
			}, nil
		}
		return nil, fmt.Errorf("failed to run gemini: %w", err)
	}

	// Parse JSON response
	var resp geminiJSONResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		// If not JSON, treat as plain text
		return &ExecuteResult{
			Output:   string(output),
			Success:  true,
			Duration: time.Since(start).Seconds(),
		}, nil
	}

	if resp.Error != nil {
		return &ExecuteResult{
			Success:  false,
			Error:    resp.Error.Message,
			Duration: time.Since(start).Seconds(),
		}, nil
	}

	// Calculate total tokens from all models
	var totalIn, totalOut int
	for _, model := range resp.Stats.Models {
		totalIn += model.Tokens.Input
		totalOut += model.Tokens.Candidates
	}

	return &ExecuteResult{
		Output:    resp.Response,
		TokensIn:  totalIn,
		TokensOut: totalOut,
		Success:   true,
		Duration:  time.Since(start).Seconds(),
	}, nil
}

// ExecuteStream runs a prompt with streaming output
func (p *GeminiProvider) ExecuteStream(ctx context.Context, opts *ExecuteOptions) (<-chan StreamEvent, error) {
	events := make(chan StreamEvent, 100)

	go func() {
		defer close(events)

		// Write prompt to temp file
		tmpFile, err := os.CreateTemp("", "hermes-gemini-*.md")
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

		// Use streaming output format
		args := []string{
			"-p", fmt.Sprintf("Read %s and follow the instructions.", tmpFile.Name()),
			"--output-format", "stream-json",
			"--yolo",
		}

		cmd := exec.CommandContext(ctx, "gemini", args...)

		if opts.WorkDir != "" {
			cmd.Dir = opts.WorkDir
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
			return
		}

		if err := cmd.Start(); err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			var gEvent geminiStreamEvent
			if err := json.Unmarshal([]byte(line), &gEvent); err != nil {
				// Plain text output
				events <- StreamEvent{
					Type: "text",
					Text: line,
				}
				continue
			}

			switch gEvent.Type {
			case "init":
				events <- StreamEvent{
					Type:  "system",
					Model: gEvent.Model,
				}
			case "message":
				if gEvent.Role == "assistant" && gEvent.Content != "" {
					events <- StreamEvent{
						Type: "text",
						Text: gEvent.Content,
					}
				}
			case "tool_use":
				events <- StreamEvent{
					Type:     "tool_use",
					ToolName: gEvent.ToolName,
					ToolID:   gEvent.ToolID,
				}
			case "tool_result":
				events <- StreamEvent{
					Type:     "tool_result",
					ToolName: gEvent.ToolName,
				}
			case "result":
				events <- StreamEvent{
					Type:     "result",
					Duration: float64(gEvent.Stats.DurationMs) / 1000,
				}
			case "error":
				events <- StreamEvent{
					Type: "error",
					Text: gEvent.Content,
				}
			}
		}

		if err := cmd.Wait(); err != nil {
			events <- StreamEvent{Type: "error", Text: err.Error()}
		}
	}()

	return events, nil
}
