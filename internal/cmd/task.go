package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"hermes/internal/task"
)

// NewTaskCmd creates the task command
func NewTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task [id]",
		Short: "Show task details",
		Long:  "Display detailed information about a specific task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTask,
	}
	return cmd
}

func runTask(cmd *cobra.Command, args []string) error {
	taskID := strings.ToUpper(args[0])
	if !strings.HasPrefix(taskID, "T") {
		// Pad with zeros if numeric (1 -> T001, 12 -> T012)
		taskID = fmt.Sprintf("T%03s", taskID)
	}

	reader := task.NewReader(".")
	tasks, err := reader.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}

	var found *task.Task
	for _, t := range tasks {
		if t.ID == taskID {
			found = &t
			break
		}
	}

	if found == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Print task details
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	
	fmt.Println()
	bold.Printf("Task: %s\n", found.ID)
	fmt.Println(strings.Repeat("-", 50))
	
	fmt.Printf("Name:     %s\n", found.Name)
	
	// Status with color
	fmt.Print("Status:   ")
	switch found.Status {
	case task.StatusCompleted:
		color.Green("%s\n", found.Status)
	case task.StatusInProgress:
		color.Yellow("%s\n", found.Status)
	case task.StatusBlocked:
		color.Red("%s\n", found.Status)
	default:
		fmt.Printf("%s\n", found.Status)
	}
	
	// Priority with color
	fmt.Print("Priority: ")
	switch found.Priority {
	case task.PriorityP1:
		color.Red("%s\n", found.Priority)
	case task.PriorityP2:
		color.Yellow("%s\n", found.Priority)
	default:
		fmt.Printf("%s\n", found.Priority)
	}
	
	fmt.Printf("Feature:  %s\n", found.FeatureID)
	
	// Files
	if len(found.FilesToTouch) > 0 {
		fmt.Println()
		cyan.Println("Files to Touch:")
		for _, f := range found.FilesToTouch {
			fmt.Printf("  - %s\n", f)
		}
	}
	
	// Dependencies
	if len(found.Dependencies) > 0 {
		fmt.Println()
		cyan.Println("Dependencies:")
		for _, d := range found.Dependencies {
			fmt.Printf("  - %s\n", d)
		}
	}
	
	// Success Criteria
	if len(found.SuccessCriteria) > 0 {
		fmt.Println()
		cyan.Println("Success Criteria:")
		for _, c := range found.SuccessCriteria {
			fmt.Printf("  - %s\n", c)
		}
	}
	
	fmt.Println()
	return nil
}
