package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/circuit"
	"hermes/internal/task"
)

// DashboardModel is the dashboard screen model
type DashboardModel struct {
	basePath       string
	width          int
	height         int
	progress       *task.Progress
	breaker        *circuit.BreakerState
	currentTask    *task.Task
	currentFeature *task.Feature
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(basePath string) *DashboardModel {
	m := &DashboardModel{
		basePath: basePath,
	}
	m.Refresh()
	return m
}

// Refresh reloads data
func (m *DashboardModel) Refresh() {
	reader := task.NewReader(m.basePath)
	m.progress, _ = reader.GetProgress()

	breaker := circuit.New(m.basePath)
	m.breaker, _ = breaker.GetState()

	m.currentTask, _ = reader.GetNextTask()
	if m.currentTask != nil {
		m.currentFeature, _ = reader.GetFeatureByID(m.currentTask.FeatureID)
	}
}

// SetSize updates the size
func (m *DashboardModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the dashboard
func (m *DashboardModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View renders the dashboard
func (m *DashboardModel) View() string {
	var b strings.Builder

	b.WriteString(RenderScreenTitle("DASHBOARD"))

	// Progress box
	progressContent := m.progressView()
	progressBox := BoxStyle.
		Width(m.width/2 - 4).
		Render(progressContent)

	// Circuit breaker box
	circuitContent := m.circuitView()
	circuitBox := BoxStyle.
		Width(m.width/2 - 4).
		Render(circuitContent)

	// Current task box
	taskContent := m.currentTaskView()
	taskBox := BoxStyle.
		Width(m.width - 4).
		Render(taskContent)

	// Layout
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, progressBox, circuitBox)

	b.WriteString(lipgloss.JoinVertical(
		lipgloss.Left,
		topRow,
		taskBox,
	))

	return b.String()
}

func (m *DashboardModel) progressView() string {
	var sb strings.Builder

	sb.WriteString(SectionStyle.Render("Progress"))
	sb.WriteString("\n\n")

	if m.progress == nil {
		sb.WriteString("No tasks found")
		return sb.String()
	}

	// Progress bar
	percent := int(m.progress.Percentage)
	barWidth := m.width/2 - 16
	sb.WriteString(fmt.Sprintf("%s %.1f%%\n\n", RenderProgressBar(percent, barWidth), m.progress.Percentage))

	sb.WriteString(fmt.Sprintf("Total:       %d\n", m.progress.Total))
	sb.WriteString(fmt.Sprintf("Completed:   %d\n", m.progress.Completed))
	sb.WriteString(fmt.Sprintf("In Progress: %d\n", m.progress.InProgress))
	sb.WriteString(fmt.Sprintf("Not Started: %d\n", m.progress.NotStarted))
	sb.WriteString(fmt.Sprintf("Blocked:     %d", m.progress.Blocked))

	return sb.String()
}

func (m *DashboardModel) circuitView() string {
	var sb strings.Builder

	sb.WriteString(SectionStyle.Render("Circuit Breaker"))
	sb.WriteString("\n\n")

	if m.breaker == nil {
		sb.WriteString("Not initialized")
		return sb.String()
	}

	stateIcon := "[OK]"
	stateStyle := SuccessStyle

	switch m.breaker.State {
	case circuit.StateClosed:
		stateStyle = SuccessStyle
	case circuit.StateHalfOpen:
		stateStyle = WarningStyle
		stateIcon = "[!!]"
	case circuit.StateOpen:
		stateStyle = ErrorStyle
		stateIcon = "[XX]"
	}

	sb.WriteString(fmt.Sprintf("State: %s %s\n\n", stateIcon, stateStyle.Render(string(m.breaker.State))))
	sb.WriteString(fmt.Sprintf("Loops since progress: %d\n", m.breaker.ConsecutiveNoProgress))
	sb.WriteString(fmt.Sprintf("Last progress: Loop #%d\n", m.breaker.LastProgress))
	sb.WriteString(fmt.Sprintf("Total opens: %d", m.breaker.TotalOpens))

	return sb.String()
}

func (m *DashboardModel) currentTaskView() string {
	var sb strings.Builder

	sb.WriteString(SectionStyle.Render("Current Task"))
	sb.WriteString("\n\n")

	if m.currentTask == nil {
		sb.WriteString("No pending tasks - all complete!")
		return sb.String()
	}

	t := m.currentTask

	// Task ID and Name
	sb.WriteString(LabelStyle.Render("ID:       "))
	sb.WriteString(fmt.Sprintf("%s\n", t.ID))
	sb.WriteString(LabelStyle.Render("Name:     "))
	sb.WriteString(fmt.Sprintf("%s\n", t.Name))
	sb.WriteString(LabelStyle.Render("Feature:  "))
	sb.WriteString(t.FeatureID)
	if m.currentFeature != nil && m.currentFeature.TargetVersion != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", m.currentFeature.TargetVersion))
	}
	sb.WriteString("\n")

	// Priority with color
	sb.WriteString(LabelStyle.Render("Priority: "))
	priorityStyle := lipgloss.NewStyle()
	switch t.Priority {
	case task.PriorityP1:
		priorityStyle = ErrorStyle
	case task.PriorityP2:
		priorityStyle = WarningStyle
	case task.PriorityP3:
		priorityStyle = SuccessStyle
	case task.PriorityP4:
		priorityStyle = MutedStyle
	}
	sb.WriteString(priorityStyle.Render(string(t.Priority)))
	sb.WriteString("\n")

	// Estimated Effort
	if t.EstimatedEffort != "" {
		sb.WriteString(LabelStyle.Render("Effort:   "))
		sb.WriteString(fmt.Sprintf("%s\n", t.EstimatedEffort))
	}

	// Status
	sb.WriteString(LabelStyle.Render("Status:   "))
	sb.WriteString(fmt.Sprintf("%s\n", t.Status))

	// Description (truncated)
	if t.Description != "" {
		sb.WriteString("\n")
		sb.WriteString(SectionStyle.Render("Description"))
		sb.WriteString("\n")
		desc := t.Description
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}
		sb.WriteString(desc)
		sb.WriteString("\n")
	}

	// Files to Touch
	if len(t.FilesToTouch) > 0 {
		sb.WriteString("\n")
		sb.WriteString(SectionStyle.Render("Files to Touch"))
		sb.WriteString("\n")
		maxFiles := 5
		for i, f := range t.FilesToTouch {
			if i >= maxFiles {
				sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(t.FilesToTouch)-maxFiles))
				break
			}
			sb.WriteString(fmt.Sprintf("  - %s\n", f))
		}
	}

	return sb.String()
}
