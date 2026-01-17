package circuit

import "time"

// State represents the circuit breaker state
type State string

const (
	StateClosed   State = "CLOSED"    // Normal operation
	StateHalfOpen State = "HALF_OPEN" // Monitoring for recovery
	StateOpen     State = "OPEN"      // Halted, no execution
)

// BreakerState contains the current state of the circuit breaker
type BreakerState struct {
	State                 State     `json:"state"`
	ConsecutiveNoProgress int       `json:"consecutiveNoProgress"`
	ConsecutiveErrors     int       `json:"consecutiveErrors"`
	LastProgress          int       `json:"lastProgress"`
	CurrentLoop           int       `json:"currentLoop"`
	TotalOpens            int       `json:"totalOpens"`
	LastUpdated           time.Time `json:"lastUpdated"`
	Reason                string    `json:"reason"`
}

// HistoryEntry records a state transition
type HistoryEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	LoopNumber int       `json:"loopNumber"`
	FromState  State     `json:"fromState"`
	ToState    State     `json:"toState"`
	Reason     string    `json:"reason"`
	Progress   bool      `json:"progress"`
	HasError   bool      `json:"hasError"`
}
