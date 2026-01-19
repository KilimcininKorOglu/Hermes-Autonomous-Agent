package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/config"
	"hermes/internal/prompt"
)

// InitModel is the model for the init/project screen
type InitModel struct {
	width       int
	height      int
	textInput   textinput.Model
	initializing bool
	result      string
	createdDirs []string
	err         error
	focusIndex  int
}

// initResultMsg is sent when initialization completes
type initResultMsg struct {
	dirs []string
	err  error
}

// NewInitModel creates a new init model
func NewInitModel() *InitModel {
	ti := textinput.New()
	ti.Placeholder = ". (current directory)"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	return &InitModel{
		textInput:  ti,
		focusIndex: 0,
	}
}

// Init initializes the model
func (m *InitModel) Init() tea.Cmd {
	return textinput.Blink
}

// SetSize sets the size of the model
func (m *InitModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textInput.Width = width - 10
}

// Update handles messages
func (m *InitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.initializing {
			return m, nil
		}

		switch msg.String() {
		case "tab", "shift+tab":
			m.focusIndex = (m.focusIndex + 1) % 2
			if m.focusIndex == 0 {
				m.textInput.Focus()
			} else {
				m.textInput.Blur()
			}
		case " ", "enter":
			if m.focusIndex == 1 {
				projectPath := m.textInput.Value()
				if projectPath == "" {
					projectPath = "."
				}
				m.initializing = true
				m.result = ""
				m.createdDirs = nil
				m.err = nil
				return m, m.initProject(projectPath)
			}
		}

	case initResultMsg:
		m.initializing = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.createdDirs = msg.dirs
			m.result = "Project initialized successfully!"
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
func (m *InitModel) View() string {
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

	b.WriteString(titleStyle.Render("INITIALIZE PROJECT"))
	b.WriteString("\n\n")

	if m.initializing {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Initializing project..."))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(labelStyle.Render("Project Path (leave empty for current directory):"))
	b.WriteString("\n")
	if m.focusIndex == 0 {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	if m.focusIndex == 1 {
		b.WriteString(selectedStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(buttonStyle.Render("Initialize Project"))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("This will create:"))
	b.WriteString("\n")
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	b.WriteString(infoStyle.Render("  - .hermes/"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  - .hermes/tasks/"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  - .hermes/logs/"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  - .hermes/docs/"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  - .hermes/config.json"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  - .hermes/PROMPT.md"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  - .gitignore"))
	b.WriteString("\n\n")

	if m.result != "" {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		b.WriteString(successStyle.Render(m.result))
		b.WriteString("\n")

		if len(m.createdDirs) > 0 {
			fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
			for _, d := range m.createdDirs {
				b.WriteString(fileStyle.Render("  Created: " + d))
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
	b.WriteString(helpStyle.Render("Tab: Navigate | Space/Enter: Select"))

	return b.String()
}

// Reset clears the model
func (m *InitModel) Reset() {
	m.textInput.SetValue("")
	m.initializing = false
	m.result = ""
	m.createdDirs = nil
	m.err = nil
	m.focusIndex = 0
	m.textInput.Focus()
}

// initProject initializes the project
func (m *InitModel) initProject(projectPath string) tea.Cmd {
	return func() tea.Msg {
		var created []string

		if projectPath != "." {
			if err := os.MkdirAll(projectPath, 0755); err != nil {
				return initResultMsg{err: fmt.Errorf("failed to create project directory: %w", err)}
			}
		}

		gitDir := filepath.Join(projectPath, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			cmd := exec.Command("git", "init")
			cmd.Dir = projectPath
			if err := cmd.Run(); err == nil {
				created = append(created, ".git/")
			}
		}

		dirs := []string{
			".hermes",
			".hermes/tasks",
			".hermes/logs",
			".hermes/docs",
		}

		for _, dir := range dirs {
			path := filepath.Join(projectPath, dir)
			if err := os.MkdirAll(path, 0755); err != nil {
				return initResultMsg{err: fmt.Errorf("failed to create %s: %w", dir, err)}
			}
			created = append(created, dir+"/")
		}

		configPath := filepath.Join(projectPath, ".hermes", "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			cfg := config.DefaultConfig()
			if err := config.Save(configPath, cfg); err != nil {
				return initResultMsg{err: fmt.Errorf("failed to create config: %w", err)}
			}
			created = append(created, ".hermes/config.json")
		}

		injector := prompt.NewInjector(projectPath)
		if err := injector.CreateDefault(); err != nil {
			return initResultMsg{err: fmt.Errorf("failed to create PROMPT.md: %w", err)}
		}
		created = append(created, ".hermes/PROMPT.md")

		gitignorePath := filepath.Join(projectPath, ".gitignore")
		createGitignoreForTUI(gitignorePath)
		created = append(created, ".gitignore")

		return initResultMsg{dirs: created}
	}
}

func createGitignoreForTUI(path string) {
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		defer f.Close()
		f.WriteString("\n# Hermes\n.hermes/\n")
		return
	}

	content := `# Hermes
.hermes/

# Dependencies
node_modules/
vendor/
.venv/
venv/
__pycache__/
*.pyc

# Build outputs
dist/
build/
out/
bin/
*.exe
*.dll
*.so
*.dylib

# Environment
.env
.env.local
.env.*.local
*.local

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log
logs/

# Testing
coverage/
.coverage
.nyc_output/

# Misc
*.tmp
*.temp
.cache/
`

	os.WriteFile(path, []byte(content), 0644)
}
