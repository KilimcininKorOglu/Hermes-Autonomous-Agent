package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/config"
)

// ConfigSavedMsg is sent when configuration is saved
type ConfigSavedMsg struct{}

// SettingsModel is the model for the settings screen
type SettingsModel struct {
	width      int
	height     int
	basePath   string
	config     *config.Config
	focusIndex int
	scrollPos  int
	saved      bool
	err        error
}

const maxFocusIndex = 28 // Total number of settings + save button

// NewSettingsModel creates a new settings model
func NewSettingsModel(basePath string) *SettingsModel {
	cfg, err := config.Load(basePath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	return &SettingsModel{
		basePath: basePath,
		config:   cfg,
	}
}

// Init initializes the model
func (m *SettingsModel) Init() tea.Cmd {
	return nil
}

// SetSize sets the size of the model
func (m *SettingsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Refresh reloads the configuration
func (m *SettingsModel) Refresh() {
	cfg, err := config.Load(m.basePath)
	if err != nil {
		cfg = config.DefaultConfig()
	}
	m.config = cfg
}

// Update handles messages
func (m *SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.focusIndex++
			if m.focusIndex > maxFocusIndex {
				m.focusIndex = 0
				m.scrollPos = 0
			}
			m.updateScroll()
		case "k", "up":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = maxFocusIndex
			}
			m.updateScroll()
		case " ", "enter":
			m.saved = false
			m.err = nil
			m.handleSelect()
		}
	}

	return m, nil
}

func (m *SettingsModel) updateScroll() {
	visibleLines := m.height - 10
	if visibleLines < 10 {
		visibleLines = 10
	}
	if m.focusIndex > m.scrollPos+visibleLines-3 {
		m.scrollPos = m.focusIndex - visibleLines + 3
	}
	if m.focusIndex < m.scrollPos+2 {
		m.scrollPos = m.focusIndex - 2
	}
	if m.scrollPos < 0 {
		m.scrollPos = 0
	}
}

func (m *SettingsModel) handleSelect() tea.Cmd {
	providers := []string{"claude", "droid", "opencode", "gemini"}
	strategies := []string{"continue", "fail-fast"}
	parallelStrategies := []string{"branch-per-task", "worktree"}
	conflictStrategies := []string{"ai-assisted", "manual", "auto-merge"}
	mergeStrategies := []string{"sequential", "parallel"}

	switch m.focusIndex {
	// AI Configuration
	case 0: // Planning Provider
		m.cycleStringOption(&m.config.AI.Planning, providers)
	case 1: // Coding Provider
		m.cycleStringOption(&m.config.AI.Coding, providers)
	case 2: // Stream Output
		m.config.AI.StreamOutput = !m.config.AI.StreamOutput
	case 3: // Timeout
		m.cycleIntOption(&m.config.AI.Timeout, []int{120, 300, 600, 900, 1200})
	case 4: // PRD Timeout
		m.cycleIntOption(&m.config.AI.PrdTimeout, []int{600, 900, 1200, 1800, 2400})
	case 5: // Max Retries
		m.config.AI.MaxRetries++
		if m.config.AI.MaxRetries > 15 {
			m.config.AI.MaxRetries = 1
		}
	case 6: // Retry Delay
		m.cycleIntOption(&m.config.AI.RetryDelay, []int{3, 5, 10, 15, 30})

	// Task Mode
	case 7: // Auto Branch
		m.config.TaskMode.AutoBranch = !m.config.TaskMode.AutoBranch
	case 8: // Auto Commit
		m.config.TaskMode.AutoCommit = !m.config.TaskMode.AutoCommit
	case 9: // Autonomous
		m.config.TaskMode.Autonomous = !m.config.TaskMode.Autonomous
	case 10: // Max Consecutive Errors
		m.config.TaskMode.MaxConsecutiveErrors++
		if m.config.TaskMode.MaxConsecutiveErrors > 10 {
			m.config.TaskMode.MaxConsecutiveErrors = 1
		}

	// Loop Configuration
	case 11: // Max Calls Per Hour
		m.cycleIntOption(&m.config.Loop.MaxCallsPerHour, []int{50, 100, 200, 500, 1000})
	case 12: // Timeout Minutes
		m.cycleIntOption(&m.config.Loop.TimeoutMinutes, []int{5, 10, 15, 30, 60})
	case 13: // Error Delay
		m.cycleIntOption(&m.config.Loop.ErrorDelay, []int{5, 10, 30, 60})

	// Paths Configuration
	case 14: // Hermes Dir
		m.cycleStringOption(&m.config.Paths.HermesDir, []string{".hermes", ".ai", ".agent"})
	case 15: // Tasks Dir
		m.cycleStringOption(&m.config.Paths.TasksDir, []string{".hermes/tasks", ".ai/tasks", "tasks"})
	case 16: // Logs Dir
		m.cycleStringOption(&m.config.Paths.LogsDir, []string{".hermes/logs", ".ai/logs", "logs"})
	case 17: // Docs Dir
		m.cycleStringOption(&m.config.Paths.DocsDir, []string{".hermes/docs", ".ai/docs", "docs"})

	// Parallel Configuration
	case 18: // Enabled
		m.config.Parallel.Enabled = !m.config.Parallel.Enabled
	case 19: // Max Workers
		m.config.Parallel.MaxWorkers++
		if m.config.Parallel.MaxWorkers > 10 {
			m.config.Parallel.MaxWorkers = 1
		}
	case 20: // Strategy
		m.cycleStringOption(&m.config.Parallel.Strategy, parallelStrategies)
	case 21: // Conflict Resolution
		m.cycleStringOption(&m.config.Parallel.ConflictResolution, conflictStrategies)
	case 22: // Isolated Workspaces
		m.config.Parallel.IsolatedWorkspaces = !m.config.Parallel.IsolatedWorkspaces
	case 23: // Merge Strategy
		m.cycleStringOption(&m.config.Parallel.MergeStrategy, mergeStrategies)
	case 24: // Max Cost Per Hour
		costs := []float64{0, 1, 5, 10, 25, 50, 100}
		for i, c := range costs {
			if c == m.config.Parallel.MaxCostPerHour {
				m.config.Parallel.MaxCostPerHour = costs[(i+1)%len(costs)]
				return nil
			}
		}
		m.config.Parallel.MaxCostPerHour = 0
	case 25: // Failure Strategy
		m.cycleStringOption(&m.config.Parallel.FailureStrategy, strategies)
	case 26: // Parallel Max Retries
		m.config.Parallel.MaxRetries++
		if m.config.Parallel.MaxRetries > 5 {
			m.config.Parallel.MaxRetries = 0
		}

	// Save Button
	case 27, 28:
		err := m.saveConfig()
		if err != nil {
			m.err = err
		} else {
			m.saved = true
			return func() tea.Msg { return ConfigSavedMsg{} }
		}
	}
	return nil
}

func (m *SettingsModel) cycleStringOption(current *string, options []string) {
	for i, opt := range options {
		if opt == *current {
			*current = options[(i+1)%len(options)]
			return
		}
	}
	*current = options[0]
}

func (m *SettingsModel) cycleIntOption(current *int, options []int) {
	for i, opt := range options {
		if opt == *current {
			*current = options[(i+1)%len(options)]
			return
		}
	}
	*current = options[0]
}

// View renders the model
func (m *SettingsModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(28)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		MarginTop(1)

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("62")).
		Padding(0, 2)

	b.WriteString(titleStyle.Render("SETTINGS"))
	b.WriteString("\n\n")

	// AI Configuration
	b.WriteString(sectionStyle.Render("AI Configuration"))
	b.WriteString("\n")
	m.renderOption(&b, 0, labelStyle, selectedStyle, valueStyle, "Planning Provider:", m.config.AI.Planning)
	m.renderOption(&b, 1, labelStyle, selectedStyle, valueStyle, "Coding Provider:", m.config.AI.Coding)
	m.renderBoolOption(&b, 2, labelStyle, selectedStyle, "Stream Output:", m.config.AI.StreamOutput)
	m.renderOption(&b, 3, labelStyle, selectedStyle, valueStyle, "Timeout:", fmt.Sprintf("%ds", m.config.AI.Timeout))
	m.renderOption(&b, 4, labelStyle, selectedStyle, valueStyle, "PRD Timeout:", fmt.Sprintf("%ds", m.config.AI.PrdTimeout))
	m.renderOption(&b, 5, labelStyle, selectedStyle, valueStyle, "Max Retries:", fmt.Sprintf("%d", m.config.AI.MaxRetries))
	m.renderOption(&b, 6, labelStyle, selectedStyle, valueStyle, "Retry Delay:", fmt.Sprintf("%ds", m.config.AI.RetryDelay))

	// Task Mode
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Task Mode"))
	b.WriteString("\n")
	m.renderBoolOption(&b, 7, labelStyle, selectedStyle, "Auto Branch:", m.config.TaskMode.AutoBranch)
	m.renderBoolOption(&b, 8, labelStyle, selectedStyle, "Auto Commit:", m.config.TaskMode.AutoCommit)
	m.renderBoolOption(&b, 9, labelStyle, selectedStyle, "Autonomous:", m.config.TaskMode.Autonomous)
	m.renderOption(&b, 10, labelStyle, selectedStyle, valueStyle, "Max Consecutive Errors:", fmt.Sprintf("%d", m.config.TaskMode.MaxConsecutiveErrors))

	// Loop Configuration
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Loop Configuration"))
	b.WriteString("\n")
	m.renderOption(&b, 11, labelStyle, selectedStyle, valueStyle, "Max Calls Per Hour:", fmt.Sprintf("%d", m.config.Loop.MaxCallsPerHour))
	m.renderOption(&b, 12, labelStyle, selectedStyle, valueStyle, "Timeout Minutes:", fmt.Sprintf("%d", m.config.Loop.TimeoutMinutes))
	m.renderOption(&b, 13, labelStyle, selectedStyle, valueStyle, "Error Delay:", fmt.Sprintf("%ds", m.config.Loop.ErrorDelay))

	// Paths Configuration
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Paths"))
	b.WriteString("\n")
	m.renderOption(&b, 14, labelStyle, selectedStyle, valueStyle, "Hermes Dir:", m.config.Paths.HermesDir)
	m.renderOption(&b, 15, labelStyle, selectedStyle, valueStyle, "Tasks Dir:", m.config.Paths.TasksDir)
	m.renderOption(&b, 16, labelStyle, selectedStyle, valueStyle, "Logs Dir:", m.config.Paths.LogsDir)
	m.renderOption(&b, 17, labelStyle, selectedStyle, valueStyle, "Docs Dir:", m.config.Paths.DocsDir)

	// Parallel Execution
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Parallel Execution"))
	b.WriteString("\n")
	m.renderBoolOption(&b, 18, labelStyle, selectedStyle, "Enabled:", m.config.Parallel.Enabled)
	m.renderOption(&b, 19, labelStyle, selectedStyle, valueStyle, "Max Workers:", fmt.Sprintf("%d", m.config.Parallel.MaxWorkers))
	m.renderOption(&b, 20, labelStyle, selectedStyle, valueStyle, "Strategy:", m.config.Parallel.Strategy)
	m.renderOption(&b, 21, labelStyle, selectedStyle, valueStyle, "Conflict Resolution:", m.config.Parallel.ConflictResolution)
	m.renderBoolOption(&b, 22, labelStyle, selectedStyle, "Isolated Workspaces:", m.config.Parallel.IsolatedWorkspaces)
	m.renderOption(&b, 23, labelStyle, selectedStyle, valueStyle, "Merge Strategy:", m.config.Parallel.MergeStrategy)
	costStr := "No Limit"
	if m.config.Parallel.MaxCostPerHour > 0 {
		costStr = fmt.Sprintf("$%.0f", m.config.Parallel.MaxCostPerHour)
	}
	m.renderOption(&b, 24, labelStyle, selectedStyle, valueStyle, "Max Cost Per Hour:", costStr)
	m.renderOption(&b, 25, labelStyle, selectedStyle, valueStyle, "Failure Strategy:", m.config.Parallel.FailureStrategy)
	m.renderOption(&b, 26, labelStyle, selectedStyle, valueStyle, "Max Retries:", fmt.Sprintf("%d", m.config.Parallel.MaxRetries))

	// Save Button
	b.WriteString("\n\n")
	if m.focusIndex == 27 || m.focusIndex == 28 {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(buttonStyle.Render("Save Configuration"))
	b.WriteString("\n\n")

	if m.saved {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		b.WriteString(successStyle.Render("Configuration saved successfully!"))
		b.WriteString("\n")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	b.WriteString(helpStyle.Render("j/k: Navigate | Space/Enter: Toggle/Select | Scroll for more"))

	return b.String()
}

func (m *SettingsModel) renderOption(b *strings.Builder, index int, labelStyle, selectedStyle, valueStyle lipgloss.Style, label, value string) {
	if m.focusIndex == index {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(labelStyle.Render(label))
	b.WriteString(valueStyle.Render(value))
	b.WriteString("\n")
}

func (m *SettingsModel) renderBoolOption(b *strings.Builder, index int, labelStyle, selectedStyle lipgloss.Style, label string, value bool) {
	if m.focusIndex == index {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(labelStyle.Render(label))
	if value {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("[x] Enabled"))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("[ ] Disabled"))
	}
	b.WriteString("\n")
}

func (m *SettingsModel) saveConfig() error {
	configPath := filepath.Join(m.basePath, ".hermes", "config.json")

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
