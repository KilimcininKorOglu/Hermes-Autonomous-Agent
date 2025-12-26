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

	ui.PrintBanner()
	ui.PrintHeader("PRD Parser")

	// Load config
	cfg, err := config.Load(".")
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Read PRD file
	prdContent, err := os.ReadFile(prdFile)
	if err != nil {
		return fmt.Errorf("failed to read PRD: %w", err)
	}

	fmt.Printf("PRD file: %s (%d chars)\n", prdFile, len(prdContent))

	// Get provider
	provider := ai.NewClaudeProvider()
	fmt.Printf("Using AI: %s\n\n", provider.Name())

	// Build prompt
	prompt := buildPrdPrompt(string(prdContent))

	// Execute with retry
	result, err := ai.ExecuteWithRetry(ctx, provider, &ai.ExecuteOptions{
		Prompt:       prompt,
		Timeout:      opts.timeout,
		StreamOutput: cfg.AI.StreamOutput,
	}, &ai.RetryConfig{
		MaxRetries: opts.maxRetries,
		Delay:      10 * time.Second,
	})

	if err != nil {
		return fmt.Errorf("failed to parse PRD: %w", err)
	}

	if opts.dryRun {
		fmt.Println("\n--- DRY RUN OUTPUT ---")
		fmt.Println(result.Output)
		return nil
	}

	// Write task files
	return writeTaskFiles(result.Output)
}

func buildPrdPrompt(prdContent string) string {
	return fmt.Sprintf(`Parse this PRD into task files.

For each feature, create a markdown file with this format:

# Feature N: Feature Name
**Feature ID:** FXXX
**Status:** NOT_STARTED

### TXXX: Task Name
**Status:** NOT_STARTED
**Priority:** P1
**Files to Touch:** file1, file2
**Dependencies:** None
**Success Criteria:**
- Criterion 1
- Criterion 2

---

PRD Content:

%s

Output each file with:
---FILE: XXX-feature-name.md---
<content>
---END_FILE---`, prdContent)
}

func writeTaskFiles(output string) error {
	// Create tasks directory
	tasksDir := filepath.Join(".hermes", "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return err
	}

	// Parse FILE markers
	fileRegex := regexp.MustCompile(`---FILE:\s*(.+?)---\s*([\s\S]*?)---END_FILE---`)
	matches := fileRegex.FindAllStringSubmatch(output, -1)

	if len(matches) == 0 {
		// No file markers, write single file
		filePath := filepath.Join(tasksDir, "001-tasks.md")
		if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
			return err
		}
		fmt.Printf("Created: %s\n", filePath)
		return nil
	}

	for _, match := range matches {
		fileName := strings.TrimSpace(match[1])
		content := strings.TrimSpace(match[2])

		filePath := filepath.Join(tasksDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return err
		}
		fmt.Printf("Created: %s\n", filePath)
	}

	fmt.Printf("\nCreated %d task files in %s\n", len(matches), tasksDir)
	return nil
}
