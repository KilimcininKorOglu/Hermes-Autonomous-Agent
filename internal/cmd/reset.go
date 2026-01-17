package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"hermes/internal/circuit"
)

// NewResetCmd creates the reset subcommand
func NewResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset circuit breaker",
		Long:  "Reset the circuit breaker to allow execution to continue",
		RunE: func(cmd *cobra.Command, args []string) error {
			return resetExecute()
		},
	}

	return cmd
}

func resetExecute() error {
	breaker := circuit.New(".")

	state, err := breaker.GetState()
	if err != nil {
		return err
	}

	if state.State == circuit.StateClosed {
		fmt.Println("Circuit breaker is already closed (normal state).")
		return nil
	}

	fmt.Printf("Current state: %s\n", state.State)
	fmt.Printf("Reason: %s\n", state.Reason)

	if err := breaker.Reset("Manual reset via CLI"); err != nil {
		return err
	}

	fmt.Println("\nCircuit breaker reset successfully.")
	fmt.Println("You can now run 'hermes run' to continue.")

	return nil
}
