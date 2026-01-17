package circuit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	HalfOpenThreshold = 2 // Loops without progress before HALF_OPEN
	OpenThreshold     = 3 // Loops without progress before OPEN
)

// Breaker implements the circuit breaker pattern
type Breaker struct {
	basePath    string
	stateFile   string
	historyFile string
}

// New creates a new circuit breaker
func New(basePath string) *Breaker {
	hermesDir := filepath.Join(basePath, ".hermes")
	return &Breaker{
		basePath:    basePath,
		stateFile:   filepath.Join(hermesDir, "circuit-state.json"),
		historyFile: filepath.Join(hermesDir, "circuit-history.json"),
	}
}

// Initialize creates the state file if it doesn't exist
func (b *Breaker) Initialize() error {
	dir := filepath.Dir(b.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if _, err := os.Stat(b.stateFile); os.IsNotExist(err) {
		state := &BreakerState{
			State:       StateClosed,
			LastUpdated: time.Now(),
		}
		return b.saveState(state)
	}

	return nil
}

// GetState returns the current circuit breaker state
func (b *Breaker) GetState() (*BreakerState, error) {
	data, err := os.ReadFile(b.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &BreakerState{State: StateClosed}, nil
		}
		return nil, err
	}

	var state BreakerState
	if err := json.Unmarshal(data, &state); err != nil {
		return &BreakerState{State: StateClosed}, nil
	}

	return &state, nil
}

func (b *Breaker) saveState(state *BreakerState) error {
	state.LastUpdated = time.Now()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(b.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(b.stateFile, data, 0644)
}

// CanExecute returns true if execution is allowed
func (b *Breaker) CanExecute() (bool, error) {
	state, err := b.GetState()
	if err != nil {
		return false, err
	}
	return state.State != StateOpen, nil
}

// AddLoopResult records a loop result and updates state
func (b *Breaker) AddLoopResult(hasProgress, hasError bool, loopNumber int) (bool, error) {
	state, err := b.GetState()
	if err != nil {
		return false, err
	}

	oldState := state.State
	state.CurrentLoop = loopNumber

	if hasProgress {
		// Progress detected - reset counters and close circuit
		state.ConsecutiveNoProgress = 0
		state.LastProgress = loopNumber
		if state.State != StateClosed {
			state.State = StateClosed
			state.Reason = "Progress detected, circuit recovered"
		}
	} else {
		// No progress
		state.ConsecutiveNoProgress++

		if state.ConsecutiveNoProgress >= OpenThreshold {
			if state.State != StateOpen {
				state.State = StateOpen
				state.TotalOpens++
				state.Reason = fmt.Sprintf("No progress for %d loops, opening circuit", state.ConsecutiveNoProgress)
			}
		} else if state.ConsecutiveNoProgress >= HalfOpenThreshold {
			if state.State == StateClosed {
				state.State = StateHalfOpen
				state.Reason = fmt.Sprintf("Monitoring: %d loops without progress", state.ConsecutiveNoProgress)
			}
		}
	}

	if hasError {
		state.ConsecutiveErrors++
	} else {
		state.ConsecutiveErrors = 0
	}

	// Log state transition
	if oldState != state.State {
		b.addHistory(&HistoryEntry{
			Timestamp:  time.Now(),
			LoopNumber: loopNumber,
			FromState:  oldState,
			ToState:    state.State,
			Reason:     state.Reason,
			Progress:   hasProgress,
			HasError:   hasError,
		})
	}

	if err := b.saveState(state); err != nil {
		return false, err
	}

	return state.State != StateOpen, nil
}

// Reset resets the circuit breaker to closed state
func (b *Breaker) Reset(reason string) error {
	oldState, _ := b.GetState()

	state := &BreakerState{
		State:       StateClosed,
		Reason:      reason,
		TotalOpens:  oldState.TotalOpens, // Preserve total opens
		LastUpdated: time.Now(),
	}

	if oldState != nil && oldState.State != StateClosed {
		b.addHistory(&HistoryEntry{
			Timestamp: time.Now(),
			FromState: oldState.State,
			ToState:   StateClosed,
			Reason:    reason,
		})
	}

	return b.saveState(state)
}

// ShouldHalt returns true if execution should stop
func (b *Breaker) ShouldHalt() (bool, error) {
	state, err := b.GetState()
	if err != nil {
		return false, err
	}
	return state.State == StateOpen, nil
}

func (b *Breaker) addHistory(entry *HistoryEntry) error {
	var history []HistoryEntry

	data, err := os.ReadFile(b.historyFile)
	if err == nil {
		json.Unmarshal(data, &history)
	}

	history = append(history, *entry)

	// Keep last 100 entries
	if len(history) > 100 {
		history = history[len(history)-100:]
	}

	data, err = json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(b.historyFile, data, 0644)
}

// GetHistory returns the state transition history
func (b *Breaker) GetHistory() ([]HistoryEntry, error) {
	data, err := os.ReadFile(b.historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var history []HistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	return history, nil
}
