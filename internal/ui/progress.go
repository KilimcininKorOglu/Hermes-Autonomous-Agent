package ui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"hermes/internal/task"
)

// FormatProgressBar creates a progress bar string
func FormatProgressBar(percentage float64, width int) string {
	filled := int(percentage / 100 * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := strings.Repeat("#", filled) + strings.Repeat("-", empty)
	return fmt.Sprintf("[%s] %.1f%%", bar, percentage)
}

// PrintProgress prints task progress to console
func PrintProgress(progress *task.Progress) {
	fmt.Println()
	fmt.Println("Task Progress")
	fmt.Println(strings.Repeat("-", 40))

	bar := FormatProgressBar(progress.Percentage, 30)
	fmt.Println(bar)

	fmt.Printf("\nTotal:       %d\n", progress.Total)

	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	gray := color.New(color.FgHiBlack)
	red := color.New(color.FgRed)

	fmt.Print("Completed:   ")
	green.Printf("%d\n", progress.Completed)

	fmt.Print("In Progress: ")
	yellow.Printf("%d\n", progress.InProgress)

	fmt.Print("Not Started: ")
	gray.Printf("%d\n", progress.NotStarted)

	fmt.Print("Blocked:     ")
	red.Printf("%d\n", progress.Blocked)

	fmt.Println(strings.Repeat("-", 40))
}

// PrintHeader prints a styled header
func PrintHeader(title string) {
	cyan := color.New(color.FgCyan, color.Bold)
	fmt.Println()
	cyan.Println(title)
	fmt.Println(strings.Repeat("=", len(title)))
	fmt.Println()
}

// PrintSection prints a section heading
func PrintSection(title string) {
	yellow := color.New(color.FgYellow)
	fmt.Println()
	yellow.Println(title)
	fmt.Println(strings.Repeat("-", len(title)))
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	green := color.New(color.FgGreen, color.Bold)
	green.Printf("[OK] %s\n", message)
}

// PrintError prints an error message
func PrintError(message string) {
	red := color.New(color.FgRed, color.Bold)
	red.Printf("[ERROR] %s\n", message)
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	yellow := color.New(color.FgYellow)
	yellow.Printf("[WARN] %s\n", message)
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	cyan := color.New(color.FgCyan)
	cyan.Printf("[INFO] %s\n", message)
}

// PrintLoopHeader prints the loop header
func PrintLoopHeader(loopNumber int) {
	cyan := color.New(color.FgCyan, color.Bold)
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	cyan.Printf("                    Loop #%d\n", loopNumber)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
}

// PrintTaskHeader prints the current task header
func PrintTaskHeader(t *task.Task) {
	yellow := color.New(color.FgYellow, color.Bold)
	fmt.Println()
	yellow.Printf("Current Task: %s\n", t.ID)
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Name:     %s\n", t.Name)
	fmt.Printf("Priority: %s\n", t.Priority)
	fmt.Printf("Feature:  %s\n", t.FeatureID)
	fmt.Println(strings.Repeat("-", 40))
}

// PrintDivider prints a divider line
func PrintDivider() {
	fmt.Println(strings.Repeat("-", 60))
}

// PrintBanner prints the Hermes banner with version
func PrintBanner(version string) {
	cyan := color.New(color.FgCyan, color.Bold)
	gray := color.New(color.FgHiBlack)
	cyan.Print(`
 _   _                                
| | | | ___ _ __ _ __ ___   ___  ___  
| |_| |/ _ \ '__| '_ ` + "`" + ` _ \ / _ \/ __| 
|  _  |  __/ |  | | | | | |  __/\__ \ 
|_| |_|\___|_|  |_| |_| |_|\___||___/ 
                                      
`)
	cyan.Print("      AI-Powered Application Development")
	if version != "" {
		gray.Printf(" %s", version)
	}
	fmt.Println()
}
