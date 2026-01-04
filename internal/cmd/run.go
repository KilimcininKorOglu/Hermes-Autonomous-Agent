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
	"hermes/internal/scheduler"
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
  hermes run --autonomous=false
  hermes run --parallel --workers 3
  hermes run --parallel --dry-run`,
		RunE: runExecute,
	}

	cmd.Flags().Bool("auto-branch", false, "Create feature branches (overrides config)")
	cmd.Flags().Bool("auto-commit", false, "Commit on task completion (overrides config)")
	cmd.Flags().Bool("autonomous", true, "Run without pausing (overrides config)")
	cmd.Flags().Int("timeout", 0, "AI timeout in seconds (0 = use config)")
	cmd.Flags().Bool("debug", false, "Enable debug output")
	cmd.Flags().String("ai", "", "AI provider: claude, droid, gemini, auto (default: from config or auto)")
	// Parallel execution flags
	cmd.Flags().Bool("parallel", false, "Enable parallel task execution")
	cmd.Flags().Int("workers", 3, "Number of parallel workers (default: 3)")
	cmd.Flags().Bool("dry-run", false, "Show execution plan without running")

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

	// Check for parallel execution mode
	parallel, _ := cmd.Flags().GetBool("parallel")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	workers, _ := cmd.Flags().GetInt("workers")

	// Override with config if flag not set
	if !cmd.Flags().Changed("parallel") {
		parallel = cfg.Parallel.Enabled
	}
	if !cmd.Flags().Changed("workers") {
		workers = cfg.Parallel.MaxWorkers
	}

	// Handle parallel execution
	if parallel || dryRun {
		return runParallel(ctx, cfg, provider, reader, logger, workers, dryRun)
	}

	// Sequential execution (original behavior)
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
		result, err := executor.ExecuteTask(ctx, nextTask, promptContent, cfg.AI.StreamOutput)

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

// runParallel executes tasks in parallel mode
func runParallel(ctx context.Context, cfg *config.Config, provider ai.Provider, reader *task.Reader, logger *ui.Logger, workers int, dryRun bool) error {
	ui.PrintHeader("Parallel Task Execution")

	// Get all tasks (including completed for dependency resolution)
	allTasks, err := reader.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	// Count pending tasks
	pendingCount := 0
	for i := range allTasks {
		if allTasks[i].Status == task.StatusNotStarted {
			pendingCount++
		}
	}

	if pendingCount == 0 {
		logger.Success("No pending tasks to execute!")
		return nil
	}

	logger.Info("Found %d pending tasks", pendingCount)
	logger.Info("Using %d parallel workers", workers)

	// Convert to pointer slice for scheduler (includes all tasks for dependency resolution)
	allTaskPtrs := make([]*task.Task, len(allTasks))
	for i := range allTasks {
		allTaskPtrs[i] = &allTasks[i]
	}

	// Update parallel config with CLI values
	parallelCfg := cfg.Parallel
	parallelCfg.MaxWorkers = workers

	// Create scheduler
	sched := scheduler.New(&parallelCfg, provider, ".", logger)

	// Get execution plan (uses all tasks for dependency resolution, but only executes pending)
	plan, err := sched.GetExecutionPlan(allTaskPtrs)
	if err != nil {
		return fmt.Errorf("failed to create execution plan: %w", err)
	}

	// Print execution plan
	sched.PrintExecutionPlan(plan)

	// If dry-run, stop here
	if dryRun {
		logger.Info("Dry run complete. Use --parallel without --dry-run to execute.")
		return nil
	}

	// Initialize parallel logger
	parallelLogger, err := scheduler.NewParallelLogger(".", workers)
	if err != nil {
		logger.Warn("Failed to initialize parallel logger: %v", err)
	} else {
		defer parallelLogger.Close()
		logger.Info("Logs will be written to: %s", parallelLogger.GetLogDirectory())
		// Connect logger to scheduler
		sched.SetParallelLogger(parallelLogger)
	}

	// Initialize resource monitor
	resourceMonitor := scheduler.NewResourceMonitor(
		0, // No memory limit
		0, // No CPU limit
		cfg.Loop.MaxCallsPerHour,
	)
	if cfg.Parallel.MaxCostPerHour > 0 {
		resourceMonitor.SetCostLimit(cfg.Parallel.MaxCostPerHour)
	}

	// Initialize rollback manager
	rollback := scheduler.NewRollback(".")
	defer func() {
		// Cleanup on exit
		if rollback.HasSnapshots() {
			rollback.CleanupWorktrees()
		}
	}()

	// Confirm execution
	fmt.Println("\nPress Enter to start parallel execution or Ctrl+C to cancel...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	// Save initial snapshot
	if err := rollback.SaveSnapshot("INITIAL"); err != nil {
		logger.Warn("Failed to save initial snapshot: %v", err)
	}

	// Log execution start
	if parallelLogger != nil {
		parallelLogger.Main("Starting parallel execution with %d workers", workers)
		parallelLogger.Main("Total tasks: %d, Batches: %d", pendingCount, len(plan.Batches))
	}

	// Execute tasks
	logger.Info("Starting parallel execution...")
	startTime := time.Now()

	result, err := sched.Execute(ctx, allTaskPtrs)
	
	executionTime := time.Since(startTime)

	if err != nil {
		logger.Error("Parallel execution failed: %v", err)
		if parallelLogger != nil {
			parallelLogger.Main("Execution failed: %v", err)
		}

		// Offer rollback on failure
		if result != nil && result.Failed > 0 {
			fmt.Println("\nExecution failed. Would you like to rollback? (y/n)")
			var response string
			fmt.Scanln(&response)
			if response == "y" || response == "Y" {
				if err := rollback.RollbackAll(); err != nil {
					logger.Error("Rollback failed: %v", err)
				} else {
					logger.Success("Rollback completed successfully")
				}
			}
		}
	}

	// Print results
	sched.PrintExecutionResult(result)

	// Log completion
	if parallelLogger != nil {
		parallelLogger.ExecutionComplete(result.Successful, result.Failed)
	}

	// Print resource stats
	stats := resourceMonitor.GetStats()
	if stats.TotalAPICalls > 0 {
		stats.Print()
	}

	// Print timing
	fmt.Printf("\n⏱️  Total execution time: %v\n", executionTime.Round(time.Second))

	// Update task statuses and check for feature completion
	statusUpdater := task.NewStatusUpdater(".")
	gitOps := git.New(".")
	completedFeatures := make(map[string]bool)

	for _, r := range result.Results {
		if r.Success {
			if err := statusUpdater.UpdateTaskStatus(r.TaskID, task.StatusCompleted); err != nil {
				logger.Warn("Failed to update task %s status: %v", r.TaskID, err)
			}

			// Track which features might be complete
			for _, t := range allTaskPtrs {
				if t.ID == r.TaskID {
					completedFeatures[t.FeatureID] = true
					break
				}
			}
		}
	}

	// Check for completed features and create tags
	for featureID := range completedFeatures {
		if featureComplete, _ := reader.IsFeatureComplete(featureID); featureComplete {
			feature, _ := reader.GetFeatureByID(featureID)
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
	}

	// Cleanup worktrees only (keep task branches for history)
	rollback.CleanupWorktrees()

	if result.Failed > 0 {
		return fmt.Errorf("%d tasks failed", result.Failed)
	}

	logger.Success("All %d tasks completed successfully!", result.Successful)
	return nil
}
