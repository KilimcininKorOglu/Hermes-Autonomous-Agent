package ai

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	systemColor   = color.New(color.FgHiBlack)
	textColor     = color.New(color.FgWhite)
	toolColor     = color.New(color.FgYellow)
	resultColor   = color.New(color.FgGreen)
	costColor     = color.New(color.FgCyan)
	errorColor    = color.New(color.FgRed)
	providerColor = color.New(color.FgCyan, color.Bold)
)

// StreamDisplay handles displaying stream events
type StreamDisplay struct {
	showTools    bool
	showCost     bool
	providerName string
	textStarted  bool
}

// NewStreamDisplay creates a new stream display
func NewStreamDisplay(showTools, showCost bool, providerName string) *StreamDisplay {
	return &StreamDisplay{
		showTools:    showTools,
		showCost:     showCost,
		providerName: providerName,
		textStarted:  false,
	}
}

// Handle processes and displays a stream event
func (d *StreamDisplay) Handle(event StreamEvent) {
	switch event.Type {
	case "system":
		systemColor.Printf("[Model: %s]\n", event.Model)

	case "text", "assistant":
		if !d.textStarted && d.providerName != "" {
			providerColor.Printf("[%s] ", d.providerName)
			d.textStarted = true
		}
		textColor.Print(event.Text)

	case "tool_use":
		d.textStarted = false
		if d.showTools {
			toolColor.Printf("\n[Tool: %s]", event.ToolName)
			// Show brief info about what the tool is doing
			if event.ToolInput != nil {
				if file, ok := event.ToolInput["file_path"].(string); ok {
					toolColor.Printf(" %s", file)
				} else if cmd, ok := event.ToolInput["command"].(string); ok {
					if len(cmd) > 50 {
						cmd = cmd[:50] + "..."
					}
					toolColor.Printf(" %s", cmd)
				} else if pattern, ok := event.ToolInput["pattern"].(string); ok {
					toolColor.Printf(" %s", pattern)
				} else if content, ok := event.ToolInput["content"].(string); ok {
					if len(content) > 30 {
						content = content[:30] + "..."
					}
					toolColor.Printf(" %s", content)
				}
			}
		}

	case "tool_result":
		if d.showTools {
			if event.ToolError != "" {
				errorColor.Printf(" [Error: %s]", event.ToolError)
			} else {
				toolColor.Print(" [Done]")
			}
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
