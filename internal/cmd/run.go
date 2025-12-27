package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"hermes/internal/ai"
	"hermes/internal/analyzer"
	"hermes/internal/circuit"
	"hermes/internal/config"
	"hermes/internal/git"
	"hermes/internal/prompt"
	"hermes/internal/task"
	"hermes/internal/ui"
)

// NewRunCmd creates the run subcommand
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run task execution loop",
		Long:  "Execute tasks from task files using Claude CLI",
		Example: `  hermes run
  hermes run --auto-branch --auto-commit
  hermes run --autonomous=false`,
		RunE: runExecute,
	}

	cmd.Flags().Bool("auto-branch", false, "Create feature branches (overrides config)")
	cmd.Flags().Bool("auto-commit", false, "Commit on task completion (overrides config)")
	cmd.Flags().Bool("autonomous", true, "Run without pausing (overrides config)")
	cmd.Flags().Int("timeout", 0, "AI timeout in seconds (0 = use config)")
	cmd.Flags().Bool("debug", false, "Enable debug output")
	cmd.Flags().String("ai", "", "AI provider: claude, droid, gemini, auto (default: from config or auto)")

	return cmd
}

func runExecute(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, shutting down...")
		cancel()
	}()

	// Load config
	cfg, err := config.Load(".")
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Apply CLI flags (override config if flag was explicitly set)
	autoBranch := cfg.TaskMode.AutoBranch
	autoCommit := cfg.TaskMode.AutoCommit
	autonomous := cfg.TaskMode.Autonomous
	debug := false

	if cmd.Flags().Changed("auto-branch") {
		autoBranch, _ = cmd.Flags().GetBool("auto-branch")
	}
	if cmd.Flags().Changed("auto-commit") {
		autoCommit, _ = cmd.Flags().GetBool("auto-commit")
	}
	if cmd.Flags().Changed("autonomous") {
		autonomous, _ = cmd.Flags().GetBool("autonomous")
	}
	if cmd.Flags().Changed("debug") {
		debug, _ = cmd.Flags().GetBool("debug")
	}

	// Initialize logger
	logger, err := ui.NewLogger(".", debug)
	if err != nil {
		return err
	}
	defer logger.Close()

	ui.PrintBanner()
	ui.PrintHeader("Task Execution Loop")

	// Initialize components
	reader := task.NewReader(".")
	breaker := circuit.New(".")
	gitOps := git.New(".")
	injector := prompt.NewInjector(".")
	respAnalyzer := analyzer.NewResponseAnalyzer()

	// Initialize circuit breaker
	if err := breaker.Initialize(); err != nil {
		return err
	}

	// Check for tasks
	if !reader.HasTasks() {
		return fmt.Errorf("no tasks found, run 'hermes prd <file>' first")
	}

	// Get AI provider
	aiFlag, _ := cmd.Flags().GetString("ai")
	var provider ai.Provider

	if aiFlag != "" && aiFlag != "auto" {
		provider = ai.GetProvider(aiFlag)
		if provider == nil {
			return fmt.Errorf("unknown AI provider: %s", aiFlag)
		}
		if !provider.IsAvailable() {
			return fmt.Errorf("AI provider %s is not available (not installed)", aiFlag)
		}
	} else {
		// Use config or auto-detect
		if cfg.AI.Coding != "" && cfg.AI.Coding != "auto" {
			provider = ai.GetProvider(cfg.AI.Coding)
		}
		if provider == nil || !provider.IsAvailable() {
			provider = ai.AutoDetectProvider()
		}
	}

	if provider == nil {
		return fmt.Errorf("no AI provider available (install claude or droid)")
	}

	logger.Info("Using AI provider: %s", provider.Name())

	loopNumber := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		loopNumber++
		ui.PrintLoopHeader(loopNumber)

		// Check circuit breaker
		canExecute, err := breaker.CanExecute()
		if err != nil {
			return err
		}
		if !canExecute {
			breaker.PrintHaltMessage()
			return nil
		}

		// Get next task
		nextTask, err := reader.GetNextTask()
		if err != nil {
			return err
		}
		if nextTask == nil {
			logger.Success("All tasks completed!")
			return nil
		}

		ui.PrintTaskHeader(nextTask)
		logger.Info("Working on task: %s - %s", nextTask.ID, nextTask.Name)

		// Set task status to IN_PROGRESS before starting
		statusUpdater := task.NewStatusUpdater(".")
		if err := statusUpdater.UpdateTaskStatus(nextTask.ID, task.StatusInProgress); err != nil {
			logger.Warn("Failed to set task IN_PROGRESS: %v", err)
		}

		// Handle branching
		if autoBranch && gitOps.IsRepository() {
			feature, _ := reader.GetFeatureByID(nextTask.FeatureID)
			if feature != nil {
				branchName, err := gitOps.CreateFeatureBranch(feature.ID, feature.Name)
				if err == nil {
					logger.Info("On branch: %s", branchName)
				}
			}
		}

		// Inject task into prompt
		if err := injector.AddTask(nextTask); err != nil {
			logger.Warn("Failed to inject task: %v", err)
		}
		promptContent, _ := injector.Read()

		// Execute AI
		executor := ai.NewTaskExecutor(provider, ".")
		result, err := executor.ExecuteTask(ctx, nextTask, promptContent)

		if err != nil {
			logger.Error("AI execution failed: %v", err)
			breaker.AddLoopResult(false, true, loopNumber)

			// Wait before retry
			time.Sleep(time.Duration(cfg.Loop.ErrorDelay) * time.Second)
			continue
		}

		// Analyze response
		analysis := respAnalyzer.Analyze(result.Output)
		logger.Debug("Analysis: progress=%v complete=%v confidence=%.2f",
			analysis.HasProgress, analysis.IsComplete, analysis.Confidence)

		// Update circuit breaker
		breaker.AddLoopResult(analysis.HasProgress, false, loopNumber)

		// Update task status if complete
		if analysis.IsComplete {
			// Remove task from prompt
			injector.RemoveTask()

			// Set task status to COMPLETED before commit
			if err := statusUpdater.UpdateTaskStatus(nextTask.ID, task.StatusCompleted); err != nil {
				logger.Warn("Failed to update task status: %v", err)
			}

			// Auto-commit (includes the status update)
			if autoCommit && gitOps.HasUncommittedChanges() {
				if err := gitOps.StageAll(); err == nil {
					if err := gitOps.CommitTask(nextTask.ID, nextTask.Name); err != nil {
						logger.Warn("Failed to commit: %v", err)
					} else {
						logger.Success("Committed task %s", nextTask.ID)
					}
				}
			}

			logger.Success("Task %s completed", nextTask.ID)

			// Check if feature is complete and create tag
			if featureComplete, _ := reader.IsFeatureComplete(nextTask.FeatureID); featureComplete {
				feature, _ := reader.GetFeatureByID(nextTask.FeatureID)
				if feature != nil {
					logger.Success("Feature %s completed: %s", feature.ID, feature.Name)

					// Create git tag if TargetVersion is set
					if feature.TargetVersion != "" && gitOps.IsRepository() {
						if err := gitOps.CreateFeatureTag(feature.ID, feature.Name, feature.TargetVersion); err != nil {
							logger.Warn("Failed to create tag: %v", err)
						} else {
							logger.Success("Created tag: %s", feature.TargetVersion)
						}
					}
				}
			}

			// Show progress
			if progress, err := reader.GetProgress(); err == nil {
				bar := ui.FormatProgressBar(progress.Percentage, 30)
				fmt.Printf("\nProgress: %s\n", bar)
			}
		}

		// Pause between tasks if not autonomous
		if !autonomous && analysis.IsComplete {
			fmt.Println("\nPress Enter to continue or Ctrl+C to stop...")
			bufio.NewReader(os.Stdin).ReadBytes('\n')
		}
	}
}
