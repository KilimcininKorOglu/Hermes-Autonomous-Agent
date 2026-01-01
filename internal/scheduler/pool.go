package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"hermes/internal/ai"
	"hermes/internal/isolation"
	"hermes/internal/task"
)

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
	workers        int
	taskQueue      chan *task.Task
	results        chan *TaskResult
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	provider       ai.Provider
	workDir        string
	mu             sync.Mutex
	running        int
	useIsolation   bool
	workspaces     map[string]*isolation.Workspace
	logger         *ParallelLogger
	streamOutput   bool
}

// WorkerPoolConfig contains configuration for the worker pool
type WorkerPoolConfig struct {
	Workers      int
	UseIsolation bool
	Logger       *ParallelLogger
	StreamOutput bool
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
	return &WorkerPool{
		workers:      cfg.Workers,
		taskQueue:    make(chan *task.Task, cfg.Workers*2),
		results:      make(chan *TaskResult, cfg.Workers*2),
		ctx:          ctx,
		cancel:       cancel,
		provider:     provider,
		workDir:      workDir,
		useIsolation: cfg.UseIsolation,
		workspaces:   make(map[string]*isolation.Workspace),
		logger:       cfg.Logger,
		streamOutput: cfg.StreamOutput,
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
			result := p.executeTask(workerID, t)
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

// executeTask executes a single task and returns the result
func (p *WorkerPool) executeTask(workerID int, t *task.Task) *TaskResult {
	startTime := time.Now()

	result := &TaskResult{
		TaskID:    t.ID,
		TaskName:  t.Name,
		StartTime: startTime,
		WorkerID:  workerID + 1, // 1-indexed for display
	}

	// Log task start
	if p.logger != nil {
		p.logger.TaskStart(workerID+1, t.ID, t.Name)
	}

	// Setup isolated workspace if enabled
	workDir := p.workDir
	var workspace *isolation.Workspace
	if p.useIsolation {
		workspace = isolation.NewWorkspace(t.ID, p.workDir)
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

	// Create task executor with appropriate work directory
	executor := ai.NewTaskExecutor(p.provider, workDir)

	// Build prompt content from task
	promptContent := p.buildPromptContent(t)

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

	result.Success = true
	result.Output = execResult.Output

	// Log task completion
	if p.logger != nil {
		p.logger.TaskComplete(workerID+1, t.ID, result.Duration)
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

// buildPromptContent builds the prompt content for a task
func (p *WorkerPool) buildPromptContent(t *task.Task) string {
	content := fmt.Sprintf(`# Current Task

## Task ID: %s
## Task Name: %s
## Priority: %s
## Estimated Effort: %s

### Description
%s

### Technical Details
%s

### Files to Modify
%v

### Success Criteria
%v
`,
		t.ID,
		t.Name,
		t.Priority,
		t.EstimatedEffort,
		t.Description,
		t.TechnicalDetails,
		t.FilesToTouch,
		t.SuccessCriteria,
	)

	return content
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
