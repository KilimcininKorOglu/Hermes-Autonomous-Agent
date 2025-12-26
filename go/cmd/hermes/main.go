package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "hermes",
		Short:   "Hermes Autonomous Agent",
		Long:    "Autonomous AI development loop using Claude Code SDK",
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hermes Autonomous Agent", version)
			fmt.Println("Use 'hermes --help' for available commands")
		},
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
