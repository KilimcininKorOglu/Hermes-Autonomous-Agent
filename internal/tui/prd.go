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
	"hermes/internal/ai"
	"hermes/internal/config"
	"hermes/internal/ui"
)

// PrdModel is the model for the PRD parser screen
type PrdModel struct {
	width        int
	height       int
	basePath     string
	textInput    textinput.Model
	dryRun       bool
	parsing      bool
	result       string
	filesCreated []string
	err          error
	focusIndex   int
	logger       *ui.Logger
}

// prdResultMsg is sent when PRD parsing completes
type prdResultMsg struct {
	files []string
	err   error
}

// NewPrdModel creates a new PRD model
func NewPrdModel(basePath string, logger *ui.Logger) *PrdModel {
	ti := textinput.New()
	ti.Placeholder = ".hermes/docs/PRD.md"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50

	return &PrdModel{
		basePath:   basePath,
		textInput:  ti,
		focusIndex: 0,
		logger:     logger,
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

	b.WriteString(RenderScreenTitle("PRD PARSER"))

	if m.parsing {
		b.WriteString(WarningStyle.Render("Parsing PRD..."))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(LabelStyle.Render("PRD File Path:"))
	b.WriteString("\n")
	if m.focusIndex == 0 {
		b.WriteString(SelectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	b.WriteString(LabelStyle.Render("Options:"))
	b.WriteString(" ")
	if m.focusIndex == 1 {
		b.WriteString(SelectedStyle.Render("> "))
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
		b.WriteString(SelectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(ButtonStyle.Render("Parse PRD"))
	b.WriteString("\n\n")

	if m.result != "" {
		b.WriteString(SuccessStyle.Render(m.result))
		b.WriteString("\n")

		if len(m.filesCreated) > 0 {
			for _, f := range m.filesCreated {
				b.WriteString(MutedStyle.Render("  - " + f))
				b.WriteString("\n")
			}
		}
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(MutedStyle.Render("Tab: Navigate | Space/Enter: Select | Default: .hermes/docs/PRD.md"))

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

		if m.logger != nil {
			m.logger.Info("Parsing PRD file: %s", prdPath)
		}

		prompt := buildPrdPromptForTUI(string(prdContent))

		result, err := ai.ExecuteWithRetry(ctx, provider, &ai.ExecuteOptions{
			Prompt:       prompt,
			Timeout:      cfg.AI.PrdTimeout,
			StreamOutput: false,
		}, &ai.RetryConfig{
			MaxRetries: cfg.AI.MaxRetries,
			Delay:      time.Duration(cfg.AI.RetryDelay) * time.Second,
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
**Target Version:** v1.0.0
**Estimated Duration:** X weeks
**Status:** NOT_STARTED

## Overview
[Feature description]

## Goals
- Goal 1
- Goal 2

## Tasks

### TXXX: Task Name
**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** X days

#### Description
[Clear, detailed description of what this task accomplishes]

#### Technical Details
[Implementation notes, architecture decisions, code patterns to follow]

#### Files to Touch
- `+"`path/to/file.go`"+` (new)
- `+"`path/to/existing.go`"+` (update)

#### Dependencies
- TYYY (if depends on another task, use actual task ID like T001, T002)
- None (if no dependencies)

IMPORTANT: Dependencies MUST be valid task IDs (T001, T002, etc.) or "None".
Do NOT use descriptions like "All backend features" or "Previous tasks".

#### Success Criteria
- [ ] Criterion 1
- [ ] Criterion 2

---

IMPORTANT RULES:
1. Create 3-6 tasks per feature, each task should be 0.5-3 days of work
2. Tasks should be atomic and independently testable
3. Use realistic effort estimates based on complexity
4. Dependencies MUST reference actual task IDs (T001, T002, etc.) or be "None"
5. Do NOT use vague dependencies like "All backend features" or "Previous tasks"
6. Success criteria must be specific and measurable
7. Priority levels: P1=Critical, P2=High, P3=Medium, P4=Low

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

	entries, err := os.ReadDir(tasksDir)
	if err == nil {
		var existingFiles []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				existingFiles = append(existingFiles, filepath.Join(tasksDir, entry.Name()))
			}
		}
		if len(existingFiles) > 0 {
			return existingFiles, nil
		}
	}

	fileRegex := regexp.MustCompile(`---FILE:\s*(.+?)---\s*([\s\S]*?)---END_FILE---`)
	matches := fileRegex.FindAllStringSubmatch(output, -1)

	if len(matches) == 0 {
		matches = parseConsecutiveFileMarkersForTUI(output)
	}

	var files []string

	if len(matches) == 0 {
		return nil, fmt.Errorf("AI output did not contain valid file markers (---FILE: ... ---END_FILE---). Please try again")
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
