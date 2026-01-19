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

// SettingsModel is the model for the settings screen
type SettingsModel struct {
	width      int
	height     int
	basePath   string
	config     *config.Config
	focusIndex int
	saved      bool
	err        error
}

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
			if m.focusIndex > 10 {
				m.focusIndex = 0
			}
		case "k", "up":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 10
			}
		case " ", "enter":
			m.saved = false
			m.err = nil
			switch m.focusIndex {
			case 0: // AI Provider toggle
				providers := []string{"claude", "droid", "opencode", "gemini", "auto"}
				current := m.config.AI.Coding
				for i, p := range providers {
					if p == current {
						m.config.AI.Coding = providers[(i+1)%len(providers)]
						m.config.AI.Planning = m.config.AI.Coding
						break
					}
				}
				if m.config.AI.Coding == current {
					m.config.AI.Coding = "claude"
					m.config.AI.Planning = "claude"
				}
			case 1: // Auto Branch
				m.config.TaskMode.AutoBranch = !m.config.TaskMode.AutoBranch
			case 2: // Auto Commit
				m.config.TaskMode.AutoCommit = !m.config.TaskMode.AutoCommit
			case 3: // Autonomous
				m.config.TaskMode.Autonomous = !m.config.TaskMode.Autonomous
			case 4: // Stream Output
				m.config.AI.StreamOutput = !m.config.AI.StreamOutput
			case 5: // Parallel Enabled
				m.config.Parallel.Enabled = !m.config.Parallel.Enabled
			case 6: // Max Workers (cycle 1-5)
				m.config.Parallel.MaxWorkers++
				if m.config.Parallel.MaxWorkers > 5 {
					m.config.Parallel.MaxWorkers = 1
				}
			case 7: // Timeout (cycle common values)
				timeouts := []int{120, 300, 600, 900, 1200}
				current := m.config.AI.Timeout
				for i, t := range timeouts {
					if t == current {
						m.config.AI.Timeout = timeouts[(i+1)%len(timeouts)]
						break
					}
				}
				if m.config.AI.Timeout == current {
					m.config.AI.Timeout = 300
				}
			case 8: // Max Retries (cycle 1-10)
				m.config.AI.MaxRetries++
				if m.config.AI.MaxRetries > 10 {
					m.config.AI.MaxRetries = 1
				}
			case 9: // Failure Strategy
				if m.config.Parallel.FailureStrategy == "continue" {
					m.config.Parallel.FailureStrategy = "fail-fast"
				} else {
					m.config.Parallel.FailureStrategy = "continue"
				}
			case 10: // Save button
				err := m.saveConfig()
				if err != nil {
					m.err = err
				} else {
					m.saved = true
				}
			}
		}
	}

	return m, nil
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
		Width(25)

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

	b.WriteString(sectionStyle.Render("AI Configuration"))
	b.WriteString("\n")
	m.renderOption(&b, 0, labelStyle, selectedStyle, valueStyle, "AI Provider:", m.config.AI.Coding)
	m.renderBoolOption(&b, 4, labelStyle, selectedStyle, "Stream Output:", m.config.AI.StreamOutput)
	m.renderOption(&b, 7, labelStyle, selectedStyle, valueStyle, "Timeout:", fmt.Sprintf("%ds", m.config.AI.Timeout))
	m.renderOption(&b, 8, labelStyle, selectedStyle, valueStyle, "Max Retries:", fmt.Sprintf("%d", m.config.AI.MaxRetries))

	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Task Mode"))
	b.WriteString("\n")
	m.renderBoolOption(&b, 1, labelStyle, selectedStyle, "Auto Branch:", m.config.TaskMode.AutoBranch)
	m.renderBoolOption(&b, 2, labelStyle, selectedStyle, "Auto Commit:", m.config.TaskMode.AutoCommit)
	m.renderBoolOption(&b, 3, labelStyle, selectedStyle, "Autonomous:", m.config.TaskMode.Autonomous)

	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Parallel Execution"))
	b.WriteString("\n")
	m.renderBoolOption(&b, 5, labelStyle, selectedStyle, "Enabled:", m.config.Parallel.Enabled)
	m.renderOption(&b, 6, labelStyle, selectedStyle, valueStyle, "Max Workers:", fmt.Sprintf("%d", m.config.Parallel.MaxWorkers))
	m.renderOption(&b, 9, labelStyle, selectedStyle, valueStyle, "Failure Strategy:", m.config.Parallel.FailureStrategy)

	b.WriteString("\n\n")
	if m.focusIndex == 10 {
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
	b.WriteString(helpStyle.Render("j/k: Navigate | Space/Enter: Toggle/Select"))

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
