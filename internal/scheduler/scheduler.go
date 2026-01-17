package scheduler

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"hermes/internal/ai"
	"hermes/internal/circuit"
	"hermes/internal/config"
	"hermes/internal/isolation"
	"hermes/internal/task"
	"hermes/internal/ui"
)

// Scheduler manages parallel task execution
type Scheduler struct {
	config         *config.ParallelConfig
	provider       ai.Provider
	workDir        string
	logger         *ui.Logger
	parallelLogger *ParallelLogger
	breaker        *circuit.Breaker
	mu             sync.Mutex
}

// ExecutionPlan represents the planned execution order
type ExecutionPlan struct {
	Batches      [][]*task.Task
	TotalTasks   int
	EstimatedTime time.Duration
}

// ExecutionResult represents the result of executing all tasks
type ExecutionResult struct {
	Results     []*TaskResult
	TotalTime   time.Duration
	Successful  int
	Failed      int
	StartTime   time.Time
	EndTime     time.Time
}

// New creates a new scheduler
func New(cfg *config.ParallelConfig, provider ai.Provider, workDir string, logger *ui.Logger) *Scheduler {
	breaker := circuit.New(workDir)
	breaker.Initialize()
	
	return &Scheduler{
		config:   cfg,
		provider: provider,
		workDir:  workDir,
		logger:   logger,
		breaker:  breaker,
	}
}

// SetParallelLogger sets the parallel logger for per-worker logging
func (s *Scheduler) SetParallelLogger(logger *ParallelLogger) {
	s.parallelLogger = logger
}

// GetExecutionPlan returns the planned execution order without executing
func (s *Scheduler) GetExecutionPlan(tasks []*task.Task) (*ExecutionPlan, error) {
	graph, err := NewTaskGraph(tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to build task graph: %w", err)
	}

	batches, err := graph.GetBatches()
	if err != nil {
		return nil, fmt.Errorf("failed to compute batches: %w", err)
	}

	return &ExecutionPlan{
		Batches:    batches,
		TotalTasks: len(tasks),
	}, nil
}

// Execute runs all tasks respecting dependencies
func (s *Scheduler) Execute(ctx context.Context, tasks []*task.Task) (*ExecutionResult, error) {
	startTime := time.Now()
	
	result := &ExecutionResult{
		Results:   make([]*TaskResult, 0),
		StartTime: startTime,
	}

	// Build task graph
	graph, err := NewTaskGraph(tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to build task graph: %w", err)
	}

	// Get execution plan
	batches, err := graph.GetBatches()
	if err != nil {
		return nil, fmt.Errorf("failed to compute execution batches: %w", err)
	}

	s.logInfo("Execution plan: %d batches, %d total tasks", len(batches), len(tasks))
	for i, batch := range batches {
		taskIDs := make([]string, len(batch))
		for j, t := range batch {
			taskIDs[j] = t.ID
		}
		s.logInfo("  Batch %d: %v", i+1, taskIDs)
	}

	// Execute each batch
	for batchNum, batch := range batches {
		select {
		case <-ctx.Done():
			result.EndTime = time.Now()
			result.TotalTime = result.EndTime.Sub(startTime)
			return result, ctx.Err()
		default:
		}

		// Check circuit breaker before batch execution
		canExecute, err := s.breaker.CanExecute()
		if err != nil {
			s.logError("Circuit breaker error: %v", err)
		}
		if !canExecute {
			s.logError("Circuit breaker OPEN - stopping execution")
			result.EndTime = time.Now()
			result.TotalTime = result.EndTime.Sub(startTime)
			s.countResults(result)
			return result, fmt.Errorf("circuit breaker open: execution halted due to no progress")
		}

		s.logInfo("Starting batch %d/%d with %d tasks", batchNum+1, len(batches), len(batch))

		batchResults, err := s.executeBatch(ctx, graph, batch)
		
		// Calculate batch progress for circuit breaker
		batchHasProgress := false
		for _, r := range batchResults {
			if r.Success {
				batchHasProgress = true
				break
			}
		}
		
		// Report to circuit breaker
		s.breaker.AddLoopResult(batchHasProgress, err != nil, batchNum+1)
		
		if err != nil {
			s.logError("Batch %d failed: %v", batchNum+1, err)
			
			// Handle based on failure strategy
			switch s.config.FailureStrategy {
			case "fail-fast":
				result.EndTime = time.Now()
				result.TotalTime = result.EndTime.Sub(startTime)
				result.Results = append(result.Results, batchResults...)
				s.countResults(result)
				return result, fmt.Errorf("batch %d failed: %w", batchNum+1, err)
			case "continue":
				// Continue with next batch
				result.Results = append(result.Results, batchResults...)
				continue
			}
		}

		result.Results = append(result.Results, batchResults...)
		s.logInfo("Batch %d completed", batchNum+1)
	}

	result.EndTime = time.Now()
	result.TotalTime = result.EndTime.Sub(startTime)
	s.countResults(result)

	return result, nil
}

// executeBatch executes a single batch of tasks in parallel
func (s *Scheduler) executeBatch(ctx context.Context, graph *TaskGraph, batch []*task.Task) ([]*TaskResult, error) {
	workers := s.config.MaxWorkers
	if workers > len(batch) {
		workers = len(batch)
	}

	pool := NewWorkerPoolWithConfig(ctx, s.provider, s.workDir, WorkerPoolConfig{
		Workers:      workers,
		UseIsolation: s.config.IsolatedWorkspaces,
		Logger:       s.parallelLogger,
		StreamOutput: false, // Parallel mode should not stream to avoid mixed output
		MaxRetries:   s.config.MaxRetries,
	})
	pool.Start()

	// Mark tasks as running and submit to pool
	for _, t := range batch {
		if err := graph.MarkRunning(t.ID); err != nil {
			s.logError("Failed to mark task %s as running: %v", t.ID, err)
		}
		if err := pool.Submit(t); err != nil {
			return nil, fmt.Errorf("failed to submit task %s: %w", t.ID, err)
		}
	}

	// Collect results
	results := pool.WaitForBatch(len(batch))

	// Update graph based on results
	var batchErr error
	var successfulTasks []string
	for _, result := range results {
		if result.Success {
			if err := graph.MarkComplete(result.TaskID); err != nil {
				s.logError("Failed to mark task %s as complete: %v", result.TaskID, err)
			}
			// Print completion message to console
			fmt.Printf("[COMPLETED] %s %s (%.0fs)\n", result.TaskID, result.TaskName, result.Duration.Seconds())
			s.logInfo("Task %s completed successfully in %v", result.TaskID, result.Duration)
			successfulTasks = append(successfulTasks, result.TaskID)
		} else {
			if err := graph.MarkFailed(result.TaskID); err != nil {
				s.logError("Failed to mark task %s as failed: %v", result.TaskID, err)
			}
			// Print failure message to console
			fmt.Printf("[FAILED] %s %s: %v\n", result.TaskID, result.TaskName, result.Error)
			s.logError("Task %s failed: %v", result.TaskID, result.Error)
			batchErr = fmt.Errorf("task %s failed: %w", result.TaskID, result.Error)
		}
	}

	// Merge and cleanup workspaces for isolated execution
	if s.config.IsolatedWorkspaces && len(successfulTasks) > 0 {
		s.logInfo("Merging %d successful task branches...", len(successfulTasks))
		for _, taskID := range successfulTasks {
			workspace := pool.GetWorkspace(taskID)
			if workspace != nil && workspace.IsIsolated() {
				// Merge branch to main
				if err := s.mergeBranch(workspace); err != nil {
					s.logError("Failed to merge branch for task %s: %v", taskID, err)
				} else {
					s.logInfo("Merged branch %s for task %s", workspace.GetBranch(), taskID)
				}
				// Cleanup worktree
				if err := workspace.Cleanup(); err != nil {
					s.logError("Failed to cleanup workspace for task %s: %v", taskID, err)
				}
			}
		}
	}

	// Stop the pool
	pool.Stop()

	return results, batchErr
}

// mergeBranch merges a workspace branch back to the base branch
func (s *Scheduler) mergeBranch(workspace *isolation.Workspace) error {
	// Get current branch (should be base branch)
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = s.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	baseBranch := strings.TrimSpace(string(output))

	// Merge the task branch
	cmd = exec.Command("git", "merge", workspace.GetBranch(), "--no-edit", "-m", 
		fmt.Sprintf("Merge branch '%s' (task %s)", workspace.GetBranch(), workspace.TaskID))
	cmd.Dir = s.workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		// Check if it's a merge conflict
		if strings.Contains(string(output), "CONFLICT") {
			s.logError("Merge conflict detected for %s, attempting auto-resolution...", workspace.TaskID)
			// Try to abort and use theirs strategy
			exec.Command("git", "merge", "--abort").Run()
			cmd = exec.Command("git", "merge", workspace.GetBranch(), "--no-edit", "-X", "theirs", "-m",
				fmt.Sprintf("Merge branch '%s' (task %s) with auto-resolution", workspace.GetBranch(), workspace.TaskID))
			cmd.Dir = s.workDir
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("merge failed even with auto-resolution: %w: %s", err, string(output))
			}
		} else {
			return fmt.Errorf("merge failed: %w: %s", err, string(output))
		}
	}

	s.logInfo("Successfully merged %s into %s", workspace.GetBranch(), baseBranch)

	// Keep merged branches for history (branch deletion disabled)

	return nil
}

// countResults updates the result counts
func (s *Scheduler) countResults(result *ExecutionResult) {
	for _, r := range result.Results {
		if r.Success {
			result.Successful++
		} else {
			result.Failed++
		}
	}
}

func (s *Scheduler) logInfo(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Info(format, args...)
	}
}

func (s *Scheduler) logError(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Error(format, args...)
	}
}

// PrintExecutionPlan prints the execution plan in a user-friendly format
func (s *Scheduler) PrintExecutionPlan(plan *ExecutionPlan) {
	fmt.Println("\n📋 Execution Plan")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Total Tasks: %d\n", plan.TotalTasks)
	fmt.Printf("Batches: %d\n", len(plan.Batches))
	fmt.Printf("Max Workers: %d\n\n", s.config.MaxWorkers)

	for i, batch := range plan.Batches {
		fmt.Printf("Batch %d (%d tasks):\n", i+1, len(batch))
		for _, t := range batch {
			parallel := "✓"
			if !t.Parallelizable {
				parallel = "✗"
			}
			fmt.Printf("  [%s] %s - %s (parallel: %s)\n", t.ID, t.Name, t.Priority, parallel)
			if len(t.DependsOn) > 0 {
				fmt.Printf("       └─ depends on: %v\n", t.DependsOn)
			}
		}
		if i < len(plan.Batches)-1 {
			fmt.Println("  ↓")
		}
	}
	fmt.Println("═══════════════════════════════════════")
}

// PrintExecutionResult prints the execution result summary
func (s *Scheduler) PrintExecutionResult(result *ExecutionResult) {
	fmt.Println("\n📊 Execution Result")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Total Time: %v\n", result.TotalTime.Round(time.Second))
	fmt.Printf("Successful: %d\n", result.Successful)
	fmt.Printf("Failed: %d\n", result.Failed)
	fmt.Println()

	for _, r := range result.Results {
		status := "✓"
		if !r.Success {
			status = "✗"
		}
		fmt.Printf("[%s] %s - %s (%v)\n", status, r.TaskID, r.TaskName, r.Duration.Round(time.Second))
		if r.Error != nil {
			fmt.Printf("     Error: %v\n", r.Error)
		}
	}
	fmt.Println("═══════════════════════════════════════")
}
