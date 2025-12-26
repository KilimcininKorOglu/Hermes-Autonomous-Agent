package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"hermes/internal/tui"
)

// NewTuiCmd creates the tui subcommand
func NewTuiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI",
		Long:  "Start the interactive terminal user interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tuiExecute()
		},
	}

	return cmd
}

func tuiExecute() error {
	app, err := tui.NewApp(".")
	if err != nil {
		return fmt.Errorf("failed to initialize TUI: %w", err)
	}

	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
