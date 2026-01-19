package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/circuit"
)

// CircuitBreakerModel is the model for the circuit breaker screen
type CircuitBreakerModel struct {
	width    int
	height   int
	basePath string
	breaker  *circuit.Breaker
	state    *circuit.BreakerState
	err      error
	message  string
}

// NewCircuitBreakerModel creates a new circuit breaker model
func NewCircuitBreakerModel(basePath string) *CircuitBreakerModel {
	breaker := circuit.New(basePath)
	state, _ := breaker.GetState()

	return &CircuitBreakerModel{
		basePath: basePath,
		breaker:  breaker,
		state:    state,
	}
}

// Init initializes the model
func (m *CircuitBreakerModel) Init() tea.Cmd {
	return nil
}

// SetSize sets the size of the model
func (m *CircuitBreakerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Refresh reloads the circuit breaker state
func (m *CircuitBreakerModel) Refresh() {
	state, err := m.breaker.GetState()
	if err != nil {
		m.err = err
	} else {
		m.state = state
		m.err = nil
	}
}

// Update handles messages
func (m *CircuitBreakerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.Refresh()
			m.message = ""
		case "enter", " ":
			if m.state != nil && m.state.State != circuit.StateClosed {
				err := m.breaker.Reset("Manual reset via TUI")
				if err != nil {
					m.err = err
				} else {
					m.message = "Circuit breaker reset successfully!"
					m.Refresh()
				}
			}
		}
	}

	return m, nil
}

// View renders the model
func (m *CircuitBreakerModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(25)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		MarginTop(1)

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("62")).
		Padding(0, 2)

	disabledButtonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("236")).
		Padding(0, 2)

	b.WriteString(titleStyle.Render("CIRCUIT BREAKER"))
	b.WriteString("\n\n")

	if m.state == nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No circuit breaker state available"))
		return b.String()
	}

	b.WriteString(sectionStyle.Render("Current State"))
	b.WriteString("\n\n")

	stateColor := "82"
	stateIcon := "[OK]"
	switch m.state.State {
	case circuit.StateHalfOpen:
		stateColor = "214"
		stateIcon = "[!]"
	case circuit.StateOpen:
		stateColor = "196"
		stateIcon = "[X]"
	}

	b.WriteString(labelStyle.Render("State:"))
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(stateColor)).Render(fmt.Sprintf("%s %s", stateIcon, m.state.State)))
	b.WriteString("\n")

	if m.state.Reason != "" {
		b.WriteString(labelStyle.Render("Reason:"))
		b.WriteString(valueStyle.Render(m.state.Reason))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Statistics"))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Current Loop:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", m.state.CurrentLoop)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Last Progress at Loop:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", m.state.LastProgress)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Consecutive No Progress:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", m.state.ConsecutiveNoProgress)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Consecutive Errors:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", m.state.ConsecutiveErrors)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Total Opens:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", m.state.TotalOpens)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Last Updated:"))
	b.WriteString(valueStyle.Render(m.state.LastUpdated.Format("2006-01-02 15:04:05")))
	b.WriteString("\n\n")

	if m.state.State != circuit.StateClosed {
		b.WriteString(buttonStyle.Render("Reset Circuit Breaker"))
	} else {
		b.WriteString(disabledButtonStyle.Render("No Reset Needed"))
	}
	b.WriteString("\n\n")

	if m.message != "" {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		b.WriteString(successStyle.Render(m.message))
		b.WriteString("\n")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	b.WriteString(helpStyle.Render("r: Refresh | Space/Enter: Reset (when OPEN/HALF_OPEN)"))

	return b.String()
}
