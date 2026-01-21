package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/circuit"
	"hermes/internal/config"
	"hermes/internal/task"
	"hermes/internal/ui"
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
	ScreenInit
	ScreenRun
	ScreenHelp
)

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
	logger     *ui.Logger

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
	initProj   *InitModel
	run        *RunModel
}

// NewApp creates a new TUI application
func NewApp(basePath string, version string) (*App, error) {
	cfg, err := config.Load(basePath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	logger, _ := ui.NewLogger(basePath, false)

	return &App{
		screen:     ScreenDashboard,
		basePath:   basePath,
		config:     cfg,
		taskReader: task.NewReader(basePath),
		breaker:    circuit.New(basePath),
		logger:     logger,
		dashboard:  NewDashboardModel(basePath),
		tasks:      NewTasksModel(basePath),
		taskDetail: NewTaskDetailModel(basePath),
		logs:       NewLogsModel(basePath),
		idea:       NewIdeaModel(basePath, logger),
		prd:        NewPrdModel(basePath, logger),
		addFeature: NewAddFeatureModel(basePath, logger),
		settings:   NewSettingsModel(basePath),
		circuit:    NewCircuitBreakerModel(basePath),
		update:     NewUpdateModel(version),
		initProj:   NewInitModel(),
		run:        NewRunModel(basePath, logger),
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
		// Update Run progress even when not on Run screen
		if a.run.IsRunning() {
			a.run.DrainProgress()
		}
		return a, tickCmd() // Schedule next tick

	case ConfigSavedMsg:
		// Reload config after settings save
		cfg, err := config.Load(a.basePath)
		if err == nil {
			a.config = cfg
		}
		return a, nil

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
		a.initProj.SetSize(msg.Width, msg.Height-4)
		a.run.SetSize(msg.Width, msg.Height-4)

	case tea.KeyMsg:
		// Check if text input is focused on current screen
		textInputFocused := false
		switch a.screen {
		case ScreenIdea:
			textInputFocused = a.idea.focusIndex == 0
		case ScreenPrd:
			textInputFocused = a.prd.focusIndex == 0
		case ScreenAddFeature:
			textInputFocused = a.addFeature.focusIndex == 0
		case ScreenInit:
			textInputFocused = a.initProj.focusIndex == 0
		}

		// If text input is focused, let the submodel handle all keys except quit
		if textInputFocused {
			if msg.String() == "ctrl+c" {
				return a, tea.Quit
			}
			// Don't handle other keys here, let them go to submodel
		} else {
			switch msg.String() {
			case "q", "ctrl+c":
				return a, tea.Quit
			case "1":
				a.screen = ScreenDashboard
				return a, nil
			case "2":
				a.screen = ScreenTasks
				return a, nil
			case "3":
				a.screen = ScreenLogs
				return a, nil
			case "4":
				a.screen = ScreenIdea
				return a, nil
			case "5":
				a.screen = ScreenPrd
				return a, nil
			case "6":
				a.screen = ScreenAddFeature
				return a, nil
			case "7":
				a.screen = ScreenSettings
				return a, nil
			case "8":
				a.screen = ScreenCircuit
				return a, nil
			case "9":
				a.screen = ScreenUpdate
				return a, nil
			case "0":
				a.screen = ScreenInit
				return a, nil
			case "?":
				a.screen = ScreenHelp
				return a, nil
			}
		}

		// Handle common keys for all screens
		switch msg.String() {
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
			// Go to Run screen
			a.screen = ScreenRun
			a.run.Refresh()
			return a, nil
		case "s":
			// Stop run (if running)
			if a.run.IsRunning() {
				a.run.Stop()
			}
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
	case ScreenInit:
		var model tea.Model
		model, cmd = a.initProj.Update(msg)
		a.initProj = model.(*InitModel)
	case ScreenRun:
		var model tea.Model
		model, cmd = a.run.Update(msg)
		a.run = model.(*RunModel)
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
	case ScreenInit:
		content = a.initProj.View()
	case ScreenRun:
		content = a.run.View()
	case ScreenHelp:
		content = a.helpView()
	}

	// Wrap content in a container with proper sizing
	contentHeight := a.height - 4 // Account for header and footer
	if contentHeight < 10 {
		contentHeight = 10
	}
	contentStyle := lipgloss.NewStyle().
		Width(a.width).
		Height(contentHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		a.headerView(),
		contentStyle.Render(content),
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

	help := "[1]Dash [2]Tasks [3]Logs [4]Idea [5]PRD [6]Add [7]Set [8]CB [9]Upd [0]Init [r]Run [?]Help [q]"
	if a.run.IsRunning() {
		help = "[RUNNING] " + a.run.status + " | [s]Stop [q]Quit"
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
  0           Initialize project screen
  r           Run tasks screen
  ?           This help screen
  Esc         Back to previous screen

Actions:
  s           Stop execution (when running)
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

Init:
  Tab         Navigate between fields
  Space/Enter Initialize project

Run:
  Space/Enter Start/Stop run
  p           Pause/Resume
  s/Esc       Stop execution

Press any key to return...
`
	return style.Render(help)
}
