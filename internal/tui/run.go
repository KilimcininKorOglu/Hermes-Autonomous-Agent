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
	parallelRunning   bool
	parallelBatch     int
	parallelTotalBatch int
	workerStatus      []string

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
			switch msg.String() {
			case "s", "esc":
				return m, m.stopRun()
			case "p":
				if !m.parallelRunning {
					m.paused = !m.paused
					if m.paused {
						m.status = "Paused"
					} else {
						m.status = "Running"
					}
				}
			}
		} else {
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
			}
		}

	case runTickMsg:
		if m.running && !m.paused {
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

		if m.logger != nil {
			m.logger.Info("Using AI provider: %s", provider.Name())
		}

		// Get all pending tasks
		allTasks, err := m.taskReader.GetAllTasks()
		if err != nil {
			return parallelCompleteMsg{err: err}
		}

		// Filter pending tasks
		var pendingTasks []*task.Task
		for i := range allTasks {
			if allTasks[i].Status == task.StatusNotStarted || allTasks[i].Status == task.StatusInProgress {
				pendingTasks = append(pendingTasks, &allTasks[i])
			}
		}

		if len(pendingTasks) == 0 {
			return parallelCompleteMsg{successful: 0, failed: 0}
		}

		if m.logger != nil {
			m.logger.Info("Found %d pending tasks", len(pendingTasks))
		}

		// Create scheduler
		workers := m.config.Parallel.MaxWorkers
		if workers < 1 {
			workers = 3
		}

		parallelCfg := &m.config.Parallel
		sched := scheduler.New(parallelCfg, provider, m.basePath, m.logger)

		if m.logger != nil {
			m.logger.Info("Using %d parallel workers", workers)
		}

		// Initialize parallel logger
		parallelLogger, err := scheduler.NewParallelLogger(m.basePath, workers)
		if err != nil {
			if m.logger != nil {
				m.logger.Warn("Failed to initialize parallel logger: %v", err)
			}
		} else {
			defer parallelLogger.Close()
			if m.logger != nil {
				m.logger.Info("Logs will be written to: %s", parallelLogger.GetLogDirectory())
			}
			sched.SetParallelLogger(parallelLogger)
		}

		if m.logger != nil {
			m.logger.Info("Starting parallel execution...")
		}

		// Execute
		result, err := sched.Execute(ctx, pendingTasks)

		if err != nil {
			if m.logger != nil {
				m.logger.Error("Parallel execution failed: %v", err)
			}
			successful := 0
			failed := 0
			if result != nil {
				successful = result.Successful
				failed = result.Failed
			}
			return parallelCompleteMsg{successful: successful, failed: failed, err: err}
		}

		if m.logger != nil {
			m.logger.Success("All %d tasks completed successfully!", result.Successful)
		}

		return parallelCompleteMsg{successful: result.Successful, failed: result.Failed}
	}
}

func (m *RunModel) stopRun() tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	return func() tea.Msg {
		return runStoppedMsg{}
	}
}

func (m *RunModel) runTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return runTickMsg(t)
	})
}

func (m *RunModel) executeNextTask() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel

		// Check circuit breaker
		canExecute, _ := m.breaker.CanExecute()
		if !canExecute {
			m.running = false
			return runTaskCompleteMsg{err: fmt.Errorf("circuit breaker open")}
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

		if m.logger != nil {
			m.logger.Info("Starting task: %s - %s", nextTask.ID, nextTask.Name)
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
			m.breaker.AddLoopResult(false, true, m.loopCount)
			return runTaskCompleteMsg{taskID: nextTask.ID, err: err}
		}

		// Analyze response
		respAnalyzer := analyzer.NewResponseAnalyzer()
		analysis := respAnalyzer.Analyze(result.Output)

		// Update circuit breaker
		m.breaker.AddLoopResult(analysis.HasProgress, false, m.loopCount)

		// Update task status if complete
		if analysis.IsComplete {
			statusUpdater := task.NewStatusUpdater(m.basePath)
			statusUpdater.UpdateTaskStatus(nextTask.ID, task.StatusCompleted)
			injector.RemoveTask()
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
	} else if msg.success {
		entry := fmt.Sprintf("[DONE] %s", msg.taskID)
		m.taskHistory = append(m.taskHistory, entry)
	} else {
		entry := fmt.Sprintf("[PROGRESS] %s", msg.taskID)
		m.taskHistory = append(m.taskHistory, entry)
	}

	// Keep only last 10 entries
	if len(m.taskHistory) > 10 {
		m.taskHistory = m.taskHistory[len(m.taskHistory)-10:]
	}
}

// View renders the model
func (m *RunModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(20)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	statusStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("82"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("62")).
		Padding(0, 2)

	runningButtonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("196")).
		Padding(0, 2)

	workerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))

	b.WriteString(titleStyle.Render("RUN TASKS"))
	b.WriteString("\n\n")

	// Progress
	b.WriteString(sectionStyle.Render("Progress"))
	b.WriteString("\n")
	progressPercent := 0
	if m.totalTasks > 0 {
		progressPercent = (m.completedTasks * 100) / m.totalTasks
	}
	progressBar := m.renderProgressBar(progressPercent, 40)
	b.WriteString(fmt.Sprintf("  %s %d/%d tasks (%d%%)\n", progressBar, m.completedTasks, m.totalTasks, progressPercent))

	// Status
	if m.running {
		elapsed := time.Since(m.startTime).Round(time.Second)
		modeStr := "Sequential"
		if m.parallelRunning {
			modeStr = fmt.Sprintf("Parallel (%d workers)", m.config.Parallel.MaxWorkers)
		}
		b.WriteString(fmt.Sprintf("  Mode: %s | Status: %s | Elapsed: %v\n", modeStr, statusStyle.Render(m.status), elapsed))
		
		if m.parallelRunning && len(m.workerStatus) > 0 {
			b.WriteString("\n")
			b.WriteString(sectionStyle.Render("Workers"))
			b.WriteString("\n")
			for _, ws := range m.workerStatus {
				b.WriteString(workerStyle.Render(fmt.Sprintf("  %s\n", ws)))
			}
		} else if m.currentTask != "" {
			b.WriteString(fmt.Sprintf("  Current: %s | Loop: %d\n", m.currentTask, m.loopCount))
		}
	}
	b.WriteString("\n")

	// Options (only editable when not running)
	b.WriteString(sectionStyle.Render("Options"))
	b.WriteString("\n")

	if !m.running {
		m.renderOption(&b, 0, labelStyle, selectedStyle, "Parallel Mode:", m.boolToStr(m.config.Parallel.Enabled))
		m.renderOption(&b, 1, labelStyle, selectedStyle, "Workers:", fmt.Sprintf("%d", m.config.Parallel.MaxWorkers))
		m.renderOption(&b, 2, labelStyle, selectedStyle, "Auto Branch:", m.boolToStr(m.config.TaskMode.AutoBranch))
		m.renderOption(&b, 3, labelStyle, selectedStyle, "Auto Commit:", m.boolToStr(m.config.TaskMode.AutoCommit))
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
			b.WriteString(selectedStyle.Render("> "))
		} else {
			b.WriteString("  ")
		}
		if m.config.Parallel.Enabled {
			b.WriteString(buttonStyle.Render("Start Parallel Run"))
		} else {
			b.WriteString(buttonStyle.Render("Start Run"))
		}
	} else {
		b.WriteString("  ")
		b.WriteString(runningButtonStyle.Render("Stop Run (s/esc)"))
		if !m.parallelRunning {
			if m.paused {
				b.WriteString("  ")
				b.WriteString(valueStyle.Render("[PAUSED - press 'p' to resume]"))
			} else {
				b.WriteString("  ")
				b.WriteString(valueStyle.Render("[press 'p' to pause]"))
			}
		}
	}
	b.WriteString("\n\n")

	// Error display
	if m.lastError != "" {
		b.WriteString(sectionStyle.Render("Last Error"))
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("  %s", m.lastError)))
		b.WriteString("\n\n")
	}

	// Task history
	if len(m.taskHistory) > 0 {
		b.WriteString(sectionStyle.Render("Recent Activity"))
		b.WriteString("\n")
		for _, entry := range m.taskHistory {
			if strings.HasPrefix(entry, "[DONE]") {
				b.WriteString(statusStyle.Render(fmt.Sprintf("  %s\n", entry)))
			} else if strings.HasPrefix(entry, "[ERROR]") {
				b.WriteString(errorStyle.Render(fmt.Sprintf("  %s\n", entry)))
			} else {
				b.WriteString(valueStyle.Render(fmt.Sprintf("  %s\n", entry)))
			}
		}
		b.WriteString("\n")
	}

	// Help
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	if m.running {
		if m.parallelRunning {
			b.WriteString(helpStyle.Render("s/esc: Stop parallel execution"))
		} else {
			b.WriteString(helpStyle.Render("s/esc: Stop | p: Pause/Resume"))
		}
	} else {
		b.WriteString(helpStyle.Render("j/k: Navigate | Space/Enter: Select | Start to begin execution"))
	}

	return b.String()
}

func (m *RunModel) renderOption(b *strings.Builder, index int, labelStyle, selectedStyle lipgloss.Style, label, value string) {
	if m.focusIndex == index {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(labelStyle.Render(label))
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

// IsRunning returns whether the run is active
func (m *RunModel) IsRunning() bool {
	return m.running
}

// Stop stops the current run
func (m *RunModel) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.running = false
	m.parallelRunning = false
}
