package ai

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	systemColor = color.New(color.FgHiBlack)
	textColor   = color.New(color.FgWhite)
	toolColor   = color.New(color.FgYellow)
	resultColor = color.New(color.FgGreen)
	costColor   = color.New(color.FgCyan)
	errorColor  = color.New(color.FgRed)
)

// StreamDisplay handles displaying stream events
type StreamDisplay struct {
	showTools bool
	showCost  bool
}

// NewStreamDisplay creates a new stream display
func NewStreamDisplay(showTools, showCost bool) *StreamDisplay {
	return &StreamDisplay{
		showTools: showTools,
		showCost:  showCost,
	}
}

// Handle processes and displays a stream event
func (d *StreamDisplay) Handle(event StreamEvent) {
	switch event.Type {
	case "system":
		systemColor.Printf("[Model: %s]\n", event.Model)

	case "assistant":
		textColor.Print(event.Text)

	case "tool_use":
		if d.showTools {
			toolColor.Printf("\n[Tool: %s]", event.ToolName)
		}

	case "tool_result":
		if d.showTools {
			toolColor.Print(" [Done]")
		}

	case "result":
		fmt.Println()
		if d.showCost {
			resultColor.Print("[Complete] ")
			costColor.Printf("%.1fs | $%.4f\n", event.Duration, event.Cost)
		}

	case "error":
		errorColor.Printf("\n[Error] %s\n", event.Text)
	}
}

// DisplayEvents consumes and displays all events from a channel
func (d *StreamDisplay) DisplayEvents(events <-chan StreamEvent) {
	for event := range events {
		d.Handle(event)
	}
}
