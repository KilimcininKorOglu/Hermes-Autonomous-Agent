package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"hermes/internal/cmd"
)

// Version is set by -ldflags during build
var Version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "hermes",
		Short:   "Hermes Autonomous Agent",
		Long:    "AI-powered autonomous application development system",
		Version: Version,
		Run: func(c *cobra.Command, args []string) {
			fmt.Println("Hermes Autonomous Agent", Version)
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
	rootCmd.AddCommand(cmd.NewTaskCmd())
	rootCmd.AddCommand(cmd.NewLogCmd())
	rootCmd.AddCommand(cmd.NewIdeaCmd())
	rootCmd.AddCommand(cmd.NewConvertPrdCmd())

	// Set version for update command
	cmd.SetUpdateVersion(Version)
	rootCmd.AddCommand(cmd.NewUpdateCmd())
	rootCmd.AddCommand(cmd.NewInstallCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
