package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/updater"
)

// UpdateModel is the model for the update screen
type UpdateModel struct {
	width          int
	height         int
	currentVersion string
	checking       bool
	updating       bool
	release        *updater.Release
	hasUpdate      bool
	message        string
	err            error
}

// updateCheckMsg is sent when update check completes
type updateCheckMsg struct {
	release   *updater.Release
	hasUpdate bool
	err       error
}

// updateInstallMsg is sent when update installation completes
type updateInstallMsg struct {
	success bool
	err     error
}

// NewUpdateModel creates a new update model
func NewUpdateModel(version string) *UpdateModel {
	return &UpdateModel{
		currentVersion: version,
	}
}

// Init initializes the model
func (m *UpdateModel) Init() tea.Cmd {
	return nil
}

// SetSize sets the size of the model
func (m *UpdateModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages
func (m *UpdateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.checking || m.updating {
			return m, nil
		}

		switch msg.String() {
		case "c":
			m.checking = true
			m.message = ""
			m.err = nil
			return m, m.checkForUpdates()
		case "u", "enter", " ":
			if m.hasUpdate && m.release != nil {
				m.updating = true
				m.message = ""
				m.err = nil
				return m, m.installUpdate()
			}
		}

	case updateCheckMsg:
		m.checking = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.release = msg.release
			m.hasUpdate = msg.hasUpdate
			if !msg.hasUpdate {
				m.message = "You are running the latest version!"
			}
		}
		return m, nil

	case updateInstallMsg:
		m.updating = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.message = "Update installed successfully! Please restart Hermes."
			m.hasUpdate = false
		}
		return m, nil
	}

	return m, nil
}

// View renders the model
func (m *UpdateModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(20)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("62")).
		Padding(0, 2)

	disabledButtonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("236")).
		Padding(0, 2)

	b.WriteString(titleStyle.Render("UPDATE"))
	b.WriteString("\n\n")

	if m.checking {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Checking for updates..."))
		b.WriteString("\n")
		return b.String()
	}

	if m.updating {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Downloading and installing update..."))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(labelStyle.Render("Current Version:"))
	b.WriteString(valueStyle.Render(m.currentVersion))
	b.WriteString("\n\n")

	if m.release != nil && m.hasUpdate {
		newVersionStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("82"))

		b.WriteString(labelStyle.Render("New Version:"))
		b.WriteString(newVersionStyle.Render(m.release.TagName))
		b.WriteString("\n")

		b.WriteString(labelStyle.Render("Release Name:"))
		b.WriteString(valueStyle.Render(m.release.Name))
		b.WriteString("\n\n")
	}

	b.WriteString(buttonStyle.Render("Check for Updates (c)"))
	b.WriteString("  ")

	if m.hasUpdate {
		b.WriteString(buttonStyle.Render("Install Update (u)"))
	} else {
		b.WriteString(disabledButtonStyle.Render("Install Update"))
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
	b.WriteString(helpStyle.Render("c: Check for updates | u: Install update"))

	return b.String()
}

// checkForUpdates checks GitHub for updates
func (m *UpdateModel) checkForUpdates() tea.Cmd {
	return func() tea.Msg {
		u := updater.New(m.currentVersion)

		release, hasUpdate, err := u.CheckUpdate()
		if err != nil {
			return updateCheckMsg{err: err}
		}

		return updateCheckMsg{
			release:   release,
			hasUpdate: hasUpdate,
		}
	}
}

// installUpdate downloads and installs the update
func (m *UpdateModel) installUpdate() tea.Cmd {
	return func() tea.Msg {
		if m.release == nil {
			return updateInstallMsg{err: fmt.Errorf("no release available")}
		}

		u := updater.New(m.currentVersion)
		asset := u.FindAsset(m.release)
		if asset == nil {
			return updateInstallMsg{err: fmt.Errorf("no binary found for your platform")}
		}

		if err := u.DownloadAndReplace(asset); err != nil {
			return updateInstallMsg{err: err}
		}

		return updateInstallMsg{success: true}
	}
}
