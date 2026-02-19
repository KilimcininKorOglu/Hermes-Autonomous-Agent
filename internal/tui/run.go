package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// RunModel is the model for the run screen
type RunModel struct {
	width      int
	height     int
	basePath   string
	config     *config.Config
	focusIndex int
	logger     *ui.Logger

	// Run state
	running     bool
	paused      bool
	loopCount   int
	status      string
	lastError   string
	currentTask string
	startTime   time.Time
	cancel      context.CancelFunc

	// Parallel execution state
	parallelRunning    bool
	parallelBatch      int
	parallelTotalBatch int
	workerStatus       []string
	progressChan       chan scheduler.ProgressEvent

	// Components
	taskReader *task.Reader
	breaker    *circuit.Breaker

	// Progress tracking
	completedTasks int
	totalTasks     int
	taskHistory    []string
}

// runTickMsg for updating elapsed time
type runTickMsg time.Time

// runTaskCompleteMsg when a task completes
type runTaskCompleteMsg struct {
	taskID  string
	success bool
	err     error
}

// runStoppedMsg when run is stopped
type runStoppedMsg struct{}

// parallelCompleteMsg when parallel execution completes
type parallelCompleteMsg struct {
	successful int
	failed     int
	err        error
}

// parallelProgressMsg for parallel execution progress updates
type parallelProgressMsg struct {
	batch       int
	totalBatch  int
	workerID    int
	taskID      string
	status      string
}

// NewRunModel creates a new run model
func NewRunModel(basePath string, logger *ui.Logger) *RunModel {
	cfg, err := config.Load(basePath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	breaker := circuit.New(basePath)
	reader := task.NewReader(basePath)
	reader.SetImplicitDocDependencies(cfg.Parallel.ImplicitDocDependencies)

	// Set up circuit breaker state change logging
	if logger != nil {
		breaker.SetStateChangeCallback(func(fromState, toState circuit.State, reason string) {
			logger.Info("Circuit breaker: %s -> %s (%s)", fromState, toState, reason)
		})
	}

	// Count tasks
	features, _ := reader.GetAllFeatures()
	totalTasks := 0
	completedTasks := 0
	for _, f := range features {
		for _, t := range f.Tasks {
			totalTasks++
			if t.Status == task.StatusCompleted {
				completedTasks++
			}
		}
	}

	return &RunModel{
		basePath:       basePath,
		config:         cfg,
		taskReader:     reader,
		breaker:        breaker,
		logger:         logger,
		totalTasks:     totalTasks,
		completedTasks: completedTasks,
		taskHistory:    make([]string, 0),
		workerStatus:   make([]string, 0),
	}
}

// Init initializes the model
func (m *RunModel) Init() tea.Cmd {
	return nil
}

// SetSize sets the size of the model
func (m *RunModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Refresh reloads the configuration and task status
func (m *RunModel) Refresh() {
	cfg, err := config.Load(m.basePath)
	if err == nil {
		m.config = cfg
	}

	features, _ := m.taskReader.GetAllFeatures()
	m.totalTasks = 0
	m.completedTasks = 0
	for _, f := range features {
		for _, t := range f.Tasks {
			m.totalTasks++
			if t.Status == task.StatusCompleted {
				m.completedTasks++
			}
		}
	}
}

// Update handles messages
func (m *RunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.running {
			// Only handle run-specific keys when running
			// Let other keys (1-9, 0, q, etc.) pass through to app.go for screen navigation
			switch msg.String() {
			case "s", "esc":
				return m, m.stopRun()
			case "p":
				if !m.parallelRunning {
					m.paused = !m.paused
					if m.paused {
						m.status = "Paused"
						if m.logger != nil {
							m.logger.Info("Execution paused by user")
						}
					} else {
						m.status = "Running"
						if m.logger != nil {
							m.logger.Info("Execution resumed by user")
						}
					}
				}
			}
			// Don't block other keys - let them pass to app.go
			return m, nil
		}
		// Not running - handle navigation keys
		switch msg.String() {
		case "j", "down":
			m.focusIndex++
			if m.focusIndex > 4 {
				m.focusIndex = 0
			}
		case "k", "up":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 4
			}
		case " ", "enter":
			return m, m.handleSelect()
		case "x":
			// Reset circuit breaker
			if m.isCircuitBreakerOpen() {
				m.breaker.Reset("Manual reset from TUI")
				m.lastError = ""
				m.Refresh()
			}
		}

	case runTickMsg:
		if m.running && !m.paused {
			// Check progress channel for updates (non-blocking)
			if m.parallelRunning && m.progressChan != nil {
				m.drainProgressChannel()
			}
			return m, m.runTickCmd()
		}

	case runTaskCompleteMsg:
		m.handleTaskComplete(msg)
		if m.running && !m.paused && !m.parallelRunning {
			return m, m.executeNextTask()
		}

	case parallelProgressMsg:
		m.parallelBatch = msg.batch
		m.parallelTotalBatch = msg.totalBatch
		if msg.workerID > 0 && msg.workerID <= len(m.workerStatus) {
			m.workerStatus[msg.workerID-1] = fmt.Sprintf("W%d: %s - %s", msg.workerID, msg.taskID, msg.status)
		}
		m.status = fmt.Sprintf("Batch %d/%d", msg.batch, msg.totalBatch)

	case parallelCompleteMsg:
		m.running = false
		m.parallelRunning = false
		m.Refresh()
		if msg.err != nil {
			m.lastError = msg.err.Error()
			m.status = "Failed"
			entry := fmt.Sprintf("[ERROR] Parallel: %s", msg.err.Error())
			m.taskHistory = append(m.taskHistory, entry)
		} else {
			m.status = fmt.Sprintf("Completed: %d success, %d failed", msg.successful, msg.failed)
			entry := fmt.Sprintf("[DONE] Parallel: %d/%d tasks", msg.successful, msg.successful+msg.failed)
			m.taskHistory = append(m.taskHistory, entry)
		}

	case runStoppedMsg:
		m.running = false
		m.parallelRunning = false
		m.status = "Stopped"
		m.currentTask = ""
	}

	return m, nil
}

func (m *RunModel) handleSelect() tea.Cmd {
	switch m.focusIndex {
	case 0: // Parallel toggle
		m.config.Parallel.Enabled = !m.config.Parallel.Enabled
	case 1: // Workers
		m.config.Parallel.MaxWorkers++
		if m.config.Parallel.MaxWorkers > 10 {
			m.config.Parallel.MaxWorkers = 1
		}
	case 2: // Auto Branch
		m.config.TaskMode.AutoBranch = !m.config.TaskMode.AutoBranch
	case 3: // Auto Commit
		m.config.TaskMode.AutoCommit = !m.config.TaskMode.AutoCommit
	case 4: // Start/Stop button
		if m.running {
			return m.stopRun()
		}
		return m.startRun()
	}
	return nil
}

func (m *RunModel) startRun() tea.Cmd {
	m.running = true
	m.paused = false
	m.loopCount = 0
	m.startTime = time.Now()
	m.status = "Starting..."
	m.lastError = ""
	m.taskHistory = make([]string, 0)

	if m.logger != nil {
		m.logger.Info("Execution started (mode: %s)", func() string {
			if m.config.Parallel.Enabled {
				return fmt.Sprintf("parallel, workers: %d", m.config.Parallel.MaxWorkers)
			}
			return "sequential"
		}())
	}

	if m.config.Parallel.Enabled {
		return m.startParallelRun()
	}
	return tea.Batch(
		m.executeNextTask(),
		m.runTickCmd(),
	)
}

func (m *RunModel) startParallelRun() tea.Cmd {
	m.parallelRunning = true
	m.status = "Starting parallel execution..."
	
	workers := m.config.Parallel.MaxWorkers
	if workers < 1 {
		workers = 3
	}
	m.workerStatus = make([]string, workers)
	for i := range m.workerStatus {
		m.workerStatus[i] = fmt.Sprintf("W%d: Idle", i+1)
	}

	return tea.Batch(
		m.executeParallel(),
		m.runTickCmd(),
	)
}

func (m *RunModel) executeParallel() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel

		// Get provider
		var provider ai.Provider
		if m.config.AI.Coding != "" {
			provider = ai.GetProvider(m.config.AI.Coding)
		}
		if provider == nil || !provider.IsAvailable() {
			provider = ai.AutoDetectProvider()
		}
		if provider == nil {
			return parallelCompleteMsg{err: fmt.Errorf("no AI provider available")}
		}

		// Get all pending tasks
		allTasks, err := m.taskReader.GetAllTasks()
		if err != nil {
			return parallelCompleteMsg{err: err}
		}

		// Convert to pointer slice for scheduler (scheduler needs all tasks for dependency resolution)
		var allTaskPtrs []*task.Task
		for i := range allTasks {
			allTaskPtrs = append(allTaskPtrs, &allTasks[i])
		}

		// Count pending tasks
		pendingCount := 0
		for _, t := range allTasks {
			if t.Status == task.StatusNotStarted || t.Status == task.StatusInProgress {
				pendingCount++
			}
		}

		if pendingCount == 0 {
			return parallelCompleteMsg{successful: 0, failed: 0}
		}

		// Create scheduler (nil logger for TUI - no stdout output)
		workers := m.config.Parallel.MaxWorkers
		if workers < 1 {
			workers = 3
		}

		parallelCfg := &m.config.Parallel
		taskTimeout := time.Duration(m.config.AI.Timeout) * time.Second
		sched := scheduler.NewWithTimeout(parallelCfg, provider, m.basePath, m.logger, taskTimeout)

		// Set up progress callback to update worker status
		m.progressChan = make(chan scheduler.ProgressEvent, 100)
		sched.SetProgressCallback(func(event scheduler.ProgressEvent) {
			select {
			case m.progressChan <- event:
			default:
				// Channel full, skip update
			}
		})

		// Initialize parallel logger (writes to files only)
		parallelLogger, err := scheduler.NewParallelLogger(m.basePath, workers)
		if err == nil {
			defer parallelLogger.Close()
			sched.SetParallelLogger(parallelLogger)
		}

		// Execute (pass all tasks so scheduler can resolve dependencies correctly)
		result, err := sched.Execute(ctx, allTaskPtrs)

		// Close progress channel
		close(m.progressChan)

		if err != nil {
			successful := 0
			failed := 0
			if result != nil {
				successful = result.Successful
				failed = result.Failed
			}
			return parallelCompleteMsg{successful: successful, failed: failed, err: err}
		}

		return parallelCompleteMsg{successful: result.Successful, failed: result.Failed}
	}
}

func (m *RunModel) stopRun() tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	if m.logger != nil {
		m.logger.Info("Execution stopped by user")
	}
	return func() tea.Msg {
		return runStoppedMsg{}
	}
}

// IsRunning returns true if the run is active
func (m *RunModel) IsRunning() bool {
	return m.running
}

// Stop stops the current run
func (m *RunModel) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// DrainProgress drains progress events from the channel (called by App on tick)
func (m *RunModel) DrainProgress() {
	if m.parallelRunning && m.progressChan != nil {
		m.drainProgressChannel()
	}
}

// GetStatus returns the current run status for display in other screens
func (m *RunModel) GetStatus() (running bool, parallel bool, status string, completed int, total int) {
	return m.running, m.parallelRunning, m.status, m.completedTasks, m.totalTasks
}

// isCircuitBreakerOpen checks if the circuit breaker is open
func (m *RunModel) isCircuitBreakerOpen() bool {
	state, err := m.breaker.GetState()
	if err != nil {
		return false
	}
	return state.State == "OPEN"
}

func (m *RunModel) runTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return runTickMsg(t)
	})
}

func (m *RunModel) executeNextTask() tea.Cmd {
	return func() tea.Msg {
		timeout := time.Duration(m.config.AI.Timeout) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		m.cancel = cancel
		defer cancel()

		// Check circuit breaker
		canExecute, _ := m.breaker.CanExecute()
		if !canExecute {
			m.running = false
			state, _ := m.breaker.GetState()
			if m.logger != nil {
				m.logger.Error("Circuit breaker OPEN: %s", state.Reason)
			}
			return runTaskCompleteMsg{err: fmt.Errorf("circuit breaker open: %s", state.Reason)}
		}

		// Get next task
		nextTask, err := m.taskReader.GetNextTask()
		if err != nil {
			return runTaskCompleteMsg{err: err}
		}
		if nextTask == nil {
			m.running = false
			m.status = "All tasks completed"
			return runStoppedMsg{}
		}

		m.loopCount++
		m.currentTask = nextTask.ID
		m.status = fmt.Sprintf("Loop #%d: %s", m.loopCount, nextTask.ID)

		// Set task status to IN_PROGRESS before starting
		statusUpdater := task.NewStatusUpdater(m.basePath)
		if err := statusUpdater.UpdateTaskStatus(nextTask.ID, task.StatusInProgress); err != nil {
			if m.logger != nil {
				m.logger.Warn("Failed to set task IN_PROGRESS: %v", err)
			}
		}

		if m.logger != nil {
			m.logger.Info("Starting task: %s - %s", nextTask.ID, nextTask.Name)
		}

		// Handle branching
		gitOps := git.New(m.basePath)
		if m.config.TaskMode.AutoBranch && gitOps.IsRepository() {
			feature, _ := m.taskReader.GetFeatureByID(nextTask.FeatureID)
			if feature != nil {
				branchName, err := gitOps.CreateFeatureBranch(feature.ID, feature.Name)
				if err == nil && m.logger != nil {
					m.logger.Info("On branch: %s", branchName)
				}
			}
		}

		// Inject task into prompt
		injector := prompt.NewInjector(m.basePath)
		injector.AddTask(nextTask)
		promptContent, _ := injector.Read()

		// Get provider from config
		var provider ai.Provider
		if m.config.AI.Coding != "" {
			provider = ai.GetProvider(m.config.AI.Coding)
		}
		if provider == nil || !provider.IsAvailable() {
			provider = ai.AutoDetectProvider()
		}
		if provider == nil {
			m.running = false
			return runTaskCompleteMsg{err: fmt.Errorf("no AI provider available")}
		}

		// Execute AI (no streaming in TUI)
		executor := ai.NewTaskExecutor(provider, m.basePath)
		result, err := executor.ExecuteTask(ctx, nextTask, promptContent, false)

		if err != nil {
			m.breaker.AddLoopResultWithErrorLimit(false, true, m.loopCount, m.config.TaskMode.MaxConsecutiveErrors)
			return runTaskCompleteMsg{taskID: nextTask.ID, err: err}
		}

		// Analyze response
		respAnalyzer := analyzer.NewResponseAnalyzer()

		// Check if HERMES_STATUS block is present - if not, treat as error
		if !respAnalyzer.HasStatusBlock(result.Output) {
			if m.logger != nil {
				m.logger.Warn("Task %s missing HERMES_STATUS block - will retry", nextTask.ID)
			}
			m.breaker.AddLoopResultWithErrorLimit(false, true, m.loopCount, m.config.TaskMode.MaxConsecutiveErrors)
			return runTaskCompleteMsg{taskID: nextTask.ID, err: fmt.Errorf("missing HERMES_STATUS block")}
		}

		analysis := respAnalyzer.AnalyzeWithCriteria(result.Output, nextTask.SuccessCriteria)

		// Update circuit breaker
		m.breaker.AddLoopResultWithErrorLimit(analysis.HasProgress, false, m.loopCount, m.config.TaskMode.MaxConsecutiveErrors)

		// Handle blocked status
		if analysis.IsBlocked {
			if m.logger != nil {
				m.logger.Warn("Task %s is BLOCKED: %s", nextTask.ID, analysis.Recommendation)
			}
			statusUpdater.UpdateTaskStatus(nextTask.ID, task.StatusBlocked)
			injector.RemoveTask()
			return runTaskCompleteMsg{taskID: nextTask.ID, success: false}
		}

		// Handle at-risk status
		if analysis.IsAtRisk {
			if m.logger != nil {
				m.logger.Warn("Task %s is AT RISK: %s", nextTask.ID, analysis.Recommendation)
			}
			statusUpdater.UpdateTaskStatus(nextTask.ID, task.StatusAtRisk)
		}

		// Handle paused status
		if analysis.IsPaused {
			if m.logger != nil {
				m.logger.Info("Task %s is PAUSED: %s", nextTask.ID, analysis.Recommendation)
			}
			statusUpdater.UpdateTaskStatus(nextTask.ID, task.StatusPaused)
			injector.RemoveTask()
			return runTaskCompleteMsg{taskID: nextTask.ID, success: false}
		}

		// Update task status if complete
		if analysis.IsComplete {
			injector.RemoveTask()
			statusUpdater.UpdateTaskStatus(nextTask.ID, task.StatusCompleted)

			// Auto-commit
			if m.config.TaskMode.AutoCommit && gitOps.HasUncommittedChanges() {
				if err := gitOps.StageAll(); err == nil {
					if err := gitOps.CommitTask(nextTask.ID, nextTask.Name); err != nil {
						if m.logger != nil {
							m.logger.Warn("Failed to commit: %v", err)
						}
					} else if m.logger != nil {
						m.logger.Success("Committed task %s", nextTask.ID)
					}
				}
			}

			// Check if feature is complete and create tag
			if featureComplete, _ := m.taskReader.IsFeatureComplete(nextTask.FeatureID); featureComplete {
				feature, _ := m.taskReader.GetFeatureByID(nextTask.FeatureID)
				if feature != nil {
					if m.logger != nil {
						m.logger.Success("Feature %s completed: %s", feature.ID, feature.Name)
					}

					// Merge feature branch to main if auto-branch is enabled
					if m.config.TaskMode.AutoBranch && gitOps.IsRepository() {
						if err := gitOps.MergeFeatureBranch(feature.ID, feature.Name); err != nil {
							if m.logger != nil {
								m.logger.Warn("Failed to merge feature branch: %v", err)
							}
						} else if m.logger != nil {
							m.logger.Success("Merged feature branch to %s", gitOps.GetMainBranch())
						}
					}

					// Create git tag if TargetVersion is set
					if feature.TargetVersion != "" && gitOps.IsRepository() {
						if err := gitOps.CreateFeatureTag(feature.ID, feature.Name, feature.TargetVersion); err != nil {
							if m.logger != nil {
								m.logger.Warn("Failed to create tag: %v", err)
							}
						} else if m.logger != nil {
							m.logger.Success("Created tag: %s", feature.TargetVersion)
						}
					}
				}
			}

			m.completedTasks++
		}

		return runTaskCompleteMsg{taskID: nextTask.ID, success: analysis.IsComplete}
	}
}

func (m *RunModel) handleTaskComplete(msg runTaskCompleteMsg) {
	if msg.err != nil {
		m.lastError = msg.err.Error()
		entry := fmt.Sprintf("[ERROR] %s: %s", msg.taskID, msg.err.Error())
		m.taskHistory = append(m.taskHistory, entry)
		if m.logger != nil {
			m.logger.Error("Task %s failed: %s", msg.taskID, msg.err.Error())
		}
	} else if msg.success {
		entry := fmt.Sprintf("[DONE] %s", msg.taskID)
		m.taskHistory = append(m.taskHistory, entry)
		if m.logger != nil {
			m.logger.Success("Task %s completed", msg.taskID)
		}
	} else {
		entry := fmt.Sprintf("[PROGRESS] %s", msg.taskID)
		m.taskHistory = append(m.taskHistory, entry)
	}

	// Keep only last 10 entries
	if len(m.taskHistory) > 10 {
		m.taskHistory = m.taskHistory[len(m.taskHistory)-10:]
	}
}

// drainProgressChannel reads all pending progress events from the channel
func (m *RunModel) drainProgressChannel() {
	for {
		select {
		case event, ok := <-m.progressChan:
			if !ok {
				return
			}
			// Update worker status
			if event.WorkerID > 0 && event.WorkerID <= len(m.workerStatus) {
				statusText := event.Status
				if event.TaskID != "" {
					if event.TaskName != "" {
						statusText = fmt.Sprintf("%s (%s): %s", event.TaskID, event.TaskName, event.Status)
					} else {
						statusText = fmt.Sprintf("%s: %s", event.TaskID, event.Status)
					}
				}
				m.workerStatus[event.WorkerID-1] = fmt.Sprintf("W%d: %s", event.WorkerID, statusText)
			}
			// Update batch info
			if event.Batch > 0 {
				m.parallelBatch = event.Batch
				m.parallelTotalBatch = event.TotalBatch
				m.status = fmt.Sprintf("Batch %d/%d", event.Batch, event.TotalBatch)
			}
			// Update completed tasks count
			if event.Status == "completed" {
				m.completedTasks++
			}
		default:
			// No more events
			return
		}
	}
}

// View renders the model
func (m *RunModel) View() string {
	var b strings.Builder

	workerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))

	b.WriteString(RenderScreenTitle("RUN TASKS"))

	// Progress
	b.WriteString(SectionStyle.Render("Progress"))
	b.WriteString("\n")
	progressPercent := 0
	if m.totalTasks > 0 {
		progressPercent = (m.completedTasks * 100) / m.totalTasks
	}
	progressBar := m.renderProgressBar(progressPercent, 40)
	b.WriteString(fmt.Sprintf("  %s %d/%d tasks (%d%%)\n", progressBar, m.completedTasks, m.totalTasks, progressPercent))

	// Circuit Breaker Warning
	if m.isCircuitBreakerOpen() {
		warningStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color("52")).
			Padding(0, 1)
		b.WriteString("\n")
		b.WriteString(warningStyle.Render("CIRCUIT BREAKER OPEN - Execution halted due to no progress"))
		b.WriteString("\n")
		b.WriteString(ErrorStyle.Render("  Press 'x' to reset and continue"))
		b.WriteString("\n")
	}

	// Status line (always show to maintain consistent layout)
	if m.running {
		elapsed := time.Since(m.startTime).Round(time.Second)
		modeStr := "Sequential"
		if m.parallelRunning {
			modeStr = fmt.Sprintf("Parallel (%d workers)", m.config.Parallel.MaxWorkers)
		}
		b.WriteString(fmt.Sprintf("  Mode: %s | Status: %s | Elapsed: %v\n", modeStr, SuccessStyle.Render(m.status), elapsed))
		
		if m.parallelRunning && len(m.workerStatus) > 0 {
			b.WriteString("\n")
			b.WriteString(SectionStyle.Render("Workers"))
			b.WriteString("\n")
			for _, ws := range m.workerStatus {
				b.WriteString(fmt.Sprintf("  %s\n", workerStyle.Render(ws)))
			}
		} else if m.currentTask != "" {
			b.WriteString(fmt.Sprintf("  Current: %s | Loop: %d\n", m.currentTask, m.loopCount))
		}
	} else {
		// Show idle status when not running
		b.WriteString(fmt.Sprintf("  Mode: - | Status: %s | Elapsed: -\n", MutedStyle.Render("Idle")))
		b.WriteString("  \n") // Placeholder for Current line
	}
	b.WriteString("\n")

	// Options (only editable when not running)
	b.WriteString(SectionStyle.Render("Options"))
	b.WriteString("\n")

	if !m.running {
		m.renderOption(&b, 0, LabelStyle, SelectedStyle, "Parallel Mode:", m.boolToStr(m.config.Parallel.Enabled))
		m.renderOption(&b, 1, LabelStyle, SelectedStyle, "Workers:", fmt.Sprintf("%d", m.config.Parallel.MaxWorkers))
		m.renderOption(&b, 2, LabelStyle, SelectedStyle, "Auto Branch:", m.boolToStr(m.config.TaskMode.AutoBranch))
		m.renderOption(&b, 3, LabelStyle, SelectedStyle, "Auto Commit:", m.boolToStr(m.config.TaskMode.AutoCommit))
	} else {
		b.WriteString(fmt.Sprintf("  Parallel: %s | Workers: %d | Branch: %s | Commit: %s\n",
			m.boolToStr(m.config.Parallel.Enabled),
			m.config.Parallel.MaxWorkers,
			m.boolToStr(m.config.TaskMode.AutoBranch),
			m.boolToStr(m.config.TaskMode.AutoCommit)))
	}
	b.WriteString("\n")

	// Start/Stop button
	if !m.running {
		if m.focusIndex == 4 {
			b.WriteString(SelectedStyle.Render("> "))
		} else {
			b.WriteString("  ")
		}
		if m.config.Parallel.Enabled {
			b.WriteString(ButtonStyle.Render("Start Parallel Run"))
		} else {
			b.WriteString(ButtonStyle.Render("Start Run"))
		}
	} else {
		b.WriteString("  ")
		b.WriteString(ActiveButtonStyle.Render("Stop Run (s/esc)"))
		if !m.parallelRunning {
			if m.paused {
				b.WriteString("  ")
				b.WriteString(ValueStyle.Render("[PAUSED - press 'p' to resume]"))
			} else {
				b.WriteString("  ")
				b.WriteString(ValueStyle.Render("[press 'p' to pause]"))
			}
		}
	}
	b.WriteString("\n\n")

	// Error display
	if m.lastError != "" {
		b.WriteString(SectionStyle.Render("Last Error"))
		b.WriteString("\n")
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("  %s", m.lastError)))
		b.WriteString("\n\n")
	}

	// Task history
	if len(m.taskHistory) > 0 {
		b.WriteString(SectionStyle.Render("Recent Activity"))
		b.WriteString("\n")
		for _, entry := range m.taskHistory {
			line := fmt.Sprintf("  %s", entry)
			if strings.HasPrefix(entry, "[DONE]") {
				b.WriteString(SuccessStyle.Render(line))
			} else if strings.HasPrefix(entry, "[ERROR]") {
				b.WriteString(ErrorStyle.Render(line))
			} else {
				b.WriteString(ValueStyle.Render(line))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *RunModel) renderOption(b *strings.Builder, index int, LabelStyle, SelectedStyle lipgloss.Style, label, value string) {
	if m.focusIndex == index {
		b.WriteString(SelectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(LabelStyle.Render(label))
	b.WriteString(value)
	b.WriteString("\n")
}

func (m *RunModel) renderProgressBar(percent, width int) string {
	filled := (percent * width) / 100
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("[%s]", bar)
}

func (m *RunModel) boolToStr(v bool) string {
	if v {
		return "On"
	}
	return "Off"
}
