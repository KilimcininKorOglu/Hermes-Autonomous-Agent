package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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

	b.WriteString(RenderScreenTitle("UPDATE"))

	if m.checking {
		b.WriteString(WarningStyle.Render("Checking for updates..."))
		b.WriteString("\n")
		return b.String()
	}

	if m.updating {
		b.WriteString(WarningStyle.Render("Downloading and installing update..."))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(LabelStyle.Render("Current Version:"))
	b.WriteString(ValueStyle.Render(m.currentVersion))
	b.WriteString("\n\n")

	if m.release != nil && m.hasUpdate {
		b.WriteString(LabelStyle.Render("New Version:"))
		b.WriteString(SuccessStyle.Render(m.release.TagName))
		b.WriteString("\n")

		b.WriteString(LabelStyle.Render("Release Name:"))
		b.WriteString(ValueStyle.Render(m.release.Name))
		b.WriteString("\n\n")
	}

	b.WriteString(ButtonStyle.Render("Check for Updates (c)"))
	b.WriteString("  ")

	if m.hasUpdate {
		b.WriteString(ButtonStyle.Render("Install Update (u)"))
	} else {
		b.WriteString(MutedStyle.Render("[ Install Update ]"))
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
	b.WriteString(MutedStyle.Render("c: Check for updates | u: Install update"))

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
