package tui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogsModel is the logs viewer model
type LogsModel struct {
	basePath string
	width    int
	height   int
	lines    []string
	scroll   int
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

	// Auto-scroll to bottom
	if m.autoScroll && len(m.lines) > 0 {
		maxScroll := len(m.lines) - m.height + 10
		if maxScroll > 0 {
			m.scroll = maxScroll
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
			// Go to bottom
			maxScroll := len(m.lines) - m.height + 10
			if maxScroll > 0 {
				m.scroll = maxScroll
			}
			m.autoScroll = true
		case "g":
			// Go to top
			m.scroll = 0
			m.autoScroll = false
		case "f":
			// Toggle auto-scroll
			m.autoScroll = !m.autoScroll
		}
	}
	return m, nil
}

// View renders the logs
func (m *LogsModel) View() string {
	var sb strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)
	
	autoScrollIndicator := ""
	if m.autoScroll {
		autoScrollIndicator = " [AUTO-SCROLL]"
	}
	sb.WriteString(headerStyle.Render(fmt.Sprintf("Logs%s", autoScrollIndicator)))
	sb.WriteString("\n\n")

	// Calculate visible lines
	visibleLines := m.height - 6
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
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
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
		lineStyle := lipgloss.NewStyle()
		if strings.Contains(line, "[ERROR]") {
			lineStyle = lineStyle.Foreground(lipgloss.Color("196"))
		} else if strings.Contains(line, "[WARN]") {
			lineStyle = lineStyle.Foreground(lipgloss.Color("226"))
		} else if strings.Contains(line, "[SUCCESS]") {
			lineStyle = lineStyle.Foreground(lipgloss.Color("42"))
		} else if strings.Contains(line, "[DEBUG]") {
			lineStyle = lineStyle.Foreground(lipgloss.Color("241"))
		}
		
		content.WriteString(lineStyle.Render(line))
		content.WriteString("\n")
	}

	sb.WriteString(boxStyle.Render(content.String()))
	sb.WriteString("\n")

	// Footer
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	scrollInfo := fmt.Sprintf("Line %d-%d of %d", startIdx+1, endIdx, len(m.lines))
	sb.WriteString(footerStyle.Render(fmt.Sprintf("%s | [j/k] Scroll [g/G] Top/Bottom [f] Auto-scroll", scrollInfo)))

	return sb.String()
}
