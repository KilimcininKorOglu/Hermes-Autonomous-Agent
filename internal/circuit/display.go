package circuit

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// PrintStatus prints the current circuit breaker status
func (b *Breaker) PrintStatus() error {
	state, err := b.GetState()
	if err != nil {
		return err
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("           Circuit Breaker Status")
	fmt.Println(strings.Repeat("=", 60))

	stateColor := color.New(color.FgGreen)
	stateIcon := "[OK]"

	switch state.State {
	case StateHalfOpen:
		stateColor = color.New(color.FgYellow)
		stateIcon = "[!!]"
	case StateOpen:
		stateColor = color.New(color.FgRed)
		stateIcon = "[XX]"
	}

	fmt.Printf("State:                 ")
	stateColor.Printf("%s %s\n", stateIcon, state.State)
	fmt.Printf("Reason:                %s\n", state.Reason)
	fmt.Printf("Loops since progress:  %d\n", state.ConsecutiveNoProgress)
	fmt.Printf("Last progress:         Loop #%d\n", state.LastProgress)
	fmt.Printf("Current loop:          #%d\n", state.CurrentLoop)
	fmt.Printf("Total opens:           %d\n", state.TotalOpens)
	fmt.Println(strings.Repeat("=", 60))

	return nil
}

// PrintHaltMessage prints the halt message when circuit is open
func (b *Breaker) PrintHaltMessage() {
	red := color.New(color.FgRed, color.Bold)

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	red.Println("  EXECUTION HALTED: Circuit Breaker Opened")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
	fmt.Println("Hermes has detected that no progress is being made.")
	fmt.Println()
	fmt.Println("Possible reasons:")
	fmt.Println("  - Project may be complete")
	fmt.Println("  - AI may be stuck on an error")
	fmt.Println("  - PROMPT.md may need clarification")
	fmt.Println()
	fmt.Println("To continue:")
	fmt.Println("  1. Review recent logs")
	fmt.Println("  2. Check AI output")
	fmt.Println("  3. Reset circuit breaker:")
	fmt.Println("     hermes reset")
}

// GetStateIcon returns an icon for the state
func GetStateIcon(state State) string {
	switch state {
	case StateClosed:
		return "[OK]"
	case StateHalfOpen:
		return "[!!]"
	case StateOpen:
		return "[XX]"
	default:
		return "[??]"
	}
}

// GetStateColor returns a color for the state
func GetStateColor(state State) *color.Color {
	switch state {
	case StateClosed:
		return color.New(color.FgGreen)
	case StateHalfOpen:
		return color.New(color.FgYellow)
	case StateOpen:
		return color.New(color.FgRed)
	default:
		return color.New(color.FgWhite)
	}
}
