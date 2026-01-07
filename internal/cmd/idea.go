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
	"hermes/internal/config"
	"hermes/internal/idea"
	"hermes/internal/ui"
)

type ideaOptions struct {
	output      string
	dryRun      bool
	interactive bool
	language    string
	timeout     int
	debug       bool
}

// NewIdeaCmd creates the idea subcommand
func NewIdeaCmd() *cobra.Command {
	opts := &ideaOptions{}

	cmd := &cobra.Command{
		Use:   "idea <description>",
		Short: "Generate PRD from idea",
		Long:  "Generate a detailed Product Requirements Document from a simple idea or description",
		Example: `  hermes idea "e-commerce website"
  hermes idea "real-time chat app" --interactive
  hermes idea "task manager" --language tr
  hermes idea "blog platform" --dry-run`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ideaText := strings.Join(args, " ")
			return ideaExecute(ideaText, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", ".hermes/docs/PRD.md", "Output file path")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Preview without writing file")
	cmd.Flags().BoolVarP(&opts.interactive, "interactive", "i", false, "Interactive mode with additional questions")
	cmd.Flags().StringVarP(&opts.language, "language", "l", "en", "PRD language (en/tr)")
	cmd.Flags().IntVar(&opts.timeout, "timeout", 600, "AI timeout in seconds")
	cmd.Flags().BoolVar(&opts.debug, "debug", false, "Enable debug output")

	return cmd
}

func ideaExecute(ideaText string, opts *ideaOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.timeout)*time.Second)
	defer cancel()

	ui.PrintBanner(GetVersion())
	ui.PrintHeader("Idea to PRD Generator")

	// Check if .hermes directory exists
	if _, err := os.Stat(".hermes"); os.IsNotExist(err) {
		return fmt.Errorf("run 'hermes init' first")
	}

	// Load config
	cfg, err := config.Load(".")
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Create logger
	logger, err := ui.NewLogger(".", opts.debug)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer logger.Close()

	// Get provider
	var provider ai.Provider
	if cfg.AI.Planning != "" && cfg.AI.Planning != "auto" {
		provider = ai.GetProvider(cfg.AI.Planning)
	}
	if provider == nil {
		provider = ai.AutoDetectProvider()
	}
	if provider == nil {
		return fmt.Errorf("AI provider not found (install claude, droid, or gemini)")
	}

	fmt.Printf("Idea: %s\n", ideaText)
	fmt.Printf("AI: %s\n", provider.Name())
	fmt.Printf("Language: %s\n", opts.language)

	// Interactive mode
	var additionalContext string
	if opts.interactive {
		answers, err := idea.AskQuestions(ideaText)
		if err != nil {
			return fmt.Errorf("failed to get answers: %w", err)
		}
		additionalContext = idea.FormatAnswers(answers)
	}

	// Create generator
	gen := idea.NewGenerator(provider, cfg, logger)

	// Ensure output directory exists
	outputPath := opts.output
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Clean(outputPath)
	}

	fmt.Println("\nGenerating PRD...")

	// Generate PRD
	result, err := gen.Generate(ctx, idea.GenerateOptions{
		Idea:              ideaText,
		Output:            outputPath,
		DryRun:            opts.dryRun,
		Interactive:       opts.interactive,
		Language:          opts.language,
		Timeout:           opts.timeout,
		AdditionalContext: additionalContext,
	})
	if err != nil {
		return err
	}

	// Show result
	if opts.dryRun {
		fmt.Println("\n=== PRD Preview ===")
		fmt.Println(result.PRDContent)
		fmt.Println("===================")
		fmt.Printf("\nWould be written to: %s\n", result.FilePath)
	} else {
		logger.Success("PRD generated: %s", result.FilePath)
		if result.TokensUsed > 0 {
			logger.Info("Tokens used: %d", result.TokensUsed)
		}
		logger.Info("Duration: %s", result.Duration.Round(time.Millisecond))

		fmt.Println("\nNext steps:")
		fmt.Printf("  1. Review: cat %s\n", result.FilePath)
		fmt.Printf("  2. Parse:  hermes prd %s\n", result.FilePath)
	}

	return nil
}
