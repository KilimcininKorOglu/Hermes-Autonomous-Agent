package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/ai"
	"hermes/internal/analyzer"
	"hermes/internal/circuit"
	"hermes/internal/config"
	"hermes/internal/prompt"
	"hermes/internal/task"
)

// tickMsg is sent on each tick for auto-refresh
type tickMsg time.Time

// Auto-refresh interval
const refreshInterval = 2 * time.Second

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Screen represents the current screen
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenTasks
	ScreenTaskDetail
	ScreenLogs
	ScreenIdea
	ScreenPrd
	ScreenAddFeature
	ScreenSettings
	ScreenCircuit
	ScreenUpdate
	ScreenHelp
)

// runResultMsg is sent when a task execution completes
type runResultMsg struct {
	taskID  string
	success bool
	err     error
}

// App is the main TUI model
type App struct {
	screen     Screen
	width      int
	height     int
	ready      bool
	basePath   string
	config     *config.Config
	taskReader *task.Reader
	breaker    *circuit.Breaker
	running    bool   // Is run loop active?
	runStatus  string
	runCancel  context.CancelFunc
	loopCount  int

	// Sub-models
	dashboard  *DashboardModel
	tasks      *TasksModel
	taskDetail *TaskDetailModel
	logs       *LogsModel
	idea       *IdeaModel
	prd        *PrdModel
	addFeature *AddFeatureModel
	settings   *SettingsModel
	circuit    *CircuitBreakerModel
	update     *UpdateModel
}

// NewApp creates a new TUI application
func NewApp(basePath string, version string) (*App, error) {
	cfg, err := config.Load(basePath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	return &App{
		screen:     ScreenDashboard,
		basePath:   basePath,
		config:     cfg,
		taskReader: task.NewReader(basePath),
		breaker:    circuit.New(basePath),
		dashboard:  NewDashboardModel(basePath),
		tasks:      NewTasksModel(basePath),
		taskDetail: NewTaskDetailModel(basePath),
		logs:       NewLogsModel(basePath),
		idea:       NewIdeaModel(basePath),
		prd:        NewPrdModel(basePath),
		addFeature: NewAddFeatureModel(basePath),
		settings:   NewSettingsModel(basePath),
		circuit:    NewCircuitBreakerModel(basePath),
		update:     NewUpdateModel(version),
	}, nil
}

// Init initializes the TUI
func (a App) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		a.dashboard.Init(),
		tickCmd(), // Start auto-refresh
	)
}

// Update handles messages
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		// Auto-refresh data
		a.dashboard.Refresh()
		a.tasks.Refresh()
		a.logs.Refresh()
		return a, tickCmd() // Schedule next tick

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		a.dashboard.SetSize(msg.Width, msg.Height-4)
		a.tasks.SetSize(msg.Width, msg.Height-4)
		a.taskDetail.SetSize(msg.Width, msg.Height-4)
		a.logs.SetSize(msg.Width, msg.Height-4)
		a.idea.SetSize(msg.Width, msg.Height-4)
		a.prd.SetSize(msg.Width, msg.Height-4)
		a.addFeature.SetSize(msg.Width, msg.Height-4)
		a.settings.SetSize(msg.Width, msg.Height-4)
		a.circuit.SetSize(msg.Width, msg.Height-4)
		a.update.SetSize(msg.Width, msg.Height-4)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "1":
			a.screen = ScreenDashboard
		case "2":
			a.screen = ScreenTasks
		case "3":
			a.screen = ScreenLogs
		case "4":
			a.screen = ScreenIdea
		case "5":
			a.screen = ScreenPrd
		case "6":
			a.screen = ScreenAddFeature
		case "7":
			a.screen = ScreenSettings
		case "8":
			a.screen = ScreenCircuit
		case "9":
			a.screen = ScreenUpdate
		case "?":
			a.screen = ScreenHelp
		case "enter":
			// Open task detail from tasks screen
			if a.screen == ScreenTasks {
				tasks := a.tasks.filteredTasks()
				if len(tasks) > 0 && a.tasks.cursor < len(tasks) {
					a.taskDetail.SetTask(&tasks[a.tasks.cursor])
					a.screen = ScreenTaskDetail
				}
			}
		case "esc":
			// Back from detail screens
			if a.screen == ScreenTaskDetail {
				a.screen = ScreenTasks
			}
		case "R":
			// Manual refresh (Shift+R)
			a.dashboard.Refresh()
			a.tasks.Refresh()
			a.logs.Refresh()
		case "r":
			// Start run
			if !a.running {
				a.running = true
				a.loopCount = 0
				a.runStatus = "Starting..."
				return a, a.startRun()
			}
		case "s":
			// Stop run
			if a.running && a.runCancel != nil {
				a.runCancel()
				a.running = false
				a.runStatus = ""
			}
		}

	case runResultMsg:
		if msg.err != nil {
			a.runStatus = fmt.Sprintf("Error: %v", msg.err)
		} else if msg.success {
			a.runStatus = fmt.Sprintf("Completed: %s", msg.taskID)
		}
		// Continue to next task
		if a.running {
			return a, a.startRun()
		}
	}

	// Update active screen
	var cmd tea.Cmd
	switch a.screen {
	case ScreenDashboard:
		var model tea.Model
		model, cmd = a.dashboard.Update(msg)
		a.dashboard = model.(*DashboardModel)
	case ScreenTasks:
		var model tea.Model
		model, cmd = a.tasks.Update(msg)
		a.tasks = model.(*TasksModel)
	case ScreenTaskDetail:
		var model tea.Model
		model, cmd = a.taskDetail.Update(msg)
		a.taskDetail = model.(*TaskDetailModel)
	case ScreenLogs:
		var model tea.Model
		model, cmd = a.logs.Update(msg)
		a.logs = model.(*LogsModel)
	case ScreenIdea:
		var model tea.Model
		model, cmd = a.idea.Update(msg)
		a.idea = model.(*IdeaModel)
	case ScreenPrd:
		var model tea.Model
		model, cmd = a.prd.Update(msg)
		a.prd = model.(*PrdModel)
	case ScreenAddFeature:
		var model tea.Model
		model, cmd = a.addFeature.Update(msg)
		a.addFeature = model.(*AddFeatureModel)
	case ScreenSettings:
		var model tea.Model
		model, cmd = a.settings.Update(msg)
		a.settings = model.(*SettingsModel)
	case ScreenCircuit:
		var model tea.Model
		model, cmd = a.circuit.Update(msg)
		a.circuit = model.(*CircuitBreakerModel)
	case ScreenUpdate:
		var model tea.Model
		model, cmd = a.update.Update(msg)
		a.update = model.(*UpdateModel)
	}

	return a, cmd
}

// View renders the TUI
func (a App) View() string {
	if !a.ready {
		return "Initializing..."
	}

	var content string
	switch a.screen {
	case ScreenDashboard:
		content = a.dashboard.View()
	case ScreenTasks:
		content = a.tasks.View()
	case ScreenTaskDetail:
		content = a.taskDetail.View()
	case ScreenLogs:
		content = a.logs.View()
	case ScreenIdea:
		content = a.idea.View()
	case ScreenPrd:
		content = a.prd.View()
	case ScreenAddFeature:
		content = a.addFeature.View()
	case ScreenSettings:
		content = a.settings.View()
	case ScreenCircuit:
		content = a.circuit.View()
	case ScreenUpdate:
		content = a.update.View()
	case ScreenHelp:
		content = a.helpView()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		a.headerView(),
		content,
		a.footerView(),
	)
}

func (a App) headerView() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Width(a.width)

	title := "HERMES AUTONOMOUS AGENT"
	return style.Render(title)
}

func (a App) footerView() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(a.width)

	help := "[1]Dash [2]Tasks [3]Logs [4]Idea [5]PRD [6]Add [7]Set [8]CB [9]Upd [?]Help [q]Quit"
	if a.running {
		help = "[RUNNING] " + a.runStatus + " | [s]Stop [q]Quit"
	}
	return style.Render(help)
}

func (a App) helpView() string {
	style := lipgloss.NewStyle().
		Padding(1, 2)

	help := `
HERMES TUI HELP

Navigation:
  1           Dashboard screen
  2           Tasks screen
  3           Logs screen
  4           Idea/PRD generator screen
  5           PRD parser screen
  6           Add feature screen
  7           Settings screen
  8           Circuit breaker screen
  9           Update screen
  ?           This help screen
  Esc         Back to previous screen

Actions:
  r           Start task execution
  s           Stop execution
  Shift+R     Manual refresh
  Enter       Open task detail (from Tasks)
  j/k         Move up/down
  q           Quit

Dashboard:
  Shows progress, circuit breaker status, and current task
  Auto-refreshes every 2 seconds

Tasks:
  a/c/p/n/b   Filter: All/Completed/InProgress/NotStarted/Blocked
  Enter       View task details

Logs:
  g           Go to top
  Shift+G     Go to bottom
  f           Toggle auto-scroll

Idea:
  Tab         Navigate between fields
  Space/Enter Select option or generate
  l           Toggle language (en/tr)

PRD Parser:
  Tab         Navigate between fields
  Space/Enter Select option or parse

Add Feature:
  Tab         Navigate between fields
  Space/Enter Select option or add

Settings:
  j/k         Navigate options
  Space/Enter Toggle value or save

Circuit Breaker:
  r           Refresh state
  Space/Enter Reset breaker (when OPEN/HALF_OPEN)

Update:
  c           Check for updates
  u           Install update (when available)

Press any key to return...
`
	return style.Render(help)
}

// startRun starts executing the next task
func (a *App) startRun() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		a.runCancel = cancel

		// Check circuit breaker
		canExecute, _ := a.breaker.CanExecute()
		if !canExecute {
			a.running = false
			return runResultMsg{err: fmt.Errorf("circuit breaker open")}
		}

		// Get next task
		nextTask, err := a.taskReader.GetNextTask()
		if err != nil {
			return runResultMsg{err: err}
		}
		if nextTask == nil {
			a.running = false
			return runResultMsg{err: fmt.Errorf("all tasks completed")}
		}

		a.loopCount++
		a.runStatus = fmt.Sprintf("Loop #%d: %s", a.loopCount, nextTask.ID)

		// Inject task into prompt
		injector := prompt.NewInjector(a.basePath)
		injector.AddTask(nextTask)
		promptContent, _ := injector.Read()

		// Execute AI
		cfg, _ := config.Load(a.basePath)
		streamOutput := true
		if cfg != nil {
			streamOutput = cfg.AI.StreamOutput
		}
		provider := ai.NewClaudeProvider()
		executor := ai.NewTaskExecutor(provider, a.basePath)
		result, err := executor.ExecuteTask(ctx, nextTask, promptContent, streamOutput)

		if err != nil {
			a.breaker.AddLoopResult(false, true, a.loopCount)
			return runResultMsg{taskID: nextTask.ID, err: err}
		}

		// Analyze response
		respAnalyzer := analyzer.NewResponseAnalyzer()
		analysis := respAnalyzer.Analyze(result.Output)

		// Update circuit breaker
		a.breaker.AddLoopResult(analysis.HasProgress, false, a.loopCount)

		// Update task status if complete
		if analysis.IsComplete {
			statusUpdater := task.NewStatusUpdater(a.basePath)
			statusUpdater.UpdateTaskStatus(nextTask.ID, task.StatusCompleted)
			injector.RemoveTask()
			return runResultMsg{taskID: nextTask.ID, success: true}
		}

		return runResultMsg{taskID: nextTask.ID, success: false}
	}
}
