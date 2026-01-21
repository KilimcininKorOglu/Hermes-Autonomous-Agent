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
		filter:   "",
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
			m.filter = ""
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

	sb.WriteString(RenderScreenTitle("TASKS"))

	// Calculate dynamic column widths
	nameWidth := m.width - 66
	if nameWidth < 20 {
		nameWidth = 20
	}
	if nameWidth > 50 {
		nameWidth = 50
	}

	// Filter bar
	filterBar := "[a]All [c]Completed [p]In Progress [n]Not Started [b]Blocked"
	if m.filter != "" {
		filterBar += fmt.Sprintf(" | Filter: %s", m.filter)
	}
	sb.WriteString(MutedStyle.Render(filterBar))
	sb.WriteString("\n\n")

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true)

	// Fixed width columns
	header := fmt.Sprintf("%-6s | %-*s | %-12s | %-8s | %-10s | %-6s", "ID", nameWidth, "Name", "Status", "Priority", "Effort", "Feature")
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")

	// Tasks
	tasks := m.filteredTasks()
	if len(tasks) == 0 {
		sb.WriteString("\n  No tasks found\n")
		return sb.String()
	}

	// Calculate visible rows based on height
	maxRows := m.height - 10
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

	for i := startIdx; i < endIdx; i++ {
		t := tasks[i]
		name := t.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}

		effort := t.EstimatedEffort
		if effort == "" {
			effort = "-"
		}
		if len(effort) > 10 {
			effort = effort[:7] + "..."
		}

		row := fmt.Sprintf("%-6s | %-*s | %-12s | %-8s | %-10s | %-6s", t.ID, nameWidth, name, string(t.Status), string(t.Priority), effort, t.FeatureID)

		rowStyle := lipgloss.NewStyle()
		if i == m.cursor {
			rowStyle = SelectedStyle.Background(lipgloss.Color("62"))
		} else {
			switch t.Status {
			case task.StatusCompleted:
				rowStyle = SuccessStyle
			case task.StatusInProgress:
				rowStyle = WarningStyle
			case task.StatusBlocked, task.StatusAtRisk:
				rowStyle = ErrorStyle
			case task.StatusPaused:
				rowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
			case task.StatusNotStarted:
				rowStyle = MutedStyle
			}
		}

		sb.WriteString(rowStyle.Render(row))
		sb.WriteString("\n")
	}

	// Footer with scroll info
	sb.WriteString("\n")
	if len(tasks) > maxRows {
		sb.WriteString(MutedStyle.Render(fmt.Sprintf("Showing %d-%d of %d tasks (j/k to scroll)", startIdx+1, endIdx, len(tasks))))
	} else {
		sb.WriteString(MutedStyle.Render(fmt.Sprintf("Showing %d tasks", len(tasks))))
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
