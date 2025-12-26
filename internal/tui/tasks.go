package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/task"
)

// TasksModel is the tasks screen model
type TasksModel struct {
	basePath string
	width    int
	height   int
	tasks    []task.Task
	cursor   int
	filter   task.Status
}

// NewTasksModel creates a new tasks model
func NewTasksModel(basePath string) *TasksModel {
	m := &TasksModel{
		basePath: basePath,
		filter:   "", // All tasks
	}
	m.Refresh()
	return m
}

// Refresh reloads tasks
func (m *TasksModel) Refresh() {
	reader := task.NewReader(m.basePath)
	m.tasks, _ = reader.GetAllTasks()
}

// SetSize updates the size
func (m *TasksModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the tasks screen
func (m *TasksModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *TasksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.filteredTasks())-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "a":
			m.filter = "" // All
			m.cursor = 0
		case "c":
			m.filter = task.StatusCompleted
			m.cursor = 0
		case "p":
			m.filter = task.StatusInProgress
			m.cursor = 0
		case "n":
			m.filter = task.StatusNotStarted
			m.cursor = 0
		case "b":
			m.filter = task.StatusBlocked
			m.cursor = 0
		}
	}
	return m, nil
}

// View renders the tasks screen
func (m *TasksModel) View() string {
	var sb strings.Builder

	// Calculate dynamic column widths
	nameWidth := m.width - 50 // ID(6) + Status(12) + Priority(8) + Feature(6) + separators(~18)
	if nameWidth < 20 {
		nameWidth = 20
	}
	if nameWidth > 60 {
		nameWidth = 60
	}

	// Filter bar
	filterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginBottom(1)

	filterBar := "[a]All [c]Completed [p]In Progress [n]Not Started [b]Blocked"
	if m.filter != "" {
		filterBar += fmt.Sprintf(" | Filter: %s", m.filter)
	}
	sb.WriteString(filterStyle.Render(filterBar))
	sb.WriteString("\n\n")

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true)

	headerFmt := fmt.Sprintf("%%-6s | %%-%ds | %%-12s | %%-8s | %%-6s", nameWidth)
	header := fmt.Sprintf(headerFmt, "ID", "Name", "Status", "Priority", "Feature")
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")

	// Tasks
	tasks := m.filteredTasks()
	if len(tasks) == 0 {
		sb.WriteString("\n  No tasks found\n")
		return sb.String()
	}

	// Calculate visible rows based on height
	maxRows := m.height - 8 // Account for header, filter bar, footer
	if maxRows < 5 {
		maxRows = 5
	}

	// Scroll offset
	startIdx := 0
	if m.cursor >= maxRows {
		startIdx = m.cursor - maxRows + 1
	}
	endIdx := startIdx + maxRows
	if endIdx > len(tasks) {
		endIdx = len(tasks)
	}

	rowFmt := fmt.Sprintf("%%-6s | %%-%ds | %%-12s | %%-8s | %%-6s", nameWidth)

	for i := startIdx; i < endIdx; i++ {
		t := tasks[i]
		name := t.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}

		row := fmt.Sprintf(rowFmt, t.ID, name, t.Status, t.Priority, t.FeatureID)

		rowStyle := lipgloss.NewStyle()
		if i == m.cursor {
			rowStyle = rowStyle.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("255"))
		} else {
			// Color by status
			switch t.Status {
			case task.StatusCompleted:
				rowStyle = rowStyle.Foreground(lipgloss.Color("42"))
			case task.StatusInProgress:
				rowStyle = rowStyle.Foreground(lipgloss.Color("226"))
			case task.StatusBlocked:
				rowStyle = rowStyle.Foreground(lipgloss.Color("196"))
			case task.StatusNotStarted:
				rowStyle = rowStyle.Foreground(lipgloss.Color("241"))
			}
		}

		sb.WriteString(rowStyle.Render(row))
		sb.WriteString("\n")
	}

	// Footer with scroll info
	if len(tasks) > maxRows {
		sb.WriteString(fmt.Sprintf("\nShowing %d-%d of %d tasks (j/k to scroll)", startIdx+1, endIdx, len(tasks)))
	} else {
		sb.WriteString(fmt.Sprintf("\nShowing %d tasks", len(tasks)))
	}

	return sb.String()
}

func (m *TasksModel) filteredTasks() []task.Task {
	if m.filter == "" {
		return m.tasks
	}

	var filtered []task.Task
	for _, t := range m.tasks {
		if t.Status == m.filter {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
