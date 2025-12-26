package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"hermes/internal/cmd"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "hermes",
		Short:   "Hermes Autonomous Agent",
		Long:    "Autonomous AI development loop using Claude Code SDK",
		Version: version,
		Run: func(c *cobra.Command, args []string) {
			fmt.Println("Hermes Autonomous Agent", version)
			fmt.Println("Use 'hermes --help' for available commands")
		},
	}

	// Add subcommands
	rootCmd.AddCommand(cmd.NewRunCmd())
	rootCmd.AddCommand(cmd.NewPrdCmd())
	rootCmd.AddCommand(cmd.NewAddCmd())
	rootCmd.AddCommand(cmd.NewInitCmd())
	rootCmd.AddCommand(cmd.NewStatusCmd())
	rootCmd.AddCommand(cmd.NewTuiCmd())
	rootCmd.AddCommand(cmd.NewResetCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
