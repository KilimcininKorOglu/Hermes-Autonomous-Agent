package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"hermes/internal/ai"
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
	workers   int
	taskQueue chan *task.Task
	results   chan *TaskResult
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	provider  ai.Provider
	workDir   string
	mu        sync.Mutex
	running   int
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(ctx context.Context, workers int, provider ai.Provider, workDir string) *WorkerPool {
	ctx, cancel := context.WithCancel(ctx)
	return &WorkerPool{
		workers:   workers,
		taskQueue: make(chan *task.Task, workers*2),
		results:   make(chan *TaskResult, workers*2),
		ctx:       ctx,
		cancel:    cancel,
		provider:  provider,
		workDir:   workDir,
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
		WorkerID:  workerID,
	}

	// Create task executor
	executor := ai.NewTaskExecutor(p.provider, p.workDir)

	// Build prompt content from task
	promptContent := p.buildPromptContent(t)

	// Execute the task
	execResult, err := executor.ExecuteTask(p.ctx, t, promptContent, false)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)

	if err != nil {
		result.Success = false
		result.Error = err
		return result
	}

	result.Success = true
	result.Output = execResult.Output

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
