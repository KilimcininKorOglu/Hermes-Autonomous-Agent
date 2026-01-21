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

	sb.WriteString(RenderScreenTitle(fmt.Sprintf("TASK: %s", t.ID)))

	// Task info box
	infoBox := BoxStyle.
		Padding(1, 2).
		Width(m.width - 4)

	var info strings.Builder
	boldStyle := lipgloss.NewStyle().Bold(true)

	// Name
	info.WriteString(boldStyle.Render("Name: "))
	info.WriteString(t.Name)
	info.WriteString("\n\n")

	// Status with color
	info.WriteString(boldStyle.Render("Status: "))
	statusStyle := MutedStyle
	switch t.Status {
	case task.StatusCompleted:
		statusStyle = SuccessStyle
	case task.StatusInProgress:
		statusStyle = WarningStyle
	case task.StatusBlocked, task.StatusAtRisk:
		statusStyle = ErrorStyle
	case task.StatusPaused:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	}
	info.WriteString(statusStyle.Render(string(t.Status)))
	info.WriteString("\n\n")

	// Priority and Effort on same line
	info.WriteString(boldStyle.Render("Priority: "))
	priorityStyle := MutedStyle
	switch t.Priority {
	case task.PriorityP1:
		priorityStyle = ErrorStyle
	case task.PriorityP2:
		priorityStyle = WarningStyle
	case task.PriorityP3:
		priorityStyle = SuccessStyle
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
		info.WriteString(SectionStyle.Render("Description"))
		info.WriteString("\n")
		info.WriteString(t.Description)
		info.WriteString("\n\n")
	}

	// Technical Details
	if t.TechnicalDetails != "" {
		info.WriteString(SectionStyle.Render("Technical Details"))
		info.WriteString("\n")
		info.WriteString(t.TechnicalDetails)
		info.WriteString("\n\n")
	}

	// Files to Touch
	if len(t.FilesToTouch) > 0 {
		info.WriteString(SectionStyle.Render("Files to Touch"))
		info.WriteString("\n")
		for _, f := range t.FilesToTouch {
			info.WriteString(fmt.Sprintf("  - %s\n", f))
		}
		info.WriteString("\n")
	}

	// Dependencies
	if len(t.Dependencies) > 0 {
		info.WriteString(SectionStyle.Render("Dependencies"))
		info.WriteString("\n")
		for _, d := range t.Dependencies {
			info.WriteString(fmt.Sprintf("  - %s\n", d))
		}
		info.WriteString("\n")
	}

	// Success Criteria
	if len(t.SuccessCriteria) > 0 {
		info.WriteString(SectionStyle.Render("Success Criteria"))
		info.WriteString("\n")
		for _, c := range t.SuccessCriteria {
			info.WriteString(fmt.Sprintf("  [ ] %s\n", c))
		}
	}

	sb.WriteString(infoBox.Render(info.String()))
	sb.WriteString("\n\n")

	sb.WriteString(MutedStyle.Render("[Esc] Back to tasks | [j/k] Scroll"))

	return sb.String()
}
