package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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

	b.WriteString(RenderScreenTitle("CIRCUIT BREAKER"))

	if m.state == nil {
		b.WriteString(MutedStyle.Render("No circuit breaker state available"))
		return b.String()
	}

	b.WriteString(SectionStyle.Render("Current State"))
	b.WriteString("\n\n")

	stateStyle := SuccessStyle
	stateIcon := "[OK]"
	switch m.state.State {
	case circuit.StateHalfOpen:
		stateStyle = WarningStyle
		stateIcon = "[!]"
	case circuit.StateOpen:
		stateStyle = ErrorStyle
		stateIcon = "[X]"
	}

	b.WriteString(LabelStyle.Render("State:"))
	b.WriteString(stateStyle.Render(fmt.Sprintf("%s %s", stateIcon, m.state.State)))
	b.WriteString("\n")

	if m.state.Reason != "" {
		b.WriteString(LabelStyle.Render("Reason:"))
		b.WriteString(ValueStyle.Render(m.state.Reason))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(SectionStyle.Render("Statistics"))
	b.WriteString("\n\n")

	b.WriteString(LabelStyle.Render("Current Loop:"))
	b.WriteString(ValueStyle.Render(fmt.Sprintf("%d", m.state.CurrentLoop)))
	b.WriteString("\n")

	b.WriteString(LabelStyle.Render("Last Progress at Loop:"))
	b.WriteString(ValueStyle.Render(fmt.Sprintf("%d", m.state.LastProgress)))
	b.WriteString("\n")

	b.WriteString(LabelStyle.Render("Consecutive No Progress:"))
	b.WriteString(ValueStyle.Render(fmt.Sprintf("%d", m.state.ConsecutiveNoProgress)))
	b.WriteString("\n")

	b.WriteString(LabelStyle.Render("Consecutive Errors:"))
	b.WriteString(ValueStyle.Render(fmt.Sprintf("%d", m.state.ConsecutiveErrors)))
	b.WriteString("\n")

	b.WriteString(LabelStyle.Render("Total Opens:"))
	b.WriteString(ValueStyle.Render(fmt.Sprintf("%d", m.state.TotalOpens)))
	b.WriteString("\n")

	b.WriteString(LabelStyle.Render("Last Updated:"))
	b.WriteString(ValueStyle.Render(m.state.LastUpdated.Format("2006-01-02 15:04:05")))
	b.WriteString("\n\n")

	if m.state.State != circuit.StateClosed {
		b.WriteString(ButtonStyle.Render("Reset Circuit Breaker"))
	} else {
		b.WriteString(MutedStyle.Render("[ No Reset Needed ]"))
	}
	b.WriteString("\n\n")

	if m.message != "" {
		b.WriteString(SuccessStyle.Render(m.message))
		b.WriteString("\n")
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(MutedStyle.Render("r: Refresh | Space/Enter: Reset (when OPEN/HALF_OPEN)"))

	return b.String()
}
