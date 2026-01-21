package tui

import "github.com/charmbracelet/lipgloss"

// Common styles for all TUI screens
var (
	// Screen title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			MarginBottom(1)

	// Section header style
	SectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))

	// Label style (for form labels)
	LabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Width(20)

	// Value style (for form values)
	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	// Selected item style
	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	// Success/completed style
	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("82"))

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	// Warning style
	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))

	// Muted/help text style
	MutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// Box style with rounded border
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	// Button style
	ButtonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62")).
			Padding(0, 2)

	// Active/running button style
	ActiveButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("196")).
				Padding(0, 2)
)

// RenderScreenTitle renders a consistent screen title
func RenderScreenTitle(title string) string {
	return TitleStyle.Render(title) + "\n\n"
}

// RenderSection renders a section with header
func RenderSection(header string) string {
	return SectionStyle.Render(header) + "\n"
}

// RenderProgressBar renders a progress bar
func RenderProgressBar(percent int, width int) string {
	if width < 10 {
		width = 10
	}
	if width > 50 {
		width = 50
	}
	filled := (percent * width) / 100
	if filled > width {
		filled = width
	}
	empty := width - filled
	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}
	return "[" + bar + "]"
}
