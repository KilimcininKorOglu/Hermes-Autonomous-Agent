package ai

import (
	"context"
	"fmt"
	"time"
)

// RetryConfig contains retry configuration
type RetryConfig struct {
	MaxRetries int
	Delay      time.Duration
	MaxDelay   time.Duration
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		Delay:      5 * time.Second,
		MaxDelay:   60 * time.Second,
	}
}

// ExecuteWithRetry executes with retry logic and exponential backoff
func ExecuteWithRetry(ctx context.Context, provider Provider, opts *ExecuteOptions, cfg *RetryConfig) (*ExecuteResult, error) {
	if cfg == nil {
		cfg = DefaultRetryConfig()
	}

	var lastErr error
	delay := cfg.Delay

	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
		var result *ExecuteResult
		var err error

		// Use streaming if enabled
		if opts.StreamOutput {
			result, err = executeWithStreaming(ctx, provider, opts)
		} else {
			result, err = provider.Execute(ctx, opts)
		}

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

// executeWithStreaming executes with real-time output to console
func executeWithStreaming(ctx context.Context, provider Provider, opts *ExecuteOptions) (*ExecuteResult, error) {
	events, err := provider.ExecuteStream(ctx, opts)
	if err != nil {
		return nil, err
	}

	var output string
	for event := range events {
		switch event.Type {
		case "text":
			fmt.Print(event.Text)
			output += event.Text
		case "error":
			return &ExecuteResult{Success: false, Output: output, Error: event.Text}, nil
		case "done":
			fmt.Println()
		}
	}

	return &ExecuteResult{Success: true, Output: output}, nil
}
