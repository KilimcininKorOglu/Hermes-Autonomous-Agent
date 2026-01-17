package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"hermes/internal/ai"
	"hermes/internal/config"
	"hermes/internal/converter"
	"hermes/internal/ui"
)

type convertPrdOptions struct {
	output      string
	dryRun      bool
	language    string
	depth       int
	exclude     string
	timeout     int
	debug       bool
}

// NewConvertPrdCmd creates the convert-prd subcommand
func NewConvertPrdCmd() *cobra.Command {
	opts := &convertPrdOptions{}

	cmd := &cobra.Command{
		Use:   "convert-prd",
		Short: "Generate PRD from existing project",
		Long:  "Analyze an existing project and generate a Product Requirements Document describing its current state",
		Example: `  hermes convert-prd
  hermes convert-prd --language tr
  hermes convert-prd --dry-run
  hermes convert-prd -o docs/PROJECT-PRD.md
  hermes convert-prd --exclude "test,docs,examples"
  hermes convert-prd --depth 4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return convertPrdExecute(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", ".hermes/docs/PRD.md", "Output file path")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Preview without writing file")
	cmd.Flags().StringVarP(&opts.language, "language", "l", "en", "PRD language (en/tr)")
	cmd.Flags().IntVarP(&opts.depth, "depth", "d", 3, "Directory analysis depth")
	cmd.Flags().StringVar(&opts.exclude, "exclude", "", "Comma-separated directories to exclude (in addition to defaults)")
	cmd.Flags().IntVar(&opts.timeout, "timeout", 900, "AI timeout in seconds")
	cmd.Flags().BoolVar(&opts.debug, "debug", false, "Enable debug output")

	return cmd
}

func convertPrdExecute(opts *convertPrdOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.timeout)*time.Second)
	defer cancel()

	ui.PrintBanner(GetVersion())
	ui.PrintHeader("Project to PRD Converter")

	// Get current directory
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if .hermes directory exists (optional for this command)
	hermesDir := ".hermes"
	if _, err := os.Stat(hermesDir); os.IsNotExist(err) {
		// Create .hermes directory structure
		if err := os.MkdirAll(hermesDir+"/docs", 0755); err != nil {
			return fmt.Errorf("failed to create .hermes directory: %w", err)
		}
		fmt.Println("Created .hermes directory structure")
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

	fmt.Printf("Project: %s\n", rootDir)
	fmt.Printf("AI: %s\n", provider.Name())
	fmt.Printf("Language: %s\n", opts.language)
	fmt.Printf("Depth: %d\n", opts.depth)

	// Parse exclude directories
	var excludeDirs []string
	if opts.exclude != "" {
		excludeDirs = strings.Split(opts.exclude, ",")
		for i := range excludeDirs {
			excludeDirs[i] = strings.TrimSpace(excludeDirs[i])
		}
		fmt.Printf("Additional excludes: %v\n", excludeDirs)
	}

	// Create generator
	gen := converter.NewGenerator(provider, cfg, logger)

	fmt.Println("\nAnalyzing project and generating PRD...")

	// Generate PRD
	result, err := gen.Generate(ctx, converter.GenerateOptions{
		RootDir:     rootDir,
		Output:      opts.output,
		DryRun:      opts.dryRun,
		Language:    opts.language,
		Depth:       opts.depth,
		ExcludeDirs: excludeDirs,
		Timeout:     opts.timeout,
	})
	if err != nil {
		return err
	}

	// Show result
	fmt.Println()
	if opts.dryRun {
		fmt.Println("=== PRD Preview ===")
		fmt.Println(result.PRDContent)
		fmt.Println("===================")
		fmt.Printf("\nWould be written to: %s\n", result.FilePath)
	} else {
		logger.Success("PRD generated: %s", result.FilePath)
		fmt.Printf("Project: %s\n", result.ProjectName)
		fmt.Printf("Analyzed: %d files, %d directories\n", result.TotalFiles, result.TotalDirs)
		fmt.Printf("Tech Stack: %s\n", strings.Join(result.TechStack, ", "))
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
