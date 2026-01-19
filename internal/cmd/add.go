package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"hermes/internal/ai"
	"hermes/internal/analyzer"
	"hermes/internal/config"
	"hermes/internal/ui"
)

type addOptions struct {
	dryRun  bool
	timeout int
	debug   bool
}

// NewAddCmd creates the add subcommand
func NewAddCmd() *cobra.Command {
	opts := &addOptions{}

	cmd := &cobra.Command{
		Use:   "add <feature-description>",
		Short: "Add a single feature",
		Long:  "Add a new feature to the task plan using AI",
		Example: `  hermes add "user authentication with JWT"
  hermes add "dark mode toggle" --dry-run
  hermes add "API rate limiting"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return addExecute(args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show output without writing")
	cmd.Flags().IntVar(&opts.timeout, "timeout", 300, "Timeout in seconds")
	cmd.Flags().BoolVar(&opts.debug, "debug", false, "Enable debug output")

	return cmd
}

func addExecute(featureDesc string, opts *addOptions) error {
	ctx := context.Background()

	ui.PrintBanner(GetVersion())
	ui.PrintHeader("Feature Add")

	fmt.Printf("Adding feature: %s\n\n", featureDesc)

	// Load config
	cfg, err := config.Load(".")
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Get next IDs
	featureAnalyzer := analyzer.NewFeatureAnalyzer(".")
	nextFeatureID, nextTaskID, err := featureAnalyzer.GetNextIDs()
	if err != nil {
		nextFeatureID = 1
		nextTaskID = 1
	}

	fmt.Printf("Next Feature ID: F%03d\n", nextFeatureID)
	fmt.Printf("Next Task ID: T%03d\n\n", nextTaskID)

	// Get provider from config
	var provider ai.Provider
	if cfg.AI.Planning != "" && cfg.AI.Planning != "auto" {
		provider = ai.GetProvider(cfg.AI.Planning)
	}
	if provider == nil || !provider.IsAvailable() {
		provider = ai.AutoDetectProvider()
	}
	if provider == nil {
		return fmt.Errorf("no AI provider available")
	}
	fmt.Printf("Using AI: %s\n\n", provider.Name())

	// Build prompt
	prompt := buildAddPrompt(featureDesc, nextFeatureID, nextTaskID)

	// Execute with retry
	result, err := ai.ExecuteWithRetry(ctx, provider, &ai.ExecuteOptions{
		Prompt:       prompt,
		Timeout:      cfg.AI.Timeout,
		StreamOutput: cfg.AI.StreamOutput,
	}, &ai.RetryConfig{
		MaxRetries: cfg.AI.MaxRetries,
		Delay:      5 * time.Second,
	})

	if err != nil {
		return fmt.Errorf("failed to add feature: %w", err)
	}

	if opts.dryRun {
		fmt.Println("\n--- DRY RUN OUTPUT ---")
		fmt.Println(result.Output)
		return nil
	}

	// Write task file
	return writeFeatureFile(result.Output, nextFeatureID, featureDesc)
}

func buildAddPrompt(desc string, featureID, taskID int) string {
	return fmt.Sprintf(`Create a comprehensive feature file for: %s

Use Feature ID: F%03d
Start Task IDs from: T%03d

Create the feature file with this EXACT format:

# Feature %d: <Feature Name based on description>

**Feature ID:** F%03d
**Priority:** P2 - HIGH
**Target Version:** v1.0.0
**Estimated Duration:** 1-2 weeks
**Status:** NOT_STARTED

## Overview

[Write 2-3 paragraphs describing the feature, its purpose, user value, and how it integrates with the system]

## Goals

- [Specific, measurable goal 1]
- [Specific, measurable goal 2]
- [Specific, measurable goal 3]

## Success Criteria

- [ ] All tasks completed (T%03d-T%03d)
- [ ] All tests passing
- [ ] Documentation updated

## Tasks

### T%03d: <First Task Name>

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

[Clear description of what this task accomplishes]

#### Technical Details

[Implementation notes, patterns to follow, architectural decisions]

#### Files to Touch

- `+"`path/to/file.go`"+` (new)
- `+"`path/to/existing.go`"+` (update)

#### Dependencies

- None

#### Success Criteria

- [ ] [Specific deliverable 1]
- [ ] [Specific deliverable 2]
- [ ] Unit tests passing

---

[Continue with more tasks...]

## Performance Targets

- Response time: < 100ms
- Memory usage: minimal overhead

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| [Potential risk] | Low | Medium | [How to mitigate] |

## Notes

[Any additional context or considerations]

---

RULES:
1. Create 3-5 tasks, each 0.5-2 days of work
2. Tasks must be atomic and testable
3. Include realistic effort estimates
4. Set proper dependencies between tasks
5. Success criteria must be specific and measurable
6. Analyze the project structure to suggest correct file paths

FILE CREATION RULES:
- Create the feature file ONLY in .hermes/tasks/ directory
- Use filename: .hermes/tasks/%03d-feature-name.md
- Do NOT create files anywhere else

Output only the markdown content, no additional explanation.`, desc, featureID, taskID, featureID, featureID, taskID, taskID+4, taskID, featureID)
}

func writeFeatureFile(output string, featureID int, desc string) error {
	// Create tasks directory
	tasksDir := filepath.Join(".hermes", "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return err
	}

	// Generate filename
	safeName := strings.ToLower(desc)
	safeName = strings.ReplaceAll(safeName, " ", "-")
	// Keep only alphanumeric and hyphens
	var result strings.Builder
	for _, r := range safeName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	safeName = result.String()
	if len(safeName) > 30 {
		safeName = safeName[:30]
	}

	fileName := fmt.Sprintf("%03d-%s.md", featureID, safeName)
	filePath := filepath.Join(tasksDir, fileName)

	if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
		return err
	}

	fmt.Printf("Created: %s\n", filePath)
	return nil
}
