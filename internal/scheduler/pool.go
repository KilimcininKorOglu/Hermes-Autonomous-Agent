package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"hermes/internal/ai"
	"hermes/internal/analyzer"
	"hermes/internal/isolation"
	"hermes/internal/prompt"
	"hermes/internal/task"
)

// ProgressEvent represents a progress update from worker pool
type ProgressEvent struct {
	WorkerID   int
	TaskID     string
	TaskName   string
	Status     string // "started", "completed", "failed", "retrying"
	Batch      int
	TotalBatch int
}

// ProgressCallback is called when progress updates occur
type ProgressCallback func(event ProgressEvent)

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID    string
	TaskName  string
	Success   bool
	Output    string
	Error     error
	Branch    string
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
	WorkerID  int
}

// WorkerPool manages multiple AI agent instances for parallel execution
type WorkerPool struct {
	workers          int
	taskQueue        chan *task.Task
	results          chan *TaskResult
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	provider         ai.Provider
	workDir          string
	mu               sync.Mutex
	running          int
	useIsolation     bool
	workspaces       map[string]*isolation.Workspace
	logger           *ParallelLogger
	streamOutput     bool
	maxRetries       int
	progressCallback ProgressCallback
	currentBatch     int
	totalBatches     int
}

// WorkerPoolConfig contains configuration for the worker pool
type WorkerPoolConfig struct {
	Workers          int
	UseIsolation     bool
	Logger           *ParallelLogger
	StreamOutput     bool
	MaxRetries       int
	ProgressCallback ProgressCallback
	CurrentBatch     int
	TotalBatches     int
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(ctx context.Context, workers int, provider ai.Provider, workDir string) *WorkerPool {
	return NewWorkerPoolWithConfig(ctx, provider, workDir, WorkerPoolConfig{
		Workers:      workers,
		UseIsolation: false,
		Logger:       nil,
	})
}

// NewWorkerPoolWithConfig creates a new worker pool with configuration
func NewWorkerPoolWithConfig(ctx context.Context, provider ai.Provider, workDir string, cfg WorkerPoolConfig) *WorkerPool {
	ctx, cancel := context.WithCancel(ctx)
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2 // Default retry count
	}
	return &WorkerPool{
		workers:          cfg.Workers,
		taskQueue:        make(chan *task.Task, cfg.Workers*2),
		results:          make(chan *TaskResult, cfg.Workers*2),
		ctx:              ctx,
		cancel:           cancel,
		provider:         provider,
		workDir:          workDir,
		useIsolation:     cfg.UseIsolation,
		workspaces:       make(map[string]*isolation.Workspace),
		logger:           cfg.Logger,
		streamOutput:     cfg.StreamOutput,
		maxRetries:       maxRetries,
		progressCallback: cfg.ProgressCallback,
		currentBatch:     cfg.CurrentBatch,
		totalBatches:     cfg.TotalBatches,
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker is the main worker goroutine
func (p *WorkerPool) worker(workerID int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case t, ok := <-p.taskQueue:
			if !ok {
				return
			}
			p.incrementRunning()
			
			// Retry loop for task execution
			var result *TaskResult
			for attempt := 1; attempt <= p.maxRetries; attempt++ {
				result = p.executeTask(workerID, t, attempt)
				
				if result.Success {
					break // Task completed successfully
				}
				
				// Check if we should retry
				if attempt < p.maxRetries {
					if p.logger != nil {
						p.logger.Worker(workerID+1, "Task %s attempt %d/%d failed, retrying...", 
							t.ID, attempt, p.maxRetries)
					}
					// Small delay before retry
					select {
					case <-time.After(2 * time.Second):
					case <-p.ctx.Done():
						break
					}
				}
			}
			
			p.decrementRunning()
			
			select {
			case p.results <- result:
			case <-p.ctx.Done():
				return
			}
		}
	}
}

func (p *WorkerPool) incrementRunning() {
	p.mu.Lock()
	p.running++
	p.mu.Unlock()
}

func (p *WorkerPool) decrementRunning() {
	p.mu.Lock()
	p.running--
	p.mu.Unlock()
}

// GetRunningCount returns the number of currently running tasks
func (p *WorkerPool) GetRunningCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// notifyProgress sends a progress event to the callback if set
func (p *WorkerPool) notifyProgress(workerID int, taskID, taskName, status string) {
	if p.progressCallback != nil {
		p.progressCallback(ProgressEvent{
			WorkerID:   workerID,
			TaskID:     taskID,
			TaskName:   taskName,
			Status:     status,
			Batch:      p.currentBatch,
			TotalBatch: p.totalBatches,
		})
	}
}

// executeTask executes a single task and returns the result
func (p *WorkerPool) executeTask(workerID int, t *task.Task, attempt int) *TaskResult {
	startTime := time.Now()

	result := &TaskResult{
		TaskID:    t.ID,
		TaskName:  t.Name,
		StartTime: startTime,
		WorkerID:  workerID + 1, // 1-indexed for display
	}

	// Notify progress: started
	p.notifyProgress(workerID+1, t.ID, t.Name, "started")

	if attempt > 1 && p.logger != nil {
		p.logger.Worker(workerID+1, "Task %s attempt %d/%d", t.ID, attempt, p.maxRetries)
		p.notifyProgress(workerID+1, t.ID, t.Name, "retrying")
	}

	// Log task start
	if p.logger != nil {
		p.logger.TaskStart(workerID+1, t.ID, t.Name)
	}

	// Update task status to IN_PROGRESS
	statusUpdater := task.NewStatusUpdater(p.workDir)
	if err := statusUpdater.UpdateTaskStatus(t.ID, task.StatusInProgress); err != nil {
		if p.logger != nil {
			p.logger.Worker(workerID+1, "Failed to set task IN_PROGRESS: %v", err)
		}
	}

	// Setup isolated workspace if enabled
	workDir := p.workDir
	var workspace *isolation.Workspace
	if p.useIsolation {
		workspace = isolation.NewWorkspaceWithName(t.ID, t.Name, p.workDir)
		if err := workspace.Setup(); err != nil {
			// Fall back to shared workspace
			if p.logger != nil {
				p.logger.Worker(workerID+1, "Failed to create isolated workspace, using shared: %v", err)
			}
		} else {
			workDir = workspace.GetWorkPath()
			result.Branch = workspace.GetBranch()
			p.mu.Lock()
			p.workspaces[t.ID] = workspace
			p.mu.Unlock()
		}
	}

	// Inject task into PROMPT.md
	injector := prompt.NewInjector(workDir)
	if err := injector.AddTask(t); err != nil {
		if p.logger != nil {
			p.logger.Worker(workerID+1, "Failed to inject task into prompt: %v", err)
		}
	}

	// Create task executor with appropriate work directory
	executor := ai.NewTaskExecutor(p.provider, workDir)

	// Read prompt content (includes injected task)
	promptContent, _ := injector.Read()

	// Execute the task
	execResult, err := executor.ExecuteTask(p.ctx, t, promptContent, p.streamOutput)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)

	if err != nil {
		result.Success = false
		result.Error = err
		if p.logger != nil {
			p.logger.TaskFailed(workerID+1, t.ID, err)
		}
		return result
	}

	result.Output = execResult.Output

	// Analyze AI response to determine if task is truly complete
	respAnalyzer := analyzer.NewResponseAnalyzer()
	analysis := respAnalyzer.Analyze(execResult.Output)

	if p.logger != nil {
		p.logger.Worker(workerID+1, "Analysis: complete=%v blocked=%v atRisk=%v paused=%v progress=%v confidence=%.2f",
			analysis.IsComplete, analysis.IsBlocked, analysis.IsAtRisk, analysis.IsPaused, analysis.HasProgress, analysis.Confidence)
	}

	// Handle blocked status
	if analysis.IsBlocked {
		result.Success = false
		result.Error = fmt.Errorf("task blocked: %s", analysis.Recommendation)
		statusUpdater.UpdateTaskStatus(t.ID, task.StatusBlocked)
		if p.logger != nil {
			p.logger.Worker(workerID+1, "Task %s is BLOCKED: %s", t.ID, analysis.Recommendation)
		}
		return result
	}

	// Handle paused status
	if analysis.IsPaused {
		result.Success = false
		result.Error = fmt.Errorf("task paused: %s", analysis.Recommendation)
		statusUpdater.UpdateTaskStatus(t.ID, task.StatusPaused)
		if p.logger != nil {
			p.logger.Worker(workerID+1, "Task %s is PAUSED: %s", t.ID, analysis.Recommendation)
		}
		return result
	}

	// Handle at-risk status (continue but log warning)
	if analysis.IsAtRisk {
		statusUpdater.UpdateTaskStatus(t.ID, task.StatusAtRisk)
		if p.logger != nil {
			p.logger.Worker(workerID+1, "Task %s is AT RISK: %s", t.ID, analysis.Recommendation)
		}
		// Continue processing - at-risk doesn't stop execution
	}

	// Task is successful only if AI indicates completion
	if analysis.IsComplete {
		result.Success = true
		if p.logger != nil {
			p.logger.TaskComplete(workerID+1, t.ID, result.Duration)
		}
		p.notifyProgress(workerID+1, t.ID, t.Name, "completed")
		// Remove task from PROMPT.md only on completion
		injector.RemoveTask()
	} else {
		// AI did not indicate task completion
		result.Success = false
		result.Error = fmt.Errorf("task not completed by AI (progress=%v, confidence=%.2f)", 
			analysis.HasProgress, analysis.Confidence)
		if p.logger != nil {
			p.logger.Worker(workerID+1, "Task %s not marked as complete by AI", t.ID)
		}
		p.notifyProgress(workerID+1, t.ID, t.Name, "failed")
		return result
	}

	// Commit changes in isolated workspace
	if workspace != nil && workspace.HasUncommittedChanges() {
		commitMsg := fmt.Sprintf("Complete task %s: %s", t.ID, t.Name)
		if err := workspace.CommitChanges(commitMsg); err != nil {
			if p.logger != nil {
				p.logger.Worker(workerID+1, "Failed to commit changes: %v", err)
			}
		}
	}

	return result
}

// Submit submits a task for execution
func (p *WorkerPool) Submit(t *task.Task) error {
	select {
	case p.taskQueue <- t:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

// SubmitBatch submits multiple tasks for execution
func (p *WorkerPool) SubmitBatch(tasks []*task.Task) error {
	for _, t := range tasks {
		if err := p.Submit(t); err != nil {
			return err
		}
	}
	return nil
}

// Results returns the results channel
func (p *WorkerPool) Results() <-chan *TaskResult {
	return p.results
}

// Wait waits for all submitted tasks to complete
func (p *WorkerPool) Wait() {
	close(p.taskQueue)
	p.wg.Wait()
	close(p.results)
}

// Stop gracefully stops the worker pool
func (p *WorkerPool) Stop() {
	p.cancel()
	p.Wait()
}

// WaitForBatch waits for a specific number of results
func (p *WorkerPool) WaitForBatch(count int) []*TaskResult {
	var results []*TaskResult
	for i := 0; i < count; i++ {
		select {
		case result, ok := <-p.results:
			if !ok {
				return results
			}
			results = append(results, result)
		case <-p.ctx.Done():
			return results
		}
	}
	return results
}

// WorkerCount returns the number of workers
func (p *WorkerPool) WorkerCount() int {
	return p.workers
}

// IsRunning returns true if the pool has running tasks
func (p *WorkerPool) IsRunning() bool {
	return p.GetRunningCount() > 0
}

// GetWorkspaces returns all workspaces created by the pool
func (p *WorkerPool) GetWorkspaces() map[string]*isolation.Workspace {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.workspaces
}

// GetWorkspace returns the workspace for a specific task
func (p *WorkerPool) GetWorkspace(taskID string) *isolation.Workspace {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.workspaces[taskID]
}
