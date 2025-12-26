package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/circuit"
	"hermes/internal/config"
	"hermes/internal/task"
)

// Screen represents the current screen
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenTasks
	ScreenLogs
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

	// Sub-models
	dashboard *DashboardModel
	tasks     *TasksModel
}

// NewApp creates a new TUI application
func NewApp(basePath string) (*App, error) {
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
	}, nil
}

// Init initializes the TUI
func (a App) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		a.dashboard.Init(),
	)
}

// Update handles messages
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		a.dashboard.SetSize(msg.Width, msg.Height-4)
		a.tasks.SetSize(msg.Width, msg.Height-4)

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
		case "?":
			a.screen = ScreenHelp
		case "r":
			// Refresh data
			a.dashboard.Refresh()
			a.tasks.Refresh()
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
	case ScreenLogs:
		content = a.logsView()
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

	help := "[1]Dashboard [2]Tasks [3]Logs [?]Help [r]Refresh [q]Quit"
	return style.Render(help)
}

func (a App) helpView() string {
	style := lipgloss.NewStyle().
		Padding(1, 2)

	help := `
HERMES TUI HELP

Navigation:
  1         Dashboard screen
  2         Tasks screen
  3         Logs screen
  ?         This help screen

Actions:
  r         Refresh data
  Enter     Select item (in lists)
  j/k       Move up/down (in lists)
  q         Quit

Dashboard:
  Shows progress, circuit breaker status, and current task

Tasks:
  Browse and filter tasks by status

Logs:
  View real-time execution logs

Press any key to return...
`
	return style.Render(help)
}

func (a App) logsView() string {
	style := lipgloss.NewStyle().
		Padding(1, 2)

	return style.Render("Logs viewer - Coming soon\n\nPress 1-4 to switch screens")
}
