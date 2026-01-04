package converter

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

// Generator generates PRD from project analysis
type Generator struct {
	provider ai.Provider
	config   *config.Config
	logger   *ui.Logger
}

// GenerateOptions contains options for PRD generation
type GenerateOptions struct {
	RootDir     string
	Output      string
	DryRun      bool
	Language    string
	Depth       int
	ExcludeDirs []string
	Timeout     int
}

// GenerateResult contains the result of PRD generation
type GenerateResult struct {
	PRDContent   string
	FilePath     string
	TokensUsed   int
	Duration     time.Duration
	ProjectName  string
	TotalFiles   int
	TotalDirs    int
	TechStack    []string
}

// NewGenerator creates a new PRD generator
func NewGenerator(provider ai.Provider, cfg *config.Config, logger *ui.Logger) *Generator {
	return &Generator{
		provider: provider,
		config:   cfg,
		logger:   logger,
	}
}

// Generate generates a PRD from project analysis
func (g *Generator) Generate(ctx context.Context, opts GenerateOptions) (*GenerateResult, error) {
	startTime := time.Now()

	// Analyze project
	g.logger.Info("Analyzing project structure...")
	analyzer := NewProjectAnalyzer(opts.RootDir, opts.Depth, opts.ExcludeDirs)
	analysis, err := analyzer.Analyze()
	if err != nil {
		return nil, fmt.Errorf("project analysis failed: %w", err)
	}

	g.logger.Info("Project: %s", analysis.ProjectName)
	g.logger.Info("Type: %s", analysis.ProjectType)
	g.logger.Info("Files: %d, Directories: %d", analysis.TotalFiles, analysis.TotalDirs)
	g.logger.Info("Tech Stack: %v", analysis.TechStack)

	// Build prompt
	prompt := BuildPrompt(analysis, opts.Language)

	g.logger.Info("Generating PRD...")
	g.logger.Debug("Prompt length: %d chars", len(prompt))

	// Execute AI with retry
	result, err := ai.ExecuteWithRetry(ctx, g.provider, &ai.ExecuteOptions{
		Prompt:       prompt,
		WorkDir:      opts.RootDir,
		Timeout:      opts.Timeout,
		StreamOutput: g.config.AI.StreamOutput,
	}, &ai.RetryConfig{
		MaxRetries: 3,
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
		PRDContent:  prdContent,
		FilePath:    opts.Output,
		TokensUsed:  result.TokensIn + result.TokensOut,
		Duration:    duration,
		ProjectName: analysis.ProjectName,
		TotalFiles:  analysis.TotalFiles,
		TotalDirs:   analysis.TotalDirs,
		TechStack:   analysis.TechStack,
	}, nil
}
