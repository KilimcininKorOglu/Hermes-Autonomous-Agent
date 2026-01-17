package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"hermes/internal/circuit"
	"hermes/internal/task"
	"hermes/internal/ui"
)

type statusOptions struct {
	filter   string
	priority string
}

// NewStatusCmd creates the status subcommand
func NewStatusCmd() *cobra.Command {
	opts := &statusOptions{}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show task status",
		Long:  "Display task progress table and statistics",
		Example: `  hermes status
  hermes status --filter IN_PROGRESS
  hermes status --priority P1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return statusExecute(opts)
		},
	}

	cmd.Flags().StringVar(&opts.filter, "filter", "", "Filter by status (NOT_STARTED, IN_PROGRESS, COMPLETED, BLOCKED)")
	cmd.Flags().StringVar(&opts.priority, "priority", "", "Filter by priority (P1, P2, P3, P4)")

	return cmd
}

func statusExecute(opts *statusOptions) error {
	reader := task.NewReader(".")

	if !reader.HasTasks() {
		fmt.Println("No tasks found. Run 'hermes prd <file>' to create tasks.")
		return nil
	}

	tasks, err := reader.GetAllTasks()
	if err != nil {
		return err
	}

	// Apply filters
	if opts.filter != "" {
		tasks = ui.FilterTasksByStatus(tasks, task.Status(opts.filter))
	}
	if opts.priority != "" {
		tasks = ui.FilterTasksByPriority(tasks, task.Priority(opts.priority))
	}

	// Display table
	ui.PrintTaskTable(tasks)

	// Show progress
	progress, err := reader.GetProgress()
	if err != nil {
		return err
	}
	ui.PrintProgress(progress)

	// Show circuit breaker status
	breaker := circuit.New(".")
	state, _ := breaker.GetState()
	if state != nil && state.State != circuit.StateClosed {
		fmt.Println()
		breaker.PrintStatus()
	}

	return nil
}
