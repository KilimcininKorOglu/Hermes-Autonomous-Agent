package tui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// LogsModel is the logs viewer model
type LogsModel struct {
	basePath   string
	width      int
	height     int
	lines      []string
	scroll     int
	autoScroll bool
}

// NewLogsModel creates a new logs model
func NewLogsModel(basePath string) *LogsModel {
	m := &LogsModel{
		basePath:   basePath,
		autoScroll: true,
	}
	m.Refresh()
	return m
}

// Refresh reloads log file
func (m *LogsModel) Refresh() {
	logPath := filepath.Join(m.basePath, ".hermes", "logs", "hermes.log")

	file, err := os.Open(logPath)
	if err != nil {
		m.lines = []string{"No log file found.", "", "Logs will appear here when you run tasks."}
		return
	}
	defer file.Close()

	m.lines = nil
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		m.lines = append(m.lines, scanner.Text())
	}

	if len(m.lines) == 0 {
		m.lines = []string{"Log file is empty.", "", "Logs will appear here when you run tasks."}
		return
	}

	// Auto-scroll to bottom
	if m.autoScroll {
		visibleLines := m.height - 8
		if visibleLines < 5 {
			visibleLines = 5
		}
		maxScroll := len(m.lines) - visibleLines
		if maxScroll > 0 {
			m.scroll = maxScroll
		} else {
			m.scroll = 0
		}
	}
}

// SetSize updates the size
func (m *LogsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model
func (m *LogsModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *LogsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			maxScroll := len(m.lines) - m.height + 10
			if m.scroll < maxScroll {
				m.scroll++
			}
			m.autoScroll = false
		case "k", "up":
			if m.scroll > 0 {
				m.scroll--
			}
			m.autoScroll = false
		case "G":
			maxScroll := len(m.lines) - m.height + 10
			if maxScroll > 0 {
				m.scroll = maxScroll
			}
			m.autoScroll = true
		case "g":
			m.scroll = 0
			m.autoScroll = false
		case "f":
			m.autoScroll = !m.autoScroll
		}
	}
	return m, nil
}

// View renders the logs
func (m *LogsModel) View() string {
	var sb strings.Builder

	// Title with auto-scroll indicator
	title := "LOGS"
	if m.autoScroll {
		title += " [AUTO-SCROLL]"
	}
	sb.WriteString(RenderScreenTitle(title))

	// Calculate visible lines
	visibleLines := m.height - 8
	if visibleLines < 5 {
		visibleLines = 5
	}

	startIdx := m.scroll
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + visibleLines
	if endIdx > len(m.lines) {
		endIdx = len(m.lines)
	}

	// Log content box
	logBox := BoxStyle.
		Width(m.width - 4).
		Height(visibleLines)

	var content strings.Builder
	for i := startIdx; i < endIdx; i++ {
		line := m.lines[i]

		// Truncate long lines
		if len(line) > m.width-8 {
			line = line[:m.width-11] + "..."
		}

		// Color based on log level
		if strings.Contains(line, "[ERROR]") {
			line = ErrorStyle.Render(line)
		} else if strings.Contains(line, "[WARN]") {
			line = WarningStyle.Render(line)
		} else if strings.Contains(line, "[SUCCESS]") {
			line = SuccessStyle.Render(line)
		} else if strings.Contains(line, "[DEBUG]") {
			line = MutedStyle.Render(line)
		}

		content.WriteString(line)
		content.WriteString("\n")
	}

	sb.WriteString(logBox.Render(content.String()))
	sb.WriteString("\n")

	// Footer
	scrollInfo := fmt.Sprintf("Line %d-%d of %d", startIdx+1, endIdx, len(m.lines))
	sb.WriteString(MutedStyle.Render(fmt.Sprintf("%s | [j/k] Scroll [g] Top [G] Bottom [f] Auto-scroll", scrollInfo)))

	return sb.String()
}
