package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/ai"
	"hermes/internal/config"
	"hermes/internal/idea"
	"hermes/internal/ui"
)

// IdeaModel is the model for the Idea/PRD generation screen
type IdeaModel struct {
	width       int
	height      int
	basePath    string
	textInput   textinput.Model
	language    string
	interactive bool
	generating  bool
	result      string
	err         error
	focusIndex  int
}

// ideaResultMsg is sent when PRD generation completes
type ideaResultMsg struct {
	result *idea.GenerateResult
	err    error
}

// NewIdeaModel creates a new idea model
func NewIdeaModel(basePath string) *IdeaModel {
	ti := textinput.New()
	ti.Placeholder = "Enter your idea description..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60

	return &IdeaModel{
		basePath:   basePath,
		textInput:  ti,
		language:   "en",
		focusIndex: 0,
	}
}

// Init initializes the model
func (m *IdeaModel) Init() tea.Cmd {
	return textinput.Blink
}

// SetSize sets the size of the model
func (m *IdeaModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textInput.Width = width - 10
}

// Update handles messages
func (m *IdeaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.generating {
			return m, nil
		}

		switch msg.String() {
		case "tab", "shift+tab":
			m.focusIndex = (m.focusIndex + 1) % 4
			if m.focusIndex == 0 {
				m.textInput.Focus()
			} else {
				m.textInput.Blur()
			}
		case "l":
			if m.focusIndex == 1 {
				if m.language == "en" {
					m.language = "tr"
				} else {
					m.language = "en"
				}
			}
		case " ", "enter":
			switch m.focusIndex {
			case 1:
				if m.language == "en" {
					m.language = "tr"
				} else {
					m.language = "en"
				}
			case 2:
				m.interactive = !m.interactive
			case 3:
				if m.textInput.Value() != "" {
					m.generating = true
					m.result = ""
					m.err = nil
					return m, m.generatePRD()
				}
			}
		}

	case ideaResultMsg:
		m.generating = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.result = fmt.Sprintf("PRD generated: %s", msg.result.FilePath)
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
func (m *IdeaModel) View() string {
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

	b.WriteString(titleStyle.Render("IDEA TO PRD GENERATOR"))
	b.WriteString("\n\n")

	if m.generating {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Generating PRD..."))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(labelStyle.Render("Idea Description:"))
	b.WriteString("\n")
	if m.focusIndex == 0 {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Language:"))
	b.WriteString(" ")
	if m.focusIndex == 1 {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	if m.language == "en" {
		b.WriteString("[x] English  [ ] Turkish")
	} else {
		b.WriteString("[ ] English  [x] Turkish")
	}
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Interactive Mode:"))
	b.WriteString(" ")
	if m.focusIndex == 2 {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	if m.interactive {
		b.WriteString("[x] Enabled (asks additional questions)")
	} else {
		b.WriteString("[ ] Disabled")
	}
	b.WriteString("\n\n")

	if m.focusIndex == 3 {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	if m.textInput.Value() != "" {
		b.WriteString(buttonStyle.Render("Generate PRD"))
	} else {
		b.WriteString(disabledButtonStyle.Render("Generate PRD"))
	}
	b.WriteString("\n\n")

	if m.result != "" {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		b.WriteString(successStyle.Render(m.result))
		b.WriteString("\n")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	b.WriteString(helpStyle.Render("Tab: Navigate | Space/Enter: Select | l: Toggle language"))

	return b.String()
}

// Reset clears the model for new input
func (m *IdeaModel) Reset() {
	m.textInput.SetValue("")
	m.language = "en"
	m.interactive = false
	m.generating = false
	m.result = ""
	m.err = nil
	m.focusIndex = 0
	m.textInput.Focus()
}

// generatePRD generates the PRD using AI
func (m *IdeaModel) generatePRD() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		cfg, err := config.Load(m.basePath)
		if err != nil {
			cfg = config.DefaultConfig()
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
			return ideaResultMsg{err: fmt.Errorf("no AI provider available")}
		}

		logger, _ := ui.NewLogger(m.basePath, false)
		generator := idea.NewGenerator(provider, cfg, logger)

		opts := idea.GenerateOptions{
			Idea:        m.textInput.Value(),
			Output:      ".hermes/docs/PRD.md",
			DryRun:      false,
			Interactive: m.interactive,
			Language:    m.language,
			Timeout:     cfg.AI.PrdTimeout,
		}

		result, err := generator.Generate(ctx, opts)
		if err != nil {
			return ideaResultMsg{err: err}
		}

		return ideaResultMsg{result: result}
	}
}
