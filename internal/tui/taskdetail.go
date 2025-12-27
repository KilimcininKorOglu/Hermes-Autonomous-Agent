package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/task"
)

// TaskDetailModel is the task detail screen model
type TaskDetailModel struct {
	basePath string
	width    int
	height   int
	task     *task.Task
	feature  *task.Feature
	scroll   int
}

// NewTaskDetailModel creates a new task detail model
func NewTaskDetailModel(basePath string) *TaskDetailModel {
	return &TaskDetailModel{
		basePath: basePath,
	}
}

// SetTask sets the task to display
func (m *TaskDetailModel) SetTask(t *task.Task) {
	m.task = t
	m.scroll = 0
	
	// Load feature info
	if t != nil {
		reader := task.NewReader(m.basePath)
		m.feature, _ = reader.GetFeatureByID(t.FeatureID)
	}
}

// SetSize updates the size
func (m *TaskDetailModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model
func (m *TaskDetailModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *TaskDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.scroll++
		case "k", "up":
			if m.scroll > 0 {
				m.scroll--
			}
		}
	}
	return m, nil
}

// View renders the task detail
func (m *TaskDetailModel) View() string {
	if m.task == nil {
		return "No task selected"
	}

	var sb strings.Builder
	t := m.task

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)
	sb.WriteString(titleStyle.Render(fmt.Sprintf("Task: %s", t.ID)))
	sb.WriteString("\n\n")

	// Task info box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(m.width - 4)

	var info strings.Builder
	boldStyle := lipgloss.NewStyle().Bold(true)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))

	// Name
	info.WriteString(boldStyle.Render("Name: "))
	info.WriteString(t.Name)
	info.WriteString("\n\n")

	// Status with color
	info.WriteString(boldStyle.Render("Status: "))
	statusStyle := lipgloss.NewStyle()
	switch t.Status {
	case task.StatusCompleted:
		statusStyle = statusStyle.Foreground(lipgloss.Color("42"))
	case task.StatusInProgress:
		statusStyle = statusStyle.Foreground(lipgloss.Color("226"))
	case task.StatusBlocked:
		statusStyle = statusStyle.Foreground(lipgloss.Color("196"))
	case task.StatusAtRisk:
		statusStyle = statusStyle.Foreground(lipgloss.Color("208"))
	case task.StatusPaused:
		statusStyle = statusStyle.Foreground(lipgloss.Color("141"))
	case task.StatusNotStarted:
		statusStyle = statusStyle.Foreground(lipgloss.Color("241"))
	}
	info.WriteString(statusStyle.Render(string(t.Status)))
	info.WriteString("\n\n")

	// Priority and Effort on same line
	info.WriteString(boldStyle.Render("Priority: "))
	priorityStyle := lipgloss.NewStyle()
	switch t.Priority {
	case task.PriorityP1:
		priorityStyle = priorityStyle.Foreground(lipgloss.Color("196"))
	case task.PriorityP2:
		priorityStyle = priorityStyle.Foreground(lipgloss.Color("226"))
	case task.PriorityP3:
		priorityStyle = priorityStyle.Foreground(lipgloss.Color("86"))
	case task.PriorityP4:
		priorityStyle = priorityStyle.Foreground(lipgloss.Color("241"))
	}
	info.WriteString(priorityStyle.Render(string(t.Priority)))
	if t.EstimatedEffort != "" {
		info.WriteString("  |  ")
		info.WriteString(boldStyle.Render("Effort: "))
		info.WriteString(t.EstimatedEffort)
	}
	info.WriteString("\n\n")

	// Feature
	info.WriteString(boldStyle.Render("Feature: "))
	info.WriteString(t.FeatureID)
	if m.feature != nil {
		info.WriteString(fmt.Sprintf(" - %s", m.feature.Name))
		if m.feature.TargetVersion != "" {
			info.WriteString(fmt.Sprintf(" (%s)", m.feature.TargetVersion))
		}
	}
	info.WriteString("\n\n")

	// Description
	if t.Description != "" {
		info.WriteString(sectionStyle.Render("Description"))
		info.WriteString("\n")
		info.WriteString(t.Description)
		info.WriteString("\n\n")
	}

	// Technical Details
	if t.TechnicalDetails != "" {
		info.WriteString(sectionStyle.Render("Technical Details"))
		info.WriteString("\n")
		info.WriteString(t.TechnicalDetails)
		info.WriteString("\n\n")
	}

	// Files to Touch
	if len(t.FilesToTouch) > 0 {
		info.WriteString(sectionStyle.Render("Files to Touch"))
		info.WriteString("\n")
		for _, f := range t.FilesToTouch {
			info.WriteString(fmt.Sprintf("  - %s\n", f))
		}
		info.WriteString("\n")
	}

	// Dependencies
	if len(t.Dependencies) > 0 {
		info.WriteString(sectionStyle.Render("Dependencies"))
		info.WriteString("\n")
		for _, d := range t.Dependencies {
			info.WriteString(fmt.Sprintf("  - %s\n", d))
		}
		info.WriteString("\n")
	}

	// Success Criteria
	if len(t.SuccessCriteria) > 0 {
		info.WriteString(sectionStyle.Render("Success Criteria"))
		info.WriteString("\n")
		for _, c := range t.SuccessCriteria {
			info.WriteString(fmt.Sprintf("  [ ] %s\n", c))
		}
	}

	sb.WriteString(boxStyle.Render(info.String()))
	sb.WriteString("\n\n")

	// Footer
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	sb.WriteString(footerStyle.Render("[Esc] Back to tasks | [j/k] Scroll"))

	return sb.String()
}
