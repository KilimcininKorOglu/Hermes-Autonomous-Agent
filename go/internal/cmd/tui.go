package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
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
	// TUI will be implemented in Phase 12
	fmt.Println("TUI is not implemented yet.")
	fmt.Println("Use 'hermes status' to view tasks or 'hermes run' to execute.")
	return nil
}
