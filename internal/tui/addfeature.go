package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/ai"
	"hermes/internal/analyzer"
	"hermes/internal/config"
	"hermes/internal/ui"
)

// AddFeatureModel is the model for the add feature screen
type AddFeatureModel struct {
	width      int
	height     int
	basePath   string
	textInput  textinput.Model
	dryRun     bool
	adding     bool
	result     string
	filePath   string
	err        error
	focusIndex int
	logger     *ui.Logger
}

// addFeatureResultMsg is sent when feature addition completes
type addFeatureResultMsg struct {
	filePath string
	err      error
}

// NewAddFeatureModel creates a new add feature model
func NewAddFeatureModel(basePath string, logger *ui.Logger) *AddFeatureModel {
	ti := textinput.New()
	ti.Placeholder = "Enter feature description..."
	ti.Focus()
	ti.CharLimit = 300
	ti.Width = 60

	return &AddFeatureModel{
		basePath:   basePath,
		textInput:  ti,
		focusIndex: 0,
		logger:     logger,
	}
}

// Init initializes the model
func (m *AddFeatureModel) Init() tea.Cmd {
	return textinput.Blink
}

// SetSize sets the size of the model
func (m *AddFeatureModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textInput.Width = width - 10
}

// Update handles messages
func (m *AddFeatureModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.adding {
			return m, nil
		}

		switch msg.String() {
		case "tab", "shift+tab":
			m.focusIndex = (m.focusIndex + 1) % 3
			if m.focusIndex == 0 {
				m.textInput.Focus()
			} else {
				m.textInput.Blur()
			}
		case " ", "enter":
			switch m.focusIndex {
			case 1:
				m.dryRun = !m.dryRun
			case 2:
				if m.textInput.Value() != "" {
					m.adding = true
					m.result = ""
					m.filePath = ""
					m.err = nil
					return m, m.addFeature()
				}
			}
		}

	case addFeatureResultMsg:
		m.adding = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.filePath = msg.filePath
			if m.dryRun {
				m.result = "Dry run completed - no files written"
			} else {
				m.result = "Feature added successfully"
			}
		}
		return m, nil
	}

	if m.focusIndex == 0 {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the model
func (m *AddFeatureModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("62")).
		Padding(0, 2)

	disabledButtonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("236")).
		Padding(0, 2)

	b.WriteString(titleStyle.Render("ADD FEATURE"))
	b.WriteString("\n\n")

	if m.adding {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Adding feature..."))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(labelStyle.Render("Feature Description:"))
	b.WriteString("\n")
	if m.focusIndex == 0 {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Options:"))
	b.WriteString(" ")
	if m.focusIndex == 1 {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	if m.dryRun {
		b.WriteString("[x] Dry Run (preview without writing)")
	} else {
		b.WriteString("[ ] Dry Run (preview without writing)")
	}
	b.WriteString("\n\n")

	if m.focusIndex == 2 {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	if m.textInput.Value() != "" {
		b.WriteString(buttonStyle.Render("Add Feature"))
	} else {
		b.WriteString(disabledButtonStyle.Render("Add Feature"))
	}
	b.WriteString("\n\n")

	if m.result != "" {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		b.WriteString(successStyle.Render(m.result))
		b.WriteString("\n")
		
		if m.filePath != "" {
			fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
			b.WriteString(fileStyle.Render("  Created: " + m.filePath))
			b.WriteString("\n")
		}
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	b.WriteString(helpStyle.Render("Tab: Navigate | Space/Enter: Select"))

	return b.String()
}

// Reset clears the model
func (m *AddFeatureModel) Reset() {
	m.textInput.SetValue("")
	m.dryRun = false
	m.adding = false
	m.result = ""
	m.filePath = ""
	m.err = nil
	m.focusIndex = 0
	m.textInput.Focus()
}

// addFeature adds the feature using AI
func (m *AddFeatureModel) addFeature() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		cfg, err := config.Load(m.basePath)
		if err != nil {
			cfg = config.DefaultConfig()
		}

		featureAnalyzer := analyzer.NewFeatureAnalyzer(m.basePath)
		nextFeatureID, nextTaskID, err := featureAnalyzer.GetNextIDs()
		if err != nil {
			nextFeatureID = 1
			nextTaskID = 1
		}

		// Get provider from config
		var provider ai.Provider
		if cfg.AI.Planning != "" && cfg.AI.Planning != "auto" {
			provider = ai.GetProvider(cfg.AI.Planning)
		}
		if provider == nil || !provider.IsAvailable() {
			provider = ai.AutoDetectProvider()
		}
		if provider == nil {
			return addFeatureResultMsg{err: fmt.Errorf("no AI provider available")}
		}

		if m.logger != nil {
			m.logger.Info("Adding feature: %s", m.textInput.Value())
		}

		prompt := buildAddPromptForTUI(m.textInput.Value(), nextFeatureID, nextTaskID)

		result, err := ai.ExecuteWithRetry(ctx, provider, &ai.ExecuteOptions{
			Prompt:       prompt,
			Timeout:      cfg.AI.Timeout,
			StreamOutput: false,
		}, &ai.RetryConfig{
			MaxRetries: cfg.AI.MaxRetries,
			Delay:      time.Duration(cfg.AI.RetryDelay) * time.Second,
		})

		if err != nil {
			return addFeatureResultMsg{err: fmt.Errorf("AI execution failed: %w", err)}
		}

		if m.dryRun {
			return addFeatureResultMsg{filePath: "(dry run)"}
		}

		filePath, err := writeFeatureFileForTUI(m.basePath, result.Output, nextFeatureID, m.textInput.Value())
		if err != nil {
			return addFeatureResultMsg{err: err}
		}

		return addFeatureResultMsg{filePath: filePath}
	}
}

func buildAddPromptForTUI(desc string, featureID, taskID int) string {
	return fmt.Sprintf(`Create a feature file for: %s

Use Feature ID: F%03d
Start Task IDs from: T%03d

Create with this format:

# Feature %d: <Feature Name>

**Feature ID:** F%03d
**Priority:** P2 - HIGH
**Status:** NOT_STARTED

## Tasks

### T%03d: <Task Name>
**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description
[Task description]

#### Files to Touch
- path/file.go (new)

#### Dependencies
- None

#### Success Criteria
- [ ] Criterion 1

---

Create 3-5 tasks. Output only markdown, no explanation.`, desc, featureID, taskID, featureID, featureID, taskID)
}

func writeFeatureFileForTUI(basePath, output string, featureID int, desc string) (string, error) {
	tasksDir := filepath.Join(basePath, ".hermes", "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return "", err
	}

	safeName := strings.ToLower(desc)
	safeName = strings.ReplaceAll(safeName, " ", "-")
	var result strings.Builder
	for _, r := range safeName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	safeName = result.String()
	if len(safeName) > 30 {
		safeName = safeName[:30]
	}

	fileName := fmt.Sprintf("%03d-%s.md", featureID, safeName)
	filePath := filepath.Join(tasksDir, fileName)

	if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}
