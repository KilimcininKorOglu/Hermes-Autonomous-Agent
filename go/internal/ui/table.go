package ui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"hermes/internal/task"
)

// Column represents a table column
type Column struct {
	Name  string
	Width int
}

var taskColumns = []Column{
	{"ID", 6},
	{"Name", 35},
	{"Status", 12},
	{"Priority", 8},
	{"Feature", 6},
}

// GetStatusColor returns the color for a status
func GetStatusColor(status task.Status) *color.Color {
	switch status {
	case task.StatusCompleted:
		return color.New(color.FgGreen)
	case task.StatusInProgress:
		return color.New(color.FgYellow)
	case task.StatusBlocked:
		return color.New(color.FgRed)
	case task.StatusNotStarted:
		return color.New(color.FgHiBlack)
	default:
		return color.New(color.FgWhite)
	}
}

// GetPriorityColor returns the color for a priority
func GetPriorityColor(priority task.Priority) *color.Color {
	switch priority {
	case task.PriorityP1:
		return color.New(color.FgRed)
	case task.PriorityP2:
		return color.New(color.FgYellow)
	case task.PriorityP3:
		return color.New(color.FgCyan)
	case task.PriorityP4:
		return color.New(color.FgHiBlack)
	default:
		return color.New(color.FgWhite)
	}
}

// FormatTaskTable formats tasks as an ASCII table
func FormatTaskTable(tasks []task.Task) string {
	if len(tasks) == 0 {
		return "No tasks found."
	}

	var sb strings.Builder

	// Header
	sb.WriteString(formatSeparator("top"))
	sb.WriteString(formatHeader())
	sb.WriteString(formatSeparator("middle"))

	// Rows
	for _, t := range tasks {
		sb.WriteString(formatTaskRow(t))
	}

	sb.WriteString(formatSeparator("bottom"))

	return sb.String()
}

func formatSeparator(position string) string {
	var left, mid, right, line string
	switch position {
	case "top":
		left, mid, right, line = "+", "+", "+", "-"
	case "middle":
		left, mid, right, line = "+", "+", "+", "-"
	case "bottom":
		left, mid, right, line = "+", "+", "+", "-"
	}

	var parts []string
	for _, col := range taskColumns {
		parts = append(parts, strings.Repeat(line, col.Width+2))
	}

	return left + strings.Join(parts, mid) + right + "\n"
}

func formatHeader() string {
	var parts []string
	for _, col := range taskColumns {
		parts = append(parts, padRight(col.Name, col.Width))
	}
	return "| " + strings.Join(parts, " | ") + " |\n"
}

func formatTaskRow(t task.Task) string {
	values := []string{
		padRight(t.ID, taskColumns[0].Width),
		truncate(t.Name, taskColumns[1].Width),
		padRight(string(t.Status), taskColumns[2].Width),
		padRight(string(t.Priority), taskColumns[3].Width),
		padRight(t.FeatureID, taskColumns[4].Width),
	}

	return "| " + strings.Join(values, " | ") + " |\n"
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func truncate(s string, width int) string {
	if len(s) <= width {
		return padRight(s, width)
	}
	return s[:width-3] + "..."
}

// PrintTaskTable prints the task table with colors
func PrintTaskTable(tasks []task.Task) {
	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return
	}

	// Header
	fmt.Print(formatSeparator("top"))
	fmt.Print(formatHeader())
	fmt.Print(formatSeparator("middle"))

	// Rows with colors
	for _, t := range tasks {
		statusColor := GetStatusColor(t.Status)
		priorityColor := GetPriorityColor(t.Priority)

		fmt.Print("| ")
		fmt.Print(padRight(t.ID, taskColumns[0].Width))
		fmt.Print(" | ")
		fmt.Print(truncate(t.Name, taskColumns[1].Width))
		fmt.Print(" | ")
		statusColor.Print(padRight(string(t.Status), taskColumns[2].Width))
		fmt.Print(" | ")
		priorityColor.Print(padRight(string(t.Priority), taskColumns[3].Width))
		fmt.Print(" | ")
		fmt.Print(padRight(t.FeatureID, taskColumns[4].Width))
		fmt.Println(" |")
	}

	fmt.Print(formatSeparator("bottom"))
}

// FilterTasksByStatus filters tasks by status
func FilterTasksByStatus(tasks []task.Task, status task.Status) []task.Task {
	var filtered []task.Task
	for _, t := range tasks {
		if t.Status == status {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// FilterTasksByPriority filters tasks by priority
func FilterTasksByPriority(tasks []task.Task, priority task.Priority) []task.Task {
	var filtered []task.Task
	for _, t := range tasks {
		if t.Priority == priority {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// FilterTasksByFeature filters tasks by feature ID
func FilterTasksByFeature(tasks []task.Task, featureID string) []task.Task {
	var filtered []task.Task
	for _, t := range tasks {
		if t.FeatureID == featureID {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
