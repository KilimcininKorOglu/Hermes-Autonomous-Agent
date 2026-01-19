package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/ai"
	"hermes/internal/config"
)

// PrdModel is the model for the PRD parser screen
type PrdModel struct {
	width      int
	height     int
	basePath   string
	textInput  textinput.Model
	dryRun     bool
	parsing    bool
	result     string
	filesCreated []string
	err        error
	focusIndex int
}

// prdResultMsg is sent when PRD parsing completes
type prdResultMsg struct {
	files []string
	err   error
}

// NewPrdModel creates a new PRD model
func NewPrdModel(basePath string) *PrdModel {
	ti := textinput.New()
	ti.Placeholder = ".hermes/docs/PRD.md"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50

	return &PrdModel{
		basePath:  basePath,
		textInput: ti,
		focusIndex: 0,
	}
}

// Init initializes the model
func (m *PrdModel) Init() tea.Cmd {
	return textinput.Blink
}

// SetSize sets the size of the model
func (m *PrdModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textInput.Width = width - 10
}

// Update handles messages
func (m *PrdModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.parsing {
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
				prdPath := m.textInput.Value()
				if prdPath == "" {
					prdPath = ".hermes/docs/PRD.md"
				}
				if _, err := os.Stat(prdPath); err == nil {
					m.parsing = true
					m.result = ""
					m.filesCreated = nil
					m.err = nil
					return m, m.parsePRD(prdPath)
				} else {
					m.err = fmt.Errorf("file not found: %s", prdPath)
				}
			}
		}

	case prdResultMsg:
		m.parsing = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.filesCreated = msg.files
			if m.dryRun {
				m.result = "Dry run completed - no files written"
			} else {
				m.result = fmt.Sprintf("Created %d task files", len(msg.files))
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
func (m *PrdModel) View() string {
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

	b.WriteString(titleStyle.Render("PRD PARSER"))
	b.WriteString("\n\n")

	if m.parsing {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Parsing PRD..."))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(labelStyle.Render("PRD File Path:"))
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
	b.WriteString(buttonStyle.Render("Parse PRD"))
	b.WriteString("\n\n")

	if m.result != "" {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		b.WriteString(successStyle.Render(m.result))
		b.WriteString("\n")
		
		if len(m.filesCreated) > 0 {
			fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
			for _, f := range m.filesCreated {
				b.WriteString(fileStyle.Render("  - " + f))
				b.WriteString("\n")
			}
		}
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	b.WriteString(helpStyle.Render("Tab: Navigate | Space/Enter: Select | Default: .hermes/docs/PRD.md"))

	return b.String()
}

// Reset clears the model
func (m *PrdModel) Reset() {
	m.textInput.SetValue("")
	m.dryRun = false
	m.parsing = false
	m.result = ""
	m.filesCreated = nil
	m.err = nil
	m.focusIndex = 0
	m.textInput.Focus()
}

// parsePRD parses the PRD file
func (m *PrdModel) parsePRD(prdPath string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		cfg, err := config.Load(m.basePath)
		if err != nil {
			cfg = config.DefaultConfig()
		}

		prdContent, err := os.ReadFile(prdPath)
		if err != nil {
			return prdResultMsg{err: fmt.Errorf("failed to read PRD: %w", err)}
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
			return prdResultMsg{err: fmt.Errorf("no AI provider available")}
		}

		prompt := buildPrdPromptForTUI(string(prdContent))

		result, err := ai.ExecuteWithRetry(ctx, provider, &ai.ExecuteOptions{
			Prompt:       prompt,
			Timeout:      cfg.AI.PrdTimeout,
			StreamOutput: false,
		}, &ai.RetryConfig{
			MaxRetries: 3,
			Delay:      5 * time.Second,
		})

		if err != nil {
			return prdResultMsg{err: fmt.Errorf("AI execution failed: %w", err)}
		}

		if m.dryRun {
			return prdResultMsg{files: []string{"(dry run)"}}
		}

		files, err := writeTaskFilesForTUI(m.basePath, result.Output)
		if err != nil {
			return prdResultMsg{err: err}
		}

		return prdResultMsg{files: files}
	}
}

func buildPrdPromptForTUI(prdContent string) string {
	return fmt.Sprintf(`Parse this PRD into task files. Create files in .hermes/tasks/ directory.

Use this format for each feature file:

# Feature N: Feature Name
**Feature ID:** FXXX
**Priority:** P1 - CRITICAL
**Status:** NOT_STARTED

## Tasks

### TXXX: Task Name
**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** X days

#### Description
[Task description]

#### Files to Touch
- path/file.go (new)

#### Dependencies
- None

#### Success Criteria
- [ ] Criterion 1

---

Output format:
---FILE: .hermes/tasks/001-feature-name.md---
[content]
---END_FILE---

PRD Content:

%s`, prdContent)
}

func writeTaskFilesForTUI(basePath, output string) ([]string, error) {
	tasksDir := filepath.Join(basePath, ".hermes", "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return nil, err
	}

	fileRegex := regexp.MustCompile(`---FILE:\s*(.+?)---\s*([\s\S]*?)---END_FILE---`)
	matches := fileRegex.FindAllStringSubmatch(output, -1)

	if len(matches) == 0 {
		matches = parseConsecutiveFileMarkersForTUI(output)
	}

	var files []string

	if len(matches) == 0 {
		filePath := filepath.Join(tasksDir, "001-tasks.md")
		if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
			return nil, err
		}
		return []string{filePath}, nil
	}

	for _, match := range matches {
		fileName := strings.TrimSpace(match[1])
		content := strings.TrimSpace(match[2])

		fileName = filepath.Base(fileName)
		if !strings.HasSuffix(fileName, ".md") {
			fileName = fileName + ".md"
		}

		filePath := filepath.Join(tasksDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return nil, err
		}
		files = append(files, filePath)
	}

	return files, nil
}

func parseConsecutiveFileMarkersForTUI(output string) [][]string {
	markerRegex := regexp.MustCompile(`---FILE:\s*(.+?)---`)
	allMatches := markerRegex.FindAllStringSubmatchIndex(output, -1)

	if len(allMatches) == 0 {
		return nil
	}

	var results [][]string
	for i, match := range allMatches {
		fileName := output[match[2]:match[3]]
		contentStart := match[1]

		var contentEnd int
		if i+1 < len(allMatches) {
			contentEnd = allMatches[i+1][0]
		} else {
			contentEnd = len(output)
		}

		content := strings.TrimSpace(output[contentStart:contentEnd])
		results = append(results, []string{"", fileName, content})
	}

	return results
}
