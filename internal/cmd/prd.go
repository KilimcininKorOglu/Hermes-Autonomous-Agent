package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"hermes/internal/ai"
	"hermes/internal/config"
	"hermes/internal/ui"
)

type prdOptions struct {
	dryRun     bool
	timeout    int
	maxRetries int
	debug      bool
}

// NewPrdCmd creates the prd subcommand
func NewPrdCmd() *cobra.Command {
	opts := &prdOptions{}

	cmd := &cobra.Command{
		Use:   "prd <file>",
		Short: "Parse PRD to task files",
		Long:  "Parse a Product Requirements Document and generate task files",
		Example: `  hermes prd docs/PRD.md
  hermes prd requirements.md --dry-run
  hermes prd spec.md --timeout 1200`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return prdExecute(args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show output without writing files")
	cmd.Flags().IntVar(&opts.timeout, "timeout", 1200, "Timeout in seconds")
	cmd.Flags().IntVar(&opts.maxRetries, "max-retries", 10, "Max retry attempts")
	cmd.Flags().BoolVar(&opts.debug, "debug", false, "Enable debug output")

	return cmd
}

func prdExecute(prdFile string, opts *prdOptions) error {
	ctx := context.Background()

	ui.PrintBanner(GetVersion())
	ui.PrintHeader("PRD Parser")

	// Load config
	cfg, err := config.Load(".")
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Initialize logger
	logger, err := ui.NewLogger(".", opts.debug)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize logger: %v\n", err)
	} else {
		defer logger.Close()
	}

	// Read PRD file
	prdContent, err := os.ReadFile(prdFile)
	if err != nil {
		return fmt.Errorf("failed to read PRD: %w", err)
	}

	fmt.Printf("PRD file: %s (%d chars)\n", prdFile, len(prdContent))
	if logger != nil {
		logger.Info("PRD parsing started: %s (%d chars)", prdFile, len(prdContent))
	}

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
	if logger != nil {
		logger.Info("Using AI provider: %s", provider.Name())
	}

	// Build prompt
	prompt := buildPrdPrompt(string(prdContent))

	// Execute with retry
	startTime := time.Now()
	result, err := ai.ExecuteWithRetry(ctx, provider, &ai.ExecuteOptions{
		Prompt:       prompt,
		Timeout:      cfg.AI.PrdTimeout,
		StreamOutput: cfg.AI.StreamOutput,
	}, &ai.RetryConfig{
		MaxRetries: cfg.AI.MaxRetries,
		Delay:      time.Duration(cfg.AI.RetryDelay) * time.Second,
	})

	duration := time.Since(startTime)

	if err != nil {
		if logger != nil {
			logger.Error("PRD parsing failed: %v", err)
		}
		return fmt.Errorf("failed to parse PRD: %w", err)
	}

	if logger != nil {
		logger.Success("PRD parsed successfully in %v", duration.Round(time.Second))
	}

	if opts.dryRun {
		fmt.Println("\n--- DRY RUN OUTPUT ---")
		fmt.Println(result.Output)
		return nil
	}

	// Write task files
	if err := writeTaskFiles(result.Output); err != nil {
		if logger != nil {
			logger.Error("Failed to write task files: %v", err)
		}
		return err
	}

	if logger != nil {
		logger.Success("Task files created successfully")
	}

	return nil
}

func buildPrdPrompt(prdContent string) string {
	return fmt.Sprintf(`Parse this PRD into comprehensive task files.

For each feature, create a markdown file with this EXACT format:

# Feature N: Feature Name

**Feature ID:** FXXX
**Priority:** P[1-4] - [CRITICAL/HIGH/MEDIUM/LOW]
**Target Version:** vX.Y.Z
**Estimated Duration:** X-Y weeks
**Status:** NOT_STARTED

## Overview

[2-3 paragraph detailed description of the feature, its purpose, and how it fits into the overall system]

## Goals

- [Specific, measurable goal 1]
- [Specific, measurable goal 2]
- [Specific, measurable goal 3]

## Success Criteria

- [ ] All tasks completed
- [ ] All tests passing
- [ ] [Feature-specific criterion]

## Tasks

### TXXX: Task Name

**Status:** NOT_STARTED
**Priority:** P[1-4]
**Estimated Effort:** X days

#### Description

[Clear, detailed description of what this task accomplishes]

#### Technical Details

[Implementation notes, architecture decisions, code patterns to follow]

#### Files to Touch

- ` + "`path/to/file.go`" + ` (new)
- ` + "`path/to/existing.go`" + ` (update)

#### Dependencies

- TYYY (if depends on another task, use actual task ID like T001, T002)
- None (if no dependencies)

IMPORTANT: Dependencies MUST be valid task IDs (T001, T002, etc.) or "None".
Do NOT use descriptions like "All backend features", "Previous tasks", or any other text.

#### Success Criteria

- [ ] [Specific deliverable 1]
- [ ] [Specific deliverable 2]
- [ ] [Specific deliverable 3]
- [ ] Unit tests passing

---

[Repeat ### TXXX for each task in the feature]

## Performance Targets

- [Response time: < Xms]
- [Throughput: X requests/second]
- [Memory usage: < XMB]

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| [Risk 1] | Low/Medium/High | Low/Medium/High | [Mitigation strategy] |

## Notes

[Any additional context, references, or considerations]

---

IMPORTANT RULES:
1. Create 3-6 tasks per feature, each task should be 0.5-3 days of work
2. Tasks should be atomic and independently testable
3. Use realistic effort estimates based on complexity
4. Dependencies MUST reference actual task IDs (T001, T002, etc.) or be "None"
5. Do NOT use vague dependencies like "All backend features" or "Previous tasks"
6. Success criteria must be specific and measurable
7. Technical details should guide implementation
8. Priority levels: P1=Critical, P2=High, P3=Medium, P4=Low

FILE CREATION RULES:
- Create task files ONLY in the .hermes/tasks/ directory
- Use filenames like: .hermes/tasks/001-feature-name.md
- Do NOT create files anywhere else (not in tasks/, not in root, not in any other location)
- Each feature should be a separate file in .hermes/tasks/

PRD Content:

%s

Create each feature file directly in .hermes/tasks/ directory with proper naming (001-xxx.md, 002-xxx.md, etc).`, prdContent)
}

func writeTaskFiles(output string) error {
	// Create tasks directory
	tasksDir := filepath.Join(".hermes", "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return err
	}

	// Check if AI already created files using Create tool
	entries, err := os.ReadDir(tasksDir)
	if err == nil {
		mdFiles := 0
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				mdFiles++
			}
		}
		if mdFiles > 0 {
			fmt.Printf("\nFound %d task files in %s (created by AI)\n", mdFiles, tasksDir)
			return nil
		}
	}

	// Try parsing with END_FILE markers first
	fileRegex := regexp.MustCompile(`---FILE:\s*(.+?)---\s*([\s\S]*?)---END_FILE---`)
	matches := fileRegex.FindAllStringSubmatch(output, -1)

	// If no END_FILE markers, try parsing consecutive FILE markers
	if len(matches) == 0 {
		matches = parseConsecutiveFileMarkers(output)
	}

	if len(matches) == 0 {
		// No file markers and no files created by AI
		return fmt.Errorf("AI output did not contain valid file markers (---FILE: ... ---END_FILE---). Please try again")
	}

	for _, match := range matches {
		fileName := strings.TrimSpace(match[1])
		content := strings.TrimSpace(match[2])

		// Sanitize filename - remove any path components, keep only basename
		fileName = filepath.Base(fileName)
		
		// Ensure filename ends with .md
		if !strings.HasSuffix(fileName, ".md") {
			fileName = fileName + ".md"
		}

		filePath := filepath.Join(tasksDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return err
		}
		fmt.Printf("Created: %s\n", filePath)
	}

	fmt.Printf("\nCreated %d task files in %s\n", len(matches), tasksDir)
	return nil
}

// parseConsecutiveFileMarkers parses FILE markers without END_FILE markers
func parseConsecutiveFileMarkers(output string) [][]string {
	// Find all ---FILE: xxx--- positions
	markerRegex := regexp.MustCompile(`---FILE:\s*(.+?)---`)
	allMatches := markerRegex.FindAllStringSubmatchIndex(output, -1)

	if len(allMatches) == 0 {
		return nil
	}

	var results [][]string
	for i, match := range allMatches {
		// match[0] = start of full match
		// match[1] = end of full match
		// match[2] = start of capture group (filename)
		// match[3] = end of capture group (filename)

		fileName := output[match[2]:match[3]]
		contentStart := match[1] // After the ---FILE: xxx--- marker

		var contentEnd int
		if i+1 < len(allMatches) {
			// Content ends where next marker starts
			contentEnd = allMatches[i+1][0]
		} else {
			// Last file - content goes to end
			contentEnd = len(output)
		}

		content := strings.TrimSpace(output[contentStart:contentEnd])
		results = append(results, []string{"", fileName, content})
	}

	return results
}
