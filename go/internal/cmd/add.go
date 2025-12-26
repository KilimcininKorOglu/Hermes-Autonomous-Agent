package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	ui.PrintBanner()
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

	// Get provider
	provider := ai.NewClaudeProvider()
	fmt.Printf("Using AI: %s\n\n", provider.Name())

	// Build prompt
	prompt := buildAddPrompt(featureDesc, nextFeatureID, nextTaskID)

	// Execute
	result, err := provider.Execute(ctx, &ai.ExecuteOptions{
		Prompt:       prompt,
		Timeout:      opts.timeout,
		StreamOutput: cfg.AI.StreamOutput,
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
	return fmt.Sprintf(`Create a feature file for: %s

Use Feature ID: F%03d
Start Task IDs from: T%03d

Format:
# Feature %d: <Feature Name based on description>
**Feature ID:** F%03d
**Status:** NOT_STARTED

### T%03d: Task Name
**Status:** NOT_STARTED
**Priority:** P1
**Files to Touch:** file1, file2
**Dependencies:** None
**Success Criteria:**
- Criterion 1
- Criterion 2

Create 3-5 tasks to implement this feature.
Output only the markdown content, no additional explanation.`, desc, featureID, taskID, featureID, featureID, taskID)
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
