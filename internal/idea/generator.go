package idea

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"hermes/internal/ai"
	"hermes/internal/config"
	"hermes/internal/ui"
)

// Generator generates PRD from idea
type Generator struct {
	provider ai.Provider
	config   *config.Config
	logger   *ui.Logger
}

// GenerateOptions contains options for PRD generation
type GenerateOptions struct {
	Idea              string
	Output            string
	DryRun            bool
	Interactive       bool
	Language          string
	Timeout           int
	AdditionalContext string
}

// GenerateResult contains the result of PRD generation
type GenerateResult struct {
	PRDContent string
	FilePath   string
	TokensUsed int
	Duration   time.Duration
}

// NewGenerator creates a new PRD generator
func NewGenerator(provider ai.Provider, cfg *config.Config, logger *ui.Logger) *Generator {
	return &Generator{
		provider: provider,
		config:   cfg,
		logger:   logger,
	}
}

// Generate generates a PRD from the given idea
func (g *Generator) Generate(ctx context.Context, opts GenerateOptions) (*GenerateResult, error) {
	startTime := time.Now()

	// Build prompt
	prompt := BuildPrompt(opts.Idea, opts.Language, opts.AdditionalContext)

	g.logger.Info("Generating PRD...")
	g.logger.Debug("Idea: %s", opts.Idea)
	g.logger.Debug("Language: %s", opts.Language)

	// Execute AI with retry
	result, err := ai.ExecuteWithRetry(ctx, g.provider, &ai.ExecuteOptions{
		Prompt:       prompt,
		WorkDir:      ".",
		Timeout:      g.config.AI.PrdTimeout,
		StreamOutput: g.config.AI.StreamOutput,
	}, &ai.RetryConfig{
		MaxRetries: g.config.AI.MaxRetries,
		Delay:      5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("AI execution failed: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("AI execution failed: %s", result.Error)
	}

	prdContent := result.Output

	// Write file if not dry-run
	if !opts.DryRun {
		// Ensure directory exists
		dir := filepath.Dir(opts.Output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}

		if err := os.WriteFile(opts.Output, []byte(prdContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write PRD: %w", err)
		}
	}

	duration := time.Since(startTime)

	return &GenerateResult{
		PRDContent: prdContent,
		FilePath:   opts.Output,
		TokensUsed: result.TokensIn + result.TokensOut,
		Duration:   duration,
	}, nil
}
